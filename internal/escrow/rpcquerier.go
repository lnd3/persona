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
// Regtest/testnet only, per A006's own mainnet gate — this type makes
// no attempt to be safe for a mainnet node's credentials.
type RPCConfig struct {
	Host       string // "host:port", no scheme
	User       string
	Pass       string
	DisableTLS bool
}

// NewRPCQuerier connects to a btcd/bitcoind RPC endpoint over plain
// HTTP POST (no websocket notifications needed for GetTxOut alone).
func NewRPCQuerier(cfg RPCConfig) (*RPCQuerier, error) {
	client, err := rpcclient.New(&rpcclient.ConnConfig{
		Host:         cfg.Host,
		User:         cfg.User,
		Pass:         cfg.Pass,
		HTTPPostMode: true,
		DisableTLS:   cfg.DisableTLS,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("escrow: connecting to RPC endpoint: %w", err)
	}
	return &RPCQuerier{client: client}, nil
}

// Shutdown closes the underlying RPC connection.
func (q *RPCQuerier) Shutdown() { q.client.Shutdown() }

// TxOut implements ChainQuerier against the real node. It treats an
// output with zero confirmations (still only in the mempool) the same
// as "not found," per A005/A006's "watching for confirmation, not
// just an unconfirmed mempool entry" requirement.
func (q *RPCQuerier) TxOut(ctx context.Context, txid string, vout uint32) ([]byte, int64, bool, error) {
	hash, err := chainhash.NewHashFromStr(txid)
	if err != nil {
		return nil, 0, false, fmt.Errorf("escrow: invalid txid %q: %w", txid, err)
	}

	result, err := q.client.GetTxOut(hash, vout, false)
	if err != nil {
		return nil, 0, false, fmt.Errorf("escrow: gettxout %s:%d: %w", txid, vout, err)
	}
	if result == nil {
		return nil, 0, false, nil // spent or never existed
	}
	if result.Confirmations < 1 {
		return nil, 0, false, nil // unconfirmed
	}

	scriptPubKey, err := hex.DecodeString(result.ScriptPubKey.Hex)
	if err != nil {
		return nil, 0, false, fmt.Errorf("escrow: decoding scriptPubKey hex: %w", err)
	}
	valueSats := int64(math.Round(result.Value * 1e8))
	return scriptPubKey, valueSats, true, nil
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
