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

type DiscoveredController struct {
	ID          int
	Name        string
	Vendor      string
	Description string
	LEDCount    int
}

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

func readU16At(data []byte, offset *int) (uint16, error) {
	if *offset+2 > len(data) {
		return 0, fmt.Errorf("not enough bytes for uint16")
	}
	v := binary.LittleEndian.Uint16(data[*offset : *offset+2])
	*offset += 2
	return v, nil
}

func readU32At(data []byte, offset *int) (uint32, error) {
	if *offset+4 > len(data) {
		return 0, fmt.Errorf("not enough bytes for uint32")
	}
	v := binary.LittleEndian.Uint32(data[*offset : *offset+4])
	*offset += 4
	return v, nil
}

func skipBytes(data []byte, offset *int, n int) error {
	if n < 0 || *offset+n > len(data) {
		return fmt.Errorf("out of bounds skip")
	}
	*offset += n
	return nil
}

func isLegacyASUSMotherboard(name, vendor string) bool {
	n := strings.ToLower(name)
	v := strings.ToLower(vendor)
	return strings.Contains(n, "asus rog strix z890-e gaming wifi") || strings.Contains(v, "asus aura")
}

// parseControllerZoneAndLEDCount explicitly parses controller payload structure:
// [len][device_type][6 strings][mode_count][active_mode][modes...][zone_count][zones...][led_list...][colors...]
// Returns total LEDs inferred from zones (or LED list fallback if zones sum is zero).
func parseControllerZoneAndLEDCount(payload []byte) (int, int, error) {
	if len(payload) < 8 {
		return 0, 0, fmt.Errorf("payload too short")
	}

	offset := 8 // skip total_len + device_type

	// name, vendor, description, fwVersion, serial, location
	for i := 0; i < 6; i++ {
		if _, err := readORGBString(payload, &offset); err != nil {
			return 0, 0, err
		}
	}

	modeCountU16, err := readU16At(payload, &offset)
	if err != nil {
		return 0, 0, err
	}
	modeCount := int(modeCountU16)

	// active_mode int32
	if err := skipBytes(payload, &offset, 4); err != nil {
		return 0, 0, err
	}

	// Modes (OpenLinkHub target-server-compatible packing observed in src/openrgb/openrgb.go):
	// mode_name (string), value(int32), flags(uint32), speed_min/max, brightness_min/max,
	// colors_min/max, speed, brightness, direction, color_mode (10x uint32), mode_color_count(uint16),
	// mode colors (count * 4 bytes)
	for i := 0; i < modeCount; i++ {
		if _, err := readORGBString(payload, &offset); err != nil {
			return 0, 0, err
		}
		// value + flags + 10 uint32 fields = 4 + 4 + 40 = 48 bytes
		if err := skipBytes(payload, &offset, 48); err != nil {
			return 0, 0, err
		}
		modeColorCount, err := readU16At(payload, &offset)
		if err != nil {
			return 0, 0, err
		}
		if err := skipBytes(payload, &offset, int(modeColorCount)*4); err != nil {
			return 0, 0, err
		}
	}

	zoneCountU16, err := readU16At(payload, &offset)
	if err != nil {
		return 0, 0, err
	}
	zoneCount := int(zoneCountU16)

	totalLEDs := 0
	for z := 0; z < zoneCount; z++ {
		if _, err := readORGBString(payload, &offset); err != nil {
			return 0, 0, err
		}

		// zone_type int32
		if err := skipBytes(payload, &offset, 4); err != nil {
			return 0, 0, err
		}

		// leds_min, leds_max, num_leds (uint32 each)
		if _, err := readU32At(payload, &offset); err != nil { // leds_min
			return 0, 0, err
		}
		if _, err := readU32At(payload, &offset); err != nil { // leds_max
			return 0, 0, err
		}
		numLEDs, err := readU32At(payload, &offset)
		if err != nil {
			return 0, 0, err
		}
		totalLEDs += int(numLEDs)

		// matrix byte length (uint16) + matrix payload bytes
		matrixLen, err := readU16At(payload, &offset)
		if err != nil {
			return 0, 0, err
		}
		if err := skipBytes(payload, &offset, int(matrixLen)); err != nil {
			return 0, 0, err
		}

		segCount, err := readU16At(payload, &offset)
		if err != nil {
			return 0, 0, err
		}
		for s := 0; s < int(segCount); s++ {
			if _, err := readORGBString(payload, &offset); err != nil {
				return 0, 0, err
			}
			// segment_type(int32), segment_start_idx(uint32), segment_led_count(uint32)
			if err := skipBytes(payload, &offset, 12); err != nil {
				return 0, 0, err
			}
		}
	}

	// Fallback from explicit LED list if zone sum not usable.
	if totalLEDs <= 0 {
		ledListCount, err := readU16At(payload, &offset)
		if err == nil {
			totalLEDs = int(ledListCount)
			for i := 0; i < int(ledListCount); i++ {
				if _, err := readORGBString(payload, &offset); err != nil {
					break
				}
				// led index uint32
				if err := skipBytes(payload, &offset, 4); err != nil {
					break
				}
			}
		}
	}

	return zoneCount, totalLEDs, nil
}

