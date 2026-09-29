package payment

import (
	"testing"

	"github.com/lnd3/persona/internal/attestation"
	"github.com/lnd3/persona/internal/identity"
)

func TestFinalizeWithReceiptRoundTrip(t *testing.T) {
	attester, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	subject, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	gateway, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}

	pending, err := NewPendingClaim(attester.PublicKey, subject.PublicKey, "net.persona.core.skill", "go")
	if err != nil {
		t.Fatalf("NewPendingClaim: %v", err)
	}
	hash := pending.Hash()
	receipt, err := SignReceipt(gateway, hash)
	if err != nil {
		t.Fatalf("SignReceipt: %v", err)
	}
	if err := VerifyReceipt(receipt, hash); err != nil {
		t.Fatalf("VerifyReceipt: %v", err)
	}

	evt, err := pending.Finalize(attester, "go skill, paid", &receipt)
	if err != nil {
		t.Fatalf("Finalize: %v", err)
	}

	claim, err := attestation.Verify(evt)
	if err != nil {
		t.Fatalf("attestation.Verify: %v", err)
	}
	if claim.Timestamp != pending.Timestamp {
		t.Errorf("event timestamp = %d, want %d (must match the hash's timestamp)", claim.Timestamp, pending.Timestamp)
	}

	extracted, ok, err := ExtractReceipt(evt)
	if err != nil {
		t.Fatalf("ExtractReceipt: %v", err)
	}
	if !ok {
		t.Fatal("expected a receipt to be present")
	}
	if extracted != receipt {
		t.Errorf("extracted receipt = %+v, want %+v", extracted, receipt)
	}

	// The verifier's own recomputed hash (from the fetched claim) must
	// match what the receipt was actually signed over.
	if err := VerifyReceipt(extracted, ClaimContentHash(claim)); err != nil {
		t.Fatalf("VerifyReceipt against recomputed claim hash: %v", err)
	}
}

func TestFinalizeWithoutReceiptStillVerifies(t *testing.T) {
	attester, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	subject, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}

	pending, err := NewPendingClaim(attester.PublicKey, subject.PublicKey, "net.persona.core.skill", "go")
	if err != nil {
		t.Fatalf("NewPendingClaim: %v", err)
	}
	evt, err := pending.Finalize(attester, "go skill, unpaid", nil)
	if err != nil {
		t.Fatalf("Finalize: %v", err)
	}

	if _, err := attestation.Verify(evt); err != nil {
		t.Fatalf("attestation.Verify of a receipt-less claim should succeed: %v", err)
	}

	_, ok, err := ExtractReceipt(evt)
	if err != nil {
		t.Fatalf("ExtractReceipt: %v", err)
	}
	if ok {
		t.Error("expected no receipt to be present")
	}
}

func TestNewPendingClaimRejectsInvalidClaimType(t *testing.T) {
	attester, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	subject, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	if _, err := NewPendingClaim(attester.PublicKey, subject.PublicKey, "notnamespaced", "go"); err == nil {
		t.Fatal("expected error for non-namespaced claim_type")
	}
}
