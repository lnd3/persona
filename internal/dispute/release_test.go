package dispute

import (
	"testing"
	"time"

	"github.com/lnd3/persona/internal/attestation"
	"github.com/lnd3/persona/internal/identity"
)

func verifiedBondClaim(t *testing.T, owner identity.Persona, eventID, escrowRef string, amountSats int64, windowDays int) attestation.Claim {
	t.Helper()
	evt, err := NewBondClaim(owner, eventID, escrowRef, amountSats, windowDays)
	if err != nil {
		t.Fatalf("NewBondClaim: %v", err)
	}
	claim, err := attestation.Verify(evt)
	if err != nil {
		t.Fatalf("attestation.Verify: %v", err)
	}
	return claim
}

func verifiedChallengeClaim(t *testing.T, challenger identity.Persona, disputedAttesterPubkey, eventID, escrowRef string, amountSats int64) attestation.Claim {
	t.Helper()
	evt, err := NewChallengeClaim(challenger, disputedAttesterPubkey, eventID, escrowRef, amountSats)
	if err != nil {
		t.Fatalf("NewChallengeClaim: %v", err)
	}
	claim, err := attestation.Verify(evt)
	if err != nil {
		t.Fatalf("attestation.Verify: %v", err)
	}
	return claim
}

func TestIsSelfReleasedTrueAfterWindowNoChallenge(t *testing.T) {
	owner := newIdentity(t)
	eventID := fakeEventID(t)
	bond := verifiedBondClaim(t, owner, eventID, "escrow", 1000, 14)

	now := time.Unix(int64(bond.Timestamp), 0).AddDate(0, 0, 15) // 15 days later
	released, err := IsSelfReleased(bond, now, nil)
	if err != nil {
		t.Fatalf("IsSelfReleased: %v", err)
	}
	if !released {
		t.Error("expected self-release after the window with no challenge")
	}
}

func TestIsSelfReleasedFalseBeforeWindow(t *testing.T) {
	owner := newIdentity(t)
	eventID := fakeEventID(t)
	bond := verifiedBondClaim(t, owner, eventID, "escrow", 1000, 14)

	now := time.Unix(int64(bond.Timestamp), 0).AddDate(0, 0, 7) // only 7 of 14 days
	released, err := IsSelfReleased(bond, now, nil)
	if err != nil {
		t.Fatalf("IsSelfReleased: %v", err)
	}
	if released {
		t.Error("expected no self-release before the window elapses")
	}
}

func TestIsSelfReleasedFalseOnceChallenged(t *testing.T) {
	owner := newIdentity(t)
	challenger := newIdentity(t)
	eventID := fakeEventID(t)
	bond := verifiedBondClaim(t, owner, eventID, "escrow", 1000, 14)
	challenge := verifiedChallengeClaim(t, challenger, owner.PublicKey, eventID, "challenger-escrow", 1000)

	// Well past the window — would self-release if unchallenged.
	now := time.Unix(int64(bond.Timestamp), 0).AddDate(1, 0, 0)
	released, err := IsSelfReleased(bond, now, []attestation.Claim{challenge})
	if err != nil {
		t.Fatalf("IsSelfReleased: %v", err)
	}
	if released {
		t.Error("expected no self-release once a matching challenge exists, regardless of how much time has passed")
	}
}

func TestIsSelfReleasedFalseForUnresolvedChallengeEvenLongAfter(t *testing.T) {
	owner := newIdentity(t)
	challenger := newIdentity(t)
	eventID := fakeEventID(t)
	bond := verifiedBondClaim(t, owner, eventID, "escrow", 1000, 14)
	challenge := verifiedChallengeClaim(t, challenger, owner.PublicKey, eventID, "challenger-escrow", 1000)

	// Self-release is about "was it ever challenged," not "was the
	// dispute resolved" — no verdict claim exists here at all, and
	// that must still block release.
	now := time.Unix(int64(bond.Timestamp), 0).AddDate(10, 0, 0)
	released, err := IsSelfReleased(bond, now, []attestation.Claim{challenge})
	if err != nil {
		t.Fatalf("IsSelfReleased: %v", err)
	}
	if released {
		t.Error("expected an unresolved (no verdict at all) challenge to still block self-release")
	}
}

func TestIsSelfReleasedIgnoresChallengeForDifferentEvent(t *testing.T) {
	owner := newIdentity(t)
	challenger := newIdentity(t)
	eventID := fakeEventID(t)
	otherEventID := fakeEventID(t)
	bond := verifiedBondClaim(t, owner, eventID, "escrow", 1000, 14)
	unrelatedChallenge := verifiedChallengeClaim(t, challenger, owner.PublicKey, otherEventID, "challenger-escrow", 1000)

	now := time.Unix(int64(bond.Timestamp), 0).AddDate(0, 0, 15)
	released, err := IsSelfReleased(bond, now, []attestation.Claim{unrelatedChallenge})
	if err != nil {
		t.Fatalf("IsSelfReleased: %v", err)
	}
	if !released {
		t.Error("expected self-release: the only challenge present is for a different disputed event")
	}
}
