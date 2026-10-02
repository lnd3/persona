package escrow

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil/psbt"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

// This file implements the PSBT-based settlement construction A006
// originally needed for cooperative *escalation* transactions — but
// under the corrected, no-escalation design (upfront.go), settlement
// is always exactly one transaction spending the bond/stake output
// directly through whichever branch of UniversalScript/
// ReinforcedScript applies. There is no multi-step transaction chain
// to construct, which is why this file is much smaller than the
// escalation-based design would have needed.
//
// The standard library's own psbt.Finalize can't be used here: its
// built-in multisig finalizer requires a script that's recognizable
// as a plain "M of N CHECKMULTISIG" (checked via
// checkIsMultiSigScript), which neither our OP_IF-branching scripts
// nor the single-key CSV fallback branch are. This file's
// FinalizeUniversal/FinalizeReinforced do that assembly by hand
// instead, reusing the exact witness ordering already validated
// against the real consensus engine in engine_test.go.

// ErrNotFinalizable is returned when a settlement packet's input
// doesn't yet have signatures satisfying any branch of its script.
var ErrNotFinalizable = errors.New("escrow: input does not yet have signatures satisfying any script branch")

// minRelayFeeSatPerVByte is the standard Bitcoin network default (1
// sat/vByte) used to reject dust payouts in NewSettlementPacket. A
// real deployment using a node with a different `-minrelaytxfee` would
// need a different threshold — not expected to vary in practice, so
// not exposed as a parameter; call IsDustOutput directly with a real
// feerate if that assumption is ever wrong for a specific deployment.
const minRelayFeeSatPerVByte = 1

// NewSettlementPacket builds an unsigned PSBT spending a single
// UniversalScript/ReinforcedScript/PreEscalationScript output. sequence
// should be 0 unless the caller specifically intends to take the
// CSV-gated fallback branch, in which case it must be at least the
// script's own required sequence (see SequenceForDays) — the branches
// requiring cooperation (mutual settlement, arbiter-assisted) have no
// timelock and work with sequence 0.
//
// Rejects any payout below the standard dust threshold (see
// IsDustOutput) — found missing while scoping A006's mainnet gate
// (2026-09-30): nothing previously stopped this package from
// constructing a transaction any real node's mempool policy would
// simply refuse to relay. Checked here, not left to the caller,
// because this is the one place that already has every payout in hand
// before any signature work happens.
func NewSettlementPacket(prevOut wire.OutPoint, prevTxOut *wire.TxOut, witnessScript []byte, payouts []*wire.TxOut, sequence uint32) (*psbt.Packet, error) {
	for i, out := range payouts {
		if IsDustOutput(out, minRelayFeeSatPerVByte) {
			return nil, fmt.Errorf("escrow: payout %d (%d sats) is below the dust threshold", i, out.Value)
		}
	}
	p, err := psbt.New([]*wire.OutPoint{&prevOut}, payouts, 2, 0, []uint32{sequence})
	if err != nil {
		return nil, fmt.Errorf("escrow: creating settlement packet: %w", err)
	}
	p.Inputs[0].WitnessUtxo = prevTxOut
	p.Inputs[0].WitnessScript = witnessScript
	return p, nil
}

// SignSettlementInput signs input 0 of a settlement packet with priv,
// storing the result as a PSBT partial signature. Cooperating parties
// (e.g. attester and subject settling directly) each call this
// independently — possibly at different times, possibly never
// exchanging anything but the resulting partial signatures — before
// Finalize* assembles the final witness once enough are present.
func SignSettlementInput(p *psbt.Packet, priv *btcec.PrivateKey) error {
	pInput := p.Inputs[0]
	if pInput.WitnessUtxo == nil || pInput.WitnessScript == nil {
		return errors.New("escrow: settlement packet input missing WitnessUtxo/WitnessScript")
	}
	fetcher := txscript.NewCannedPrevOutputFetcher(pInput.WitnessUtxo.PkScript, pInput.WitnessUtxo.Value)
	sigHashes := txscript.NewTxSigHashes(p.UnsignedTx, fetcher)
	sig, err := txscript.RawTxInWitnessSignature(
		p.UnsignedTx, sigHashes, 0, pInput.WitnessUtxo.Value, pInput.WitnessScript, txscript.SigHashAll, priv,
	)
	if err != nil {
		return fmt.Errorf("escrow: signing settlement input: %w", err)
	}
	updater, err := psbt.NewUpdater(p)
	if err != nil {
		return fmt.Errorf("escrow: creating updater: %w", err)
	}
	pubKey := priv.PubKey().SerializeCompressed()
	if _, err := updater.Sign(0, sig, pubKey, nil, pInput.WitnessScript); err != nil {
		return fmt.Errorf("escrow: attaching partial signature: %w", err)
	}
	return nil
}

func partialSig(p *psbt.Packet, pubKey []byte) []byte {
	for _, ps := range p.Inputs[0].PartialSigs {
		if bytes.Equal(ps.PubKey, pubKey) {
			return ps.Signature
		}
	}
	return nil
}

