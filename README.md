# arahin-mini

An educational backend for the **AInnovate** Backend Sharing workshop.

`arahin-mini` is a tiny, simplified version of **Arah.in** — an AI-powered
learning platform. The production idea is:

> Turn any PDF, book, documentation or course into an interactive,
> Duolingo-style learning journey with measured mastery.

This workshop build keeps exactly **one** use case: a user clicks
**"Generate Practice"**, and the backend produces a multiple-choice quiz.

The frontend sees **one button**. The backend sees a **workflow**:

```
Request
  -> Validate
  -> Load lesson (PostgreSQL)
  -> Load mastery (PostgreSQL)
  -> Build AI context
  -> Call AI
  -> Validate AI output
  -> Begin transaction
  -> Create quiz
  -> Batch-insert questions
  -> Commit
  -> Return result
```

---

## What will you learn?

| # | Concept | Where in the code |
|---|---------|-------------------|
| 1 | REST API & HTTP status codes | `internal/quiz/handler.go` |
| 2 | Go HTTP server with `go-chi/chi/v5` | `cmd/api/main.go` |
| 3 | Handler -> Service -> Repository layers | `internal/quiz/*` |
| 4 | PostgreSQL with `pgx/v5` (no ORM) | `internal/quiz/repository.go` |
| 5 | Raw SQL + parameterized queries (`$1`, `$2`, ...) | `internal/quiz/repository.go` |
| 6 | Database transactions (all or nothing) | `internal/quiz/repository.go` (`TODO 5`) |
| 7 | `pgx.Batch` / `SendBatch` (one round-trip, many inserts) | `internal/quiz/repository.go` (`TODO 6`) |
| 8 | AI client abstraction (interface, easy to swap) | `internal/ai/client.go` |
| 9 | Structured AI output + validation | `internal/ai/client.go` |
| 10 | Dependency injection (manual wiring) | `cmd/api/main.go` |
| 11 | Error handling with a stable JSON shape | `internal/platform/http.go` |
| 12 | Async job: POST fast, poll for result | `internal/quiz/service.go` |

> This workshop is **not** a generic "AI tutorial".
> The main topic is: **how a backend integrates AI into a reliable
> application** — validation, transactions, and safe persistence.

---

## Architecture

```
                    Postman / Frontend
                           |
                           v
                    +--------------+
                    | Go + Chi API |
                    +------+-------+
                           |
                           v
                    +--------------+
                    | Quiz Service |
                    +------+-------+
                           |
               +-----------+-----------+
               |                       |
               v                       v
        +-------------+         +-------------+
        | PostgreSQL  |         |  AI Client  |
        |    + pgx    |         +------+------+
        +-------------+                |
                                +-------+-------+
                                |               |
                                v               v
                           Stub AI         Real AI
                         (Workshop)       (Optional)
```

The dependency flow in `main.go` is fully manual and visible:

```
db -> repository -> service -> handler
                          \
                           +-> aiClient (stub / real)
```

---

## Prerequisites

- **Go 1.25 or newer**
  - check: `go version`
- **PostgreSQL 13 or newer** running locally
  - check: `pg_isready`
- **`psql`** CLI (comes with PostgreSQL)
- Optional but nice: Postman or a REST client (curl is enough)

> No Docker needed. No AI API key needed. The project runs 100% offline.

---

## 1. Set up PostgreSQL

Create a role/database or reuse an existing one.

```sh
# example using the default Homebrew/managed postgres user
createdb arahin_mini
```

If your local setup uses a password, remember it — you'll put it in `.env`.

---

## 2. Environment variables

Copy the example file and adjust:

```sh
cp .env.example .env
```

Open `.env` and set at least `DATABASE_URL`:

```env
HTTP_ADDR=:8080
DATABASE_URL=postgres://postgres:postgres@localhost:5432/arahin_mini?sslmode=disable
AI_STUB_DELAY_MS=400
AI_STUB_FAIL_RATE=0.0
```

> `.env` is already in `.gitignore`. Never commit real credentials.

---

## 3. Migrate

```sh
make migrate
```

This applies `migrations/001_init.sql` and creates 5 tables:
`users`, `lessons`, `masteries`, `quizzes`, `questions`.

---

## 4. Seed

```sh
make seed
```

This inserts predictable data you'll use during the workshop:

| Entity | ID | Value |
|--------|----|-------|
| user | `11111111-1111-1111-1111-111111111111` | **Alice** |
| lesson | `22222222-2222-2222-2222-222222222222` | **Understand Hash Function** |
| mastery | (Alice, Hash Function) | score **40** |

---

## 5. Run the server

```sh
make run
```

or, without the Makefile:

```sh
DATABASE_URL=postgres://postgres:postgres@localhost:5432/arahin_mini?sslmode=disable go run ./cmd/api
```

Expected output (JSON logs):

```
{"time":"...","level":"INFO","msg":"connecting to postgres"}
{"time":"...","level":"INFO","msg":"server listening","addr":":8080"}
```

---

## 6. Test the endpoints

### Health check

```sh
curl http://localhost:8080/health
```

```json
{"status":"ok"}
```

### Start quiz generation (async)

```sh
curl -X POST http://localhost:8080/v1/quiz-requests \
  -H "Content-Type: application/json" \
  -d '{
        "user_id": "11111111-1111-1111-1111-111111111111",
        "lesson_id": "22222222-2222-2222-2222-222222222222"
      }'
```

HTTP **202 Accepted** — the request is accepted, the job starts in the
background:

```json
{"request_id":"9a2c...","status":"processing"}
```

