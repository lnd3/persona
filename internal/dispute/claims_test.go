package dispute

import (
	"strings"
	"testing"

	"github.com/lnd3/persona/internal/attestation"
	"github.com/lnd3/persona/internal/identity"
)

func fakeEventID(t *testing.T) string {
	t.Helper()
	p, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	return p.PublicKey // any 64-hex-char string works as a stand-in event id
}

func TestBondClaimRoundTrip(t *testing.T) {
	owner, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	eventID := fakeEventID(t)

	evt, err := NewBondClaim(owner, eventID, "bc1qescrowaddress", 50000, 14)
	if err != nil {
		t.Fatalf("NewBondClaim: %v", err)
	}
	claim, err := attestation.Verify(evt)
	if err != nil {
		t.Fatalf("attestation.Verify: %v", err)
	}
	if claim.AttesterKey != owner.PublicKey || claim.SubjectKey != owner.PublicKey {
		t.Errorf("bond claim attester/subject = %s/%s, want both %s (self-referential)", claim.AttesterKey, claim.SubjectKey, owner.PublicKey)
	}

	info, err := ParseBondClaim(claim)
	if err != nil {
		t.Fatalf("ParseBondClaim: %v", err)
	}
	if info.DisputedEventID != eventID || info.EscrowRef != "bc1qescrowaddress" || info.AmountSats != 50000 || info.WindowDays != 14 {
		t.Errorf("parsed = %+v", info)
	}
}

func TestBondClaimEscrowRefMayContainColons(t *testing.T) {
	owner, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	eventID := fakeEventID(t)
	// A descriptor-like escrow ref containing colons must still
	// round-trip: the greedy middle capture must not be broken by
	// colons inside it, only by the trailing numeric fields.
	escrowRef := "wsh(multi(2,key1:fp/path,key2))"

	evt, err := NewBondClaim(owner, eventID, escrowRef, 1, 1)
	if err != nil {
		t.Fatalf("NewBondClaim: %v", err)
	}
	claim, err := attestation.Verify(evt)
	if err != nil {
		t.Fatalf("attestation.Verify: %v", err)
	}
	info, err := ParseBondClaim(claim)
	if err != nil {
		t.Fatalf("ParseBondClaim: %v", err)
	}
	if info.EscrowRef != escrowRef {
		t.Errorf("EscrowRef = %q, want %q", info.EscrowRef, escrowRef)
	}
}

func TestBondClaimRejectsInvalidEventID(t *testing.T) {
	owner, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	if _, err := NewBondClaim(owner, "not-an-event-id", "escrow", 1, 1); err == nil {
		t.Fatal("expected error for invalid disputed_event_id")
	}
}

func TestParseBondClaimRejectsMalformedValue(t *testing.T) {
	claim := attestation.Claim{ClaimType: BondClaimType, ClaimValue: "garbage"}
	if _, err := ParseBondClaim(claim); err == nil {
		t.Fatal("expected error for malformed claim_value")
	}
}

func TestChallengeClaimRoundTrip(t *testing.T) {
	challenger, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	attester, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	eventID := fakeEventID(t)

	evt, err := NewChallengeClaim(challenger, attester.PublicKey, eventID, "bc1qchallengerescrow", 50000)
	if err != nil {
		t.Fatalf("NewChallengeClaim: %v", err)
	}
	claim, err := attestation.Verify(evt)
	if err != nil {
		t.Fatalf("attestation.Verify: %v", err)
	}
	if claim.AttesterKey != challenger.PublicKey || claim.SubjectKey != attester.PublicKey {
		t.Errorf("challenge attester/subject = %s/%s, want %s/%s", claim.AttesterKey, claim.SubjectKey, challenger.PublicKey, attester.PublicKey)
	}
	info, err := ParseChallengeClaim(claim)
	if err != nil {
		t.Fatalf("ParseChallengeClaim: %v", err)
	}
	if info.DisputedEventID != eventID || info.EscrowRef != "bc1qchallengerescrow" || info.AmountSats != 50000 {
		t.Errorf("parsed = %+v", info)
	}
}

func TestParseChallengeClaimRejectsMalformedValue(t *testing.T) {
	claim := attestation.Claim{ClaimType: ChallengeClaimType, ClaimValue: "garbage"}
	if _, err := ParseChallengeClaim(claim); err == nil {
		t.Fatal("expected error for malformed claim_value")
	}
}

func TestPanelClaimRoundTrip(t *testing.T) {
	publisher, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	attester, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	a1, _ := identity.New()
	a2, _ := identity.New()
	a3, _ := identity.New()
	eventID := fakeEventID(t)
	arbiters := []string{a1.PublicKey, a2.PublicKey, a3.PublicKey}

	evt, err := NewPanelClaim(publisher, attester.PublicKey, eventID, arbiters)
	if err != nil {
		t.Fatalf("NewPanelClaim: %v", err)
	}
	claim, err := attestation.Verify(evt)
	if err != nil {
		t.Fatalf("attestation.Verify: %v", err)
	}
	info, err := ParsePanelClaim(claim)
	if err != nil {
		t.Fatalf("ParsePanelClaim: %v", err)
	}
	if info.DisputedEventID != eventID || strings.Join(info.Arbiters, ",") != strings.Join(arbiters, ",") {
		t.Errorf("parsed = %+v", info)
	}
}

func TestNewPanelClaimRejectsInvalidSize(t *testing.T) {
	publisher, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	attester, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	a1, _ := identity.New()
	a2, _ := identity.New()
	eventID := fakeEventID(t)

	if _, err := NewPanelClaim(publisher, attester.PublicKey, eventID, []string{a1.PublicKey, a2.PublicKey}); err == nil {
		t.Fatal("expected error for even panel size 2")
	}
}

func TestParsePanelClaimRejectsMalformedValue(t *testing.T) {
	claim := attestation.Claim{ClaimType: PanelClaimType, ClaimValue: ""}
	if _, err := ParsePanelClaim(claim); err == nil {
		t.Fatal("expected error for malformed claim_value")
	}
}

func TestVerdictClaimRoundTrip(t *testing.T) {
	arbiter, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	attester, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	eventID := fakeEventID(t)

	evt, err := NewVerdictClaim(arbiter, attester.PublicKey, eventID, AttesterWins)
	if err != nil {
		t.Fatalf("NewVerdictClaim: %v", err)
	}
	claim, err := attestation.Verify(evt)
	if err != nil {
		t.Fatalf("attestation.Verify: %v", err)
	}
	info, err := ParseVerdictClaim(claim)
	if err != nil {
		t.Fatalf("ParseVerdictClaim: %v", err)
	}
	if info.DisputedEventID != eventID || info.Verdict != AttesterWins {
		t.Errorf("parsed = %+v", info)
	}
}

func TestNewVerdictClaimRejectsInvalidVerdict(t *testing.T) {
	arbiter, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	attester, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	eventID := fakeEventID(t)
	if _, err := NewVerdictClaim(arbiter, attester.PublicKey, eventID, Verdict("draw")); err == nil {
		t.Fatal("expected error for invalid verdict value")
	}
}

func TestParseVerdictClaimRejectsMalformedValue(t *testing.T) {
	claim := attestation.Claim{ClaimType: VerdictClaimType, ClaimValue: "garbage"}
	if _, err := ParseVerdictClaim(claim); err == nil {
		t.Fatal("expected error for malformed claim_value")
	}
}
