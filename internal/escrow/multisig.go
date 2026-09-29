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

// TieBreakScript builds a 2-of-3 output naming two disagreeing
// parties plus a third — a general-purpose building block, kept for
// potential reuse, though it is not one of the branches
// UniversalScript/ReinforcedScript build directly (see upfront.go):
// under A006's corrected upfront-configured design (2026-09-30),
// escalation transactions no longer exist, so there is no live
// "moment" at which two disagreeing arbiters could cooperatively
// escalate to a jointly-nominated third the way the original
// (superseded) escalation design assumed. Any 2 of the 3 named here
// can settle.
func TieBreakScript(arbiterA, arbiterB, arbiterC []byte) ([]byte, error) {
	return MultisigScript(2, [][]byte{arbiterA, arbiterB, arbiterC})
}
