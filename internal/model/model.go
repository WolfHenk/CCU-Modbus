package model

import "time"

type Quality string

const (
	QualityGood        Quality = "GOOD"
	QualityStale       Quality = "STALE"
	QualityReadError   Quality = "READ_ERROR"
	QualityConfigError Quality = "CONFIG_ERROR"
	QualityDisabled    Quality = "DISABLED"
)

type Value struct {
	Raw       any       `json:"raw,omitempty"`
	Value     any       `json:"value,omitempty"`
	Timestamp time.Time `json:"timestamp,omitempty"`
	Quality   Quality   `json:"quality"`
	Error     string    `json:"error,omitempty"`
}

type RegisterState struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Value Value  `json:"value"`
}

type DeviceState struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Host       string          `json:"host"`
	Port       int             `json:"port"`
	UnitID     uint8           `json:"unit_id"`
	Connected  bool            `json:"connected"`
	State      string          `json:"state"`
	LastSeen   *time.Time      `json:"last_seen,omitempty"`
	LastError  string          `json:"last_error,omitempty"`
	ErrorCount uint64          `json:"error_count"`
	Registers  []RegisterState `json:"registers"`
}
