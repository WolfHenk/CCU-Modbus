package config

import "testing"

func TestCoilDirectionValidation(t *testing.T) {
	base := Register{ID: "c1", Name: "coil", Enabled: true, Type: "coil", Address: 1, DataType: "bool", PollSeconds: 3}
	for _, d := range []string{"", "input", "output"} {
		r := base
		r.CoilDirection = d
		if err := ValidateRegister(r); err != nil {
			t.Errorf("valid direction %q: %v", d, err)
		}
	}
	for _, d := range []string{"invalid", "Input"} {
		r := base
		r.CoilDirection = d
		if err := ValidateRegister(r); err == nil {
			t.Errorf("invalid direction %q accepted", d)
		}
	}
	r := base
	r.Type = "discrete"
	r.CoilDirection = "input"
	if err := ValidateRegister(r); err == nil {
		t.Error("non-coil direction accepted")
	}
}
