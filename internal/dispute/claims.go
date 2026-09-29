// Package dispute implements D001's Bonding/slashing mechanism, per
// A005's own scoping: dispute type 1 only (behavioral/quality
// disputes between two identified parties). This package owns the
// protocol/decision layer only — four new claim types, panel
// confirmation, verdict resolution, and self-release timing — never
// the actual Bitcoin escrow script construction, funding, or on-chain
// verification, which A005 deliberately defers to a separate,
// not-yet-filed follow-up.
package dispute

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/nbd-wtf/go-nostr"

	"github.com/lnd3/persona/internal/attestation"
	"github.com/lnd3/persona/internal/identity"
)

// The four net.persona.core.* claim types this action defines, per
// A005's Decisions — a bond, a challenge, an arbiter panel, and a
// verdict are each just attestations, following the same
// claim-type-based precedent A003 set for recovery.
const (
	BondClaimType      = "net.persona.core.bond"
	ChallengeClaimType = "net.persona.core.dispute_challenge"
	PanelClaimType     = "net.persona.core.arbiter_panel"
	VerdictClaimType   = "net.persona.core.dispute_verdict"
)

const eventIDPattern = `[0-9a-f]{64}`

var (
	bondValueRE      = regexp.MustCompile(`^(` + eventIDPattern + `):(.+):([0-9]+):([0-9]+)$`)
	challengeValueRE = regexp.MustCompile(`^(` + eventIDPattern + `):(.+):([0-9]+)$`)
	panelValueRE     = regexp.MustCompile(`^(` + eventIDPattern + `):(.+)$`)
	verdictValueRE   = regexp.MustCompile(`^(` + eventIDPattern + `):(attester_wins|challenger_wins)$`)
)

// ErrMalformedClaimValue is returned when a dispute-related
// claim_value doesn't match its expected shape.
var ErrMalformedClaimValue = fmt.Errorf("dispute: malformed claim_value")

// BondInfo is a parsed net.persona.core.bond claim_value: the
// attester's own attestation event (DisputedEventID) is secured by
// AmountSats held in EscrowRef, self-releasing after WindowDays with
// no challenge.
type BondInfo struct {
	DisputedEventID string
	EscrowRef       string
	AmountSats      int64
	WindowDays      int
}

// NewBondClaim builds and signs a bond claim: attester = subject =
// owner (self-referential — the bond secures the owner's own
// disputed attestation).
func NewBondClaim(owner identity.Persona, disputedEventID, escrowRef string, amountSats int64, windowDays int) (*nostr.Event, error) {
	if !isEventID(disputedEventID) {
		return nil, fmt.Errorf("%w: disputed_event_id %q is not a 64-char hex event id", ErrMalformedClaimValue, disputedEventID)
	}
	value := fmt.Sprintf("%s:%s:%d:%d", disputedEventID, escrowRef, amountSats, windowDays)
	return attestation.New(owner, owner.PublicKey, BondClaimType, value, "bonded attestation")
}

// ParseBondClaim extracts a BondInfo from a verified
// attestation.Claim of BondClaimType.
func ParseBondClaim(claim attestation.Claim) (BondInfo, error) {
	if claim.ClaimType != BondClaimType {
		return BondInfo{}, fmt.Errorf("dispute: claim_type %q is not %q", claim.ClaimType, BondClaimType)
	}
	m := bondValueRE.FindStringSubmatch(claim.ClaimValue)
	if m == nil {
		return BondInfo{}, fmt.Errorf("%w: got %q", ErrMalformedClaimValue, claim.ClaimValue)
	}
	amount, err := strconv.ParseInt(m[3], 10, 64)
	if err != nil {
		return BondInfo{}, fmt.Errorf("%w: amount_sats %q: %v", ErrMalformedClaimValue, m[3], err)
	}
	window, err := strconv.Atoi(m[4])
	if err != nil {
		return BondInfo{}, fmt.Errorf("%w: window_days %q: %v", ErrMalformedClaimValue, m[4], err)
	}
	return BondInfo{DisputedEventID: m[1], EscrowRef: m[2], AmountSats: amount, WindowDays: window}, nil
}

// ChallengeInfo is a parsed net.persona.core.dispute_challenge
// claim_value: the challenger's symmetric stake against a specific
// disputed attestation, per D001's anti-griefing requirement.
type ChallengeInfo struct {
	DisputedEventID string
	EscrowRef       string
	AmountSats      int64
}

// NewChallengeClaim builds and signs a dispute_challenge claim:
// attester = challenger, subject = the disputed attestation's own
// attester.
func NewChallengeClaim(challenger identity.Persona, disputedAttesterPubkey, disputedEventID, escrowRef string, amountSats int64) (*nostr.Event, error) {
	if !isEventID(disputedEventID) {
		return nil, fmt.Errorf("%w: disputed_event_id %q is not a 64-char hex event id", ErrMalformedClaimValue, disputedEventID)
	}
	value := fmt.Sprintf("%s:%s:%d", disputedEventID, escrowRef, amountSats)
	return attestation.New(challenger, disputedAttesterPubkey, ChallengeClaimType, value, "dispute challenge")
}

