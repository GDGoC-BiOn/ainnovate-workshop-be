package quiz

import (
	"errors"
	"net/http"

	"arahin-mini/internal/platform"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Handler translates HTTP into service calls and back. It must stay thin:
// no SQL, no AI logic — only request parsing, validation, and responses.
type Handler struct {
	service *Service
}

// NewHandler wires the HTTP layer to the service (manual DI).
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Routes returns the chi router for all /v1 quiz endpoints.
func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()

	r.Post("/quiz-requests", h.handlePostQuizRequest)
	r.Get("/quiz-requests/{requestID}", h.handleGetQuizRequest)
	r.Get("/quizzes/{quizID}", h.handleGetQuiz)

	return r
}

// createQuizRequestBody is the POST /v1/quiz-requests contract.
type createQuizRequestBody struct {
	UserID   uuid.UUID `json:"user_id"`
	LessonID uuid.UUID `json:"lesson_id"`
}

// handlePostQuizRequest starts quiz generation and returns HTTP 202 quickly.
//
// The key teaching point: the HTTP request is NOT the AI operation.
// POST returns as soon as the job is accepted; the heavy work happens in a
// background goroutine and is polled via GET /v1/quiz-requests/{id}.
func (h *Handler) handlePostQuizRequest(w http.ResponseWriter, r *http.Request) {
	var body createQuizRequestBody

	if err := platform.DecodeJSON(w, r, &body); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	if body.UserID == uuid.Nil {
		platform.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "user_id is required")
		return
	}
	if body.LessonID == uuid.Nil {
		platform.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "lesson_id is required")
		return
	}

	req, err := h.service.StartGeneration(r.Context(), body.UserID, body.LessonID)
	if err != nil {
		platform.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not start quiz generation")
		return
	}

	platform.WriteJSON(w, http.StatusAccepted, req)
}

// handleGetQuizRequest returns the status of a quiz-generation job.
func (h *Handler) handleGetQuizRequest(w http.ResponseWriter, r *http.Request) {
	requestID, err := uuid.Parse(chi.URLParam(r, "requestID"))
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request_id")
		return
	}

	req, err := h.service.GetQuizRequest(r.Context(), requestID)
	if errors.Is(err, ErrRequestNotFound) {
		platform.WriteError(w, http.StatusNotFound, "NOT_FOUND", "quiz request not found")
		return
	}
	if err != nil {
		platform.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not read quiz request")
		return
	}

	platform.WriteJSON(w, http.StatusOK, req)
}

// handleGetQuiz returns a generated quiz WITHOUT the answer key.
func (h *Handler) handleGetQuiz(w http.ResponseWriter, r *http.Request) {
	quizID, err := uuid.Parse(chi.URLParam(r, "quizID"))
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid quiz_id")
		return
	}

	quiz, err := h.service.GetQuiz(r.Context(), quizID)
	if errors.Is(err, ErrQuizNotFound) {
		platform.WriteError(w, http.StatusNotFound, "NOT_FOUND", "quiz not found")
		return
	}
	if err != nil {
		platform.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not read quiz")
		return
	}

	platform.WriteJSON(w, http.StatusOK, quiz)
}
