package server

import (
	"LumenForge/src/devices/openrgbimport"
	"encoding/json"
	"reflect"
	"testing"
)

func TestSnapshotDeviceForResponsePreservesOpenRGBJSONShape(t *testing.T) {
	config := &openrgbimport.DeviceConfig{
		Serial:  "openrgb-import-api-snapshot",
		Product: "API Snapshot Device",
		Zones:   []openrgbimport.ZoneConfig{{Name: "Zone 1", LedCount: 2}},
	}
	live := &openrgbimport.Device{
		Product:            config.Product,
		Serial:             config.Serial,
		IsOpenRGB:          true,
		DisplaySerial:      "external-api-serial",
		DisplaySerialLabel: "SERIAL",
		LEDCount:           2,
		ZoneAmount:         1,
		Version:            "1.0",
		Description:        "API snapshot",
		Config:             config,
		RGBModes:           []string{"static", "off"},
	}

	presented := snapshotDeviceForResponse(live)
	snapshot, ok := presented.(*openrgbimport.DeviceSnapshot)
	if !ok {
		t.Fatalf("response device type = %T, want *openrgbimport.DeviceSnapshot", presented)
	}
	liveJSON, err := json.Marshal(live)
	if err != nil {
		t.Fatal(err)
	}
	snapshotJSON, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	var liveFields map[string]json.RawMessage
	var snapshotFields map[string]json.RawMessage
	if err = json.Unmarshal(liveJSON, &liveFields); err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(snapshotJSON, &snapshotFields); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(liveFields, snapshotFields) {
		t.Fatalf("snapshot JSON changed response shape:\nlive: %s\nsnapshot: %s", liveJSON, snapshotJSON)
	}

	live.Config.Zones[0].Name = "mutated"
	live.RGBModes[0] = "mutated"
	if snapshot.Config.Zones[0].Name != "Zone 1" || snapshot.RGBModes[0] != "static" {
		t.Fatal("response snapshot shares mutable slices with the live device")
	}

	ordinary := &struct{ Name string }{Name: "ordinary"}
	if got := snapshotDeviceForResponse(ordinary); got != ordinary {
		t.Fatal("non-OpenRGB response device was replaced")
	}
}
