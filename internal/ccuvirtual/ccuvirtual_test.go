package ccuvirtual

import (
	"testing"
	"unicode/utf8"

	"github.com/mdzio/go-hmccu/itf"
	"github.com/mdzio/go-hmccu/itf/xmlrpc"
)

func TestDecodeReGaLegacyLatin1(t *testing.T) {
	got := decodeReGaText([]byte{'T', 0xfc, 'r'})
	if got != "Tür" {
		t.Fatalf("decodeReGaText Latin-1 = %q", got)
	}
	if !utf8.ValidString(got) {
		t.Fatalf("invalid UTF-8: %q", got)
	}
}

func TestDecodeReGaKeepsUTF8(t *testing.T) {
	got := decodeReGaText([]byte("TürMeldungen"))
	if got != "TürMeldungen" {
		t.Fatalf("decodeReGaText UTF-8 = %q", got)
	}
}

func TestDecodeReGaMixedUTF8AndLatin1(t *testing.T) {
	raw := append([]byte("Tür|"), []byte{'T', 0xfc, 'r'}...)
	got := decodeReGaText(raw)
	if got != "Tür|Tür" {
		t.Fatalf("decodeReGaText mixed = %q", got)
	}
}

func TestSetReadyConfigCompatibilityRPC(t *testing.T) {
	dispatcher := itf.NewDispatcher()
	registerCCUCompatibilityMethods(dispatcher)

	args := &xmlrpc.Value{Array: &xmlrpc.Array{Data: []*xmlrpc.Value{}}}
	if _, err := dispatcher.Dispatch("setReadyConfig", args); err != nil {
		t.Fatalf("setReadyConfig dispatch failed: %v", err)
	}
}
