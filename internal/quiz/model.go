package quiz

import (
	"time"

	"github.com/google/uuid"
)

// Lesson is a learning unit (e.g. "Understand Hash Function").
type Lesson struct {
	ID        uuid.UUID
	Title     string
	Objective string
	CreatedAt time.Time
}

// Mastery records how well a user knows a lesson on a 0..100 scale.
type Mastery struct {
	UserID   uuid.UUID
	LessonID uuid.UUID
	Score    int
}

// Quiz is a generated quiz with its multiple-choice questions.
type Quiz struct {
	ID         uuid.UUID  `json:"id"`
	UserID     uuid.UUID  `json:"-"`
	LessonID   uuid.UUID  `json:"-"`
	Difficulty string     `json:"difficulty"`
	CreatedAt  time.Time  `json:"-"`
	Questions  []Question `json:"questions"`
}

// Question is a single multiple-choice question.
//
// SECURITY NOTE: CorrectOption is serialized with `json:"-"`, so it can
// never leak into an API response, no matter where Quiz is marshaled.
// The correct answer stays server-side. The frontend only ever receives
// (id, question, options).
type Question struct {
	ID            uuid.UUID `json:"id"`
	QuizID        uuid.UUID `json:"-"`
	Text          string    `json:"question"`
	Options       []string  `json:"options"`
	CorrectOption int       `json:"-"`
	Position      int       `json:"-"`
	CreatedAt     time.Time `json:"-"`
}
