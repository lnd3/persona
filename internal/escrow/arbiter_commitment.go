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
// protocol layer (not a change to any of A005's already-shipped claim
// types): a *standing* declaration — "if anyone bonds a claim naming
// me as attester or subject, this is my own fallback arbiter" —
// published once, independent of any specific bond, per A006's
// corrected upfront-configured design (2026-09-30, see plan/actions/
// A006's Log). Not tied to a bond/challenge event id: a persona
// publishes this at most whenever they want to change their standing
// choice, and whichever commitment is current at the moment a
// specific bond's script is actually built and funded is the one that
// gets baked into that bond, permanently, for that bond.
const ArbiterCommitmentClaimType = "net.persona.core.arbiter_commitment"

var arbiterCommitmentValueRE = regexp.MustCompile(`^([0-9a-f]{66})$`)

// ErrMalformedArbiterCommitment is returned when an
// arbiter_commitment claim_value isn't a bare hex-encoded arbiter
// pubkey. The arbiter pubkey here is a 33-byte compressed secp256k1
// ECDSA key (this package's own Bitcoin script convention — see
// multisig.go), not a Nostr x-only pubkey, since it must actually
// satisfy a Bitcoin OP_CHECKMULTISIG, not sign Nostr events.
var ErrMalformedArbiterCommitment = fmt.Errorf("escrow: malformed arbiter_commitment claim_value")

// ArbiterCommitment is a parsed arbiter_commitment claim_value.
type ArbiterCommitment struct {
	ArbiterPubKey []byte // 33-byte compressed secp256k1 pubkey
}

// NewArbiterCommitmentClaim builds and signs a standing
// arbiter_commitment attestation: owner commits to arbiterPubKey as
// their own default fallback arbiter for any future dispute,
// self-referentially (subject = owner).
func NewArbiterCommitmentClaim(owner identity.Persona, arbiterPubKey []byte) (*nostr.Event, error) {
	if err := validatePubKey(arbiterPubKey); err != nil {
		return nil, err
	}
	value := fmt.Sprintf("%x", arbiterPubKey)
	return attestation.New(owner, owner.PublicKey, ArbiterCommitmentClaimType, value, "standing arbiter commitment")
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
	pubKey, err := hex.DecodeString(m[1])
	if err != nil {
		return ArbiterCommitment{}, fmt.Errorf("%w: invalid pubkey hex: %v", ErrMalformedArbiterCommitment, err)
	}
	return ArbiterCommitment{ArbiterPubKey: pubKey}, nil
}
