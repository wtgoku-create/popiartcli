package popiart

import (
	"net"
	"net/url"
	"strings"
)

// PublicMediaBaseURL 是对外展示稳定媒体地址时统一使用的公开域名。
const PublicMediaBaseURL = "https://server.popi.art"

// StableMediaURL 把本地、回环或旧格式媒体地址归一化成公开稳定地址。
func StableMediaURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	if strings.HasPrefix(raw, "/") {
		return PublicMediaBaseURL + stableMediaPath(raw)
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
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
