# SOLUTION.md — Complete Reference Implementation

Reference solution for the 6 workshop TODOs. Every snippet below was
**verified end-to-end** against a local PostgreSQL instance
(`go build`, `go test -race ./...`, live API calls).

> Suggestion for the mentor: do **not** paste this file on the projector.
> Use it as your private reference and for review at the end.

---

## Table of contents

1. [TODO 1 — GetLesson](#todo-1--getlesson)
2. [TODO 2 — GetMastery](#todo-2--getmastery)
3. [TODO 3 — buildAIRequest](#todo-3--buildairequest)
4. [TODO 4 — validate the AI response](#todo-4--validate-the-ai-response)
5. [TODO 5 — CreateQuiz transaction](#todo-5--createquiz-transaction)
6. [TODO 6 — pgx.Batch insert](#todo-6--pgxbatch-insert)
7. [Full final `CreateQuiz`](#full-final-createquiz)
8. [Extra: happy-path service test](#extra-happy-path-service-test)
9. [How to replace the stub with a real AI](#how-to-replace-the-stub-with-a-real-ai)
10. [How this maps to the real Arah.in backend](#how-this-maps-to-the-real-arahin-backend)

---

## TODO 1 — GetLesson

Replace the body of `GetLesson` in `internal/quiz/repository.go`:

```go
func (r *PostgresRepository) GetLesson(ctx context.Context, id uuid.UUID) (Lesson, error) {
	var lesson Lesson

	err := r.pool.QueryRow(ctx, `
		SELECT id, title, objective, created_at
		FROM lessons
		WHERE id = $1`, id,
	).Scan(&lesson.ID, &lesson.Title, &lesson.Objective, &lesson.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Lesson{}, ErrLessonNotFound
	}
	if err != nil {
		return Lesson{}, fmt.Errorf("query lesson: %w", err)
	}

	return lesson, nil
}
```

Notes:
- `$1` is a placeholder; pgx never concatenates user input into SQL.
- `pgx.ErrNoRows` must become a domain error, not a raw "no rows".

## TODO 2 — GetMastery

```go
func (r *PostgresRepository) GetMastery(ctx context.Context, userID uuid.UUID, lessonID uuid.UUID) (Mastery, error) {
	var mastery Mastery

	err := r.pool.QueryRow(ctx, `
		SELECT user_id, lesson_id, score
		FROM masteries
		WHERE user_id = $1 AND lesson_id = $2`, userID, lessonID,
	).Scan(&mastery.UserID, &mastery.LessonID, &mastery.Score)
	if errors.Is(err, pgx.ErrNoRows) {
		return Mastery{}, ErrMasteryNotFound
	}
	if err != nil {
		return Mastery{}, fmt.Errorf("query mastery: %w", err)
	}

	return mastery, nil
}
```

## TODO 3 — buildAIRequest

In `internal/quiz/service.go`:

```go
func (s *Service) buildAIRequest(ctx context.Context, lesson Lesson, mastery Mastery) (ai.GenerateQuizRequest, error) {
	return ai.GenerateQuizRequest{
		Objective: lesson.Objective,
		Mastery:   mastery.Score,
	}, nil
}
```

Bonus (optional stretch): tune the desired difficulty from mastery before
sending it to a real provider.

## TODO 4 — validate the AI response

In `internal/quiz/service.go`, inside `runGeneration` step 5 (replace the
commented block):

```go
	// 5. Validate the AI response BEFORE saving.
	if err := ai.ValidateResponse(aiResponse); err != nil {
		slog.Error("validation failed",
			"request_id", req.ID,
			"error", err,
		)
		return Quiz{}, fmt.Errorf("validate AI response: %w", err)
	}

	// 6. Persist quiz + questions atomically.
```

Remember the existing function already enforces:
- difficulty ∈ {easy, medium, hard}
- at least one question
- non-empty question text
- exactly 4 options per question
- `correct_option` inside the valid range

## TODO 5 — CreateQuiz transaction

The transaction opens, defers the rollback safety net, inserts the quiz with
`RETURNING`, then lets TODO 6 queue the questions, and commits.

## TODO 6 — pgx.Batch insert

The batch queues every question, is sent once, and **must be `Close()`d**
before commit.

## Full final CreateQuiz

Combined TODO 5 + 6 in `internal/quiz/repository.go`:

```go
func (r *PostgresRepository) CreateQuiz(ctx context.Context, params CreateQuizParams) (Quiz, error) {
	if len(params.Questions) == 0 {
		return Quiz{}, errors.New("cannot create a quiz without questions")
	}

	// 1. Open a transaction.
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Quiz{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) // safe no-op if Commit already succeeded

	// 2. Insert the quiz.
	var created Quiz
	err = tx.QueryRow(ctx, `
		INSERT INTO quizzes (id, user_id, lesson_id, difficulty)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, lesson_id, difficulty, created_at`,
		uuid.New(), params.UserID, params.LessonID, params.Difficulty,
	).Scan(&created.ID, &created.UserID, &created.LessonID, &created.Difficulty, &created.CreatedAt)
	if err != nil {
		return Quiz{}, fmt.Errorf("insert quiz: %w", err)
	}

	// 3. Queue every question into one pgx.Batch (single round-trip).
	batch := &pgx.Batch{}

	for _, question := range params.Questions {
		optionsJSON, err := json.Marshal(question.Options)
		if err != nil {
			return Quiz{}, fmt.Errorf("encode question options: %w", err)
		}

		batch.Queue(`
			INSERT INTO questions (id, quiz_id, question, options, correct_option, position)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			question.ID, created.ID, question.Text, optionsJSON, question.CorrectOption, question.Position,
		)
	}

	// 4. Send the whole batch in one round-trip, then close it (required!).
	results := tx.SendBatch(ctx, batch)
	if err := results.Close(); err != nil {
		return Quiz{}, fmt.Errorf("insert questions: %w", err)
	}

	// 5. All-or-nothing: commit everything together.
	if err := tx.Commit(ctx); err != nil {
		return Quiz{}, fmt.Errorf("commit transaction: %w", err)
	}

	return created, nil
}
```

**Why is `Close()` required?** `SendBatch` returns a `BatchResults`;
calling `Close()` flushes/executes the queued statements and releases the
connection. Skipping it in this flow is a classic bug.

---

## Extra: happy-path service test

The starter repo tests **only** the pieces that are not TODOs, so the suite
passes on day one. After the workshop, add a full happy-path test in
`internal/quiz/service_test.go`:

```go
func TestRunGeneration_Success(t *testing.T) {
	svc, repo := newTestService()
	repo.mastery = Mastery{UserID: uuid.MustParse(testUserID), LessonID: uuid.MustParse(testLessonID), Score: 40}
	repo.quiz = Quiz{
		ID:         uuid.New(),
		Difficulty: "medium",
		Questions: []Question{
			{ID: uuid.New(), Text: "Q1", Options: []string{"a", "b", "c", "d"}, CorrectOption: 1},
		},
	}

	var capture CreateQuizParams
	repo.createQuizFn = func(ctx context.Context, params CreateQuizParams) (Quiz, error) {
		capture = params
		return repo.quiz, repo.quizErr
	}

	req := &QuizRequest{ID: uuid.New(), UserID: uuid.MustParse(testUserID), LessonID: uuid.MustParse(testLessonID)}
	created, err := svc.runGeneration(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created.ID != repo.quiz.ID {
		t.Fatalf("expected quiz %s, got %s", repo.quiz.ID, created.ID)
	}
	if capture.Difficulty == "" {
		t.Fatal("expected the quiz to be persisted with difficulty")
	}
	if len(capture.Questions) == 0 {
		t.Fatal("expected the quiz to be persisted with questions")
	}
}
```

To support `createQuizFn`, add a field to `fakeRepo`:

```go
type fakeRepo struct {
	// ...existing fields...
	createQuizFn func(ctx context.Context, params CreateQuizParams) (Quiz, error)
}

func (f *fakeRepo) CreateQuiz(ctx context.Context, params CreateQuizParams) (Quiz, error) {
	if f.createQuizFn != nil {
		return f.createQuizFn(ctx, params)
	}
	return f.quiz, f.quizErr
}
```

(Optional, for mentors with a local Postgres handy: a repository-level test
using a real `pgxpool` answering the same `Repository` interface.)

---

## How to replace the stub with a real AI

The swap is intentionally a **3-step, zero-rewrite** operation.

**Step 1 — keep the interface intact** (`internal/ai/client.go`):

```go
type AIClient interface {
	GenerateQuiz(ctx context.Context, req GenerateQuizRequest) (GenerateQuizResponse, error)
}
```

**Step 2 — add a provider implementation** e.g. `internal/ai/httpclient.go`:

```go
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// HTTPAIClient calls any OpenAI-compatible /chat/completions endpoint.
type HTTPAIClient struct {
	BaseURL   string        // e.g. "https://api.openai.com/v1"
	APIKey    string        // read from env, never hard-coded
	Timeout   time.Duration
	Model     string
	client    *http.Client
}

func NewHTTPAIClient(baseURL, apiKey, model string) *HTTPAIClient {
	return &HTTPAIClient{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Model:   model,
		Timeout: 30 * time.Second,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *HTTPAIClient) GenerateQuiz(ctx context.Context, req GenerateQuizRequest) (GenerateQuizResponse, error) {
	prompt := fmt.Sprintf(
		`You are a quiz generator for a learning app.
Objective: %s
Learner mastery: %d
Respond with STRICT JSON only:
{"difficulty":"easy|medium|hard","questions":[{"question":"...","options":["a","b","c","d"],"correct_option":0}]}`,
		req.Objective, req.Mastery)

	body := map[string]any{
		"model": c.Model,
		"messages": []map[string]string{
			{"role": "system", "content": "You produce quizzes as strict JSON."},
			{"role": "user", "content": prompt},
		},
		"response_format": map[string]string{"type": "json_object"},
	}

	raw, _ := json.Marshal(body)
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/chat/completions", bytes.NewReader(raw))
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return GenerateQuizResponse{}, fmt.Errorf("AI HTTP call: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return GenerateQuizResponse{}, fmt.Errorf("AI provider status %d", resp.StatusCode)
	}

	// ... decode "choices[0].message.content" then json.Unmarshal into
	// GenerateQuizResponse ...
	return GenerateQuizResponse{}, fmt.Errorf("TODO: finish decoding provider JSON")
}
```

**Step 3 — wire it in `cmd/api/main.go`:**

```go
aiClient := ai.NewHTTPAIClient(
	envString("AI_BASE_URL", "https://api.openai.com/v1"),
	envString("AI_API_KEY", ""), // from environment, never committed
	envString("AI_MODEL", "gpt-4o-mini"),
)
```

Nothing else changes: validation, transactions, persistence, and the handler
all keep working because the service depends on the `AIClient` **interface**.

> **Remember:** even a real AI's output still passes through the same
> `ValidateResponse` gate. AI generates. Backend governs.

---

## How this maps to the real Arah.in backend

| arahin-mini concept | Production Arah.in counterpart |
|---------------------|--------------------------------|
| `POST /v1/quiz-requests` + `202` | Long-running generation tasks delivered to workers (Cloud Tasks) |
| In-memory `requestStore` | Durable job table, retries, DLQ |
| `Service.runGeneration` | Content generation orchestration (concepts → lessons → quizzes → activities) |
| `ai.AIClient` interface | Provider adapters for LLM services |
| `ValidateResponse` | Schema-validated AI output before any write |
| Transaction + `pgx.Batch` | Atomic writes for complex entities across many tables |
| `masteries` table | Real mastery & spaced-repetition engine (progress, reviews) |
| `Quiz`/`Question` model | Concepts, quizzes, activities, learning spaces |

The workshop deliberately keeps **one** vertical slice so students see the
whole journey from button to database — and can swap any layer later.

## Final verification (the whole project)

```sh
go fmt ./...
go vet ./...
go build ./...
go test ./...
```

All four must be silent and green.