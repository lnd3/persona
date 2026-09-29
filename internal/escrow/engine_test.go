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

// withSelectors removes witnessScript (the trailing element of w),
// appends the OP_IF selectors in innermost-to-outermost order (since
// the outermost OP_IF executes first and therefore must pop the
// topmost — i.e. last-pushed-before-the-script — witness item), then
// re-appends witnessScript.
func withSelectors(w wire.TxWitness, selectors ...[]byte) wire.TxWitness {
	script := w[len(w)-1]
	out := append(wire.TxWitness{}, w[:len(w)-1]...)
	out = append(out, selectors...)
	return append(out, script)
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

func TestUniversalScript_AllFourBranches(t *testing.T) {
	attester, attesterPub := newTestKey(t)
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

	newTx := func(seq uint32) (*wire.MsgTx, *txscript.TxSigHashes) {
		tx := buildSpendingTx(seq)
		fetcher := txscript.NewCannedPrevOutputFetcher(scriptPubKey, testAmount)
		return tx, txscript.NewTxSigHashes(tx, fetcher)
	}

	t.Run("branch 1: mutual settlement, any time, no timelock needed", func(t *testing.T) {
		tx, sigHashes := newTx(0)
		w := multisigWitness(t, tx, sigHashes, witnessScript, attester, subject)
		tx.TxIn[0].Witness = withSelectors(w, selTrue)
		if err := execute(t, scriptPubKey, tx); err != nil {
			t.Errorf("expected mutual settlement to succeed, got: %v", err)
		}
	})

	t.Run("branch 2: arbiter sides with attester", func(t *testing.T) {
		tx, sigHashes := newTx(0)
		w := multisigWitness(t, tx, sigHashes, witnessScript, attester, arbiter)
		tx.TxIn[0].Witness = withSelectors(w, selTrue, selFalse)
		if err := execute(t, scriptPubKey, tx); err != nil {
			t.Errorf("expected attester+arbiter to succeed, got: %v", err)
		}
	})

	t.Run("branch 3: arbiter sides with subject", func(t *testing.T) {
		tx, sigHashes := newTx(0)
		w := multisigWitness(t, tx, sigHashes, witnessScript, subject, arbiter)
		tx.TxIn[0].Witness = withSelectors(w, selTrue, selFalse, selFalse)
		if err := execute(t, scriptPubKey, tx); err != nil {
			t.Errorf("expected subject+arbiter to succeed, got: %v", err)
		}
	})

	t.Run("branch 4: last resort, self-release after window, nobody engaged", func(t *testing.T) {
		tx, sigHashes := newTx(sequence)
		sig, err := txscript.RawTxInWitnessSignature(tx, sigHashes, 0, testAmount, witnessScript, txscript.SigHashAll, attester)
		if err != nil {
			t.Fatalf("RawTxInWitnessSignature: %v", err)
		}
		tx.TxIn[0].Witness = withSelectors(wire.TxWitness{sig, witnessScript}, selFalse, selFalse, selFalse)
		if err := execute(t, scriptPubKey, tx); err != nil {
			t.Errorf("expected self-release fallback to succeed after the window, got: %v", err)
		}
	})

	t.Run("branch 4 fails before the window", func(t *testing.T) {
		tx, sigHashes := newTx(sequence - 1)
		sig, err := txscript.RawTxInWitnessSignature(tx, sigHashes, 0, testAmount, witnessScript, txscript.SigHashAll, attester)
		if err != nil {
			t.Fatalf("RawTxInWitnessSignature: %v", err)
		}
		tx.TxIn[0].Witness = withSelectors(wire.TxWitness{sig, witnessScript}, selFalse, selFalse, selFalse)
		if err := execute(t, scriptPubKey, tx); err == nil {
			t.Error("expected self-release fallback to fail before the window has elapsed")
		}
	})

	t.Run("arbiter alone cannot satisfy any branch", func(t *testing.T) {
		tx, sigHashes := newTx(0)
		sig, err := txscript.RawTxInWitnessSignature(tx, sigHashes, 0, testAmount, witnessScript, txscript.SigHashAll, arbiter)
		if err != nil {
			t.Fatalf("RawTxInWitnessSignature: %v", err)
		}
		tx.TxIn[0].Witness = withSelectors(wire.TxWitness{sig, witnessScript}, selFalse, selFalse, selFalse)
		if err := execute(t, scriptPubKey, tx); err == nil {
			t.Error("expected the arbiter alone to be unable to satisfy the fallback (attester-only) branch")
		}
	})

	t.Run("attester alone cannot take the mutual-settlement branch", func(t *testing.T) {
		tx, sigHashes := newTx(0)
		sig, err := txscript.RawTxInWitnessSignature(tx, sigHashes, 0, testAmount, witnessScript, txscript.SigHashAll, attester)
		if err != nil {
			t.Fatalf("RawTxInWitnessSignature: %v", err)
		}
		tx.TxIn[0].Witness = withSelectors(wire.TxWitness{{}, sig, witnessScript}, selTrue)
		if err := execute(t, scriptPubKey, tx); err == nil {
			t.Error("expected attester alone to be unable to satisfy the 2-of-2 mutual-settlement branch")
		}
	})
}

func TestReinforcedScript_AllFiveBranches(t *testing.T) {
	attester, attesterPub := newTestKey(t)
	subject, subjectPub := newTestKey(t)
	arbiterA, arbiterAPub := newTestKey(t) // attester's own standing arbiter
	arbiterB, arbiterBPub := newTestKey(t) // subject's own standing arbiter
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

	newTx := func(seq uint32) (*wire.MsgTx, *txscript.TxSigHashes) {
		tx := buildSpendingTx(seq)
		fetcher := txscript.NewCannedPrevOutputFetcher(scriptPubKey, testAmount)
		return tx, txscript.NewTxSigHashes(tx, fetcher)
	}

	t.Run("branch 1: mutual settlement", func(t *testing.T) {
		tx, sigHashes := newTx(0)
		w := multisigWitness(t, tx, sigHashes, witnessScript, attester, subject)
		tx.TxIn[0].Witness = withSelectors(w, selTrue)
		if err := execute(t, scriptPubKey, tx); err != nil {
			t.Errorf("expected mutual settlement to succeed, got: %v", err)
		}
	})

	t.Run("branch 2: both independent arbiters agree, no disputant needed", func(t *testing.T) {
		tx, sigHashes := newTx(0)
		w := multisigWitness(t, tx, sigHashes, witnessScript, arbiterA, arbiterB)
		tx.TxIn[0].Witness = withSelectors(w, selTrue, selFalse)
		if err := execute(t, scriptPubKey, tx); err != nil {
			t.Errorf("expected both-arbiters-agree to succeed, got: %v", err)
		}
	})

	t.Run("branch 3: attester's own arbiter sides with attester", func(t *testing.T) {
		tx, sigHashes := newTx(0)
		w := multisigWitness(t, tx, sigHashes, witnessScript, attester, arbiterA)
		tx.TxIn[0].Witness = withSelectors(w, selTrue, selFalse, selFalse)
		if err := execute(t, scriptPubKey, tx); err != nil {
			t.Errorf("expected attester+own-arbiter to succeed, got: %v", err)
		}
	})

	t.Run("branch 4: subject's own arbiter sides with subject", func(t *testing.T) {
		tx, sigHashes := newTx(0)
		w := multisigWitness(t, tx, sigHashes, witnessScript, subject, arbiterB)
		tx.TxIn[0].Witness = withSelectors(w, selTrue, selFalse, selFalse, selFalse)
		if err := execute(t, scriptPubKey, tx); err != nil {
			t.Errorf("expected subject+own-arbiter to succeed, got: %v", err)
		}
	})

	t.Run("branch 5: last-resort self-release after window", func(t *testing.T) {
		tx, sigHashes := newTx(sequence)
		sig, err := txscript.RawTxInWitnessSignature(tx, sigHashes, 0, testAmount, witnessScript, txscript.SigHashAll, attester)
		if err != nil {
			t.Fatalf("RawTxInWitnessSignature: %v", err)
		}
		tx.TxIn[0].Witness = withSelectors(wire.TxWitness{sig, witnessScript}, selFalse, selFalse, selFalse, selFalse)
		if err := execute(t, scriptPubKey, tx); err != nil {
			t.Errorf("expected self-release fallback to succeed after the window, got: %v", err)
		}
	})

	t.Run("cross-pairing fails: attester + subject's own arbiter", func(t *testing.T) {
		tx, sigHashes := newTx(0)
		w := multisigWitness(t, tx, sigHashes, witnessScript, attester, arbiterB)
		tx.TxIn[0].Witness = withSelectors(w, selTrue, selFalse, selFalse)
		if err := execute(t, scriptPubKey, tx); err == nil {
			t.Error("expected attester paired with subject's own arbiter to fail on the attester-arbiter branch")
		}
	})
}
