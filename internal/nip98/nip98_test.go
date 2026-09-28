package nip98

import (
	"testing"
	"time"

	"github.com/lnd3/persona/internal/identity"
)

func TestBuildAndVerifyRoundTrip(t *testing.T) {
	p, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}

	const url = "https://example.com/login"
	const method = "POST"

	header, err := BuildAuthHeader(p, url, method)
	if err != nil {
		t.Fatalf("BuildAuthHeader: %v", err)
	}

	pubkey, err := Verify(header, url, method, DefaultMaxAge)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if pubkey != p.PublicKey {
		t.Errorf("Verify returned pubkey %s, want %s", pubkey, p.PublicKey)
	}
}

func TestVerifyRejectsWrongURL(t *testing.T) {
	p, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	header, err := BuildAuthHeader(p, "https://example.com/login", "POST")
	if err != nil {
		t.Fatalf("BuildAuthHeader: %v", err)
	}
	if _, err := Verify(header, "https://example.com/other", "POST", DefaultMaxAge); err == nil {
		t.Fatal("expected error for mismatched url")
	}
}

func TestVerifyRejectsWrongMethod(t *testing.T) {
	p, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	header, err := BuildAuthHeader(p, "https://example.com/login", "POST")
	if err != nil {
		t.Fatalf("BuildAuthHeader: %v", err)
	}
	if _, err := Verify(header, "https://example.com/login", "GET", DefaultMaxAge); err == nil {
		t.Fatal("expected error for mismatched method")
	}
}

func TestVerifyRejectsStaleEvent(t *testing.T) {
	p, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	header, err := BuildAuthHeader(p, "https://example.com/login", "POST")
	if err != nil {
		t.Fatalf("BuildAuthHeader: %v", err)
	}
	time.Sleep(20 * time.Millisecond)
	if _, err := Verify(header, "https://example.com/login", "POST", 1*time.Millisecond); err == nil {
		t.Fatal("expected error for event older than max age")
	}
}
