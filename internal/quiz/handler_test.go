package quiz

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func newTestHandler() (*Handler, *fakeRepo) {
	svc, repo := newTestService()
	return NewHandler(svc), repo
}

func doRequest(t *testing.T, h *Handler, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()

	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, target, nil)
	} else {
		req = httptest.NewRequest(method, target, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}

	rec := httptest.NewRecorder()
	h.Routes().ServeHTTP(rec, req)
	return rec
}

func decodeBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not valid JSON: %v\nbody: %s", err, rec.Body.String())
	}
	return body
}

// SECURITY test: the answer key must never appear in the API response.
func TestGetQuiz_HidesAnswerKey(t *testing.T) {
	h, repo := newTestHandler()
	repo.quiz = Quiz{
		ID:         uuid.New(),
		Difficulty: "medium",
		Questions: []Question{
			{
				ID:            uuid.New(),
				Text:          "What does a hash function do?",
				Options:       []string{"A", "B", "C", "D"},
				CorrectOption: 2, // <- must never leak
			},
		},
	}

	rec := doRequest(t, h, http.MethodGet, "/quizzes/"+repo.quiz.ID.String(), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	raw := rec.Body.String()
	if strings.Contains(raw, "correct_option") {
		t.Fatalf("answer key leaked into response: %s", raw)
	}
	if strings.Contains(raw, "CorrectOption") {
		t.Fatalf("answer key leaked into response: %s", raw)
	}

	body := decodeBody(t, rec)
	if body["difficulty"] != "medium" {
		t.Fatalf("expected difficulty medium, got %v", body["difficulty"])
	}

	questions, ok := body["questions"].([]any)
	if !ok || len(questions) != 1 {
		t.Fatalf("expected 1 question, got: %v", body["questions"])
	}
}

func TestGetQuizEndpoint_NotFound(t *testing.T) {
	h, repo := newTestHandler()
	repo.quizErr = ErrQuizNotFound

	rec := doRequest(t, h, http.MethodGet, "/quizzes/"+uuid.New().String(), "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}

	body := decodeBody(t, rec)
	errObj, ok := body["error"].(map[string]any)
	if !ok || errObj["code"] != "NOT_FOUND" {
		t.Fatalf("expected error code NOT_FOUND, got: %v", body)
	}
}

func TestGetQuiz_InvalidID(t *testing.T) {
	h, _ := newTestHandler()

	rec := doRequest(t, h, http.MethodGet, "/quizzes/not-a-uuid", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestPostQuizRequest_Accepted(t *testing.T) {
	h, _ := newTestHandler()

	body := `{"user_id":"` + testUserID + `","lesson_id":"` + testLessonID + `"}`
	rec := doRequest(t, h, http.MethodPost, "/quiz-requests", body)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}

	resp := decodeBody(t, rec)
	if resp["status"] != "processing" {
		t.Fatalf("expected status processing, got %v", resp["status"])
	}
	if _, ok := resp["request_id"].(string); !ok {
		t.Fatalf("expected a request_id in the response: %v", resp)
	}
}

func TestPostQuizRequest_MissingUserID(t *testing.T) {
	h, _ := newTestHandler()

	body := `{"lesson_id":"` + testLessonID + `"}`
	rec := doRequest(t, h, http.MethodPost, "/quiz-requests", body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	bodyOut := decodeBody(t, rec)
	errObj, ok := bodyOut["error"].(map[string]any)
	if !ok || errObj["code"] != "INVALID_REQUEST" {
		t.Fatalf("expected error code INVALID_REQUEST, got: %v", bodyOut)
	}
}

func TestPostQuizRequest_InvalidUUID(t *testing.T) {
	h, _ := newTestHandler()

	body := `{"user_id":"not-a-uuid","lesson_id":"` + testLessonID + `"}`
	rec := doRequest(t, h, http.MethodPost, "/quiz-requests", body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestPostQuizRequest_MalformedJSON(t *testing.T) {
	h, _ := newTestHandler()

	rec := doRequest(t, h, http.MethodPost, "/quiz-requests", "{this is not json")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestGetQuizRequestEndpoint_UnknownID(t *testing.T) {
	h, _ := newTestHandler()

	rec := doRequest(t, h, http.MethodGet, "/quiz-requests/"+uuid.New().String(), "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}
