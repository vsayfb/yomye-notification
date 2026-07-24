# CLAUDE.md

## Purpose

You are implementing an **existing Notification Service** for a distributed Go backend. Do **not redesign the architecture**. The architecture has already been decided. Your responsibility is to scaffold and implement it cleanly, idiomatically, and in an extensible manner.

---

# Existing Ecosystem

- Go
- PostgreSQL
- AWS Lambda
- AWS SQS
- Firebase Cloud Messaging (FCM)
- Clean Architecture
- Dependency Injection
- Repository pattern

There are multiple business services (Chat, Gig, etc.).

The Notification Service is an independent service.

Business services publish **business events** to SQS.

Notification Service:

1. Consumes SQS events.
2. Renders notifications.
3. Persists notifications.
4. Looks up recipient device tokens.
5. Pushes notifications via Firebase.

Notification Service owns presentation.

Producer services never own notification wording.

---

# Philosophy

Producer services publish facts.

Good:

```go
type NewMessageNotificationEvent struct {
    RecipientID    string `json:"recipient_id"`
    SenderID       string `json:"sender_id"`
    SenderName     string `json:"sender_name"`
    ThreadID       string `json:"thread_id"`
    GigID          string `json:"gig_id"`
    MessageID      string `json:"message_id"`
    MessagePreview string `json:"message_preview"`
}
```

Bad:

```go
Title: "Alice sent you a message"
Body: "Hello!"
```

Rendering belongs only to Notification Service.

---

# Notification Table

```sql
CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    actor_id UUID REFERENCES users(id) ON DELETE SET NULL,

    type TEXT NOT NULL,

    entity_type TEXT NOT NULL,
    entity_id UUID NOT NULL,

    title TEXT NOT NULL,
    body TEXT NOT NULL,

    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,

    read_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

Model should mirror this schema.

Metadata:

```go
map[string]any
```

---

# Package Structure

```text
internal/

    config/

    handler/

    service/
        service.go
        dispatcher.go

    notification/
        notification.go
        repository.go
        push.go
        types.go

    renderer/
        renderer.go
        new_message.go

    repository/
        postgres/
        firebase/

    event/

    db/
```

Do not invent unnecessary packages.

---

# Flow

```text
SQS

↓

Handler

↓

NotificationService

↓

Dispatcher

↓

Renderer

↓

Notification

↓

Persist

↓

Lookup FCM Tokens

↓

Push
```

---

# Handler

Handler only:

- receive SQS event
- unmarshal envelope
- call service

No SQL.

No Firebase.

No rendering.

---

# Events

Every notification has its own event DTO.

Future events:

- new_message
- gig_approved
- application_received
- application_accepted
- application_rejected
- message_request_accepted
- review_received

Architecture must support adding them with minimal changes.

---

# Dispatcher

Dispatcher owns:

```go
map[NotificationType]Renderer
```

One registration per notification type.

Adding a notification should require:

1. Event DTO
2. Renderer
3. Registration

Nothing else.

Avoid a giant switch throughout the codebase.

---

# Renderer

Each notification has its own renderer.

Renderer converts an event into BOTH:

- Notification (database)
- PushNotification (mobile)

Interface:

```go
type Renderer interface {
    Render(ctx context.Context, payload json.RawMessage) (
        *notification.Notification,
        *notification.PushNotification,
        error,
    )
}
```

Renderer contains notification-specific business logic.

Repositories should not.

---

# PushNotification

Create an internal abstraction.

```go
type PushNotification struct {
    RecipientID string

    Title string
    Body string

    Data map[string]string
}
```

This is NOT an FCM model.

Firebase adapter translates it.

---

# Repository Interfaces

```go
type NotificationRepository interface {
    Create(ctx context.Context, n *Notification) error
}
```

```go
type DeviceTokenRepository interface {
    Tokens(ctx context.Context, userID string) ([]string, error)
}
```

Implement PostgreSQL versions.

---

# Push Provider

```go
type PushProvider interface {
    Send(ctx context.Context, tokens []string, push *PushNotification) error
}
```

Implement Firebase adapter.

Service depends only on interface.

---

# Notification Service Responsibilities

Service coordinates.

Pseudo flow:

1. Dispatch event.
2. Render notification.
3. Persist notification.
4. Lookup recipient device tokens.
5. Send push.

Business logic should remain inside renderers.

---

# First Notification Type

Event:

```go
type NewMessageNotificationEvent struct {
    RecipientID    string `json:"recipient_id"`
    SenderID       string `json:"sender_id"`
    SenderName     string `json:"sender_name"`
    ThreadID       string `json:"thread_id"`
    GigID          string `json:"gig_id"`
    MessageID      string `json:"message_id"`
    MessagePreview string `json:"message_preview"`
}
```

Persist:

- UserID = RecipientID
- ActorID = SenderID
- Type = "message_received"
- EntityType = "thread"
- EntityID = ThreadID
- Title = SenderName
- Body = MessagePreview
- Metadata:
  - gig_id
  - message_id

Push:

Title:

```
SenderName
```

Body:

```
MessagePreview
```

Data:

```json
{
  "type":"message_received",
  "thread_id":"...",
  "gig_id":"...",
  "message_id":"..."
}
```

---

# Device Tokens

Producer services must NOT send FCM tokens.

Notification Service owns device token lookup.

---

# Coding Guidelines

- Idiomatic Go.
- Small files.
- Dependency Injection.
- Interfaces near consumers.
- No globals.
- No God objects.
- Keep repositories persistence-only.
- Keep handler transport-only.
- Keep renderer transformation-only.
- Keep service orchestration-only.

---

# Deliverables

Generate a complete compile-ready scaffold including:

- models
- interfaces
- dispatcher
- renderer abstraction
- new message renderer
- service
- repositories
- firebase provider
- lambda handler
- dependency wiring

Assume infrastructure already exists.

Implement architecture exactly as described. Do not redesign it.
