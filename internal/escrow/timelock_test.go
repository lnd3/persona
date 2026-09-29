package escrow

import (
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
)

func TestSequenceForDaysRoundsUpAndSetsTimeFlag(t *testing.T) {
	seq, err := SequenceForDays(14)
	if err != nil {
		t.Fatalf("SequenceForDays: %v", err)
	}
	if seq&csvTimeFlag == 0 {
		t.Error("expected the BIP68 time-flag bit to be set")
	}
	const totalSeconds = 14 * 86400
	wantUnits := uint32((totalSeconds + secondsPerUnit - 1) / secondsPerUnit) // rounds up
	if seq&0xFFFF != wantUnits {
		t.Errorf("units = %d, want %d", seq&0xFFFF, wantUnits)
	}
}

func TestSequenceForDaysRoundsUpPartialUnit(t *testing.T) {
	// 1 day = 168.75 units of 512s; must round up so the enforced
	// window is never shorter than requested.
	seq, err := SequenceForDays(1)
	if err != nil {
		t.Fatalf("SequenceForDays: %v", err)
	}
	units := seq & 0xFFFF
	enforcedSeconds := int64(units) * secondsPerUnit
	if enforcedSeconds < 86400 {
		t.Errorf("enforced window %ds is shorter than the requested 1 day", enforcedSeconds)
	}
}

func TestSequenceForDaysRejectsNonPositive(t *testing.T) {
	if _, err := SequenceForDays(0); err == nil {
		t.Error("expected error for window_days = 0")
	}
	if _, err := SequenceForDays(-1); err == nil {
		t.Error("expected error for negative window_days")
	}
}

func TestSequenceForDaysRejectsTooLong(t *testing.T) {
	if _, err := SequenceForDays(400); err == nil {
		t.Error("expected error for a window exceeding BIP68's ~389-day ceiling")
	}
}

func TestFundingAddressRegtest(t *testing.T) {
	_, pubKey := newTestKey(t)
	seq, err := SequenceForDays(14)
	if err != nil {
		t.Fatalf("SequenceForDays: %v", err)
	}
	witnessScript, err := PreEscalationScript(pubKey, seq)
	if err != nil {
		t.Fatalf("PreEscalationScript: %v", err)
	}
	addr, err := FundingAddress(witnessScript, &chaincfg.RegressionNetParams)
	if err != nil {
		t.Fatalf("FundingAddress: %v", err)
	}
	if addr.String() == "" {
		t.Error("expected a non-empty regtest funding address")
	}
	if !addr.IsForNet(&chaincfg.RegressionNetParams) {
		t.Error("expected address to be valid for regtest params")
	}
}

func TestPreEscalationScriptRejectsInvalidPubKey(t *testing.T) {
	seq, err := SequenceForDays(14)
	if err != nil {
		t.Fatalf("SequenceForDays: %v", err)
	}
	if _, err := PreEscalationScript([]byte{0x02, 0x03}, seq); err == nil {
		t.Error("expected error for a malformed (non-33-byte) pubkey")
	}
}
