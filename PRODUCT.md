# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

API-key holders check their own quota, spending, usage trends, model prices, and model permissions. Administrators manage billing through the management panel.

## Product Purpose

Expose per-key billing and quota information within CLIProxyAPI's existing plugin capabilities.

## Capabilities and Constraints

The user usage page accepts API keys only. Request Events, Error Events, Auth Files, and upstream quota actions belong exclusively to administrators. User routing responses contain model permissions only, with no credential identities, selectors, or credential permissions. User usage distribution is model-only and excludes upstream source identities. Preserve existing billing history and quota semantics. Use the existing self-contained HTML, theme system, and English/Chinese translations.

## Product Principles

- Make current quota remaining and used explicit per independent window; keep historical spending separate.
- Enforce role boundaries at the API as well as in navigation.
- Show only usage supplied by the host.
