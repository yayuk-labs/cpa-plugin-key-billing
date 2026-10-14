function renderAccountProfile() {
  const profile = resources.account.profile.value;
  const tracked = profile?.tracked === true;
  $("account-untracked").classList.toggle("hidden", !profile || tracked);
  $("account-content").classList.toggle("hidden", !tracked);
  if (tracked) $("account-identity").replaceChildren(displayText(apiKeyIdentity(profile.identity || {}).text));
  else closeSharedTimePopover(true);
  return tracked;
}

function renderAccountSubscription() {
  const data = resources.account.subscription;
  const subscription = data.value?.subscription || {};
  const concurrency = data.value?.concurrency || {};
  setText($("account-concurrency"), !data.value ? "—" : int(concurrency.current));
  setText(
    $("account-concurrency-limit"),
    !data.value ? "—" : Number(concurrency.limit || 0) > 0 ? int(concurrency.limit) : m("ui.unlimited")
  );
  setText(
    $("account-plan-name"),
    !data.value
      ? m("ui.subscription_quota")
      : displayLabel(subscription.name || (subscription.unlimited ? m("ui.no_subscription_plan") : m("ui.subscription_quota")))
  );
  $("account-plan-status").className = "tag " + (subscription.blocked ? "bad" : "plain");
  setText(
    $("account-plan-status"),
    !data.value ? "" : subscription.unlimited ? m("ui.unlimited") : subscription.blocked ? quotaBlockedLabel(subscription) : m("ui.active")
  );
  const target = $("account-windows");
  if (renderDataPending(target, [data], m("ui.loading_subscription_status"))) return;
  target.replaceChildren(renderAccountQuotaWindows(subscription));
}

function renderAccountQuotaWindows(view) {
  if (view.unlimited) return el("p", { class: "account-quota-unlimited", text: m("ui.no_subscription_quota_limit") });
  return el(
    "div",
    { class: "account-quota-windows" },
    view.windows.map((window) =>
      el(
        "section",
        { class: "account-quota-window", "aria-label": window.name },
        el(
          "div",
          { class: "row spread account-quota-window-heading" },
          el("h3", { text: displayLabel(window.name) }),
          el("span", {
            class: "muted account-quota-reset",
            text: window.started ? m("ui.resets_at_value", { v0: quotaWindowResetLabel(window, true) }) : m("ui.not_started"),
            "data-tip": window.started ? when(window.end_at) : null
          })
        ),
        el(
          "div",
          { class: "account-quota-balances" },
          window.dimensions.map((balance) => {
            const metric = QUOTA_DIMENSIONS.find((item) => item.metric === balance.metric);
            const quota = remainingQuota(balance.used_percent);
            const label = m("ui.metric_remaining", { v0: metric.label });
            const used = m("ui.used_of_limit", { v0: metric.format(balance.used), v1: metric.format(balance.limit) });
            return el(
              "div",
              { class: "account-quota-balance" + (balance.blocked ? " blocked" : "") },
              el("dl", {}, el("dt", { text: label }), el("dd", { class: "account-quota-value", text: metric.format(balance.remaining) })),
              el("p", { class: "account-quota-used", text: used }),
              el(
                "div",
                {
                  class: "bar",
                  role: "progressbar",
                  "aria-label": m("ui.valuevaluevalue_remaining_quota", { v0: window.name, v1: DISPLAY_SEPARATOR, v2: metric.label }),
                  "aria-valuemin": "0",
                  "aria-valuemax": "100",
                  "aria-valuenow": String(Math.round(quota.percent)),
                  "aria-valuetext": metric.exact(balance.remaining) + DISPLAY_SEPARATOR + label + DISPLAY_SEPARATOR + used
                },
                el("i", { class: "quota-fill " + quota.level, style: "width:" + quota.percent + "%" })
              ),
              balance.blocked ? el("span", { class: "tag bad", text: m("ui.quota_exhausted") }) : null
            );
          })
        ),
        window.models?.length
          ? settingsChipList(window.models.map((text) => ({ text })))
          : el("p", { class: "muted account-quota-scope", text: m("quota.other_models") })
      )
    )
  );
}

