package config

import "testing"

func TestValidateRegister64Bit(t *testing.T) {
	for _, dt := range []string{"uint64", "int64"} {
		r := Register{
			ID:          "r1",
			Name:        "64-bit",
			Enabled:     true,
			Type:        "holding",
			Address:     100,
			DataType:    dt,
			PollSeconds: 10,
		}
		if err := ValidateRegister(r); err != nil {
			t.Fatalf("%s rejected: %v", dt, err)
		}
	}
}

func TestValidateRegister64BitAddressRange(t *testing.T) {
	r := Register{
		ID:          "r1",
		Name:        "64-bit",
		Enabled:     true,
		Type:        "holding",
		Address:     65533,
		DataType:    "uint64",
		PollSeconds: 10,
	}
	if err := ValidateRegister(r); err == nil {
		t.Fatal("expected address range error for UINT64 at 65533")
	}
}

func TestValidateRejectsLongCCUAddress(t *testing.T) {
	r := Register{ID: "r1", Name: "R", Enabled: true, Type: "holding", Address: 0, DataType: "uint16", PollSeconds: 10}
	c := Config{Schema: 2, Devices: []Device{{ID: "d1", Name: "D", Enabled: true, Host: "192.168.1.2", Port: 502, UnitID: 1, TimeoutMS: 1500, CCUAddress: "MB1234567890123", Registers: []Register{r}}}}
	if errs := Validate(&c); len(errs) == 0 {
		t.Fatal("expected long ccu_address to be rejected")
	}
}

func TestValidateRejectsDuplicateCCUAddress(t *testing.T) {
	mk := func(id string) Device {
		return Device{ID: id, Name: id, Enabled: true, Host: "192.168.1.2", Port: 502, UnitID: 1, TimeoutMS: 1500, CCUAddress: "MB001002502001"}
	}
	c := Config{Schema: 2, Devices: []Device{mk("d1"), mk("d2")}}
	if errs := Validate(&c); len(errs) == 0 {
		t.Fatal("expected duplicate ccu_address to be rejected")
	}
}