func isImportableController(name, vendor string, ledCount int) bool {
	if name == "" && vendor == "" {
		return false
	}
	if isLegacyASUSMotherboard(name, vendor) {
		return true
	}

	s := strings.ToLower(name + " " + vendor)

	// Allow explicit Strimer matches even if LED parsing returned 0.
	if strings.Contains(s, "strimer") || strings.Contains(s, "strimmer") {
		return true
	}

	if ledCount <= 0 {
		return false
	}

	allowPhrases := []string{
		"motherboard",
		"mainboard",
		"asus aura",
		"aura sync",
		"lian li strimer",
		"lian li strimmer",
	}
	for _, p := range allowPhrases {
		if strings.Contains(s, p) {
			return true
		}
	}
	return false
}

func DiscoverControllers() ([]DiscoveredController, error) {
	conn, err := dial()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	packet := new(bytes.Buffer)
	if err := writeHeader(packet, 0, opcodeRequestControllerCount, 0); err != nil {
		return nil, err
	}
	if _, err := conn.Write(packet.Bytes()); err != nil {
		return nil, err
	}

	_, _, size, err := readHeader(conn)
	if err != nil {
		return nil, err
	}
	payload, err := readPayload(conn, size)
	if err != nil {
		return nil, err
	}
	if len(payload) < 4 {
		return nil, fmt.Errorf("controller count payload too short")
	}

	count := binary.LittleEndian.Uint32(payload[:4])
	result := make([]DiscoveredController, 0, count)

	for i := uint32(0); i < count; i++ {
		packet.Reset()
		if err := writeHeader(packet, i, opcodeRequestControllerData, 0); err != nil {
			continue
		}
		if _, err := conn.Write(packet.Bytes()); err != nil {
			continue
		}

		_, _, size, err = readHeader(conn)
		if err != nil {
			continue
		}
		payload, err = readPayload(conn, size)
		if err != nil || len(payload) < 8 {
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
		description, err := readORGBString(payload, &offset)
		if err != nil {
			description = ""
		}

		_, ledCount, err := parseControllerZoneAndLEDCount(payload)
		fmt.Println("DEBUG OpenRGB:", name, "|", vendor, "| LEDCount:", ledCount, "| err:", err)
		if err != nil && !isLegacyASUSMotherboard(name, vendor) {
			continue
		}

		if !isImportableController(name, vendor, ledCount) {
			continue
		}

		result = append(result, DiscoveredController{
			ID:          int(i),
			Name:        name,
			Vendor:      vendor,
			Description: description,
			LEDCount:    ledCount,
		})
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no importable OpenRGB controllers discovered")
	}
	return result, nil
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
