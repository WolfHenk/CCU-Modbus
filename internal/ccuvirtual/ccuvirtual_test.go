package ccuvirtual

import (
	"testing"
	"unicode/utf8"
)

func TestLatin1DecodesOpenCCUReGaText(t *testing.T) {
	got := latin1([]byte{'T', 0xfc, 'r'})
	want := "Tür"
	if got != want {
		t.Fatalf("latin1() = %q, want %q", got, want)
	}
	if !utf8.ValidString(got) {
		t.Fatalf("latin1() returned invalid UTF-8: %q", got)
	}
}
