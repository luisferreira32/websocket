package websocket

import (
	"bufio"
	"encoding/binary"
	"errors"
	"io"
	"math/rand"
	"net"
)

type OpCode byte

const (
	OpContinuation OpCode = 0x0
	OpText         OpCode = 0x1
	OpBinary       OpCode = 0x2
	OpClose        OpCode = 0x8
	OpPing         OpCode = 0x9
	OpPong         OpCode = 0xA
)

type Connection struct {
	c    net.Conn
	r    *bufio.Reader
	w    *bufio.Writer
	mask bool
}

func newConnection(c net.Conn, isClient bool) Connection {
	return Connection{
		c:    c,
		r:    bufio.NewReader(c),
		w:    bufio.NewWriter(c),
		mask: isClient,
	}
}

func (c Connection) Close() error {
	closeFrame := encodeFrame(true, OpClose, []byte{0x03, 0xE8}, c.mask)

	_, err := c.w.Write(closeFrame)
	if err == nil {
		err = c.w.Flush()
	}

	if closeErr := c.c.Close(); err == nil {
		err = closeErr
	}

	return err
}

func (c Connection) Write(p []byte) (n int, err error) {
	frame := encodeFrame(true, OpText, p, c.mask)

	n, err = c.w.Write(frame)
	if err != nil {
		return 0, err
	}

	err = c.w.Flush()
	if err != nil {
		return 0, err
	}

	return len(p), nil
}

func (c Connection) Read(p []byte) (n int, err error) {
	header := make([]byte, 2)
	if _, err := io.ReadFull(c.r, header); err != nil {
		return 0, err
	}

	_ = (header[0] & 0x80) != 0
	opcode := OpCode(header[0] & 0x0F)
	masked := (header[1] & 0x80) != 0
	payloadLen := int(header[1] & 0x7F)

	if opcode >= OpClose {
		switch opcode {
		case OpClose:
			payload := make([]byte, payloadLen)
			if payloadLen > 0 {
				if _, err := io.ReadFull(c.r, payload); err != nil {
					return 0, err
				}
			}

			closeFrame := encodeFrame(true, OpClose, payload, c.mask)
			if _, err := c.w.Write(closeFrame); err != nil {
				return 0, err
			}
			if err := c.w.Flush(); err != nil {
				return 0, err
			}

			return 0, io.EOF

		case OpPing:
			payload := make([]byte, payloadLen)
			if _, err := io.ReadFull(c.r, payload); err != nil {
				return 0, err
			}

			pongFrame := encodeFrame(true, OpPong, payload, c.mask)
			if _, err := c.w.Write(pongFrame); err != nil {
				return 0, err
			}
			if err := c.w.Flush(); err != nil {
				return 0, err
			}

			return c.Read(p)

		case OpPong:
			if payloadLen > 0 {
				skipBuf := make([]byte, payloadLen)
				if _, err := io.ReadFull(c.r, skipBuf); err != nil {
					return 0, err
				}
			}
			return c.Read(p)
		}
	}

	var extendedLen int
	if payloadLen == 126 {
		extLen := make([]byte, 2)
		if _, err := io.ReadFull(c.r, extLen); err != nil {
			return 0, err
		}
		extendedLen = int(binary.BigEndian.Uint16(extLen))
	} else if payloadLen == 127 {
		extLen := make([]byte, 8)
		if _, err := io.ReadFull(c.r, extLen); err != nil {
			return 0, err
		}
		extendedLen = int(binary.BigEndian.Uint64(extLen))
	} else {
		extendedLen = payloadLen
	}

	if extendedLen > len(p) {
		return 0, errors.New("payload too large for buffer")
	}

	var mask []byte
	if masked {
		mask = make([]byte, 4)
		if _, err := io.ReadFull(c.r, mask); err != nil {
			return 0, err
		}
	}

	payload := p[:extendedLen]
	if _, err := io.ReadFull(c.r, payload); err != nil {
		return 0, err
	}

	if masked {
		for i := 0; i < extendedLen; i++ {
			payload[i] ^= mask[i%4]
		}
	} else if !c.mask {
		closeFrame := encodeFrame(true, OpClose, []byte{0x03, 0xEA}, c.mask)
		c.w.Write(closeFrame)
		c.w.Flush()
		return 0, errors.New("protocol violation: client frames must be masked")
	}

	return extendedLen, nil
}

func encodeFrame(fin bool, opcode OpCode, payload []byte, mask bool) []byte {
	length := len(payload)
	var headerSize int
	if length < 126 {
		headerSize = 2
	} else if length < 65536 {
		headerSize = 4
	} else {
		headerSize = 10
	}

	if mask {
		headerSize += 4
	}

	frame := make([]byte, headerSize+length)

	frame[0] = byte(opcode)
	if fin {
		frame[0] |= 0x80
	}

	if mask {
		frame[1] = 0x80
	} else {
		frame[1] = 0x00
	}

	if length < 126 {
		frame[1] |= byte(length)
	} else if length < 65536 {
		frame[1] |= 126
		binary.BigEndian.PutUint16(frame[2:4], uint16(length))
	} else {
		frame[1] |= 127
		binary.BigEndian.PutUint64(frame[2:10], uint64(length))
	}

	if mask {
		maskKey := make([]byte, 4)
		rand.Read(maskKey)

		maskPos := headerSize - 4
		copy(frame[maskPos:maskPos+4], maskKey)

		for i := 0; i < length; i++ {
			frame[headerSize+i] = payload[i] ^ maskKey[i%4]
		}
	} else {
		copy(frame[headerSize:], payload)
	}

	return frame
}
