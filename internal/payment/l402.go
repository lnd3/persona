package payment

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
)

// This file implements the client-side half of an L402 request: only
// recognizing the 402 challenge and building the retry header, never
// paying the invoice. Actually acquiring a preimage (talking to a
// Lightning wallet) is explicitly out of scope for this action — see
// A004's Context. Which Go L402 client library to build a fuller
// client on (invoice payment, wallet integration) is deliberately
// left open; this file only needs to exist independent of that
// choice, since parsing a standard HTTP 402/WWW-Authenticate exchange
// and building an Authorization header require no third-party library
// at all.

// Challenge is a parsed L402 402-response challenge.
type Challenge struct {
	Macaroon string
	Invoice  string
}

// ErrNotL402Challenge is returned when a response isn't a 402 with a
// recognizable LSAT/L402 WWW-Authenticate header.
var ErrNotL402Challenge = errors.New("payment: response is not an L402 challenge")

var challengeRE = regexp.MustCompile(`^LSAT\s+macaroon="([^"]+)"\s*,\s*invoice="([^"]+)"$`)

// ParseChallenge extracts the macaroon and invoice from a
// WWW-Authenticate header of the shape
// `LSAT macaroon="...", invoice="..."`.
func ParseChallenge(wwwAuthenticate string) (Challenge, error) {
	m := challengeRE.FindStringSubmatch(wwwAuthenticate)
	if m == nil {
		return Challenge{}, fmt.Errorf("%w: header %q", ErrNotL402Challenge, wwwAuthenticate)
	}
	return Challenge{Macaroon: m[1], Invoice: m[2]}, nil
}

// ChallengeFromResponse reads and parses the L402 challenge from an
// HTTP response, erroring if the response isn't actually a 402 with a
// well-formed WWW-Authenticate header.
func ChallengeFromResponse(resp *http.Response) (Challenge, error) {
	if resp.StatusCode != http.StatusPaymentRequired {
		return Challenge{}, fmt.Errorf("%w: status %d", ErrNotL402Challenge, resp.StatusCode)
	}
	header := resp.Header.Get("WWW-Authenticate")
	if header == "" {
		return Challenge{}, fmt.Errorf("%w: missing WWW-Authenticate header", ErrNotL402Challenge)
	}
	return ParseChallenge(header)
}

// AuthorizationHeader builds the retry request's L402 Authorization
// header value, given the challenge's macaroon and an already-obtained
// hex preimage (paying the invoice to get that preimage is the
// caller's own responsibility, via whatever wallet they use).
func AuthorizationHeader(macaroon, preimageHex string) string {
	return fmt.Sprintf("LSAT %s:%s", macaroon, preimageHex)
}
