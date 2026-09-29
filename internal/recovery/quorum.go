package recovery

import (
	"errors"
	"fmt"

	"github.com/lnd3/persona/internal/attestation"
)

// ErrInconsistentGuardianSet is returned when recovery_guardian claims
// sharing a groupID disagree about the threshold — each claim in a
// single designation round must state the same "<threshold>-of-<n>".
var ErrInconsistentGuardianSet = errors.New("recovery: recovery_guardian claims for this group_id disagree on threshold/total")

// GuardianSet collects the guardian pubkeys and threshold for a given
// designation round (groupID) out of a set of already-verified
// recovery_guardian claims (e.g. from attestation.FetchForSubject
// against the owner's own pubkey).
func GuardianSet(claims []attestation.Claim, groupID string) (threshold int, guardians map[string]bool, err error) {
	guardians = make(map[string]bool)
	for _, c := range claims {
		if c.ClaimType != GuardianClaimType {
			continue
		}
		d, err := ParseGuardianClaim(c)
		if err != nil || d.GroupID != groupID {
			continue
		}
		if threshold == 0 {
			threshold = d.Threshold
		} else if threshold != d.Threshold {
			return 0, nil, ErrInconsistentGuardianSet
		}
		guardians[c.SubjectKey] = true
	}
	if threshold == 0 {
		return 0, nil, fmt.Errorf("recovery: no recovery_guardian claims found for group_id %q", groupID)
	}
	return threshold, guardians, nil
}

// ThresholdMet reports whether enough designated guardians have
// confirmed a specific recovery attempt (groupID, nonce): counts one
// vote per distinct guardian pubkey that is both a member of that
// group's guardian set and the attester of a matching
// recovery_confirm claim — a confirm from a non-guardian, or for a
// different group_id/nonce, doesn't count at all.
func ThresholdMet(guardianClaims, confirmClaims []attestation.Claim, groupID, nonce string) (met bool, count int, threshold int, err error) {
	threshold, guardians, err := GuardianSet(guardianClaims, groupID)
	if err != nil {
		return false, 0, 0, err
	}

	confirmed := make(map[string]bool)
	for _, c := range confirmClaims {
		if c.ClaimType != ConfirmClaimType {
			continue
		}
		rc, err := ParseConfirmClaim(c)
		if err != nil || rc.GroupID != groupID || rc.Nonce != nonce {
			continue
		}
		if !guardians[c.AttesterKey] {
			continue // confirm from a pubkey never designated as a guardian in this round
		}
		confirmed[c.AttesterKey] = true // de-duplicate: same guardian confirming twice counts once
	}

	count = len(confirmed)
	return count >= threshold, count, threshold, nil
}
