package engine

import (
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/WolfHenk/ccu-modbus/internal/config"
)

func quantity(r config.Register) uint16 {
	if r.Type == "coil" || r.Type == "discrete" {
		return 1
	}
	switch strings.ToLower(r.DataType) {
	case "uint32", "int32", "float32":
		return 2
	case "uint64", "int64":
		return 4
	default:
		return 1
	}
}

func applyByteOrder(r config.Register, buf []byte) {
	if r.ByteSwap {
		for i := 0; i+1 < len(buf); i += 2 {
			buf[i], buf[i+1] = buf[i+1], buf[i]
		}
	}

	// WordSwap exchanges the two 16-bit words inside each 32-bit block.
	// This preserves the existing 32-bit behaviour and extends naturally to
	// 64-bit values as two independently swapped 32-bit blocks.
	if r.WordSwap {
		for i := 0; i+3 < len(buf); i += 4 {
			buf[i], buf[i+2] = buf[i+2], buf[i]
			buf[i+1], buf[i+3] = buf[i+3], buf[i+1]
		}
	}

	// DWordSwap exchanges the two 32-bit halves of a 64-bit value. Together
	// with WordSwap this also represents complete 16-bit word reversal.
	if r.DWordSwap && len(buf) == 8 {
		for i := 0; i < 4; i++ {
			buf[i], buf[i+4] = buf[i+4], buf[i]
		}
	}
}

func decode(r config.Register, data []byte) (any, any, error) {
	if r.Type == "coil" || r.Type == "discrete" {
		if len(data) < 1 {
			return nil, nil, fmt.Errorf("leere bool-antwort")
		}
		v := data[0]&0x01 != 0
		return v, v, nil
	}

	need := int(quantity(r)) * 2
	if len(data) < need {
		return nil, nil, fmt.Errorf("zu kurze registerantwort: %d statt %d bytes", len(data), need)
	}

	buf := append([]byte(nil), data[:need]...)
	applyByteOrder(r, buf)

	factor := r.Factor
	if factor == 0 {
		factor = 1
	}
	apply := func(v float64) float64 { return v*factor + r.Offset }

	switch strings.ToLower(r.DataType) {
	case "uint8":
		raw := buf[1] // low byte of the 16-bit Modbus register, after optional byte swap
		return raw, apply(float64(raw)), nil
	case "int8":
		raw := int8(buf[1])
		return raw, apply(float64(raw)), nil
	case "uint16":
		raw := binary.BigEndian.Uint16(buf)
		return raw, apply(float64(raw)), nil
	case "int16":
		raw := int16(binary.BigEndian.Uint16(buf))
		return raw, apply(float64(raw)), nil
	case "uint32":
		raw := binary.BigEndian.Uint32(buf)
		return raw, apply(float64(raw)), nil
	case "int32":
		raw := int32(binary.BigEndian.Uint32(buf))
		return raw, apply(float64(raw)), nil
	case "uint64":
		raw := binary.BigEndian.Uint64(buf)
		rawText := strconv.FormatUint(raw, 10)
		if factor == 1 && r.Offset == 0 {
			return rawText, rawText, nil
		}
		return rawText, apply(float64(raw)), nil
	case "int64":
		raw := int64(binary.BigEndian.Uint64(buf))
		rawText := strconv.FormatInt(raw, 10)
		if factor == 1 && r.Offset == 0 {
			return rawText, rawText, nil
		}
		return rawText, apply(float64(raw)), nil
	case "float32":
		raw := math.Float32frombits(binary.BigEndian.Uint32(buf))
		return raw, apply(float64(raw)), nil
	default:
		return nil, nil, fmt.Errorf("datentyp %q nicht unterstuetzt", r.DataType)
	}
}
