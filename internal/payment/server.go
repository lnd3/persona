package payment

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/lnd3/persona/internal/identity"
)

// maxRequestBody bounds the receipt-mint request body: a hex-encoded
// sha256 is always 64 bytes, a generous cap avoids reading an
// unbounded body from a misbehaving or malicious caller.
const maxRequestBody = 4096

// NewGatewayHandler builds the reference attestation-cost gateway's
// only HTTP surface, per A004's Decisions: POST /receipt, body = the
// content hash as hex, response = the signed Receipt as JSON.
//
// This handler does no payment verification of any kind — it trusts
// that Aperture, sitting in front of it (per cinder's D005 shape,
// registering this handler as one fixed-price backend service), has
// already gated the request on a valid L402 payment before it ever
// reaches here. No macaroon, invoice, or Lightning code exists
// anywhere in this package, matching D005's own result for cinder.
//
// Deployment note: run this behind Aperture exactly as D005 describes
// cinder's own paid listener — bound to a private address, fronted
// publicly only through `aperturecli services create --name
// persona-attestation-receipt --address <this handler's addr> --price
// <sats>`. This project doesn't re-derive that shape; see D005.
func NewGatewayHandler(gateway identity.Persona) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		hash, err := readContentHash(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		receipt, err := SignReceipt(gateway, hash)
		if err != nil {
			http.Error(w, "failed to sign receipt", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(receipt); err != nil {
			http.Error(w, "failed to encode receipt", http.StatusInternalServerError)
			return
		}
	})
}

func readContentHash(r *http.Request) ([32]byte, error) {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBody))
	if err != nil {
		return [32]byte{}, errors.New("payment: reading request body")
	}
	raw, err := hex.DecodeString(strings.TrimSpace(string(body)))
	if err != nil {
		return [32]byte{}, errors.New("payment: content hash must be hex-encoded")
	}
	if len(raw) != 32 {
		return [32]byte{}, errors.New("payment: content hash must be 32 bytes (sha256)")
	}
	var hash [32]byte
	copy(hash[:], raw)
	return hash, nil
}
