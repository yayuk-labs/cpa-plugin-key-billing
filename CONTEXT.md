# Subscription Billing

Subscription plans assign recurring usage budgets to API keys.

## Language

**Quota window**:
A shared budget for a set of models over a recurring period, with amount, token, or request limits.

**Model selection**:
The client-visible model IDs explicitly covered by a quota window. Selected models share that window's budget.

**Fallback window**:
A quota window with no model selection, covering all models not explicitly selected by another window in the plan.
