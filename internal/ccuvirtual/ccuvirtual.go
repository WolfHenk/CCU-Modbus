package ccuvirtual

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
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
	regaURL     = "http://127.0.0.1:8181/rega.exe"
)

type switchKey struct {
	deviceID   string
	registerID string
}

type Integration struct {
	handler *vdevices.Handler
	devices *vdevices.Container

	mu          sync.RWMutex
	switches    map[switchKey]*vdevices.DigitalChannel
	inputs      map[switchKey]*vdevices.DigitalChannel
	stateSynced map[switchKey]bool
	regaReady   bool
	current     *config.Config

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
		handler:     h,
		devices:     vd,
		switches:    make(map[switchKey]*vdevices.DigitalChannel),
		inputs:      make(map[switchKey]*vdevices.DigitalChannel),
		stateSynced: make(map[switchKey]bool),
		regaReady:   false,
		current:     cfg,
		cancel:      cancel,
	}

	for _, d := range cfg.Devices {
		in.addDevice(d, eng)
	}

	go in.connectReGa(ctx, cfg)
	go in.syncLoop(ctx, eng)
	return in
}

func (i *Integration) connectReGa(ctx context.Context, cfg *config.Config) {
	t := time.NewTicker(2 * time.Second)
	defer t.Stop()

	for {
		if id, err := resolveInterfaceID(); err == nil && id != "" {
			// ReGa identifies virtual channels by their stable address. Calling
			// newDevices again for an existing address updates its channel type
			// in place and therefore preserves ISE ID, programs, rooms and trades.
			// This migrates older SHUTTER_CONTACT inputs to DIGITAL_INPUT without
			// deleting/recreating the channel. Failure here is non-fatal: Modbus
			// operation must not depend on a presentation migration.
			_ = i.migrateDigitalInputTypes(id)
			_ = i.migrateDigitalInputControls(cfg)

			if err := i.handler.Init("xmlrpc_bin://127.0.0.1:31999", id); err == nil {
				i.mu.Lock()
				i.regaReady = true
				i.mu.Unlock()
				time.Sleep(800 * time.Millisecond)
				_ = ApplyMetadata(cfg)
				return
			}
		}

		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

func (i *Integration) migrateDigitalInputTypes(interfaceID string) error {
	cln := &itf.LogicLayerClient{
		Name:   "CCU-Modbus input migration",
		Caller: &xmlrpc.Client{Addr: "127.0.0.1:1999"},
	}

	current, err := cln.ListDevices(interfaceID)
	if err != nil {
		return err
	}
	currentByAddress := make(map[string]*itf.DeviceDescription, len(current))
	for _, d := range current {
		currentByAddress[d.Address] = d
	}

	var updates []*itf.DeviceDescription
	for _, dev := range i.devices.Devices() {
		for _, ch := range dev.Channels() {
			desired := ch.Description()
			if desired.Type != "DIGITAL_INPUT" {
				continue
			}
			if existing := currentByAddress[desired.Address]; existing != nil && existing.Type != desired.Type {
				updates = append(updates, desired)
			}
		}
	}

	if len(updates) == 0 {
		return nil
	}
	return cln.NewDevices(interfaceID, updates)
}

func (i *Integration) migrateDigitalInputControls(cfg *config.Config) error {
	if cfg == nil {
		return nil
	}

	const script = `string cid; foreach(cid,dom.GetObject(ID_CHANNELS).EnumIDs()){object ch=dom.GetObject(cid); if(ch && ch.HssType()=="DIGITAL_INPUT"){object dev=dom.GetObject(ch.Device()); if(dev && dev.HssType()=="ModBus"){string did; foreach(did,ch.DPs().EnumEnabledVisibleIDs()){object dp=dom.GetObject(did); if(dp && dp.HssType()=="STATE"){dp.MetaData("CONTROL","SWITCH_TRANSMITTER.STATE");}}}}}`
	_, err := runReGa(script)
	return err
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

// ReplaceConfig keeps already commissioned devices in ReGa whenever their
// CCU-visible topology did not change. Ordinary Modbus edits (name, address,
// scaling, polling, room/function, etc.) must therefore not send
// an already accepted device back to the inbox.
func (i *Integration) ReplaceConfig(cfg *config.Config, eng *engine.Engine) {
	if i == nil || i.devices == nil || cfg == nil {
		return
	}

	i.mu.Lock()

	if i.canReuseModel(cfg) {
		i.current = cfg
		i.mu.Unlock()
		_ = ApplyMetadata(cfg)
		return
	}

	// Never delete/recreate the complete virtual device while ReGa is live.
	// ReGa drops room/function memberships when channels are removed this way.
	// The CGI restarts the daemon after every successful save. On the fresh
	// daemon start go-hmccu compares the existing ReGa model with the new
	// configuration and adds/removes only channels whose topology really
	// changed. Existing channels keep their ReGa objects and assignments.
	i.current = cfg
	i.mu.Unlock()
}

func (i *Integration) canReuseModel(next *config.Config) bool {
	if i.current == nil || next == nil || len(i.current.Devices) != len(next.Devices) {
		return false
	}

	oldByAddress := make(map[string]config.Device, len(i.current.Devices))
	for _, d := range i.current.Devices {
		oldByAddress[deviceAddress(d)] = d
	}

	for _, nd := range next.Devices {
		od, ok := oldByAddress[deviceAddress(nd)]
		if !ok || od.ID != nd.ID {
			return false
		}
		if !sameVirtualChannels(od, nd) {
			return false
		}
	}
	return true
}

func sameVirtualChannels(a, b config.Device) bool {
	as := virtualChannelSignature(a)
	bs := virtualChannelSignature(b)
	if len(as) != len(bs) {
		return false
	}
	for n := range as {
		if as[n] != bs[n] {
			return false
		}
	}
	return true
}

func virtualChannelSignature(d config.Device) []string {
	out := make([]string, 0, len(d.Registers))
	for _, r := range d.Registers {
		if !r.Enabled || strings.ToLower(r.DataType) != "bool" || (r.Type != "coil" && r.Type != "discrete") {
			continue
		}
		out = append(out, r.Type+"|"+r.ID)
	}
	return out
}

// RemoveAll unregisters all virtual ModBus devices while ReGa is connected.
func (i *Integration) RemoveAll() {
	if i == nil || i.devices == nil {
		return
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	for _, d := range i.devices.Devices() {
		_ = i.devices.RemoveDevice(d.Description().Address)
	}
	i.handler.Synchronize()
	i.switches = make(map[switchKey]*vdevices.DigitalChannel)
	i.inputs = make(map[switchKey]*vdevices.DigitalChannel)
	i.stateSynced = make(map[switchKey]bool)
	time.Sleep(800 * time.Millisecond)
}

func (i *Integration) addDevice(d config.Device, eng *engine.Engine) {
	addr := deviceAddress(d)
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
			// Use Homematic's neutral DIGITAL_INPUT channel instead of a
			// SHUTTER_CONTACT. Modbus discrete inputs are generic binary
			// inputs, not necessarily door/window contacts.
			ch := vdevices.NewDigitalChannel(dev, "DIGITAL_INPUT", "SWITCH_TRANSMITTER.STATE")
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
	i.mu.Lock()
	defer i.mu.Unlock()

	if !i.regaReady {
		return
	}

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
				if !i.stateSynced[key] || ch.State() != v {
					ch.SetState(v)
					i.stateSynced[key] = true
				}
				continue
			}
			if ch := i.inputs[key]; ch != nil {
				if !i.stateSynced[key] || ch.State() != v {
					ch.SetState(v)
					i.stateSynced[key] = true
				}
			}
		}
	}
}

func resolveInterfaceID() (string, error) {
	out, err := runReGa(`object o=dom.GetObject("CCU-Modbus"); if(o){Write(o.ID());}`)
	if err != nil {
		return "", err
	}
	id := strings.TrimSpace(out)
	n, err := strconv.Atoi(id)
	if err != nil || n <= 0 {
		return "", fmt.Errorf("CCU-Modbus interface ID nicht gefunden: %q", id)
	}
	return id, nil
}

// SyncNamesFromReGa imports user-visible device and channel names from ReGa
// into the persistent Modbus configuration. ReGa is authoritative for names
// changed in the normal CCU device/channel UI.
func SyncNamesFromReGa(cfg *config.Config) (bool, error) {
	if cfg == nil {
		return false, nil
	}

	var b strings.Builder
	for _, d := range cfg.Devices {
		addr := deviceAddress(d)
		fmt.Fprintf(&b, `string did; foreach(did,dom.GetObject(ID_DEVICES).EnumIDs()){object dev=dom.GetObject(did); if(dev && dev.Address()==%s){WriteLine("D|"+dev.Address()+"|"+dev.Name());}}`, regaQuote(addr))

		channel := 1
		for _, r := range d.Registers {
			if !r.Enabled || strings.ToLower(r.DataType) != "bool" || (r.Type != "coil" && r.Type != "discrete") {
				continue
			}
			chAddr := fmt.Sprintf("%s:%d", addr, channel)
			fmt.Fprintf(&b, `string cid; foreach(cid,dom.GetObject(ID_CHANNELS).EnumIDs()){object ch=dom.GetObject(cid); if(ch && ch.Address()==%s){WriteLine("C|"+ch.Address()+"|"+ch.Name());}}`, regaQuote(chAddr))
			channel++
		}
	}

	out, err := runReGa(b.String())
	if err != nil {
		return false, err
	}

	names := make(map[string]string)
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSuffix(line, "\r")
		parts := strings.SplitN(line, "|", 3)
		if len(parts) != 3 || (parts[0] != "D" && parts[0] != "C") {
			continue
		}
		if strings.TrimSpace(parts[2]) == "" {
			continue
		}
		names[parts[0]+"|"+parts[1]] = parts[2]
	}

	changed := false
	for di := range cfg.Devices {
		d := &cfg.Devices[di]
		addr := deviceAddress(*d)
		if name, ok := names["D|"+addr]; ok && d.Name != name {
			d.Name = name
			changed = true
		}

		channel := 1
		for ri := range d.Registers {
			r := &d.Registers[ri]
			if !r.Enabled || strings.ToLower(r.DataType) != "bool" || (r.Type != "coil" && r.Type != "discrete") {
				continue
			}
			chAddr := fmt.Sprintf("%s:%d", addr, channel)
			if name, ok := names["C|"+chAddr]; ok && r.Name != name {
				r.Name = name
				changed = true
			}
			channel++
		}
	}
	return changed, nil
}

