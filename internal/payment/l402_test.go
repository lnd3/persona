package payment

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseChallenge(t *testing.T) {
	header := `LSAT macaroon="AGIAJEem...", invoice="lnbc1..."`
	c, err := ParseChallenge(header)
	if err != nil {
		t.Fatalf("ParseChallenge: %v", err)
	}
	if c.Macaroon != "AGIAJEem..." || c.Invoice != "lnbc1..." {
		t.Errorf("parsed = %+v, want macaroon/invoice extracted", c)
	}
}

func TestParseChallengeRejectsMalformed(t *testing.T) {
	cases := []string{
		"",
		"Bearer sometoken",
		`LSAT macaroon="onlyone"`,
		`LSAT invoice="onlyone"`,
	}
	for _, header := range cases {
		if _, err := ParseChallenge(header); err == nil {
			t.Errorf("ParseChallenge(%q): expected error", header)
		}
	}
}

func TestChallengeFromResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("WWW-Authenticate", `LSAT macaroon="macbytes", invoice="lnbc1invoice"`)
		w.WriteHeader(http.StatusPaymentRequired)
	}))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("http.Get: %v", err)
	}
	defer resp.Body.Close()

	c, err := ChallengeFromResponse(resp)
	if err != nil {
		t.Fatalf("ChallengeFromResponse: %v", err)
	}
	if c.Macaroon != "macbytes" || c.Invoice != "lnbc1invoice" {
		t.Errorf("parsed = %+v", c)
	}
}

func TestChallengeFromResponseRejectsNon402(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("http.Get: %v", err)
	}
	defer resp.Body.Close()

	if _, err := ChallengeFromResponse(resp); err == nil {
		t.Fatal("expected error for non-402 response")
	}
}

func TestAuthorizationHeader(t *testing.T) {
	got := AuthorizationHeader("macbytes", "deadbeef")
	want := "LSAT macbytes:deadbeef"
	if got != want {
		t.Errorf("AuthorizationHeader = %q, want %q", got, want)
	}
}
