package engine

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/WolfHenk/ccu-modbus/internal/config"
	mb "github.com/WolfHenk/ccu-modbus/internal/modbus"
	"github.com/WolfHenk/ccu-modbus/internal/model"
)

type writeRequest struct {
	registerID string
	value      bool
}

type Engine struct {
	mu      sync.RWMutex
	states  map[string]model.DeviceState
	writers map[string]chan writeRequest
	cancel  context.CancelFunc
}

func New() *Engine {
	return &Engine{
		states:  map[string]model.DeviceState{},
		writers: map[string]chan writeRequest{},
	}
}

func (e *Engine) Start(parent context.Context, cfg *config.Config) {
	ctx, cancel := context.WithCancel(parent)
	e.cancel = cancel
	for _, d := range cfg.Devices {
		d := d
		if !d.Enabled {
			continue
		}
		ch := make(chan writeRequest, 32)
		e.mu.Lock()
		e.writers[d.ID] = ch
		e.mu.Unlock()
		go e.runDevice(ctx, d, ch)
	}
}

func (e *Engine) Stop() {
	if e.cancel != nil {
		e.cancel()
	}
}

func (e *Engine) EnqueueCoil(deviceID, registerID string, value bool) bool {
	e.mu.RLock()
	ch := e.writers[deviceID]
	e.mu.RUnlock()
	if ch == nil {
		return false
	}
	select {
	case ch <- writeRequest{registerID: registerID, value: value}:
		return true
	default:
		return false
	}
}

