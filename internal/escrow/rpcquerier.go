package escrow

import (
	"context"
	"encoding/hex"
	"fmt"
	"math"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/wire"
)

// FirstMinedBlock is a regtest-only convenience so an integration
// test/caller can locate the earliest block's coinbase — which, after
// GenerateBlocks(101), is the first UTXO actually spendable (coinbase
// maturity is 100 confirmations).
func (q *RPCQuerier) FirstMinedBlock(hash *chainhash.Hash) (*wire.MsgTx, error) {
	block, err := q.client.GetBlock(hash)
	if err != nil {
		return nil, fmt.Errorf("escrow: getblock %s: %w", hash, err)
	}
	if len(block.Transactions) == 0 {
		return nil, fmt.Errorf("escrow: block %s has no transactions", hash)
	}
	return block.Transactions[0], nil
}

// RPCQuerier is a ChainQuerier backed by a real btcd/bitcoind JSON-RPC
// connection — the concrete backend verify.go's ChainQuerier
// interface was left abstract for, pending a reachable node (see
// plan/actions/A006 and A007's own Logs). Built and exercised against
// a real regtest btcd node built directly from this project's own
// already-vendored btcd module source (no new binary dependency).
type RPCQuerier struct {
	client *rpcclient.Client
}

// RPCConfig holds what's needed to reach a btcd/bitcoind RPC endpoint.
//
// Hardened 2026-10-02, closing the real gap A006's own mainnet-gate
// scoping found (2026-09-30) by reading rpcclient's actual dial code
// rather than trusting its doc comments: with Certificates left empty
// (this type's only option before this change), enabling TLS
// (DisableTLS: false) falls back to the system CA trust store, which
// a self-hosted btcd/bitcoind node's default self-signed cert never
// satisfies — a real TLS connection attempt against a normal node
// would simply fail. This type could, in effect, only ever connect
// with DisableTLS: true, regardless of intent.
//
//   - Certificates (below) fixes that directly — supply the node's own
//     cert (self-signed or real CA-issued) and TLS verification works.
//   - CookiePath (below) is bitcoind's/btcd's own standard production
//     auth mechanism — a node-managed, rotated credential file — used
//     instead of User/Pass when set (same precedence rpcclient.ConnConfig
//     itself documents). Avoids embedding a real credential in
//     application config at all, which User/Pass alone cannot.
//
// One accepted residual, stated rather than solved: User/Pass are
// plain Go strings, which can't be securely zeroed after use the way
// a []byte could. Real but minor, and not worth a bigger API change
// for this project's threat model — noted in A006's own Log rather
// than silently left unmentioned.
type RPCConfig struct {
	Host       string // "host:port", no scheme
	User       string
	Pass       string
	// CookiePath, if non-empty, is used instead of User/Pass — see
	// this type's own doc comment above.
	CookiePath string
	DisableTLS bool
	// Certificates is a PEM-encoded certificate chain for verifying the
	// node's TLS certificate. Has no effect when DisableTLS is true.
	// Required in practice for DisableTLS: false to work against a
	// self-hosted node's own default self-signed cert — see this
	// type's own doc comment above for why.
	Certificates []byte
}

// NewRPCQuerier connects to a btcd/bitcoind RPC endpoint over plain
// HTTP POST (no websocket notifications needed for GetTxOut alone).
func NewRPCQuerier(cfg RPCConfig) (*RPCQuerier, error) {
	client, err := rpcclient.New(&rpcclient.ConnConfig{
		Host:         cfg.Host,
		User:         cfg.User,
		Pass:         cfg.Pass,
		CookiePath:   cfg.CookiePath,
		HTTPPostMode: true,
		DisableTLS:   cfg.DisableTLS,
		Certificates: cfg.Certificates,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("escrow: connecting to RPC endpoint: %w", err)
	}
	return &RPCQuerier{client: client}, nil
}

// Shutdown closes the underlying RPC connection.
func (q *RPCQuerier) Shutdown() { q.client.Shutdown() }

// TxOut implements ChainQuerier against the real node. Reports the
// real confirmation count rather than collapsing "mempool only" and
// "confirmed" into one boolean — VerifyFunded's own minConfirmations
// parameter is where the actual required-depth policy decision lives
// (see that function's own doc comment for why 2026-09-30 moved it
// there instead of hardcoding it here).
func (q *RPCQuerier) TxOut(ctx context.Context, txid string, vout uint32) ([]byte, int64, int64, bool, error) {
	hash, err := chainhash.NewHashFromStr(txid)
	if err != nil {
		return nil, 0, 0, false, fmt.Errorf("escrow: invalid txid %q: %w", txid, err)
	}

	result, err := q.client.GetTxOut(hash, vout, false)
	if err != nil {
		return nil, 0, 0, false, fmt.Errorf("escrow: gettxout %s:%d: %w", txid, vout, err)
	}
	if result == nil {
		return nil, 0, 0, false, nil // spent or never existed
	}

	scriptPubKey, err := hex.DecodeString(result.ScriptPubKey.Hex)
	if err != nil {
		return nil, 0, 0, false, fmt.Errorf("escrow: decoding scriptPubKey hex: %w", err)
	}
	valueSats := int64(math.Round(result.Value * 1e8))
	return scriptPubKey, valueSats, result.Confirmations, true, nil
}

// GenerateBlocks is a thin regtest-only convenience wrapper (mining
// blocks so a funding transaction actually confirms) — not part of
// the ChainQuerier interface itself, since generating blocks isn't
// something a mainnet caller would ever do.
func (q *RPCQuerier) GenerateBlocks(numBlocks uint32) ([]*chainhash.Hash, error) {
	return q.client.Generate(numBlocks)
}

// SendRawTransaction broadcasts an already-signed transaction — a
// thin wrapper so callers don't need their own rpcclient import
// alongside this package's.
func (q *RPCQuerier) SendRawTransaction(tx *wire.MsgTx) (*chainhash.Hash, error) {
	return q.client.SendRawTransaction(tx, false)
}
