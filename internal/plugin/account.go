package plugin

import (
	"net/http"
	"strings"

	"cpa-key-billing/internal/billing"
	"cpa-key-billing/internal/messages"
)

type accountIdentity struct {
	Preview string `json:"preview,omitempty"`
	Label   string `json:"label,omitempty"`
}

type accountSubscription struct {
	Name string `json:"name,omitempty"`
	billing.QuotaView
}

type accountConcurrency struct {
	Limit   int `json:"limit"`
	Current int `json:"current"`
}

type accountProfileResponse struct {
	Tracked           bool            `json:"tracked"`
	Identity          accountIdentity `json:"identity"`
	CanResetAuthQuota bool            `json:"can_reset_auth_quota"`
}

type accountSubscriptionResponse struct {
	Subscription accountSubscription `json:"subscription"`
	Concurrency  accountConcurrency  `json:"concurrency"`
}

type accountRoutingResponse struct {
	Models          []string           `json:"models"`
	DeniedModels    []string           `json:"denied_models"`
	RoutingValid    bool               `json:"routing_valid"`
	Warnings        []string           `json:"warnings"`
	WarningMessages []messages.Message `json:"warning_messages,omitempty"`
}

func (a *App) accountProfile(access viewAccess) ManagementResponse {
	response := accountProfileResponse{Tracked: access.Tracked}
	if access.Tracked {
		response.Identity = accountIdentity{Preview: access.Key.Preview, Label: access.Key.Label}
	}
	return viewJSON(access, http.StatusOK, response)
}

func (a *App) accountSubscription(access viewAccess) ManagementResponse {
	if !access.Tracked {
		return apiKeyUnauthorized()
	}
	view := access.Key
	return viewJSON(access, http.StatusOK, accountSubscriptionResponse{
		Subscription: accountSubscription{Name: view.PlanName, QuotaView: view.QuotaView},
		Concurrency:  accountConcurrency{Limit: view.ConcurrencyLimit, Current: view.CurrentConcurrency},
	})
}

func (a *App) accountRouting(access viewAccess) ManagementResponse {
	if !access.Tracked {
		return apiKeyUnauthorized()
	}
	decision := a.store.ResolveRouting(access.Scope, "", "")
	response := accountRoutingResponse{
		Models: decision.Models, DeniedModels: decision.DeniedModels,
		RoutingValid: decision.ConfigurationError == "", Warnings: []string{},
	}
	if decision.ConfigurationError != "" {
		response.Warnings = append(response.Warnings, "The routing rule no longer exists; contact your administrator")
		response.WarningMessages = append(response.WarningMessages, messages.New("The routing rule no longer exists; contact your administrator"))
	}
	return viewJSON(access, http.StatusOK, response)
}

func accountScope(headers http.Header) (string, bool) {
	values := headers.Values("Authorization")
	if len(values) != 1 {
		return "", false
	}
	parts := strings.Fields(values[0])
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || len(parts[1]) > 8192 {
		return "", false
	}
	scope := billing.CallerScope(parts[1])
	return scope, scope != ""
}

func apiKeyUnauthorized() ManagementResponse {
	response := apiKeyJSONError(http.StatusUnauthorized, "unauthorized", "Invalid API key")
	response.Headers.Set("WWW-Authenticate", `Bearer realm="cpa-key-billing-account"`)
	return response
}

func apiKeyJSON(status int, payload any) ManagementResponse {
	response := JSONResponse(status, payload)
	secureAPIKeyResponse(&response)
	return response
}

func apiKeyJSONError(status int, code, message string) ManagementResponse {
	response := JSONError(status, code, message)
	secureAPIKeyResponse(&response)
	return response
}

func secureAPIKeyResponse(response *ManagementResponse) {
	response.Headers.Set("Cache-Control", "private, no-store")
	response.Headers.Set("Pragma", "no-cache")
	response.Headers.Set("Vary", "Authorization")
	response.Headers.Set("Referrer-Policy", "no-referrer")
	response.Headers.Set("X-Content-Type-Options", "nosniff")
}
