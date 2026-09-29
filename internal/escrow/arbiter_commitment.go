package escrow

import (
	"encoding/hex"
	"fmt"
	"regexp"

	"github.com/nbd-wtf/go-nostr"

	"github.com/lnd3/persona/internal/attestation"
	"github.com/lnd3/persona/internal/identity"
)

// ArbiterCommitmentClaimType is A006's own addition to A005's
// protocol layer (not a change to any of A005's already-shipped
// claim types): each side independently, immutably commits to their
// own chosen arbiter's identity at funding time — the part of the
// larger-bond tier's design that genuinely needs no cooperation,
// per A006's corrected Decisions (see plan/actions/A006's Log).
const ArbiterCommitmentClaimType = "net.persona.core.arbiter_commitment"

var arbiterCommitmentValueRE = regexp.MustCompile(`^([0-9a-f]{64}):([0-9a-f]{66})$`)

// ErrMalformedArbiterCommitment is returned when an
// arbiter_commitment claim_value doesn't match its expected
// "<bond_or_challenge_event_id>:<arbiter_pubkey>" shape. The arbiter
// pubkey here is a 33-byte compressed secp256k1 ECDSA key (this
// package's own Bitcoin script convention — see multisig.go), not a
// Nostr x-only pubkey, since it must actually satisfy a Bitcoin
// OP_CHECKMULTISIG, not sign Nostr events.
var ErrMalformedArbiterCommitment = fmt.Errorf("escrow: malformed arbiter_commitment claim_value")

// ArbiterCommitment is a parsed arbiter_commitment claim_value.
type ArbiterCommitment struct {
	FundingEventID string // the bond or challenge event this commitment belongs to
	ArbiterPubKey  []byte // 33-byte compressed secp256k1 pubkey
}

// NewArbiterCommitmentClaim builds and signs an arbiter_commitment
// attestation: owner (the bond's attester, or a challenge's
// challenger) commits to arbiterPubKey for fundingEventID, self-
// referentially (subject = owner, same shape as A005's own bond
// claim).
func NewArbiterCommitmentClaim(owner identity.Persona, fundingEventID string, arbiterPubKey []byte) (*nostr.Event, error) {
	if err := validatePubKey(arbiterPubKey); err != nil {
		return nil, err
	}
	value := fmt.Sprintf("%s:%x", fundingEventID, arbiterPubKey)
	return attestation.New(owner, owner.PublicKey, ArbiterCommitmentClaimType, value, "arbiter commitment")
}

// ParseArbiterCommitmentClaim extracts an ArbiterCommitment from a
// verified attestation.Claim of ArbiterCommitmentClaimType.
func ParseArbiterCommitmentClaim(claim attestation.Claim) (ArbiterCommitment, error) {
	if claim.ClaimType != ArbiterCommitmentClaimType {
		return ArbiterCommitment{}, fmt.Errorf("escrow: claim_type %q is not %q", claim.ClaimType, ArbiterCommitmentClaimType)
	}
	m := arbiterCommitmentValueRE.FindStringSubmatch(claim.ClaimValue)
	if m == nil {
		return ArbiterCommitment{}, fmt.Errorf("%w: got %q", ErrMalformedArbiterCommitment, claim.ClaimValue)
	}
	pubKey, err := hex.DecodeString(m[2])
	if err != nil {
		return ArbiterCommitment{}, fmt.Errorf("%w: invalid pubkey hex: %v", ErrMalformedArbiterCommitment, err)
	}
	return ArbiterCommitment{FundingEventID: m[1], ArbiterPubKey: pubKey}, nil
}
