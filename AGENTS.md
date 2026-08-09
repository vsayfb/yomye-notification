# Repository instructions for Codex

## First rule: secrets and user work

- Never read, print, search, copy, summarize, source, or inspect `.env`, `.env.*`, Firebase credential JSON, private keys, FCM registration tokens, or secret values.
- It is acceptable to tell the user which variable, SSM parameter, or secret must be configured. Do not expose its value.
- FCM tokens are credentials. Never include complete tokens in logs, errors, tests, examples, or replies. Log a device index or count instead.
- Preserve unrelated user changes in a dirty worktree. Inspect `git status` before editing.
- Do not commit or push unless explicitly requested. A request to push must identify or clearly authorize the configured remote.
- Use `apply_patch` for edits and `rg`/`rg --files` for discovery.

## What this project is

This repository is Yövmiye's Go notification service. It runs as an AWS Lambda, consumes business-event envelopes from SQS, renders semantic notifications, persists the in-app inbox row in PostgreSQL, resolves FCM device tokens, and sends push notifications through Firebase Cloud Messaging.

Treat this file plus the current implementation and current sibling-service migrations as the source of truth. `CLAUDE.md` is an older architectural brief and contains stale examples (including older schema/type assumptions); preserve its useful architectural intent, but do not override newer contracts documented here with it.

The service is an independent consumer in a distributed backend. Nearby repositories may be available as siblings:

- `../worker`: categorization worker and producer of listing/category events.
- `../core-service`: shared schema owner and HTTP API that returns persisted notifications and registers FCM tokens.
- `../chat-service`: chat event producer.

Inspect the relevant sibling implementation when a cross-service contract is unclear, but only edit this repository unless the user explicitly expands scope.

The current launch scope is gig/service-only. Generic `listings` are canonical, but Yövmiye does not currently expose for-sale behavior. Do not add or re-enable `for-sale` notification behavior without an explicit product change.

## Architecture is intentional

Preserve this flow:

```text
SQS batch
  -> handler (transport only)
  -> service orchestration
  -> optional fan-out resolver
  -> event-specific renderer
  -> notification persistence
  -> FCM token lookup
  -> Firebase push
  -> pushed_at update
```

Responsibilities:

- `internal/handler`: unmarshal SQS envelopes and invoke the service. No SQL, Firebase, or wording.
- `internal/service`: orchestration, fan-out, retry-relevant outcomes, stale-token cleanup.
- `internal/service/dispatcher.go`: renderer and optional resolver registrations.
- `internal/event`: inbound and synthesized event DTOs.
- `internal/renderer`: event-specific validation and conversion into persisted and push models.
- `internal/notification`: domain models and shared semantic payload construction.
- `internal/repository/postgres`: persistence adapters only.
- `internal/repository/firebase`: Firebase delivery adapter and per-device outcome classification.
- `internal/config`: environment-specific configuration loading.
- `pkg/fb`: Firebase Admin SDK wrapper and structured error diagnostics.
- `internal/bootstrap`: dependency wiring and dispatcher registration.

Do not move event-specific switches into the handler or repositories. Adding a normal notification type should usually require an event DTO, renderer, constants/catalog key, dispatcher registration, and tests. Add a resolver only when recipients must be computed at consume time.

## Event envelope and current events

Every SQS body uses:

```json
{
  "type": "event_type",
  "payload": {}
}
```

Registered event types:

- `new_message`
- `listing_approved`
- `gig_category_matched`

`listing_approved` is canonical. Do not restore `gig_approved`.

Worker-originated events (`listing_approved` and `gig_category_matched`) use a versioned envelope with deterministic `event_id`. Version 1 is currently supported. The ID belongs to the envelope and is the domain-event idempotency key; SQS message ID is logging/transport metadata only. A PostgreSQL claim is acquired before work, processed duplicates are acknowledged, active duplicates remain retryable, and `processed_at` is written only after all required work succeeds.

### listing_approved

Expected producer contract:

