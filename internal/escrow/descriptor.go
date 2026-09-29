package escrow

import (
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
)

// This file implements A005's `EscrowRef` as a BIP380-shaped output
// descriptor, per A006's Decisions — but only a formatter/parser for
// this package's own fixed script template, not a general-purpose
// descriptor engine. `wsh(and_v(v:pk(<K>),older(<n>)))` is valid,
// standard miniscript for exactly PreEscalationScript's shape
// ("signed by K AND at least n old"); recognizing only this one
// template (rather than implementing miniscript compilation
// generally) is a deliberately narrow, honestly-scoped slice — a
// generic descriptor/miniscript compiler is real, separate work this
// action doesn't take on.

var preEscalationDescriptorRE = regexp.MustCompile(`^wsh\(and_v\(v:pk\(([0-9a-f]{66})\),older\((\d+)\)\)\)$`)

// ErrMalformedDescriptor is returned when a string isn't a
// recognized pre-escalation output descriptor.
var ErrMalformedDescriptor = fmt.Errorf("escrow: not a recognized pre-escalation descriptor")

// FormatPreEscalationDescriptor renders the standard descriptor
// string for a PreEscalationScript(ownerPubKey, sequence) output,
// suitable for A005's `EscrowRef`.
func FormatPreEscalationDescriptor(ownerPubKey []byte, sequence uint32) (string, error) {
	if err := validatePubKey(ownerPubKey); err != nil {
		return "", err
	}
	return fmt.Sprintf("wsh(and_v(v:pk(%s),older(%d)))", hex.EncodeToString(ownerPubKey), sequence), nil
}

// ParsePreEscalationDescriptor extracts the owner pubkey and sequence
// from a descriptor produced by FormatPreEscalationDescriptor.
func ParsePreEscalationDescriptor(descriptor string) (ownerPubKey []byte, sequence uint32, err error) {
	m := preEscalationDescriptorRE.FindStringSubmatch(descriptor)
	if m == nil {
		return nil, 0, fmt.Errorf("%w: got %q", ErrMalformedDescriptor, descriptor)
	}
	pubKey, err := hex.DecodeString(m[1])
	if err != nil {
		return nil, 0, fmt.Errorf("%w: invalid pubkey hex: %v", ErrMalformedDescriptor, err)
	}
	seq, err := strconv.ParseUint(m[2], 10, 32)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: invalid sequence: %v", ErrMalformedDescriptor, err)
	}
	return pubKey, uint32(seq), nil
}
