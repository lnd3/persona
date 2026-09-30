package payment

import (
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lnd3/persona/internal/attestation"
	"github.com/lnd3/persona/internal/identity"
)

// This file is a template use case: one cohesive test demonstrating
// the actual intended usage pattern of every piece A004 built
// (l402.go, receipt.go, event.go, server.go) as a single real-world
// flow, rather than each piece's own isolated unit tests. It fronts
// the real gateway handler (NewGatewayHandler, server.go) with a
// *simulated* L402 gate — not a real Aperture instance, no real
// macaroon minting, no real Lightning payment — so it proves
// persona's own code is wired correctly without needing that heavier
// infrastructure. A real Aperture + Lightning-regtest integration is
// deliberately a separate, much larger undertaking — see A008's own
// plan file for why and what it would take.
//
// The simulated gate: on a request with no (or the wrong)
// Authorization header, respond 402 with a real L402-shaped
// WWW-Authenticate challenge; on a request whose Authorization header
// exactly matches AuthorizationHeader(challenge.Macaroon,
// simulatedPreimageHex), forward to the real gateway handler. A real
// Aperture instance would instead cryptographically verify the
// preimage hashes to the invoice's own payment hash and the macaroon
// is genuinely valid — this simulation stands in for that check so
// the *client-side* code (challenge parsing, header construction) and
// the *gateway* code (receipt signing) can both be exercised for real.
const simulatedMacaroon = "AGIAJEem_simulated_macaroon_not_real"
const simulatedInvoice = "lnbc1_simulated_invoice_not_real"
const simulatedPreimageHex = "deadbeefcafebabe0000000000000000000000000000000000000000000000"

func newSimulatedAppertureGateway(t *testing.T, gateway identity.Persona) *httptest.Server {
	t.Helper()
	realGateway := NewGatewayHandler(gateway)
	expectedAuth := AuthorizationHeader(simulatedMacaroon, simulatedPreimageHex)

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != expectedAuth {
			w.Header().Set("WWW-Authenticate", `LSAT macaroon="`+simulatedMacaroon+`", invoice="`+simulatedInvoice+`"`)
			w.WriteHeader(http.StatusPaymentRequired)
			return
		}
		realGateway.ServeHTTP(w, r)
	}))
}

// TestTemplate_AttestationCostGatewayFlow walks through the entire
// intended real-world usage pattern: an attester wants to publish a
// paid attestation, requests a receipt from a (simulated)
// Aperture-fronted gateway, embeds the receipt in the finalized
// attestation event, and a third-party verifier — using nothing but
// the fetched event itself — independently confirms both the
// attestation's own signature and the receipt's validity.
func TestTemplate_AttestationCostGatewayFlow(t *testing.T) {
	gateway, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New(gateway): %v", err)
	}
	attester, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New(attester): %v", err)
	}
	subject, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New(subject): %v", err)
	}

	srv := newSimulatedAppertureGateway(t, gateway)
	defer srv.Close()

	// Step 1: the attester fixes the claim's content and timestamp —
	// this is what the receipt will actually be signed over, and it
	// can never be re-rolled afterward without invalidating the
	// receipt.
	pending, err := NewPendingClaim(attester.PublicKey, subject.PublicKey, "net.persona.core.skill", "go")
	if err != nil {
		t.Fatalf("NewPendingClaim: %v", err)
	}
	hash := pending.Hash()

	// Step 2: request a receipt without having paid yet — the
	// (simulated) gateway, fronted by (simulated) Aperture, responds
	// with a real L402 402 challenge.
	resp, err := http.Post(srv.URL, "text/plain", hexReader(hash))
	if err != nil {
		t.Fatalf("initial request: %v", err)
	}
	defer resp.Body.Close()

	challenge, err := ChallengeFromResponse(resp)
	if err != nil {
		t.Fatalf("ChallengeFromResponse: %v", err)
	}
	if challenge.Macaroon != simulatedMacaroon || challenge.Invoice != simulatedInvoice {
		t.Fatalf("parsed challenge = %+v, want macaroon/invoice from the simulated gate", challenge)
	}

	// Step 3: pay the invoice. This is the one step this template
	// doesn't do for real — see this file's own doc comment and A008's
	// plan file for what a real payment integration needs instead.
	preimageHex := simulatedPreimageHex // stand-in for "wallet paid the invoice, here's the preimage"

	// Step 4: retry with the paid Authorization header — the
	// (simulated) gate now lets the request through to the real
	// gateway handler, which signs and returns a real Receipt.
	req, err := http.NewRequest(http.MethodPost, srv.URL, hexReader(hash))
	if err != nil {
		t.Fatalf("building retry request: %v", err)
	}
	req.Header.Set("Authorization", AuthorizationHeader(challenge.Macaroon, preimageHex))
	resp2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("retry request: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("retry status = %d, want 200", resp2.StatusCode)
	}
	var receipt Receipt
	if err := json.NewDecoder(resp2.Body).Decode(&receipt); err != nil {
		t.Fatalf("decoding receipt: %v", err)
	}
	if err := VerifyReceipt(receipt, hash); err != nil {
		t.Fatalf("VerifyReceipt: %v", err)
	}

	// Step 5: finalize and publish the attestation event, with the
	// receipt embedded — this is what actually gets signed and (in a
	// real deployment) sent to relays.
	evt, err := pending.Finalize(attester, "go skill, paid", &receipt)
	if err != nil {
		t.Fatalf("Finalize: %v", err)
	}

	// Step 6: a third-party verifier fetches the event later and
	// checks it from scratch — no access to `pending` or the original
	// `hash`, only the event itself.
	claim, err := attestation.Verify(evt)
	if err != nil {
		t.Fatalf("a verifier's attestation.Verify: %v", err)
	}
	verifierReceipt, ok, err := ExtractReceipt(evt)
	if err != nil {
		t.Fatalf("a verifier's ExtractReceipt: %v", err)
	}
	if !ok {
		t.Fatal("expected a verifier to find a receipt on this event")
	}
	if err := VerifyReceipt(verifierReceipt, ClaimContentHash(claim)); err != nil {
		t.Fatalf("a verifier's VerifyReceipt against the independently recomputed claim hash: %v", err)
	}

	// The verifier's own trust decision — does *this* verifier trust
	// *this* gateway's pubkey enough to weight the claim as paid-for —
	// is deliberately not this template's job; see A004's own
	// Decisions on why that's a per-verifier call, not baked into
	// VerifyReceipt.
	if verifierReceipt.GatewayPubkey != gateway.PublicKey {
		t.Errorf("receipt.GatewayPubkey = %s, want %s", verifierReceipt.GatewayPubkey, gateway.PublicKey)
	}
}

func hexReader(hash [32]byte) *strings.Reader {
	return strings.NewReader(hex.EncodeToString(hash[:]))
}