```json
{
  "type": "listing_approved",
  "payload": {
    "recipient_id": "user-uuid",
    "listing_id": "listing-uuid",
    "base_category_slug": "gig",
    "title": "Bahçe Bakımı",
    "occurred_at": "2026-08-02T00:00:00Z"
  }
}
```

Rules:

- `listing_id` is the canonical entity identifier.
- Persist and route with `entity_type = "listing"` and `entity_id = listing_id`.
- Only `base_category_slug = "gig"` is accepted. The slug is a static listing-family discriminator, not category identity.
- Lambda does not query listing details for approval rendering. The worker publishes only after classified typed details and title are persisted and the listing is opened.
- Trim the event title. A usable title uses `notifications.listing_approved` with `{"title": "..."}`.
- A missing/empty/whitespace title must use `notifications.listing_approved_generic` with `{}`. Never send `{ "title": "" }`.

### gig_category_matched

Expected producer contract:

```json
{
  "type": "gig_category_matched",
  "payload": {
    "gig_id": "listing-uuid",
    "category_id": "category-uuid",
    "country_id": "optional-country-uuid",
    "place_id": "optional-place-uuid",
    "occurred_at": "2026-08-02T00:00:00Z"
  }
}
```

Despite the legacy field name `gig_id`, its value is the canonical listing UUID. The resulting notification target is `entity_type = "listing"`.

Location rules:

- `country_id` and `place_id` are optional UUIDs.
- `place_id` requires `country_id`.
- Nil UUIDs are invalid when a field is present.
- Never translate UUID locations into legacy integer city/district IDs.
- Never emit `city_id` or `district_id`.
- Legacy events containing those fields are rejected explicitly; silently treating them as locationless would misroute notifications.
- No country matches locationless/remote users.
- Country-only matches subscribers throughout that country.
- Country plus place matches the exact opaque place UUID. A place is not assumed to be a city or district.

This event fans out live by joining `user_categories.category_id` and current user location. Category identity remains only the UUID.

## Category-domain boundaries

The global category taxonomy is owned by the worker. This Lambda is read-only.

- Category UUID is the sole identity. Never join or deduplicate by category name, slug, locale, or localization.
- Canonical English fields are `canonical_name`, `canonical_slug`, and `canonical_description`.
- Localized presentation belongs to `category_localizations`.
- This service currently does not read or present category labels, build breadcrumbs, cache category documents, or index categories. Therefore it does not need locale fallback queries.
- `gig_category_matched` carries only `category_id`; notification wording is category-neutral.
- Do not create categories/localizations or infer `is_selectable` from hierarchy. The worker owns category mutation and assignment invariants.

## Semantic, locale-neutral notification contract

Lambda does not know or select the user's UI locale. Flutter localizes using its active catalog, with bundled Turkish fallback.

Every FCM data payload must contain:

```json
{
  "type": "listing_approved",
  "entity_type": "listing",
  "entity_id": "listing-uuid",
  "localization_key": "notifications.listing_approved",
  "localization_args": "{\"title\":\"Bahçe Bakımı\"}"
}
```

Important details:

- FCM data values must be strings, so `localization_args` is a JSON-encoded object string.
- Persisted notification metadata stores `localization_key` plus `localization_args` as an actual JSON object. Core returns `metadata` unchanged from its notifications endpoint.
- Persisted `type`, `entity_type`, and `entity_id` columns complete the semantic inbox contract.
- `type` controls client behavior; `entity_type`/`entity_id` control navigation; localization keys control catalog lookup.
- Additional routing values are allowed only when genuinely needed. The shared builder forbids overriding reserved semantic fields.
- Legacy English `title` and `body` remain temporarily in the FCM notification/display payload and database columns for old clients. They are not authoritative for modern clients.
- Build semantic pushes through `notification.NewSemanticPush`; persist localization data through `notification.AddLocalizationMetadata`. Use the same key and argument map for both.

Current catalog keys:

- `notifications.listing_approved`
- `notifications.listing_approved_generic`
- `notifications.message_received`
- `notifications.gig_category_matched`

When changing a key or argument name, treat it as a Flutter contract change. Argument spelling is exact; approval uses `title`, and message received uses `name`.

## Persistence, idempotency, and delivery trade-offs

