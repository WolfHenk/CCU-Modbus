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

func TestEnsureDeviceAddressesMigratesUniqueLegacyDevice(t *testing.T) {
	d := config.Device{ID: "legacy", Host: "192.168.3.207", Port: 502, UnitID: 1}
	cfg := &config.Config{Schema: 1, Devices: []config.Device{d}}
	if !EnsureDeviceAddresses(cfg, true) {
		t.Fatal("migration reported no change")
	}
	if got, want := cfg.Devices[0].CCUAddress, "Mod68003207"; got != want {
		t.Fatalf("legacy address = %q, want %q", got, want)
	}
	if cfg.Schema != 2 {
		t.Fatalf("schema = %d, want 2", cfg.Schema)
	}
}

func TestEnsureDeviceAddressesAssignsCompactAddressToNewDevice(t *testing.T) {
	d := config.Device{ID: "new", Host: "192.168.3.203", Port: 504, UnitID: 1}
	cfg := &config.Config{Schema: 2, Devices: []config.Device{d}}
	EnsureDeviceAddresses(cfg, false)
	if got, want := cfg.Devices[0].CCUAddress, "MB003203504001"; got != want {
		t.Fatalf("new address = %q, want %q", got, want)
	}
}

func TestPersistentAddressDoesNotChangeWhenSecondUnitAdded(t *testing.T) {
	first := config.Device{ID: "unit-1", Host: "192.168.3.203", Port: 504, UnitID: 1}
	cfg := &config.Config{Schema: 2, Devices: []config.Device{first}}
	EnsureDeviceAddresses(cfg, false)
	if got, want := cfg.Devices[0].CCUAddress, "MB003203504001"; got != want {
		t.Fatalf("first address = %q, want %q", got, want)
	}

	firstAddress := cfg.Devices[0].CCUAddress
	cfg.Devices = append(cfg.Devices, config.Device{ID: "unit-247", Host: "192.168.3.203", Port: 504, UnitID: 247})
	EnsureDeviceAddresses(cfg, false)
	if cfg.Devices[0].CCUAddress != firstAddress {
		t.Fatalf("first address changed from %q to %q", firstAddress, cfg.Devices[0].CCUAddress)
	}
	if got, want := cfg.Devices[1].CCUAddress, "MB003203504247"; got != want {
		t.Fatalf("second address = %q, want %q", got, want)
	}
}

func TestEnsureDeviceAddressesSeparatesLegacyCollisions(t *testing.T) {
	d1 := config.Device{ID: "plant-1", Host: "192.168.3.207", Port: 502, UnitID: 1}
	d247 := config.Device{ID: "plant-247", Host: "192.168.3.207", Port: 502, UnitID: 247}
	cfg := &config.Config{Schema: 1, Devices: []config.Device{d1, d247}}
	EnsureDeviceAddresses(cfg, true)
	if got, want := cfg.Devices[0].CCUAddress, "MB003207502001"; got != want {
		t.Fatalf("Unit 1 address = %q, want %q", got, want)
	}
	if got, want := cfg.Devices[1].CCUAddress, "MB003207502247"; got != want {
		t.Fatalf("Unit 247 address = %q, want %q", got, want)
	}
}

func TestEnsureDeviceAddressesUses14CharacterFallback(t *testing.T) {
	d := config.Device{ID: "port-1502", Host: "192.168.3.207", Port: 1502, UnitID: 1}
	cfg := &config.Config{Schema: 2, Devices: []config.Device{d}}
	EnsureDeviceAddresses(cfg, false)
	got := cfg.Devices[0].CCUAddress
	if len(got) != 14 || got[:2] != "MB" {
		t.Fatalf("fallback address = %q, want 14-character MB serial", got)
	}
}

func TestEnsureDeviceAddressesSeparatesDuplicateLogicalDevices(t *testing.T) {
	dA := config.Device{ID: "dev-a", Host: "192.168.3.207", Port: 502, UnitID: 1}
	dB := config.Device{ID: "dev-b", Host: "192.168.3.207", Port: 502, UnitID: 1}
	cfg := &config.Config{Schema: 2, Devices: []config.Device{dA, dB}}
	EnsureDeviceAddresses(cfg, false)
	a := cfg.Devices[0].CCUAddress
	b := cfg.Devices[1].CCUAddress
	if a == b {
		t.Fatalf("duplicate logical devices collide: %q", a)
	}
	if len(a) > 14 || len(b) > 14 {
		t.Fatalf("address too long: %q / %q", a, b)
	}
}

func TestDeviceAddressUsesPersistedAddress(t *testing.T) {
	d := config.Device{ID: "x", Host: "192.168.3.207", Port: 502, UnitID: 1, CCUAddress: "MB003207502001"}
	if got := deviceAddress(nil, d); got != d.CCUAddress {
		t.Fatalf("deviceAddress = %q, want persisted %q", got, d.CCUAddress)
	}
}

func TestDisplayedInputState(t *testing.T) {
	for _, c := range []struct{ raw, invert, want bool }{{false, false, false}, {true, false, true}, {false, true, true}, {true, true, false}} {
		if got := displayedInputState(c.raw, c.invert); got != c.want {
			t.Errorf("raw=%t invert=%t got=%t want=%t", c.raw, c.invert, got, c.want)
		}
	}
}
