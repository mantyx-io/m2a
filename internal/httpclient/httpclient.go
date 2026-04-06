package httpclient

import (
	"net/http"
	"time"
)

// WithHeaders returns an HTTP client that sets the given headers on each outgoing request
// when those headers are not already present.
func WithHeaders(headers map[string]string) *http.Client {
	return &http.Client{
		Timeout: 5 * time.Minute,
		Transport: &headerRoundTripper{
			base:    http.DefaultTransport,
			headers: headers,
		},
	}
}

type headerRoundTripper struct {
	base    http.RoundTripper
	headers map[string]string
}

func (h *headerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	for k, v := range h.headers {
		if req.Header.Get(k) == "" {
			req.Header.Set(k, v)
		}
	}
	return h.base.RoundTrip(req)
}