// finalizeBranch writes FinalScriptWitness for a 2-of-2 branch, given
// the branch's two pubkeys (in script order) and the OP_IF selectors
// (innermost to outermost) needed to reach it. Returns
// ErrNotFinalizable, not a hard error, if the needed signatures
// simply aren't present yet — callers try branches in sequence.
func finalizeBranch(p *psbt.Packet, witnessScript []byte, pubKeyA, pubKeyB []byte, selectors [][]byte) error {
	sigA, sigB := partialSig(p, pubKeyA), partialSig(p, pubKeyB)
	if sigA == nil || sigB == nil {
		return ErrNotFinalizable
	}
	elements := wire.TxWitness{{}, sigA, sigB} // leading dummy: CHECKMULTISIG's historic off-by-one
	elements = append(elements, selectors...)
	elements = append(elements, witnessScript)
	return setFinalWitness(p, elements)
}

// finalizeFallback writes FinalScriptWitness for the single-key CSV
// fallback branch. It refuses (ErrNotFinalizable) if the packet's own
// input sequence doesn't yet satisfy requiredSequence — otherwise it
// would happily produce a witness that real script validation (and
// any relay/miner) would reject at broadcast time.
func finalizeFallback(p *psbt.Packet, witnessScript []byte, ownerPubKey []byte, requiredSequence uint32, selectors [][]byte) error {
	if p.UnsignedTx.TxIn[0].Sequence < requiredSequence {
		return ErrNotFinalizable
	}
	sig := partialSig(p, ownerPubKey)
	if sig == nil {
		return ErrNotFinalizable
	}
	elements := wire.TxWitness{sig}
	elements = append(elements, selectors...)
	elements = append(elements, witnessScript)
	return setFinalWitness(p, elements)
}

func setFinalWitness(p *psbt.Packet, elements wire.TxWitness) error {
	var buf bytes.Buffer
	if err := psbt.WriteTxWitness(&buf, elements); err != nil {
		return fmt.Errorf("escrow: serializing final witness: %w", err)
	}
	newInput := psbt.NewPsbtInput(nil, p.Inputs[0].WitnessUtxo)
	newInput.FinalScriptWitness = buf.Bytes()
	p.Inputs[0] = *newInput
	return nil
}

// FinalizeUniversal finalizes a settlement packet built against a
// UniversalScript output, trying each branch (mutual settlement,
// attester+arbiter, subject+arbiter, then the attester-alone
// fallback) in turn, using whichever partial signatures are actually
// present. requiredSequence is the script's own CSV requirement (see
// SequenceForDays) — the fallback branch refuses to finalize unless
// the packet's input sequence already satisfies it, since a witness
// that doesn't would simply be rejected at broadcast time anyway.
func FinalizeUniversal(p *psbt.Packet, witnessScript, attesterPubKey, subjectPubKey, arbiterPubKey []byte, requiredSequence uint32) error {
	if err := finalizeBranch(p, witnessScript, attesterPubKey, subjectPubKey, [][]byte{selTrue}); err == nil {
		return nil
	}
	if err := finalizeBranch(p, witnessScript, attesterPubKey, arbiterPubKey, [][]byte{selTrue, selFalse}); err == nil {
		return nil
	}
	if err := finalizeBranch(p, witnessScript, subjectPubKey, arbiterPubKey, [][]byte{selTrue, selFalse, selFalse}); err == nil {
		return nil
	}
	if err := finalizeFallback(p, witnessScript, attesterPubKey, requiredSequence, [][]byte{selFalse, selFalse, selFalse}); err == nil {
		return nil
	}
	return ErrNotFinalizable
}

// FinalizeReinforced is FinalizeUniversal's counterpart for
// ReinforcedScript, additionally trying the "both independent
// arbiters agree" branch (before the weaker single-arbiter-plus-
// disputant branches).
func FinalizeReinforced(p *psbt.Packet, witnessScript, attesterPubKey, subjectPubKey, attesterArbiterPubKey, subjectArbiterPubKey []byte, requiredSequence uint32) error {
	if err := finalizeBranch(p, witnessScript, attesterPubKey, subjectPubKey, [][]byte{selTrue}); err == nil {
		return nil
	}
	if err := finalizeBranch(p, witnessScript, attesterArbiterPubKey, subjectArbiterPubKey, [][]byte{selTrue, selFalse}); err == nil {
		return nil
	}
	if err := finalizeBranch(p, witnessScript, attesterPubKey, attesterArbiterPubKey, [][]byte{selTrue, selFalse, selFalse}); err == nil {
		return nil
	}
	if err := finalizeBranch(p, witnessScript, subjectPubKey, subjectArbiterPubKey, [][]byte{selTrue, selFalse, selFalse, selFalse}); err == nil {
		return nil
	}
	if err := finalizeFallback(p, witnessScript, attesterPubKey, requiredSequence, [][]byte{selFalse, selFalse, selFalse, selFalse}); err == nil {
		return nil
	}
	return ErrNotFinalizable
}

// ExtractSettlement returns the final, broadcastable transaction from
// a fully finalized settlement packet.
func ExtractSettlement(p *psbt.Packet) (*wire.MsgTx, error) {
	tx, err := psbt.Extract(p)
	if err != nil {
		return nil, fmt.Errorf("escrow: extracting settlement tx: %w", err)
	}
	return tx, nil
}
