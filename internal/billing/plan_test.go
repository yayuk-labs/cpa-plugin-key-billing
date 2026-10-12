package billing

import (
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"cpa-key-billing/internal/messages"
)

func TestPlanModelSelectionsRetainUsageAndOwnTheirData(t *testing.T) {
	store, _, now := quotaStore(t, QuotaWindow{AmountUSD: 10})
	store.RecordUsage(subsetEvent("s", now))
	before := currentQuotaCycle(t, store)
	windows := store.Plans()[0].Windows
	windows[0].Models = []string{" gpt-5.5 ", "GPT-5.5", "gpt-6-luna"}
	updated, err := store.UpdatePlanWithBindings(PlanPatch{ID: "p", Windows: &windows}, nil)
	if err != nil || !reflect.DeepEqual(updated.Windows[0].Models, []string{"gpt-5.5", "gpt-6-luna"}) {
		t.Fatalf("model selection not normalized: %+v, %v", updated, err)
	}
	if after := currentQuotaCycle(t, store); after != before {
		t.Fatalf("model edit changed usage or schedule: %+v, want %+v", after, before)
	}
	windows[0].Models[0] = "mutated-input"
	updated.Windows[0].Models[0] = "mutated-output"
	views := store.Authorize("s", now, "gpt-5.5")
	views.Windows[0].Models[0] = "mutated-view"
	if models := store.Plans()[0].Windows[0].Models; !reflect.DeepEqual(models, []string{"gpt-5.5", "gpt-6-luna"}) {
		t.Fatalf("caller mutated stored model selections: %+v", models)
	}
	for _, invalid := range []string{"", " ", strings.Repeat("m", maxRouteValueBytes+1)} {
		windows = store.Plans()[0].Windows
		windows[0].Models = []string{invalid}
		_, err := store.UpdatePlanWithBindings(PlanPatch{ID: "p", Windows: &windows}, nil)
		if err == nil {
			t.Fatalf("invalid model ID accepted: %q", invalid)
		}
		if detail := messages.FromError(err); detail.Key != "backend.window_value_model_ids_must_be_non_empty_and_must_not_exceed_value_bytes" {
			t.Fatalf("model validation missing translation metadata: %+v", detail)
		}
	}
}