function renderAccountModels() {
  const data = resources.account;
  const target = $("account-models");
  if (renderDataPending(target, [data.models, data.catalog, data.routing], m("ui.loading_model_prices"))) return;
  const access = data.routing.value;
  const directory = data.models.value;
  const modelScope = new Set(access.models.map((model) => model.toLowerCase()));
  const deniedModels = new Set((access.denied_models || []).map((model) => model.toLowerCase()));
  const models = (
    access.routing_valid === false
      ? []
      : !modelScope.size ? directory : directory.length ? directory.filter((model) => modelScope.has(model.toLowerCase())) : access.models
  )
    .filter((model) => !deniedModels.has(model.toLowerCase()))
    .slice()
    .sort(compareModelId);
  renderCollection(target, models, { attrs: { class: "display-contents" }, empty: m("ui.no_model_prices"), render: renderAccountModel });
}

function renderAccountRouting() {
  const data = resources.account.routing;
  const target = $("account-access");
  if (renderDataPending(target, [data], m("ui.loading_routing_permissions"))) return;
  target.replaceChildren(renderAccountAccess(data.value));
}

function renderAccountAccess(access) {
  const routingValid = access.routing_valid !== false;
  const deniedModels = new Set((access.denied_models || []).map((model) => model.toLowerCase()));
  const modelItems = access.models.map((text) => ({
    text,
    state: deniedModels.has(text.toLowerCase()) ? "deny" : "allow",
    iconName: "astroid"
  }));
  if (!modelItems.length) modelItems.push({ text: routingValid ? m("ui.all_models") : m("ui.unknown"), iconName: "astroid" });
  for (const text of access.denied_models || [])
    if (!access.models.some((model) => model.toLowerCase() === text.toLowerCase()))
      modelItems.push({ text, state: "deny", iconName: "astroid" });
  const nodes = [
    el(
      "div",
      { class: "account-access-summary" },
      el(
        "div",
        { class: "inset-card" },
        el("div", { class: "account-access-scope-label", text: m("ui.model_access") }),
        settingsChipList(modelItems)
      )
    )
  ];
  if (access.warnings?.length)
    nodes.push(
      el(
        "div",
        { class: "account-access-warnings" },
        access.warnings.map((warning, index) =>
          el("span", { class: "tag warn", text: window.billingI18n.serverMessage(access.warning_messages?.[index], warning) })
        )
      )
    );
  return el("div", { class: "display-contents" }, nodes);
}

function renderAccountModel(model) {
  const candidate = resources.account.catalog.value.prices.find((price) => price.model_id.toLowerCase() === model.toLowerCase());
  const price = candidate?.source === "none" ? null : candidate;
  const fields = [[m("ui.input"), "input_per_1m"], [m("ui.output"), "output_per_1m"]];
  const longContext = price?.long_context;
  for (const [label, field] of [[m("ui.cache_read"), "cache_read_per_1m"], [m("ui.cache_write"), "cache_write_per_1m"]]) {
    if (price?.[field] != null || longContext?.[field] != null) fields.push([label, field]);
  }
  const threshold = longContext ? ">" + tokenThreshold(longContext.threshold_input_tokens) : "";
  const priceRows = price
    ? fields.map(([label, field]) => ({
        label,
        base: price[field] == null ? null : Number(price[field]),
        long: longContext?.[field] == null ? null : Number(longContext[field])
      }))
    : [];
  const priceRow = (item) =>
    el(
      "div",
      { class: "account-model-price-row" + (longContext ? " tiered" : "") },
      el("span", { class: "account-model-price-label" }, item.label),
      el("strong", { class: "mono account-model-base-price" }, item.base == null ? "—" : usd(item.base)),
      longContext
        ? el(
            "span",
            { class: "account-model-long-price" },
            item.long != null
              ? [el("span", { class: "account-model-threshold" }, threshold), el("strong", { class: "mono" }, usd(item.long))]
              : el("span", { class: "muted" }, "—")
          )
        : null
    );
  return el(
    "div",
    { class: "account-model-card" },
    el("b", { class: "mono account-model-name", "data-tip": model, "data-tip-overflow": "" }, model),
    price
      ? el("div", { class: "account-model-price-list" }, priceRows.map(priceRow))
      : el("span", { class: "muted small", text: m("ui.no_price_configured_currently_unavailable") })
  );
}
