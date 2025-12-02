Yeah, this is a very sane impulse. “Google, but made of cardboard and duct tape” is exactly what you want for client dev.

Let’s treat this as building three *stateful emulators*:

* Same HTTP surface (URLs, methods, JSON shapes) for:

  * Google Calendar API
  * Google People API
  * Gmail API
* But backed by your own local/ephemeral store instead of Google.

---

## Overall shape: “Fake Google Box”

Think: one service, three modules:

* `fake-gmail`
* `fake-calendar`
* `fake-people`

Running as a single HTTP server:

```text
http://localhost:9000/gmail/v1/...
http://localhost:9000/calendar/v3/...
http://localhost:9000/people/v1/...
```

Then your clients point at it via:

* Config/env: `GOOGLE_API_BASE_URL=http://localhost:9000`
  instead of `https://www.googleapis.com`.

You *do not* need to perfectly reproduce everything. Just:

* The endpoints you actually hit
* The fields you actually read/write
* Enough behavior to surface:

  * pagination
  * error codes
  * rate limiting / flakiness (optional but fun)

---

## Auth: fake it aggressively

You probably don’t want real OAuth here at all.

Do something like:

* Accept `Authorization: Bearer <anything>`

* Parse out “user” from the token in a very dumb way, e.g.:

  ```text
  Authorization: Bearer user:harper
  ```

* Map that to a local “account” in your store.

* Enforce scopes *optionally* with a toy header or query param if you want to test “missing scope” behavior.

The point is: keep the contract that “there is a user + scopes”, without any real Google security story involved.

---

## Storage model

Use something simple and inspectable:

* `SQLite` or `badger/bolt` file per environment
* Or `data/` with JSON files if you want git-friendly fixtures

Rough schema:

### Gmail-ish

Tables/entities:

* `users`
* `labels`
* `threads`
* `messages`

Minimal fields:

```jsonc
{
  "id": "msg_123",
  "threadId": "thr_456",
  "labelIds": ["INBOX", "STARRED"],
  "snippet": "first 100 chars...",
  "internalDate": 1733000000000,
  "payload": {
    "headers": [
      {"name": "From", "value": "Alice <alice@example.com>"},
      {"name": "To", "value": "you@example.com"},
      {"name": "Subject", "value": "hi"}
    ],
    "body": {
      "size": 42,
      "data": "base64url..."
    },
    "parts": []
  }
}
```

You just need enough shape so your client code can’t tell it’s not Google.

### Calendar-ish

Entities:

* `calendars`
* `events`
* (optionally) `acl`

Minimal event:

```jsonc
{
  "id": "evt_123",
  "summary": "Coffee",
  "description": "Discuss the future of email",
  "start": {"dateTime": "2025-12-01T10:00:00-06:00"},
  "end": {"dateTime": "2025-12-01T11:00:00-06:00"},
  "attendees": [
    {"email": "you@example.com", "responseStatus": "accepted"}
  ]
}
```

### People-ish

Entities:

* `people`

Minimal:

```jsonc
{
  "resourceName": "people/harper",
  "names": [{ "displayName": "Harper Reed" }],
  "emailAddresses": [{ "value": "harper@example.com" }],
  "photos": [{ "url": "https://example.com/avatar.png" }]
}
```

---

## Behavior to emulate (the fun bits)

At minimum:

* Pagination: `pageToken`, `maxResults`
* Filtering: simple `q=` or `query=` or `timeMin/timeMax`
* 404s: unknown IDs
* 400s: invalid params

Optional but really useful:

* **Chaos mode**: env flags for:

  * `FAKE_GAPI_RATE_LIMIT=1/10` → return `429` 10% of the time
  * `FAKE_GAPI_RANDOM_500=1/50`
* **Latency**: sleep based on endpoint
* **Consistency weirdness**: “event insert” succeeds but doesn’t show up in list for X ms (to test eventual consistency assumptions)

---

## Implementation sketch (Go example)

Very barebones, just to show wiring.

```go
// main.go
package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type GmailMessage struct {
	ID        string   `json:"id"`
	ThreadID  string   `json:"threadId"`
	LabelIDs  []string `json:"labelIds"`
	Snippet   string   `json:"snippet"`
	InternalDate int64 `json:"internalDate"`
	// ...payload omitted
}

type Store struct {
	Messages map[string][]GmailMessage // userID -> messages
	// add calendars, people, etc
}

func main() {
	store := &Store{
		Messages: map[string][]GmailMessage{
			"harper": {
				{ID: "msg_1", ThreadID: "thr_1", LabelIDs: []string{"INBOX"}, Snippet: "hello there", InternalDate: 1733000000000},
			},
		},
	}

	r := chi.NewRouter()

	// Gmail list messages
	r.Get("/gmail/v1/users/{userId}/messages", func(w http.ResponseWriter, r *http.Request) {
		userID := chi.URLParam(r, "userId")
		if userID == "me" {
			userID = fakeUserFromAuth(r)
		}

		msgs := store.Messages[userID]
		resp := map[string]any{
			"messages": msgs,
			"resultSizeEstimate": len(msgs),
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	// Calendar, People handlers similar shape under /calendar/v3 and /people/v1

	log.Println("Fake Google box on :9000")
	log.Fatal(http.ListenAndServe(":9000", r))
}

func fakeUserFromAuth(r *http.Request) string {
	// ex: "Bearer user:harper"
	auth := r.Header.Get("Authorization")
	// parse or just default
	if auth == "" {
		return "default"
	}
	// TODO: real parsing, but you get the idea
	return "harper"
}
```

Commented enough? Barely. You’d flesh that out per endpoint you actually use.

---

## Client-side integration pattern

Abstract Google behind interfaces *anyway*, then `impl = real | fake`:

```ts
// googleClient.ts
export interface GmailClient {
  listMessages(params: { q?: string }): Promise<Message[]>;
  getMessage(id: string): Promise<Message>;
  // ...
}

export function makeGmailClient(baseUrl: string, token: string): GmailClient {
  return {
    async listMessages({ q }) {
      const url = new URL(`${baseUrl}/gmail/v1/users/me/messages`);
      if (q) url.searchParams.set("q", q);
      const res = await fetch(url, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!res.ok) throw new Error(`Gmail error: ${res.status}`);
      const body = await res.json();
      return body.messages ?? [];
    },
    async getMessage(id) { /* ... */ }
  };
}
```

Then:

* `GOOGLE_API_BASE_URL=https://www.googleapis.com` in prod
* `GOOGLE_API_BASE_URL=http://localhost:9000` in dev/tests

Same client, different backend.

---

## How I’d actually build it, step-by-step

1. **Log traffic** from your real client for a bit:

   * Capture: which endpoints, which fields you care about.
2. **Define minimal contracts** in OpenAPI/JSON schema for:

   * `gmail.listMessages`, `gmail.getMessage`, `gmail.send`
   * `calendar.listEvents`, `calendar.insert`, `calendar.patch`
   * `people.connections.list`
3. Generate or hand-write a tiny Go/Node service that implements just those.
4. Add fixtures + a tiny CLI:

   * `fakegapi seed demo-user`
   * `fakegapi load fixtures/email_simple.json`
5. Sprinkle in chaos toggles and pagination.
6. Switch your clients to `GOOGLE_API_BASE_URL` and never point dev at real Google again.

You can later get fancy with a record–replay mode, but you don’t need that to get 90% of the value.

---

If you want, next iteration we can explicitly spec the *minimal* endpoint set per API (like a little table) and design the SQLite schema so you can just `go run ./cmd/fakegapi` and be done.

