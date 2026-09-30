package engine

import (
	"fmt"
	"net"
	"time"

	"github.com/WolfHenk/ccu-modbus/internal/config"
	mb "github.com/WolfHenk/ccu-modbus/internal/modbus"
)

type TestResult struct {
	OK    bool   `json:"ok"`
	Raw   any    `json:"raw,omitempty"`
	Value any    `json:"value,omitempty"`
	Unit  string `json:"unit,omitempty"`
	Error string `json:"error,omitempty"`
}

func TestConnection(d config.Device) TestResult {
	if errs := config.ValidateDevice(d); len(errs) > 0 {
		return TestResult{Error: errs[0].Error()}
	}
	addr := net.JoinHostPort(d.Host, fmt.Sprint(d.Port))
	conn, err := net.DialTimeout("tcp", addr, d.Timeout())
	if err != nil {
		return TestResult{Error: humanError(err)}
	}
	_ = conn.Close()
	return TestResult{OK: true}
}

func TestRegister(d config.Device, r config.Register) TestResult {
	if errs := config.ValidateDevice(d); len(errs) > 0 {
		return TestResult{Error: errs[0].Error()}
	}
	if err := config.ValidateRegister(r); err != nil {
		return TestResult{Error: err.Error()}
	}
	fc := byte(0)
	switch r.Type {
	case "coil":
		fc = 1
	case "discrete":
		fc = 2
	case "holding":
		fc = 3
	case "input":
		fc = 4
	}
	client := mb.NewClient(net.JoinHostPort(d.Host, fmt.Sprint(d.Port)), byte(d.UnitID), d.Timeout())
	defer client.Close()

	data, err := client.Read(fc, r.Address, quantity(r))
	if err != nil {
		return TestResult{Error: humanError(err)}
	}
	raw, val, err := decode(r, data)
	if err != nil {
		return TestResult{Error: err.Error()}
	}
	return TestResult{OK: true, Raw: raw, Value: val, Unit: r.Unit}
}

func retryDelay() time.Duration { return 100 * time.Millisecond }
