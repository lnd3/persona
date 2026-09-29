package escrow

import (
	"context"
	"errors"
	"testing"
)

// mockQuerier is a trivial ChainQuerier for testing VerifyFunded's own
// logic in isolation from any real chain backend (see verify.go's own
// doc comment on why no live implementation exists yet).
type mockQuerier struct {
	scriptPubKey []byte
	valueSats    int64
	found        bool
	err          error
}

func (m mockQuerier) TxOut(ctx context.Context, txid string, vout uint32) ([]byte, int64, bool, error) {
	return m.scriptPubKey, m.valueSats, m.found, m.err
}

func TestVerifyFundedSucceeds(t *testing.T) {
	script := []byte{0x00, 0x01, 0x02}
	q := mockQuerier{scriptPubKey: script, valueSats: 50_000, found: true}
	if err := VerifyFunded(context.Background(), q, "txid", 0, script, 50_000); err != nil {
		t.Errorf("VerifyFunded: %v", err)
	}
}

func TestVerifyFundedRejectsUnfunded(t *testing.T) {
	q := mockQuerier{found: false}
	if err := VerifyFunded(context.Background(), q, "txid", 0, []byte{0x00}, 50_000); !errors.Is(err, ErrNotFunded) {
		t.Errorf("VerifyFunded = %v, want ErrNotFunded", err)
	}
}

func TestVerifyFundedRejectsUnderfunded(t *testing.T) {
	script := []byte{0x00, 0x01}
	q := mockQuerier{scriptPubKey: script, valueSats: 10_000, found: true}
	if err := VerifyFunded(context.Background(), q, "txid", 0, script, 50_000); !errors.Is(err, ErrNotFunded) {
		t.Errorf("VerifyFunded = %v, want ErrNotFunded (underfunded)", err)
	}
}

func TestVerifyFundedRejectsWrongScript(t *testing.T) {
	q := mockQuerier{scriptPubKey: []byte{0xAA}, valueSats: 50_000, found: true}
	if err := VerifyFunded(context.Background(), q, "txid", 0, []byte{0xBB}, 50_000); !errors.Is(err, ErrNotFunded) {
		t.Errorf("VerifyFunded = %v, want ErrNotFunded (wrong script)", err)
	}
}

func TestVerifyFundedPropagatesQueryError(t *testing.T) {
	wantErr := errors.New("node unreachable")
	q := mockQuerier{err: wantErr}
	err := VerifyFunded(context.Background(), q, "txid", 0, []byte{0x00}, 50_000)
	if err == nil || !errors.Is(err, wantErr) {
		t.Errorf("VerifyFunded = %v, want wrapping %v", err, wantErr)
	}
}
