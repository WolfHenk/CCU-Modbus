package engine

import (
	"encoding/binary"
	"fmt"
	"math"
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
	default:
		return 1
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
	if r.WordSwap && len(buf) == 4 {
		buf[0], buf[2] = buf[2], buf[0]
		buf[1], buf[3] = buf[3], buf[1]
	}
	if r.ByteSwap {
		for i := 0; i < len(buf); i += 2 {
			buf[i], buf[i+1] = buf[i+1], buf[i]
		}
	}

	factor := r.Factor
	if factor == 0 {
		factor = 1
	}
	apply := func(v float64) float64 { return v*factor + r.Offset }

	switch strings.ToLower(r.DataType) {
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
	case "float32":
		raw := math.Float32frombits(binary.BigEndian.Uint32(buf))
		return raw, apply(float64(raw)), nil
	default:
		return nil, nil, fmt.Errorf("datentyp %q nicht unterstuetzt", r.DataType)
	}
}
