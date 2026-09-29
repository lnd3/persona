package escrow

import (
	"testing"

	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

// These tests exercise the full PSBT round trip — build, sign
// (independently, as if by separate parties), finalize, extract —
// and then re-validate the extracted transaction against the real
// txscript engine, exactly as a relay/miner would.

func settlementPrevOut(scriptPubKey []byte) (wire.OutPoint, *wire.TxOut) {
	return wire.OutPoint{Index: 0}, &wire.TxOut{Value: testAmount, PkScript: scriptPubKey}
}

func payout() []*wire.TxOut {
	return []*wire.TxOut{{Value: testAmount - 1000, PkScript: []byte{txscript.OP_TRUE}}}
}

func TestSettlement_UniversalMutualSettlement(t *testing.T) {
	attester, attesterPub := newTestKey(t)
	subject, subjectPub := newTestKey(t)
	_, arbiterPub := newTestKey(t)
	sequence, err := SequenceForDays(14)
	if err != nil {
		t.Fatalf("SequenceForDays: %v", err)
	}

	witnessScript, err := UniversalScript(attesterPub, subjectPub, arbiterPub, sequence)
	if err != nil {
		t.Fatalf("UniversalScript: %v", err)
	}
	scriptPubKey, err := WitnessScriptHash(witnessScript)
	if err != nil {
		t.Fatalf("WitnessScriptHash: %v", err)
	}
	prevOut, prevTxOut := settlementPrevOut(scriptPubKey)

	p, err := NewSettlementPacket(prevOut, prevTxOut, witnessScript, payout(), 0)
	if err != nil {
		t.Fatalf("NewSettlementPacket: %v", err)
	}

	// Attester and subject sign independently, as separate parties
	// would in practice — order doesn't matter.
	if err := SignSettlementInput(p, subject); err != nil {
		t.Fatalf("SignSettlementInput(subject): %v", err)
	}
	if err := SignSettlementInput(p, attester); err != nil {
		t.Fatalf("SignSettlementInput(attester): %v", err)
	}

	if err := FinalizeUniversal(p, witnessScript, attesterPub, subjectPub, arbiterPub, sequence); err != nil {
		t.Fatalf("FinalizeUniversal: %v", err)
	}

	tx, err := ExtractSettlement(p)
	if err != nil {
		t.Fatalf("ExtractSettlement: %v", err)
	}

	if err := execute(t, scriptPubKey, tx); err != nil {
		t.Errorf("extracted mutual-settlement tx failed real script validation: %v", err)
	}
}

func TestSettlement_UniversalArbiterSidesWithSubject(t *testing.T) {
	_, attesterPub := newTestKey(t)
	subject, subjectPub := newTestKey(t)
	arbiter, arbiterPub := newTestKey(t)
	sequence, err := SequenceForDays(14)
	if err != nil {
		t.Fatalf("SequenceForDays: %v", err)
	}

	witnessScript, err := UniversalScript(attesterPub, subjectPub, arbiterPub, sequence)
	if err != nil {
		t.Fatalf("UniversalScript: %v", err)
	}
	scriptPubKey, err := WitnessScriptHash(witnessScript)
	if err != nil {
		t.Fatalf("WitnessScriptHash: %v", err)
	}
	prevOut, prevTxOut := settlementPrevOut(scriptPubKey)

	p, err := NewSettlementPacket(prevOut, prevTxOut, witnessScript, payout(), 0)
	if err != nil {
		t.Fatalf("NewSettlementPacket: %v", err)
	}
	if err := SignSettlementInput(p, arbiter); err != nil {
		t.Fatalf("SignSettlementInput(arbiter): %v", err)
	}
	if err := SignSettlementInput(p, subject); err != nil {
		t.Fatalf("SignSettlementInput(subject): %v", err)
	}

	if err := FinalizeUniversal(p, witnessScript, attesterPub, subjectPub, arbiterPub, sequence); err != nil {
		t.Fatalf("FinalizeUniversal: %v", err)
	}
	tx, err := ExtractSettlement(p)
	if err != nil {
		t.Fatalf("ExtractSettlement: %v", err)
	}
	if err := execute(t, scriptPubKey, tx); err != nil {
		t.Errorf("extracted arbiter-sides-with-subject tx failed real script validation: %v", err)
	}
}

func TestSettlement_UniversalFallbackAfterWindow(t *testing.T) {
	attester, attesterPub := newTestKey(t)
	_, subjectPub := newTestKey(t)
	_, arbiterPub := newTestKey(t)
	sequence, err := SequenceForDays(14)
	if err != nil {
		t.Fatalf("SequenceForDays: %v", err)
	}

	witnessScript, err := UniversalScript(attesterPub, subjectPub, arbiterPub, sequence)
	if err != nil {
		t.Fatalf("UniversalScript: %v", err)
	}
	scriptPubKey, err := WitnessScriptHash(witnessScript)
	if err != nil {
		t.Fatalf("WitnessScriptHash: %v", err)
	}
	prevOut, prevTxOut := settlementPrevOut(scriptPubKey)

	p, err := NewSettlementPacket(prevOut, prevTxOut, witnessScript, payout(), sequence)
	if err != nil {
		t.Fatalf("NewSettlementPacket: %v", err)
	}
	if err := SignSettlementInput(p, attester); err != nil {
		t.Fatalf("SignSettlementInput(attester): %v", err)
	}

	if err := FinalizeUniversal(p, witnessScript, attesterPub, subjectPub, arbiterPub, sequence); err != nil {
		t.Fatalf("FinalizeUniversal: %v", err)
	}
	tx, err := ExtractSettlement(p)
	if err != nil {
		t.Fatalf("ExtractSettlement: %v", err)
	}
	if err := execute(t, scriptPubKey, tx); err != nil {
		t.Errorf("extracted fallback tx failed real script validation: %v", err)
	}
}

func TestSettlement_NotFinalizableWithInsufficientSignatures(t *testing.T) {
	attester, attesterPub := newTestKey(t)
	_, subjectPub := newTestKey(t)
	_, arbiterPub := newTestKey(t)
	sequence, err := SequenceForDays(14)
	if err != nil {
		t.Fatalf("SequenceForDays: %v", err)
	}
	witnessScript, err := UniversalScript(attesterPub, subjectPub, arbiterPub, sequence)
	if err != nil {
		t.Fatalf("UniversalScript: %v", err)
	}
	scriptPubKey, err := WitnessScriptHash(witnessScript)
	if err != nil {
		t.Fatalf("WitnessScriptHash: %v", err)
	}
	prevOut, prevTxOut := settlementPrevOut(scriptPubKey)

	p, err := NewSettlementPacket(prevOut, prevTxOut, witnessScript, payout(), 0)
	if err != nil {
		t.Fatalf("NewSettlementPacket: %v", err)
	}
	// Only attester signs — not enough for any branch (fallback needs
	// sequence >= the window, which this packet doesn't have).
	if err := SignSettlementInput(p, attester); err != nil {
		t.Fatalf("SignSettlementInput(attester): %v", err)
	}

	if err := FinalizeUniversal(p, witnessScript, attesterPub, subjectPub, arbiterPub, sequence); err == nil {
		t.Error("expected FinalizeUniversal to fail with only one signature and sequence=0")
	}
}

func TestSettlement_ReinforcedBothArbitersAgree(t *testing.T) {
	_, attesterPub := newTestKey(t)
	_, subjectPub := newTestKey(t)
	arbiterA, arbiterAPub := newTestKey(t)
	arbiterB, arbiterBPub := newTestKey(t)
	sequence, err := SequenceForDays(14)
	if err != nil {
		t.Fatalf("SequenceForDays: %v", err)
	}

	witnessScript, err := ReinforcedScript(attesterPub, subjectPub, arbiterAPub, arbiterBPub, sequence)
	if err != nil {
		t.Fatalf("ReinforcedScript: %v", err)
	}
	scriptPubKey, err := WitnessScriptHash(witnessScript)
	if err != nil {
		t.Fatalf("WitnessScriptHash: %v", err)
	}
	prevOut, prevTxOut := settlementPrevOut(scriptPubKey)

	p, err := NewSettlementPacket(prevOut, prevTxOut, witnessScript, payout(), 0)
	if err != nil {
		t.Fatalf("NewSettlementPacket: %v", err)
	}
	if err := SignSettlementInput(p, arbiterB); err != nil {
		t.Fatalf("SignSettlementInput(arbiterB): %v", err)
	}
	if err := SignSettlementInput(p, arbiterA); err != nil {
		t.Fatalf("SignSettlementInput(arbiterA): %v", err)
	}

	if err := FinalizeReinforced(p, witnessScript, attesterPub, subjectPub, arbiterAPub, arbiterBPub, sequence); err != nil {
		t.Fatalf("FinalizeReinforced: %v", err)
	}
	tx, err := ExtractSettlement(p)
	if err != nil {
		t.Fatalf("ExtractSettlement: %v", err)
	}
	if err := execute(t, scriptPubKey, tx); err != nil {
		t.Errorf("extracted both-arbiters-agree tx failed real script validation: %v", err)
	}
}
