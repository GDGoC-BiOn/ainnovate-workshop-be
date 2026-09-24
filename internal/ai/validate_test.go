package ai

import (
	"errors"
	"testing"
)

func TestValidateResponse_Valid(t *testing.T) {
	resp := GenerateQuizResponse{
		Difficulty: "medium",
		Questions: []Question{
			{
				Question:      "What does a hash function do?",
				Options:       []string{"a", "b", "c", "d"},
				CorrectOption: 1,
			},
		},
	}

	if err := ValidateResponse(resp); err != nil {
		t.Fatalf("expected valid response, got error: %v", err)
	}
}

func TestValidateResponse_AcceptsAllDifficulties(t *testing.T) {
	for _, difficulty := range []string{"easy", "medium", "hard"} {
		resp := GenerateQuizResponse{
			Difficulty: difficulty,
			Questions: []Question{
				{Question: "q", Options: []string{"a", "b", "c", "d"}, CorrectOption: 0},
			},
		}
		if err := ValidateResponse(resp); err != nil {
			t.Errorf("difficulty %q should be accepted: %v", difficulty, err)
		}
	}
}

func TestValidateResponse_InvalidDifficulty(t *testing.T) {
	resp := GenerateQuizResponse{
		Difficulty: "extreme",
		Questions:  []Question{{Question: "q", Options: []string{"a", "b", "c", "d"}, CorrectOption: 0}},
	}

	err := ValidateResponse(resp)
	if !errors.Is(err, ErrInvalidDifficulty) {
		t.Fatalf("expected ErrInvalidDifficulty, got: %v", err)
	}
}

func TestValidateResponse_EmptyQuestions(t *testing.T) {
	resp := GenerateQuizResponse{
		Difficulty: "easy",
		Questions:  nil,
	}

	err := ValidateResponse(resp)
	if !errors.Is(err, ErrEmptyQuestions) {
		t.Fatalf("expected ErrEmptyQuestions, got: %v", err)
	}
}

func TestValidateResponse_EmptyQuestionText(t *testing.T) {
	resp := GenerateQuizResponse{
		Difficulty: "easy",
		Questions: []Question{
			{Question: "   ", Options: []string{"a", "b", "c", "d"}, CorrectOption: 0},
		},
	}

	err := ValidateResponse(resp)
	if !errors.Is(err, ErrEmptyQuestionText) {
		t.Fatalf("expected ErrEmptyQuestionText, got: %v", err)
	}
}

func TestValidateResponse_WrongOptionCount(t *testing.T) {
	resp := GenerateQuizResponse{
		Difficulty: "easy",
		Questions: []Question{
			{Question: "q", Options: []string{"a", "b", "c"}, CorrectOption: 0},
		},
	}

	err := ValidateResponse(resp)
	if !errors.Is(err, ErrInvalidOptionCount) {
		t.Fatalf("expected ErrInvalidOptionCount, got: %v", err)
	}
}

func TestValidateResponse_InvalidCorrectOption(t *testing.T) {
	cases := []int{-1, 4, 99}
	for _, correctOption := range cases {
		resp := GenerateQuizResponse{
			Difficulty: "easy",
			Questions: []Question{
				{Question: "q", Options: []string{"a", "b", "c", "d"}, CorrectOption: correctOption},
			},
		}

		err := ValidateResponse(resp)
		if !errors.Is(err, ErrInvalidCorrectOption) {
			t.Errorf("correct_option %d: expected ErrInvalidCorrectOption, got: %v", correctOption, err)
		}
	}
}
