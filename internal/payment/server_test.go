package payment

import (
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lnd3/persona/internal/identity"
)

func TestGatewayHandlerSignsValidHash(t *testing.T) {
	gateway, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	srv := httptest.NewServer(NewGatewayHandler(gateway))
	defer srv.Close()

	hash := ContentHash("attester", "subject", "net.persona.core.skill", "go", 1000)
	resp, err := http.Post(srv.URL, "text/plain", strings.NewReader(hex.EncodeToString(hash[:])))
	if err != nil {
		t.Fatalf("http.Post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var receipt Receipt
	if err := json.NewDecoder(resp.Body).Decode(&receipt); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if err := VerifyReceipt(receipt, hash); err != nil {
		t.Fatalf("VerifyReceipt on gateway-issued receipt: %v", err)
	}
	if receipt.GatewayPubkey != gateway.PublicKey {
		t.Errorf("receipt.GatewayPubkey = %s, want %s", receipt.GatewayPubkey, gateway.PublicKey)
	}
}

func TestGatewayHandlerRejectsMalformedBody(t *testing.T) {
	gateway, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	srv := httptest.NewServer(NewGatewayHandler(gateway))
	defer srv.Close()

	cases := []string{
		"not-hex",
		"deadbeef", // valid hex, but not 32 bytes
		"",
	}
	for _, body := range cases {
		resp, err := http.Post(srv.URL, "text/plain", strings.NewReader(body))
		if err != nil {
			t.Fatalf("http.Post(%q): %v", body, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("body %q: status = %d, want 400", body, resp.StatusCode)
		}
	}
}

func TestGatewayHandlerRejectsWrongMethod(t *testing.T) {
	gateway, err := identity.New()
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	srv := httptest.NewServer(NewGatewayHandler(gateway))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("http.Get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", resp.StatusCode)
	}
}
