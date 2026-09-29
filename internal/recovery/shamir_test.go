package recovery

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func randomSecret(t *testing.T, n int) []byte {
	t.Helper()
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}
	return b
}

func TestSplitCombineExactThreshold(t *testing.T) {
	secret := randomSecret(t, 32)
	shares, err := Split(secret, 3, 5)
	if err != nil {
		t.Fatalf("Split: %v", err)
	}
	got, err := Combine(shares[:3])
	if err != nil {
		t.Fatalf("Combine: %v", err)
	}
	if !bytes.Equal(got, secret) {
		t.Errorf("Combine with exactly threshold shares = %x, want %x", got, secret)
	}
}

func TestCombineFailsWithFewerThanThreshold(t *testing.T) {
	secret := randomSecret(t, 32)
	shares, err := Split(secret, 3, 5)
	if err != nil {
		t.Fatalf("Split: %v", err)
	}
	if _, err := Combine(shares[:2]); err == nil {
		t.Fatal("expected error combining fewer than threshold shares")
	}
}

func TestCombineArbitrarySubset(t *testing.T) {
	secret := randomSecret(t, 32)
	shares, err := Split(secret, 3, 6)
	if err != nil {
		t.Fatalf("Split: %v", err)
	}

	subsets := [][]int{
		{0, 1, 2},
		{3, 4, 5},
		{0, 2, 4},
		{1, 3, 5},
	}
	for _, idxs := range subsets {
		var subset []Share
		for _, i := range idxs {
			subset = append(subset, shares[i])
		}
		got, err := Combine(subset)
		if err != nil {
			t.Fatalf("Combine(%v): %v", idxs, err)
		}
		if !bytes.Equal(got, secret) {
			t.Errorf("Combine(%v) = %x, want %x", idxs, got, secret)
		}
	}
}

func TestCombineWithMoreThanThreshold(t *testing.T) {
	secret := randomSecret(t, 32)
	shares, err := Split(secret, 3, 5)
	if err != nil {
		t.Fatalf("Split: %v", err)
	}
	got, err := Combine(shares) // all 5
	if err != nil {
		t.Fatalf("Combine: %v", err)
	}
	if !bytes.Equal(got, secret) {
		t.Errorf("Combine(all) = %x, want %x", got, secret)
	}
}

func TestSplitRejectsInvalidThreshold(t *testing.T) {
	if _, err := Split([]byte("secret"), 0, 5); err == nil {
		t.Error("expected error for threshold=0")
	}
	if _, err := Split([]byte("secret"), 6, 5); err == nil {
		t.Error("expected error for threshold > total")
	}
}

func TestDuplicateSharesDontCountTwice(t *testing.T) {
	secret := randomSecret(t, 16)
	shares, err := Split(secret, 3, 5)
	if err != nil {
		t.Fatalf("Split: %v", err)
	}
	dup := []Share{shares[0], shares[0], shares[1]} // only 2 distinct x-coordinates
	if _, err := Combine(dup); err == nil {
		t.Fatal("expected error: duplicate shares should not satisfy the threshold")
	}
}
