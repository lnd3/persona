package trust

import (
	"context"
	"math"
	"net"
	"testing"
	"time"

	"github.com/nbd-wtf/go-nostr"

	"github.com/lnd3/persona/internal/attestation"
	"github.com/lnd3/persona/internal/identity"
)

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

// graphFetcher builds an EdgeFetcher over a static adjacency map, so
// the graph-walk algorithm can be tested without a live relay.
func graphFetcher(graph map[string][]string) EdgeFetcher {
	return func(_ context.Context, pubkey string) ([]string, error) {
		return graph[pubkey], nil
	}
}

func TestComputeDirectTrust(t *testing.T) {
	w, err := Compute(context.Background(), graphFetcher(nil), []string{"a", "b"})
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if !almostEqual(w["a"], 1.0) || !almostEqual(w["b"], 1.0) {
		t.Errorf("direct trust = %v, want 1.0 for both seeds", w)
	}
}

func TestComputeTransitiveDecay(t *testing.T) {
	graph := map[string][]string{
		"a": {"b"},
		"b": {"c"},
		"c": {"d"},
	}
	w, err := Compute(context.Background(), graphFetcher(graph), []string{"a"})
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if !almostEqual(w["a"], 1.0) {
		t.Errorf("depth 1 (seed) = %v, want 1.0", w["a"])
	}
	if !almostEqual(w["b"], 0.5) {
		t.Errorf("depth 2 = %v, want 0.5", w["b"])
	}
	if !almostEqual(w["c"], 0.25) {
		t.Errorf("depth 3 = %v, want 0.25", w["c"])
	}
	if _, ok := w["d"]; ok {
		t.Errorf("depth 4 (%q) should be unreachable at MaxDepth=%d, got weight %v", "d", MaxDepth, w["d"])
	}
}

func TestComputeCycleDoesNotLoopOrInflate(t *testing.T) {
	graph := map[string][]string{
		"a": {"b"},
		"b": {"c"},
		"c": {"a"}, // cycle back to the seed
	}
	done := make(chan struct{})
	var w Weights
	var err error
	go func() {
		w, err = Compute(context.Background(), graphFetcher(graph), []string{"a"})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Compute did not terminate on a cyclic graph")
	}
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if !almostEqual(w["a"], 1.0) {
		t.Errorf("cycle inflated seed weight: a = %v, want 1.0", w["a"])
	}
}

func TestComputeOutsideSeedSetUnreachable(t *testing.T) {
	graph := map[string][]string{
		"a": {"b"},
	}
	w, err := Compute(context.Background(), graphFetcher(graph), []string{"a"})
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if _, ok := w["stranger"]; ok {
		t.Errorf("unrelated pubkey should not appear in weights, got %v", w["stranger"])
	}
}

func TestComputeMultiPathUsesBestNotSum(t *testing.T) {
	// "target" is reachable directly from seed "a" (weight 0.5) and
	// also via a longer path a->mid->target (weight 0.25). The best
	// path (0.5) must win, not 0.5+0.25.
	graph := map[string][]string{
		"a":   {"target", "mid"},
		"mid": {"target"},
	}
	w, err := Compute(context.Background(), graphFetcher(graph), []string{"a"})
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if !almostEqual(w["target"], 0.5) {
		t.Errorf("target weight = %v, want 0.5 (best path, not summed)", w["target"])
	}
}

func TestScoreExcludesUnreachableAttester(t *testing.T) {
	trusted, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	stranger, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	subject, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}

	trustedEvt, err := attestation.New(trusted, subject.PublicKey, "net.persona.core.skill", "go", "vouches for go skill")
	if err != nil {
		t.Fatalf("attestation.New: %v", err)
	}
	strangerEvt, err := attestation.New(stranger, subject.PublicKey, "net.persona.core.skill", "go", "vouches for go skill")
	if err != nil {
		t.Fatalf("attestation.New: %v", err)
	}

	w := Weights{trusted.PublicKey: 1.0}
	scored := Score([]*nostr.Event{trustedEvt, strangerEvt}, w)

	claims := scored["net.persona.core.skill"]
	if len(claims) != 1 {
		t.Fatalf("got %d scored claims, want 1 (stranger excluded)", len(claims))
	}
	if claims[0].AttesterKey != trusted.PublicKey {
		t.Errorf("scored claim attester = %s, want %s", claims[0].AttesterKey, trusted.PublicKey)
	}
	if !almostEqual(claims[0].Weight, 1.0) {
		t.Errorf("scored claim weight = %v, want 1.0", claims[0].Weight)
	}
}

func TestScoreOrdersByWeightDescending(t *testing.T) {
	high, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	low, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	subject, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}

	highEvt, err := attestation.New(high, subject.PublicKey, "net.persona.core.skill", "go", "")
	if err != nil {
		t.Fatalf("attestation.New: %v", err)
	}
	lowEvt, err := attestation.New(low, subject.PublicKey, "net.persona.core.skill", "go", "")
	if err != nil {
		t.Fatalf("attestation.New: %v", err)
	}

	w := Weights{high.PublicKey: 1.0, low.PublicKey: 0.25}
	scored := Score([]*nostr.Event{lowEvt, highEvt}, w)["net.persona.core.skill"]
	if len(scored) != 2 {
		t.Fatalf("got %d scored claims, want 2", len(scored))
	}
	if scored[0].AttesterKey != high.PublicKey || scored[1].AttesterKey != low.PublicKey {
		t.Errorf("scored claims not ordered by descending weight: %+v", scored)
	}
}

// relayReachable probes a relay URL's host:port before running a live
// network test, so this test skips gracefully offline instead of
// failing, per A001's own live-test pattern.
func relayReachable(hostPort string) bool {
	conn, err := net.DialTimeout("tcp", hostPort, 3*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func TestFetchTrustEdgesRoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live relay test in -short mode")
	}
	const relayURL = "wss://nos.lol"
	if !relayReachable("nos.lol:443") {
		t.Skip("relay unreachable, skipping live test")
	}

	attester, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	trustee, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}

	evt, err := attestation.New(attester, trustee.PublicKey, EdgeClaimType, "", "trust vouch")
	if err != nil {
		t.Fatalf("attestation.New: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := attestation.Publish(ctx, evt, []string{relayURL}); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	// Relays may take a moment to index; retry briefly.
	var edges []string
	fetch := RelayEdgeFetcher(relayURL)
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		edges, err = fetch(ctx, attester.PublicKey)
		if err != nil {
			t.Fatalf("RelayEdgeFetcher: %v", err)
		}
		if len(edges) > 0 {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	found := false
	for _, e := range edges {
		if e == trustee.PublicKey {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected trustee %s among fetched trust edges, got %v", trustee.PublicKey, edges)
	}
}
