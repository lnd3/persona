package escrow

import (
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/wire"
)

// testWitnessScripts builds a real UniversalScript and ReinforcedScript
// with fresh keys, for exercising fee estimation against real script
// shapes rather than arbitrary byte slices.
func testWitnessScripts(t *testing.T) (universal, reinforced []byte) {
	t.Helper()
	attesterPriv, _ := btcec.NewPrivateKey()
	subjectPriv, _ := btcec.NewPrivateKey()
	arbiterPriv, _ := btcec.NewPrivateKey()
	attesterArbiterPriv, _ := btcec.NewPrivateKey()
	subjectArbiterPriv, _ := btcec.NewPrivateKey()
	attesterPub := attesterPriv.PubKey().SerializeCompressed()
	subjectPub := subjectPriv.PubKey().SerializeCompressed()
	arbiterPub := arbiterPriv.PubKey().SerializeCompressed()
	attesterArbiterPub := attesterArbiterPriv.PubKey().SerializeCompressed()
	subjectArbiterPub := subjectArbiterPriv.PubKey().SerializeCompressed()

	seq, err := SequenceForDays(14)
	if err != nil {
		t.Fatalf("SequenceForDays: %v", err)
	}
	uni, err := UniversalScript(attesterPub, subjectPub, arbiterPub, seq)
	if err != nil {
		t.Fatalf("UniversalScript: %v", err)
	}
	rein, err := ReinforcedScript(attesterPub, subjectPub, attesterArbiterPub, subjectArbiterPub, seq)
	if err != nil {
		t.Fatalf("ReinforcedScript: %v", err)
	}
	return uni, rein
}

// TestEstimateSettlementVSizeMatchesMeasuredValues pins this package's
// own fee estimation against the exact real numbers measured while
// scoping A006's mainnet gate (2026-09-30, recorded in that action's
// own plan file) — a regression here means either the scripts or the
// estimation logic changed in a way that silently invalidates those
// real measurements.
func TestEstimateSettlementVSizeMatchesMeasuredValues(t *testing.T) {
	uni, rein := testWitnessScripts(t)
	payouts := []*wire.TxOut{{Value: 99_999_000, PkScript: make([]byte, 34)}}

	cases := []struct {
		name          string
		witnessScript []byte
		branch        Branch
		wantVSize     int64
	}{
		{"Universal mutual settlement / arbiter-2of2", uni, BranchMutualSettlement, 199},
		{"Universal fallback", uni, BranchUniversalFallback, 181},
		{"Reinforced 2of2 branch", rein, BranchReinforcedBothArbitersAgree, 218},
		// 199, not the 200 originally quoted while scoping this: that
		// ad-hoc measurement used {0x01} as a placeholder for every
		// selector, including the three selFalse ones, which are
		// actually empty ([]byte{}) — one byte smaller each in the
		// witness than the scoping pass assumed. This implementation
		// uses the real selFalse/selTrue values (see Branch's own
		// vars), so this is the corrected, accurate number.
		{"Reinforced fallback", rein, BranchReinforcedFallback, 199},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			vsize, err := EstimateSettlementVSize(c.witnessScript, payouts, 0, c.branch)
			if err != nil {
				t.Fatalf("EstimateSettlementVSize: %v", err)
			}
			if vsize != c.wantVSize {
				t.Errorf("vsize = %d, want %d (measured value from A006's own scoping)", vsize, c.wantVSize)
			}
		})
	}
}

// TestEstimateSettlementVSizeAllBranchesPositive is a broader sanity
// check across every named branch (not just the four cross-checked
// against hand-measured numbers above), confirming each produces a
// sane, strictly positive vsize with no panics.
func TestEstimateSettlementVSizeAllBranchesPositive(t *testing.T) {
	uni, rein := testWitnessScripts(t)
	payouts := []*wire.TxOut{{Value: 50_000, PkScript: make([]byte, 34)}}

	branches := map[string]struct {
		script []byte
		branch Branch
	}{
		"universal/mutual":          {uni, BranchMutualSettlement},
		"universal/arbiter-sides-a": {uni, BranchUniversalArbiterSidesAttester},
		"universal/arbiter-sides-s": {uni, BranchUniversalArbiterSidesSubject},
		"universal/fallback":        {uni, BranchUniversalFallback},
		"reinforced/mutual":         {rein, BranchMutualSettlement},
		"reinforced/both-arbiters":  {rein, BranchReinforcedBothArbitersAgree},
		"reinforced/attester-own":   {rein, BranchReinforcedAttesterOwnArbiter},
		"reinforced/subject-own":    {rein, BranchReinforcedSubjectOwnArbiter},
		"reinforced/fallback":       {rein, BranchReinforcedFallback},
	}
	for name, b := range branches {
		t.Run(name, func(t *testing.T) {
			vsize, err := EstimateSettlementVSize(b.script, payouts, 0, b.branch)
			if err != nil {
				t.Fatalf("EstimateSettlementVSize: %v", err)
			}
			if vsize <= 0 {
				t.Errorf("vsize = %d, want > 0", vsize)
			}
		})
	}
}

