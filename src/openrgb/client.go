package openrgb

import (
	"LumenForge/src/config"
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strconv"
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

const (
	connectTimeout     = 2 * time.Second
	ioTimeout          = 3 * time.Second
	operationTimeout   = 10 * time.Second
	maxPayloadSize     = 16 * 1024 * 1024
	maxControllerCount = 1024
	maxLEDCount        = 65535
)

var (
	sdkAddress = func() (string, error) {
		port := config.GetConfig().OpenRGBPort
		if port <= 0 || port > 65535 {
			return "", fmt.Errorf("OpenRGB port is not configured")
		}
		return net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), nil
	}
	dialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		dialer := &net.Dialer{Timeout: connectTimeout}
		return dialer.DialContext(ctx, network, address)
	}
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

// SetNotConfigured marks the SDK client inactive because no imports are configured.
func SetNotConfigured() {
	setStatus(StateNotConfigured, nil)
}

// SetDisconnected records an SDK communication failure.
func SetDisconnected(err error) {
	setStatus(StateOffline, err)
}

// SetConnected records a successful SDK protocol exchange.
func SetConnected() {
	setStatus(StateConnected, nil)
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
	if size > maxPayloadSize {
		return 0, 0, 0, fmt.Errorf("OpenRGB payload size %d exceeds limit %d", size, maxPayloadSize)
	}

	return controllerId, opcode, size, nil
}

func readPayload(conn net.Conn, size uint32) ([]byte, error) {
	if size > maxPayloadSize {
		return nil, fmt.Errorf("OpenRGB payload size %d exceeds limit %d", size, maxPayloadSize)
	}
	buf := make([]byte, size)
	_, err := io.ReadFull(conn, buf)
	return buf, err
}

func readResponse(conn net.Conn, expectedControllerID, expectedOpcode uint32) ([]byte, error) {
	controllerID, opcode, size, err := readHeader(conn)
	if err != nil {
		return nil, err
	}
	if controllerID != expectedControllerID {
		return nil, fmt.Errorf("unexpected OpenRGB controller ID %d, expected %d", controllerID, expectedControllerID)
	}
	if opcode != expectedOpcode {
		return nil, fmt.Errorf("unexpected OpenRGB opcode %d, expected %d", opcode, expectedOpcode)
	}
	return readPayload(conn, size)
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

func dial(ctx context.Context) (net.Conn, error) {
	address, err := sdkAddress()
	if err != nil {
		return nil, err
	}
	return dialContext(ctx, "tcp", address)
}

func HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
	defer cancel()
	return HealthCheckContext(ctx)
}

func HealthCheckContext(ctx context.Context) error {
	conn, err := dial(ctx)
	if err != nil {
		SetDisconnected(err)
		return err
	}
	defer conn.Close()
	stopContextWatch := watchContext(ctx, conn)
	defer stopContextWatch()

	packet := new(bytes.Buffer)
	if err := writeHeader(packet, 0, opcodeRequestControllerCount, 0); err != nil {
		setStatus(StateOffline, err)
		return err
	}
	if err := writePacket(ctx, conn, packet.Bytes()); err != nil {
		setStatus(StateOffline, err)
		return err
	}

	if err := setReadDeadline(ctx, conn); err != nil {
		SetDisconnected(err)
		return err
	}
	payload, err := readResponse(conn, 0, opcodeRequestControllerCount)
	if err != nil {
		setStatus(StateOffline, err)
		return err
	}
	if len(payload) != 4 {
		err := fmt.Errorf("invalid controller count payload size %d", len(payload))
		setStatus(StateOffline, err)
		return err
	}
	if count := binary.LittleEndian.Uint32(payload); count > maxControllerCount {
		err := fmt.Errorf("OpenRGB controller count %d exceeds limit %d", count, maxControllerCount)
		setStatus(StateOffline, err)
		return err
	}

	setStatus(StateConnected, nil)
	return nil
}

func deadlineFor(ctx context.Context, limit time.Duration) time.Time {
	deadline := time.Now().Add(limit)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		return ctxDeadline
	}
	return deadline
}

func setReadDeadline(ctx context.Context, conn net.Conn) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return conn.SetReadDeadline(deadlineFor(ctx, ioTimeout))
}

func setWriteDeadline(ctx context.Context, conn net.Conn) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return conn.SetWriteDeadline(deadlineFor(ctx, ioTimeout))
}

