package ccuvirtual

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/WolfHenk/ccu-modbus/internal/config"
	"github.com/WolfHenk/ccu-modbus/internal/engine"
	"github.com/WolfHenk/ccu-modbus/internal/model"
	"github.com/mdzio/go-hmccu/itf"
	"github.com/mdzio/go-hmccu/itf/vdevices"
	"github.com/mdzio/go-hmccu/itf/xmlrpc"
)

const (
	InterfaceID = "CCU-Modbus"
	RPCPath     = "/RPC3"
)

type switchKey struct {
	deviceID   string
	registerID string
}

type Integration struct {
	handler *vdevices.Handler
	devices *vdevices.Container

	mu       sync.RWMutex
	switches map[switchKey]*vdevices.DigitalChannel
	inputs   map[switchKey]*vdevices.DigitalChannel

	cancel context.CancelFunc
}

func Attach(parent context.Context, mux *http.ServeMux, cfg *config.Config, eng *engine.Engine) *Integration {
	ctx, cancel := context.WithCancel(parent)
	vd := vdevices.NewContainer()
	h := vdevices.NewHandler("127.0.0.1", false, vd, func(string) {})
	vd.Synchronizer = h

	dispatcher := itf.NewDispatcher()
	dispatcher.AddDeviceLayer(h)
	mux.Handle(RPCPath, &xmlrpc.Handler{Dispatcher: dispatcher})

	in := &Integration{
		handler:  h,
		devices:  vd,
		switches: make(map[switchKey]*vdevices.DigitalChannel),
		inputs:   make(map[switchKey]*vdevices.DigitalChannel),
		cancel:   cancel,
	}

	for _, d := range cfg.Devices {
		in.addDevice(d, eng)
	}

	go in.syncLoop(ctx, eng)
	return in
}

func (i *Integration) Close() {
	if i == nil {
		return
	}
	if i.cancel != nil {
		i.cancel()
	}
	if i.handler != nil {
		i.handler.Close()
	}
	if i.devices != nil {
		i.devices.Dispose()
	}
}

// ReplaceConfig updates the CCU device model while ReGa is still connected.
// This is called before the daemon restart after a successful configuration save.
func (i *Integration) ReplaceConfig(cfg *config.Config, eng *engine.Engine) {
	if i == nil || i.devices == nil || cfg == nil {
		return
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	for _, d := range i.devices.Devices() {
		_ = i.devices.RemoveDevice(d.Description().Address)
	}
	i.switches = make(map[switchKey]*vdevices.DigitalChannel)
	i.inputs = make(map[switchKey]*vdevices.DigitalChannel)

	for _, d := range cfg.Devices {
		i.addDevice(d, eng)
	}
}

// RemoveAll unregisters all virtual ModBus devices while ReGa is connected.
// It is intended for add-on uninstall only.
func (i *Integration) RemoveAll() {
	if i == nil || i.devices == nil {
		return
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	for _, d := range i.devices.Devices() {
		_ = i.devices.RemoveDevice(d.Description().Address)
	}
	i.switches = make(map[switchKey]*vdevices.DigitalChannel)
	i.inputs = make(map[switchKey]*vdevices.DigitalChannel)
}

func (i *Integration) addDevice(d config.Device, eng *engine.Engine) {
	addr := deviceAddress(d)
	// The CCU device class is intentionally independent of vendor/model.
	// The configurable instance name and channel layout are separate concerns.
	dev := vdevices.NewDevice(addr, "ModBus", i.handler)
	vdevices.NewMaintenanceChannel(dev)

	for _, r := range d.Registers {
		if !r.Enabled || strings.ToLower(r.DataType) != "bool" {
			continue
		}

		reg := r
		key := switchKey{deviceID: d.ID, registerID: reg.ID}

		switch r.Type {
		case "coil":
			ch := vdevices.NewSwitchChannel(dev)
			ch.OnSetState = func(value bool) bool {
				return eng.EnqueueCoil(d.ID, reg.ID, value)
			}
			i.switches[key] = ch

		case "discrete":
			// Standard CCU contact channel, but explicitly read-only.
			ch := vdevices.NewDoorSensorChannel(dev)
			if p, err := ch.ValueParamset().Parameter("STATE"); err == nil {
				p.Description().Operations = itf.ParameterOperationRead | itf.ParameterOperationEvent
			}
			i.inputs[key] = ch
		}
	}

	_ = i.devices.AddDevice(dev)
}

func (i *Integration) syncLoop(ctx context.Context, eng *engine.Engine) {
	t := time.NewTicker(500 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			i.syncStates(eng.Snapshot())
		}
	}
}

func (i *Integration) syncStates(states []model.DeviceState) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	for _, ds := range states {
		for _, rs := range ds.Registers {
			if rs.Value.Quality != model.QualityGood {
				continue
			}
			v, ok := rs.Value.Value.(bool)
			if !ok {
				continue
			}

			key := switchKey{deviceID: ds.ID, registerID: rs.ID}
			if ch := i.switches[key]; ch != nil {
				if ch.State() != v {
					ch.SetState(v)
				}
				continue
			}
			if ch := i.inputs[key]; ch != nil {
				if ch.State() != v {
					ch.SetState(v)
				}
			}
		}
	}
}

func deviceAddress(d config.Device) string {
	id := cleanToken(d.ID)
	if id == "" {
		id = "DEVICE"
	}
	return "CCUMODBUS-" + id
}

func cleanToken(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
