package openrgb

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strings"
)

const (
	opcodeRequestControllerCount uint32 = 0
	opcodeRequestControllerData  uint32 = 1
	opcodeSetCustomMode          uint32 = 1100
	opcodeUpdateLeds             uint32 = 1050
)

func writeHeader(buf *bytes.Buffer, controllerId uint32, opcode uint32, size uint32) error {
	if _, err := buf.WriteString("ORGB"); err != nil {
		return err
	}
	if err := binary.Write(buf, binary.LittleEndian, controllerId); err != nil {
		return err
	}
	if err := binary.Write(buf, binary.LittleEndian, opcode); err != nil {
		return err
	}
	if err := binary.Write(buf, binary.LittleEndian, size); err != nil {
		return err
	}
	return nil
}

func readHeader(conn net.Conn) (uint32, uint32, uint32, error) {
	buf := make([]byte, 16)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return 0, 0, 0, err
	}

	if string(buf[:4]) != "ORGB" {
		return 0, 0, 0, fmt.Errorf("invalid OpenRGB header magic")
	}

	controllerId := binary.LittleEndian.Uint32(buf[4:8])
	opcode := binary.LittleEndian.Uint32(buf[8:12])
	size := binary.LittleEndian.Uint32(buf[12:16])

	return controllerId, opcode, size, nil
}

func readPayload(conn net.Conn, size uint32) ([]byte, error) {
	buf := make([]byte, size)
	_, err := io.ReadFull(conn, buf)
	return buf, err
}

func readORGBString(data []byte, offset *int) (string, error) {
	if *offset+2 > len(data) {
		return "", fmt.Errorf("not enough data for string length")
	}

	n := int(binary.LittleEndian.Uint16(data[*offset : *offset+2]))
	*offset += 2

	if *offset+n > len(data) {
		return "", fmt.Errorf("invalid string length: %d", n)
	}

	s := string(data[*offset : *offset+n])
	*offset += n
	return s, nil
}

func dial() (net.Conn, error) {
	return net.Dial("tcp", "127.0.0.1:6742")
}

func FindControllerIDByNameOrVendor(nameMatch string, vendorMatch string) (int, error) {
	conn, err := dial()
	if err != nil {
		return -1, err
	}
	defer conn.Close()

	packet := new(bytes.Buffer)
	if err := writeHeader(packet, 0, opcodeRequestControllerCount, 0); err != nil {
		return -1, err
	}
	if _, err := conn.Write(packet.Bytes()); err != nil {
		return -1, err
	}

	_, _, size, err := readHeader(conn)
	if err != nil {
		return -1, err
	}

	payload, err := readPayload(conn, size)
	if err != nil {
		return -1, err
	}
	if len(payload) < 4 {
		return -1, fmt.Errorf("controller count payload too short")
	}

	count := binary.LittleEndian.Uint32(payload[:4])

	nameMatch = strings.ToLower(nameMatch)
	vendorMatch = strings.ToLower(vendorMatch)

	for i := uint32(0); i < count; i++ {
		packet.Reset()
		if err := writeHeader(packet, i, opcodeRequestControllerData, 0); err != nil {
			return -1, err
		}
		if _, err := conn.Write(packet.Bytes()); err != nil {
			return -1, err
		}

		_, _, size, err = readHeader(conn)
		if err != nil {
			return -1, err
		}

		payload, err = readPayload(conn, size)
		if err != nil {
			return -1, err
		}

		if len(payload) < 8 {
			continue
		}

		offset := 8

		name, err := readORGBString(payload, &offset)
		if err != nil {
			continue
		}
		vendor, err := readORGBString(payload, &offset)
		if err != nil {
			continue
		}

		nameOK := nameMatch != "" && strings.Contains(strings.ToLower(name), nameMatch)
		vendorOK := vendorMatch != "" && strings.Contains(strings.ToLower(vendor), vendorMatch)

		if nameOK || vendorOK {
			return int(i), nil
		}
	}

	return -1, fmt.Errorf("no matching OpenRGB controller found")
}

func SendColor(controllerId uint32, colorCount int, rgb []byte) error {
	conn, err := dial()
	if err != nil {
		return err
	}
	defer conn.Close()

	// Switch device into direct/custom mode
	{
		packet := new(bytes.Buffer)
		if err := writeHeader(packet, controllerId, opcodeSetCustomMode, 0); err != nil {
			return err
		}
		if _, err := conn.Write(packet.Bytes()); err != nil {
			return err
		}
	}

	const opcodeUpdateSingleLED uint32 = 1052

	color := []byte{0, 0, 0, 0}
	if len(rgb) >= 3 {
		color[0] = rgb[0]
		color[1] = rgb[1]
		color[2] = rgb[2]
	}

	for i := 0; i < colorCount; i++ {
		packet := new(bytes.Buffer)

		if err := writeHeader(packet, controllerId, opcodeUpdateSingleLED, 8); err != nil {
			return err
		}

		if err := binary.Write(packet, binary.LittleEndian, uint32(i)); err != nil {
			return err
		}

		if _, err := packet.Write(color); err != nil {
			return err
		}

		if _, err := conn.Write(packet.Bytes()); err != nil {
			return err
		}
	}

	return nil
}

func SendFrame(controllerId uint32, frame []byte) error {
	conn, err := dial()
	if err != nil {
		return err
	}
	defer conn.Close()

	// Switch device into direct/custom mode
	{
		packet := new(bytes.Buffer)
		if err := writeHeader(packet, controllerId, opcodeSetCustomMode, 0); err != nil {
			return err
		}
		if _, err := conn.Write(packet.Bytes()); err != nil {
			return err
		}
	}

	const opcodeUpdateSingleLED uint32 = 1052

	total := len(frame) / 3

	for i := 0; i < total; i++ {
		packet := new(bytes.Buffer)

		if err := writeHeader(packet, controllerId, opcodeUpdateSingleLED, 8); err != nil {
			return err
		}

		if err := binary.Write(packet, binary.LittleEndian, uint32(i)); err != nil {
			return err
		}

		color := []byte{
			frame[i*3],
			frame[i*3+1],
			frame[i*3+2],
			0,
		}

		if _, err := packet.Write(color); err != nil {
			return err
		}

		if _, err := conn.Write(packet.Bytes()); err != nil {
			return err
		}
	}

	return nil
}
