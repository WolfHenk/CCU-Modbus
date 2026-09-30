package engine

import (
	"math"
	"testing"

	"github.com/WolfHenk/ccu-modbus/internal/config"
)

func TestDecodeINT16Factor(t *testing.T) {
	r := config.Register{Type: "holding", DataType: "int16", Factor: 0.1}
	raw, value, err := decode(r, []byte{0x01, 0xB5}) // 437
	if err != nil {
		t.Fatal(err)
	}
	if raw.(int16) != 437 {
		t.Fatalf("raw=%v", raw)
	}
	if math.Abs(value.(float64)-43.7) > 0.0001 {
		t.Fatalf("value=%v", value)
	}
}

func TestDecodeWordSwapFloat32(t *testing.T) {
	r := config.Register{Type: "holding", DataType: "float32", Factor: 1, WordSwap: true}
	// 12.5 = 0x41480000; words deliberately swapped on the wire.
	_, value, err := decode(r, []byte{0x00, 0x00, 0x41, 0x48})
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(value.(float64)-12.5) > 0.0001 {
		t.Fatalf("value=%v", value)
	}
}
