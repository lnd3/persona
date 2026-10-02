package escrow

import (
	"fmt"

	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

// This file closes the fee/dust-limit gap found while scoping A006's
// mainnet gate (2026-09-30): until now, nothing in this package
// estimated a real transaction fee or rejected a dust payout — the
// only fee value anywhere in the codebase was a hardcoded constant
// inside test helpers. Real measurements taken while scoping this
// (UniversalScript branch vsize 181-199, ReinforcedScript 200-218,
// depending on branch) are now backed by the functions below rather
// than quoted as one-off numbers.

// maxDERSignatureLen is a safe real-world upper bound for a DER-
// encoded ECDSA signature plus its trailing sighash-type byte (BIP66
// DER, plus SIGHASH_ALL) — 72 bytes covers every signature
// SignSettlementInput can actually produce. Using this upper bound for
// fee estimation means the estimate is never too low for a real,
// canonical low-S signature, which is all this package ever produces.
const maxDERSignatureLen = 72

// Branch identifies one spending path of UniversalScript or
// ReinforcedScript — the information fee estimation needs to know the
// real witness shape before any real signature exists. The values
// below mirror exactly the signature counts and OP_IF selectors
// FinalizeUniversal/FinalizeReinforced already use internally
// (settle.go) — this is deliberately the single source of truth for
// "how many selectors does branch X need," not a second copy of it.
type Branch struct {
	sigCount  int
	selectors [][]byte
}

// The eight real branches across both scripts, named and ordered to
// match FinalizeUniversal's/FinalizeReinforced's own branch order and
// comments exactly.
var (
	BranchMutualSettlement              = Branch{sigCount: 2, selectors: [][]byte{selTrue}}
	BranchUniversalArbiterSidesAttester = Branch{sigCount: 2, selectors: [][]byte{selTrue, selFalse}}
	BranchUniversalArbiterSidesSubject  = Branch{sigCount: 2, selectors: [][]byte{selTrue, selFalse, selFalse}}
	BranchUniversalFallback             = Branch{sigCount: 1, selectors: [][]byte{selFalse, selFalse, selFalse}}
	BranchReinforcedBothArbitersAgree   = Branch{sigCount: 2, selectors: [][]byte{selTrue, selFalse}}
	BranchReinforcedAttesterOwnArbiter  = Branch{sigCount: 2, selectors: [][]byte{selTrue, selFalse, selFalse}}
	BranchReinforcedSubjectOwnArbiter   = Branch{sigCount: 2, selectors: [][]byte{selTrue, selFalse, selFalse, selFalse}}
	BranchReinforcedFallback            = Branch{sigCount: 1, selectors: [][]byte{selFalse, selFalse, selFalse, selFalse}}
)

// maxWitness returns the worst-case witness stack for settling through
// branch against witnessScript: a leading dummy element if this branch
// needs two signatures (matching finalizeBranch's own CHECKMULTISIG-
// bug workaround), maxDERSignatureLen-sized placeholder signatures,
// the branch's own real selector values (the same selTrue/selFalse
// values Finalize* actually writes), and the witness script itself —
// the exact shape setFinalWitness (settle.go) produces for real, just
// with maximum-length placeholder signatures instead of real ones.
func (b Branch) maxWitness(witnessScript []byte) wire.TxWitness {
	var elements wire.TxWitness
	if b.sigCount == 2 {
		elements = append(elements, []byte{}) // leading dummy, same as finalizeBranch
	}
	for i := 0; i < b.sigCount; i++ {
		elements = append(elements, make([]byte, maxDERSignatureLen))
	}
	elements = append(elements, b.selectors...)
	elements = append(elements, witnessScript)
	return elements
}

// EstimateSettlementVSize returns the virtual size (vbytes, per
// BIP141) a settlement transaction spending through branch would have,
// using branch's own worst-case witness — exact for this package's own
// known script shapes, not a generic heuristic. Every settlement's
// witness shape is one of a small, named set (Branch), so this package
// can measure its own real spending cost directly instead of
// estimating it the way a general-purpose wallet has to.
func EstimateSettlementVSize(witnessScript []byte, payouts []*wire.TxOut, sequence uint32, branch Branch) (int64, error) {
	if len(payouts) == 0 {
		return 0, fmt.Errorf("escrow: EstimateSettlementVSize: at least one payout required")
	}
	tx := wire.NewMsgTx(2)
	tx.AddTxIn(&wire.TxIn{
		PreviousOutPoint: wire.OutPoint{Hash: chainhash.Hash{}, Index: 0},
		Witness:          branch.maxWitness(witnessScript),
		Sequence:         sequence,
	})
	for _, out := range payouts {
		tx.AddTxOut(out)
	}
	weight := blockchain.GetTransactionWeight(btcutil.NewTx(tx))
	return (weight + 3) / 4, nil // BIP141 vsize = ceil(weight / 4)
}

// EstimateSettlementFee is EstimateSettlementVSize scaled by a chosen
// feerate. feeRateSatPerVByte is the caller's own choice — a real
// deployment should source it from a live estimator (e.g.
// RPCQuerier's own node via its `estimatesmartfee` RPC, not yet
// wrapped here — see A006's own Log for why that still needs testing
// against a node with real fee history before being added) rather than
// a hardcoded value. This package has no opinion on what a "normal"
// feerate is, only on how many vbytes its own transactions actually
// cost to relay.
func EstimateSettlementFee(witnessScript []byte, payouts []*wire.TxOut, sequence uint32, branch Branch, feeRateSatPerVByte int64) (vsize int64, fee int64, err error) {
	vsize, err = EstimateSettlementVSize(witnessScript, payouts, sequence, branch)
	if err != nil {
		return 0, 0, err
	}
	return vsize, vsize * feeRateSatPerVByte, nil
}

// IsDustOutput reports whether txOut's value is below the dust
// threshold at minRelayFeeSatPerVByte: the cost to the network of
// later spending this output would exceed a fixed multiple (3x, the
// same policy btcd/bitcoind use by default) of what a minimum-relay-fee
// transaction spending it would cost.
//
// Hand-rolled rather than importing
// github.com/btcsuite/btcd/mempool's own IsDust/GetDustThreshold —
// found while scoping A006's mainnet gate (2026-09-30) that importing
// it drags in unrelated dependencies (github.com/aead/siphash,
// github.com/kkdai/bstream, github.com/stretchr/testify, pulled in by
// mempool's own unrelated compact-block-filter support) for what's
// really a ~15-line formula — the same class of gotcha
// hashicorp/vault/shamir taught in A003. Matches btcd's own
// well-documented formula (see that package's GetDustThreshold for the
// full byte-budget reasoning this mirrors), generalized to take an
// explicit feerate parameter rather than hardcoding btcd's own default
// 1 sat/vByte minimum relay fee.
func IsDustOutput(txOut *wire.TxOut, minRelayFeeSatPerVByte int64) bool {
	if txscript.IsUnspendable(txOut.PkScript) {
		return true
	}
	return txOut.Value < dustThreshold(txOut, minRelayFeeSatPerVByte)
}

// dustThreshold computes the minimum non-dust value for txOut: its own
// serialized size plus a 41-byte preamble (36-byte prevout reference +
// 1-byte script-length + 4-byte sequence) plus a typical spending
// input's own cost — witness programs discount that spending input's
// signature/pubkey bytes by BIP141's 4x witness scale factor, matching
// real P2WPKH/P2WSH spending cost; anything else assumes a legacy
// P2PKH-sized input, the same conservative default btcd's own
// GetDustThreshold uses. The result scales linearly with
// minRelayFeeSatPerVByte so a caller isn't locked into the network's
// historical 1 sat/vByte default.
func dustThreshold(txOut *wire.TxOut, minRelayFeeSatPerVByte int64) int64 {
	const (
		preambleSize           = 41
		legacySpendInputSize   = 107
		witnessScaleFactor     = 4
		dustRelayFeeMultiplier = 3
	)
	totalSize := int64(txOut.SerializeSize() + preambleSize)
	if txscript.IsWitnessProgram(txOut.PkScript) {
		totalSize += legacySpendInputSize / witnessScaleFactor
	} else {
		totalSize += legacySpendInputSize
	}
	return dustRelayFeeMultiplier * totalSize * minRelayFeeSatPerVByte
}
