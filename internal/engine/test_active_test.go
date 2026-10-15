package engine

import (
	"context"
	"encoding/binary"
	"github.com/WolfHenk/ccu-modbus/internal/config"
	"io"
	"net"
	"sync/atomic"
	"testing"
	"time"
)

func TestActiveUsesExistingConnection(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	var connections atomic.Int32
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			c, e := listener.Accept()
			if e != nil {
				return
			}
			connections.Add(1)
			go func() {
				defer c.Close()
				for {
					head := make([]byte, 7)
					if _, err := io.ReadFull(c, head); err != nil {
						return
					}
					n := int(binary.BigEndian.Uint16(head[4:6])) - 1
					pdu := make([]byte, n)
					if _, err := io.ReadFull(c, pdu); err != nil {
						return
					}
					reply := []byte{pdu[0], 1, 1}
					out := make([]byte, 7+len(reply))
					copy(out, head)
					binary.BigEndian.PutUint16(out[4:6], uint16(len(reply)+1))
					copy(out[7:], reply)
					if _, err := c.Write(out); err != nil {
						return
					}
				}
			}()
		}
	}()
	port := listener.Addr().(*net.TCPAddr).Port
	d := config.Device{ID: "same-client", Name: "Same Client", Enabled: true, Host: "127.0.0.1", Port: port, UnitID: 1, TimeoutMS: 400, Registers: []config.Register{{ID: "coil", Name: "Coil", Enabled: true, Type: "coil", Address: 0, DataType: "bool", PollSeconds: 1}}}
	e := New()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	e.Start(ctx, &config.Config{Devices: []config.Device{d}})
	defer e.Stop()
	var online bool
	for i := 0; i < 30; i++ {
		for _, v := range e.Snapshot() {
			if v.ID == d.ID && v.State == "ONLINE" {
				online = true
			}
		}
		if online {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !online {
		t.Fatal("poller never online")
	}
	initial := connections.Load()
	cr, used := e.TestActive(d, nil)
	if !used || !cr.OK {
		t.Fatalf("connection test: %+v used=%v", cr, used)
	}
	rr, used := e.TestActive(d, &d.Registers[0])
	if !used || !rr.OK || rr.Value != true {
		t.Fatalf("register test: %+v used=%v", rr, used)
	}
	if connections.Load() != initial {
		t.Fatalf("extra tcp connection: before=%d after=%d", initial, connections.Load())
	}
	other := d
	other.UnitID = 2
	if _, used := e.TestActive(other, nil); used {
		t.Fatal("changed unit must fall back to standalone test")
	}
	listener.Close()
}
