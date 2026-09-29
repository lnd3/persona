package dispute

import (
	"time"

	"github.com/lnd3/persona/internal/attestation"
)

// IsSelfReleased reports whether bond has passed its own self-release
// window with no matching challenge — pure arithmetic, never an
// on-chain check (whether the escrow has actually moved is the
// not-yet-filed escrow-settlement action's job, per A005's Context).
//
// A bond that has passed its window but has *any* matching
// dispute_challenge — resolved or not — is never self-released: this
// answers "was it ever challenged," not "was the challenge upheld,"
// so an open, unresolved dispute can't have its bond released out
// from under an active arbitration.
func IsSelfReleased(bond attestation.Claim, now time.Time, challenges []attestation.Claim) (bool, error) {
	info, err := ParseBondClaim(bond)
	if err != nil {
		return false, err
	}

	deadline := time.Unix(int64(bond.Timestamp), 0).AddDate(0, 0, info.WindowDays)
	if !now.After(deadline) {
		return false, nil
	}

	for _, c := range challenges {
		if c.ClaimType != ChallengeClaimType {
			continue
		}
		challengeInfo, err := ParseChallengeClaim(c)
		if err != nil {
			continue // malformed claims aren't real challenges
		}
		if challengeInfo.DisputedEventID == info.DisputedEventID {
			return false, nil
		}
	}

	return true, nil
}
