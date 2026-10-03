package service

import (
	"testing"

	"gitlab.com/tozd/go/errors"
)

func TestUrlPath_Coverage(t *testing.T) {
	cases := []struct{ in, want string }{
		{"https://example.com/v1/start?foo=bar", "/v1/start"},
		{"https://example.com/v1/session/sess-123?x=1", "/v1/session/sess-123"},
		{"https://example.com", "/"},
		{"https://example.com/", "/"},
		{"", "/"},
		{"/v1/start", "/v1/start"},
		{"v1/start", "/v1/start"},
		{"/v1/session/abc?x=1&y=2", "/v1/session/abc"},
		{"v1/session/abc?x=1", "/v1/session/abc"},
		{"/", "/"},
		{"http://example.com:8080/v1/clients", "/v1/clients"},
	}
	for _, tc := range cases {
		if got := urlPath(tc.in); got != tc.want {
			t.Fatalf("urlPath(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestIsAlreadyExists_Coverage(t *testing.T) {
	if isAlreadyExists(nil) {
		t.Fatal("nil should be false")
	}
	if !isAlreadyExists(errors.Errorf("register client failed 409: {\"error\":\"already registered\"}")) {
		t.Fatal("should detect 409")
	}
	if !isAlreadyExists(errors.Errorf("already registered")) {
		t.Fatal("should detect already registered")
	}
	if isAlreadyExists(errors.Errorf("register client failed 500: server error")) {
		t.Fatal("500 should be false")
	}
}