func TestEstimateSettlementVSizeRejectsNoPayouts(t *testing.T) {
	uni, _ := testWitnessScripts(t)
	if _, err := EstimateSettlementVSize(uni, nil, 0, BranchMutualSettlement); err == nil {
		t.Error("EstimateSettlementVSize with no payouts: expected an error, got nil")
	}
}

func TestEstimateSettlementFeeScalesWithRate(t *testing.T) {
	uni, _ := testWitnessScripts(t)
	payouts := []*wire.TxOut{{Value: 50_000, PkScript: make([]byte, 34)}}

	vsize, fee, err := EstimateSettlementFee(uni, payouts, 0, BranchMutualSettlement, 10)
	if err != nil {
		t.Fatalf("EstimateSettlementFee: %v", err)
	}
	if fee != vsize*10 {
		t.Errorf("fee = %d, want vsize(%d)*10 = %d", fee, vsize, vsize*10)
	}

	_, feeAtZero, err := EstimateSettlementFee(uni, payouts, 0, BranchMutualSettlement, 0)
	if err != nil {
		t.Fatalf("EstimateSettlementFee at feerate 0: %v", err)
	}
	if feeAtZero != 0 {
		t.Errorf("fee at feerate 0 = %d, want 0", feeAtZero)
	}
}

// TestIsDustOutput covers the real boundary cases: a P2WPKH-shaped
// output (what this package's own regtest helpers actually pay to)
// just above and below its own threshold, plus the unconditional
// "unspendable is always dust" rule regardless of value.
func TestIsDustOutput(t *testing.T) {
	p2wpkh := make([]byte, 22) // OP_0 <20-byte-hash> shape, length is what matters for IsWitnessProgram
	p2wpkh[0] = 0x00
	p2wpkh[1] = 0x14 // push 20 bytes

	threshold := dustThreshold(&wire.TxOut{PkScript: p2wpkh}, 1)

	if !IsDustOutput(&wire.TxOut{Value: threshold - 1, PkScript: p2wpkh}, 1) {
		t.Errorf("value %d (threshold-1): want dust", threshold-1)
	}
	if IsDustOutput(&wire.TxOut{Value: threshold, PkScript: p2wpkh}, 1) {
		t.Errorf("value %d (== threshold): want not dust", threshold)
	}

	opReturn := []byte{0x6a, 0x04, 't', 'e', 's', 't'} // OP_RETURN push
	if !IsDustOutput(&wire.TxOut{Value: 1_000_000, PkScript: opReturn}, 1) {
		t.Error("OP_RETURN output with a large value: want dust (unspendable), got not dust")
	}
}

func TestIsDustOutputScalesWithFeeRate(t *testing.T) {
	p2wpkh := make([]byte, 22)
	p2wpkh[1] = 0x14
	out := &wire.TxOut{Value: 500, PkScript: p2wpkh}
	if IsDustOutput(out, 1) {
		t.Fatalf("precondition failed: 500 sats should not be dust at 1 sat/vByte")
	}
	if !IsDustOutput(out, 10) {
		t.Error("500 sats at 10x the feerate: want dust, got not dust")
	}
}

// TestNewSettlementPacketRejectsDustPayout is the real gap this file
// closes, not just the standalone helper: NewSettlementPacket itself
// must refuse to build a transaction any real node's mempool policy
// would reject at broadcast time.
func TestNewSettlementPacketRejectsDustPayout(t *testing.T) {
	uni, _ := testWitnessScripts(t)
	prevOut := wire.OutPoint{}
	prevTxOut := &wire.TxOut{Value: 100_000, PkScript: make([]byte, 34)}
	dustPayout := []*wire.TxOut{{Value: 1, PkScript: make([]byte, 22)}}

	if _, err := NewSettlementPacket(prevOut, prevTxOut, uni, dustPayout, 0); err == nil {
		t.Error("NewSettlementPacket with a 1-sat payout: expected an error, got nil")
	}

	realPayout := []*wire.TxOut{{Value: 99_000, PkScript: make([]byte, 34)}}
	if _, err := NewSettlementPacket(prevOut, prevTxOut, uni, realPayout, 0); err != nil {
		t.Errorf("NewSettlementPacket with a real payout: %v", err)
	}
}