func TestQuotaWindowsValidationAndIdentity(t *testing.T) {
	now := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	input := []QuotaWindow{{Name: " Long ", AmountUSD: 10, PeriodSeconds: 7200}, {Name: "Short", AmountUSD: 2, PeriodSeconds: 3600}}
	windows, err := prepareWindows(input, nil, now)
	if err != nil || windows[0].Name != "Short" || windows[1].Name != "Long" || windows[0].ID == windows[1].ID {
		t.Fatalf("windows = %+v, %v", windows, err)
	}
	valid := Plan{ID: "p", Windows: windows}
	if err := valid.Validate(); err != nil {
		t.Fatal(err)
	}
	edited, err := prepareWindows(windows, windows, now)
	if err != nil || edited[0].ID != windows[0].ID {
		t.Fatal("editing changed window identity")
	}
	if _, err := prepareWindows(windows, nil, now); err == nil {
		t.Fatal("unknown IDs accepted")
	}
	for i := range valid.Windows {
		valid.Windows[i].CycleAnchorAt = now.Add(time.Duration(valid.Windows[i].PeriodSeconds) * time.Second)
	}
	if err := valid.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*Plan){
		func(p *Plan) { p.Windows = nil },
		func(p *Plan) { p.Windows[0].Name = " long " },
		func(p *Plan) { p.Windows[0].PeriodSeconds = 7200 },
		func(p *Plan) { p.Windows[0].PeriodSeconds = 0 },
		func(p *Plan) { p.Windows[0].PeriodSeconds = -1 },
		func(p *Plan) { p.Windows[0].PeriodSeconds = maxPeriodSeconds + 1 },
		func(p *Plan) { p.Windows[0].AmountUSD = math.NaN() },
		func(p *Plan) { p.Windows[0].AmountUSD = math.Inf(1) },
		func(p *Plan) { p.Windows[0].AmountUSD = 0 },
		func(p *Plan) { p.Windows[0].AmountUSD = -1 },
		func(p *Plan) { p.Windows[0].RequestLimit = -1 },
		func(p *Plan) { p.Windows[0].RequestLimit = maxQuotaCount + 1 },
		func(p *Plan) { p.Windows[0].TokenLimit = -1 },
		func(p *Plan) { p.Windows[0].TokenLimit = maxQuotaCount + 1 },
		func(p *Plan) { p.Windows[0].CycleAnchorAt = time.Time{} },
		func(p *Plan) { p.Windows[0].CycleAnchorAt = now.Add(time.Nanosecond) },
		func(p *Plan) { p.Windows[0].CycleAnchorAt = time.Date(1969, 1, 1, 0, 0, 0, 0, time.UTC) },
	} {
		invalid := clonePlan(valid)
		change(&invalid)
		if invalid.Validate() == nil {
			t.Fatalf("invalid plan accepted: %+v", invalid)
		}
	}
	for _, anchor := range []time.Time{now.Add(-time.Second), now, now.Add(time.Hour + time.Second)} {
		input := []QuotaWindow{{Name: "Short", AmountUSD: 1, PeriodSeconds: 3600, CycleAnchorAt: anchor}}
		if _, err := prepareWindows(input, nil, now); err == nil {
			t.Fatalf("invalid next start accepted: %v", anchor)
		}
	}
	edited, err = prepareWindows(valid.Windows, valid.Windows, now.Add(100*24*time.Hour))
	if err != nil || !edited[0].CycleAnchorAt.Equal(valid.Windows[0].CycleAnchorAt) {
		t.Fatalf("unchanged past anchor: %+v, %v", edited, err)
	}
	window := valid.Windows[0]
	for _, at := range []time.Time{now.Add(-17 * time.Minute), time.Date(2500, 1, 1, 12, 17, 0, 0, time.UTC)} {
		cycle := window.newCycle(valid.ID, at)
		wantStart := time.Date(at.Year(), at.Month(), at.Day(), at.Hour(), 0, 0, 0, time.UTC)
		if !cycle.StartAt.Equal(wantStart) || !cycle.EndAt.Equal(wantStart.Add(time.Hour)) {
			t.Fatalf("invalid fixed interval at %v: %+v", at, cycle)
		}
	}
	cycle := window.newCycle(valid.ID, now)
	cycle.UsageSince = cycle.EndAt
	key := KeyState{PlanID: valid.ID, Cycles: map[string]QuotaCycle{window.ID: cycle}}
	if key.ValidateCycles(valid) == nil {
		t.Fatal("usage cutoff outside cycle accepted")
	}
	shifted := clonePlan(valid)
	shifted.Windows[0].CycleAnchorAt = shifted.Windows[0].CycleAnchorAt.Add(time.Hour)
	edited, err = prepareWindows(shifted.Windows, valid.Windows, now)
	if err != nil || !edited[0].CycleAnchorAt.Equal(valid.Windows[0].CycleAnchorAt) {
		t.Fatalf("equivalent schedule changed: %+v, %v", edited, err)
	}
}

func TestPlanAcceptsIndependentQuotaDimensions(t *testing.T) {
	plan := Plan{ID: "p", Windows: []QuotaWindow{
		{ID: "requests", Name: "请求", PeriodSeconds: 3600, RequestLimit: maxQuotaCount},
		{ID: "tokens", Name: "Token", PeriodSeconds: 86400, TokenLimit: maxQuotaCount},
		{ID: "mixed", Name: "综合", PeriodSeconds: 604800, AmountUSD: 100, TokenLimit: 10000, RequestLimit: 100},
	}}
	if err := plan.Validate(); err != nil {
		t.Fatal(err)
	}
}
