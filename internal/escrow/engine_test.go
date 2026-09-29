package escrow

import (
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

// This file validates scripts by actually executing them against
// btcd's real consensus script engine (the same one btcd itself uses
// to test its own opcodes), rather than only checking that
// script-building code runs without error. No live Bitcoin node is
// needed for this — CSV's relative-locktime check compares the
// script's required sequence directly against the spending input's
// own declared nSequence value, which a test can just set directly,
// without needing to simulate the actual passage of chain time (see
// plan/actions/A006's Log on why no live regtest node is used here).

const testAmount = int64(100_000) // sats, arbitrary for test purposes

func newTestKey(t *testing.T) (*btcec.PrivateKey, []byte) {
	t.Helper()
	priv, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("btcec.NewPrivateKey: %v", err)
	}
	return priv, priv.PubKey().SerializeCompressed()
}

// buildSpendingTx constructs a minimal version-2 transaction spending
// a single fake previous output, with the given input sequence.
func buildSpendingTx(sequence uint32) *wire.MsgTx {
	tx := wire.NewMsgTx(2)
	tx.AddTxIn(&wire.TxIn{
		PreviousOutPoint: wire.OutPoint{Index: 0},
		Sequence:         sequence,
	})
	tx.AddTxOut(&wire.TxOut{Value: testAmount - 1000, PkScript: []byte{txscript.OP_TRUE}})
	return tx
}

// execute runs the witness-satisfying tx against scriptPubKey and
// returns the engine's verdict.
func execute(t *testing.T, scriptPubKey []byte, tx *wire.MsgTx) error {
	t.Helper()
	fetcher := txscript.NewCannedPrevOutputFetcher(scriptPubKey, testAmount)
	sigHashes := txscript.NewTxSigHashes(tx, fetcher)
	vm, err := txscript.NewEngine(scriptPubKey, tx, 0, txscript.StandardVerifyFlags, nil, sigHashes, testAmount, fetcher)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	return vm.Execute()
}

func TestPreEscalationScript_OwnerSpendsAfterWindow(t *testing.T) {
	owner, ownerPub := newTestKey(t)
	sequence, err := SequenceForDays(14)
	if err != nil {
		t.Fatalf("SequenceForDays: %v", err)
	}
	witnessScript, err := PreEscalationScript(ownerPub, sequence)
	if err != nil {
		t.Fatalf("PreEscalationScript: %v", err)
	}
	scriptPubKey, err := WitnessScriptHash(witnessScript)
	if err != nil {
		t.Fatalf("WitnessScriptHash: %v", err)
	}

	tx := buildSpendingTx(sequence) // exactly the required sequence: should succeed
	fetcher := txscript.NewCannedPrevOutputFetcher(scriptPubKey, testAmount)
	sigHashes := txscript.NewTxSigHashes(tx, fetcher)
	sig, err := txscript.RawTxInWitnessSignature(tx, sigHashes, 0, testAmount, witnessScript, txscript.SigHashAll, owner)
	if err != nil {
		t.Fatalf("RawTxInWitnessSignature: %v", err)
	}
	tx.TxIn[0].Witness = wire.TxWitness{sig, witnessScript}

	if err := execute(t, scriptPubKey, tx); err != nil {
		t.Errorf("expected owner spend to succeed after the window, got: %v", err)
	}
}

func TestPreEscalationScript_FailsBeforeWindow(t *testing.T) {
	owner, ownerPub := newTestKey(t)
	sequence, err := SequenceForDays(14)
	if err != nil {
		t.Fatalf("SequenceForDays: %v", err)
	}
	witnessScript, err := PreEscalationScript(ownerPub, sequence)
	if err != nil {
		t.Fatalf("PreEscalationScript: %v", err)
	}
	scriptPubKey, err := WitnessScriptHash(witnessScript)
	if err != nil {
		t.Fatalf("WitnessScriptHash: %v", err)
	}

	tx := buildSpendingTx(sequence - 1) // one unit short of the required window
	fetcher := txscript.NewCannedPrevOutputFetcher(scriptPubKey, testAmount)
	sigHashes := txscript.NewTxSigHashes(tx, fetcher)
	sig, err := txscript.RawTxInWitnessSignature(tx, sigHashes, 0, testAmount, witnessScript, txscript.SigHashAll, owner)
	if err != nil {
		t.Fatalf("RawTxInWitnessSignature: %v", err)
	}
	tx.TxIn[0].Witness = wire.TxWitness{sig, witnessScript}

	if err := execute(t, scriptPubKey, tx); err == nil {
		t.Error("expected owner spend to fail before the window has elapsed")
	}
}

