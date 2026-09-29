// Package recovery implements D001's Recovery resolution and A003's
// concrete choices: SSKR-style key splitting (a plain GF(256) Shamir
// scheme, see shamir.go) authorized via delegation attestation
// (recovery_guardian/recovery_confirm claim types, see claims.go and
// quorum.go). Guardian shares travel only as NIP-44-encrypted
// payloads, never as plaintext or as a public claim_value.
package recovery

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip44"

	"github.com/lnd3/persona/internal/identity"
)

// ShareEventKind is the Nostr kind for a guardian-share delivery
// event: regular-event range, adjacent to A001's own attestation kind
// (3300) in the same open gap that decision already identified,
// rather than reusing kind 4 (NIP-04) — this event's content is
// NIP-44-encrypted, not NIP-04, so a client expecting kind 4's
// encryption scheme would fail to decrypt it; a dedicated kind avoids
// that mismatch. Documented openly here per this project's established
// "coordination, not permission" posture toward kind numbers, same as
// A001 — not re-checked against the live NIPs registry at filing time,
// worth a final check before shipping, same caveat A001 carried.
const ShareEventKind = 3301

// ErrShareRecipientMismatch is returned when a share event's "p" tag
// doesn't match the guardian pubkey extraction was attempted with.
var ErrShareRecipientMismatch = errors.New("recovery: share event recipient does not match")

// EncryptShare encrypts share to guardianPubKey using NIP-44, so it
// can travel as a direct-message-style event payload. It builds only
// the ciphertext payload — delivering it as an actual Nostr DM event
// is the caller's concern, not this function's.
func EncryptShare(owner identity.Persona, guardianPubKey string, share Share) (string, error) {
	payload, err := json.Marshal(share)
	if err != nil {
		return "", fmt.Errorf("recovery: marshal share: %w", err)
	}
	key, err := nip44.GenerateConversationKey(guardianPubKey, owner.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("recovery: derive conversation key: %w", err)
	}
	ciphertext, err := nip44.Encrypt(string(payload), key)
	if err != nil {
		return "", fmt.Errorf("recovery: encrypt share: %w", err)
	}
	return ciphertext, nil
}

// DecryptShare decrypts a share previously encrypted by EncryptShare.
// The guardian must pass ownerPubKey (whose share this claims to be)
// and their own keypair — NIP-44's shared secret is symmetric, so
// decrypting with the wrong owner pubkey either fails outright or
// silently produces garbage, per NIP-44's own MAC check.
func DecryptShare(guardian identity.Persona, ownerPubKey, ciphertext string) (Share, error) {
	key, err := nip44.GenerateConversationKey(ownerPubKey, guardian.PrivateKey)
	if err != nil {
		return Share{}, fmt.Errorf("recovery: derive conversation key: %w", err)
	}
	plaintext, err := nip44.Decrypt(ciphertext, key)
	if err != nil {
		return Share{}, fmt.Errorf("recovery: decrypt share: %w", err)
	}
	var share Share
	if err := json.Unmarshal([]byte(plaintext), &share); err != nil {
		return Share{}, fmt.Errorf("recovery: unmarshal share: %w", err)
	}
	return share, nil
}

// BuildShareEvent signs and returns the actual Nostr event that
// delivers an encrypted share to a guardian: a "p" tag naming the
// recipient (so a relay/client can route it, same convention as
// A001's attestation "p" tag), and NIP-44-encrypted content. This
// only builds the event — publishing it, and any DM inbox/client to
// receive it, is out of this action's scope per its own Context
// section.
func BuildShareEvent(owner identity.Persona, guardianPubKey string, share Share) (*nostr.Event, error) {
	if !nostr.IsValidPublicKey(guardianPubKey) {
		return nil, errors.New("recovery: invalid guardian public key")
	}
	ciphertext, err := EncryptShare(owner, guardianPubKey, share)
	if err != nil {
		return nil, err
	}
	evt := &nostr.Event{
		Kind:      ShareEventKind,
		CreatedAt: nostr.Now(),
		Content:   ciphertext,
		Tags:      nostr.Tags{{"p", guardianPubKey}},
	}
	if err := evt.Sign(owner.PrivateKey); err != nil {
		return nil, err
	}
	return evt, nil
}

// ExtractShare verifies a share-delivery event's signature and
// decrypts its content, given the guardian's own keypair. It rejects
// an event not addressed to this guardian via the "p" tag.
func ExtractShare(evt *nostr.Event, guardian identity.Persona) (Share, error) {
	if evt.Kind != ShareEventKind {
		return Share{}, fmt.Errorf("recovery: unexpected kind %d, want %d", evt.Kind, ShareEventKind)
	}
	ok, err := evt.CheckSignature()
	if err != nil {
		return Share{}, fmt.Errorf("recovery: signature check failed: %w", err)
	}
	if !ok {
		return Share{}, errors.New("recovery: invalid signature")
	}
	recipient := evt.Tags.Find("p").Value()
	if recipient != guardian.PublicKey {
		return Share{}, ErrShareRecipientMismatch
	}
	return DecryptShare(guardian, evt.PubKey, evt.Content)
}
