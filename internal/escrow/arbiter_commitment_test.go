package escrow

import (
	"testing"

	"github.com/lnd3/persona/internal/attestation"
	"github.com/lnd3/persona/internal/identity"
)

func TestArbiterCommitmentClaimRoundTrip(t *testing.T) {
	owner, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	_, arbiterPub := newTestKey(t)
	fundingEventID := owner.PublicKey // any 64-hex-char stand-in works

	evt, err := NewArbiterCommitmentClaim(owner, fundingEventID, arbiterPub)
	if err != nil {
		t.Fatalf("NewArbiterCommitmentClaim: %v", err)
	}
	claim, err := attestation.Verify(evt)
	if err != nil {
		t.Fatalf("attestation.Verify: %v", err)
	}
	if claim.AttesterKey != owner.PublicKey || claim.SubjectKey != owner.PublicKey {
		t.Errorf("attester/subject = %s/%s, want both %s (self-referential)", claim.AttesterKey, claim.SubjectKey, owner.PublicKey)
	}

	commitment, err := ParseArbiterCommitmentClaim(claim)
	if err != nil {
		t.Fatalf("ParseArbiterCommitmentClaim: %v", err)
	}
	if commitment.FundingEventID != fundingEventID {
		t.Errorf("FundingEventID = %s, want %s", commitment.FundingEventID, fundingEventID)
	}
	if string(commitment.ArbiterPubKey) != string(arbiterPub) {
		t.Errorf("ArbiterPubKey = %x, want %x", commitment.ArbiterPubKey, arbiterPub)
	}
}

func TestNewArbiterCommitmentClaimRejectsInvalidPubKey(t *testing.T) {
	owner, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	if _, err := NewArbiterCommitmentClaim(owner, owner.PublicKey, []byte{0x01}); err == nil {
		t.Fatal("expected error for a malformed arbiter pubkey")
	}
}

func TestParseArbiterCommitmentClaimRejectsMalformedValue(t *testing.T) {
	claim := attestation.Claim{ClaimType: ArbiterCommitmentClaimType, ClaimValue: "garbage"}
	if _, err := ParseArbiterCommitmentClaim(claim); err == nil {
		t.Fatal("expected error for malformed claim_value")
	}
}

func TestParseArbiterCommitmentClaimRejectsWrongType(t *testing.T) {
	claim := attestation.Claim{ClaimType: "net.persona.core.bond", ClaimValue: "x"}
	if _, err := ParseArbiterCommitmentClaim(claim); err == nil {
		t.Fatal("expected error for wrong claim_type")
	}
}
