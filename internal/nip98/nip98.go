// Package nip98 implements NIP-98 HTTP Auth (signed Nostr events as
// an HTTP Authorization header) — D001's base sign-in mechanism: a
// site challenges, the persona's client signs, the site verifies. No
// billing/payment gate here; that's a separate, later action.
//
// go-nostr (v0.52.3) only exposes the kind-27235 constant, not a
// challenge/sign/verify flow, so this package implements NIP-98
// itself on top of the base Event type.
package nip98

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/nbd-wtf/go-nostr"

	"github.com/lnd3/persona/internal/identity"
)

// Kind is NIP-98's HTTP Auth event kind.
const Kind = 27235

// DefaultMaxAge is how far a NIP-98 event's created_at may drift from
// "now" and still be accepted, per NIP-98's own recommendation.
const DefaultMaxAge = 60 * time.Second

// BuildAuthHeader signs a NIP-98 event for the given absolute URL and
// HTTP method, and returns it already formatted as the value of an
// Authorization header.
func BuildAuthHeader(p identity.Persona, url, method string) (string, error) {
	evt := &nostr.Event{
		Kind:      Kind,
		CreatedAt: nostr.Now(),
		Content:   "",
		Tags: nostr.Tags{
			{"u", url},
			{"method", method},
		},
	}
	if err := evt.Sign(p.PrivateKey); err != nil {
		return "", err
	}
	j, err := json.Marshal(evt)
	if err != nil {
		return "", err
	}
	return "Nostr " + base64.StdEncoding.EncodeToString(j), nil
}

// Verify checks a NIP-98 Authorization header value against the
// expected URL and method, and returns the signing pubkey on success.
func Verify(header, expectedURL, expectedMethod string, maxAge time.Duration) (string, error) {
	const prefix = "Nostr "
	if len(header) <= len(prefix) || header[:len(prefix)] != prefix {
		return "", errors.New("nip98: missing \"Nostr \" prefix")
	}
	raw, err := base64.StdEncoding.DecodeString(header[len(prefix):])
	if err != nil {
		return "", fmt.Errorf("nip98: invalid base64: %w", err)
	}

	var evt nostr.Event
	if err := json.Unmarshal(raw, &evt); err != nil {
		return "", fmt.Errorf("nip98: invalid event JSON: %w", err)
	}

	if evt.Kind != Kind {
		return "", fmt.Errorf("nip98: unexpected kind %d, want %d", evt.Kind, Kind)
	}

	ok, err := evt.CheckSignature()
	if err != nil {
		return "", fmt.Errorf("nip98: signature check failed: %w", err)
	}
	if !ok {
		return "", errors.New("nip98: invalid signature")
	}

	if evt.Tags.Find("u").Value() != expectedURL {
		return "", errors.New("nip98: url tag does not match")
	}
	if evt.Tags.Find("method").Value() != expectedMethod {
		return "", errors.New("nip98: method tag does not match")
	}

	if maxAge <= 0 {
		maxAge = DefaultMaxAge
	}
	age := time.Since(evt.CreatedAt.Time())
	if age < -maxAge || age > maxAge {
		return "", fmt.Errorf("nip98: created_at outside allowed window (%s)", age)
	}

	return evt.PubKey, nil
}
