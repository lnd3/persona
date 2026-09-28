// Package attestation implements D001's attestation primitive: a
// signed claim (attester_key, subject_key, claim_type, claim_value,
// timestamp, signature) published as a Nostr event, and D001's
// claim_type namespace-governance resolution (reverse-domain
// namespacing, permission-free to mint).
package attestation

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/nbd-wtf/go-nostr"

	"github.com/lnd3/persona/internal/identity"
)

// Kind is this design's attestation event kind, decided 2026-09-28
// (see plan/actions/A001): regular-event range (1000-9999), not
// parameterized-replaceable — an attestation is a permanent,
// independent record, not "latest wins."
const Kind = 3300

// claimTypeRE enforces D001's reverse-domain namespacing convention
// for claim_type (e.g. "org.solemn.skill.rust"): at least two
// lowercase, dot-separated labels, so minting a new claim_type never
// collides with anyone else's without needing a registry to prevent it.
var claimTypeRE = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*(\.[a-z0-9]+(-[a-z0-9]+)*)+$`)

// ErrInvalidClaimType is returned when a claim_type isn't
// reverse-domain namespaced per D001's governance resolution.
var ErrInvalidClaimType = errors.New("attestation: claim_type must be reverse-domain namespaced (e.g. \"org.example.skill.rust\")")

// ValidateClaimType reports whether claimType follows the
// reverse-domain namespacing convention.
func ValidateClaimType(claimType string) error {
	if !claimTypeRE.MatchString(claimType) {
		return fmt.Errorf("%w: got %q", ErrInvalidClaimType, claimType)
	}
	return nil
}

// New builds and signs an attestation event: attester vouches for
// subjectPubKey with the given claim_type/claim_value. summary is a
// human-readable sentence stored in the event's content field, so
// generic (non-persona-aware) Nostr clients can still render
// something meaningful — D001's interop-scope resolution.
func New(attester identity.Persona, subjectPubKey, claimType, claimValue, summary string) (*nostr.Event, error) {
	if err := ValidateClaimType(claimType); err != nil {
		return nil, err
	}
	if !nostr.IsValidPublicKey(subjectPubKey) {
		return nil, errors.New("attestation: invalid subject public key")
	}

	evt := &nostr.Event{
		Kind:      Kind,
		CreatedAt: nostr.Now(),
		Content:   summary,
		Tags: nostr.Tags{
			{"p", subjectPubKey},
			{"claim_type", claimType},
			{"claim_value", claimValue},
		},
	}
	if err := evt.Sign(attester.PrivateKey); err != nil {
		return nil, err
	}
	return evt, nil
}

// Claim is an attestation event's fields pulled out into their own
// type, per D001's (attester_key, subject_key, claim_type,
// claim_value, timestamp, signature) primitive.
type Claim struct {
	AttesterKey string
	SubjectKey  string
	ClaimType   string
	ClaimValue  string
	Timestamp   nostr.Timestamp
	Summary     string
	Signature   string
}

// Verify checks an attestation event's signature and shape, and
// extracts it into a Claim. It does not evaluate trust — per D001,
// trust is computed relative to the verifier, not decided here.
func Verify(evt *nostr.Event) (Claim, error) {
	if evt.Kind != Kind {
		return Claim{}, fmt.Errorf("attestation: unexpected kind %d, want %d", evt.Kind, Kind)
	}
	ok, err := evt.CheckSignature()
	if err != nil {
		return Claim{}, fmt.Errorf("attestation: signature check failed: %w", err)
	}
	if !ok {
		return Claim{}, errors.New("attestation: invalid signature")
	}

	subjectKey := evt.Tags.Find("p").Value()
	claimType := evt.Tags.Find("claim_type").Value()
	claimValue := evt.Tags.Find("claim_value").Value()
	if subjectKey == "" || claimType == "" {
		return Claim{}, errors.New("attestation: missing required tag (p or claim_type)")
	}
	if err := ValidateClaimType(claimType); err != nil {
		return Claim{}, err
	}

	return Claim{
		AttesterKey: evt.PubKey,
		SubjectKey:  subjectKey,
		ClaimType:   claimType,
		ClaimValue:  claimValue,
		Timestamp:   evt.CreatedAt,
		Summary:     evt.Content,
		Signature:   evt.Sig,
	}, nil
}

// Publish signs nothing itself — it publishes an already-built event
// to every given relay URL, per D001's "public Nostr relay network,
// multi-relay for redundancy" resolution. It returns the first error
// encountered but still attempts every relay.
func Publish(ctx context.Context, evt *nostr.Event, relayURLs []string) error {
	if len(relayURLs) == 0 {
		return errors.New("attestation: no relays given")
	}
	var firstErr error
	for _, url := range relayURLs {
		if err := publishOne(ctx, evt, url); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func publishOne(ctx context.Context, evt *nostr.Event, url string) error {
	relay, err := nostr.RelayConnect(ctx, url)
	if err != nil {
		return fmt.Errorf("attestation: connect %s: %w", url, err)
	}
	defer relay.Close()
	if err := relay.Publish(ctx, *evt); err != nil {
		return fmt.Errorf("attestation: publish to %s: %w", url, err)
	}
	return nil
}

// FetchForSubject queries relayURL for attestation events about
// subjectPubKey and returns every one it finds, without evaluating
// trust — per D001, "cheap to verify, no central lookup required."
func FetchForSubject(ctx context.Context, relayURL, subjectPubKey string) ([]*nostr.Event, error) {
	relay, err := nostr.RelayConnect(ctx, relayURL)
	if err != nil {
		return nil, fmt.Errorf("attestation: connect %s: %w", relayURL, err)
	}
	defer relay.Close()

	events, err := relay.QuerySync(ctx, nostr.Filter{
		Kinds: []int{Kind},
		Tags:  nostr.TagMap{"p": []string{subjectPubKey}},
	})
	if err != nil {
		return nil, fmt.Errorf("attestation: query %s: %w", relayURL, err)
	}
	return events, nil
}
