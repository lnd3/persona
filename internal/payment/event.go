package payment

import (
	"errors"
	"fmt"

	"github.com/nbd-wtf/go-nostr"

	"github.com/lnd3/persona/internal/attestation"
	"github.com/lnd3/persona/internal/identity"
)

// receiptTagKey is the attestation event's opaque, single-tag receipt
// encoding, per A004's Decisions: one tag, base64-JSON, so a generic
// Nostr client can pass it through unmodified without understanding
// its shape.
const receiptTagKey = "payment_receipt"

// PendingClaim captures an attestation claim's content fields and a
// fixed timestamp *before* signing, so the same content hash a
// gateway signs a Receipt against is exactly what the final signed
// event carries — the timestamp can't be re-rolled at signing time
// without invalidating the receipt (see ContentHash's doc comment on
// why the hash, not the event id, is what binds them together).
type PendingClaim struct {
	AttesterPubkey string
	SubjectPubkey  string
	ClaimType      string
	ClaimValue     string
	Timestamp      nostr.Timestamp
}

// NewPendingClaim fixes a claim's content and timestamp, ready to be
// hashed (for a payment request) and later finalized into a signed
// event, with or without a receipt.
func NewPendingClaim(attesterPubkey, subjectPubkey, claimType, claimValue string) (PendingClaim, error) {
	if err := attestation.ValidateClaimType(claimType); err != nil {
		return PendingClaim{}, err
	}
	if !nostr.IsValidPublicKey(subjectPubkey) {
		return PendingClaim{}, errors.New("payment: invalid subject public key")
	}
	return PendingClaim{
		AttesterPubkey: attesterPubkey,
		SubjectPubkey:  subjectPubkey,
		ClaimType:      claimType,
		ClaimValue:     claimValue,
		Timestamp:      nostr.Now(),
	}, nil
}

// Hash computes this pending claim's ContentHash — what a gateway's
// Receipt actually signs over.
func (p PendingClaim) Hash() [32]byte {
	return ContentHash(p.AttesterPubkey, p.SubjectPubkey, p.ClaimType, p.ClaimValue, p.Timestamp)
}

// Finalize signs and returns the attestation event for this pending
// claim, at the exact fixed timestamp Hash() was computed against.
// receipt is optional (nil for no payment receipt attached — an
// attestation event remains structurally valid either way, per A004's
// "opt-in, not required by Verify" decision); when given, it must
// already have been verified by the caller (VerifyReceipt) against
// p.Hash() — Finalize does not re-verify it, only embeds it.
func (p PendingClaim) Finalize(attester identity.Persona, summary string, receipt *Receipt) (*nostr.Event, error) {
	if attester.PublicKey != p.AttesterPubkey {
		return nil, errors.New("payment: attester does not match this pending claim's attester_pubkey")
	}
	tags := nostr.Tags{
		{"p", p.SubjectPubkey},
		{"claim_type", p.ClaimType},
		{"claim_value", p.ClaimValue},
	}
	if receipt != nil {
		encoded, err := EncodeReceipt(*receipt)
		if err != nil {
			return nil, err
		}
		tags = append(tags, nostr.Tag{receiptTagKey, encoded})
	}
	evt := &nostr.Event{
		Kind:      attestation.Kind,
		CreatedAt: p.Timestamp,
		Content:   summary,
		Tags:      tags,
	}
	if err := evt.Sign(attester.PrivateKey); err != nil {
		return nil, err
	}
	return evt, nil
}

// ExtractReceipt reads an already-verified attestation event's
// payment_receipt tag, if present. ok is false (with a nil error)
// when the tag is simply absent — an attestation with no receipt is
// not an error, per A004's opt-in decision.
func ExtractReceipt(evt *nostr.Event) (receipt Receipt, ok bool, err error) {
	encoded := evt.Tags.Find(receiptTagKey).Value()
	if encoded == "" {
		return Receipt{}, false, nil
	}
	r, err := DecodeReceipt(encoded)
	if err != nil {
		return Receipt{}, false, fmt.Errorf("payment: extracting receipt: %w", err)
	}
	return r, true, nil
}
