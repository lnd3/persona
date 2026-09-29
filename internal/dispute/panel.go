package dispute

import (
	"errors"
	"sort"
	"strings"

	"github.com/lnd3/persona/internal/attestation"
)

// ErrInvalidPanelSize is returned when an arbiter panel isn't exactly
// one arbiter, or an odd number of at least three — D001 describes a
// single arbiter for small bonds or a jointly-selected panel for
// larger ones, but never addresses even-panel tie-breaking, so this
// package refuses to invent one rather than silently allow it.
var ErrInvalidPanelSize = errors.New("dispute: arbiter panel must be exactly 1 arbiter, or an odd number >= 3")

// ValidatePanelSize enforces A005's panel-size decision.
func ValidatePanelSize(n int) error {
	if n == 1 {
		return nil
	}
	if n >= 3 && n%2 == 1 {
		return nil
	}
	return ErrInvalidPanelSize
}

func joinCommaHex(pubkeys []string) string { return strings.Join(pubkeys, ",") }

func splitCommaHex(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ",")
}

func sortedCopy(s []string) []string {
	out := make([]string, len(s))
	copy(out, s)
	sort.Strings(out)
	return out
}

func sameSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	sa, sb := sortedCopy(a), sortedCopy(b)
	for i := range sa {
		if sa[i] != sb[i] {
			return false
		}
	}
	return true
}

// ConfirmedPanel checks whether sideA and sideB (the disputed
// attestation's original attester, and the challenger) have each
// published an arbiter_panel claim for disputedEventID naming the
// identical arbiter set — D001's joint-selection requirement,
// represented as two independent attestations that must agree rather
// than a new co-signing primitive. If either side published more than
// one (e.g. a redesignation), the most recently timestamped one is
// used. Returns the confirmed arbiter set and true only when both
// sides agree; otherwise (nil, false).
func ConfirmedPanel(panelClaims []attestation.Claim, disputedEventID, sideA, sideB string) ([]string, bool, error) {
	latest := make(map[string]attestation.Claim) // attester -> most recent matching claim

	for _, c := range panelClaims {
		if c.ClaimType != PanelClaimType {
			continue
		}
		if c.AttesterKey != sideA && c.AttesterKey != sideB {
			continue
		}
		info, err := ParsePanelClaim(c)
		if err != nil || info.DisputedEventID != disputedEventID {
			continue
		}
		if existing, ok := latest[c.AttesterKey]; !ok || c.Timestamp > existing.Timestamp {
			latest[c.AttesterKey] = c
		}
	}

	claimA, okA := latest[sideA]
	claimB, okB := latest[sideB]
	if !okA || !okB {
		return nil, false, nil
	}
	infoA, err := ParsePanelClaim(claimA)
	if err != nil {
		return nil, false, err
	}
	infoB, err := ParsePanelClaim(claimB)
	if err != nil {
		return nil, false, err
	}
	if !sameSet(infoA.Arbiters, infoB.Arbiters) {
		return nil, false, nil
	}
	return infoA.Arbiters, true, nil
}
