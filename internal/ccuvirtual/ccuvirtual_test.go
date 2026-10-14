package ccuvirtual

import (
	"testing"
	"unicode/utf8"

	"github.com/WolfHenk/ccu-modbus/internal/config"
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

func TestDeviceAddressKeepsLegacyAddressWhenUnique(t *testing.T) {
	d := config.Device{ID: "plant-247", Host: "192.168.3.207", Port: 502, UnitID: 247}
	cfg := &config.Config{Devices: []config.Device{d}}
	if got, want := deviceAddress(cfg, d), "Mod68003207"; got != want {
		t.Fatalf("deviceAddress() = %q, want %q", got, want)
	}
}

func TestDeviceAddressSeparatesUnitsOnSameEndpoint(t *testing.T) {
	d1 := config.Device{ID: "plant-1", Host: "192.168.3.207", Port: 502, UnitID: 1}
	d247 := config.Device{ID: "plant-247", Host: "192.168.3.207", Port: 502, UnitID: 247}
	cfg := &config.Config{Devices: []config.Device{d1, d247}}

	if got, want := deviceAddress(cfg, d1), "MB003207502001"; got != want {
		t.Fatalf("Unit 1 address = %q, want %q", got, want)
	}
	if got, want := deviceAddress(cfg, d247), "MB003207502247"; got != want {
		t.Fatalf("Unit 247 address = %q, want %q", got, want)
	}
	if deviceAddress(cfg, d1) == deviceAddress(cfg, d247) {
		t.Fatal("same IP/port with different Unit-ID must create distinct CCU devices")
	}
}

func TestDeviceAddressSeparatesPortsOnSameIP(t *testing.T) {
	d502 := config.Device{ID: "port-502", Host: "192.168.3.207", Port: 502, UnitID: 1}
	d1502 := config.Device{ID: "port-1502", Host: "192.168.3.207", Port: 1502, UnitID: 1}
	cfg := &config.Config{Devices: []config.Device{d502, d1502}}

	if got, want := deviceAddress(cfg, d502), "MB003207502001"; got != want {
		t.Fatalf("port 502 address = %q, want %q", got, want)
	}
	got := deviceAddress(cfg, d1502)
	if len(got) != 14 || got[:2] != "MB" {
		t.Fatalf("port 1502 fallback address = %q, want 14-character MB serial", got)
	}
	if got == deviceAddress(cfg, d502) {
		t.Fatalf("different ports collide: %q", got)
	}
}

func TestDeviceAddressSeparatesDuplicateLogicalDevices(t *testing.T) {
	dA := config.Device{ID: "dev-a", Host: "192.168.3.207", Port: 502, UnitID: 1}
	dB := config.Device{ID: "dev-b", Host: "192.168.3.207", Port: 502, UnitID: 1}
	cfg := &config.Config{Devices: []config.Device{dA, dB}}

	a := deviceAddress(cfg, dA)
	b := deviceAddress(cfg, dB)
	if a == b {
		t.Fatalf("duplicate logical devices collide: %q", a)
	}
	if len(a) != 14 || len(b) != 14 || a[:2] != "MB" || b[:2] != "MB" {
		t.Fatalf("duplicate fallback addresses must be 14-character MB serials: %q / %q", a, b)
	}
}

func TestDeviceAddressCompactFormatExample(t *testing.T) {
	d1 := config.Device{ID: "unit-1", Host: "192.168.3.203", Port: 504, UnitID: 1}
	d2 := config.Device{ID: "unit-247", Host: "192.168.3.203", Port: 504, UnitID: 247}
	cfg := &config.Config{Devices: []config.Device{d1, d2}}
	if got, want := deviceAddress(cfg, d1), "MB003203504001"; got != want {
		t.Fatalf("compact address = %q, want %q", got, want)
	}
}

func TestDeviceAddressNeverExceeds14Characters(t *testing.T) {
	devices := []config.Device{
		{ID: "a", Host: "192.168.3.203", Port: 502, UnitID: 1},
		{ID: "b", Host: "192.168.3.203", Port: 502, UnitID: 247},
		{ID: "c", Host: "192.168.3.203", Port: 1502, UnitID: 1},
	}
	cfg := &config.Config{Devices: devices}
	for _, d := range devices {
		if got := deviceAddress(cfg, d); len(got) > 14 {
			t.Fatalf("address too long: %q (%d)", got, len(got))
		}
	}
}