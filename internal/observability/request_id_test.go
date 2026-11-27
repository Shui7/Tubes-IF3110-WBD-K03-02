package observability

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestIDMiddlewareAddsResponseHeaderAndContext(t *testing.T) {
	var contextID string
	handler := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contextID = RequestIDFromContext(r.Context())
		w.WriteHeader(http.StatusNoContent)
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))

	headerID := response.Header().Get(RequestIDHeader)
	if response.Code != http.StatusNoContent || len(headerID) != 32 || contextID != headerID {
		t.Fatalf("status=%d header=%q context=%q", response.Code, headerID, contextID)
	}
}
