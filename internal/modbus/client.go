package modbus

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync/atomic"
	"time"
)

var transaction uint32

type Client struct {
	Address string
	UnitID  byte
	Timeout time.Duration
}

func (c Client) Read(function byte, address, quantity uint16) ([]byte, error) {
	if function < 1 || function > 4 {
		return nil, fmt.Errorf("unsupported function %d", function)
	}
	if quantity == 0 {
		return nil, errors.New("quantity must be > 0")
	}

	conn, err := net.DialTimeout("tcp", c.Address, c.Timeout)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(c.Timeout))

	tx := uint16(atomic.AddUint32(&transaction, 1))
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

	if _, err := conn.Write(adu); err != nil {
		return nil, err
	}

	hdr := make([]byte, 7)
	if _, err := io.ReadFull(conn, hdr); err != nil {
		return nil, err
	}
	if binary.BigEndian.Uint16(hdr[0:2]) != tx {
		return nil, errors.New("transaction id mismatch")
	}
	if binary.BigEndian.Uint16(hdr[2:4]) != 0 {
		return nil, errors.New("invalid protocol id")
	}

	length := int(binary.BigEndian.Uint16(hdr[4:6]))
	if length < 2 || length > 260 {
		return nil, fmt.Errorf("invalid response length %d", length)
	}
	body := make([]byte, length-1)
	if _, err := io.ReadFull(conn, body); err != nil {
		return nil, err
	}
	if len(body) < 2 {
		return nil, errors.New("short modbus response")
	}
	if body[0] == function|0x80 {
		return nil, fmt.Errorf("modbus exception %d", body[1])
	}
	if body[0] != function {
		return nil, fmt.Errorf("unexpected function %d", body[0])
	}

	byteCount := int(body[1])
	if byteCount != len(body)-2 {
		return nil, errors.New("byte count mismatch")
	}
	return body[2:], nil
}
