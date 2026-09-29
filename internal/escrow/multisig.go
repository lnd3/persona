package escrow

import (
	"fmt"

	"github.com/btcsuite/btcd/txscript"
)

// ErrInvalidThreshold is returned when a multisig threshold isn't
// between 1 and the number of provided pubkeys.
var ErrInvalidThreshold = fmt.Errorf("escrow: multisig threshold must be between 1 and the number of pubkeys")

// MultisigScript builds a bare M-of-N OP_CHECKMULTISIG script:
//
//	<threshold> <pubkey1> ... <pubkeyN> <N> OP_CHECKMULTISIG
func MultisigScript(threshold int, pubkeys [][]byte) ([]byte, error) {
	if threshold < 1 || threshold > len(pubkeys) {
		return nil, ErrInvalidThreshold
	}
	if len(pubkeys) > 16 {
		return nil, fmt.Errorf("escrow: at most 16 pubkeys supported in a bare multisig, got %d", len(pubkeys))
	}
	builder := txscript.NewScriptBuilder().AddInt64(int64(threshold))
	for _, pk := range pubkeys {
		if err := validatePubKey(pk); err != nil {
			return nil, err
		}
		builder.AddData(pk)
	}
	builder.AddInt64(int64(len(pubkeys))).AddOp(txscript.OP_CHECKMULTISIG)
	return builder.Script()
}

// SmallTierEscalatedScript builds the small-bond tier's escalated
// output script, per A006's Decisions: the *first* resort is a plain
// 2-of-2 mutual settlement between the attester and challenger
// themselves (any split they agree to, no arbiter needed); the
// fallback, only reached if that fails, is a single mutually-agreed
// arbiter deciding alone.
//
//	OP_IF
//	    2 <attesterPubKey> <challengerPubKey> 2 OP_CHECKMULTISIG
//	OP_ELSE
//	    <arbiterPubKey> OP_CHECKSIG
//	OP_ENDIF
func SmallTierEscalatedScript(attesterPubKey, challengerPubKey, arbiterPubKey []byte) ([]byte, error) {
	mutualSettlement, err := MultisigScript(2, [][]byte{attesterPubKey, challengerPubKey})
	if err != nil {
		return nil, err
	}
	if err := validatePubKey(arbiterPubKey); err != nil {
		return nil, err
	}

	builder := txscript.NewScriptBuilder().
		AddOp(txscript.OP_IF).
		AddOps(mutualSettlement).
		AddOp(txscript.OP_ELSE).
		AddData(arbiterPubKey).
		AddOp(txscript.OP_CHECKSIG).
		AddOp(txscript.OP_ENDIF)
	return builder.Script()
}

// LargeTierEscalatedScript builds the larger-bond tier's escalated
// output script: a plain 2-of-2 between the two independently
// pre-committed arbiters — no path back to the disputants themselves,
// per A006's corrected Decisions (this tier's whole point is a
// decisive process once escalated, not a renegotiable one).
func LargeTierEscalatedScript(arbiterA, arbiterB []byte) ([]byte, error) {
	return MultisigScript(2, [][]byte{arbiterA, arbiterB})
}

// TieBreakScript builds the 2-of-3 output the two disagreeing
// arbiters jointly escalate to, naming themselves plus a
// jointly-nominated third arbiter. Any 2 of the 3 can then settle —
// per A006's Decisions, this always terminates in one step, since a
// binary verdict means the third arbiter's vote necessarily sides
// with one of the original two.
func TieBreakScript(arbiterA, arbiterB, arbiterC []byte) ([]byte, error) {
	return MultisigScript(2, [][]byte{arbiterA, arbiterB, arbiterC})
}
