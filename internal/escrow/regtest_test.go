package escrow

import (
	"context"
	"encoding/hex"
	"net"
	"testing"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

// This file is A007's own regtest/testnet prerequisite: actually
// funding, spending, and confirming UniversalScript's branches
// against a *real* node — not just in-process engine validation
// (engine_test.go/settle_test.go) — before any independent review
// spends time on it. It requires a real btcd/bitcoind regtest node
// reachable at 127.0.0.1:18443 (rpcuser/rpcpass "test"/"test",
// --notls, --miningaddr set to a P2WPKH address whose private key is
// regtestMinerPrivHex below); it skips gracefully otherwise, the same
// pattern A001's live-relay test already established for this
// project. No node is reachable in most environments, including CI —
// this exists to be run deliberately, with a real node stood up, not
// as part of routine `go test ./...`.
//
// To stand up a matching node (built directly from this project's own
// already-vendored btcd module, no new binary dependency):
//
//	go install github.com/btcsuite/btcd@v0.24.2
//	btcd --regtest --notls --rpcuser=test --rpcpass=test \
//	  --rpclisten=127.0.0.1:18443 --miningaddr=<address for regtestMinerPrivHex>

const regtestMinerPrivHex = "b1373b5aaa815c051ef44f2e4bea525a8d09147213defda760f7eda5a03c8557"

func regtestReachable() bool {
	conn, err := net.DialTimeout("tcp", "127.0.0.1:18443", 500*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func TestRegtest_UniversalScriptMutualSettlement(t *testing.T) {
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

	minerPrivBytes, err := hex.DecodeString(regtestMinerPrivHex)
	if err != nil {
		t.Fatalf("decoding miner privkey: %v", err)
	}
	minerPriv, _ := btcec.PrivKeyFromBytes(minerPrivBytes)
	minerPub := minerPriv.PubKey().SerializeCompressed()
	minerHash160 := btcutil.Hash160(minerPub)
	minerAddr, err := btcutil.NewAddressWitnessPubKeyHash(minerHash160, &chaincfg.RegressionNetParams)
	if err != nil {
		t.Fatalf("NewAddressWitnessPubKeyHash: %v", err)
	}
	minerPkScript, err := txscript.PayToAddrScript(minerAddr)
	if err != nil {
		t.Fatalf("PayToAddrScript: %v", err)
	}

	// Mine 101 blocks so the first one's coinbase matures.
	hashes, err := q.GenerateBlocks(101)
	if err != nil {
		t.Fatalf("GenerateBlocks: %v", err)
	}
	coinbaseTx, err := q.FirstMinedBlock(hashes[0])
	if err != nil {
		t.Fatalf("FirstMinedBlock: %v", err)
	}
	coinbaseTxHash := coinbaseTx.TxHash()
	var coinbaseVout uint32
	var coinbaseValue int64
	found := false
	for i, out := range coinbaseTx.TxOut {
		if string(out.PkScript) == string(minerPkScript) {
			coinbaseVout, coinbaseValue, found = uint32(i), out.Value, true
			break
		}
	}
	if !found {
		t.Fatal("coinbase output paying the miner address not found")
	}

	// Build a UniversalScript escrow output.
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

	const fundAmount = int64(50_000_000)
	const fee = int64(1000)

	// Fund the escrow output for real, from the matured coinbase.
	fundingTx := wire.NewMsgTx(2)
	fundingTx.AddTxIn(&wire.TxIn{PreviousOutPoint: wire.OutPoint{Hash: coinbaseTxHash, Index: coinbaseVout}})
	fundingTx.AddTxOut(&wire.TxOut{Value: fundAmount, PkScript: scriptPubKey})
	fundingTx.AddTxOut(&wire.TxOut{Value: coinbaseValue - fundAmount - fee, PkScript: minerPkScript})

	fetcher := txscript.NewCannedPrevOutputFetcher(minerPkScript, coinbaseValue)
	sigHashes := txscript.NewTxSigHashes(fundingTx, fetcher)
	sig, err := txscript.RawTxInWitnessSignature(fundingTx, sigHashes, 0, coinbaseValue, minerPkScript, txscript.SigHashAll, minerPriv)
	if err != nil {
		t.Fatalf("RawTxInWitnessSignature: %v", err)
	}
	fundingTx.TxIn[0].Witness = wire.TxWitness{sig, minerPub}

	fundingTxHash, err := q.SendRawTransaction(fundingTx)
	if err != nil {
		t.Fatalf("broadcasting funding tx: %v", err)
	}
	if _, err := q.GenerateBlocks(1); err != nil {
		t.Fatalf("confirming funding tx: %v", err)
	}

	spk, val, ok, err := q.TxOut(context.Background(), fundingTxHash.String(), 0)
	if err != nil {
		t.Fatalf("TxOut on the real funding output: %v", err)
	}
	if !ok {
		t.Fatal("expected the funding output to be confirmed on-chain")
	}
	if string(spk) != string(scriptPubKey) {
		t.Error("confirmed output's scriptPubKey doesn't match the escrow script")
	}
	if val != fundAmount {
		t.Errorf("confirmed value = %d, want %d", val, fundAmount)
	}

	// Settle via branch 1 (mutual settlement) and broadcast for real.
	prevOut := wire.OutPoint{Hash: *fundingTxHash, Index: 0}
	prevTxOut := &wire.TxOut{Value: fundAmount, PkScript: scriptPubKey}
	payouts := []*wire.TxOut{{Value: fundAmount - fee, PkScript: minerPkScript}}

	p, err := NewSettlementPacket(prevOut, prevTxOut, witnessScript, payouts, 0)
	if err != nil {
		t.Fatalf("NewSettlementPacket: %v", err)
	}
	if err := SignSettlementInput(p, attesterPriv); err != nil {
		t.Fatalf("SignSettlementInput(attester): %v", err)
	}
	if err := SignSettlementInput(p, subjectPriv); err != nil {
		t.Fatalf("SignSettlementInput(subject): %v", err)
	}
	if err := FinalizeUniversal(p, witnessScript, attesterPub, subjectPub, arbiterPub, sequence); err != nil {
		t.Fatalf("FinalizeUniversal: %v", err)
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

	_, _, settled, err := q.TxOut(context.Background(), settleTxHash.String(), 0)
	if err != nil {
		t.Fatalf("TxOut on the settlement output: %v", err)
	}
	if !settled {
		t.Error("expected the settlement transaction's own output to be confirmed on-chain")
	}
}

// TestRegtest_UniversalScriptArbiterBranches funds two more escrow
// outputs, each from its own freshly-mined and matured coinbase, and
// settles them via branches 2 and 3 (arbiter sides with attester,
// arbiter sides with subject) — cheap to exercise live, since neither
// needs any real elapsed time, unlike branch 4 (see this file's own
// doc comment on why the CSV fallback branch's live-node timing isn't
// covered here). Each coinbase's own value is fetched fresh rather
// than reused across spends, since regtest's subsidy actually halves
// every 150 blocks (chaincfg.RegressionNetParams.SubsidyReductionInterval)
// — reusing a stale value would produce an invalid signature amount.
func TestRegtest_UniversalScriptArbiterBranches(t *testing.T) {
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

	minerPrivBytes, err := hex.DecodeString(regtestMinerPrivHex)
	if err != nil {
		t.Fatalf("decoding miner privkey: %v", err)
	}
	minerPriv, _ := btcec.PrivKeyFromBytes(minerPrivBytes)
	minerPub := minerPriv.PubKey().SerializeCompressed()
	minerHash160 := btcutil.Hash160(minerPub)
	minerAddr, err := btcutil.NewAddressWitnessPubKeyHash(minerHash160, &chaincfg.RegressionNetParams)
	if err != nil {
		t.Fatalf("NewAddressWitnessPubKeyHash: %v", err)
	}
	minerPkScript, err := txscript.PayToAddrScript(minerAddr)
	if err != nil {
		t.Fatalf("PayToAddrScript: %v", err)
	}

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

	const fundAmount = int64(10_000_000)
	const fee = int64(1000)

	// fundFreshEscrowOutput mines a brand-new coinbase, matures it,
	// and spends it straight into a new escrow output — self-
	// contained, no shared/stale state across calls.
	fundFreshEscrowOutput := func(t *testing.T) (*wire.OutPoint, *wire.TxOut) {
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
			if string(out.PkScript) == string(minerPkScript) {
				vout, value, found = uint32(i), out.Value, true
				break
			}
		}
		if !found {
			t.Fatal("fresh coinbase output paying the miner address not found")
		}

		fundingTx := wire.NewMsgTx(2)
		fundingTx.AddTxIn(&wire.TxIn{PreviousOutPoint: wire.OutPoint{Hash: coinbaseTxHash, Index: vout}})
		fundingTx.AddTxOut(&wire.TxOut{Value: fundAmount, PkScript: scriptPubKey})
		fetcher := txscript.NewCannedPrevOutputFetcher(minerPkScript, value)
		sigHashes := txscript.NewTxSigHashes(fundingTx, fetcher)
		sig, err := txscript.RawTxInWitnessSignature(fundingTx, sigHashes, 0, value, minerPkScript, txscript.SigHashAll, minerPriv)
		if err != nil {
			t.Fatalf("RawTxInWitnessSignature: %v", err)
		}
		fundingTx.TxIn[0].Witness = wire.TxWitness{sig, minerPub}

		fundTxHash, err := q.SendRawTransaction(fundingTx)
		if err != nil {
			t.Fatalf("broadcasting funding tx: %v", err)
		}
		if _, err := q.GenerateBlocks(1); err != nil {
			t.Fatalf("confirming funding tx: %v", err)
		}
		return &wire.OutPoint{Hash: *fundTxHash, Index: 0}, &wire.TxOut{Value: fundAmount, PkScript: scriptPubKey}
	}

	settleAndBroadcast := func(t *testing.T, prevOut *wire.OutPoint, prevTxOut *wire.TxOut, signers []*btcec.PrivateKey) {
		t.Helper()
		payouts := []*wire.TxOut{{Value: fundAmount - fee, PkScript: minerPkScript}}
		p, err := NewSettlementPacket(*prevOut, prevTxOut, witnessScript, payouts, 0)
		if err != nil {
			t.Fatalf("NewSettlementPacket: %v", err)
		}
		for _, s := range signers {
			if err := SignSettlementInput(p, s); err != nil {
				t.Fatalf("SignSettlementInput: %v", err)
			}
		}
		if err := FinalizeUniversal(p, witnessScript, attesterPub, subjectPub, arbiterPub, sequence); err != nil {
			t.Fatalf("FinalizeUniversal: %v", err)
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
		_, _, settled, err := q.TxOut(context.Background(), settleTxHash.String(), 0)
		if err != nil {
			t.Fatalf("TxOut on the settlement output: %v", err)
		}
		if !settled {
			t.Error("expected the settlement transaction's own output to be confirmed on-chain")
		}
	}

	t.Run("branch 2: arbiter sides with attester", func(t *testing.T) {
		prevOut, prevTxOut := fundFreshEscrowOutput(t)
		settleAndBroadcast(t, prevOut, prevTxOut, []*btcec.PrivateKey{attesterPriv, arbiterPriv})
	})

	t.Run("branch 3: arbiter sides with subject", func(t *testing.T) {
		prevOut, prevTxOut := fundFreshEscrowOutput(t)
		settleAndBroadcast(t, prevOut, prevTxOut, []*btcec.PrivateKey{subjectPriv, arbiterPriv})
	})
}
