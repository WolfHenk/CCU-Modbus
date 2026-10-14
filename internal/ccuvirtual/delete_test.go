package ccuvirtual

import (
	"errors"
	"github.com/mdzio/go-hmccu/itf/vdevices"
	"testing"
)

func TestPersistentDeviceLayerDelete(t *testing.T) {
	container := vdevices.NewContainer()
	handler := vdevices.NewHandler("127.0.0.1", false, container, func(string) {})
	container.Synchronizer = handler
	device := vdevices.NewDevice("Mod68003207", "ModBus", handler)
	if err := container.AddDevice(device); err != nil {
		t.Fatal(err)
	}
	calls := 0
	layer := &persistentDeviceLayer{Handler: handler, onDelete: func(address string) error {
		if address != "Mod68003207" {
			t.Fatalf("unexpected address %q", address)
		}
		calls++
		return nil
	}}
	if err := layer.DeleteDevice("Mod68003207:0", 0); err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatal("channel deletion must not delete configuration")
	}
	if err := layer.DeleteDevice("Mod68003207", 0); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("persistence callback calls=%d", calls)
	}
	if len(container.Devices()) != 0 {
		t.Fatal("device still registered")
	}
}

func TestPersistentDeviceLayerRejectsFailedPersistence(t *testing.T) {
	container := vdevices.NewContainer()
	handler := vdevices.NewHandler("127.0.0.1", false, container, func(string) {})
	container.Synchronizer = handler
	if err := container.AddDevice(vdevices.NewDevice("Mod68003207", "ModBus", handler)); err != nil {
		t.Fatal(err)
	}
	layer := &persistentDeviceLayer{Handler: handler, onDelete: func(string) error { return errors.New("disk unavailable") }}
	if err := layer.DeleteDevice("Mod68003207", 0); err == nil {
		t.Fatal("expected persistence failure")
	}
	if len(container.Devices()) != 1 {
		t.Fatal("device unregistered despite persistence error")
	}
}
