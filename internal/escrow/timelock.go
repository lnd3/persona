// Package escrow implements A006: the actual Bitcoin script
// construction, funding, and settlement logic behind D001's
// bonding/slashing mechanism. Per A006's corrected design (2026-09-30,
// see plan/actions/A006's Log for the earlier escalation-based model
// this superseded and why): UniversalScript and ReinforcedScript are
// single, static scripts, fully configured at bond-creation time —
// there is no escalation transaction, ever. The disputed attestation
// already names its own subject_key (D001/A005 scope this mechanism
// to disputes between two *identified* parties), so the "future"
// counterparty was never actually unknown; only the arbiter is, and
// that is solved via a standing arbiter_commitment claim rather than
// a per-dispute selection.
//
// This package builds, signs, and validates scripts and settlement
// transactions; ChainQuerier (verify.go) is the interface a caller
// uses to check an escrow is actually funded on-chain — RPCQuerier
// (rpcquerier.go) is a real implementation of it, backed by a btcd/
// bitcoind JSON-RPC connection, exercised in this package's own live
// regtest tests (regtest_test.go). See internal/escrow/SECURITY_REVIEW.md
// for this package's own review package: scope, test coverage, and
// known residuals, assembled for A007.
package escrow

import (
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
)

// csvTimeFlag (BIP68 bit 22) marks a relative-locktime sequence as
// time-based (512-second units) rather than block-count-based — time
// units map onto `window_days` without needing a block-time
// assumption, unlike a block-count CSV.
const csvTimeFlag = 1 << 22

// secondsPerUnit is BIP68's fixed granularity for time-based relative
// locktimes.
const secondsPerUnit = 512

// maxCSVValue is BIP68's 16-bit value field ceiling: the longest
// representable time-based relative locktime is
// 0xFFFF*512 seconds =~ 388.9 days.
const maxCSVValue = 0xFFFF

// ErrWindowTooLong is returned when window_days can't be represented
// in a single BIP68 time-based relative locktime (~389 days).
var ErrWindowTooLong = errors.New("escrow: window_days exceeds the ~389-day ceiling a single BIP68 time-based relative locktime can represent")

// SequenceForDays converts a self-release window (A005's own
// per-bond `window_days` field) into a BIP68 time-based relative
// locktime sequence number, rounding up to the nearest 512-second
// unit so the actual enforced window is never shorter than requested.
func SequenceForDays(windowDays int) (uint32, error) {
	if windowDays <= 0 {
		return 0, fmt.Errorf("escrow: window_days must be positive, got %d", windowDays)
	}
	totalSeconds := int64(windowDays) * 86400
	units := (totalSeconds + secondsPerUnit - 1) / secondsPerUnit // round up
	if units > maxCSVValue {
		return 0, fmt.Errorf("%w: %d days", ErrWindowTooLong, windowDays)
	}
	return csvTimeFlag | uint32(units), nil
}

// PreEscalationScript builds a standalone CSV-timelocked single-sig
// witness script: the owner alone can spend, and only after `sequence`
// (from SequenceForDays) has elapsed relative to this output's
// confirmation. This exact fragment is reused as the final ("nobody
// engaged at all") fallback branch inside both UniversalScript and
// ReinforcedScript (upfront.go) — those are the scripts an actual bond
// or challenge stake gets funded to; this function is not itself a
// complete bond/stake output script on its own in the current design
// (the name predates that design — see A006's Log for the superseded
// model this was originally the *entire* output script for).
//
//	<sequence> OP_CHECKSEQUENCEVERIFY OP_DROP <ownerPubKey> OP_CHECKSIG
func PreEscalationScript(ownerPubKey []byte, sequence uint32) ([]byte, error) {
	if err := validatePubKey(ownerPubKey); err != nil {
		return nil, err
	}
	return txscript.NewScriptBuilder().
		AddInt64(int64(sequence)).
		AddOp(txscript.OP_CHECKSEQUENCEVERIFY).
		AddOp(txscript.OP_DROP).
		AddData(ownerPubKey).
		AddOp(txscript.OP_CHECKSIG).
		Script()
}

func validatePubKey(pubKey []byte) error {
	if len(pubKey) != 33 {
		return fmt.Errorf("escrow: public key must be 33 bytes (compressed), got %d", len(pubKey))
	}
	return nil
}

// WitnessScriptHash computes the P2WSH scriptPubKey for a given
// witness script: OP_0 <sha256(witnessScript)>.
func WitnessScriptHash(witnessScript []byte) ([]byte, error) {
	hash := sha256.Sum256(witnessScript)
	return txscript.NewScriptBuilder().
		AddOp(txscript.OP_0).
		AddData(hash[:]).
		Script()
}

// FundingAddress derives the actual bech32 P2WSH address to send
// funds to for a given witness script — what a caller populating a
// bond or challenge's `EscrowRef` (via FormatPreEscalationDescriptor)
// actually pays. Network-parameterized since mainnet and
// regtest/testnet addresses differ; A006's own mainnet gate means
// only regtest/testnet params should be used until the independent
// security review this action's Tasks list requires has happened.
func FundingAddress(witnessScript []byte, net *chaincfg.Params) (btcutil.Address, error) {
	hash := sha256.Sum256(witnessScript)
	return btcutil.NewAddressWitnessScriptHash(hash[:], net)
}
