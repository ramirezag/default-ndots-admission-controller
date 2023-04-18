package internal_test

import (
	"default-ndots-admission-controller/internal"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHealth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	handlers := internal.NewHandlers(2, time.Second*2)
	handlers.ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
		return
	}
	assert.Equal(t, "{\"health\":\"UP\"}", string(data))
}
