package dispute

import (
	"testing"

	"github.com/lnd3/persona/internal/attestation"
	"github.com/lnd3/persona/internal/identity"
)

func verifiedVerdictClaim(t *testing.T, arbiter identity.Persona, disputedAttesterPubkey, eventID string, v Verdict) attestation.Claim {
	t.Helper()
	evt, err := NewVerdictClaim(arbiter, disputedAttesterPubkey, eventID, v)
	if err != nil {
		t.Fatalf("NewVerdictClaim: %v", err)
	}
	claim, err := attestation.Verify(evt)
	if err != nil {
		t.Fatalf("attestation.Verify: %v", err)
	}
	return claim
}

func TestResolveSingleArbiterDecidesImmediately(t *testing.T) {
	attester := newIdentity(t)
	arbiter := newIdentity(t)
	eventID := fakeEventID(t)
	panel := []string{arbiter.PublicKey}

	verdicts := []attestation.Claim{
		verifiedVerdictClaim(t, arbiter, attester.PublicKey, eventID, AttesterWins),
	}

	winner, decided, err := Resolve(verdicts, eventID, panel)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if !decided || winner != AttesterWins {
		t.Errorf("winner=%v decided=%v, want AttesterWins/true", winner, decided)
	}
}

func TestResolveThreeArbiterPanelNeedsTwoVotes(t *testing.T) {
	attester := newIdentity(t)
	a1, a2, a3 := newIdentity(t), newIdentity(t), newIdentity(t)
	eventID := fakeEventID(t)
	panel := []string{a1.PublicKey, a2.PublicKey, a3.PublicKey}

	// 1-1 split (a3 hasn't voted yet): not decided.
	verdicts := []attestation.Claim{
		verifiedVerdictClaim(t, a1, attester.PublicKey, eventID, AttesterWins),
		verifiedVerdictClaim(t, a2, attester.PublicKey, eventID, ChallengerWins),
	}
	_, decided, err := Resolve(verdicts, eventID, panel)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if decided {
		t.Error("expected undecided at 1-1 with a 3-arbiter panel")
	}

	// a3 breaks the tie for challenger: now decided 2-1.
	verdicts = append(verdicts, verifiedVerdictClaim(t, a3, attester.PublicKey, eventID, ChallengerWins))
	winner, decided, err := Resolve(verdicts, eventID, panel)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if !decided || winner != ChallengerWins {
		t.Errorf("winner=%v decided=%v, want ChallengerWins/true at 2-1", winner, decided)
	}
}

func TestResolveDuplicateVerdictFromSameArbiterDoesNotCountTwice(t *testing.T) {
	attester := newIdentity(t)
	a1, a2, a3 := newIdentity(t), newIdentity(t), newIdentity(t)
	eventID := fakeEventID(t)
	panel := []string{a1.PublicKey, a2.PublicKey, a3.PublicKey}

	// a1 votes twice (e.g. republished); a2 votes once the other way.
	// Without dedup this would look like 2-1 for AttesterWins; with
	// dedup it's 1-1, still undecided.
	verdicts := []attestation.Claim{
		verifiedVerdictClaim(t, a1, attester.PublicKey, eventID, AttesterWins),
		verifiedVerdictClaim(t, a1, attester.PublicKey, eventID, AttesterWins),
		verifiedVerdictClaim(t, a2, attester.PublicKey, eventID, ChallengerWins),
	}
	_, decided, err := Resolve(verdicts, eventID, panel)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if decided {
		t.Error("expected undecided: a1's duplicate vote must not count twice")
	}
}

func TestResolveIgnoresVerdictFromOutsidePanel(t *testing.T) {
	attester := newIdentity(t)
	a1 := newIdentity(t)
	outsider := newIdentity(t)
	eventID := fakeEventID(t)
	panel := []string{a1.PublicKey} // 1-arbiter panel; outsider is not on it

	verdicts := []attestation.Claim{
		verifiedVerdictClaim(t, outsider, attester.PublicKey, eventID, AttesterWins),
	}
	_, decided, err := Resolve(verdicts, eventID, panel)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if decided {
		t.Error("expected undecided: a verdict from outside the confirmed panel must not count")
	}
}
