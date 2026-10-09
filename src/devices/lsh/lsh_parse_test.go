package lsh

import "testing"

func TestParseDevicesResponse(t *testing.T) {
	if _, _, err := parseDevicesResponse(make([]byte, 3)); err == nil {
		t.Fatal("short response accepted")
	}
	// single 512-byte packet whose data type is not 21 00 (no second chunk appended)
	bad := make([]byte, 512)
	bad[4], bad[5], bad[6] = 0x10, 0x00, 0xff
	if _, _, err := parseDevicesResponse(bad); err == nil {
		t.Fatal("wrong data type accepted")
	}
	ok := make([]byte, 1020)
	ok[4], ok[5], ok[6] = 0x21, 0x00, 0x7c
	n, data, err := parseDevicesResponse(ok)
	if err != nil || n != 0x7c || len(data) != 1013 {
		t.Fatalf("n=%d len=%d err=%v", n, len(data), err)
	}
}
