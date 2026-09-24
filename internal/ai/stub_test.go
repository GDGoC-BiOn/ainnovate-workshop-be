package ai

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestStubClient_ReturnsValidResponse(t *testing.T) {
	client := NewStubClient(StubOptions{})

	resp, err := client.GenerateQuiz(context.Background(), GenerateQuizRequest{
		Objective: "Understand how hash functions map input data to a fixed-size value.",
		Mastery:   40,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := ValidateResponse(resp); err != nil {
		t.Fatalf("stub returned an invalid response: %v", err)
	}
	if len(resp.Questions) == 0 {
		t.Fatal("expected at least one question")
	}
}

func TestStubClient_UsesObjectiveInOutput(t *testing.T) {
	const objective = "Understand how hash functions map input data to a fixed-size value."
	client := NewStubClient(StubOptions{})

	resp, err := client.GenerateQuiz(context.Background(), GenerateQuizRequest{
		Objective: objective,
		Mastery:   40,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, q := range resp.Questions {
		if strings.Contains(q.Question, objective) {
			found = true
			break
		}
		for _, opt := range q.Options {
			if strings.Contains(opt, objective) {
				found = true
				break
			}
		}
	}

	if !found {
		t.Fatal("expected the stub output to reference the lesson objective")
	}
}

func TestStubClient_DifficultyFollowsMastery(t *testing.T) {
	cases := []struct {
		mastery int
		want    string
	}{
		{mastery: 10, want: "easy"},
		{mastery: 40, want: "medium"},
		{mastery: 90, want: "hard"},
	}

	for _, tc := range cases {
		client := NewStubClient(StubOptions{})
		resp, err := client.GenerateQuiz(context.Background(), GenerateQuizRequest{
			Objective: "obj",
			Mastery:   tc.mastery,
		})
		if err != nil {
			t.Fatalf("mastery %d: unexpected error: %v", tc.mastery, err)
		}
		if resp.Difficulty != tc.want {
			t.Errorf("mastery %d: got difficulty %q, want %q", tc.mastery, resp.Difficulty, tc.want)
		}
	}
}

func TestStubClient_IsDeterministic(t *testing.T) {
	client := NewStubClient(StubOptions{})
	req := GenerateQuizRequest{Objective: "obj", Mastery: 40}

	first, err := client.GenerateQuiz(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	second, err := client.GenerateQuiz(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if first.Difficulty != second.Difficulty || len(first.Questions) != len(second.Questions) {
		t.Fatal("expected deterministic output for the same input")
	}
	for i := range first.Questions {
		if first.Questions[i].Question != second.Questions[i].Question {
			t.Fatalf("question %d differs between calls", i)
		}
		if first.Questions[i].CorrectOption != second.Questions[i].CorrectOption {
			t.Fatalf("question %d correct option differs between calls", i)
		}
	}
}

func TestStubClient_RespectsContextCancellation(t *testing.T) {
	client := NewStubClient(StubOptions{Delay: 500 * time.Millisecond})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	start := time.Now()
	_, err := client.GenerateQuiz(ctx, GenerateQuizRequest{Objective: "obj", Mastery: 40})
	if err == nil {
		t.Fatal("expected an error when the context is cancelled")
	}
	if time.Since(start) > 250*time.Millisecond {
		t.Fatal("expected cancellation to return promptly, not after the full delay")
	}
}
