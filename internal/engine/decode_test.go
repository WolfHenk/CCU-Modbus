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

func TestQuantity64Bit(t *testing.T) {
	for _, dt := range []string{"uint64", "int64"} {
		r := config.Register{Type: "holding", DataType: dt}
		if got := quantity(r); got != 4 {
			t.Fatalf("quantity(%s)=%d, want 4", dt, got)
		}
	}
}

func TestDecodeUINT64Exact(t *testing.T) {
	r := config.Register{Type: "holding", DataType: "uint64", Factor: 1}
	raw, value, err := decode(r, []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff})
	if err != nil {
		t.Fatal(err)
	}
	const want = "18446744073709551615"
	if raw != want {
		t.Fatalf("raw=%v, want %s", raw, want)
	}
	if value != want {
		t.Fatalf("value=%v, want exact string %s", value, want)
	}
}

func TestDecodeINT64Exact(t *testing.T) {
	r := config.Register{Type: "holding", DataType: "int64", Factor: 1}
	raw, value, err := decode(r, []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xfe})
	if err != nil {
		t.Fatal(err)
	}
	if raw != "-2" || value != "-2" {
		t.Fatalf("raw=%v value=%v, want -2", raw, value)
	}
}

func TestDecodeUINT64Scaled(t *testing.T) {
	r := config.Register{Type: "holding", DataType: "uint64", Factor: 0.01}
	raw, value, err := decode(r, []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0xe2, 0x40}) // 123456
	if err != nil {
		t.Fatal(err)
	}
	if raw != "123456" {
		t.Fatalf("raw=%v", raw)
	}
	if math.Abs(value.(float64)-1234.56) > 0.0001 {
		t.Fatalf("value=%v", value)
	}
}

func TestDecodeUINT64SwapCombinations(t *testing.T) {
	const want = "81985529216486895" // 0x0123456789ABCDEF
	tests := []struct {
		name string
		reg  config.Register
		wire []byte
	}{
		{
			name: "normal",
			reg:  config.Register{Type: "holding", DataType: "uint64", Factor: 1},
			wire: []byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef},
		},
		{
			name: "byte",
			reg:  config.Register{Type: "holding", DataType: "uint64", Factor: 1, ByteSwap: true},
			wire: []byte{0x23, 0x01, 0x67, 0x45, 0xab, 0x89, 0xef, 0xcd},
		},
		{
			name: "word",
			reg:  config.Register{Type: "holding", DataType: "uint64", Factor: 1, WordSwap: true},
			wire: []byte{0x45, 0x67, 0x01, 0x23, 0xcd, 0xef, 0x89, 0xab},
		},
		{
			name: "dword",
			reg:  config.Register{Type: "holding", DataType: "uint64", Factor: 1, DWordSwap: true},
			wire: []byte{0x89, 0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67},
		},
		{
			name: "all",
			reg:  config.Register{Type: "holding", DataType: "uint64", Factor: 1, ByteSwap: true, WordSwap: true, DWordSwap: true},
			wire: []byte{0xef, 0xcd, 0xab, 0x89, 0x67, 0x45, 0x23, 0x01},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw, value, err := decode(tt.reg, tt.wire)
			if err != nil {
				t.Fatal(err)
			}
			if raw != want || value != want {
				t.Fatalf("raw=%v value=%v, want %s", raw, value, want)
			}
		})
	}
}

func TestDecodeBytesHighLow(t *testing.T) {
	cases := []struct {
		dt   string
		swap bool
		want float64
	}{{"uint8", false, 254}, {"uint8", true, 127}, {"int8", false, -2}, {"int8", true, 127}}
	for _, tc := range cases {
		r := config.Register{Type: "holding", DataType: tc.dt, ByteSwap: tc.swap, Factor: 1}
		_, v, e := decode(r, []byte{0x7f, 0xfe})
		if e != nil {
			t.Fatal(e)
		}
		if v != tc.want {
			t.Errorf("%s swap=%v: got %v, want %v", tc.dt, tc.swap, v, tc.want)
		}
	}
}