func TestPreEscalationScript_WrongSignerFails(t *testing.T) {
	_, ownerPub := newTestKey(t)
	stranger, _ := newTestKey(t)
	sequence, err := SequenceForDays(14)
	if err != nil {
		t.Fatalf("SequenceForDays: %v", err)
	}
	witnessScript, err := PreEscalationScript(ownerPub, sequence)
	if err != nil {
		t.Fatalf("PreEscalationScript: %v", err)
	}
	scriptPubKey, err := WitnessScriptHash(witnessScript)
	if err != nil {
		t.Fatalf("WitnessScriptHash: %v", err)
	}

	tx := buildSpendingTx(sequence)
	fetcher := txscript.NewCannedPrevOutputFetcher(scriptPubKey, testAmount)
	sigHashes := txscript.NewTxSigHashes(tx, fetcher)
	sig, err := txscript.RawTxInWitnessSignature(tx, sigHashes, 0, testAmount, witnessScript, txscript.SigHashAll, stranger)
	if err != nil {
		t.Fatalf("RawTxInWitnessSignature: %v", err)
	}
	tx.TxIn[0].Witness = wire.TxWitness{sig, witnessScript}

	if err := execute(t, scriptPubKey, tx); err == nil {
		t.Error("expected spend signed by a non-owner key to fail")
	}
}

// multisigWitness signs tx input 0 with each of keys (in the same
// order their pubkeys appear in witnessScript) and returns the
// witness stack for a CHECKMULTISIG spend: a leading empty dummy
// (the historic off-by-one bug preserved by consensus), then the
// signatures in pubkey order, then the witness script itself.
func multisigWitness(t *testing.T, tx *wire.MsgTx, sigHashes *txscript.TxSigHashes, witnessScript []byte, keys ...*btcec.PrivateKey) wire.TxWitness {
	t.Helper()
	w := wire.TxWitness{{}} // dummy
	for _, key := range keys {
		sig, err := txscript.RawTxInWitnessSignature(tx, sigHashes, 0, testAmount, witnessScript, txscript.SigHashAll, key)
		if err != nil {
			t.Fatalf("RawTxInWitnessSignature: %v", err)
		}
		w = append(w, sig)
	}
	w = append(w, witnessScript)
	return w
}

func TestLargeTierEscalatedScript_BothArbitersRequired(t *testing.T) {
	arbiterA, arbiterAPub := newTestKey(t)
	arbiterB, arbiterBPub := newTestKey(t)

	witnessScript, err := LargeTierEscalatedScript(arbiterAPub, arbiterBPub)
	if err != nil {
		t.Fatalf("LargeTierEscalatedScript: %v", err)
	}
	scriptPubKey, err := WitnessScriptHash(witnessScript)
	if err != nil {
		t.Fatalf("WitnessScriptHash: %v", err)
	}

	t.Run("both sign: succeeds", func(t *testing.T) {
		tx := buildSpendingTx(0)
		fetcher := txscript.NewCannedPrevOutputFetcher(scriptPubKey, testAmount)
		sigHashes := txscript.NewTxSigHashes(tx, fetcher)
		tx.TxIn[0].Witness = multisigWitness(t, tx, sigHashes, witnessScript, arbiterA, arbiterB)
		if err := execute(t, scriptPubKey, tx); err != nil {
			t.Errorf("expected both-arbiters spend to succeed, got: %v", err)
		}
	})

	t.Run("only one signs: fails", func(t *testing.T) {
		tx := buildSpendingTx(0)
		fetcher := txscript.NewCannedPrevOutputFetcher(scriptPubKey, testAmount)
		sigHashes := txscript.NewTxSigHashes(tx, fetcher)
		w := wire.TxWitness{{}}
		sig, err := txscript.RawTxInWitnessSignature(tx, sigHashes, 0, testAmount, witnessScript, txscript.SigHashAll, arbiterA)
		if err != nil {
			t.Fatalf("RawTxInWitnessSignature: %v", err)
		}
		w = append(w, sig, witnessScript)
		tx.TxIn[0].Witness = w
		if err := execute(t, scriptPubKey, tx); err == nil {
			t.Error("expected a single appointed arbiter to be unable to move the escalated output alone")
		}
	})
}

func TestTieBreakScript_AnyTwoOfThreeSettle(t *testing.T) {
	arbiterA, arbiterAPub := newTestKey(t)
	arbiterB, arbiterBPub := newTestKey(t)
	arbiterC, arbiterCPub := newTestKey(t)

	witnessScript, err := TieBreakScript(arbiterAPub, arbiterBPub, arbiterCPub)
	if err != nil {
		t.Fatalf("TieBreakScript: %v", err)
	}
	scriptPubKey, err := WitnessScriptHash(witnessScript)
	if err != nil {
		t.Fatalf("WitnessScriptHash: %v", err)
	}

	combinations := [][2]*btcec.PrivateKey{
		{arbiterA, arbiterB},
		{arbiterA, arbiterC},
		{arbiterB, arbiterC},
	}
	for i, combo := range combinations {
		tx := buildSpendingTx(0)
		fetcher := txscript.NewCannedPrevOutputFetcher(scriptPubKey, testAmount)
		sigHashes := txscript.NewTxSigHashes(tx, fetcher)
		tx.TxIn[0].Witness = multisigWitness(t, tx, sigHashes, witnessScript, combo[0], combo[1])
		if err := execute(t, scriptPubKey, tx); err != nil {
			t.Errorf("combination %d: expected any 2-of-3 to settle, got: %v", i, err)
		}
	}
}

