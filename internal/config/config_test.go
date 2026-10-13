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
