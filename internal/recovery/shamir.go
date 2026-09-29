package recovery

import (
	"crypto/rand"
	"errors"
	"fmt"
)

// This file implements the standard GF(256) Shamir secret-sharing
// scheme (the same finite-field construction AES and BlockchainCommons'
// SSKR use — polynomial x^8+x^4+x^3+x+1, primitive element 3), written
// directly rather than pulled in as a dependency: the only readily
// available Go implementation of it (hashicorp/vault/shamir) isn't an
// independently versioned module — importing it pulls in the entire
// `hashicorp/vault` repository and forces a Go toolchain bump, wildly
// disproportionate for one small, standard algorithm with no external
// dependencies of its own. See plan/actions/A003's Log for this
// correction (A003 originally named hashicorp/vault/shamir before this
// was discovered at implementation time).

// gf256Exp and gf256Log are lookup tables for GF(256) multiplication,
// built from the primitive element 3 over the reduction polynomial
// 0x11B (x^8+x^4+x^3+x+1).
var (
	gf256Exp [255]byte
	gf256Log [256]byte
)

func init() {
	x := byte(1)
	for i := 0; i < 255; i++ {
		gf256Exp[i] = x
		gf256Log[x] = byte(i)
		x = gf256MulNoTable(x, 3)
	}
}

func gf256MulNoTable(a, b byte) byte {
	var p byte
	for i := 0; i < 8; i++ {
		if b&1 != 0 {
			p ^= a
		}
		hiBitSet := a&0x80 != 0
		a <<= 1
		if hiBitSet {
			a ^= 0x1B
		}
		b >>= 1
	}
	return p
}

func gf256Mul(a, b byte) byte {
	if a == 0 || b == 0 {
		return 0
	}
	logSum := int(gf256Log[a]) + int(gf256Log[b])
	return gf256Exp[logSum%255]
}

func gf256Div(a, b byte) byte {
	if a == 0 {
		return 0
	}
	if b == 0 {
		panic("recovery: division by zero in GF(256)")
	}
	logDiff := int(gf256Log[a]) - int(gf256Log[b])
	if logDiff < 0 {
		logDiff += 255
	}
	return gf256Exp[logDiff]
}

// Share is one Shamir share of a split secret: x is this share's
// coordinate (never 0 — 0 is reserved for the secret itself), Y holds
// one evaluated byte per secret byte, and Threshold/Total are carried
// alongside so Combine can validate it has enough shares without the
// caller needing to track that out of band.
type Share struct {
	X         byte
	Threshold int
	Total     int
	Y         []byte
}

// ErrInsufficientShares is returned by Combine when fewer than the
// recorded threshold of distinct shares is given.
var ErrInsufficientShares = errors.New("recovery: fewer than threshold distinct shares given")

// Split divides secret into total Shamir shares, any threshold of
// which reconstruct it exactly, via a degree-(threshold-1) random
// polynomial per secret byte evaluated at total distinct nonzero
// x-coordinates.
func Split(secret []byte, threshold, total int) ([]Share, error) {
	if threshold < 1 || total < threshold {
		return nil, fmt.Errorf("recovery: invalid threshold=%d/total=%d", threshold, total)
	}
	if total > 255 {
		return nil, errors.New("recovery: total shares cannot exceed 255")
	}
	if len(secret) == 0 {
		return nil, errors.New("recovery: empty secret")
	}

	shares := make([]Share, total)
	for i := 0; i < total; i++ {
		shares[i] = Share{X: byte(i + 1), Threshold: threshold, Total: total, Y: make([]byte, len(secret))}
	}

	coeffs := make([]byte, threshold)
	for byteIdx, secretByte := range secret {
		coeffs[0] = secretByte
		if _, err := rand.Read(coeffs[1:]); err != nil {
			return nil, fmt.Errorf("recovery: generating random polynomial: %w", err)
		}
		for _, share := range shares {
			shares[share.X-1].Y[byteIdx] = evalPoly(coeffs, share.X)
		}
	}
	return shares, nil
}

// evalPoly evaluates the polynomial with the given coefficients
// (coeffs[0] is the constant term) at x, in GF(256), via Horner's
// method.
func evalPoly(coeffs []byte, x byte) byte {
	result := byte(0)
	for i := len(coeffs) - 1; i >= 0; i-- {
		result = gf256Mul(result, x) ^ coeffs[i]
	}
	return result
}

// Combine reconstructs the original secret from shares via Lagrange
// interpolation at x=0, given at least the threshold recorded on the
// shares themselves. Any subset of at least threshold shares (not
// just the first ones Split produced) reconstructs correctly.
func Combine(shares []Share) ([]byte, error) {
	if len(shares) == 0 {
		return nil, errors.New("recovery: no shares given")
	}

	threshold := shares[0].Threshold
	secretLen := len(shares[0].Y)
	seen := make(map[byte]Share)
	for _, s := range shares {
		if s.Threshold != threshold {
			return nil, errors.New("recovery: shares carry mismatched thresholds")
		}
		if len(s.Y) != secretLen {
			return nil, errors.New("recovery: shares carry mismatched secret lengths")
		}
		if s.X == 0 {
			return nil, errors.New("recovery: invalid share with x=0")
		}
		seen[s.X] = s // de-duplicate by x-coordinate
	}
	if len(seen) < threshold {
		return nil, ErrInsufficientShares
	}

	distinct := make([]Share, 0, len(seen))
	for _, s := range seen {
		distinct = append(distinct, s)
		if len(distinct) == threshold {
			break // any `threshold` of them suffices
		}
	}

	secret := make([]byte, secretLen)
	for byteIdx := 0; byteIdx < secretLen; byteIdx++ {
		secret[byteIdx] = lagrangeInterpolateZero(distinct, byteIdx)
	}
	return secret, nil
}

// lagrangeInterpolateZero evaluates, at x=0, the Lagrange polynomial
// through the points (share.X, share.Y[byteIdx]) for each share.
func lagrangeInterpolateZero(shares []Share, byteIdx int) byte {
	var result byte
	for i, si := range shares {
		numerator := byte(1)
		denominator := byte(1)
		for j, sj := range shares {
			if i == j {
				continue
			}
			numerator = gf256Mul(numerator, sj.X)
			denominator = gf256Mul(denominator, sj.X^si.X)
		}
		term := gf256Mul(si.Y[byteIdx], gf256Div(numerator, denominator))
		result ^= term
	}
	return result
}
