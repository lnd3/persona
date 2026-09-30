package escrow

import (
	"context"
	"encoding/hex"
	"net"
	"testing"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/btcutil/psbt"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

// This file is A007's own regtest/testnet prerequisite: actually
// funding, spending, and confirming UniversalScript's and
// ReinforcedScript's branches against a *real* node — not just
// in-process engine validation (engine_test.go/settle_test.go) —
// before any independent review spends time on it. It requires a real
// btcd/bitcoind regtest node reachable at 127.0.0.1:18443 (rpcuser/
// rpcpass "test"/"test", --notls, --miningaddr set to a P2WPKH
// address whose private key is regtestMinerPrivHex below); it skips
// gracefully otherwise, the same pattern A001's live-relay test
// already established for this project. No node is reachable in most
// environments, including CI — this exists to be run deliberately,
// with a real node stood up, not as part of routine `go test ./...`.
//
// To stand up a matching node (built directly from this project's own
// already-vendored btcd module, no new binary dependency):
//
//	go install github.com/btcsuite/btcd@v0.24.2
//	btcd --regtest --notls --rpcuser=test --rpcpass=test \
//	  --rpclisten=127.0.0.1:18443 --miningaddr=<address for regtestMinerPrivHex>
//
// Deliberately not covered anywhere in this file: any branch's
// CSV-fallback path's live-node *timing*. BIP68's time-based relative
// lock needs real median-time-past to advance ~14 days, which isn't
// practical to wait out in a test, and btcd has no `setmocktime` RPC.
// That specific mechanism — comparing the script's required sequence
// against the spending input's actual sequence — is already exercised
// identically at the consensus-engine level in
// TestPreEscalationScript_*, the same code path a real node uses
// internally. A deliberate, stated scope boundary, not a silent gap.

const regtestMinerPrivHex = "b1373b5aaa815c051ef44f2e4bea525a8d09147213defda760f7eda5a03c8557"

func regtestReachable() bool {
	conn, err := net.DialTimeout("tcp", "127.0.0.1:18443", 500*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// regtestMiner is the shared, well-known regtest mining keypair/
// address every test in this file spends coinbases from.
type regtestMiner struct {
	priv     *btcec.PrivateKey
	pub      []byte
	pkScript []byte
}

func newRegtestMiner(t *testing.T) regtestMiner {
	t.Helper()
	privBytes, err := hex.DecodeString(regtestMinerPrivHex)
	if err != nil {
		t.Fatalf("decoding miner privkey: %v", err)
	}
	priv, _ := btcec.PrivKeyFromBytes(privBytes)
	pub := priv.PubKey().SerializeCompressed()
	addr, err := btcutil.NewAddressWitnessPubKeyHash(btcutil.Hash160(pub), &chaincfg.RegressionNetParams)
	if err != nil {
		t.Fatalf("NewAddressWitnessPubKeyHash: %v", err)
	}
	pkScript, err := txscript.PayToAddrScript(addr)
	if err != nil {
		t.Fatalf("PayToAddrScript: %v", err)
	}
	return regtestMiner{priv: priv, pub: pub, pkScript: pkScript}
}

// regtestFundFreshOutput mines a brand-new coinbase, matures it (100
// confirmations), and spends *half of its actual current value*
// straight into a new output paying scriptPubKey — self-contained, no
// shared/stale state across calls. The funded amount is deliberately
// relative to the coinbase's own freshly-fetched value, not a fixed
// absolute number: regtest's subsidy actually halves every 150 blocks
// (chaincfg.RegressionNetParams.SubsidyReductionInterval), and this
// file's own tests mine hundreds of blocks per run — a fixed absolute
// `fundAmount` constant works the first few times against a
// long-lived node and then starts failing with "inputs less than
// amount spent" once enough halvings have passed, exactly the kind of
// test-suite flakiness a real committed test must not have. Returns
// the actual funded amount alongside the outpoint/output so callers
// can compute their own settlement fee against it.
//
// This still isn't unbounded: run enough full test passes against the
// same never-reset node (dozens, each mining ~300-700 blocks) and even
// half the current coinbase eventually gets small enough to trip
// btcd's mempool free-transaction "priority" policy threshold, or
// small enough that half of it is less than the settlement fee.
// Neither is a bug in the code under test — it's regtest's own
// aggressive 150-block subsidy-halving interval doing exactly what
// it's supposed to on a node nobody ever resets. Wipe the node's
// datadir and restart if this file's tests start failing this way.
func regtestFundFreshOutput(t *testing.T, q *RPCQuerier, miner regtestMiner, scriptPubKey []byte) (*wire.OutPoint, *wire.TxOut, int64) {
	t.Helper()
	hashes, err := q.GenerateBlocks(1)
	if err != nil {
		t.Fatalf("GenerateBlocks(1): %v", err)
	}
	coinbaseTx, err := q.FirstMinedBlock(hashes[0])
	if err != nil {
		t.Fatalf("FirstMinedBlock: %v", err)
	}
	if _, err := q.GenerateBlocks(100); err != nil {
		t.Fatalf("GenerateBlocks(100) to mature: %v", err)
	}

	coinbaseTxHash := coinbaseTx.TxHash()
	var vout uint32
	var value int64
	found := false
	for i, out := range coinbaseTx.TxOut {
		if string(out.PkScript) == string(miner.pkScript) {
			vout, value, found = uint32(i), out.Value, true
			break
		}
	}
	if !found {
		t.Fatal("fresh coinbase output paying the miner address not found")
	}
	amount := value / 2 // always well within this specific coinbase's own actual value

	fundingTx := wire.NewMsgTx(2)
	fundingTx.AddTxIn(&wire.TxIn{PreviousOutPoint: wire.OutPoint{Hash: coinbaseTxHash, Index: vout}})
	fundingTx.AddTxOut(&wire.TxOut{Value: amount, PkScript: scriptPubKey})
	fetcher := txscript.NewCannedPrevOutputFetcher(miner.pkScript, value)
	sigHashes := txscript.NewTxSigHashes(fundingTx, fetcher)
	sig, err := txscript.RawTxInWitnessSignature(fundingTx, sigHashes, 0, value, miner.pkScript, txscript.SigHashAll, miner.priv)
	if err != nil {
		t.Fatalf("RawTxInWitnessSignature: %v", err)
	}
	fundingTx.TxIn[0].Witness = wire.TxWitness{sig, miner.pub}

	fundTxHash, err := q.SendRawTransaction(fundingTx)
	if err != nil {
		t.Fatalf("broadcasting funding tx: %v", err)
	}
	if _, err := q.GenerateBlocks(1); err != nil {
		t.Fatalf("confirming funding tx: %v", err)
	}

	spk, val, _, ok, err := q.TxOut(context.Background(), fundTxHash.String(), 0)
	if err != nil {
		t.Fatalf("TxOut on the real funding output: %v", err)
	}
	if !ok {
		t.Fatal("expected the funding output to be confirmed on-chain")
	}
	if string(spk) != string(scriptPubKey) || val != amount {
		t.Fatalf("confirmed funding output = (script matches: %v, value: %d), want (true, %d)", string(spk) == string(scriptPubKey), val, amount)
	}
	return &wire.OutPoint{Hash: *fundTxHash, Index: 0}, &wire.TxOut{Value: amount, PkScript: scriptPubKey}, amount
}

// regtestSettleAndConfirm builds a settlement PSBT spending prevOut,
// signs it with signers, finalizes it via finalize (a closure over
// whichever of FinalizeUniversal/FinalizeReinforced and its specific
// arguments applies), and broadcasts + confirms the result against a
// real node — failing the test loudly if the real mempool rejects it
// or it never confirms.
func regtestSettleAndConfirm(
	t *testing.T, q *RPCQuerier, miner regtestMiner,
	prevOut *wire.OutPoint, prevTxOut *wire.TxOut, witnessScript []byte,
	payoutAmount int64, signers []*btcec.PrivateKey, finalize func(*psbt.Packet) error,
) {
	t.Helper()
	payouts := []*wire.TxOut{{Value: payoutAmount, PkScript: miner.pkScript}}
	p, err := NewSettlementPacket(*prevOut, prevTxOut, witnessScript, payouts, 0)
	if err != nil {
		t.Fatalf("NewSettlementPacket: %v", err)
	}
	for _, s := range signers {
		if err := SignSettlementInput(p, s); err != nil {
			t.Fatalf("SignSettlementInput: %v", err)
		}
	}
	if err := finalize(p); err != nil {
		t.Fatalf("finalize: %v", err)
	}
	settleTx, err := ExtractSettlement(p)
	if err != nil {
		t.Fatalf("ExtractSettlement: %v", err)
	}
	settleTxHash, err := q.SendRawTransaction(settleTx)
	if err != nil {
		t.Fatalf("a real node's mempool rejected the settlement tx: %v", err)
	}
	if _, err := q.GenerateBlocks(1); err != nil {
		t.Fatalf("confirming settlement tx: %v", err)
	}
	_, _, _, settled, err := q.TxOut(context.Background(), settleTxHash.String(), 0)
	if err != nil {
		t.Fatalf("TxOut on the settlement output: %v", err)
	}
	if !settled {
		t.Error("expected the settlement transaction's own output to be confirmed on-chain")
	}
}

func TestRegtest_UniversalScriptBranches(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live regtest test in -short mode")
	}
	if !regtestReachable() {
		t.Skip("no regtest node reachable at 127.0.0.1:18443, skipping live test")
	}

	q, err := NewRPCQuerier(RPCConfig{Host: "127.0.0.1:18443", User: "test", Pass: "test", DisableTLS: true})
	if err != nil {
		t.Fatalf("NewRPCQuerier: %v", err)
	}
	defer q.Shutdown()

	miner := newRegtestMiner(t)
	attesterPriv, _ := btcec.NewPrivateKey()
	subjectPriv, _ := btcec.NewPrivateKey()
	arbiterPriv, _ := btcec.NewPrivateKey()
	attesterPub := attesterPriv.PubKey().SerializeCompressed()
	subjectPub := subjectPriv.PubKey().SerializeCompressed()
	arbiterPub := arbiterPriv.PubKey().SerializeCompressed()

	sequence, err := SequenceForDays(14)
	if err != nil {
		t.Fatalf("SequenceForDays: %v", err)
	}
	witnessScript, err := UniversalScript(attesterPub, subjectPub, arbiterPub, sequence)
	if err != nil {
		t.Fatalf("UniversalScript: %v", err)
	}
	scriptPubKey, err := WitnessScriptHash(witnessScript)
	if err != nil {
		t.Fatalf("WitnessScriptHash: %v", err)
	}

	const fee = int64(1000)

	finalize := func(p *psbt.Packet) error {
		return FinalizeUniversal(p, witnessScript, attesterPub, subjectPub, arbiterPub, sequence)
	}

	t.Run("branch 1: mutual settlement", func(t *testing.T) {
		prevOut, prevTxOut, amount := regtestFundFreshOutput(t, q, miner, scriptPubKey)
		regtestSettleAndConfirm(t, q, miner, prevOut, prevTxOut, witnessScript, amount-fee,
			[]*btcec.PrivateKey{attesterPriv, subjectPriv}, finalize)
	})

	t.Run("branch 2: arbiter sides with attester", func(t *testing.T) {
		prevOut, prevTxOut, amount := regtestFundFreshOutput(t, q, miner, scriptPubKey)
		regtestSettleAndConfirm(t, q, miner, prevOut, prevTxOut, witnessScript, amount-fee,
			[]*btcec.PrivateKey{attesterPriv, arbiterPriv}, finalize)
	})

	t.Run("branch 3: arbiter sides with subject", func(t *testing.T) {
		prevOut, prevTxOut, amount := regtestFundFreshOutput(t, q, miner, scriptPubKey)
		regtestSettleAndConfirm(t, q, miner, prevOut, prevTxOut, witnessScript, amount-fee,
			[]*btcec.PrivateKey{subjectPriv, arbiterPriv}, finalize)
	})
}

// TestRegtest_ReinforcedScriptBranches exercises ReinforcedScript's
// first four branches against a real node — mutual settlement, both
// independent arbiters agreeing (no disputant needed), and each
// arbiter siding with their own side's disputant. Branch 5 (the CSV
// fallback) is skipped live for the same reason as every other
// fallback branch in this file — see the file's own top-level doc
// comment.
func TestRegtest_ReinforcedScriptBranches(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live regtest test in -short mode")
	}
	if !regtestReachable() {
		t.Skip("no regtest node reachable at 127.0.0.1:18443, skipping live test")
	}

	q, err := NewRPCQuerier(RPCConfig{Host: "127.0.0.1:18443", User: "test", Pass: "test", DisableTLS: true})
	if err != nil {
		t.Fatalf("NewRPCQuerier: %v", err)
	}
	defer q.Shutdown()

	miner := newRegtestMiner(t)
	attesterPriv, _ := btcec.NewPrivateKey()
	subjectPriv, _ := btcec.NewPrivateKey()
	attesterArbiterPriv, _ := btcec.NewPrivateKey() // attester's own standing arbiter
	subjectArbiterPriv, _ := btcec.NewPrivateKey()  // subject's own standing arbiter
	attesterPub := attesterPriv.PubKey().SerializeCompressed()
	subjectPub := subjectPriv.PubKey().SerializeCompressed()
	attesterArbiterPub := attesterArbiterPriv.PubKey().SerializeCompressed()
	subjectArbiterPub := subjectArbiterPriv.PubKey().SerializeCompressed()

	sequence, err := SequenceForDays(14)
	if err != nil {
		t.Fatalf("SequenceForDays: %v", err)
	}
	witnessScript, err := ReinforcedScript(attesterPub, subjectPub, attesterArbiterPub, subjectArbiterPub, sequence)
	if err != nil {
		t.Fatalf("ReinforcedScript: %v", err)
	}
	scriptPubKey, err := WitnessScriptHash(witnessScript)
	if err != nil {
		t.Fatalf("WitnessScriptHash: %v", err)
	}

	const fee = int64(1000)

	finalize := func(p *psbt.Packet) error {
		return FinalizeReinforced(p, witnessScript, attesterPub, subjectPub, attesterArbiterPub, subjectArbiterPub, sequence)
	}

	t.Run("branch 1: mutual settlement", func(t *testing.T) {
		prevOut, prevTxOut, amount := regtestFundFreshOutput(t, q, miner, scriptPubKey)
		regtestSettleAndConfirm(t, q, miner, prevOut, prevTxOut, witnessScript, amount-fee,
			[]*btcec.PrivateKey{attesterPriv, subjectPriv}, finalize)
	})

	t.Run("branch 2: both independent arbiters agree, no disputant needed", func(t *testing.T) {
		prevOut, prevTxOut, amount := regtestFundFreshOutput(t, q, miner, scriptPubKey)
		regtestSettleAndConfirm(t, q, miner, prevOut, prevTxOut, witnessScript, amount-fee,
			[]*btcec.PrivateKey{attesterArbiterPriv, subjectArbiterPriv}, finalize)
	})

	t.Run("branch 3: attester's own arbiter sides with attester", func(t *testing.T) {
		prevOut, prevTxOut, amount := regtestFundFreshOutput(t, q, miner, scriptPubKey)
		regtestSettleAndConfirm(t, q, miner, prevOut, prevTxOut, witnessScript, amount-fee,
			[]*btcec.PrivateKey{attesterPriv, attesterArbiterPriv}, finalize)
	})

	t.Run("branch 4: subject's own arbiter sides with subject", func(t *testing.T) {
		prevOut, prevTxOut, amount := regtestFundFreshOutput(t, q, miner, scriptPubKey)
		regtestSettleAndConfirm(t, q, miner, prevOut, prevTxOut, witnessScript, amount-fee,
			[]*btcec.PrivateKey{subjectPriv, subjectArbiterPriv}, finalize)
	})
}

// TestRegtest_VerifyFundedEnforcesRealConfirmationDepth is A006's own
// mainnet-gate follow-up (2026-09-30): VerifyFunded's minConfirmations
// parameter, exercised against a real node's real confirmation count,
// not just the mock ChainQuerier verify_test.go already covers. Funds
// a fresh output with exactly one confirmation, confirms a verifier
// requiring more is correctly rejected, then mines further and
// confirms the same call succeeds once the real depth catches up.
func TestRegtest_VerifyFundedEnforcesRealConfirmationDepth(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live regtest test in -short mode")
	}
	if !regtestReachable() {
		t.Skip("no regtest node reachable at 127.0.0.1:18443, skipping live test")
	}

	q, err := NewRPCQuerier(RPCConfig{Host: "127.0.0.1:18443", User: "test", Pass: "test", DisableTLS: true})
	if err != nil {
		t.Fatalf("NewRPCQuerier: %v", err)
	}
	defer q.Shutdown()

	miner := newRegtestMiner(t)
	attesterPriv, _ := btcec.NewPrivateKey()
	subjectPriv, _ := btcec.NewPrivateKey()
	arbiterPriv, _ := btcec.NewPrivateKey()
	attesterPub := attesterPriv.PubKey().SerializeCompressed()
	subjectPub := subjectPriv.PubKey().SerializeCompressed()
	arbiterPub := arbiterPriv.PubKey().SerializeCompressed()

	sequence, err := SequenceForDays(14)
	if err != nil {
		t.Fatalf("SequenceForDays: %v", err)
	}
	witnessScript, err := UniversalScript(attesterPub, subjectPub, arbiterPub, sequence)
	if err != nil {
		t.Fatalf("UniversalScript: %v", err)
	}
	scriptPubKey, err := WitnessScriptHash(witnessScript)
	if err != nil {
		t.Fatalf("WitnessScriptHash: %v", err)
	}

	// regtestFundFreshOutput already mines 1 block to confirm the
	// funding tx (see its own doc comment) — so at this point the
	// output has exactly 1 real confirmation.
	prevOut, _, amount := regtestFundFreshOutput(t, q, miner, scriptPubKey)

	// A verifier requiring 3 confirmations must be rejected at 1.
	err = VerifyFunded(context.Background(), q, prevOut.Hash.String(), prevOut.Index, scriptPubKey, amount, 3)
	if err == nil {
		t.Fatal("VerifyFunded with minConfirmations=3 at 1 real confirmation: expected an error, got nil")
	}

	// A verifier requiring exactly 1 confirmation must already pass.
	if err := VerifyFunded(context.Background(), q, prevOut.Hash.String(), prevOut.Index, scriptPubKey, amount, 1); err != nil {
		t.Errorf("VerifyFunded with minConfirmations=1 at 1 real confirmation: %v", err)
	}

	// Mine 2 more blocks (3 total) — now the 3-confirmation requirement
	// must pass too.
	if _, err := q.GenerateBlocks(2); err != nil {
		t.Fatalf("GenerateBlocks(2): %v", err)
	}
	if err := VerifyFunded(context.Background(), q, prevOut.Hash.String(), prevOut.Index, scriptPubKey, amount, 3); err != nil {
		t.Errorf("VerifyFunded with minConfirmations=3 at 3 real confirmations: %v", err)
	}
}
