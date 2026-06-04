package openrgb

import (
	"OpenLinkHub/src/config"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"
)

type ConnectionState string

const (
	StateConnected     ConnectionState = "Connected"
	StateOffline       ConnectionState = "Offline"
	StateNotConfigured ConnectionState = "Not Configured"
)

var (
	statusMutex   sync.RWMutex
	currentStatus ConnectionState = StateOffline
	lastError     error
)

func GetStatus() (ConnectionState, error) {
	statusMutex.RLock()
	defer statusMutex.RUnlock()
	return currentStatus, lastError
}

func setStatus(state ConnectionState, err error) {
	statusMutex.Lock()
	defer statusMutex.Unlock()
	currentStatus = state
	lastError = err
}

var startMonitorOnce sync.Once

func startMonitorLoop() {
	go func() {
		for {
			time.Sleep(15 * time.Second)
			_ = HealthCheck()
		}
	}()
}

const (
	opcodeRequestControllerCount uint32 = 0
	opcodeRequestControllerData  uint32 = 1
	opcodeSetCustomMode          uint32 = 1100
	opcodeUpdateLeds             uint32 = 1050
)

type DiscoveredController struct {
	ID            int
	Name          string
	Version       string
	Location      string
	Serial        string
	Vendor        string
	Description   string
	ParsedStrings []string
	LEDCount      int
	Zones         []DiscoveredZone
}

type DiscoveredZone struct {
	Name           string
	Type           int32
	MinLEDCount    int
	MaxLEDCount    int
	LEDCount       int
	SegmentCount   int
	Classification string
}

func classifyZone(name string, ledCount int, minLEDCount int, maxLEDCount int, segmentCount int) string {
	lowerName := strings.ToLower(strings.TrimSpace(name))

	switch {
	case strings.Contains(lowerName, "addressable"):
		return "addressable"
	case strings.Contains(lowerName, "argb"):
		return "addressable"
	case strings.Contains(lowerName, "strip"):
		return "addressable"
	case strings.Contains(lowerName, "mainboard"):
		return "zone-based"
	case strings.Contains(lowerName, "logo"):
		return "zone-based"
	case strings.Contains(lowerName, "backplate"):
		return "zone-based"
	case segmentCount > 0:
		return "addressable"
	case ledCount > 1 && maxLEDCount > 1:
		return "addressable"
	default:
		return "zone-based"
	}
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

	raw := data[*offset : *offset+n]
	*offset += n
	if len(raw) > 0 && raw[len(raw)-1] == 0 {
		raw = raw[:len(raw)-1]
	}
	return string(raw), nil
}

func dial() (net.Conn, error) {
	port := config.GetConfig().OpenRGBPort
	if port <= 0 {
		err := fmt.Errorf("OpenRGB port is not configured")
		setStatus(StateNotConfigured, err)
		return nil, err
	}
	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		setStatus(StateOffline, err)
		return nil, err
	}
	setStatus(StateConnected, nil)
	return conn, nil
}

