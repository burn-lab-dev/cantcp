package domain

import (
	"net"
	"net/url"
	"strconv"
)

// checkHostPort validates a "host:port" address and returns the host part.
// The port must be in 1..65535; the host may be empty (all interfaces), a
// host name or an IP address. The field name is used in error messages.
func checkHostPort(field, addr string) (string, Error) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return "", ErrInvalid("%s: %q is not a host:port address", field, addr)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		return "", ErrInvalid("%s: %q has an invalid port (want 1..65535)", field, addr)
	}
	return host, nil
}

// checkHTTPURL validates an http:// or https:// URL without a query or a
// fragment.
func checkHTTPURL(field, raw string) Error {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return ErrInvalid("%s: %q is not an http:// or https:// URL", field, raw)
	}
	return nil
}

// hostIsLoopback reports whether the host part of an address is a loopback
// host: "localhost", a loopback IP, or empty (all interfaces, never loopback).
func hostIsLoopback(host string) bool {
	if host == "" {
		return false
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