func ApplyMetadata(cfg *config.Config) error {
	if cfg == nil {
		return nil
	}

	var b strings.Builder
	for _, d := range cfg.Devices {
		addr := deviceAddress(d)

		// ReGa object names are user-visible and may differ from the interface
		// address. Resolve virtual devices/channels by their stable HSS address.
		fmt.Fprintf(&b, `string did; foreach(did,dom.GetObject(ID_DEVICES).EnumIDs()){object dev=dom.GetObject(did); if(dev && dev.Address()==%s){dev.Name(%s);}}`, regaQuote(addr), regaQuote(d.Name))

		channel := 1
		for _, r := range d.Registers {
			if !r.Enabled || strings.ToLower(r.DataType) != "bool" || (r.Type != "coil" && r.Type != "discrete") {
				continue
			}
			chAddr := fmt.Sprintf("%s:%d", addr, channel)
			fmt.Fprintf(&b, `string cid; foreach(cid,dom.GetObject(ID_CHANNELS).EnumIDs()){object ch=dom.GetObject(cid); if(ch && ch.Address()==%s){ch.Name(%s);}}`, regaQuote(chAddr), regaQuote(r.Name))
			channel++
		}
	}
	_, err := runReGa(b.String())
	return err
}

func runReGa(script string) (string, error) {
	req, err := http.NewRequest(http.MethodPost, regaURL, bytes.NewBufferString(script))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "text/plain")
	cln := &http.Client{Timeout: 4 * time.Second}
	resp, err := cln.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ReGa HTTP %d", resp.StatusCode)
	}
	if p := bytes.Index(body, []byte("<xml>")); p >= 0 {
		body = body[:p]
	}
	return latin1(body), nil
}

func latin1(b []byte) string {
	r := make([]rune, len(b))
	for n, c := range b {
		r[n] = rune(c)
	}
	return string(r)
}

func regaQuote(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return "\"" + s + "\""
}

func deviceAddress(d config.Device) string {
	// User-visible HSS serial: "Mod" plus the last eight digits of the
	// zero-padded IPv4 address. Example: 192.168.3.207 ->
	// 192168003207 -> Mod68003207.
	if ip := net.ParseIP(strings.TrimSpace(d.Host)); ip != nil {
		if v4 := ip.To4(); v4 != nil {
			digits := fmt.Sprintf("%03d%03d%03d%03d", v4[0], v4[1], v4[2], v4[3])
			return "Mod" + digits[len(digits)-8:]
		}
	}

	// Hostnames are still accepted by the Modbus engine. For the virtual CCU
	// identity keep a deterministic fallback rather than changing the serial on
	// every restart.
	id := cleanToken(d.ID)
	if id == "" {
		id = "DEVICE"
	}
	if len(id) > 8 {
		id = id[len(id)-8:]
	}
	return "Mod" + id
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
