package escrow

import (
	"bytes"
	"context"
	"errors"
	"fmt"
)

// ChainQuerier is the minimal chain-query surface this package needs
// to verify an escrow is actually funded on-chain — an interface, not
// a concrete client, since no regtest/bitcoind node is reachable in
// this environment to build and test a real implementation against
// (see plan/actions/A006's Log; a concrete implementation, e.g.
// against btcd's or bitcoind's RPC, is deferred to whenever a real
// node is available to test it with, rather than shipped untested).
type ChainQuerier interface {
	// TxOut returns the confirmed output at txid:vout — its
	// scriptPubKey and value in satoshis — or found=false if it's
	// unconfirmed, already spent, or doesn't exist. Confirmation
	// status, not just mempool presence, is the caller's
	// responsibility to have this return true for, per A005/A006's
	// "watching for confirmation, not just an unconfirmed mempool
	// entry" requirement.
	TxOut(ctx context.Context, txid string, vout uint32) (scriptPubKey []byte, valueSats int64, found bool, err error)
}

// ErrNotFunded is returned when the referenced output isn't a
// confirmed, correctly-valued, correctly-scripted funding of the
// expected escrow.
var ErrNotFunded = errors.New("escrow: referenced output is not a confirmed funding of the expected escrow script")

// VerifyFunded checks that txid:vout is a confirmed on-chain output
// paying at least expectedAmountSats to expectedScriptPubKey — the
// P2WSH scriptPubKey computed from the escrow's own witness script
// (see WitnessScriptHash), not a free-text claim. A verifier should
// call this before trusting a bond/challenge's `EscrowRef` as real
// economic backing.
func VerifyFunded(ctx context.Context, q ChainQuerier, txid string, vout uint32, expectedScriptPubKey []byte, expectedAmountSats int64) error {
	scriptPubKey, valueSats, found, err := q.TxOut(ctx, txid, vout)
	if err != nil {
		return fmt.Errorf("escrow: querying %s:%d: %w", txid, vout, err)
	}
	if !found {
		return fmt.Errorf("%w: %s:%d not found or unconfirmed", ErrNotFunded, txid, vout)
	}
	if !bytes.Equal(scriptPubKey, expectedScriptPubKey) {
		return fmt.Errorf("%w: %s:%d pays a different script than expected", ErrNotFunded, txid, vout)
	}
	if valueSats < expectedAmountSats {
		return fmt.Errorf("%w: %s:%d holds %d sats, want at least %d", ErrNotFunded, txid, vout, valueSats, expectedAmountSats)
	}
	return nil
}