func (e *Engine) Snapshot() []model.DeviceState {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]model.DeviceState, 0, len(e.states))
	for _, s := range e.states {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (e *Engine) setState(s model.DeviceState) {
	e.mu.Lock()
	e.states[s.ID] = s
	e.mu.Unlock()
}

func registerInterval(r config.Register) time.Duration {
	sec := r.PollSeconds
	if sec <= 0 {
		sec = 10
	}
	if sec < 1 {
		sec = 1
	}
	return time.Duration(sec) * time.Second
}

func (e *Engine) runDevice(ctx context.Context, d config.Device, writes <-chan writeRequest) {
	state := model.DeviceState{ID: d.ID, Name: d.Name, Host: d.Host, Port: d.Port, UnitID: uint8(d.UnitID), State: "STARTING"}
	nextPoll := make([]time.Time, len(d.Registers))

	now := time.Now()
	for i, r := range d.Registers {
		q := model.QualityDisabled
		if r.Enabled {
			q = model.QualityStale
			nextPoll[i] = now
		}
		if err := config.ValidateRegister(r); err != nil {
			q = model.QualityConfigError
		}
		state.Registers = append(state.Registers, model.RegisterState{ID: r.ID, Name: r.Name, Value: model.Value{Quality: q}})
	}

	if errs := config.ValidateDevice(d); len(errs) > 0 {
		state.State = "CONFIG_ERROR"
		state.LastError = errs[0].Error()
		e.setState(state)
		return
	}
	e.setState(state)

	client := mb.NewClient(net.JoinHostPort(d.Host, fmt.Sprint(d.Port)), byte(d.UnitID), d.Timeout())
	defer client.Close()

	// 250 ms is only the local scheduler resolution. It does not generate
	// Modbus traffic unless a register is due.
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	for {
		now = time.Now()
		changed := false
		for i, r := range d.Registers {
			if !r.Enabled || nextPoll[i].IsZero() || now.Before(nextPoll[i]) {
				continue
			}
			e.pollRegister(d, client, &state, i, r)
			nextPoll[i] = now.Add(registerInterval(r))
			changed = true
		}
		if changed {
			recomputeDeviceState(&state)
			e.setState(state)
		}

		select {
		case <-ctx.Done():
			state.State = "STOPPED"
			state.Connected = false
			e.setState(state)
			return
		case wr := <-writes:
			e.writeCoil(client, d, &state, wr)
			recomputeDeviceState(&state)
			e.setState(state)
		case <-ticker.C:
		}
	}
}

func (e *Engine) writeCoil(client *mb.Client, d config.Device, state *model.DeviceState, wr writeRequest) {
	idx := -1
	var reg config.Register
	for i, r := range d.Registers {
		if r.ID == wr.registerID {
			idx = i
			reg = r
			break
		}
	}
	if idx < 0 {
		state.LastError = "Unbekanntes Register fuer Schreibzugriff: " + wr.registerID
		state.ErrorCount++
		return
	}
	if !reg.Enabled || reg.Type != "coil" || reg.DataType != "bool" {
		state.LastError = "Register ist nicht als schaltbarer Coil konfiguriert: " + reg.Name
		state.ErrorCount++
		return
	}
	if err := client.WriteSingleCoil(reg.Address, wr.value); err != nil {
		state.Registers[idx].Value.Quality = model.QualityReadError
		state.Registers[idx].Value.Error = humanError(err)
		state.LastError = humanError(err)
		state.ErrorCount++
		return
	}

	now := time.Now()
	state.Registers[idx].Value = model.Value{
		Raw:       wr.value,
		Value:     wr.value,
		Timestamp: now,
		Quality:   model.QualityGood,
	}
	state.LastSeen = &now
}

func (e *Engine) pollRegister(d config.Device, client *mb.Client, state *model.DeviceState, i int, r config.Register) {
	if err := config.ValidateRegister(r); err != nil {
		state.Registers[i].Value = model.Value{Quality: model.QualityConfigError, Error: err.Error()}
		return
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

	tries := d.Retries + 1
	if tries < 1 {
		tries = 1
	}
	if tries > 3 {
		tries = 3
	}

	var data []byte
	var err error
	for n := 0; n < tries; n++ {
		data, err = client.Read(fc, r.Address, quantity(r))
		if err == nil {
			break
		}
		if n+1 < tries {
			time.Sleep(100 * time.Millisecond)
		}
	}
	if err != nil {
		state.Registers[i].Value = model.Value{Quality: model.QualityReadError, Error: humanError(err)}
		state.ErrorCount++
		state.LastError = humanError(err)
		return
	}

	raw, val, err := decode(r, data)
	if err != nil {
		state.Registers[i].Value = model.Value{Quality: model.QualityReadError, Error: err.Error()}
		state.ErrorCount++
		state.LastError = err.Error()
		return
	}

	now := time.Now()
	state.Registers[i].Value = model.Value{Raw: raw, Value: val, Timestamp: now, Quality: model.QualityGood}
	state.LastSeen = &now
}

func recomputeDeviceState(state *model.DeviceState) {
	anyGood := false
	anyError := false
	active := false

	for _, r := range state.Registers {
		switch r.Value.Quality {
		case model.QualityGood:
			anyGood = true
			active = true
		case model.QualityReadError, model.QualityConfigError:
			anyError = true
			active = true
		case model.QualityStale:
			active = true
		}
	}

	state.Connected = anyGood
	switch {
	case !active:
		state.State = "IDLE"
	case anyGood && anyError:
		state.State = "DEGRADED"
	case anyGood:
		state.State = "ONLINE"
		state.LastError = ""
	case anyError:
		state.State = "OFFLINE"
	default:
		state.State = "STARTING"
	}
}

func humanError(err error) string {
	if ne, ok := err.(net.Error); ok && ne.Timeout() {
		return "Zeitueberschreitung"
	}
	var ex *mb.ExceptionError
	if errors.As(err, &ex) {
		switch ex.Code {
		case 1:
			return "Funktion wird vom Geraet nicht unterstuetzt"
		case 2:
			return "Registeradresse wird vom Geraet nicht unterstuetzt"
		case 3:
			return "Ungueltiger Registerwert"
		case 4:
			return "Geraetefehler bei der Modbus-Anfrage"
		default:
			return fmt.Sprintf("Modbus-Ausnahme %d", ex.Code)
		}
	}
	return err.Error()
}

func LogSnapshot(states []model.DeviceState) {
	for _, s := range states {
		log.Printf("%s: status=%s connected=%v errors=%d", s.Name, s.State, s.Connected, s.ErrorCount)
	}
}
