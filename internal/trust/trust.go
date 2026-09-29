// Package trust implements D001's relative-trust computation (A002):
// scoring a subject's attestations by weighting them against a
// verifier-supplied seed set of directly-trusted pubkeys, never a
// global reputation score. See plan/actions/A002 for the algorithmic
// decisions this package makes (bounded depth-3 decay-by-half
// propagation, a dedicated trust-edge claim_type, best-path scoring).
package trust

import (
	"context"
	"fmt"

	"github.com/nbd-wtf/go-nostr"

	"github.com/lnd3/persona/internal/attestation"
)

// EdgeClaimType is the dedicated claim_type for trust vouches,
// distinct from content claims (skill/ownership/recovery attestations
// aren't trust edges, per A002's Decisions section).
const EdgeClaimType = "net.persona.core.trust"

// MaxDepth bounds trust propagation so it can't leak arbitrarily far
// from the verifier — a judgment call, not derived, and revisitable.
const MaxDepth = 3

// directWeight is the weight assigned to a verifier's own seed
// pubkeys (depth 1). Each additional hop halves it.
const directWeight = 1.0

// Weights maps a pubkey to its trust weight in [0, 1] relative to the
// verifier that computed it, given their seed set.
type Weights map[string]float64

// FetchTrustEdges queries relayURL for attestation events authored by
// attesterPubKey (trust edges are looked up outward from who vouched,
// not inward by who's being vouched for, unlike content-claim
// lookups), then filters to trust edges client-side. The relay-side
// filter only constrains on Kind/Authors, not the claim_type tag:
// NIP-01 only guarantees single-letter tags (like "p") are indexed
// and queryable — "claim_type" is a multi-character tag most public
// relays won't filter on server-side, so results would silently come
// back empty on relays that don't bother indexing it.
func FetchTrustEdges(ctx context.Context, relayURL, attesterPubKey string) ([]*nostr.Event, error) {
	relay, err := nostr.RelayConnect(ctx, relayURL)
	if err != nil {
		return nil, fmt.Errorf("trust: connect %s: %w", relayURL, err)
	}
	defer relay.Close()

	events, err := relay.QuerySync(ctx, nostr.Filter{
		Kinds:   []int{attestation.Kind},
		Authors: []string{attesterPubKey},
	})
	if err != nil {
		return nil, fmt.Errorf("trust: query %s: %w", relayURL, err)
	}
	return events, nil
}

// trustedTo returns the pubkeys attesterPubKey has issued a trust
// edge to, by verifying and filtering the events FetchTrustEdges
// returned.
func trustedTo(events []*nostr.Event) []string {
	var out []string
	for _, evt := range events {
		claim, err := attestation.Verify(evt)
		if err != nil || claim.ClaimType != EdgeClaimType {
			continue
		}
		out = append(out, claim.SubjectKey)
	}
	return out
}

// EdgeFetcher returns the pubkeys a given pubkey has issued a trust
// edge to. RelayEdgeFetcher is the real, network-backed
// implementation; Compute takes an EdgeFetcher so the graph-walk
// logic itself can be tested offline against a synthetic graph.
type EdgeFetcher func(ctx context.Context, pubkey string) ([]string, error)

// RelayEdgeFetcher returns an EdgeFetcher backed by a real Nostr
// relay at relayURL.
func RelayEdgeFetcher(relayURL string) EdgeFetcher {
	return func(ctx context.Context, pubkey string) ([]string, error) {
		edges, err := FetchTrustEdges(ctx, relayURL, pubkey)
		if err != nil {
			return nil, err
		}
		return trustedTo(edges), nil
	}
}

// Compute walks the trust graph outward from seed (the verifier's own
// directly-trusted pubkeys, each starting at weight 1.0), using fetch
// to look up each pubkey's outgoing trust edges, up to MaxDepth hops,
// halving weight per hop. When a pubkey is reachable by more than one
// path, it keeps the best (highest-weight) path rather than summing
// them — summing would let an attacker inflate weight just by
// collecting many low-weight paths, the same gameable-aggregate
// failure a global score would be.
func Compute(ctx context.Context, fetch EdgeFetcher, seed []string) (Weights, error) {
	weights := make(Weights, len(seed))
	type frontierEntry struct {
		pubkey string
		weight float64
		depth  int
	}

	var frontier []frontierEntry
	for _, pk := range seed {
		if w, ok := weights[pk]; !ok || directWeight > w {
			weights[pk] = directWeight
		}
		frontier = append(frontier, frontierEntry{pk, directWeight, 1})
	}

	for len(frontier) > 0 {
		var next []frontierEntry
		for _, entry := range frontier {
			if entry.depth >= MaxDepth {
				continue
			}
			children, err := fetch(ctx, entry.pubkey)
			if err != nil {
				return nil, err
			}
			childWeight := entry.weight / 2
			for _, child := range children {
				if existing, ok := weights[child]; ok && existing >= childWeight {
					continue
				}
				weights[child] = childWeight
				next = append(next, frontierEntry{child, childWeight, entry.depth + 1})
			}
		}
		frontier = next
	}

	return weights, nil
}

// ScoredClaim pairs a verified attestation.Claim with the trust
// weight of the pubkey that made it, relative to the verifier's own
// Weights.
type ScoredClaim struct {
	attestation.Claim
	Weight float64
}

// Score weights subject's fetched attestations by the verifier's
// computed trust Weights. A claim from an attester outside the
// reachable set scores 0 and is excluded from the returned slice.
// Claims are grouped by ClaimType, and within each group ordered by
// descending weight — the caller can then take index 0 of a group as
// "the subject's best-supported claim for X."
func Score(events []*nostr.Event, w Weights) map[string][]ScoredClaim {
	byType := make(map[string][]ScoredClaim)
	for _, evt := range events {
		claim, err := attestation.Verify(evt)
		if err != nil {
			continue
		}
		weight, ok := w[claim.AttesterKey]
		if !ok || weight <= 0 {
			continue
		}
		byType[claim.ClaimType] = append(byType[claim.ClaimType], ScoredClaim{Claim: claim, Weight: weight})
	}
	for claimType, scored := range byType {
		sortByWeightDesc(scored)
		byType[claimType] = scored
	}
	return byType
}

func sortByWeightDesc(s []ScoredClaim) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j].Weight > s[j-1].Weight; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
