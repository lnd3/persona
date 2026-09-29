package payment

import (
	"testing"

	"github.com/lnd3/persona/internal/identity"
)

func TestContentHashDeterministicAndOrderSensitive(t *testing.T) {
	base := ContentHash("attester", "subject", "net.persona.core.skill", "go", 1000)
	same := ContentHash("attester", "subject", "net.persona.core.skill", "go", 1000)
	if base != same {
		t.Error("ContentHash is not deterministic for identical inputs")
	}

	variants := []struct {
		name string
		hash [32]byte
	}{
		{"attester", ContentHash("other-attester", "subject", "net.persona.core.skill", "go", 1000)},
		{"subject", ContentHash("attester", "other-subject", "net.persona.core.skill", "go", 1000)},
		{"claim_type", ContentHash("attester", "subject", "net.persona.core.other", "go", 1000)},
		{"claim_value", ContentHash("attester", "subject", "net.persona.core.skill", "rust", 1000)},
		{"timestamp", ContentHash("attester", "subject", "net.persona.core.skill", "go", 1001)},
	}
	for _, v := range variants {
		if v.hash == base {
			t.Errorf("changing %s did not change ContentHash", v.name)
		}
	}
}

func TestSignVerifyReceiptRoundTrip(t *testing.T) {
	gateway, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	hash := ContentHash("attester", "subject", "net.persona.core.skill", "go", 1000)

	receipt, err := SignReceipt(gateway, hash)
	if err != nil {
		t.Fatalf("SignReceipt: %v", err)
	}
	if err := VerifyReceipt(receipt, hash); err != nil {
		t.Fatalf("VerifyReceipt: %v", err)
	}
}

func TestVerifyReceiptRejectsTamperedSignature(t *testing.T) {
	gateway, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	hash := ContentHash("attester", "subject", "net.persona.core.skill", "go", 1000)
	receipt, err := SignReceipt(gateway, hash)
	if err != nil {
		t.Fatalf("SignReceipt: %v", err)
	}

	tampered := receipt
	tampered.Signature = tampered.Signature[:len(tampered.Signature)-2] + "00"
	if err := VerifyReceipt(tampered, hash); err == nil {
		t.Fatal("expected error for tampered signature")
	}
}

func TestVerifyReceiptRejectsMismatchedHash(t *testing.T) {
	gateway, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	hash := ContentHash("attester", "subject", "net.persona.core.skill", "go", 1000)
	receipt, err := SignReceipt(gateway, hash)
	if err != nil {
		t.Fatalf("SignReceipt: %v", err)
	}

	otherHash := ContentHash("attester", "subject", "net.persona.core.skill", "rust", 1000)
	if err := VerifyReceipt(receipt, otherHash); err == nil {
		t.Fatal("expected error for mismatched content hash")
	}
}

func TestEncodeDecodeReceiptRoundTrip(t *testing.T) {
	gateway, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	hash := ContentHash("attester", "subject", "net.persona.core.skill", "go", 1000)
	receipt, err := SignReceipt(gateway, hash)
	if err != nil {
		t.Fatalf("SignReceipt: %v", err)
	}

	encoded, err := EncodeReceipt(receipt)
	if err != nil {
		t.Fatalf("EncodeReceipt: %v", err)
	}
	decoded, err := DecodeReceipt(encoded)
	if err != nil {
		t.Fatalf("DecodeReceipt: %v", err)
	}
	if decoded != receipt {
		t.Errorf("decoded receipt = %+v, want %+v", decoded, receipt)
	}
}
