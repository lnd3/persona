package dispute

import (
	"testing"

	"github.com/lnd3/persona/internal/attestation"
	"github.com/lnd3/persona/internal/identity"
)

func TestValidatePanelSize(t *testing.T) {
	valid := []int{1, 3, 5, 7}
	for _, n := range valid {
		if err := ValidatePanelSize(n); err != nil {
			t.Errorf("ValidatePanelSize(%d) = %v, want nil", n, err)
		}
	}
	invalid := []int{0, 2, 4, 6}
	for _, n := range invalid {
		if err := ValidatePanelSize(n); err == nil {
			t.Errorf("ValidatePanelSize(%d) = nil, want error", n)
		}
	}
}

func newIdentity(t *testing.T) identity.Persona {
	t.Helper()
	p, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	return p
}

func verifiedPanelClaim(t *testing.T, publisher identity.Persona, disputedAttesterPubkey, eventID string, arbiters []string) attestation.Claim {
	t.Helper()
	evt, err := NewPanelClaim(publisher, disputedAttesterPubkey, eventID, arbiters)
	if err != nil {
		t.Fatalf("NewPanelClaim: %v", err)
	}
	claim, err := attestation.Verify(evt)
	if err != nil {
		t.Fatalf("attestation.Verify: %v", err)
	}
	return claim
}

func TestConfirmedPanelBothSidesAgree(t *testing.T) {
	attester := newIdentity(t)
	challenger := newIdentity(t)
	a1, a2, a3 := newIdentity(t), newIdentity(t), newIdentity(t)
	arbiters := []string{a1.PublicKey, a2.PublicKey, a3.PublicKey}
	eventID := fakeEventID(t)

	claims := []attestation.Claim{
		verifiedPanelClaim(t, attester, attester.PublicKey, eventID, arbiters),
		verifiedPanelClaim(t, challenger, attester.PublicKey, eventID, arbiters),
	}

	got, confirmed, err := ConfirmedPanel(claims, eventID, attester.PublicKey, challenger.PublicKey)
	if err != nil {
		t.Fatalf("ConfirmedPanel: %v", err)
	}
	if !confirmed {
		t.Fatal("expected panel to be confirmed when both sides agree")
	}
	if !sameSet(got, arbiters) {
		t.Errorf("confirmed arbiters = %v, want %v", got, arbiters)
	}
}

func TestConfirmedPanelMismatchedSidesNotConfirmed(t *testing.T) {
	attester := newIdentity(t)
	challenger := newIdentity(t)
	a1, a2, a3 := newIdentity(t), newIdentity(t), newIdentity(t)
	b1, b2, b3 := newIdentity(t), newIdentity(t), newIdentity(t)
	eventID := fakeEventID(t)

	claims := []attestation.Claim{
		verifiedPanelClaim(t, attester, attester.PublicKey, eventID, []string{a1.PublicKey, a2.PublicKey, a3.PublicKey}),
		verifiedPanelClaim(t, challenger, attester.PublicKey, eventID, []string{b1.PublicKey, b2.PublicKey, b3.PublicKey}),
	}

	_, confirmed, err := ConfirmedPanel(claims, eventID, attester.PublicKey, challenger.PublicKey)
	if err != nil {
		t.Fatalf("ConfirmedPanel: %v", err)
	}
	if confirmed {
		t.Fatal("expected panel not to be confirmed when sides disagree")
	}
}

func TestConfirmedPanelOneSidedNotConfirmed(t *testing.T) {
	attester := newIdentity(t)
	challenger := newIdentity(t)
	a1, a2, a3 := newIdentity(t), newIdentity(t), newIdentity(t)
	eventID := fakeEventID(t)

	claims := []attestation.Claim{
		verifiedPanelClaim(t, attester, attester.PublicKey, eventID, []string{a1.PublicKey, a2.PublicKey, a3.PublicKey}),
	}

	_, confirmed, err := ConfirmedPanel(claims, eventID, attester.PublicKey, challenger.PublicKey)
	if err != nil {
		t.Fatalf("ConfirmedPanel: %v", err)
	}
	if confirmed {
		t.Fatal("expected panel not to be confirmed when only one side has published")
	}
}
