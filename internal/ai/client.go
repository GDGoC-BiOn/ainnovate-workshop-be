package ai

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// GenerateQuizRequest is the context we hand to the AI provider.
// It is deliberately small: it contains everything the AI needs to
// produce a quiz for a specific learner.
type GenerateQuizRequest struct {
	Objective string
	Mastery   int
}

// Question is a single multiple-choice question produced by the AI.
type Question struct {
	Question      string
	Options       []string
	CorrectOption int
}

// GenerateQuizResponse is the structured output of the AI provider.
type GenerateQuizResponse struct {
	Difficulty string
	Questions  []Question
}

// AIClient abstracts any AI backend (real LLM, stub, HTTP service, ...).
//
// The quiz service depends on this interface only, so the workshop stub
// can be swapped for a real provider without touching quiz/service.go.
type AIClient interface {
	GenerateQuiz(ctx context.Context, req GenerateQuizRequest) (GenerateQuizResponse, error)
}

// Validation errors. The service surfaces these so the workshop can see
// that the backend refuses to accept bad AI output.
var (
	ErrInvalidDifficulty    = errors.New("AI response has invalid difficulty")
	ErrEmptyQuestions       = errors.New("AI response has no questions")
	ErrEmptyQuestionText    = errors.New("AI response contains an empty question")
	ErrInvalidOptionCount   = errors.New("AI question must have exactly 4 options")
	ErrInvalidCorrectOption = errors.New("AI correct_option is out of range")
)

// ValidateResponse enforces the rules the backend demands from the AI.
//
// Principle: "AI generates. Backend governs."
// The AI may send whatever it wants; we reject anything that does not
// match our contract BEFORE it is persisted.
//
// Rules:
//   - difficulty must be easy, medium, or hard
//   - there must be at least one question
//   - every question must have non-empty text
//   - every question must have exactly 4 options
//   - correct_option must point to a valid option index
func ValidateResponse(resp GenerateQuizResponse) error {
	switch resp.Difficulty {
	case "easy", "medium", "hard":
		// valid
	default:
		return fmt.Errorf("%w: %q", ErrInvalidDifficulty, resp.Difficulty)
	}

	if len(resp.Questions) == 0 {
		return ErrEmptyQuestions
	}

	for i, q := range resp.Questions {
		if strings.TrimSpace(q.Question) == "" {
			return fmt.Errorf("%w (question %d)", ErrEmptyQuestionText, i+1)
		}
		if len(q.Options) != 4 {
			return fmt.Errorf("%w (question %d, got %d)", ErrInvalidOptionCount, i+1, len(q.Options))
		}
		if q.CorrectOption < 0 || q.CorrectOption >= len(q.Options) {
			return fmt.Errorf("%w (question %d, got %d)", ErrInvalidCorrectOption, i+1, q.CorrectOption)
		}
	}

	return nil
}
