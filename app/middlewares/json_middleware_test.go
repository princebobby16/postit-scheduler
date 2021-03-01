package middlewares

import (
	"net/http"
	"testing"
)

func TestJSONMiddleware(t *testing.T) {
	var newH http.Handler
	h := JSONMiddleware(newH)
	t.Log(h)
}
