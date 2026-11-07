package httpserver

import (
	"context"
	"net/http/httptest"
	"testing"
)

func TestClientIPRequiresTrustedDirectPeer(t *testing.T) {
	request := httptest.NewRequestWithContext(context.Background(), "GET", "/", nil)
	request.RemoteAddr = "203.0.113.10:4321"
	request.Header.Set("X-Forwarded-For", "198.51.100.20")

	if got := ClientIP(request, nil); got != "203.0.113.10" {
		t.Fatalf("default result = %q, want direct peer", got)
	}
	if got := ClientIP(request, []string{"10.0.0.0/8"}); got != "203.0.113.10" {
		t.Fatalf("untrusted peer result = %q, want direct peer", got)
	}
}

func TestClientIPWalksTrustedProxyChainFromRight(t *testing.T) {
	request := httptest.NewRequestWithContext(context.Background(), "GET", "/", nil)
	request.RemoteAddr = "10.0.0.3:4321"
	request.Header.Set("X-Forwarded-For", "198.51.100.20, 10.0.0.2, 10.0.0.1")

	got := ClientIP(request, []string{"10.0.0.0/8"})
	if got != "198.51.100.20" {
		t.Fatalf("result = %q, want client address", got)
	}
}

func TestClientIPStopsAtFirstUntrustedForwarder(t *testing.T) {
	request := httptest.NewRequestWithContext(context.Background(), "GET", "/", nil)
	request.RemoteAddr = "10.0.0.3:4321"
	request.Header.Set("X-Forwarded-For", "198.51.100.20, 203.0.113.9")

	got := ClientIP(request, []string{"10.0.0.0/8"})
	if got != "203.0.113.9" {
		t.Fatalf("result = %q, want nearest untrusted forwarder", got)
	}
}

func TestClientIPRejectsMalformedForwardedChain(t *testing.T) {
	request := httptest.NewRequestWithContext(context.Background(), "GET", "/", nil)
	request.RemoteAddr = "10.0.0.3:4321"
	request.Header.Set("X-Forwarded-For", "198.51.100.20, not-an-ip")

	got := ClientIP(request, []string{"10.0.0.0/8"})
	if got != "10.0.0.3" {
		t.Fatalf("result = %q, want trusted direct peer after malformed chain", got)
	}
}

func TestClientIPSupportsMultipleHeadersAndIPv6(t *testing.T) {
	request := httptest.NewRequestWithContext(context.Background(), "GET", "/", nil)
	request.RemoteAddr = "[2001:db8::3]:4321"
	request.Header.Add("X-Forwarded-For", "2001:db8:1::20")
	request.Header.Add("X-Forwarded-For", "2001:db8::2")

	got := ClientIP(request, []string{"2001:db8::/32"})
	if got != "2001:db8:1::20" {
		t.Fatalf("result = %q, want IPv6 client address", got)
	}
}
