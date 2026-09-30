package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	Schema  int      `json:"schema"`
	Listen  string   `json:"listen"`
	Devices []Device `json:"devices"`
}

type Device struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Enabled   bool       `json:"enabled"`
	Host      string     `json:"host"`
	Port      int        `json:"port"`
	UnitID    int        `json:"unit_id"`
	TimeoutMS int        `json:"timeout_ms"`
	Retries   int        `json:"retries"`
	Registers []Register `json:"registers"`
}

type Register struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Enabled     bool    `json:"enabled"`
	Type        string  `json:"type"`
	Address     uint16  `json:"address"`
	DataType    string  `json:"datatype"`
	Factor      float64 `json:"factor"`
	Offset      float64 `json:"offset"`
	ByteSwap    bool    `json:"byte_swap"`
	WordSwap    bool    `json:"word_swap"`
	PollSeconds  int     `json:"poll_seconds"`
	Unit         string  `json:"unit"`
	TrueMeansOpen bool   `json:"true_means_open,omitempty"`
}

func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	ApplyDefaults(&c)
	return &c, nil
}

func ApplyDefaults(c *Config) {
	if c.Schema == 0 {
		c.Schema = 1
	}
	if c.Listen == "" {
		c.Listen = "127.0.0.1:18701"
	}
	for i := range c.Devices {
		if c.Devices[i].Port == 0 {
			c.Devices[i].Port = 502
		}
		if c.Devices[i].TimeoutMS <= 0 {
			c.Devices[i].TimeoutMS = 1500
		}
	}
}

func WriteAtomic(path string, c *Config) error {
	ApplyDefaults(c)
	if errs := Validate(c); len(errs) > 0 {
		return errs[0]
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	tmp := path + ".new"
	if err := os.WriteFile(tmp, b, 0644); err != nil {
		return err
	}
	f, err := os.OpenFile(tmp, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if old, err := os.ReadFile(path); err == nil {
		_ = os.WriteFile(path+".bak", old, 0644)
	}
	return os.Rename(tmp, path)
}

func (d Device) Timeout() time.Duration {
	ms := d.TimeoutMS
	if ms <= 0 {
		ms = 1500
	}
	return time.Duration(ms) * time.Millisecond
}

func Validate(c *Config) []error {
	var out []error
	ids := map[string]bool{}
	for i, d := range c.Devices {
		for _, err := range ValidateDevice(d) {
			out = append(out, fmt.Errorf("Geraet %d: %w", i+1, err))
		}
		if ids[d.ID] {
			out = append(out, fmt.Errorf("Geraete-ID %q ist doppelt", d.ID))
		}
		ids[d.ID] = true
		rids := map[string]bool{}
		for n, r := range d.Registers {
			if err := ValidateRegister(r); err != nil {
				out = append(out, fmt.Errorf("%s, Register %d: %w", d.Name, n+1, err))
			}
			if rids[r.ID] {
				out = append(out, fmt.Errorf("%s: Register-ID %q ist doppelt", d.Name, r.ID))
			}
			rids[r.ID] = true
		}
	}
	return out
}

func ValidateDevice(d Device) []error {
	var out []error
	if strings.TrimSpace(d.ID) == "" {
		out = append(out, errors.New("device id fehlt"))
	}
	if strings.TrimSpace(d.Name) == "" {
		out = append(out, errors.New("device name fehlt"))
	}
	if strings.TrimSpace(d.Host) == "" {
		out = append(out, errors.New("host fehlt"))
	}
	if d.Port < 1 || d.Port > 65535 {
		out = append(out, fmt.Errorf("port %d ist ungueltig", d.Port))
	}
	if d.UnitID < 0 || d.UnitID > 255 {
		out = append(out, fmt.Errorf("unit_id %d ist ungueltig", d.UnitID))
	}
	if d.TimeoutMS < 100 || d.TimeoutMS > 30000 {
		out = append(out, fmt.Errorf("timeout_ms %d ist ungueltig", d.TimeoutMS))
	}
	if d.Retries < 0 || d.Retries > 2 {
		out = append(out, fmt.Errorf("retries %d ist ungueltig", d.Retries))
	}
	return out
}

func ValidateRegister(r Register) error {
	if strings.TrimSpace(r.ID) == "" {
		return errors.New("register id fehlt")
	}
	if strings.TrimSpace(r.Name) == "" {
		return errors.New("register name fehlt")
	}
	switch r.Type {
	case "coil", "discrete", "holding", "input":
	default:
		return fmt.Errorf("unbekannter registertyp %q", r.Type)
	}
	switch strings.ToLower(r.DataType) {
	case "bool", "uint16", "int16", "uint32", "int32", "float32":
	default:
		return fmt.Errorf("unbekannter datentyp %q", r.DataType)
	}
	if (r.Type == "coil" || r.Type == "discrete") && strings.ToLower(r.DataType) != "bool" {
		return errors.New("coil/discrete erfordert datatype bool")
	}
	if r.PollSeconds < 1 || r.PollSeconds > 86400 {
		return fmt.Errorf("poll_seconds %d ist ungueltig", r.PollSeconds)
	}
	return nil
}
