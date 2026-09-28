package identity

import "testing"

func TestNewAndEncode(t *testing.T) {
	p, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if len(p.PrivateKey) != 64 || len(p.PublicKey) != 64 {
		t.Fatalf("unexpected key lengths: sk=%d pk=%d", len(p.PrivateKey), len(p.PublicKey))
	}

	npub, err := p.Npub()
	if err != nil {
		t.Fatalf("Npub: %v", err)
	}
	if npub[:5] != "npub1" {
		t.Fatalf("npub has wrong prefix: %s", npub)
	}

	decoded, err := DecodeNpub(npub)
	if err != nil {
		t.Fatalf("DecodeNpub: %v", err)
	}
	if decoded != p.PublicKey {
		t.Fatalf("round trip mismatch: got %s, want %s", decoded, p.PublicKey)
	}
}

func TestFromPrivateKey(t *testing.T) {
	p, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	p2, err := FromPrivateKey(p.PrivateKey)
	if err != nil {
		t.Fatalf("FromPrivateKey: %v", err)
	}
	if p2.PublicKey != p.PublicKey {
		t.Fatalf("derived public key mismatch")
	}
}
