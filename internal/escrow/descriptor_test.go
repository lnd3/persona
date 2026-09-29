package escrow

import "testing"

func TestDescriptorRoundTrip(t *testing.T) {
	_, pubKey := newTestKey(t)
	seq, err := SequenceForDays(14)
	if err != nil {
		t.Fatalf("SequenceForDays: %v", err)
	}

	descriptor, err := FormatPreEscalationDescriptor(pubKey, seq)
	if err != nil {
		t.Fatalf("FormatPreEscalationDescriptor: %v", err)
	}

	gotPubKey, gotSeq, err := ParsePreEscalationDescriptor(descriptor)
	if err != nil {
		t.Fatalf("ParsePreEscalationDescriptor: %v", err)
	}
	if string(gotPubKey) != string(pubKey) {
		t.Errorf("pubkey = %x, want %x", gotPubKey, pubKey)
	}
	if gotSeq != seq {
		t.Errorf("sequence = %d, want %d", gotSeq, seq)
	}
}

func TestParsePreEscalationDescriptorRejectsMalformed(t *testing.T) {
	cases := []string{
		"",
		"not a descriptor",
		"wsh(pk(deadbeef))",
		"wsh(and_v(v:pk(tooshort),older(100)))",
	}
	for _, d := range cases {
		if _, _, err := ParsePreEscalationDescriptor(d); err == nil {
			t.Errorf("ParsePreEscalationDescriptor(%q): expected error", d)
		}
	}
}

func TestFormatPreEscalationDescriptorRejectsInvalidPubKey(t *testing.T) {
	if _, err := FormatPreEscalationDescriptor([]byte{0x01}, 100); err == nil {
		t.Error("expected error for a malformed pubkey")
	}
}
