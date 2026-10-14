package ccuvirtual

import (
	"github.com/WolfHenk/ccu-modbus/internal/config"
	"github.com/WolfHenk/ccu-modbus/internal/model"
	"github.com/mdzio/go-hmccu/itf"
	"github.com/mdzio/go-hmccu/itf/vdevices"
	"testing"
)

func TestNumericChannelTypesAndAddresses(t *testing.T) {
	d := config.Device{ID: "d", Enabled: true, Registers: []config.Register{
		{ID: "u64", Name: "uint64", Enabled: true, Type: "holding", DataType: "uint64"},
		{ID: "binary", Name: "contact", Enabled: true, Type: "discrete", DataType: "bool"},
		{ID: "u16", Name: "byteword", Enabled: true, Type: "input", DataType: "uint16"},
		{ID: "i32", Name: "dword", Enabled: true, Type: "holding", DataType: "int32"},
		{ID: "f32", Name: "float32", Enabled: true, Type: "input", DataType: "float32"},
		{ID: "i64", Name: "int64", Enabled: true, Type: "holding", DataType: "int64"},
	}}
	vd := vdevices.NewContainer()
	vd.Synchronizer = vdevices.NewHandler("127.0.0.1", false, vd, func(string) {})
	i := &Integration{devices: vd, switches: map[switchKey]*vdevices.DigitalChannel{}, inputs: map[switchKey]*vdevices.DigitalChannel{}, inputInvert: map[switchKey]bool{}, numeric: map[switchKey]*vdevices.FloatParameter{}, exact64: map[switchKey]*vdevices.StringParameter{}, stateSynced: map[switchKey]bool{}, numericSynced: map[switchKey]string{}}
	i.addDevice(d, "MB003205502001", nil)
	devices := i.devices.Devices()
	if len(devices) != 1 {
		t.Fatalf("devices %d", len(devices))
	}
	ch := devices[0].Channels()
	if len(ch) != 7 {
		t.Fatalf("channels %d, want 7", len(ch))
	}
	if ch[1].Description().Type != "DIGITAL_INPUT" {
		t.Fatalf("existing binary channel shifted: %v", ch[1].Description().Type)
	}
	for n := 2; n < 7; n++ {
		if ch[n].Description().Type != "ANALOG_INPUT_TRANSMITTER" {
			t.Fatalf("channel %d: %v", n, ch[n].Description().Type)
		}
	}
	sig := virtualChannelSignature(d)
	if len(sig) != 6 || sig[0] != "discrete|binary" || sig[1] != "holding|u64|uint64" {
		t.Fatalf("signature: %v", sig)
	}
	// No remote CCU event target exists in this isolated unit test.
	for _, p := range i.numeric {
		p.Description().Operations = itf.ParameterOperationRead
	}
	for _, p := range i.exact64 {
		p.Description().Operations = itf.ParameterOperationRead
	}
	i.regaReady = true
	i.syncStates([]model.DeviceState{{ID: "d", Registers: []model.RegisterState{
		{ID: "u64", Value: model.Value{Quality: model.QualityGood, Value: "18446744073709551615"}},
		{ID: "u16", Value: model.Value{Quality: model.QualityGood, Value: float64(65535)}},
		{ID: "i32", Value: model.Value{Quality: model.QualityGood, Value: float64(-2147483648)}},
		{ID: "i64", Value: model.Value{Quality: model.QualityGood, Value: "-9223372036854775808"}},
	}}})
	if got := i.exact64[switchKey{"d", "u64"}].Value(); got != "18446744073709551615" {
		t.Errorf("uint64: %v", got)
	}
	if got := i.exact64[switchKey{"d", "i64"}].Value(); got != "-9223372036854775808" {
		t.Errorf("int64: %v", got)
	}
	if got := i.numeric[switchKey{"d", "u16"}].Value(); got != float64(65535) {
		t.Errorf("uint16: %v", got)
	}
	if got := i.numeric[switchKey{"d", "i32"}].Value(); got != float64(-2147483648) {
		t.Errorf("int32: %v", got)
	}
}
