package dispute

import (
	"github.com/lnd3/persona/internal/attestation"
)

// Resolve counts dispute_verdict claims against a confirmedPanel (the
// arbiter set ConfirmedPanel already agreed on for disputedEventID)
// and reports the winner once strictly more than half the panel has
// voted the same way — one vote per distinct arbiter pubkey (a
// duplicate verdict from the same arbiter doesn't count twice, the
// same distinct-attester counting A002 uses for trust edges), and a
// verdict from outside the confirmed panel doesn't count at all.
// decided is false ("undecided") whenever no side has reached strict
// majority yet — including a tie, which a majority threshold makes
// structurally impossible to resolve without one, by design.
func Resolve(verdicts []attestation.Claim, disputedEventID string, confirmedPanel []string) (winner Verdict, decided bool, err error) {
	inPanel := make(map[string]bool, len(confirmedPanel))
	for _, p := range confirmedPanel {
		inPanel[p] = true
	}

	votes := make(map[string]Verdict) // arbiter -> their vote, deduplicated
	for _, c := range verdicts {
		if c.ClaimType != VerdictClaimType || !inPanel[c.AttesterKey] {
			continue
		}
		info, perr := ParseVerdictClaim(c)
		if perr != nil || info.DisputedEventID != disputedEventID {
			continue
		}
		votes[c.AttesterKey] = info.Verdict
	}

	var attesterVotes, challengerVotes int
	for _, v := range votes {
		switch v {
		case AttesterWins:
			attesterVotes++
		case ChallengerWins:
			challengerVotes++
		}
	}

	majority := len(confirmedPanel)/2 + 1
	switch {
	case attesterVotes >= majority:
		return AttesterWins, true, nil
	case challengerVotes >= majority:
		return ChallengerWins, true, nil
	default:
		return "", false, nil
	}
}
