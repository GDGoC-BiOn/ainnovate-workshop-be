# WORKSHOP.md — Mentor Guide

> **arahin-mini** live coding workshop — 30 minutes.
> Audience: SMA/SMK + university students + beginners.
> Main topic: **how a backend integrates AI into a reliable application.**

This document is the mentor's running guide. The full reference solution is in
**[SOLUTION.md](./SOLUTION.md)**.

---

## 1. Workshop objective

By the end of the session, students can explain and implement the backbone of
an AI-driven feature:

```
Frontend: one button          Backend: an ordered workflow
----------------------------------------------
"Generate Practice"     ->  Validate
                              Load lesson (SQL)
                              Load mastery (SQL)
                              Build AI context
                              Call AI
                              Validate AI output
                              Transaction (create quiz + batch questions)
                              Return quiz
```

## 2. Architecture in one slide

```
Handler  ->  Service  ->  Repository  ->  PostgreSQL
                |
                +------>  AIClient (stub / real)
```

- **Handler**: parses HTTP, validates the request, responds. No business logic.
- **Service**: the workflow (what happens when, in what order). No SQL.
- **Repository**: SQL only. Reads + the transaction + the batch.
- **AIClient**: an interface. The backend never hard-codes an AI provider.

## 3. Demo to start (2-3 min)

Run the server with the TODOs still in place:

```sh
make run
```

1. `POST /v1/quiz-requests` → `202` proves the API is instant.
2. `GET /v1/quiz-requests/{id}` → shows `failed` with
   `"load lesson: TODO 1: ..."`.
3. Explain: **every failed step has an owner** — students will become that
   owner while implementing each TODO.

Now stop the server, keep a terminal ready, and start coding.

---

## 4. Step-by-step tasks

Order matters: build **top-down with the compiler as your guide**, or
**bottom-up as below**. The bottom-up order is recommended because each TODO
builds on the previous one.

### TODO 1 — `GetLesson`

**File:** `internal/quiz/repository.go`
**Goal:** read one lesson from PostgreSQL.

**Hints**
- Use `r.pool.QueryRow(ctx, sql, id)`.
- Scan into a `Lesson` value: `(&lesson.ID, &lesson.Title, &lesson.Objective, &lesson.CreatedAt)`.
- Map `pgx.ErrNoRows` → `ErrLessonNotFound`.
- Column list: `id, title, objective, created_at`.

**Expected result**
```sh
curl http://localhost:8080/v1/quiz-requests/<id>   # after a POST
# status now changes from "load lesson: TODO 1" to the next error
```

### TODO 2 — `GetMastery`

**File:** `internal/quiz/repository.go`
**Goal:** read the user's mastery score for the lesson.

**Hints**
- `WHERE user_id = $1 AND lesson_id = $2`.
- Map `pgx.ErrNoRows` → `ErrMasteryNotFound`.

### TODO 3 — `buildAIRequest`

**File:** `internal/quiz/service.go`
**Goal:** turn domain data into the AI's input.

```go
return ai.GenerateQuizRequest{
    Objective: lesson.Objective,
    Mastery:   mastery.Score,
}, nil
```

**Teaching moment:** the service only speaks its own language (Lesson/Mastery).
The AI only understands `Objective` + `Mastery`. `buildAIRequest` is the
translator.

### TODO 4 — Validate the AI response

**File:** `internal/quiz/service.go` (inside `runGeneration`, step 5)

**Goal:** refuse bad AI output before saving.

`ai.ValidateResponse` already exists and is pre-tested. Wire it in:

```go
if err := ai.ValidateResponse(aiResponse); err != nil {
    slog.Error("validation failed", "request_id", req.ID, "error", err)
    return Quiz{}, fmt.Errorf("validate AI response: %w", err)
}
```

**Teaching moment:** *"AI generates. Backend governs."* An AI can fail. The
database must never receive unvalidated content.

Common mistakes:
- validating *after* saving (too late),
- only checking `len(questions) > 0` and skipping option count / difficulty.

### TODO 5 — Transaction for quiz creation

**File:** `internal/quiz/repository.go` — `CreateQuiz`

**Goal:** insert quiz + questions atomically.

```go
tx, err := r.pool.Begin(ctx)
if err != nil { ... }
defer tx.Rollback(ctx) // no-op after a successful Commit

// insert quiz with RETURNING id, created_at (tx.QueryRow, not pool)
// ... insert questions (next TODO) ...
err = tx.Commit(ctx)
```

**Teaching moment:** *all succeed together, or all fail together.* If question
#2 of 3 fails, a rolled-back quiz leaves no orphan rows.

### TODO 6 — `pgx.Batch` for questions

**File:** same, inside the transaction.

**Goal:** send all `INSERT INTO questions ...` in a **single round-trip**.

```go
batch := &pgx.Batch{}
for _, question := range params.Questions {
    opts, _ := json.Marshal(question.Options) // []string -> JSONB
    batch.Queue(sql, question.ID, created.ID, question.Text, opts,
        question.CorrectOption, question.Position)
}
results := tx.SendBatch(ctx, batch)
if err := results.Close(); err != nil { ... }
```

**Teaching moment:** 3 questions + 1 quiz = 4 inserts. Without a batch that is
4 network round-trips; with a batch it is **1**.

---

## 5. Definition of done

The workshop is complete when this works:

```sh
curl -X POST http://localhost:8080/v1/quiz-requests \
  -H "Content-Type: application/json" \
  -d '{"user_id":"11111111-1111-1111-1111-111111111111",
       "lesson_id":"22222222-2222-2222-2222-222222222222"}'
# 202 {"request_id":"...","status":"processing"}

curl http://localhost:8080/v1/quiz-requests/<request_id>
# "status":"ready","quiz_id":"..."

curl http://localhost:8080/v1/quizzes/<quiz_id>
# difficulties + questions, NO correct_option
```

## 6. Common mistakes (call these out!)

1. **Building SQL with string concatenation** — always use `$1`/`$2`.
2. **Scanning into the wrong field** — compare argument order with the SQL.
3. **Forgetting `pgx.ErrNoRows`** → the API returns 500 instead of a clean
   "not found".
4. **Returning early inside the transaction** without relying on the
   `defer tx.Rollback` safety net.
5. **Forgetting `results.Close()`** on the batch.
6. **Repeating SQL in a loop with `Exec`** instead of using the batch.
7. **Exposing `correct_option`** — keep the model's `json:"-"` tag intact.

## 7. Advancing the students (if done early)

- Set `AI_STUB_FAIL_RATE=0.2` and watch the job land in `failed`.
- Lower `AI_STUB_DELAY_MS=1500` so students can observe `processing`
  while polling.
- Write a short integration test in `SOLUTION.md` that exercises the
  repository against a real local PostgreSQL.

## 8. Final explanation to close the session

1. The `202` + poll loop shows **HTTP requests and long AI work are different
   things** — the same pattern used by Cloud Tasks in production.
2. Validation makes the backend the **owner of its data model**, never the AI.
3. A transaction guarantees your data is never half-written.
4. Interfaces like `AIClient` are how you swap a stub for a real LLM without
   rewriting the app.

## 9. Rollback / reset

Because students may deviate, the truly safe command sequence before the
workshop is:

```sh
make migrate   # idempotent (IF NOT EXISTS)
make seed      # idempotent (ON CONFLICT DO NOTHING)
go test ./...  # sanity check
```