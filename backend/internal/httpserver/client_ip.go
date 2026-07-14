// SPDX-License-Identifier: MIT

package httpserver

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// NormalizeClientIP replaces RemoteAddr with the forwarded client IP only when
// the direct peer belongs to an explicitly trusted proxy network. Otherwise it
// preserves RemoteAddr so an upgrade without proxy configuration keeps the
// existing rate-limit key behaviour.
func NormalizeClientIP(trustedProxies []netip.Prefix) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			peer, ok := parseAddress(r.RemoteAddr)
			if !ok {
				next.ServeHTTP(w, r)
				return
			}
			if addressIsTrusted(peer, trustedProxies) {
				if forwarded := forwardedAddresses(r); len(forwarded) > 0 {
					r.RemoteAddr = rightmostUntrusted(forwarded, trustedProxies).String()
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func forwardedAddresses(r *http.Request) []netip.Addr {
	values := make([]netip.Addr, 0, 4)
	for _, raw := range strings.Split(r.Header.Get("X-Forwarded-For"), ",") {
		if addr, ok := parseAddress(raw); ok {
			values = append(values, addr)
		}
	}
	if len(values) > 0 {
		return values
	}
	for _, element := range strings.Split(r.Header.Get("Forwarded"), ",") {
		for _, parameter := range strings.Split(element, ";") {
			key, value, found := strings.Cut(parameter, "=")
			if !found || !strings.EqualFold(strings.TrimSpace(key), "for") {
				continue
			}
			if addr, ok := parseAddress(value); ok {
				values = append(values, addr)
			}
		}
	}
	return values
}

func rightmostUntrusted(addresses []netip.Addr, trusted []netip.Prefix) netip.Addr {
	for index := len(addresses) - 1; index >= 0; index-- {
		if !addressIsTrusted(addresses[index], trusted) {
			return addresses[index]
		}
	}
	return addresses[0]
}

func addressIsTrusted(address netip.Addr, trusted []netip.Prefix) bool {
	address = address.Unmap()
	for _, prefix := range trusted {
		if prefix.Contains(address) {
			return true
		}
	}
	return false
}

func parseAddress(value string) (netip.Addr, bool) {
	value = strings.Trim(strings.TrimSpace(value), `"`)
	if value == "" || strings.EqualFold(value, "unknown") || strings.HasPrefix(value, "_") {
		return netip.Addr{}, false
	}
	if host, _, err := net.SplitHostPort(value); err == nil {
		value = host
	}
	value = strings.Trim(value, "[]")
	address, err := netip.ParseAddr(value)
	if err != nil {
		return netip.Addr{}, false
	}
	return address.Unmap(), true
}