// ParseChallengeClaim extracts a ChallengeInfo from a verified
// attestation.Claim of ChallengeClaimType.
func ParseChallengeClaim(claim attestation.Claim) (ChallengeInfo, error) {
	if claim.ClaimType != ChallengeClaimType {
		return ChallengeInfo{}, fmt.Errorf("dispute: claim_type %q is not %q", claim.ClaimType, ChallengeClaimType)
	}
	m := challengeValueRE.FindStringSubmatch(claim.ClaimValue)
	if m == nil {
		return ChallengeInfo{}, fmt.Errorf("%w: got %q", ErrMalformedClaimValue, claim.ClaimValue)
	}
	amount, err := strconv.ParseInt(m[3], 10, 64)
	if err != nil {
		return ChallengeInfo{}, fmt.Errorf("%w: amount_sats %q: %v", ErrMalformedClaimValue, m[3], err)
	}
	return ChallengeInfo{DisputedEventID: m[1], EscrowRef: m[2], AmountSats: amount}, nil
}

// PanelInfo is a parsed net.persona.core.arbiter_panel claim_value:
// one side's named arbiter set for a specific dispute.
type PanelInfo struct {
	DisputedEventID string
	Arbiters        []string
}

// NewPanelClaim builds and signs an arbiter_panel claim. publisher is
// either the original attester or the challenger (either side may
// publish their own copy); subject is always the disputed
// attestation's own attester, so both sides' claims are directly
// comparable regardless of who published them.
func NewPanelClaim(publisher identity.Persona, disputedAttesterPubkey, disputedEventID string, arbiters []string) (*nostr.Event, error) {
	if !isEventID(disputedEventID) {
		return nil, fmt.Errorf("%w: disputed_event_id %q is not a 64-char hex event id", ErrMalformedClaimValue, disputedEventID)
	}
	if err := ValidatePanelSize(len(arbiters)); err != nil {
		return nil, err
	}
	value := fmt.Sprintf("%s:%s", disputedEventID, joinCommaHex(arbiters))
	return attestation.New(publisher, disputedAttesterPubkey, PanelClaimType, value, "arbiter panel selection")
}

// ParsePanelClaim extracts a PanelInfo from a verified
// attestation.Claim of PanelClaimType.
func ParsePanelClaim(claim attestation.Claim) (PanelInfo, error) {
	if claim.ClaimType != PanelClaimType {
		return PanelInfo{}, fmt.Errorf("dispute: claim_type %q is not %q", claim.ClaimType, PanelClaimType)
	}
	m := panelValueRE.FindStringSubmatch(claim.ClaimValue)
	if m == nil {
		return PanelInfo{}, fmt.Errorf("%w: got %q", ErrMalformedClaimValue, claim.ClaimValue)
	}
	arbiters := splitCommaHex(m[2])
	if err := ValidatePanelSize(len(arbiters)); err != nil {
		return PanelInfo{}, err
	}
	return PanelInfo{DisputedEventID: m[1], Arbiters: arbiters}, nil
}

// Verdict is one arbiter's ruling: AttesterWins or ChallengerWins.
type Verdict string

const (
	AttesterWins   Verdict = "attester_wins"
	ChallengerWins Verdict = "challenger_wins"
)

// VerdictInfo is a parsed net.persona.core.dispute_verdict
// claim_value.
type VerdictInfo struct {
	DisputedEventID string
	Verdict         Verdict
}

// NewVerdictClaim builds and signs a dispute_verdict claim: attester =
// the ruling arbiter, subject = the disputed attestation's own
// attester.
func NewVerdictClaim(arbiter identity.Persona, disputedAttesterPubkey, disputedEventID string, verdict Verdict) (*nostr.Event, error) {
	if !isEventID(disputedEventID) {
		return nil, fmt.Errorf("%w: disputed_event_id %q is not a 64-char hex event id", ErrMalformedClaimValue, disputedEventID)
	}
	if verdict != AttesterWins && verdict != ChallengerWins {
		return nil, fmt.Errorf("%w: verdict %q is neither %q nor %q", ErrMalformedClaimValue, verdict, AttesterWins, ChallengerWins)
	}
	value := fmt.Sprintf("%s:%s", disputedEventID, verdict)
	return attestation.New(arbiter, disputedAttesterPubkey, VerdictClaimType, value, "dispute verdict")
}

// ParseVerdictClaim extracts a VerdictInfo from a verified
// attestation.Claim of VerdictClaimType.
func ParseVerdictClaim(claim attestation.Claim) (VerdictInfo, error) {
	if claim.ClaimType != VerdictClaimType {
		return VerdictInfo{}, fmt.Errorf("dispute: claim_type %q is not %q", claim.ClaimType, VerdictClaimType)
	}
	m := verdictValueRE.FindStringSubmatch(claim.ClaimValue)
	if m == nil {
		return VerdictInfo{}, fmt.Errorf("%w: got %q", ErrMalformedClaimValue, claim.ClaimValue)
	}
	return VerdictInfo{DisputedEventID: m[1], Verdict: Verdict(m[2])}, nil
}

var eventIDRE = regexp.MustCompile(`^` + eventIDPattern + `$`)

func isEventID(s string) bool { return eventIDRE.MatchString(s) }
