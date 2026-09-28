// Package identity implements D001's identity layer: a persona is a
// secp256k1 keypair using Nostr's own key format, not a custom scheme.
package identity

import (
	"errors"

	"github.com/nbd-wtf/go-nostr/nip19"

	"github.com/nbd-wtf/go-nostr"
)

// Persona is a keypair identifying one persona, per D001's Identity
// layer: a secp256k1 keypair using Nostr's own key format.
type Persona struct {
	PrivateKey string // hex-encoded secp256k1 private key
	PublicKey  string // hex-encoded x-only public key (Nostr's own format)
}

// New generates a fresh persona keypair.
func New() (Persona, error) {
	sk := nostr.GeneratePrivateKey()
	if sk == "" {
		return Persona{}, errors.New("identity: failed to generate private key")
	}
	pk, err := nostr.GetPublicKey(sk)
	if err != nil {
		return Persona{}, err
	}
	return Persona{PrivateKey: sk, PublicKey: pk}, nil
}

// FromPrivateKey derives a Persona from an existing hex-encoded private key.
func FromPrivateKey(sk string) (Persona, error) {
	pk, err := nostr.GetPublicKey(sk)
	if err != nil {
		return Persona{}, err
	}
	return Persona{PrivateKey: sk, PublicKey: pk}, nil
}

// Npub returns the bech32 (NIP-19) encoding of the public key, for
// human-facing display/copy-paste.
func (p Persona) Npub() (string, error) {
	return nip19.EncodePublicKey(p.PublicKey)
}

// Nsec returns the bech32 (NIP-19) encoding of the private key. Callers
// must treat this the same as the raw private key — it is not a
// separate secret, just a different encoding of the same one.
func (p Persona) Nsec() (string, error) {
	return nip19.EncodePrivateKey(p.PrivateKey)
}

// DecodeNpub decodes a bech32 npub back into a hex public key.
func DecodeNpub(npub string) (string, error) {
	prefix, value, err := nip19.Decode(npub)
	if err != nil {
		return "", err
	}
	if prefix != "npub" {
		return "", errors.New("identity: not an npub")
	}
	pk, ok := value.(string)
	if !ok {
		return "", errors.New("identity: malformed npub payload")
	}
	return pk, nil
}
