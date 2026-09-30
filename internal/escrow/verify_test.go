package escrow

import (
	"context"
	"errors"
	"testing"
)

// mockQuerier is a trivial ChainQuerier for testing VerifyFunded's own
// logic in isolation from any real chain backend.
type mockQuerier struct {
	scriptPubKey  []byte
	valueSats     int64
	confirmations int64
	found         bool
	err           error
}

func (m mockQuerier) TxOut(ctx context.Context, txid string, vout uint32) ([]byte, int64, int64, bool, error) {
	return m.scriptPubKey, m.valueSats, m.confirmations, m.found, m.err
}

func TestVerifyFundedSucceeds(t *testing.T) {
	script := []byte{0x00, 0x01, 0x02}
	q := mockQuerier{scriptPubKey: script, valueSats: 50_000, confirmations: 1, found: true}
	if err := VerifyFunded(context.Background(), q, "txid", 0, script, 50_000, 1); err != nil {
		t.Errorf("VerifyFunded: %v", err)
	}
}

func TestVerifyFundedRejectsUnfunded(t *testing.T) {
	q := mockQuerier{found: false}
	if err := VerifyFunded(context.Background(), q, "txid", 0, []byte{0x00}, 50_000, 1); !errors.Is(err, ErrNotFunded) {
		t.Errorf("VerifyFunded = %v, want ErrNotFunded", err)
	}
}

func TestVerifyFundedRejectsUnderfunded(t *testing.T) {
	script := []byte{0x00, 0x01}
	q := mockQuerier{scriptPubKey: script, valueSats: 10_000, confirmations: 1, found: true}
	if err := VerifyFunded(context.Background(), q, "txid", 0, script, 50_000, 1); !errors.Is(err, ErrNotFunded) {
		t.Errorf("VerifyFunded = %v, want ErrNotFunded (underfunded)", err)
	}
}

func TestVerifyFundedRejectsWrongScript(t *testing.T) {
	q := mockQuerier{scriptPubKey: []byte{0xAA}, valueSats: 50_000, confirmations: 1, found: true}
	if err := VerifyFunded(context.Background(), q, "txid", 0, []byte{0xBB}, 50_000, 1); !errors.Is(err, ErrNotFunded) {
		t.Errorf("VerifyFunded = %v, want ErrNotFunded (wrong script)", err)
	}
}

func TestVerifyFundedPropagatesQueryError(t *testing.T) {
	wantErr := errors.New("node unreachable")
	q := mockQuerier{err: wantErr}
	err := VerifyFunded(context.Background(), q, "txid", 0, []byte{0x00}, 50_000, 1)
	if err == nil || !errors.Is(err, wantErr) {
		t.Errorf("VerifyFunded = %v, want wrapping %v", err, wantErr)
	}
}

// TestVerifyFundedRejectsInsufficientConfirmations covers the real gap
// found while scoping A006's mainnet gate (2026-09-30): a caller that
// requires more than the minimum depth (e.g. a large bond wanting 6
// confirmations, not 1) must actually get rejected below that
// threshold, not just below "unconfirmed."
func TestVerifyFundedRejectsInsufficientConfirmations(t *testing.T) {
	script := []byte{0x00, 0x01, 0x02}
	q := mockQuerier{scriptPubKey: script, valueSats: 50_000, confirmations: 2, found: true}
	err := VerifyFunded(context.Background(), q, "txid", 0, script, 50_000, 6)
	if !errors.Is(err, ErrNotFunded) {
		t.Errorf("VerifyFunded = %v, want ErrNotFunded (insufficient confirmations)", err)
	}
}

// TestVerifyFundedAcceptsExactlyMinConfirmations is the boundary case
// for the check above — confirmations == minConfirmations must pass,
// not just confirmations > minConfirmations.
func TestVerifyFundedAcceptsExactlyMinConfirmations(t *testing.T) {
	script := []byte{0x00, 0x01, 0x02}
	q := mockQuerier{scriptPubKey: script, valueSats: 50_000, confirmations: 6, found: true}
	if err := VerifyFunded(context.Background(), q, "txid", 0, script, 50_000, 6); err != nil {
		t.Errorf("VerifyFunded = %v, want success at exactly minConfirmations", err)
	}
}

// TestVerifyFundedRejectsInvalidMinConfirmations guards against
// silently treating a caller's minConfirmations<=0 as "mempool
// presence is enough" — exactly the risk A005/A006 required guarding
// against in the first place.
func TestVerifyFundedRejectsInvalidMinConfirmations(t *testing.T) {
	script := []byte{0x00, 0x01, 0x02}
	q := mockQuerier{scriptPubKey: script, valueSats: 50_000, confirmations: 100, found: true}
	if err := VerifyFunded(context.Background(), q, "txid", 0, script, 50_000, 0); err == nil {
		t.Error("VerifyFunded with minConfirmations=0: expected an error, got nil")
	}
	if err := VerifyFunded(context.Background(), q, "txid", 0, script, 50_000, -1); err == nil {
		t.Error("VerifyFunded with minConfirmations=-1: expected an error, got nil")
	}
}