func HealthCheck() error {
	conn, err := dial()
	if err != nil {
		return err
	}
	defer conn.Close()

	packet := new(bytes.Buffer)
	if err := writeHeader(packet, 0, opcodeRequestControllerCount, 0); err != nil {
		setStatus(StateOffline, err)
		return err
	}
	if _, err := conn.Write(packet.Bytes()); err != nil {
		setStatus(StateOffline, err)
		return err
	}

	_, _, size, err := readHeader(conn)
	if err != nil {
		setStatus(StateOffline, err)
		return err
	}

	payload, err := readPayload(conn, size)
	if err != nil {
		setStatus(StateOffline, err)
		return err
	}
	if len(payload) < 4 {
		err := fmt.Errorf("controller count payload too short")
		setStatus(StateOffline, err)
		return err
	}

	setStatus(StateConnected, nil)
	return nil
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

func hasBytes(data []byte, offset int, n int) bool {
	return n >= 0 && offset >= 0 && offset+n <= len(data)
}

func readSaneORGBString(data []byte, offset *int, maxLen int) (string, error) {
	if !hasBytes(data, *offset, 2) {
		return "", fmt.Errorf("not enough data for string length")
	}
	n := int(binary.LittleEndian.Uint16(data[*offset : *offset+2]))
	if n < 0 || n > maxLen {
		return "", fmt.Errorf("implausible string length: %d", n)
	}
	if !hasBytes(data, *offset, 2+n) {
		return "", fmt.Errorf("string out of bounds")
	}
	return readORGBString(data, offset)
}

func parseZoneBlockAt(payload []byte, zoneOffset int) (int, int, []DiscoveredZone, bool, string, int) {
	if !hasBytes(payload, zoneOffset, 2) {
		return 0, 0, nil, false, "zoneCount out of bounds", 0
	}

	offset := zoneOffset
	zoneCountU16, err := readU16At(payload, &offset)
	if err != nil {
		return 0, 0, nil, false, "zoneCount read failed", 0
	}
	zoneCount := int(zoneCountU16)
	if zoneCount <= 0 || zoneCount > 128 {
		return zoneCount, 0, nil, false, fmt.Sprintf("implausible zoneCount=%d", zoneCount), 0
	}

	totalLEDs := 0
	discoveredZones := make([]DiscoveredZone, 0, zoneCount)
	score := 0
	for z := 0; z < zoneCount; z++ {
		zoneName, err := readSaneORGBString(payload, &offset, 256)
		if err != nil {
			return zoneCount, totalLEDs, discoveredZones, false, fmt.Sprintf("zone %d name rejected: %v", z, err), score
		}
		zoneName = strings.TrimSpace(zoneName)
		hadRealName := zoneName != ""
		if zoneName == "" {
			zoneName = fmt.Sprintf("Zone %d", z+1)
		}
		if !hasBytes(payload, offset, 16) {
			return zoneCount, totalLEDs, discoveredZones, false, fmt.Sprintf("zone %d metadata out of bounds", z), score
		}

		zoneTypeU32, err := readU32At(payload, &offset)
		if err != nil {
			return zoneCount, totalLEDs, discoveredZones, false, fmt.Sprintf("zone %d type read failed", z), score
		}
		ledsMin, err := readU32At(payload, &offset)
		if err != nil {
			return zoneCount, totalLEDs, discoveredZones, false, fmt.Sprintf("zone %d leds_min read failed", z), score
		}
		ledsMax, err := readU32At(payload, &offset)
		if err != nil {
			return zoneCount, totalLEDs, discoveredZones, false, fmt.Sprintf("zone %d leds_max read failed", z), score
		}
		numLEDs, err := readU32At(payload, &offset)
		if err != nil {
			return zoneCount, totalLEDs, discoveredZones, false, fmt.Sprintf("zone %d num_leds read failed", z), score
		}
		if ledsMin > 16384 || ledsMax > 16384 || numLEDs > 16384 {
			return zoneCount, totalLEDs, discoveredZones, false, fmt.Sprintf("zone %d led metadata implausible min=%d max=%d num=%d", z, ledsMin, ledsMax, numLEDs), score
		}
		totalLEDs += int(numLEDs)
		score += 20
		if hadRealName {
			score += 10
		}
		if numLEDs > 0 {
			score += 10
		}
		if ledsMin == numLEDs && ledsMax == numLEDs {
			score += 5
		}

		if !hasBytes(payload, offset, 2) {
			return zoneCount, totalLEDs, discoveredZones, false, fmt.Sprintf("zone %d matrix length out of bounds", z), score
		}
		matrixLen, err := readU16At(payload, &offset)
		if err != nil {
			return zoneCount, totalLEDs, discoveredZones, false, fmt.Sprintf("zone %d matrix length read failed", z), score
		}
		if !hasBytes(payload, offset, int(matrixLen)) {
			return zoneCount, totalLEDs, discoveredZones, false, fmt.Sprintf("zone %d matrix out of bounds", z), score
		}
		if err := skipBytes(payload, &offset, int(matrixLen)); err != nil {
			return zoneCount, totalLEDs, discoveredZones, false, fmt.Sprintf("zone %d matrix skip failed", z), score
		}

		if !hasBytes(payload, offset, 2) {
			return zoneCount, totalLEDs, discoveredZones, false, fmt.Sprintf("zone %d segment count out of bounds", z), score
		}
		segCount, err := readU16At(payload, &offset)
		if err != nil {
			return zoneCount, totalLEDs, discoveredZones, false, fmt.Sprintf("zone %d segment count read failed", z), score
		}
		if segCount > 128 {
			return zoneCount, totalLEDs, discoveredZones, false, fmt.Sprintf("zone %d segment count implausible: %d", z, segCount), score
		}
		for s := 0; s < int(segCount); s++ {
			if _, err := readSaneORGBString(payload, &offset, 256); err != nil {
				return zoneCount, totalLEDs, discoveredZones, false, fmt.Sprintf("zone %d segment %d name rejected: %v", z, s, err), score
			}
			if !hasBytes(payload, offset, 12) {
				return zoneCount, totalLEDs, discoveredZones, false, fmt.Sprintf("zone %d segment %d metadata out of bounds", z, s), score
			}
			if err := skipBytes(payload, &offset, 12); err != nil {
				return zoneCount, totalLEDs, discoveredZones, false, fmt.Sprintf("zone %d segment %d metadata skip failed", z, s), score
			}
		}

		classification := classifyZone(zoneName, int(numLEDs), int(ledsMin), int(ledsMax), int(segCount))
		discoveredZones = append(discoveredZones, DiscoveredZone{
			Name:           zoneName,
			Type:           int32(zoneTypeU32),
			MinLEDCount:    int(ledsMin),
			MaxLEDCount:    int(ledsMax),
			LEDCount:       int(numLEDs),
			SegmentCount:   int(segCount),
			Classification: classification,
		})
	}

	if totalLEDs <= 0 && hasBytes(payload, offset, 2) {
		ledListCount, err := readU16At(payload, &offset)
		if err == nil {
			totalLEDs = int(ledListCount)
			for i := 0; i < int(ledListCount); i++ {
				if _, err := readSaneORGBString(payload, &offset, 256); err != nil {
					break
				}
				if !hasBytes(payload, offset, 4) {
					break
				}
				if err := skipBytes(payload, &offset, 4); err != nil {
					break
				}
			}
		}
	}

	score += zoneCount * 25
	if zoneCount > 1 {
		score += 50
	}
	if totalLEDs > 0 {
		score += 20
	}

	return zoneCount, totalLEDs, discoveredZones, true, "", score
}

func findPlausibleZoneBlock(payload []byte, startOffset int) (int, int, int, []DiscoveredZone, bool, int, int, string, int) {
	if startOffset < 0 {
		startOffset = 0
	}

	windowEnd := startOffset + 16384
	if windowEnd > len(payload)-2 {
		windowEnd = len(payload) - 2
	}

	bestOffset := 0
	bestZoneCount := 0
	bestReason := ""
	bestScore := -1
	bestTotalLEDs := 0
	var bestZones []DiscoveredZone
	bestAccepted := false
	seen := make(map[int]struct{})
	for candidate := startOffset; candidate <= windowEnd; candidate++ {
		for delta := -4; delta <= 4; delta++ {
			probeOffset := candidate + delta
			if probeOffset < startOffset || probeOffset > windowEnd {
				continue
			}
			if _, ok := seen[probeOffset]; ok {
				continue
			}
			seen[probeOffset] = struct{}{}

			zoneCount, totalLEDs, discoveredZones, ok, reason, score := parseZoneBlockAt(payload, probeOffset)
			if ok {
				if !bestAccepted || score > bestScore || (score == bestScore && (zoneCount > bestZoneCount || totalLEDs > bestTotalLEDs)) {
					bestAccepted = true
					bestOffset = probeOffset
					bestZoneCount = zoneCount
					bestTotalLEDs = totalLEDs
					bestZones = discoveredZones
					bestScore = score
				}
				continue
			}
			if !bestAccepted && (score > bestScore || (score == bestScore && zoneCount > bestZoneCount)) {
				bestOffset = probeOffset
				bestZoneCount = zoneCount
				bestReason = reason
				bestScore = score
			}
		}
	}

	if bestAccepted {
		return bestOffset, bestZoneCount, bestTotalLEDs, bestZones, true, 0, 0, "", bestScore
	}

	return 0, 0, 0, nil, false, bestOffset, bestZoneCount, bestReason, bestScore
}

func findAnchoredZoneBlock(payload []byte, startOffset int) (int, int, int, []DiscoveredZone, bool, string, int) {
	anchors := []string{
		"24 Pin ATX Strip",
		"8 Pin GPU Strip",
		"RGB Header",
		"Aura Mainboard",
	}

	bestOffset := 0
	bestZoneCount := 0
	bestTotalLEDs := 0
	bestScore := -1
	var bestZones []DiscoveredZone
	bestAnchor := ""

	for _, anchor := range anchors {
		searchFrom := startOffset
		needle := []byte(anchor)
		for {
			idx := bytes.Index(payload[searchFrom:], needle)
			if idx < 0 {
				break
			}
			anchorPos := searchFrom + idx
			candidateStart := anchorPos - 512
			if candidateStart < startOffset {
				candidateStart = startOffset
			}
			candidateEnd := anchorPos
			seen := make(map[int]struct{})
			for candidate := candidateStart; candidate <= candidateEnd; candidate++ {
				for delta := -4; delta <= 4; delta++ {
					probeOffset := candidate + delta
					if probeOffset < startOffset || probeOffset > candidateEnd {
						continue
					}
					if _, ok := seen[probeOffset]; ok {
						continue
					}
					seen[probeOffset] = struct{}{}

					zoneCount, totalLEDs, discoveredZones, ok, _, score := parseZoneBlockAt(payload, probeOffset)
					if !ok {
						continue
					}

					matchedAnchor := false
					for _, zone := range discoveredZones {
						if strings.Contains(zone.Name, anchor) {
							matchedAnchor = true
							break
						}
					}
					if !matchedAnchor {
						continue
					}

					score += 200
					if score > bestScore || (score == bestScore && (zoneCount > bestZoneCount || totalLEDs > bestTotalLEDs)) {
						bestOffset = probeOffset
						bestZoneCount = zoneCount
						bestTotalLEDs = totalLEDs
						bestZones = discoveredZones
						bestScore = score
						bestAnchor = anchor
					}
				}
			}

			searchFrom = anchorPos + len(needle)
			if searchFrom >= len(payload) {
				break
			}
		}
	}

	if bestScore >= 0 {
		return bestOffset, bestZoneCount, bestTotalLEDs, bestZones, true, bestAnchor, bestScore
	}

	return 0, 0, 0, nil, false, "", 0
}

func isLegacyASUSMotherboard(name, vendor string) bool {
	n := strings.ToLower(name)
	v := strings.ToLower(vendor)
	return strings.Contains(n, "asus rog strix z890-e gaming wifi") || strings.Contains(v, "asus aura")
}

// parseControllerZoneAndLEDCount explicitly parses controller payload structure:
// [len][device_type][5 strings][mode_count][active_mode][mode data...][zone_count][zones...][led_list...][colors...]
// The mode section is treated as opaque and scanned past by searching for a plausible zone block.
func parseControllerZoneAndLEDCount(payload []byte) (int, int, []DiscoveredZone, error) {
	if len(payload) < 8 {
		return 0, 0, nil, fmt.Errorf("payload too short")
	}

	offset := 8 // skip total_len + device_type

	// name, vendor, description, fwVersion, location, serial
	for i := 0; i < 5; i++ {
		if _, err := readORGBString(payload, &offset); err != nil {
			return 0, 0, nil, err
		}
	}

	_, err := readU16At(payload, &offset)
	if err != nil {
		return 0, 0, nil, err
	}

	// active_mode int32
	if hasBytes(payload, offset, 4) {
		if err := skipBytes(payload, &offset, 4); err != nil {
			return 0, 0, nil, err
		}
	} else {
		return 0, 0, nil, fmt.Errorf("active_mode out of bounds")
	}

	_, zoneCount, totalLEDs, discoveredZones, ok, _, _ := findAnchoredZoneBlock(payload, offset)
	if ok {
		return zoneCount, totalLEDs, discoveredZones, nil
	}

	_, zoneCount, totalLEDs, discoveredZones, ok, _, _, _, _ = findPlausibleZoneBlock(payload, offset)
	if !ok {
		return 0, 0, nil, fmt.Errorf("no plausible zone block found")
	}

	return zoneCount, totalLEDs, discoveredZones, nil
}

func isImportableController(name, vendor string, ledCount int) bool {
	if name == "" && vendor == "" {
		return false
	}
	if isLegacyASUSMotherboard(name, vendor) {
		return true
	}

	return true
}

func DiscoverControllers() ([]DiscoveredController, error) {
	startMonitorOnce.Do(startMonitorLoop)
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
			vendor = ""
		}

		description, err := readORGBString(payload, &offset)
		if err != nil {
			description = ""
		}

		fwVersion, err := readORGBString(payload, &offset)
		if err != nil {
			fwVersion = ""
		}

		serial, err := readORGBString(payload, &offset)
		if err != nil {
			serial = ""
		}

		location, err := readORGBString(payload, &offset)
		if err != nil {
			location = ""
		}

		_, ledCount, zones, err := parseControllerZoneAndLEDCount(payload)
		if err != nil {
			ledCount = 0
			zones = nil
		}

		if !isImportableController(name, vendor, ledCount) {
			continue
		}

		result = append(result, DiscoveredController{
			ID:            int(i),
			Name:          name,
			Version:       fwVersion,
			Location:      location,
			Serial:        serial,
			Vendor:        vendor,
			Description:   description,
			ParsedStrings: []string{name, vendor, description, fwVersion, location, serial},
			LEDCount:      ledCount,
			Zones:         zones,
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

	packet := new(bytes.Buffer)
	payloadSize := uint32(4 + 2 + colorCount*4)
	if err := writeHeader(packet, controllerId, opcodeUpdateLeds, payloadSize); err != nil {
		return err
	}

	dataSize := payloadSize
	if err := binary.Write(packet, binary.LittleEndian, dataSize); err != nil {
		return err
	}

	if err := binary.Write(packet, binary.LittleEndian, uint16(colorCount)); err != nil {
		return err
	}

	color := []byte{0, 0, 0, 0}
	if len(rgb) >= 3 {
		color[0] = rgb[0]
		color[1] = rgb[1]
		color[2] = rgb[2]
	}

	for i := 0; i < colorCount; i++ {
		if _, err := packet.Write(color); err != nil {
			return err
		}
	}

	_, err = conn.Write(packet.Bytes())
	return err
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

	total := len(frame) / 3
	packet := new(bytes.Buffer)
	payloadSize := uint32(4 + 2 + total*4)
	if err := writeHeader(packet, controllerId, opcodeUpdateLeds, payloadSize); err != nil {
		return err
	}

	dataSize := payloadSize
	if err := binary.Write(packet, binary.LittleEndian, dataSize); err != nil {
		return err
	}

	if err := binary.Write(packet, binary.LittleEndian, uint16(total)); err != nil {
		return err
	}

	for i := 0; i < total; i++ {
		color := []byte{
			frame[i*3],
			frame[i*3+1],
			frame[i*3+2],
			0,
		}

		if _, err := packet.Write(color); err != nil {
			return err
		}
	}

	_, err = conn.Write(packet.Bytes())
	return err
}

func SendSingleLED(controllerId uint32, ledIndex uint32, rgb []byte) error {
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
	packet := new(bytes.Buffer)
	if err := writeHeader(packet, controllerId, opcodeUpdateSingleLED, 8); err != nil {
		return err
	}

	if err := binary.Write(packet, binary.LittleEndian, ledIndex); err != nil {
		return err
	}

	color := []byte{0, 0, 0, 0}
	if len(rgb) >= 3 {
		color[0] = rgb[0]
		color[1] = rgb[1]
		color[2] = rgb[2]
	}

	if _, err := packet.Write(color); err != nil {
		return err
	}

	_, err = conn.Write(packet.Bytes())
	return err
}

func SendFramePersistent(conn net.Conn, controllerId uint32, frame []byte) (net.Conn, error) {
	var err error
	if conn == nil {
		conn, err = dial()
		if err != nil {
			return nil, err
		}
		// Switch device into direct/custom mode
		packet := new(bytes.Buffer)
		if err := writeHeader(packet, controllerId, opcodeSetCustomMode, 0); err != nil {
			conn.Close()
			return nil, err
		}
		if _, err := conn.Write(packet.Bytes()); err != nil {
			conn.Close()
			return nil, err
		}
	}

	total := len(frame) / 3
	packet := new(bytes.Buffer)
	payloadSize := uint32(4 + 2 + total*4)
	if err := writeHeader(packet, controllerId, opcodeUpdateLeds, payloadSize); err != nil {
		conn.Close()
		return nil, err
	}

	dataSize := payloadSize
	if err := binary.Write(packet, binary.LittleEndian, dataSize); err != nil {
		conn.Close()
		return nil, err
	}

	if err := binary.Write(packet, binary.LittleEndian, uint16(total)); err != nil {
		conn.Close()
		return nil, err
	}

	for i := 0; i < total; i++ {
		color := []byte{
			frame[i*3],
			frame[i*3+1],
			frame[i*3+2],
			0,
		}

		if _, err := packet.Write(color); err != nil {
			conn.Close()
			return nil, err
		}
	}

	if _, err = conn.Write(packet.Bytes()); err != nil {
		conn.Close()
		return nil, err
	}

	return conn, nil
}