func writePacket(ctx context.Context, conn net.Conn, data []byte) error {
	if err := setWriteDeadline(ctx, conn); err != nil {
		return err
	}
	for len(data) > 0 {
		written, err := conn.Write(data)
		if err != nil {
			return err
		}
		if written == 0 {
			return io.ErrUnexpectedEOF
		}
		data = data[written:]
	}
	return nil
}

func watchContext(ctx context.Context, conn net.Conn) func() {
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.SetDeadline(time.Now())
		case <-done:
		}
	}()
	return func() {
		close(done)
	}
}

func FindControllerIDByNameOrVendor(nameMatch string, vendorMatch string) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
	defer cancel()

	controllers, err := DiscoverControllersContext(ctx)
	if err != nil {
		return -1, err
	}

	nameMatch = strings.ToLower(nameMatch)
	vendorMatch = strings.ToLower(vendorMatch)
	for _, controller := range controllers {
		nameOK := nameMatch != "" && strings.Contains(strings.ToLower(controller.Name), nameMatch)
		vendorOK := vendorMatch != "" && strings.Contains(strings.ToLower(controller.Vendor), vendorMatch)

		if nameOK || vendorOK {
			return controller.ID, nil
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
// [len][device_type][6 strings][mode_count][active_mode][mode data...][zone_count][zones...][led_list...][colors...]
// The mode section is treated as opaque and scanned past by searching for a plausible zone block.
func parseControllerZoneAndLEDCount(payload []byte) (int, int, []DiscoveredZone, error) {
	if len(payload) < 8 {
		return 0, 0, nil, fmt.Errorf("payload too short")
	}

	offset := 8 // skip total_len + device_type

	// name, vendor, description, fwVersion, location, serial
	for i := 0; i < 6; i++ {
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
	ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
	defer cancel()
	return DiscoverControllersContext(ctx)
}

func DiscoverControllersContext(ctx context.Context) ([]DiscoveredController, error) {
	return discoverControllersContext(ctx, true)
}

// DiscoverControllersStatusNeutralContext performs one bounded SDK discovery
// without changing the process-wide importer connection status.
func DiscoverControllersStatusNeutralContext(ctx context.Context) ([]DiscoveredController, error) {
	return discoverControllersContext(ctx, false)
}

func discoverControllersContext(ctx context.Context, updateStatus bool) ([]DiscoveredController, error) {
	recordFailure := func(err error) {
		if updateStatus {
			SetDisconnected(err)
		}
	}

	conn, err := dial(ctx)
	if err != nil {
		recordFailure(err)
		return nil, err
	}
	defer conn.Close()
	stopContextWatch := watchContext(ctx, conn)
	defer stopContextWatch()

	packet := new(bytes.Buffer)
	if err := writeHeader(packet, 0, opcodeRequestControllerCount, 0); err != nil {
		recordFailure(err)
		return nil, err
	}
	if err := writePacket(ctx, conn, packet.Bytes()); err != nil {
		recordFailure(err)
		return nil, err
	}

	if err := setReadDeadline(ctx, conn); err != nil {
		recordFailure(err)
		return nil, err
	}
	payload, err := readResponse(conn, 0, opcodeRequestControllerCount)
	if err != nil {
		recordFailure(err)
		return nil, err
	}
	if len(payload) != 4 {
		err = fmt.Errorf("invalid controller count payload size %d", len(payload))
		recordFailure(err)
		return nil, err
	}

	count := binary.LittleEndian.Uint32(payload[:4])
	if count > maxControllerCount {
		err = fmt.Errorf("OpenRGB controller count %d exceeds limit %d", count, maxControllerCount)
		recordFailure(err)
		return nil, err
	}
	result := make([]DiscoveredController, 0, count)

	for i := uint32(0); i < count; i++ {
		packet.Reset()
		if err := writeHeader(packet, i, opcodeRequestControllerData, 0); err != nil {
			continue
		}
		if err := writePacket(ctx, conn, packet.Bytes()); err != nil {
			recordFailure(err)
			return nil, err
		}

		if err := setReadDeadline(ctx, conn); err != nil {
			recordFailure(err)
			return nil, err
		}
		payload, err = readResponse(conn, i, opcodeRequestControllerData)
		if err != nil {
			recordFailure(err)
			return nil, err
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

	if updateStatus {
		setStatus(StateConnected, nil)
	}
	return result, nil
}

func SendColor(controllerId uint32, colorCount int, rgb []byte) error {
	err := SendColorContext(context.Background(), controllerId, colorCount, rgb)
	if err != nil {
		SetDisconnected(err)
	}
	return err
}

// SendColorContext sends a static color and allows the caller to cancel the SDK operation.
func SendColorContext(parent context.Context, controllerId uint32, colorCount int, rgb []byte) error {
	if colorCount < 0 || colorCount > maxLEDCount {
		return fmt.Errorf("invalid OpenRGB LED count %d", colorCount)
	}
	ctx, cancel := boundedOperationContext(parent)
	defer cancel()
	conn, err := dial(ctx)
	if err != nil {
		return outputOperationError(ctx, err)
	}
	defer conn.Close()
	stopContextWatch := watchContext(ctx, conn)
	defer stopContextWatch()

	// Switch device into direct/custom mode
	{
		packet := new(bytes.Buffer)
		if err := writeHeader(packet, controllerId, opcodeSetCustomMode, 0); err != nil {
			return err
		}
		if err := writePacket(ctx, conn, packet.Bytes()); err != nil {
			return outputOperationError(ctx, err)
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

	err = writePacket(ctx, conn, packet.Bytes())
	return outputOperationError(ctx, err)
}

func SendFrame(controllerId uint32, frame []byte) error {
	err := SendFrameContext(context.Background(), controllerId, frame)
	if err != nil {
		SetDisconnected(err)
	}
	return err
}

// SendFrameContext sends an LED frame and allows the caller to cancel the SDK operation.
func SendFrameContext(parent context.Context, controllerId uint32, frame []byte) error {
	if len(frame)%3 != 0 || len(frame)/3 > maxLEDCount {
		return fmt.Errorf("invalid OpenRGB frame length %d", len(frame))
	}
	ctx, cancel := boundedOperationContext(parent)
	defer cancel()
	conn, err := dial(ctx)
	if err != nil {
		return outputOperationError(ctx, err)
	}
	defer conn.Close()
	stopContextWatch := watchContext(ctx, conn)
	defer stopContextWatch()

	// Switch device into direct/custom mode
	{
		packet := new(bytes.Buffer)
		if err := writeHeader(packet, controllerId, opcodeSetCustomMode, 0); err != nil {
			return err
		}
		if err := writePacket(ctx, conn, packet.Bytes()); err != nil {
			return outputOperationError(ctx, err)
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

	err = writePacket(ctx, conn, packet.Bytes())
	return outputOperationError(ctx, err)
}

func boundedOperationContext(parent context.Context) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithTimeout(parent, operationTimeout)
}

func outputOperationError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	return err
}

func SendSingleLED(controllerId uint32, ledIndex uint32, rgb []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
	defer cancel()
	conn, err := dial(ctx)
	if err != nil {
		SetDisconnected(err)
		return err
	}
	defer conn.Close()
	stopContextWatch := watchContext(ctx, conn)
	defer stopContextWatch()

	// Switch device into direct/custom mode
	{
		packet := new(bytes.Buffer)
		if err := writeHeader(packet, controllerId, opcodeSetCustomMode, 0); err != nil {
			return err
		}
		if err := writePacket(ctx, conn, packet.Bytes()); err != nil {
			SetDisconnected(err)
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

	err = writePacket(ctx, conn, packet.Bytes())
	if err != nil {
		SetDisconnected(err)
	}
	return err
}

func SendFramePersistent(conn net.Conn, controllerId uint32, frame []byte) (net.Conn, error) {
	if len(frame)%3 != 0 || len(frame)/3 > maxLEDCount {
		return nil, fmt.Errorf("invalid OpenRGB frame length %d", len(frame))
	}
	ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
	defer cancel()
	var err error
	if conn == nil {
		conn, err = dial(ctx)
		if err != nil {
			SetDisconnected(err)
			return nil, err
		}
		// Switch device into direct/custom mode
		packet := new(bytes.Buffer)
		if err := writeHeader(packet, controllerId, opcodeSetCustomMode, 0); err != nil {
			conn.Close()
			return nil, err
		}
		if err := writePacket(ctx, conn, packet.Bytes()); err != nil {
			conn.Close()
			SetDisconnected(err)
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

	if err = writePacket(ctx, conn, packet.Bytes()); err != nil {
		conn.Close()
		SetDisconnected(err)
		return nil, err
	}

	return conn, nil
}
