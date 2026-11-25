// Package oauth makes the MCP address an OAuth 2.1 authorization server, as the MCP specification describes: the
// metadata, an open dynamic client registration, the consent page and the token endpoint. A connection is a row of
// tokens of kind oauth: listed, revoked and read-only like an API token.
package oauth

import (
	"net"
	"net/http"
	"strings"
)

// Issuer is the public URL of the MCP address, as the client reached it: the tunnel or proxy in front gives HTTPS and
// keeps the Host. Plain HTTP on the loopback only, for a test on this machine.
func Issuer(r *http.Request) string {
	scheme := "https"
	if r.TLS == nil && loopback(hostname(r.Host)) {
		scheme = "http"
	}
	return scheme + "://" + r.Host
}

// Challenge is the WWW-Authenticate of a 401 on /mcp: where a client finds how to sign in (RFC 9728).
func Challenge(r *http.Request) string {
	return `Bearer resource_metadata="` + Issuer(r) + `/.well-known/oauth-protected-resource"`
}

func loopback(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// hostname drops the port and the brackets of an IPv6 host.
func hostname(host string) string {
	if name, _, err := net.SplitHostPort(host); err == nil {
		return name
	}
	return strings.Trim(host, "[]")
}
