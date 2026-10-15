package ccuvirtual

import (
	"github.com/WolfHenk/ccu-modbus/internal/config"
	"testing"
)

func TestCoilDirectionChannelSignature(t *testing.T) {
	base := config.Device{Registers: []config.Register{{ID: "a", Type: "coil", Enabled: true, DataType: "bool"}, {ID: "d", Type: "discrete", Enabled: true, DataType: "bool"}}}
	sig := virtualChannelSignature(base)
	if len(sig) != 2 || sig[0] != "coil|a|output" || sig[1] != "discrete|d" {
		t.Fatalf("legacy topology: %v", sig)
	}
	explicit := base
	explicit.Registers = append([]config.Register(nil), base.Registers...)
	explicit.Registers[0].CoilDirection = "output"
	if !sameVirtualChannels(base, explicit) {
		t.Fatal("legacy and output must share topology")
	}
	explicit.Registers[0].CoilDirection = "input"
	if sameVirtualChannels(base, explicit) {
		t.Fatal("input direction must change topology")
	}
}
