package recovery

import (
	"testing"

	"github.com/lnd3/persona/internal/attestation"
	"github.com/lnd3/persona/internal/identity"
)

// buildGuardianSet designates n guardians (threshold m) for owner,
// all under one groupID, returning their verified recovery_guardian
// claims and the guardian personas themselves.
func buildGuardianSet(t *testing.T, owner identity.Persona, m, n int, groupID string) ([]attestation.Claim, []identity.Persona) {
	t.Helper()
	var claims []attestation.Claim
	var guardians []identity.Persona
	for i := 0; i < n; i++ {
		g, err := identity.New()
		if err != nil {
			t.Fatalf("identity.New: %v", err)
		}
		evt, err := NewGuardianClaim(owner, g.PublicKey, m, n, groupID)
		if err != nil {
			t.Fatalf("NewGuardianClaim: %v", err)
		}
		claim, err := attestation.Verify(evt)
		if err != nil {
			t.Fatalf("attestation.Verify: %v", err)
		}
		claims = append(claims, claim)
		guardians = append(guardians, g)
	}
	return claims, guardians
}

func confirmFrom(t *testing.T, guardian identity.Persona, ownerPubKey, groupID, nonce string) attestation.Claim {
	t.Helper()
	evt, err := NewConfirmClaim(guardian, ownerPubKey, groupID, nonce)
	if err != nil {
		t.Fatalf("NewConfirmClaim: %v", err)
	}
	claim, err := attestation.Verify(evt)
	if err != nil {
		t.Fatalf("attestation.Verify: %v", err)
	}
	return claim
}

func TestThresholdMetAtExactThreshold(t *testing.T) {
	owner, err := identity.New()
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
	guardianClaims, guardians := buildGuardianSet(t, owner, 3, 5, groupID)

	var confirms []attestation.Claim
	for _, g := range guardians[:3] {
		confirms = append(confirms, confirmFrom(t, g, owner.PublicKey, groupID, nonce))
	}

	met, count, threshold, err := ThresholdMet(guardianClaims, confirms, groupID, nonce)
	if err != nil {
		t.Fatalf("ThresholdMet: %v", err)
	}
	if !met || count != 3 || threshold != 3 {
		t.Errorf("met=%v count=%d threshold=%d, want met=true count=3 threshold=3", met, count, threshold)
	}
}

func TestThresholdNotMetBelowThreshold(t *testing.T) {
	owner, err := identity.New()
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
	guardianClaims, guardians := buildGuardianSet(t, owner, 3, 5, groupID)

	var confirms []attestation.Claim
	for _, g := range guardians[:2] {
		confirms = append(confirms, confirmFrom(t, g, owner.PublicKey, groupID, nonce))
	}

	met, count, _, err := ThresholdMet(guardianClaims, confirms, groupID, nonce)
	if err != nil {
		t.Fatalf("ThresholdMet: %v", err)
	}
	if met || count != 2 {
		t.Errorf("met=%v count=%d, want met=false count=2", met, count)
	}
}

func TestThresholdMetIgnoresDuplicateConfirmFromSameGuardian(t *testing.T) {
	owner, err := identity.New()
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
	guardianClaims, guardians := buildGuardianSet(t, owner, 3, 5, groupID)

	confirms := []attestation.Claim{
		confirmFrom(t, guardians[0], owner.PublicKey, groupID, nonce),
		confirmFrom(t, guardians[0], owner.PublicKey, groupID, nonce), // same guardian again
		confirmFrom(t, guardians[1], owner.PublicKey, groupID, nonce),
	}

	met, count, _, err := ThresholdMet(guardianClaims, confirms, groupID, nonce)
	if err != nil {
		t.Fatalf("ThresholdMet: %v", err)
	}
	if met || count != 2 {
		t.Errorf("met=%v count=%d, want met=false count=2 (duplicate shouldn't count twice)", met, count)
	}
}

func TestThresholdMetIgnoresConfirmFromNonGuardian(t *testing.T) {
	owner, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	stranger, err := identity.New()
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
	guardianClaims, guardians := buildGuardianSet(t, owner, 2, 3, groupID)

	confirms := []attestation.Claim{
		confirmFrom(t, guardians[0], owner.PublicKey, groupID, nonce),
		confirmFrom(t, stranger, owner.PublicKey, groupID, nonce), // never designated a guardian
	}

	met, count, _, err := ThresholdMet(guardianClaims, confirms, groupID, nonce)
	if err != nil {
		t.Fatalf("ThresholdMet: %v", err)
	}
	if met || count != 1 {
		t.Errorf("met=%v count=%d, want met=false count=1 (stranger's confirm shouldn't count)", met, count)
	}
}

func TestThresholdMetIgnoresConfirmForDifferentAttempt(t *testing.T) {
	owner, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	groupID, err := NewGroupID()
	if err != nil {
		t.Fatalf("NewGroupID: %v", err)
	}
	nonceA, err := NewNonce()
	if err != nil {
		t.Fatalf("NewNonce: %v", err)
	}
	nonceB, err := NewNonce()
	if err != nil {
		t.Fatalf("NewNonce: %v", err)
	}
	guardianClaims, guardians := buildGuardianSet(t, owner, 2, 3, groupID)

	confirms := []attestation.Claim{
		confirmFrom(t, guardians[0], owner.PublicKey, groupID, nonceA),
		confirmFrom(t, guardians[1], owner.PublicKey, groupID, nonceB), // different attempt
	}

	met, count, _, err := ThresholdMet(guardianClaims, confirms, groupID, nonceA)
	if err != nil {
		t.Fatalf("ThresholdMet: %v", err)
	}
	if met || count != 1 {
		t.Errorf("met=%v count=%d, want met=false count=1 (different nonce shouldn't count)", met, count)
	}
}