### Poll the job status

```sh
curl http://localhost:8080/v1/quiz-requests/9a2c...
```

While the AI is "thinking" (or while TODOs are not yet implemented):

```json
{"request_id":"9a2c...","status":"processing"}
```

If generation succeeded:

```json
{"request_id":"9a2c...","status":"ready","quiz_id":"6f41..."}
```

If generation failed (e.g. a step is still a TODO, or the AI failed):

```json
{"request_id":"9a2c...","status":"failed","message":"load lesson: TODO 1: ..."}
```

### Fetch the generated quiz

```sh
curl http://localhost:8080/v1/quizzes/6f41...
```

```json
{
  "id": "6f41...",
  "difficulty": "medium",
  "questions": [
    {
      "id": "...",
      "question": "What does a hash function map its input to?",
      "options": [
        "A variable-length value",
        "A fixed-size value",
        "An image file",
        "A username"
      ]
    }
  ]
}
```

> **Note:** `correct_option` is **never** returned to the client.

### Error examples

```sh
curl -X POST http://localhost:8080/v1/quiz-requests \
  -H "Content-Type: application/json" -d '{"lesson_id":"..."}'
```

```json
{"error":{"code":"INVALID_REQUEST","message":"user_id is required"}}
```

---

## 7. Workshop flow in 30 minutes

| Minute | Activity |
|--------|----------|
| 0-3 | Quick intro to Arah.in + this repo |
| 3-8 | Walk through `main.go` wiring and the 3 endpoints |
| 8-30 | Implement the 6 TODOs (see `WORKSHOP.md`) |

The 6 TODOs:

1. `GetLesson` — read a lesson from PostgreSQL
2. `GetMastery` — read a mastery score from PostgreSQL
3. `buildAIRequest` — build the AI context from lesson + mastery
4. Wire AI response validation before saving
5. `CreateQuiz` — save quiz + questions in one **transaction**
6. Use `pgx.Batch` to insert all questions in **one round-trip**

See **[WORKSHOP.md](./WORKSHOP.md)** for the step-by-step guide and
**[SOLUTION.md](./SOLUTION.md)** for the complete reference implementation.

---

## 8. How the AI stub works

`internal/ai/stub.go` implements the same `AIClient` interface a real
provider would use. With **no API key** it:

- waits a few hundred milliseconds (simulated latency),
- returns a deterministic 3-question quiz that references the lesson
  objective,
- can be configured to fail randomly to demo the `failed` status:

```env
AI_STUB_DELAY_MS=400   # simulated thinking time
AI_STUB_FAIL_RATE=0.1  # 10% chance a generation fails
```

To replace it with a real AI later:

1. Keep the `AIClient` interface.
2. Add e.g. `internal/ai/openai.go` implementing it (HTTP call to the
   provider, parse JSON into `GenerateQuizResponse`).
3. In `main.go`, create the real client instead of the stub.

The rest of the backend does not change.

---

## 9. Async job — workshop vs production

- **Here (workshop):** job state lives in an in-memory Go map, and the job
  runs in a goroutine. It disappears on restart. Zero infrastructure.
- **Real Arah.in (production):** job delivery uses a durable queue such as
  **Google Cloud Tasks**, job records live in the database, and workers can
  be retried and inspected.

The **API contract is the same shape** in both cases — which is exactly the
point: you can learn the pattern here and swap the plumbing later.

---

## 10. How this differs from the real Arah.in backend

| Aspect | arahin-mini (this repo) | Arah.in production |
|--------|-------------------------|--------------------|
| Scope | 1 lesson + 1 user workflow | Learning spaces, lessons, concepts, quizzes, activities |
| Persistence | 5 tables | Many more (activity logs, review scheduling, mastery events, ...) |
| AI content | A stub returning one fixed quiz | Real AI-generated lessons, quizzes, and explanations |
| Jobs | In-memory goroutine | Google Cloud Tasks + durable state |
| Auth | none (workshop) | Full authentication & authorization |
| API | 3 endpoints | Full REST surface + OpenAPI contract |

---

## Project structure

```
arahin-mini/
│
├── cmd/
│   └── api/            main.go         # entrypoint + manual dependency wiring
│
├── internal/
│   ├── quiz/
│   │   ├── handler.go                  # HTTP layer (parse / validate / respond)
│   │   ├── service.go                  # business workflow + async job
│   │   ├── repository.go               # raw SQL: reading + transaction + batch
│   │   └── model.go                    # domain types (Quiz, Question, ...)
│   │
│   ├── ai/
│   │   ├── client.go                   # AIClient interface + response validation
│   │   └── stub.go                     # offline AI simulator
│   │
│   └── platform/
│       ├── db.go                       # pgx connection pool
│       └── http.go                     # JSON / error helpers
│
├── migrations/  001_init.sql           # schema
├── seed/        seed.sql               # Alice + hash-function lesson
├── api/         openapi.yaml           # API contract
├── .env.example
├── go.mod
└── Makefile
```

## Useful commands

```sh
make run      # start the server
make migrate  # apply migrations
make seed     # insert seed data
make test     # go test ./...
make fmt      # check gofmt
```

---

## Learning objectives recap

After this workshop you should be able to explain:

1. Why the handler returns `202` instead of blocking on the AI call.
2. Why `correct_option` must never appear in an API response.
3. Why we validate AI output before saving it.
4. Why quiz + questions must be inserted in one transaction.
5. What `pgx.Batch` does and why it matters.
6. How to swap the AI stub for a real provider without breaking anything.