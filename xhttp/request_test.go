package xhttp

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequestStringer(t *testing.T) {
	r, err := http.NewRequest(http.MethodPost, "https://test.com/path?query=name#fragment=anchor", http.NoBody)
	r.Header.Add("Authentication", "Bearer secret")
	assert.NoError(t, err)
	r.RemoteAddr = "10.0.0.1:8080"

	t.Log(RequestStringer(r).String())
}
