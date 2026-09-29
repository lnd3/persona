// Package payment implements D001's Payment integration section and
// A004's concrete choices: an attestation-cost gateway that, once a
// request has already cleared Aperture's L402 check, mints a small
// signed Receipt binding to a specific attestation claim's content —
// never to Lightning payment details, which stay entirely Aperture's
// concern, per cinder's own D005 reference architecture.
package payment

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/nbd-wtf/go-nostr"

	"github.com/lnd3/persona/internal/attestation"
	"github.com/lnd3/persona/internal/identity"
)

// Receipt is a gateway operator's signed proof that a specific
// attestation claim's content was published behind an already-paid
// L402 request. It is never published as a standalone Nostr event —
// only embedded inside an attestation event's own "payment_receipt"
// tag (see event.go) — so it's a plain struct, Schnorr-signed with
// the same primitive Nostr events use (btcec/schnorr, already a
// go-nostr dependency), not a new signature scheme.
type Receipt struct {
	ContentHash   string          `json:"content_hash"` // hex sha256, see ContentHash
	GatewayPubkey string          `json:"gateway_pubkey"`
	IssuedAt      nostr.Timestamp `json:"issued_at"`
	Signature     string          `json:"signature"` // hex Schnorr signature over ContentHash's raw bytes
}

// ErrReceiptMismatch is returned by VerifyReceipt when a receipt's
// content_hash doesn't match the hash the caller expected.
var ErrReceiptMismatch = errors.New("payment: receipt content_hash does not match expected claim content")

// ContentHash computes the deterministic hash a receipt binds to:
// sha256 over the claim's immutable content fields, in a fixed order,
// each length-prefixed so no ambiguity is introduced by field values
// that might otherwise run together (e.g. an empty claim_value next
// to a claim_type that could be mistaken for part of it). Computable
// identically by an attester (before publishing, from the same field
// values `attestation.New` will use) and by a verifier (after
// fetching, from an already-verified attestation.Claim) — this is
// what lets a receipt exist without referencing the final event's own
// `id`, which would be circular (see A004's Decisions).
func ContentHash(attesterPubkey, subjectPubkey, claimType, claimValue string, timestamp nostr.Timestamp) [32]byte {
	h := sha256.New()
	writeField(h, attesterPubkey)
	writeField(h, subjectPubkey)
	writeField(h, claimType)
	writeField(h, claimValue)
	writeField(h, fmt.Sprintf("%d", timestamp))
	var sum [32]byte
	copy(sum[:], h.Sum(nil))
	return sum
}

// ClaimContentHash is ContentHash applied to an already-verified
// attestation.Claim, for the verifier side of the round trip.
func ClaimContentHash(claim attestation.Claim) [32]byte {
	return ContentHash(claim.AttesterKey, claim.SubjectKey, claim.ClaimType, claim.ClaimValue, claim.Timestamp)
}

func writeField(h interface{ Write([]byte) (int, error) }, s string) {
	h.Write([]byte(fmt.Sprintf("%d:%s|", len(s), s)))
}

// SignReceipt mints a Receipt over hash, signed with the gateway
// operator's own key. The gateway trusts that whatever called this
// already cleared Aperture's L402 check (see server.go) — no payment
// logic lives here at all.
func SignReceipt(gateway identity.Persona, hash [32]byte) (Receipt, error) {
	skBytes, err := hex.DecodeString(gateway.PrivateKey)
	if err != nil {
		return Receipt{}, fmt.Errorf("payment: invalid gateway private key: %w", err)
	}
	sk, _ := btcec.PrivKeyFromBytes(skBytes)
	sig, err := schnorr.Sign(sk, hash[:], schnorr.FastSign())
	if err != nil {
		return Receipt{}, fmt.Errorf("payment: signing receipt: %w", err)
	}
	return Receipt{
		ContentHash:   hex.EncodeToString(hash[:]),
		GatewayPubkey: gateway.PublicKey,
		IssuedAt:      nostr.Now(),
		Signature:     hex.EncodeToString(sig.Serialize()),
	}, nil
}

// VerifyReceipt checks that receipt is a genuine, unmodified receipt
// for expectedHash: a valid Schnorr signature by receipt.GatewayPubkey
// over receipt.ContentHash, and receipt.ContentHash matching
// expectedHash. It does not decide whether the caller trusts
// receipt.GatewayPubkey — that's a per-verifier policy decision, per
// D001's relative-trust model, not this function's job.
func VerifyReceipt(receipt Receipt, expectedHash [32]byte) error {
	if receipt.ContentHash != hex.EncodeToString(expectedHash[:]) {
		return ErrReceiptMismatch
	}
	pkBytes, err := hex.DecodeString(receipt.GatewayPubkey)
	if err != nil {
		return fmt.Errorf("payment: invalid gateway pubkey: %w", err)
	}
	pubkey, err := schnorr.ParsePubKey(pkBytes)
	if err != nil {
		return fmt.Errorf("payment: parsing gateway pubkey: %w", err)
	}
	sigBytes, err := hex.DecodeString(receipt.Signature)
	if err != nil {
		return fmt.Errorf("payment: invalid signature encoding: %w", err)
	}
	sig, err := schnorr.ParseSignature(sigBytes)
	if err != nil {
		return fmt.Errorf("payment: parsing signature: %w", err)
	}
	if !sig.Verify(expectedHash[:], pubkey) {
		return errors.New("payment: invalid receipt signature")
	}
	return nil
}

// EncodeReceipt/DecodeReceipt implement the "payment_receipt" tag's
// opaque base64-JSON encoding (see event.go).
func EncodeReceipt(r Receipt) (string, error) {
	b, err := json.Marshal(r)
	if err != nil {
		return "", fmt.Errorf("payment: marshal receipt: %w", err)
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

func DecodeReceipt(s string) (Receipt, error) {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return Receipt{}, fmt.Errorf("payment: invalid receipt encoding: %w", err)
	}
	var r Receipt
	if err := json.Unmarshal(b, &r); err != nil {
		return Receipt{}, fmt.Errorf("payment: unmarshal receipt: %w", err)
	}
	return r, nil
}
