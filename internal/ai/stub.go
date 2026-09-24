package ai

import (
	"context"
	"errors"
	"math/rand/v2"
	"time"
)

// StubOptions controls the simulated AI behaviour.
type StubOptions struct {
	// Delay simulates AI latency (how long an LLM would take).
	// Zero means the stub answers instantly.
	Delay time.Duration

	// FailRate in [0.0, 1.0] is the probability that a generation fails.
	// This exists only so you can demo the "failed" status of a job.
	// Zero means the stub never fails.
	FailRate float64
}

// StubAIClient implements AIClient without any API key.
//
// It simulates a real AI service:
//   - it "thinks" for a short time (Delay)
//   - it produces a structured quiz that references the lesson objective
//   - it can occasionally fail (FailRate) like a flaky provider
//
// Because it implements the same AIClient interface as a real provider,
// the whole backend works exactly the same way either way.
type StubAIClient struct {
	opts StubOptions
}

// NewStubClient creates a stub AI client.
func NewStubClient(opts StubOptions) *StubAIClient {
	return &StubAIClient{opts: opts}
}

// GenerateQuiz simulates an AI call and returns a validated-shaped quiz.
func (s *StubAIClient) GenerateQuiz(ctx context.Context, req GenerateQuizRequest) (GenerateQuizResponse, error) {
	// Simulate network + model latency, respecting context cancellation.
	select {
	case <-time.After(s.opts.Delay):
	case <-ctx.Done():
		return GenerateQuizResponse{}, ctx.Err()
	}

	// Simulate a provider that is occasionally unavailable.
	if s.opts.FailRate > 0 && rand.Float64() < s.opts.FailRate {
		return GenerateQuizResponse{}, errors.New("simulated AI provider failure")
	}

	return generateStubQuiz(req), nil
}

// generateStubQuiz builds a deterministic quiz from the lesson objective.
// The first question proves the stub "read" the context (like a real LLM would).
func generateStubQuiz(req GenerateQuizRequest) GenerateQuizResponse {
	difficulty := "medium"
	switch {
	case req.Mastery < 30:
		difficulty = "easy"
	case req.Mastery >= 70:
		difficulty = "hard"
	}

	return GenerateQuizResponse{
		Difficulty: difficulty,
		Questions: []Question{
			{
				Question: "Which statement best matches this lesson's objective?",
				Options: []string{
					req.Objective,
					"A random fact unrelated to the lesson",
					"The definition of a database index",
					"An instruction to rewrite the file",
				},
				CorrectOption: 0,
			},
			{
				Question: "What does a hash function map its input to?",
				Options: []string{
					"A variable-length value",
					"A fixed-size value",
					"An image file",
					"A username",
				},
				CorrectOption: 1,
			},
			{
				Question: "Which property is important for a hash function to be useful in practice?",
				Options: []string{
					"It must return different values on every call",
					"The same input always gives the same output",
					"It must require a password to run",
					"It only works on files larger than 1 MB",
				},
				CorrectOption: 1,
			},
		},
	}
}
