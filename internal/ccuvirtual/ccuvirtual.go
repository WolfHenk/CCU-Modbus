package ccuvirtual

import (
	"bytes"
	"context"
	"fmt"
	"io"
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

type MetadataItem struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Metadata struct {
	Rooms     []MetadataItem `json:"rooms"`
	Functions []MetadataItem `json:"functions"`
}

type Integration struct {
	handler *vdevices.Handler
	devices *vdevices.Container

	mu                 sync.RWMutex
	switches           map[switchKey]*vdevices.DigitalChannel
	inputs             map[switchKey]*vdevices.DigitalChannel
	inputTrueMeansOpen map[switchKey]bool

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
		handler:            h,
		devices:            vd,
		switches:           make(map[switchKey]*vdevices.DigitalChannel),
		inputs:             make(map[switchKey]*vdevices.DigitalChannel),
		inputTrueMeansOpen: make(map[switchKey]bool),
		cancel:             cancel,
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
			if err := i.handler.Init("xmlrpc_bin://127.0.0.1:31999", id); err == nil {
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

// ReplaceConfig deliberately removes the complete virtual model first. ReGa
// does not reliably accept new child channels for an already existing virtual
// device when newDevices contains only the missing children.
func (i *Integration) ReplaceConfig(cfg *config.Config, eng *engine.Engine) {
	if i == nil || i.devices == nil || cfg == nil {
		return
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	for _, d := range i.devices.Devices() {
		_ = i.devices.RemoveDevice(d.Description().Address)
	}
	i.handler.Synchronize()
	time.Sleep(1200 * time.Millisecond)

	i.switches = make(map[switchKey]*vdevices.DigitalChannel)
	i.inputs = make(map[switchKey]*vdevices.DigitalChannel)
	i.inputTrueMeansOpen = make(map[switchKey]bool)

	for _, d := range cfg.Devices {
		i.addDevice(d, eng)
	}
	i.handler.Synchronize()
	time.Sleep(1200 * time.Millisecond)
	_ = ApplyMetadata(cfg)
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
	i.inputTrueMeansOpen = make(map[switchKey]bool)
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
			ch := vdevices.NewDoorSensorChannel(dev)
			if p, err := ch.ValueParamset().Parameter("STATE"); err == nil {
				p.Description().Operations = itf.ParameterOperationRead | itf.ParameterOperationEvent
			}
			i.inputs[key] = ch
			i.inputTrueMeansOpen[key] = reg.TrueMeansOpen
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
				ccuState := v
				if !i.inputTrueMeansOpen[key] {
					ccuState = !v
				}
				if ch.State() != ccuState {
					ch.SetState(ccuState)
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

func ReadMetadata() (Metadata, error) {
	script := `string id;
foreach(id, dom.GetObject(ID_ROOMS).EnumIDs()){object o=dom.GetObject(id); if(o){WriteLine("R\t"#o.ID()#"\t"#o.Name());}}
foreach(id, dom.GetObject(ID_FUNCTIONS).EnumIDs()){object o=dom.GetObject(id); if(o){WriteLine("F\t"#o.ID()#"\t"#o.Name());}}`
	out, err := runReGa(script)
	if err != nil {
		return Metadata{}, err
	}

	var m Metadata
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSuffix(line, "\r")
		p := strings.SplitN(line, "\t", 3)
		if len(p) != 3 {
			continue
		}
		id, err := strconv.Atoi(strings.TrimSpace(p[1]))
		if err != nil {
			continue
		}
		item := MetadataItem{ID: id, Name: p[2]}
		switch p[0] {
		case "R":
			m.Rooms = append(m.Rooms, item)
		case "F":
			m.Functions = append(m.Functions, item)
		}
	}
	return m, nil
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
			fmt.Fprintf(&b, `string cid; foreach(cid,dom.GetObject(ID_CHANNELS).EnumIDs()){object ch=dom.GetObject(cid); if(ch && ch.Address()==%s){ch.Name(%s); string x; foreach(x,dom.GetObject(ID_ROOMS).EnumIDs()){object e=dom.GetObject(x); if(e){e.Remove(ch.ID());}} foreach(x,dom.GetObject(ID_FUNCTIONS).EnumIDs()){object e=dom.GetObject(x); if(e){e.Remove(ch.ID());}}`, regaQuote(chAddr), regaQuote(r.Name))
			if r.RoomID > 0 {
				fmt.Fprintf(&b, ` object room=dom.GetObject(%d); if(room){room.Add(ch.ID());}`, r.RoomID)
			}
			if r.FunctionID > 0 {
				fmt.Fprintf(&b, ` object fn=dom.GetObject(%d); if(fn){fn.Add(ch.ID());}`, r.FunctionID)
			}
			b.WriteString("}}")
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
