package cmd

import (
	"net"
	"net/url"
	"strings"
)

const publicMediaBaseURL = "https://server.popi.art"

func stableMediaURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	if strings.HasPrefix(raw, "/") {
		return publicMediaBaseURL + stableMediaPath(raw)
	}

	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return raw
	}
	if !isLoopbackHost(u.Hostname()) && !isHTTPMediaURL(u) {
		return raw
	}

	u.Scheme = "https"
	u.Host = "server.popi.art"
	u.Path = stableMediaPath(u.Path)
	return u.String()
}

func addStableURLFields(result map[string]any, rawURL string) {
	stableURL := stableMediaURL(rawURL)
	if stableURL == "" {
		return
	}
	result["url"] = stableURL
	result["stable_url"] = stableURL
	result["public_url"] = stableURL
	if rawURL != "" && rawURL != stableURL {
		result["original_url"] = rawURL
	}
}

func stableMediaPath(path string) string {
	switch {
	case strings.HasPrefix(path, "/v1/media/"), strings.HasPrefix(path, "/v1/artifacts/"):
		return path
	case strings.HasPrefix(path, "/media/"), strings.HasPrefix(path, "/artifacts/"):
		return "/v1" + path
	default:
		return path
	}
}

func isHTTPMediaURL(u *url.URL) bool {
	return strings.EqualFold(u.Scheme, "http") &&
		(strings.HasPrefix(u.Path, "/v1/media/") ||
			strings.HasPrefix(u.Path, "/media/") ||
			strings.HasPrefix(u.Path, "/v1/artifacts/") ||
			strings.HasPrefix(u.Path, "/artifacts/"))
}

func isLoopbackHost(host string) bool {
	switch strings.ToLower(host) {
	case "localhost":
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
