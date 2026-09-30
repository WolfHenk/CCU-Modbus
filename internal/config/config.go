package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
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
	PollSeconds int     `json:"poll_seconds"`
	Unit        string  `json:"unit"`
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
	if c.Schema == 0 {
		c.Schema = 1
	}
	if c.Listen == "" {
		c.Listen = "127.0.0.1:18701"
	}
	return &c, nil
}

func (d Device) Timeout() time.Duration {
	ms := d.TimeoutMS
	if ms <= 0 {
		ms = 1500
	}
	return time.Duration(ms) * time.Millisecond
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
	return nil
}