func TestSmallTierEscalatedScript_MutualSettlementPath(t *testing.T) {
	attester, attesterPub := newTestKey(t)
	challenger, challengerPub := newTestKey(t)
	_, arbiterPub := newTestKey(t)

	witnessScript, err := SmallTierEscalatedScript(attesterPub, challengerPub, arbiterPub)
	if err != nil {
		t.Fatalf("SmallTierEscalatedScript: %v", err)
	}
	scriptPubKey, err := WitnessScriptHash(witnessScript)
	if err != nil {
		t.Fatalf("WitnessScriptHash: %v", err)
	}

	tx := buildSpendingTx(0)
	fetcher := txscript.NewCannedPrevOutputFetcher(scriptPubKey, testAmount)
	sigHashes := txscript.NewTxSigHashes(tx, fetcher)
	w := multisigWitness(t, tx, sigHashes, witnessScript, attester, challenger)
	// Insert the IF-branch selector (true) just below the witness
	// script, which multisigWitness already appended last.
	selector := w[len(w)-1]
	w[len(w)-1] = []byte{1}
	w = append(w, selector)
	tx.TxIn[0].Witness = w

	if err := execute(t, scriptPubKey, tx); err != nil {
		t.Errorf("expected 2-of-2 mutual settlement to succeed, got: %v", err)
	}
}

func TestSmallTierEscalatedScript_ArbiterFallbackPath(t *testing.T) {
	_, attesterPub := newTestKey(t)
	_, challengerPub := newTestKey(t)
	arbiter, arbiterPub := newTestKey(t)

	witnessScript, err := SmallTierEscalatedScript(attesterPub, challengerPub, arbiterPub)
	if err != nil {
		t.Fatalf("SmallTierEscalatedScript: %v", err)
	}
	scriptPubKey, err := WitnessScriptHash(witnessScript)
	if err != nil {
		t.Fatalf("WitnessScriptHash: %v", err)
	}

	tx := buildSpendingTx(0)
	fetcher := txscript.NewCannedPrevOutputFetcher(scriptPubKey, testAmount)
	sigHashes := txscript.NewTxSigHashes(tx, fetcher)
	sig, err := txscript.RawTxInWitnessSignature(tx, sigHashes, 0, testAmount, witnessScript, txscript.SigHashAll, arbiter)
	if err != nil {
		t.Fatalf("RawTxInWitnessSignature: %v", err)
	}
	// ELSE branch: just the arbiter's signature, then the false
	// selector, then the witness script.
	tx.TxIn[0].Witness = wire.TxWitness{sig, {}, witnessScript}

	if err := execute(t, scriptPubKey, tx); err != nil {
		t.Errorf("expected arbiter-fallback spend to succeed, got: %v", err)
	}
}

func TestSmallTierEscalatedScript_ArbiterAloneCannotTakeMutualPath(t *testing.T) {
	_, attesterPub := newTestKey(t)
	_, challengerPub := newTestKey(t)
	arbiter, arbiterPub := newTestKey(t)

	witnessScript, err := SmallTierEscalatedScript(attesterPub, challengerPub, arbiterPub)
	if err != nil {
		t.Fatalf("SmallTierEscalatedScript: %v", err)
	}
	scriptPubKey, err := WitnessScriptHash(witnessScript)
	if err != nil {
		t.Fatalf("WitnessScriptHash: %v", err)
	}

	tx := buildSpendingTx(0)
	fetcher := txscript.NewCannedPrevOutputFetcher(scriptPubKey, testAmount)
	sigHashes := txscript.NewTxSigHashes(tx, fetcher)
	sig, err := txscript.RawTxInWitnessSignature(tx, sigHashes, 0, testAmount, witnessScript, txscript.SigHashAll, arbiter)
	if err != nil {
		t.Fatalf("RawTxInWitnessSignature: %v", err)
	}
	// Try to take the IF (mutual-settlement) branch with only the
	// arbiter's signature standing in for the 2-of-2 — must fail.
	tx.TxIn[0].Witness = wire.TxWitness{{}, sig, {1}, witnessScript}

	if err := execute(t, scriptPubKey, tx); err == nil {
		t.Error("expected the arbiter alone to be unable to satisfy the 2-of-2 mutual-settlement branch")
	}
}
