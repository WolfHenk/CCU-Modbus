package ccuvirtual

import (
	"bytes"
	"fmt"
	"github.com/mdzio/go-hmccu/itf"
	"github.com/mdzio/go-hmccu/itf/vdevices"
	"github.com/mdzio/go-hmccu/itf/xmlrpc"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Wire-level XML-RPC test: no real CCU or Modbus hardware is contacted.
func TestVirtualRPCWireProtocol(t *testing.T) {
	container := vdevices.NewContainer()
	handler := vdevices.NewHandler("127.0.0.1", false, container, func(string) {})
	container.Synchronizer = handler
	device := vdevices.NewDevice("MB003207502001", "ModBus", handler)
	vdevices.NewMaintenanceChannel(device)
	input := vdevices.NewDigitalChannel(device, "DIGITAL_INPUT", "SWITCH_TRANSMITTER.STATE")
	input.ValueParamset().Parameter("STATE")
	analog := &vdevices.Channel{}
	analog.Init("ANALOG_INPUT_TRANSMITTER")
	device.AddChannel(analog)
	fp := vdevices.NewFloatParameter("VOLTAGE")
	fp.Description().Operations = itf.ParameterOperationRead | itf.ParameterOperationEvent
	analog.AddValueParam(fp)
	exact := &vdevices.Channel{}
	exact.Init("ANALOG_INPUT_TRANSMITTER")
	device.AddChannel(exact)
	sp := vdevices.NewStringParameter("VALUE")
	sp.Description().Operations = itf.ParameterOperationRead | itf.ParameterOperationEvent
	exact.AddValueParam(sp)
	if err := container.AddDevice(device); err != nil {
		t.Fatal(err)
	}
	dispatcher := itf.NewDispatcher()
	dispatcher.AddDeviceLayer(&persistentDeviceLayer{Handler: handler, onDelete: func(string) error { return nil }})
	registerCCUCompatibilityMethods(dispatcher)
	srv := httptest.NewServer(&xmlrpc.Handler{Dispatcher: dispatcher})
	defer srv.Close()
	client := xmlrpc.Client{Addr: strings.TrimPrefix(srv.URL, "http://")}
	if err := sp.InternalSetValue("18446744073709551615"); err != nil {
		t.Fatal(err)
	}
	if err := fp.InternalSetValue(float64(65535)); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ address, key, want string }{
		{"MB003207502001:3", "VALUE", "18446744073709551615"},
	} {
		res, err := client.Call("getParamset", xmlrpc.Values{xmlrpc.NewString(c.address), xmlrpc.NewString("VALUES")})
		if err != nil {
			t.Fatal(err)
		}
		if got := xmlrpc.Q(res).Key(c.key).String(); got != c.want {
			t.Errorf("%s.%s = %q, want %q", c.address, c.key, got, c.want)
		}
	}
	for _, tc := range []struct {
		method string
		params xmlrpc.Values
	}{
		{"listDevices", nil},
		{"getParamsetDescription", xmlrpc.Values{xmlrpc.NewString("MB003207502001:2"), xmlrpc.NewString("VALUES")}},
		{"getParamsetDescription", xmlrpc.Values{xmlrpc.NewString("MB003207502001:3"), xmlrpc.NewString("VALUES")}},
		{"getParamset", xmlrpc.Values{xmlrpc.NewString("MB003207502001:3"), xmlrpc.NewString("VALUES")}},
		{"setReadyConfig", nil},
	} {
		if _, err := client.Call(tc.method, tc.params); err != nil {
			t.Errorf("RPC %s: %v", tc.method, err)
		}
	}
}

// TestRPCRegistrationEventsReconnect exercises the outbound RPC sequence against
// a local simulated ReGa receiver, including re-registration after deinit.
func TestRPCRegistrationEventsReconnect(t *testing.T) {
	calls := make(chan string, 32)
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request: %v", err)
			return
		}
		var method string
		switch {
		case bytes.Contains(body, []byte("<methodName>listDevices</methodName>")):
			method = "listDevices"
		case bytes.Contains(body, []byte("<methodName>newDevices</methodName>")):
			method = "newDevices"
		case bytes.Contains(body, []byte("<methodName>event</methodName>")):
			method = "event"
		default:
			t.Errorf("unexpected RPC: %s", body)
			return
		}
		if method == "event" && (!bytes.Contains(body, []byte("18446744073709551615")) || !bytes.Contains(body, []byte("VALUE"))) {
			t.Errorf("event lost UINT64 decimal value or key: %s", body)
		}
		calls <- method
		w.Header().Set("Content-Type", "text/xml")
		if method == "listDevices" {
			fmt.Fprint(w, `<?xml version="1.0"?><methodResponse><params><param><value><array><data></data></array></value></param></params></methodResponse>`)
		} else {
			fmt.Fprint(w, `<?xml version="1.0"?><methodResponse><params><param><value><string></string></value></param></params></methodResponse>`)
		}
	}))
	defer receiver.Close()
	endpoint := strings.TrimPrefix(receiver.URL, "http://")
	container := vdevices.NewContainer()
	handler := vdevices.NewHandler("127.0.0.1", true, container, func(string) {})
	container.Synchronizer = handler
	dev := vdevices.NewDevice("MB003207502001", "ModBus", handler)
	ch := &vdevices.Channel{}
	ch.Init("ANALOG_INPUT_TRANSMITTER")
	dev.AddChannel(ch)
	p := vdevices.NewStringParameter("VALUE")
	p.Description().Operations = itf.ParameterOperationRead | itf.ParameterOperationEvent
	ch.AddValueParam(p)
	if err := container.AddDevice(dev); err != nil {
		t.Fatal(err)
	}
	defer handler.Close()
	expect := func(want string) {
		t.Helper()
		select {
		case got := <-calls:
			if got != want {
				t.Fatalf("RPC order: got %s, want %s", got, want)
			}
		case <-time.After(4 * time.Second):
			t.Fatalf("timeout waiting for %s", want)
		}
	}
	if err := handler.Init("http://"+endpoint, "CCU-Modbus"); err != nil {
		t.Fatal(err)
	}
	expect("listDevices")
	expect("newDevices")
	if err := p.InternalSetValue("18446744073709551615"); err != nil {
		t.Fatal(err)
	}
	expect("event")
	if err := handler.Deinit("http://" + endpoint); err != nil {
		t.Fatal(err)
	}
	if err := handler.Init("http://"+endpoint, "CCU-Modbus"); err != nil {
		t.Fatal(err)
	}
	expect("listDevices")
	expect("newDevices")
	if err := p.InternalSetValue("18446744073709551615"); err != nil {
		t.Fatal(err)
	}
	expect("event")
}
