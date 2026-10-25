package common

import (
	"os"
	"path/filepath"
	"testing"
)

// fakeSysfs builds a minimal sysfs tree with hidraw0 under a USB interface of usb1/1-2
// and points sysRoot at it.
func fakeSysfs(t *testing.T, withVendor bool) {
	t.Helper()
	root := t.TempDir()
	usbDev := filepath.Join(root, "devices/pci0000:00/0000:00:14.0/usb1/1-2")
	hidDev := filepath.Join(usbDev, "1-2:1.0/0003:1B1C:1C08.0001")
	if err := os.MkdirAll(hidDev, 0755); err != nil {
		t.Fatal(err)
	}
	if withVendor {
		if err := os.WriteFile(filepath.Join(usbDev, "idVendor"), []byte("1b1c\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	hidraw := filepath.Join(root, "class/hidraw/hidraw0")
	if err := os.MkdirAll(hidraw, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(hidDev, filepath.Join(hidraw, "device")); err != nil {
		t.Fatal(err)
	}

	old := sysRoot
	sysRoot, _ = filepath.EvalSymlinks(root)
	t.Cleanup(func() { sysRoot = old })
}

func TestGetShortUSBDevPath(t *testing.T) {
	fakeSysfs(t, true)
	got, err := GetShortUSBDevPath("hidraw0")
	if err != nil {
		t.Fatal(err)
	}
	if want := "/devices/pci0000:00/0000:00:14.0/usb1/1-2"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestGetShortUSBDevPathErrors(t *testing.T) {
	fakeSysfs(t, false)
	if _, err := GetShortUSBDevPath("hidraw0"); err == nil {
		t.Fatal("expected error when no USB device node has idVendor")
	}
	if _, err := GetShortUSBDevPath("hidraw9"); err == nil {
		t.Fatal("expected error for missing hidraw node")
	}
}
