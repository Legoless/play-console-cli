# Monetization

Create and manage **subscriptions**, **in-app products (IAP)**, and **one-time purchases** on Google Play from the command line — no clicking through Play Console. Works standalone or as the Play-side companion to [RevenueCat](https://www.revenuecat.com/).

## In-app products (IAP)

```bash
gplay iap list --package com.example.app
gplay iap create --package com.example.app --sku premium_upgrade --json @product.json
gplay iap update --package com.example.app --sku premium_upgrade --json @product.json
gplay iap batch-update --package com.example.app --json @products.json
```

## One-time products

```bash
# Purchase options (batch state management)
gplay purchase-options batch-update-states --package com.example.app --json @states.json
gplay purchase-options batch-delete --package com.example.app --json @options.json

# One-time product offers
gplay otp-offers list --package com.example.app --product-id lifetime
gplay otp-offers activate --package com.example.app --product-id lifetime --offer <id>
gplay otp-offers deactivate --package com.example.app --product-id lifetime --offer <id>
```

## Subscriptions

```bash
gplay subscriptions list --package com.example.app
gplay subscriptions create --package com.example.app --json @subscription.json

# Base plans
gplay baseplans activate --package com.example.app --product-id sub_premium --base-plan monthly
gplay baseplans deactivate --package com.example.app --product-id sub_premium --base-plan monthly

# Promotional offers
gplay offers list --package com.example.app --product-id sub_premium --base-plan monthly
gplay offers create --package com.example.app --product-id sub_premium --base-plan monthly --json @offer.json
```

## Regional pricing

```bash
# Convert a base price across all Play regions
gplay pricing convert --package com.example.app --json @price.json
```

## Purchase verification & orders

Server-side verification of purchase tokens, acknowledgement, and refunds:

```bash
# Verify purchases
gplay purchases products get --package com.example.app --product-id premium --token <token>
gplay purchases products acknowledge --package com.example.app --product-id premium --token <token>
gplay purchases subscriptions get --package com.example.app --token <token>

# Orders
gplay orders get --package com.example.app --order-id <id>
gplay orders refund --package com.example.app --order-id <id> --revoke

# External transactions (EU compliance)
gplay external-transactions create --package com.example.app --json @transaction.json
```

## Real-Time Developer Notifications (RTDN)

Set up and decode Play billing webhooks:

```bash
gplay rtdn setup --project <gcp-project> --topic play-rtdn
gplay rtdn status --project <gcp-project>
gplay rtdn decode --file payload.json      # typed subscription/one-time/voided decoder
```

## Using gplay with RevenueCat

RevenueCat imports products from Google Play — it doesn't create them. `gplay` covers the Play-side half of the workflow:

1. **Create products in Play** with `gplay subscriptions create` / `gplay iap create` (scriptable, repeatable across apps).
2. **Import them into RevenueCat** (dashboard, API, or the RevenueCat MCP/CLI tooling) and attach them to entitlements and offerings.
3. **Verify and debug** live purchases from the terminal with `gplay purchases ...` and `gplay orders ...` when reconciling RevenueCat events against Play.
4. **Decode RTDN payloads** with `gplay rtdn decode` when debugging RevenueCat's Play Store server notifications.

Because both product catalogs are scriptable, an AI agent can set up the full monetization stack — Play products + RevenueCat offerings — in one session.
