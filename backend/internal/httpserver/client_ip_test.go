// SPDX-License-Identifier: MIT

package httpserver

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
)

func TestNormalizeClientIPIgnoresForwardingHeadersFromUntrustedPeer(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.9:54321"
	req.Header.Set("X-Forwarded-For", "198.51.100.7")

	got := remoteAddressSeenByHandler(t, nil, req)
	if got != "203.0.113.9:54321" {
		t.Fatalf("RemoteAddr = %q, want unchanged direct peer address", got)
	}
}

func TestNormalizeClientIPUsesRightmostUntrustedAddressFromTrustedProxyChain(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.2:443"
	req.Header.Set("X-Forwarded-For", "198.51.100.7, 10.0.0.3")

	got := remoteAddressSeenByHandler(t, []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")}, req)
	if got != "198.51.100.7" {
		t.Fatalf("normalized RemoteAddr = %q, want original client IP", got)
	}
}

func TestNormalizeClientIPSupportsForwardedHeaderFromTrustedProxy(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.2:443"
	req.Header.Set("Forwarded", `for=198.51.100.8;proto=https;host=auth.example.test`)

	got := remoteAddressSeenByHandler(t, []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")}, req)
	if got != "198.51.100.8" {
		t.Fatalf("normalized RemoteAddr = %q, want Forwarded client IP", got)
	}
}

func remoteAddressSeenByHandler(t *testing.T, trusted []netip.Prefix, req *http.Request) string {
	t.Helper()
	var got string
	handler := NormalizeClientIP(trusted)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = r.RemoteAddr
	}))
	handler.ServeHTTP(httptest.NewRecorder(), req)
	return got
}
