/*
Package main (openrgbinspect)

This is a standalone developer diagnostic tool for inspecting raw OpenRGB SDK controller data.
It is not part of the daemon runtime path, but it gives maintainers and users a simple way
to gather payload details when reporting unsupported or incorrectly detected OpenRGB hardware.
*/
package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"time"
)

const (
	headerSize = 16
	magic      = "ORGB"

	OPCODE_REQUEST_CONTROLLER_DATA  = 1
	OPCODE_REQUEST_PROTOCOL_VERSION = 40
)

type header struct {
	DeviceID uint32
	Opcode   uint32
	Length   uint32
}

func sendHeader(conn net.Conn, deviceID, opcode, length uint32) error {
	buf := new(bytes.Buffer)
	buf.WriteString(magic)
	if err := binary.Write(buf, binary.LittleEndian, deviceID); err != nil {
		return err
	}
	if err := binary.Write(buf, binary.LittleEndian, opcode); err != nil {
		return err
	}
	if err := binary.Write(buf, binary.LittleEndian, length); err != nil {
		return err
	}
	_, err := conn.Write(buf.Bytes())
	return err
}

func readHeader(conn net.Conn) (*header, error) {
	buf := make([]byte, headerSize)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return nil, err
	}
	if string(buf[:4]) != magic {
		return nil, fmt.Errorf("invalid magic: %q", string(buf[:4]))
	}
	return &header{
		DeviceID: binary.LittleEndian.Uint32(buf[4:8]),
		Opcode:   binary.LittleEndian.Uint32(buf[8:12]),
		Length:   binary.LittleEndian.Uint32(buf[12:16]),
	}, nil
}

func readPayload(conn net.Conn, n uint32) ([]byte, error) {
	buf := make([]byte, n)
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

func main() {
	conn, err := net.DialTimeout("tcp", "127.0.0.1:6742", 3*time.Second)
	if err != nil {
		fmt.Println("connect failed:", err)
		os.Exit(1)
	}
	defer conn.Close()

	if err := sendHeader(conn, 0, OPCODE_REQUEST_PROTOCOL_VERSION, 0); err != nil {
		fmt.Println("protocol request failed:", err)
		os.Exit(1)
	}
	h, err := readHeader(conn)
	if err != nil {
		fmt.Println("protocol header failed:", err)
		os.Exit(1)
	}
	payload, err := readPayload(conn, h.Length)
	if err != nil {
		fmt.Println("protocol payload failed:", err)
		os.Exit(1)
	}
	fmt.Println("Protocol version:", binary.LittleEndian.Uint32(payload[:4]))

	// motherboard was controller 1 in your earlier output
	if err := sendHeader(conn, 1, OPCODE_REQUEST_CONTROLLER_DATA, 0); err != nil {
		fmt.Println("controller request failed:", err)
		os.Exit(1)
	}
	h, err = readHeader(conn)
	if err != nil {
		fmt.Println("controller header failed:", err)
		os.Exit(1)
	}
	payload, err = readPayload(conn, h.Length)
	if err != nil {
		fmt.Println("controller payload failed:", err)
		os.Exit(1)
	}

	fmt.Println("Payload length:", len(payload))

	totalLen := binary.LittleEndian.Uint32(payload[0:4])
	deviceType := binary.LittleEndian.Uint32(payload[4:8])
	fmt.Println("TotalLen:", totalLen)
	fmt.Println("DeviceType:", deviceType)

	offset := 8
	name, _ := readORGBString(payload, &offset)
	vendor, _ := readORGBString(payload, &offset)
	desc, _ := readORGBString(payload, &offset)

	fmt.Println("Name:", name)
	fmt.Println("Vendor:", vendor)
	fmt.Println("Description:", desc)
	fmt.Println("Offset after strings:", offset)
	fmt.Printf("Next 64 bytes: % x\n", payload[offset:min(offset+64, len(payload))])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
