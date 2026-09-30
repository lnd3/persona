package escrow

import (
	"bytes"
	"context"
	"errors"
	"fmt"
)

// ChainQuerier is the minimal chain-query surface this package needs
// to verify an escrow is actually funded on-chain.
type ChainQuerier interface {
	// TxOut returns the output at txid:vout — its scriptPubKey, value
	// in satoshis, and confirmation count — or found=false if it's
	// spent or doesn't exist. found=true with confirmations=0 means
	// "exists, mempool only, not yet confirmed" — callers that care
	// about confirmation depth (all of them, via VerifyFunded's own
	// minConfirmations parameter) decide their own required threshold;
	// this interface just reports the real number rather than baking a
	// threshold into the query itself.
	TxOut(ctx context.Context, txid string, vout uint32) (scriptPubKey []byte, valueSats int64, confirmations int64, found bool, err error)
}

// ErrNotFunded is returned when the referenced output isn't a
// sufficiently-confirmed, correctly-valued, correctly-scripted funding
// of the expected escrow.
var ErrNotFunded = errors.New("escrow: referenced output is not a confirmed funding of the expected escrow script")

// VerifyFunded checks that txid:vout is an on-chain output, confirmed
// to at least minConfirmations, paying at least expectedAmountSats to
// expectedScriptPubKey — the P2WSH scriptPubKey computed from the
// escrow's own witness script (see WitnessScriptHash), not a free-text
// claim. A verifier should call this before trusting a bond/
// challenge's `EscrowRef` as real economic backing.
//
// minConfirmations is a real, caller-chosen policy decision, not a
// library default — found live while scoping A006's mainnet gate
// (2026-09-30) as a gap nobody had actually decided: 1 confirmation is
// not reorg-safe for real value, but the "right" number depends on the
// amount at stake and the caller's own risk tolerance, not something
// this package can pick for every caller. minConfirmations <= 0 is
// rejected outright rather than silently treated as "mempool
// presence is enough" — accepting an unconfirmed entry as funding is
// exactly the risk A005/A006 both required guarding against.
func VerifyFunded(ctx context.Context, q ChainQuerier, txid string, vout uint32, expectedScriptPubKey []byte, expectedAmountSats int64, minConfirmations int64) error {
	if minConfirmations <= 0 {
		return fmt.Errorf("escrow: minConfirmations must be >= 1, got %d", minConfirmations)
	}
	scriptPubKey, valueSats, confirmations, found, err := q.TxOut(ctx, txid, vout)
	if err != nil {
		return fmt.Errorf("escrow: querying %s:%d: %w", txid, vout, err)
	}
	if !found {
		return fmt.Errorf("%w: %s:%d not found or already spent", ErrNotFunded, txid, vout)
	}
	if confirmations < minConfirmations {
		return fmt.Errorf("%w: %s:%d has %d confirmations, want at least %d", ErrNotFunded, txid, vout, confirmations, minConfirmations)
	}
	if !bytes.Equal(scriptPubKey, expectedScriptPubKey) {
		return fmt.Errorf("%w: %s:%d pays a different script than expected", ErrNotFunded, txid, vout)
	}
	if valueSats < expectedAmountSats {
		return fmt.Errorf("%w: %s:%d holds %d sats, want at least %d", ErrNotFunded, txid, vout, valueSats, expectedAmountSats)
	}
	return nil
}