The notification row is created before attempting push delivery. This is deliberate: the in-app inbox remains durable even when there is no token or FCM is unavailable.

The database deduplication key is:

```text
(user_id, entity_type, entity_id, type)
```

`Create` returns either the new row or the existing conflict row. `pushed_at` prevents an SQS redelivery from pushing an already completed notification again.

Worker-event delivery is deduplicated independently by deterministic envelope `event_id` in `notification_processed_events`. That prevents re-executing the same logical worker event, including concurrent duplicates. Claims have a five-minute lease so a crashed invocation does not block the event forever.

Known limitation: the notification row's own unique key is still not a source-event ID. Two distinct legitimate events with the same user/entity/type can collapse even though their envelope IDs differ. `new_message` events in the same thread are the clearest example, and chat has not adopted the worker's versioned envelope contract. Do not "fix" this with ad hoc metadata comparisons; adding `source_event_id` to notification persistence requires a coordinated Core schema/API change.

`pushed_at` currently means push processing completed without a retryable error. It may be set when invalid/unregistered tokens were permanently resolved, even if no device received the push. Do not present it as a delivery receipt.

If the user has no FCM tokens, the inbox row remains persisted and `pushed_at` remains null, while the SQS record succeeds. The service does not replay old notifications automatically when a client later registers a token.

The SQS handler processes up to five records concurrently. It currently returns one aggregate error if any record fails, which makes Lambda/SQS retry the whole batch. Successful rows are normally protected by `pushed_at`, but partial batch response support would be a cleaner future improvement. Do not assume record-level acknowledgement exists.

## FCM error handling and token lifecycle

The client must register/upsert its current token on login/startup and after Firebase token changes. Lambda owns cleanup when Firebase proves a token is permanently invalid.

Rules:

- Delete a token only when the original Firebase Admin SDK error is structurally classified as `UNREGISTERED` via `messaging.IsUnregistered`.
- Do not delete for generic errors, HTTP 404 alone, `INVALID_ARGUMENT` alone, timeouts, quota failures, network errors, or message text such as `NotRegistered`.
- Unregistered tokens are deleted and treated as permanently resolved so they do not poison SQS retries.
- Transient errors retain the token and fail the record for retry.
- Mixed delivery may send successfully to some devices, delete unregistered devices, and retry only when another device has a transient failure. Successful devices can still see duplicates on a mixed transient retry because per-token delivery state is not persisted.
- Log structured Firebase diagnostics (`http_status`, platform status, FCM error code, complete SDK message) without token values.

The Firebase SDK's public classifier expects its exact internal error type and does not recognize a contextual `fmt.Errorf("...: %w", err)` wrapper. `pkg/fb` captures classification before wrapping and preserves structured diagnostics. It currently reflects exported fields from the SDK's internal error because the SDK exposes no public diagnostic accessor. Treat Firebase SDK upgrades as requiring regression tests of this adapter.

## Configuration and deployment

`APP_ENV` routing:

- empty or `development`: local environment variables plus `FIREBASE_CREDENTIALS_PATH`.
- `staging`: AWS configuration.
- `production`: GCP configuration.
- any other value: explicit error.

AWS deployment must set `APP_ENV=staging` in the Lambda runtime configuration. Merely selecting the GitHub Actions `staging` environment does not automatically set the Lambda environment variable.

AWS staging configuration:

- Parameter Store prefix: `/yevmiye/staging/`
- `db-host`
- `db-port`
- `db-name`
- `firebase-credentials` (SecureString containing the full service-account JSON)
- `rds-secret-arn`
- RDS username/password are loaded from Secrets Manager.
- SSM uses `WithDecryption=true`.
- In-memory Firebase credentials are selected by presence, not by environment-name hacks.

The Lambda role needs appropriate SSM, KMS, and Secrets Manager permissions. A VPC-hosted Lambda also needs network access. SSM/Secrets Manager interface endpoints can reach AWS APIs, but Firebase OAuth/FCM requires outbound internet/NAT.

Production GCP configuration is deliberately not implemented yet. `loadGCP` returns a clear error rather than a nil config. Do not claim production support until DB and Firebase credential loading are implemented for GCP.

