package escrow

import (
	"github.com/btcsuite/btcd/txscript"
)

// selTrue/selFalse are the OP_IF selector values used both when
// building a spending witness by hand and by the settle.go finalizer
// — pushed in innermost-to-outermost order, since the outermost
// OP_IF executes first and therefore pops the topmost (last-pushed)
// witness item.
var (
	selTrue  = []byte{1}
	selFalse = []byte{}
)

// This file implements A006's corrected design (2026-09-30,
// superseding the earlier escalation-based two-tier model): a single
// script, fully configured at bond-creation time, with no escalation
// transaction and no key that doesn't exist yet.
//
// The insight that makes this possible: D001/A005 scoped
// bonding/slashing to dispute type 1 only — "behavioral/quality
// disputes between two *identified* parties." The bond secures the
// attester's own attestation, and that attestation already names a
// subject_key — the very party who would become the challenger if
// they disagree. The counterparty isn't an unknown future stranger;
// it's a real, already-public keypair the moment the attestation
// exists. The only genuinely unknown-at-funding-time party is the
// arbiter, which this design resolves via a *standing* arbiter
// commitment (arbiter_commitment.go) — published once, independent
// of any specific bond, reusable across every future dispute where a
// persona is named as either attester or subject — rather than a
// per-dispute selection that would reopen the original chicken-and-egg
// problem.

// UniversalScript builds the always-available bond/stake script:
// every key it needs (attester, subject, attester's own standing
// arbiter) is knowable at bond-creation time with no dependency on
// the subject having published anything.
//
//	IF
//	    2-of-2 (attester, subject)            -- mutual settlement, any time
//	ELSE
//	    IF
//	        2-of-2 (attester, arbiter)         -- arbiter sides with attester
//	    ELSE
//	        IF
//	            2-of-2 (subject, arbiter)      -- arbiter sides with subject
//	        ELSE
//	            <sequence> CSV DROP <attester> CHECKSIG  -- last resort: nobody engaged
//	        ENDIF
//	    ENDIF
//	ENDIF
//
// The mutual-settlement branch needs no timelock at all — attester
// and subject can settle directly, immediately, any time, which is
// what makes voluntary resolution actually possible during an active
// dispute (the superseded design's single CSV-gated path blocked
// this entirely — see plan/actions/A006's Log). The CSV-gated
// self-release is now a genuine last resort: it is only reachable if
// *nobody* — not subject, not arbiter — ever engaged at all.
func UniversalScript(attesterPubKey, subjectPubKey, arbiterPubKey []byte, sequence uint32) ([]byte, error) {
	mutual, err := MultisigScript(2, [][]byte{attesterPubKey, subjectPubKey})
	if err != nil {
		return nil, err
	}
	attesterArbiter, err := MultisigScript(2, [][]byte{attesterPubKey, arbiterPubKey})
	if err != nil {
		return nil, err
	}
	subjectArbiter, err := MultisigScript(2, [][]byte{subjectPubKey, arbiterPubKey})
	if err != nil {
		return nil, err
	}
	fallback, err := PreEscalationScript(attesterPubKey, sequence)
	if err != nil {
		return nil, err
	}

	return txscript.NewScriptBuilder().
		AddOp(txscript.OP_IF).
		AddOps(mutual).
		AddOp(txscript.OP_ELSE).
		AddOp(txscript.OP_IF).
		AddOps(attesterArbiter).
		AddOp(txscript.OP_ELSE).
		AddOp(txscript.OP_IF).
		AddOps(subjectArbiter).
		AddOp(txscript.OP_ELSE).
		AddOps(fallback).
		AddOp(txscript.OP_ENDIF).
		AddOp(txscript.OP_ENDIF).
		AddOp(txscript.OP_ENDIF).
		Script()
}

// ReinforcedScript builds UniversalScript's branches plus one
// additional, stronger branch: the two independently, standing
// pre-committed arbiters (attester's own, and subject's own) agreeing
// with each other directly, with *neither* disputant's own signature
// needed at all. Only available when the subject has independently
// published their own standing arbiter_commitment — a real, bounded
// precondition, not hidden: if the subject hasn't opted in, only
// UniversalScript is buildable for a claim naming them.
//
//	IF        2-of-2 (attester, subject)                    -- mutual settlement
//	ELSE IF   2-of-2 (attesterArbiter, subjectArbiter)       -- both arbiters agree, no disputant needed
//	ELSE IF   2-of-2 (attester, attesterArbiter)             -- arbiter sides with attester
//	ELSE IF   2-of-2 (subject, subjectArbiter)               -- arbiter sides with subject
//	ELSE      <sequence> CSV DROP <attester> CHECKSIG        -- last resort
//
// Residual, stated honestly: if the two independent arbiters disagree
// (attesterArbiter sides with attester via the third branch while
// subjectArbiter separately sides with subject via the fourth), there
// is no further tie-break baked into this script — under this static,
// fully-upfront design there is no live moment to escalate to a third
// arbiter the way the superseded design imagined. Whichever competing
// spend is actually confirmed first wins; this is an accepted, rare
// (both arbiters must actually disagree) residual, not silently
// glossed over.
func ReinforcedScript(attesterPubKey, subjectPubKey, attesterArbiterPubKey, subjectArbiterPubKey []byte, sequence uint32) ([]byte, error) {
	mutual, err := MultisigScript(2, [][]byte{attesterPubKey, subjectPubKey})
	if err != nil {
		return nil, err
	}
	bothArbiters, err := MultisigScript(2, [][]byte{attesterArbiterPubKey, subjectArbiterPubKey})
	if err != nil {
		return nil, err
	}
	attesterArbiter, err := MultisigScript(2, [][]byte{attesterPubKey, attesterArbiterPubKey})
	if err != nil {
		return nil, err
	}
	subjectArbiter, err := MultisigScript(2, [][]byte{subjectPubKey, subjectArbiterPubKey})
	if err != nil {
		return nil, err
	}
	fallback, err := PreEscalationScript(attesterPubKey, sequence)
	if err != nil {
		return nil, err
	}

	return txscript.NewScriptBuilder().
		AddOp(txscript.OP_IF).
		AddOps(mutual).
		AddOp(txscript.OP_ELSE).
		AddOp(txscript.OP_IF).
		AddOps(bothArbiters).
		AddOp(txscript.OP_ELSE).
		AddOp(txscript.OP_IF).
		AddOps(attesterArbiter).
		AddOp(txscript.OP_ELSE).
		AddOp(txscript.OP_IF).
		AddOps(subjectArbiter).
		AddOp(txscript.OP_ELSE).
		AddOps(fallback).
		AddOp(txscript.OP_ENDIF).
		AddOp(txscript.OP_ENDIF).
		AddOp(txscript.OP_ENDIF).
		AddOp(txscript.OP_ENDIF).
		Script()
}
