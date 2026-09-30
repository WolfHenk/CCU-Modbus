package engine

import (
	"context"
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

type Engine struct {
	mu     sync.RWMutex
	states map[string]model.DeviceState
	cancel context.CancelFunc
}

func New() *Engine {
	return &Engine{states: map[string]model.DeviceState{}}
}

func (e *Engine) Start(parent context.Context, cfg *config.Config) {
	ctx, cancel := context.WithCancel(parent)
	e.cancel = cancel
	for _, d := range cfg.Devices {
		d := d
		if !d.Enabled {
			continue
		}
		go e.runDevice(ctx, d)
	}
}

func (e *Engine) Stop() {
	if e.cancel != nil {
		e.cancel()
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

func (e *Engine) runDevice(ctx context.Context, d config.Device) {
	state := model.DeviceState{
		ID: d.ID, Name: d.Name, Host: d.Host, Port: d.Port,
		UnitID: uint8(d.UnitID), State: "STARTING",
	}

	for _, r := range d.Registers {
		q := model.QualityDisabled
		if r.Enabled {
			q = model.QualityStale
		}
		if err := config.ValidateRegister(r); err != nil {
			q = model.QualityConfigError
		}
		state.Registers = append(state.Registers, model.RegisterState{
			ID: r.ID, Name: r.Name, Value: model.Value{Quality: q},
		})
	}

	if errs := config.ValidateDevice(d); len(errs) > 0 {
		state.State = "CONFIG_ERROR"
		state.LastError = errs[0].Error()
		e.setState(state)
		return
	}
	e.setState(state)

	interval := 10 * time.Second
	for _, r := range d.Registers {
		if r.Enabled && r.PollSeconds > 0 && time.Duration(r.PollSeconds)*time.Second < interval {
			interval = time.Duration(r.PollSeconds) * time.Second
		}
	}
	if interval < time.Second {
		interval = time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		e.pollDevice(d, &state)
		e.setState(state)
		select {
		case <-ctx.Done():
			state.State = "STOPPED"
			e.setState(state)
			return
		case <-ticker.C:
		}
	}
}

func (e *Engine) pollDevice(d config.Device, state *model.DeviceState) {
	client := mb.Client{
		Address: net.JoinHostPort(d.Host, fmt.Sprint(d.Port)),
		UnitID:  byte(d.UnitID),
		Timeout: d.Timeout(),
	}

	allOK := true
	anyOK := false

	for i, r := range d.Registers {
		if !r.Enabled {
			continue
		}
		if err := config.ValidateRegister(r); err != nil {
			state.Registers[i].Value = model.Value{Quality: model.QualityConfigError, Error: err.Error()}
			allOK = false
			continue
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

		var data []byte
		var err error
		tries := d.Retries + 1
		if tries < 1 {
			tries = 1
		}
		if tries > 3 {
			tries = 3
		}

		for n := 0; n < tries; n++ {
			data, err = client.Read(fc, r.Address, quantity(r))
			if err == nil {
				break
			}
		}

		if err != nil {
			state.Registers[i].Value = model.Value{Quality: model.QualityReadError, Error: humanError(err)}
			state.ErrorCount++
			state.LastError = humanError(err)
			allOK = false
			continue
		}

		raw, val, err := decode(r, data)
		if err != nil {
			state.Registers[i].Value = model.Value{Quality: model.QualityReadError, Error: err.Error()}
			state.ErrorCount++
			state.LastError = err.Error()
			allOK = false
			continue
		}

		now := time.Now()
		state.Registers[i].Value = model.Value{
			Raw: raw, Value: val, Timestamp: now, Quality: model.QualityGood,
		}
		state.LastSeen = &now
		anyOK = true
	}

	state.Connected = anyOK
	switch {
	case allOK && anyOK:
		state.State = "ONLINE"
		state.LastError = ""
	case anyOK:
		state.State = "DEGRADED"
	default:
		state.State = "OFFLINE"
	}
}

func humanError(err error) string {
	if ne, ok := err.(net.Error); ok && ne.Timeout() {
		return "Zeitueberschreitung"
	}
	return err.Error()
}

func LogSnapshot(states []model.DeviceState) {
	for _, s := range states {
		log.Printf("%s: status=%s connected=%v errors=%d", s.Name, s.State, s.Connected, s.ErrorCount)
	}
}
