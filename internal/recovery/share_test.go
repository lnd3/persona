package recovery

import (
	"testing"

	"github.com/lnd3/persona/internal/identity"
)

func TestEncryptDecryptShareRoundTrip(t *testing.T) {
	owner, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	guardian, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}

	secret := randomSecret(t, 32)
	shares, err := Split(secret, 3, 5)
	if err != nil {
		t.Fatalf("Split: %v", err)
	}

	ciphertext, err := EncryptShare(owner, guardian.PublicKey, shares[0])
	if err != nil {
		t.Fatalf("EncryptShare: %v", err)
	}

	got, err := DecryptShare(guardian, owner.PublicKey, ciphertext)
	if err != nil {
		t.Fatalf("DecryptShare: %v", err)
	}
	if got.X != shares[0].X || got.Threshold != shares[0].Threshold || got.Total != shares[0].Total {
		t.Errorf("decrypted share metadata = %+v, want %+v", got, shares[0])
	}
	if string(got.Y) != string(shares[0].Y) {
		t.Errorf("decrypted share Y = %x, want %x", got.Y, shares[0].Y)
	}
}

func TestDecryptShareWrongRecipientFails(t *testing.T) {
	owner, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	guardian, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	stranger, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}

	shares, err := Split(randomSecret(t, 32), 3, 5)
	if err != nil {
		t.Fatalf("Split: %v", err)
	}
	ciphertext, err := EncryptShare(owner, guardian.PublicKey, shares[0])
	if err != nil {
		t.Fatalf("EncryptShare: %v", err)
	}

	if _, err := DecryptShare(stranger, owner.PublicKey, ciphertext); err == nil {
		t.Fatal("expected error decrypting a share meant for a different guardian")
	}
}

func TestBuildAndExtractShareEventRoundTrip(t *testing.T) {
	owner, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	guardian, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	shares, err := Split(randomSecret(t, 32), 3, 5)
	if err != nil {
		t.Fatalf("Split: %v", err)
	}

	evt, err := BuildShareEvent(owner, guardian.PublicKey, shares[0])
	if err != nil {
		t.Fatalf("BuildShareEvent: %v", err)
	}
	if evt.Kind != ShareEventKind {
		t.Errorf("event kind = %d, want %d", evt.Kind, ShareEventKind)
	}

	got, err := ExtractShare(evt, guardian)
	if err != nil {
		t.Fatalf("ExtractShare: %v", err)
	}
	if got.X != shares[0].X || string(got.Y) != string(shares[0].Y) {
		t.Errorf("extracted share = %+v, want %+v", got, shares[0])
	}
}

func TestExtractShareRejectsWrongRecipient(t *testing.T) {
	owner, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	guardian, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	stranger, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	shares, err := Split(randomSecret(t, 32), 3, 5)
	if err != nil {
		t.Fatalf("Split: %v", err)
	}

	evt, err := BuildShareEvent(owner, guardian.PublicKey, shares[0])
	if err != nil {
		t.Fatalf("BuildShareEvent: %v", err)
	}
	if _, err := ExtractShare(evt, stranger); err == nil {
		t.Fatal("expected error extracting a share event addressed to a different guardian")
	}
}
