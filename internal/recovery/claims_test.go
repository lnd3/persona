package recovery

import (
	"testing"

	"github.com/lnd3/persona/internal/attestation"
	"github.com/lnd3/persona/internal/identity"
)

func TestGuardianClaimRoundTrip(t *testing.T) {
	owner, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	guardian, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	groupID, err := NewGroupID()
	if err != nil {
		t.Fatalf("NewGroupID: %v", err)
	}

	evt, err := NewGuardianClaim(owner, guardian.PublicKey, 3, 5, groupID)
	if err != nil {
		t.Fatalf("NewGuardianClaim: %v", err)
	}
	claim, err := attestation.Verify(evt)
	if err != nil {
		t.Fatalf("attestation.Verify: %v", err)
	}
	d, err := ParseGuardianClaim(claim)
	if err != nil {
		t.Fatalf("ParseGuardianClaim: %v", err)
	}
	if d.Threshold != 3 || d.Total != 5 || d.GroupID != groupID {
		t.Errorf("parsed = %+v, want threshold=3 total=5 group=%s", d, groupID)
	}
	if claim.SubjectKey != guardian.PublicKey {
		t.Errorf("subject = %s, want %s", claim.SubjectKey, guardian.PublicKey)
	}
}

func TestGuardianClaimRejectsInvalidThreshold(t *testing.T) {
	owner, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	guardian, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	if _, err := NewGuardianClaim(owner, guardian.PublicKey, 6, 5, "abc123"); err == nil {
		t.Fatal("expected error for threshold > total")
	}
}

func TestParseGuardianClaimRejectsMalformedValue(t *testing.T) {
	claim := attestation.Claim{ClaimType: GuardianClaimType, ClaimValue: "not-a-valid-value"}
	if _, err := ParseGuardianClaim(claim); err == nil {
		t.Fatal("expected error for malformed claim_value")
	}
}

func TestConfirmClaimRoundTrip(t *testing.T) {
	owner, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	guardian, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	groupID, err := NewGroupID()
	if err != nil {
		t.Fatalf("NewGroupID: %v", err)
	}
	nonce, err := NewNonce()
	if err != nil {
		t.Fatalf("NewNonce: %v", err)
	}

	evt, err := NewConfirmClaim(guardian, owner.PublicKey, groupID, nonce)
	if err != nil {
		t.Fatalf("NewConfirmClaim: %v", err)
	}
	claim, err := attestation.Verify(evt)
	if err != nil {
		t.Fatalf("attestation.Verify: %v", err)
	}
	rc, err := ParseConfirmClaim(claim)
	if err != nil {
		t.Fatalf("ParseConfirmClaim: %v", err)
	}
	if rc.GroupID != groupID || rc.Nonce != nonce {
		t.Errorf("parsed = %+v, want group=%s nonce=%s", rc, groupID, nonce)
	}
	if claim.AttesterKey != guardian.PublicKey || claim.SubjectKey != owner.PublicKey {
		t.Errorf("attester/subject = %s/%s, want %s/%s", claim.AttesterKey, claim.SubjectKey, guardian.PublicKey, owner.PublicKey)
	}
}

func TestParseConfirmClaimRejectsMalformedValue(t *testing.T) {
	claim := attestation.Claim{ClaimType: ConfirmClaimType, ClaimValue: "not-valid"}
	if _, err := ParseConfirmClaim(claim); err == nil {
		t.Fatal("expected error for malformed claim_value")
	}
}
