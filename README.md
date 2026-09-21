# Yömye Notification Service

The notification service turns backend events into durable in-app notifications and Firebase Cloud Messaging (FCM) pushes.

It has two deployment entrypoints:

- **Staging:** AWS Lambda consumes batches from Amazon SQS.
- **Production:** Google Cloud Run receives authenticated Google Cloud Pub/Sub push requests.

Both entrypoints call the same notification pipeline. Transport code only decodes the event and reports whether it should be retried.

## How it works

For each supported event, the service:

1. Validates the event envelope and payload.
2. Claims the event in PostgreSQL to prevent duplicate processing.
3. Finds the registered renderer and, when needed, resolves recipients.
4. Builds locale-neutral notification metadata.
5. Saves the notification in PostgreSQL before attempting push delivery.
6. Loads the recipient's FCM tokens and sends the push through Firebase.
7. Removes tokens only when Firebase classifies them as `UNREGISTERED`.
8. Marks push processing as complete when no retryable delivery error remains.

Saving the inbox row first means users can still see a notification when push delivery is unavailable or when no device token is registered.

## Supported events

| Event | Producer | Purpose |
| --- | --- | --- |
| `new_message` | Chat | Notifies one recipient about a new direct or listing conversation message. |
| `listing_approved` | Worker | Tells a listing owner that classification finished and the listing is open. |
| `gig_category_matched` | Worker | Finds users subscribed to the matched profession and location, then creates a notification for each user. |

Adding a normal event generally requires an event DTO, renderer, localization key, dispatcher registration, and contract tests. A recipient resolver is needed only for fan-out events.

## Repository structure

```text
cmd/notification             AWS Lambda entrypoint
cmd/cloudrun                 Cloud Run HTTP entrypoint
internal/handler             SQS and Pub/Sub transport adapters
internal/service             orchestration, fan-out, retries, and token cleanup
internal/renderer            event validation and notification rendering
internal/repository/postgres notification, token, subscriber, and event-claim storage
internal/repository/firebase FCM delivery and per-device outcome handling
internal/config              development, AWS staging, and GCP production config
internal/bootstrap           dependency and dispatcher wiring
pkg/fb                       Firebase Admin SDK adapter
migrations                   local reference for Core-owned shared tables
tools                        GitHub environment configuration scripts
```

## Local development

Requirements:

- Go 1.26.4 or a compatible newer Go release
- PostgreSQL with the shared Core schema applied
- Firebase service-account credentials for a development project
- An OpenTelemetry collector when telemetry export is required

The service does not load `.env` files itself. Environment values must be exported by the shell or supplied by the local development environment.

Required development variables:

```text
APP_ENV=development
DB_HOST
DB_PORT
DB_USER
DB_PASSWORD
DB_NAME
FIREBASE_CREDENTIALS_PATH
```

`OTEL_COLLECTOR_ADDR` is optional and defaults to `localhost:4317`. Development database SSL defaults to `disable`.

Run the Cloud Run-compatible HTTP entrypoint locally:

```bash
go run ./cmd/cloudrun
```

It listens on `PORT`, or port `8080` when `PORT` is not set. `GET /health` returns the process health status. The root endpoint accepts Pub/Sub push envelopes with `POST`.

Run verification:

```bash
go test ./...
go build ./cmd/notification
go build ./cmd/cloudrun
git diff --check
```

Build the AWS Lambda archive:

```bash
make build
```

Remove Lambda build artifacts:

```bash
make clean
```

## Configuration by environment

`APP_ENV` selects the configuration source:

| Value | Runtime | Configuration source |
| --- | --- | --- |
| empty or `development` | Local HTTP process | Environment variables and `FIREBASE_CREDENTIALS_PATH` |
| `staging` | AWS Lambda | Systems Manager Parameter Store and Secrets Manager |
| `production` | Google Cloud Run | Parameter Manager, Secret Manager, and Application Default Credentials |

The cloud environments provide database and Firebase settings through infrastructure-managed parameters and secrets. Static cloud credentials are not stored in GitHub Actions.

## Deployment

Both deployments are manual GitHub Actions workflows:

- `.github/workflows/staging.yml` builds the Lambda archive, uploads it to S3, sets `APP_ENV=staging`, and updates the AWS Lambda function.
- `.github/workflows/production.yml` builds a container, pushes it to Artifact Registry, and updates the GCP Cloud Run service.

The workflows authenticate through GitHub OIDC. Their environment variables can be populated after Terraform is applied by using the scripts documented in [`tools/README.md`](tools/README.md).

Cloud resources are managed in the [Yömye infrastructure repository](https://github.com/vsayfb/yomye-infrastructure).

## Shared database ownership

Core owns the shared PostgreSQL schema and public notification API. This service writes `notifications` and reads or updates related tables such as `fcm_tokens`, `users`, `user_categories`, and `notification_processed_events`.

The migrations in this repository are a local reference and may not contain the newest shared schema. Schema changes must be made in coordination with Core; Core's current migrations are authoritative.

## Delivery semantics

- Event processing is idempotent, but delivery is not a device-level exactly-once guarantee.
- `pushed_at` means push processing finished without a retryable error. It is not proof that a device displayed the notification.
- Unknown or malformed poison events are acknowledged so they do not retry forever.
- Database and transient provider failures remain retryable.
- A mixed result can deliver to one device and retry another, so a successful device may receive a duplicate during a later retry.
- Tokens are deleted only after Firebase returns the structured `UNREGISTERED` classification.

See [`AGENTS.md`](AGENTS.md) for detailed implementation contracts and maintenance rules.
