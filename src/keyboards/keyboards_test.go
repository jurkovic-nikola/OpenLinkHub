package keyboards

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
)

func TestK60RGBProGKeyPacketIndex(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "database", "keyboard", "k60rgbpro.json"))
	if err != nil {
		t.Fatalf("read K60 RGB PRO layout: %v", err)
	}

	var keyboard Keyboard
	if err = json.Unmarshal(data, &keyboard); err != nil {
		t.Fatalf("decode K60 RGB PRO layout: %v", err)
	}
	if keyboard.Version != 2 {
		t.Fatalf("layout version = %d, want 2", keyboard.Version)
	}

	var gKey, fiveKey *Key
	for _, row := range keyboard.Row {
		for _, key := range row.Keys {
			key := key
			switch {
			case key.KeyName == "G" && slices.Contains(key.KeyHash, "1024"):
				gKey = &key
			case key.KeyName == "5" && slices.Contains(key.KeyHash, "17179869184"):
				fiveKey = &key
			}
		}
	}
	if gKey == nil || !reflect.DeepEqual(gKey.PacketIndex, []int{6}) {
		t.Fatalf("G key packet index = %#v, want [6]", gKey)
	}
	if fiveKey == nil || !reflect.DeepEqual(fiveKey.PacketIndex, []int{30}) {
		t.Fatalf("5 key packet index = %#v, want [30]", fiveKey)
	}
}
