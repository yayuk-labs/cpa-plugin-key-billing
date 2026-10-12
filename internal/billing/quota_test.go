package billing

import (
	"encoding/json"
	"errors"
	"math"
	"sync"
	"testing"
	"time"
)

func quotaStore(t *testing.T, window QuotaWindow) (*Store, *memoryRepository, time.Time) {
	t.Helper()
	now := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	store, repo := newAccountStoreWithRepository(t, now)
	window.ID, window.Name, window.PeriodSeconds = "w", "额度", 3600
	store.ReplaceAll(func(state *State) {
		state.Plans = []Plan{{ID: "p", Windows: []QuotaWindow{window}}}
		state.Keys["s"] = &KeyState{PlanID: "p", Preview: "sk-tes…0001"}
	})
	store.Authorize("s", now, "")
	return store, repo, now
}

func currentQuotaCycle(t *testing.T, store *Store) QuotaCycle {
	t.Helper()
	var cycle QuotaCycle
	store.Read(func(state *State) { cycle = state.Keys["s"].Cycles["w"] })
	return cycle
}

func TestModelQuotaWindowsShareBudgetAndExcludeFallback(t *testing.T) {
	now := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	store := newAccountStore(t, now)
	plan, err := store.CreatePlanWithBindings(Plan{Name: "Model budgets", Windows: []QuotaWindow{
		{Name: "Selected daily", PeriodSeconds: 86400, AmountUSD: 5, Models: []string{" gpt-6-luna ", "gpt-5.6-luna"}},
		{Name: "Other monthly", PeriodSeconds: 30 * 86400, AmountUSD: 10},
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	store.ReplaceAll(func(state *State) {
		state.Keys["s"] = &KeyState{PlanID: plan.ID}
		for _, model := range []string{"gpt-6-luna", "gpt-5.6-luna", "gpt-5.5"} {
			state.Prices[model] = CustomPrice{ModelID: model, PriceRates: PriceRates{InputPer1M: 1}}
		}
	})
	charge := func(model string, dollars int64, at time.Time) {
		t.Helper()
		if decision := store.Authorize("s", at, model); !decision.Allowed {
			t.Fatalf("%s refused before usage: %+v", model, decision)
		}
		event := subsetEvent("s", at)
		event.RouteModel, event.UpstreamModel = model, "provider-alias"
		event.Breakdown = completeBreakdown(dollars*1_000_000, 0, 0, 0, 0)
		store.RecordUsage(event)
	}
	charge("GPT-6-LUNA(high)", 3, now)
	charge("gpt-5.6-luna", 2, now)
	if decision := store.Authorize("s", now, "gpt-6-luna"); decision.Allowed || len(decision.Windows) != 1 || decision.Windows[0].Dimensions[0].Used != "5" || !decision.RetryAt.Equal(now.Add(24*time.Hour)) {
		t.Fatalf("selected models did not share daily budget: %+v", decision)
	}
	charge("gpt-5.5", 10, now)
	if decision := store.Authorize("s", now, "gpt-5.5"); decision.Allowed || len(decision.Windows) != 1 || decision.Windows[0].Dimensions[0].Used != "10" {
		t.Fatalf("fallback budget did not exclude selected models: %+v", decision)
	}
	if decision := store.Authorize("s", now.Add(24*time.Hour), "gpt-6-luna"); !decision.Allowed {
		t.Fatalf("daily budget did not reset independently: %+v", decision)
	}
	if decision := store.Authorize("s", now.Add(24*time.Hour), "gpt-5.5"); decision.Allowed {
		t.Fatal("daily rollover reset monthly budget")
	}
}

func TestModelQuotaOverlappingSelectionsAndFailedUsage(t *testing.T) {
	now := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	store := newAccountStore(t, now)
	store.ReplaceAll(func(state *State) {
		state.Plans = []Plan{{ID: "p", Windows: []QuotaWindow{
			{ID: "daily", Name: "Daily", Models: []string{"gpt-5.5"}, PeriodSeconds: 86400, TokenLimit: 2000},
			{ID: "monthly", Name: "Monthly", Models: []string{"gpt-5.5", "gpt-6-luna"}, PeriodSeconds: 30 * 86400, RequestLimit: 2},
			{ID: "fallback", Name: "Fallback", PeriodSeconds: 7 * 86400, RequestLimit: 1},
		}}}
		state.Keys["s"] = &KeyState{PlanID: "p"}
	})
	if decision := store.Authorize("s", now, "gpt-5.5"); !decision.Allowed || len(decision.Windows) != 2 {
		t.Fatalf("overlapping selections not applied: %+v", decision)
	}
	store.RecordUsageError(subsetEvent("s", now), RequestError{StatusCode: 502})
	decision := store.Authorize("s", now, "gpt-5.5")
	if !decision.Allowed || decision.Windows[0].Dimensions[0].Used != "1500" || decision.Windows[1].Dimensions[0].Used != "0" {
		t.Fatalf("failed usage counted incorrectly: %+v", decision)
	}
	store.RecordUsage(subsetEvent("s", now))
	if decision = store.Authorize("s", now, "gpt-5.5"); decision.Allowed || decision.Windows[0].Dimensions[0].Used != "3000" || decision.Windows[1].Dimensions[0].Used != "1" {
		t.Fatalf("matching windows did not both receive usage: %+v", decision)
	}
	if decision = store.Authorize("s", now, "gpt-6-luna"); !decision.Allowed || len(decision.Windows) != 1 {
		t.Fatalf("unmatched exhausted window blocked another selection: %+v", decision)
	}
	if decision = store.Authorize("s", now, "other-model"); !decision.Allowed || len(decision.Windows) != 1 || decision.Windows[0].Started == false || decision.Windows[0].Dimensions[0].Used != "0" {
		t.Fatalf("fallback charged selected model usage: %+v", decision)
	}
}

func TestQuotaCountsReportedUsage(t *testing.T) {
	store, _, now := quotaStore(t, QuotaWindow{RequestLimit: 2, TokenLimit: 2500})
	for range 3 {
		if d := store.Authorize("s", now, ""); !d.Allowed || currentQuotaCycle(t, store).UsedRequests != 0 {
			t.Fatalf("admission consumed quota: %+v", d)
		}
	}
	store.RecordUsageError(subsetEvent("s", now), RequestError{StatusCode: 502})
	d := store.Authorize("s", now, "")
	if !d.Allowed || currentQuotaCycle(t, store).UsedRequests != 0 || currentQuotaCycle(t, store).UsedTokens != 1500 {
		t.Fatalf("failed usage: %+v", d)
	}
	zero := subsetEvent("s", now)
	zero.Breakdown = TokenBreakdown{}
	store.RecordUsage(zero)
	d = store.Authorize("s", now, "")
	if !d.Allowed || currentQuotaCycle(t, store).UsedRequests != 1 {
		t.Fatalf("successful missing usage: %+v", d)
	}
	store.RecordUsage(subsetEvent("s", now))
	d = store.Authorize("s", now, "")
	w := d.Windows[0]
	if d.Allowed || currentQuotaCycle(t, store).UsedRequests != 2 || currentQuotaCycle(t, store).UsedTokens != 3000 || len(w.Dimensions) != 2 || !w.Dimensions[0].Blocked || !w.Dimensions[1].Blocked ||
		w.Dimensions[0].Metric != QuotaTokens || w.Dimensions[0].Remaining != "0" {
		t.Fatalf("combined exhaustion: %+v", d)
	}
}

func TestQuotaTracksTokensWithoutPricing(t *testing.T) {
	store, _, now := quotaStore(t, QuotaWindow{TokenLimit: 1000})
	store.ReplaceAll(func(state *State) { state.Prices["gpt-5.5"] = CustomPrice{ModelID: "gpt-5.5"} })
	store.RecordUsage(subsetEvent("s", now))
	w := store.Authorize("s", now, "").Windows[0]
	if !w.Blocked || currentQuotaCycle(t, store).UsedTokens != 1500 || currentQuotaCycle(t, store).SpentUSD != 0 {
		t.Fatalf("zero price lost tokens: %+v", w)
	}

	store, _, now = quotaStore(t, QuotaWindow{TokenLimit: 1000})
	event := subsetEvent("s", now)
	event.Breakdown = TokenBreakdown{Quality: TokenAccountingUnclassified, TotalTokens: 1200, UnclassifiedTokens: 1200}
	store.RecordUsage(event)
	if w = store.Authorize("s", now, "").Windows[0]; currentQuotaCycle(t, store).UsedTokens != 1200 || !w.Blocked || currentQuotaCycle(t, store).SpentUSD != 0 {
		t.Fatalf("available tokens lost: %+v", w)
	}
	event.Breakdown.Quality = TokenAccountingInconsistent
	store.RecordUsage(event)
	if w = store.Authorize("s", now, "").Windows[0]; currentQuotaCycle(t, store).UsedTokens != 1200 {
		t.Fatalf("contradictory tokens charged: %+v", w)
	}
}

func TestQuotaPreservesOverageAndRetriesWrites(t *testing.T) {
	store, repo, now := quotaStore(t, QuotaWindow{RequestLimit: 1, TokenLimit: 1})
	repo.fail = errors.New("dummy write failure")
	var group sync.WaitGroup
	for range 8 {
		group.Add(1)
		go func() {
			defer group.Done()
			store.RecordUsage(subsetEvent("s", now))
		}()
	}
	group.Wait()
	w := store.Authorize("s", now, "").Windows[0]
	if currentQuotaCycle(t, store).UsedRequests != 8 || currentQuotaCycle(t, store).UsedTokens != 12000 || !w.Blocked {
		t.Fatalf("concurrent overage lost: %+v", w)
	}
	repo.fail = nil
	store.Authorize("s", now, "")
	if len(repo.requestEvents) != 8 || !store.dirty.empty() {
		t.Fatal("write retry duplicated or lost usage")
	}
}

func TestLimitEditsKeepCurrentCounters(t *testing.T) {
	store, _, now := quotaStore(t, QuotaWindow{AmountUSD: 10})
	store.RecordUsage(subsetEvent("s", now))
	windows := []QuotaWindow{{ID: "w", Name: "额度", PeriodSeconds: 3600, RequestLimit: 1}}
	if _, err := store.UpdatePlanWithBindings(PlanPatch{ID: "p", Windows: &windows}, nil); err != nil {
		t.Fatal(err)
	}
	if d := store.Authorize("s", now, ""); d.Allowed || currentQuotaCycle(t, store).UsedRequests != 1 {
		t.Fatalf("enabling request limit reset counters: %+v", d)
	}
	windows[0].RequestLimit = 2
	if _, err := store.UpdatePlanWithBindings(PlanPatch{ID: "p", Windows: &windows}, nil); err != nil {
		t.Fatal(err)
	}
	if !store.Authorize("s", now, "").Allowed {
		t.Fatal("raised limit did not take effect")
	}
}

func TestQuotaPreservesNumericBoundaries(t *testing.T) {
	store, _, now := quotaStore(t, QuotaWindow{AmountUSD: 1, TokenLimit: maxQuotaCount, RequestLimit: maxQuotaCount})
	store.ReplaceAll(func(state *State) {
		key := state.Keys["s"]
		cycle := key.Cycles["w"]
		cycle.SpentUSD = math.MaxFloat64 * 0.75
		cycle.UsedTokens, cycle.UsedRequests = maxQuotaCount-1, maxQuotaCount-1
		key.Cycles["w"] = cycle
		key.chargeCycles(state.Plans[0], "gpt-5.5", now, quotaUsage{AmountUSD: math.MaxFloat64 * 0.75, Tokens: 1, Requests: 1})
	})
	view := store.Authorize("s", now, "")
	for _, balance := range view.Windows[0].Dimensions {
		if !balance.Blocked || balance.Remaining != "0" || balance.UsedPercent != 100 {
			t.Fatalf("boundary not exhausted: %+v", balance)
		}
		if balance.Metric != QuotaAmount && balance.Used != "9007199254740991" {
			t.Fatalf("integer counter lost precision: %+v", balance)
		}
	}
	cycle := currentQuotaCycle(t, store)
	if _, err := json.Marshal(cycle); err != nil || cycle.SpentUSD != math.MaxFloat64 {
		t.Fatalf("overflow made the cycle unpersistable: %+v, %v", cycle, err)
	}
	store.ReplaceAll(func(state *State) {
		cycle.UsedTokens, cycle.UsedRequests = math.MaxInt64-1, math.MaxInt64
		state.Keys["s"].Cycles["w"] = cycle
	})
	store.RecordUsage(subsetEvent("s", now))
	cycle = currentQuotaCycle(t, store)
	if cycle.UsedTokens != math.MaxInt64 || cycle.UsedRequests != math.MaxInt64 {
		t.Fatalf("usage counter overflow wrapped: %+v", cycle)
	}
}
