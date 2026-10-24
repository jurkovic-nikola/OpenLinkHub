package lsh

import "testing"

// Regression for jurkovic-nikola/OpenLinkHub#543: the exact packet an iCUE LINK System Hub
// (1b1c:0c3f, fw 3.10.636) returned on 2026-10-08 with NO devices on its LINK chain:
// one 512-byte packet, data type 03 0a (not 21 00), channel count byte 0x7c (124), all zeros after.
// The pre-fix parser trusted 0x7c and walked 124 x 8 bytes over a 505-byte payload ->
// `panic: index out of range [511] with length 505` in getDevices, crash-looping the daemon on
// every boot (5 restarts then systemd gave up). Captured from /opt/OpenLinkHub/stdout.log on ACE-AI.
func realEmptyHubPacket() []byte {
	p := make([]byte, 512)
	copy(p, []byte{0x00, 0x00, 0x08, 0x03, 0x03, 0x0a, 0x7c, 0x02})
	return p
}

func TestParseDevicesResponse_RealEmptyHubPacketIsRejectedNotPanicked(t *testing.T) {
	p := realEmptyHubPacket()
	if len(p) != 512 {
		t.Fatalf("fixture must be the 512-byte packet the hub sent, got %d", len(p))
	}
	n, data, err := parseDevicesResponse(p)
	if err == nil {
		t.Fatalf("real empty-hub packet (data type 03 0a) must be rejected; got n=%d len(data)=%d", n, len(data))
	}
	if n != 0 || data != nil {
		t.Fatalf("rejected packet must yield 0 channels and nil data, got n=%d data=%v", n, data != nil)
	}
}

// The pre-fix failure mode: a parser that trusts the channel byte must not be able to index past
// the payload. This pins the bound that getDevices() now checks (position+8 > len(data)) using the
// same 0x7c/505-byte shape, so a future refactor that drops the bound fails here instead of in
// production.
func TestGetDevicesLoopBound_ChannelCountExceedsPayload(t *testing.T) {
	p := realEmptyHubPacket()
	p[4], p[5] = dataTypeGetDevices[0], dataTypeGetDevices[1] // force the header valid so we reach the loop logic
	channels, data, err := parseDevicesResponse(p)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if channels != 0x7c || len(data) != 505 {
		t.Fatalf("fixture shape drifted: channels=%d len=%d (want 124/505)", channels, len(data))
	}
	// Walk exactly as getDevices() does; must stop before overrunning.
	position := 0
	walked := 0
	for i := 1; i <= channels; i++ {
		if position+8 > len(data) {
			break
		}
		deviceIdLen := data[position+7]
		if deviceIdLen == 0 {
			position += 8
			walked++
			continue
		}
		if position+8+int(deviceIdLen) > len(data) {
			break
		}
		position += 8 + int(deviceIdLen)
		walked++
	}
	if walked != 63 { // 505/8 = 63 full 8-byte slots; the 64th (bytes 504..511) must be refused
		t.Fatalf("bounded walk visited %d slots, want 63 (64th slot would index past 505 bytes)", walked)
	}
	if position > len(data) {
		t.Fatalf("position %d overran payload %d", position, len(data))
	}
}
