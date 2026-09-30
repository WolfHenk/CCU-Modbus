package modbus

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

// ExceptionError represents a Modbus exception response.
type ExceptionError struct {
	Function byte
	Code     byte
}

func (e *ExceptionError) Error() string {
	return fmt.Sprintf("modbus exception %d", e.Code)
}

// Client maintains one TCP connection for one Modbus device.
// Any I/O error invalidates the connection. The next call reconnects
// automatically. One client is used by one device worker, but the mutex
// keeps the type safe if that changes later.
type Client struct {
	Address string
	UnitID  byte
	Timeout time.Duration

	mu   sync.Mutex
	conn net.Conn
	tx   uint16
}

func NewClient(address string, unitID byte, timeout time.Duration) *Client {
	return &Client{Address: address, UnitID: unitID, Timeout: timeout}
}

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closeLocked()
}

func (c *Client) closeLocked() error {
	if c.conn == nil {
		return nil
	}
	err := c.conn.Close()
	c.conn = nil
	return err
}

func (c *Client) connectLocked() error {
	if c.conn != nil {
		return nil
	}
	conn, err := net.DialTimeout("tcp", c.Address, c.Timeout)
	if err != nil {
		return err
	}
	c.conn = conn
	return nil
}

func (c *Client) failLocked(err error) error {
	_ = c.closeLocked()
	return err
}

func (c *Client) Read(function byte, address, quantity uint16) ([]byte, error) {
	if function < 1 || function > 4 {
		return nil, fmt.Errorf("unsupported function %d", function)
	}
	if quantity == 0 {
		return nil, errors.New("quantity must be > 0")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.connectLocked(); err != nil {
		return nil, err
	}
	_ = c.conn.SetDeadline(time.Now().Add(c.Timeout))

	c.tx++
	if c.tx == 0 {
		c.tx++
	}
	tx := c.tx

	pdu := make([]byte, 5)
	pdu[0] = function
	binary.BigEndian.PutUint16(pdu[1:3], address)
	binary.BigEndian.PutUint16(pdu[3:5], quantity)

	adu := make([]byte, 7+len(pdu))
	binary.BigEndian.PutUint16(adu[0:2], tx)
	binary.BigEndian.PutUint16(adu[2:4], 0)
	binary.BigEndian.PutUint16(adu[4:6], uint16(1+len(pdu)))
	adu[6] = c.UnitID
	copy(adu[7:], pdu)

	if _, err := c.conn.Write(adu); err != nil {
		return nil, c.failLocked(err)
	}

	hdr := make([]byte, 7)
	if _, err := io.ReadFull(c.conn, hdr); err != nil {
		return nil, c.failLocked(err)
	}
	if binary.BigEndian.Uint16(hdr[0:2]) != tx {
		return nil, c.failLocked(errors.New("transaction id mismatch"))
	}
	if binary.BigEndian.Uint16(hdr[2:4]) != 0 {
		return nil, c.failLocked(errors.New("invalid protocol id"))
	}
	if hdr[6] != c.UnitID {
		return nil, c.failLocked(errors.New("unit id mismatch"))
	}

	length := int(binary.BigEndian.Uint16(hdr[4:6]))
	if length < 2 || length > 260 {
		return nil, c.failLocked(fmt.Errorf("invalid response length %d", length))
	}
	body := make([]byte, length-1)
	if _, err := io.ReadFull(c.conn, body); err != nil {
		return nil, c.failLocked(err)
	}
	if len(body) < 2 {
		return nil, c.failLocked(errors.New("short modbus response"))
	}
	if body[0] == function|0x80 {
		// A Modbus exception is a valid Modbus response; keep the TCP
		// connection alive and report only this request as failed.
		return nil, &ExceptionError{Function: function, Code: body[1]}
	}
	if body[0] != function {
		return nil, c.failLocked(fmt.Errorf("unexpected function %d", body[0]))
	}

	byteCount := int(body[1])
	if byteCount != len(body)-2 {
		return nil, c.failLocked(errors.New("byte count mismatch"))
	}
	return body[2:], nil
}


func (c *Client) WriteSingleCoil(address uint16, value bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.connectLocked(); err != nil {
		return err
	}
	_ = c.conn.SetDeadline(time.Now().Add(c.Timeout))

	c.tx++
	if c.tx == 0 {
		c.tx++
	}
	tx := c.tx

	raw := uint16(0x0000)
	if value {
		raw = 0xFF00
	}
	pdu := make([]byte, 5)
	pdu[0] = 5
	binary.BigEndian.PutUint16(pdu[1:3], address)
	binary.BigEndian.PutUint16(pdu[3:5], raw)

	adu := make([]byte, 12)
	binary.BigEndian.PutUint16(adu[0:2], tx)
	binary.BigEndian.PutUint16(adu[2:4], 0)
	binary.BigEndian.PutUint16(adu[4:6], 6)
	adu[6] = c.UnitID
	copy(adu[7:], pdu)

	if _, err := c.conn.Write(adu); err != nil {
		return c.failLocked(err)
	}

	resp := make([]byte, 12)
	if _, err := io.ReadFull(c.conn, resp); err != nil {
		return c.failLocked(err)
	}
	if binary.BigEndian.Uint16(resp[0:2]) != tx {
		return c.failLocked(errors.New("transaction id mismatch"))
	}
	if binary.BigEndian.Uint16(resp[2:4]) != 0 {
		return c.failLocked(errors.New("invalid protocol id"))
	}
	if resp[6] != c.UnitID {
		return c.failLocked(errors.New("unit id mismatch"))
	}
	if resp[7] == 0x85 {
		return &ExceptionError{Function: 5, Code: resp[8]}
	}
	if resp[7] != 5 {
		return c.failLocked(fmt.Errorf("unexpected function %d", resp[7]))
	}
	if binary.BigEndian.Uint16(resp[8:10]) != address || binary.BigEndian.Uint16(resp[10:12]) != raw {
		return c.failLocked(errors.New("write echo mismatch"))
	}
	return nil
}