The GitHub workflow currently deploys manually (`workflow_dispatch`) to the GitHub `staging` environment, uploads a zip to S3, updates the Lambda, and waits for the update. Verify repository/environment variables and the actual Lambda `APP_ENV` separately.

## Shared database contracts

This service writes shared Core-owned tables:

- `notifications`
- `fcm_tokens`

It reads:

- `users`
- `user_categories`

Core owns the public notification API and returns `metadata` to Flutter. Before changing columns or query assumptions, inspect current Core migrations and notification repository. Do not use the old `gigs` compatibility view in new code.

Relevant notification columns include:

- `type`
- `entity_type`
- `entity_id` (text in the current shared schema)
- legacy `title`/`body`
- `metadata` JSONB
- `read_at`
- `pushed_at`
- timestamps

The checked-in Lambda migration may lag the complete Core-owned schema. Treat the live/Core migration as authoritative for shared tables and coordinate any schema change rather than silently diverging.

## Logging and operational diagnosis

- Use structured `slog` fields.
- Never log secret contents or full FCM tokens.
- `Device unregistered` means Firebase was reached and rejected that token; it is not a Firebase credential-loading failure.
- `dispatcher: no renderer registered` almost always indicates stale deployed code/version/alias when local bootstrap registration exists.
- LocalStack `errResetReceived` and `Reset initiated: Timeout` are consequences of invocation timeout, not application error causes. Inspect earlier record logs and external call deadlines.
- A persisted notification without browser delivery is expected when no token exists or Firebase permanently rejects all tokens. Inspect structured FCM logs and token rows.
- `pushed_at` is not proof that Chrome displayed a notification.
- If duplicate pushes appear, inspect whether one device succeeded while another retryable/permanently misclassified device caused whole-record retries.

## Testing and verification

The Go binaries are installed under `/usr/local/go/bin` and may not be on `PATH`. The default Go build cache is read-only in this environment, so use a writable cache:

```bash
/usr/local/go/bin/gofmt -w <changed-go-files>
GOCACHE=/tmp/notification-lambda-go-cache /usr/local/go/bin/go test ./...
GOCACHE=/tmp/notification-lambda-go-cache /usr/local/go/bin/go build -o /tmp/notification-lambda-build ./cmd/notification
git diff --check
git status --short --branch
```

Do not redirect `GOMODCACHE` to a fresh directory unless dependency downloads are authorized; network access may be unavailable. The Go command may emit a harmless read-only module stat-cache warning after otherwise successful tests/builds.

Tests should cover contracts, not only implementation:

- exact event JSON fields and optional-field invariants;
- exact semantic localization keys and argument names;
- JSON-encoded FCM `localization_args`;
- persisted metadata localization object;
- canonical entity type/id navigation;
- unregistered versus transient FCM outcomes;
- reserved routing-field protection;
- environment routing and credential source selection.

## Working style for future agents

- Lead with evidence. Inspect the event producer, Core schema/API consumer, and this Lambda when a cross-service issue spans them.
- Distinguish diagnosis from implementation. Do not mutate external infrastructure or sibling repositories merely because it would help.
- Preserve architecture and existing user changes; make focused patches and proportionate tests.
- Prefer explicit validation over silent fallback when fallback can misroute a notification.
- Do not invent missing product contracts. If a new chat event cannot be distinguished as direct versus application conversation from its facts, request/establish the producer contract instead of guessing.
- When a deployment log contradicts local code, verify artifact/version/alias/runtime configuration before rewriting working code.
- After changing a client-facing payload, provide the exact FCM and persisted shapes the Flutter team must support.

## Deliberate deferred work

Do not casually fold these into unrelated changes:

- GCP production configuration.
- SQS partial batch failure responses.
- A `source_event_id` on notification rows replacing the coarse inbox unique constraint, coordinated with Core.
- Per-token delivery state that prevents duplicate delivery to successful devices during mixed transient failures.
- Removal of legacy English title/body after client migration is complete.
- Backfilling old notification metadata with semantic localization fields.
