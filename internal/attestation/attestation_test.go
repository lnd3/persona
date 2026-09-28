package attestation

import (
	"context"
	"testing"
	"time"

	"github.com/nbd-wtf/go-nostr"

	"github.com/lnd3/persona/internal/identity"
)

func mustPersona(t *testing.T) identity.Persona {
	t.Helper()
	p, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	return p
}

func TestNewAndVerify(t *testing.T) {
	attester := mustPersona(t)
	subject := mustPersona(t)

	evt, err := New(attester, subject.PublicKey, "org.example.skill.rust", "expert", "vouches for Rust skill")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	claim, err := Verify(evt)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if claim.AttesterKey != attester.PublicKey {
		t.Errorf("AttesterKey = %s, want %s", claim.AttesterKey, attester.PublicKey)
	}
	if claim.SubjectKey != subject.PublicKey {
		t.Errorf("SubjectKey = %s, want %s", claim.SubjectKey, subject.PublicKey)
	}
	if claim.ClaimType != "org.example.skill.rust" {
		t.Errorf("ClaimType = %s", claim.ClaimType)
	}
	if claim.ClaimValue != "expert" {
		t.Errorf("ClaimValue = %s", claim.ClaimValue)
	}
	if claim.Summary != "vouches for Rust skill" {
		t.Errorf("Summary = %s", claim.Summary)
	}
}

func TestRejectNonNamespacedClaimType(t *testing.T) {
	attester := mustPersona(t)
	subject := mustPersona(t)

	_, err := New(attester, subject.PublicKey, "skill", "expert", "")
	if err == nil {
		t.Fatal("expected error for non-namespaced claim_type, got nil")
	}
}

func TestVerifyRejectsTamperedEvent(t *testing.T) {
	attester := mustPersona(t)
	subject := mustPersona(t)

	evt, err := New(attester, subject.PublicKey, "org.example.skill.rust", "expert", "")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	evt.Content = "tampered"

	if _, err := Verify(evt); err == nil {
		t.Fatal("expected signature verification to fail on tampered event")
	}
}

// TestPublishAndFetchRoundTrip hits a real public Nostr relay, per
// D001's "runs on the public Nostr relay network" resolution. Skips
// gracefully if the relay is unreachable, rather than failing the
// suite in offline/firewalled environments.
func TestPublishAndFetchRoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network round trip in -short mode")
	}

	const relayURL = "wss://relay.damus.io"

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	probe, err := nostr.RelayConnect(ctx, relayURL)
	if err != nil {
		t.Skipf("relay %s unreachable, skipping: %v", relayURL, err)
	}
	probe.Close()

	attester := mustPersona(t)
	subject := mustPersona(t)

	evt, err := New(attester, subject.PublicKey, "net.persona.test.roundtrip", "ok", "A001 round-trip test")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := Publish(ctx, evt, []string{relayURL}); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	// Relays may take a moment to index a just-published event before
	// it's queryable back.
	var events []*nostr.Event
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		events, err = FetchForSubject(ctx, relayURL, subject.PublicKey)
		if err != nil {
			t.Fatalf("FetchForSubject: %v", err)
		}
		if len(events) > 0 {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	found := false
	for _, e := range events {
		if e.ID == evt.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("published event %s not found in %d fetched events", evt.ID, len(events))
	}
}
