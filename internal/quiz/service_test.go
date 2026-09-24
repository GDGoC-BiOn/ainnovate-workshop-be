package quiz

import (
	"context"
	"errors"
	"testing"

	"arahin-mini/internal/ai"

	"github.com/google/uuid"
)

const (
	testUserID   = "11111111-1111-1111-1111-111111111111"
	testLessonID = "22222222-2222-2222-2222-222222222222"
)

// fakeRepo is an in-memory Repository used so the service can be tested
// without a real PostgreSQL instance.
type fakeRepo struct {
	lesson     Lesson
	mastery    Mastery
	quiz       Quiz
	lessonErr  error
	masteryErr error
	quizErr    error
}

func (f *fakeRepo) GetLesson(_ context.Context, _ uuid.UUID) (Lesson, error) {
	return f.lesson, f.lessonErr
}

func (f *fakeRepo) GetMastery(_ context.Context, _ uuid.UUID, _ uuid.UUID) (Mastery, error) {
	return f.mastery, f.masteryErr
}

func (f *fakeRepo) CreateQuiz(_ context.Context, params CreateQuizParams) (Quiz, error) {
	return f.quiz, f.quizErr
}

func (f *fakeRepo) GetQuiz(_ context.Context, _ uuid.UUID) (Quiz, error) {
	return f.quiz, f.quizErr
}

func newTestService() (*Service, *fakeRepo) {
	repo := &fakeRepo{
		lesson: Lesson{
			ID:        uuid.MustParse(testLessonID),
			Title:     "Understand Hash Function",
			Objective: "Understand how hash functions map input data to a fixed-size value.",
		},
		mastery: Mastery{UserID: uuid.MustParse(testUserID), LessonID: uuid.MustParse(testLessonID), Score: 40},
	}
	svc := NewService(repo, ai.NewStubClient(ai.StubOptions{}))
	return svc, repo
}

func TestStartGeneration_ReturnsProcessingJob(t *testing.T) {
	svc, _ := newTestService()

	req, err := svc.StartGeneration(context.Background(),
		uuid.MustParse(testUserID), uuid.MustParse(testLessonID))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if req.ID == uuid.Nil {
		t.Fatal("expected a non-zero request_id")
	}
	if req.Status != StatusProcessing {
		t.Fatalf("expected status %q, got %q", StatusProcessing, req.Status)
	}
}

func TestGetQuizRequest_NotFound(t *testing.T) {
	svc, _ := newTestService()

	_, err := svc.GetQuizRequest(context.Background(), uuid.New())
	if !errors.Is(err, ErrRequestNotFound) {
		t.Fatalf("expected ErrRequestNotFound, got: %v", err)
	}
}

func TestGetQuiz_ReturnsQuiz(t *testing.T) {
	svc, repo := newTestService()
	repo.quiz = Quiz{
		ID:         uuid.New(),
		Difficulty: "medium",
		Questions: []Question{
			{ID: uuid.New(), Text: "Q1", Options: []string{"a", "b", "c", "d"}, CorrectOption: 1},
		},
	}

	quiz, err := svc.GetQuiz(context.Background(), repo.quiz.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if quiz.Difficulty != "medium" {
		t.Fatalf("expected difficulty medium, got %q", quiz.Difficulty)
	}
	if len(quiz.Questions) != 1 {
		t.Fatalf("expected 1 question, got %d", len(quiz.Questions))
	}
}

func TestGetQuiz_NotFound(t *testing.T) {
	svc, repo := newTestService()
	repo.quizErr = ErrQuizNotFound

	_, err := svc.GetQuiz(context.Background(), uuid.New())
	if !errors.Is(err, ErrQuizNotFound) {
		t.Fatalf("expected ErrQuizNotFound, got: %v", err)
	}
}
