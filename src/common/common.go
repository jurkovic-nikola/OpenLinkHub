package common

import (
	"bytes"
	"fmt"
	"golang.org/x/image/draw"
	"image"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Device struct {
	ProductType uint16
	Product     string
	Serial      string
	Firmware    string
	Image       string
	GetDevice   interface{}
	Instance    interface{}
	Hidden      bool
}

// runUdevadmInfo executes `udevadm info --query=property` on a given device and returns the result.
func runUdevadmInfo(devicePath string) (string, error) {
	// Construct the udevadm command to get device properties
	cmd := exec.Command("udevadm", "info", "--query=property", "--name="+devicePath)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	// Run the command
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to run udevadm: %v", err)
	}

	return out.String(), nil
}

// GetDeviceUSBPath retrieves the ID_PATH_WITH_USB_REVISION properties from udevadm info output
func GetDeviceUSBPath(devicePath string) (string, error) {
	output, err := runUdevadmInfo(devicePath)
	if err != nil {
		return "", err
	}

	var idPath string
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "ID_PATH_WITH_USB_REVISION=") {
			idPath = strings.TrimPrefix(line, "ID_PATH_WITH_USB_REVISION=")
		}
	}

	return idPath, nil
}

// GetShortUSBDevPath will get USB device PCI path
func GetShortUSBDevPath(device string) (string, error) {
	start := fmt.Sprintf("/sys/class/hidraw/%s/device", device)

	path, err := filepath.EvalSymlinks(start)
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(path, "idVendor")); err == nil {
			break // Found the USB device node
		}

		parent := filepath.Dir(path)
		if parent == path || parent == "/" {
			return "", fmt.Errorf("usb device node not found for %s", device)
		}
		path = parent
	}
	devPath := strings.TrimPrefix(path, "/sys")
	return devPath, nil
}

// FileExists will check if given filename exists
func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return err == nil
}

// ReadFile will return file with given path
func ReadFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// ParseUEvent will parse kernel event when device is plugged / unplugged
func ParseUEvent(msg []byte) map[string]string {
	result := make(map[string]string)
	parts := bytes.Split(msg, []byte{0x00})
	for _, part := range parts {
		if kv := strings.SplitN(string(part), "=", 2); len(kv) == 2 {
			result[kv[0]] = kv[1]
		}
	}
	return result
}

// Lerp performs linear interpolation between two values
func Lerp(a, b, t float64) float64 {
	return a + t*(b-a)
}

// Clamp function restricts the value within the specified range [min, max].
func Clamp(value, min, max int) int {
	if value < min {
		return min
	} else if value > max {
		return max
	}

	return value
}

// Atoi string to integer
func Atoi(s string) int {
	val, _ := strconv.Atoi(s)
	return val
}

// FClamp function restricts the value within the specified range [min, max].
func FClamp(value, min, max float64) float64 {
	if value < min {
		return min
	} else if value > max {
		return max
	}
	return value
}

// ProcessMultiChunkPacket will process a byte array in chunks with a specified max size
func ProcessMultiChunkPacket(data []byte, maxChunkSize int) [][]byte {
	var result [][]byte

	for len(data) > 0 {
		// Calculate the end index for the current chunk
		end := maxChunkSize
		if len(data) < maxChunkSize {
			end = len(data)
		}

		// Get the current chunk to process
		chunk := data[:end]

		// Append the chunk to the result
		result = append(result, chunk)

		// If the current chunk size is less than max size, break the loop
		if len(data) <= maxChunkSize {
			break
		}

		// Move to the next chunk
		data = data[end:]
	}

	return result
}

func InBetween(i, min, max float32) bool {
	if (i >= min) && (i <= max) {
		return true
	} else {
		return false
	}
}

// FractionOfByte will return a fraction of given value
func FractionOfByte(ratio float64, percentage *float64) int {
	if percentage != nil {
		ratio = *percentage / 100
	}
	if ratio != 0 {
		if ratio < 0 || ratio > 1 {
			return 0
		}
		return int(math.Round(ratio * 255))
	}
	return 0
}

// IsValidExtension will compare a path extension with a given extension
func IsValidExtension(path, extension string) bool {
	ext := filepath.Ext(path)
	if ext != extension {
		return false
	}
	return true
}

func IndexOfString(slice []string, target string) int {
	for i, v := range slice {
		if v == target {
			return i
		}
	}
	return -1 // Return -1 if the target is not found
}

func FromLinear11(bytes []byte) float32 {
	val := int(bytes[2]) | int(bytes[3])<<8
	fraction := val & 0x7FF
	if fraction > 1023 {
		fraction -= 2048
	}
	exp := val >> 11
	if exp > 15 {
		exp -= 32
	}
	return float32(fraction) * float32(math.Pow(2, float64(exp)))
}

// GetTime will return current time as string
func GetTime() string {
	t := time.Now()
	hour, minute, _ := t.Clock()
	return itoaTwoDigits(hour) + ":" + itoaTwoDigits(minute)
}

// GetDate will return the current date as string
func GetDate() string {
	return time.Now().Format("02, Jan 2006")
}

// itoaTwoDigits time.Clock returns one digit on values, so we make sure to convert to two digits
func itoaTwoDigits(i int) string {
	b := "0" + strconv.Itoa(i)
	return b[len(b)-2:]
}

// ResizeImage will resize image with given width and height
func ResizeImage(src image.Image, width, height int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return dst
}

// MuteWithPulseAudio will mute / unmute mic via pulse audio
func MuteWithPulseAudio() error {
	cmd := exec.Command("pactl", "set-source-mute", "@DEFAULT_SOURCE@", "toggle")
	return cmd.Run()
}

// MuteWithALSA will mute / unmute mic via alsa
func MuteWithALSA() error {
	// Try muting with ALSA (assuming 'Capture' as the control name)
	cmd := exec.Command("amixer", "set", "Capture", "toggle")
	return cmd.Run()
}

// PidVidToUint16 will convert string based productId or vendorId to uint16 value
func PidVidToUint16(value string) uint16 {
	val, err := strconv.ParseUint(value, 16, 16)
	if err != nil {
		return 0
	}
	return uint16(val)
}

// GetBcdDevice will return device bcdDevice value
func GetBcdDevice(path string) (string, error) {
	base := filepath.Base(path)
	sysClassPath := filepath.Join("/sys/class/hidraw", base, "device")
	resolvedPath, err := filepath.EvalSymlinks(sysClassPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve symlink: %v", err)
	}
	usbDevicePath := filepath.Join(resolvedPath, "../..")
	usbDevicePath = filepath.Clean(usbDevicePath)
	bcdDevicePath := filepath.Join(usbDevicePath, "bcdDevice")
	data, err := os.ReadFile(bcdDevicePath)
	if err != nil {
		return "", fmt.Errorf("failed to read bcdDevice: %v", err)
	}
	bcdStr := strings.TrimSpace(string(data))
	if len(bcdStr) != 4 {
		return "", fmt.Errorf("unexpected bcdDevice length: %s", bcdStr)
	}
	major := bcdStr[:2]
	minor := bcdStr[2:]
	return fmt.Sprintf("%s.%s", major, minor), nil
}
