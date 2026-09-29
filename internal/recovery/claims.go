package recovery

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"

	"github.com/nbd-wtf/go-nostr"

	"github.com/lnd3/persona/internal/attestation"
	"github.com/lnd3/persona/internal/identity"
)

// GuardianClaimType and ConfirmClaimType are A003's two new
// net.persona.core.* claim types, both reusing A001's attestation
// primitive and claim_type namespacing rather than a separate system.
const (
	GuardianClaimType = "net.persona.core.recovery_guardian"
	ConfirmClaimType  = "net.persona.core.recovery_confirm"
)

var (
	guardianValueRE = regexp.MustCompile(`^([1-9][0-9]*)-of-([1-9][0-9]*):([0-9a-f]+)$`)
	confirmValueRE  = regexp.MustCompile(`^([0-9a-f]+):([0-9a-f]+)$`)
)

// ErrInvalidClaimValue is returned when a recovery claim_value doesn't
// match its expected "<threshold>-of-<n>:<group_id>" or
// "<group_id>:<nonce>" shape.
var ErrInvalidClaimValue = errors.New("recovery: malformed claim_value")

// NewGroupID and NewNonce generate the random hex tokens a guardian
// designation round and an individual recovery attempt are each keyed
// by, so a verifier can tell which recovery_guardian claims belong to
// the same designation round, and which recovery_confirm claims
// belong to the same recovery attempt, per A003's Decisions.
func NewGroupID() (string, error) { return randomHex(8) }
func NewNonce() (string, error)   { return randomHex(8) }

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("recovery: generating random token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// GuardianDesignation is a recovery_guardian claim's parsed
// claim_value: this guardian is one of Total guardians, Threshold of
// whom must confirm to authorize recovery, as part of designation
// round GroupID.
type GuardianDesignation struct {
	Threshold int
	Total     int
	GroupID   string
}

// NewGuardianClaim builds and signs a recovery_guardian attestation:
// owner designates guardianPubKey as one of total guardians in
// designation round groupID, threshold of whom must confirm to
// authorize recovery.
func NewGuardianClaim(owner identity.Persona, guardianPubKey string, threshold, total int, groupID string) (*nostr.Event, error) {
	if threshold < 1 || total < threshold {
		return nil, fmt.Errorf("recovery: invalid threshold=%d/total=%d", threshold, total)
	}
	value := fmt.Sprintf("%d-of-%d:%s", threshold, total, groupID)
	return attestation.New(owner, guardianPubKey, GuardianClaimType, value, "recovery guardian designation")
}

// ParseGuardianClaim extracts a GuardianDesignation from a verified
// attestation.Claim of GuardianClaimType.
func ParseGuardianClaim(claim attestation.Claim) (GuardianDesignation, error) {
	if claim.ClaimType != GuardianClaimType {
		return GuardianDesignation{}, fmt.Errorf("recovery: claim_type %q is not %q", claim.ClaimType, GuardianClaimType)
	}
	m := guardianValueRE.FindStringSubmatch(claim.ClaimValue)
	if m == nil {
		return GuardianDesignation{}, fmt.Errorf("%w: got %q", ErrInvalidClaimValue, claim.ClaimValue)
	}
	var threshold, total int
	fmt.Sscanf(m[1], "%d", &threshold)
	fmt.Sscanf(m[2], "%d", &total)
	if threshold < 1 || total < threshold {
		return GuardianDesignation{}, fmt.Errorf("%w: threshold=%d/total=%d", ErrInvalidClaimValue, threshold, total)
	}
	return GuardianDesignation{Threshold: threshold, Total: total, GroupID: m[3]}, nil
}

// NewConfirmClaim builds and signs a recovery_confirm attestation:
// guardian agrees to participate in the recovery attempt identified
// by (groupID, nonce), for the owner's ownerPubKey.
func NewConfirmClaim(guardian identity.Persona, ownerPubKey, groupID, nonce string) (*nostr.Event, error) {
	value := fmt.Sprintf("%s:%s", groupID, nonce)
	return attestation.New(guardian, ownerPubKey, ConfirmClaimType, value, "recovery confirmation")
}

// RecoveryConfirmation is a recovery_confirm claim's parsed
// claim_value: this guardian confirms recovery attempt Nonce under
// designation round GroupID.
type RecoveryConfirmation struct {
	GroupID string
	Nonce   string
}

// ParseConfirmClaim extracts a RecoveryConfirmation from a verified
// attestation.Claim of ConfirmClaimType.
func ParseConfirmClaim(claim attestation.Claim) (RecoveryConfirmation, error) {
	if claim.ClaimType != ConfirmClaimType {
		return RecoveryConfirmation{}, fmt.Errorf("recovery: claim_type %q is not %q", claim.ClaimType, ConfirmClaimType)
	}
	m := confirmValueRE.FindStringSubmatch(claim.ClaimValue)
	if m == nil {
		return RecoveryConfirmation{}, fmt.Errorf("%w: got %q", ErrInvalidClaimValue, claim.ClaimValue)
	}
	return RecoveryConfirmation{GroupID: m[1], Nonce: m[2]}, nil
}
