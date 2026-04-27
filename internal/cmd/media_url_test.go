package cmd

import "testing"

func TestStableMediaURLRewritesLoopbackMediaURL(t *testing.T) {
	got := stableMediaURL("http://127.0.0.1:18080/v1/media/med_demo/content")
	want := "https://server.popi.art/v1/media/med_demo/content"
	if got != want {
		t.Fatalf("stableMediaURL() = %q, want %q", got, want)
	}
}

func TestStableMediaURLKeepsPublicURL(t *testing.T) {
	raw := "https://media.popi.test/v1/media/med_demo/content"
	if got := stableMediaURL(raw); got != raw {
		t.Fatalf("stableMediaURL() = %q, want %q", got, raw)
	}
}

func TestStableMediaURLRewritesHTTPMediaURL(t *testing.T) {
	got := stableMediaURL("http://101.42.99.35:18080/v1/media/med_demo/content")
	want := "https://server.popi.art/v1/media/med_demo/content"
	if got != want {
		t.Fatalf("stableMediaURL() = %q, want %q", got, want)
	}
}

func TestStableMediaURLTurnsRelativeMediaPathIntoPublicURL(t *testing.T) {
	got := stableMediaURL("/media/med_demo/content")
	want := "https://server.popi.art/v1/media/med_demo/content"
	if got != want {
		t.Fatalf("stableMediaURL() = %q, want %q", got, want)
	}
}
