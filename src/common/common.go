package common

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"image"
	"io"
	"math"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/image/draw"
	"image/gif"
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

type KeyboardPerformance struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Value    bool   `json:"value"`
	Internal string `json:"internal"`
}

type KeyboardPerformanceData struct {
	WinKey   bool
	ShiftTab bool
	AltTab   bool
	AltF4    bool
}

const (
	ProductTypeLinkHub              = 0
	ProductTypeCC                   = 1
	ProductTypeCCXT                 = 2
	ProductTypeElite                = 3
	ProductTypeLNCore               = 4
	ProductTypeLnPro                = 5
	ProductTypeCPro                 = 6
	ProductTypeXC7                  = 7
	ProductTypeMemory               = 8
	ProductTypeNexus                = 9
	ProductTypePlatinum             = 10
	ProductTypeHydro                = 11
	ProductTypeK65PM                = 101
	ProductTypeK70Core              = 102
	ProductTypeK55Core              = 103
	ProductTypeK70Pro               = 104
	ProductTypeK65Plus              = 105
	ProductTypeK65PlusW             = 106
	ProductTypeK100AirWU            = 107
	ProductTypeK100AirW             = 108
	ProductTypeK100                 = 109
	ProductTypeK70MK2               = 110
	ProductTypeK70CoreTkl           = 111
	ProductTypeK70CoreTklWU         = 112
	ProductTypeK70CoreTklW          = 113
	ProductTypeK70ProTkl            = 114
	ProductTypeK70RgbTkl            = 115
	ProductTypeK55Pro               = 116
	ProductTypeK55ProXT             = 117
	ProductTypeK55                  = 118
	ProductTypeK95Platinum          = 119
	ProductTypeK60RgbPro            = 120
	ProductTypeK70PMW               = 121
	ProductTypeK70PMWU              = 122
	ProductTypeK70Max               = 123
	ProductTypeMakr75WU             = 124
	ProductTypeMakr75W              = 125
	ProductTypeK95PlatinumXT        = 126
	ProductTypeK70LUX               = 127
	ProductTypeK65Rgb               = 128
	ProductTypeKatarPro             = 201
	ProductTypeIronClawRgb          = 202
	ProductTypeIronClawRgbW         = 203
	ProductTypeIronClawRgbWU        = 204
	ProductTypeNightsabreW          = 205
	ProductTypeNightsabreWU         = 206
	ProductTypeScimitarRgbElite     = 207
	ProductTypeScimitarRgbEliteW    = 208
	ProductTypeScimitarRgbEliteWU   = 209
	ProductTypeM55                  = 210
	ProductTypeM55W                 = 211
	ProductTypeM55RgbPro            = 212
	ProductTypeKatarProW            = 213
	ProductTypeDarkCoreRgbProSEW    = 214
	ProductTypeDarkCoreRgbProSEWU   = 215
	ProductTypeDarkCoreRgbProW      = 216
	ProductTypeDarkCoreRgbProWU     = 217
	ProductTypeM75                  = 218
	ProductTypeM75AirW              = 219
	ProductTypeM75AirWU             = 220
	ProductTypeM75W                 = 221
	ProductTypeM75WU                = 222
	ProductTypeM65RgbUltra          = 223
	ProductTypeHarpoonRgbPro        = 224
	ProductTypeHarpoonRgbW          = 225
	ProductTypeHarpoonRgbWU         = 226
	ProductTypeKatarProXT           = 227
	ProductTypeDarkstarWU           = 228
	ProductTypeDarkstarW            = 229
	ProductTypeScimitarRgbEliteSEW  = 230
	ProductTypeScimitarRgbEliteSEWU = 231
	ProductTypeM65RgbUltraW         = 232
	ProductTypeM65RgbUltraWU        = 233
	ProductTypeSabreRgbProWU        = 234
	ProductTypeSabreRgbProW         = 235
	ProductTypeNightswordRgb        = 236
	ProductTypeSabreRgbPro          = 237
	ProductTypeScimitarProRgb       = 238
	ProductTypeScimitarRgb          = 239
	ProductTypeDarkCoreRgbSEWU      = 240
	ProductTypeDarkCoreRgbSEW       = 241
	ProductTypeSabreProCs           = 242
	ProductTypeM65RgbElite          = 243
	ProductTypeVirtuosoXTW          = 300
	ProductTypeVirtuosoXTWU         = 301
	ProductTypeVirtuosoMAXW         = 302
	ProductTypeHS80RGBW             = 303
	ProductTypeHS80MAXW             = 304
	ProductTypeHS80RGB              = 305
	ProductTypeVirtuosoSEWU         = 306
	ProductTypeVirtuosoSEW          = 307
	ProductTypeST100                = 401
	ProductTypeMM700                = 402
	ProductTypeLT100                = 403
	ProductTypeMM800                = 404
	ProductTypePSUHid               = 501
	ProductTypeDongle               = 997
	ProductTypeSlipstream           = 998
	ProductTypeCluster              = 999
)

const (
	DeviceTypeMotherboard = uint32(iota)
	DeviceTypeDram
	DeviceTypeGpu
	DeviceTypeCooler
	DeviceTypeLedstrip
	DeviceTypeKeyboard
	DeviceTypeMouse
	DeviceTypeMousemat
	DeviceTypeHeadset
	DeviceTypeHeadsetStand
	DeviceTypeGamepad
	DeviceTypeLight
	DeviceTypeSpeaker
	DeviceTypeVirtual
	DeviceTypeStorage
	DeviceTypeCase
	DeviceTypeMicrophone
	DeviceTypeAccessory
	DeviceTypeKeypad
	DeviceTypeLaptop
	DeviceTypeMonitor
	DeviceTypeUnknown
)

const (
	ZoneTypeSingle = uint32(iota)
	ZoneTypeLinear
	ZoneTypeMatrix
)

const (
	ColorModeNone = uint32(iota)
	ColorModePerLed
	ColorModeSpecific
	ColorModeRandom
)

type OpenRGBSegment struct {
	Name     string
	Type     int32
	StartIdx uint32
	LedCount uint32
}

type OpenRGBZone struct {
	Name     string
	NumLEDs  uint32
	MinLeds  uint32
	ZoneType uint32
	Segments []OpenRGBSegment
}

type OpenRGBController struct {
	Name         string
	Vendor       string
	Description  string
	FwVersion    string
	Serial       string
	Location     string
	Zones        []OpenRGBZone
	Colors       []byte
	ActiveMode   int32
	WriteColorEx func([]byte, int)
	ChannelId    int
	DeviceType   uint32
	ColorMode    uint32
}

type ClusterController struct {
	Product      string
	Serial       string
	LedChannels  uint32
	ChannelId    int
	WriteColorEx func([]byte, int)
}

type LogLevel int

const (
	LogInfo LogLevel = iota
	LogWarn
	LogError
	LogFatal
	LogSilent
)

const NA = 0xFFFFFFFF

var MatrixMaps = map[uint32][][]uint32{
	29: { // iCUE Commander Core AIOs
		{28, NA, 27, NA, 26, NA, 25},
		{NA, 16, NA, 15, NA, 14, NA},
		{17, NA, 0, 5, 3, NA, 24},
		{NA, 9, 4, 8, 6, 13, NA},
		{18, NA, 1, 7, 2, NA, 23},
		{NA, 10, NA, 11, NA, 12, NA},
		{19, NA, 20, NA, 21, NA, 22},
	},
	24: { // iCUE Commander Core AIOs
		{NA, NA, NA, NA, NA, 6, NA, NA, NA, NA, NA},
		{NA, NA, NA, 4, 5, NA, 7, 8, NA, NA, NA},
		{NA, NA, 3, NA, NA, NA, NA, NA, 9, NA, NA},
		{NA, 2, NA, NA, NA, NA, NA, NA, NA, 10, NA},
		{NA, 1, NA, NA, NA, NA, NA, NA, NA, 11, NA},
		{0, NA, NA, NA, NA, NA, NA, NA, NA, NA, 12},
		{NA, 23, NA, NA, NA, NA, NA, NA, NA, 13, NA},
		{NA, 22, NA, NA, NA, NA, NA, NA, NA, 14, NA},
		{NA, NA, 21, NA, NA, NA, NA, NA, 15, NA, NA},
		{NA, NA, NA, 20, 19, NA, 17, 16, NA, NA, NA},
		{NA, NA, NA, NA, NA, 18, NA, NA, NA, NA, NA},
	},
	16: { // ELITE, PLATINUM AIOs
		{NA, 11, 12, 13, NA},
		{10, NA, 1, NA, 14},
		{9, 0, NA, 2, 15},
		{8, NA, 3, NA, 4},
		{NA, 7, 6, 5, NA},
	},
	20: { // LINK AIOs
		{NA, NA, 10, 11, 12, NA, NA},
		{NA, 9, NA, NA, NA, 13, NA},
		{8, NA, NA, 16, NA, NA, 14},
		{7, NA, 19, NA, 17, NA, 15},
		{6, NA, NA, 18, NA, NA, 0},
		{NA, 5, NA, NA, NA, 1, NA},
		{NA, NA, 4, 3, 2, NA, NA},
	},
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

// ResizeGifImage will resize image with given width and height
func ResizeGifImage(g *gif.GIF, width, height int) []*image.RGBA {
	imageBuffer := make([]*image.RGBA, len(g.Image))
	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	for i, frame := range g.Image {
		resized := ResizeImage(frame, width, height)
		draw.Draw(canvas, canvas.Bounds().Add(resized.Bounds().Min), resized, resized.Bounds().Min, draw.Over)

		canvasCopy := image.NewRGBA(canvas.Bounds())
		copy(canvasCopy.Pix, canvas.Pix)
		imageBuffer[i] = canvasCopy
	}
	return imageBuffer
}

// MuteWithPulseAudio will mute / unmute mic via pulse audio
func MuteWithPulseAudio() error {
	cmd := exec.Command("pactl", "set-source-mute", "@DEFAULT_SOURCE@", "toggle")
	return cmd.Run()
}

// MuteWithPulseAudioEx will mute / unmute mic via pulse audio and return status
func MuteWithPulseAudioEx() (bool, error) {
	// Toggle mute state
	cmd := exec.Command("pactl", "set-source-mute", "@DEFAULT_SOURCE@", "toggle")
	if err := cmd.Run(); err != nil {
		return false, err
	}

	// Query new mute state
	out, err := exec.Command("pactl", "get-source-mute", "@DEFAULT_SOURCE@").Output()
	if err != nil {
		return false, err
	}

	// Parse output
	s := strings.TrimSpace(string(out))
	if strings.HasSuffix(s, "yes") {
		return true, nil // muted
	}
	return false, nil // unmuted
}

// GetPulseAudioMuteStatus will get mute status
func GetPulseAudioMuteStatus() (bool, error) {
	// Query new mute state
	out, err := exec.Command("pactl", "get-source-mute", "@DEFAULT_SOURCE@").Output()
	if err != nil {
		return false, err
	}

	// Parse output
	s := strings.TrimSpace(string(out))
	if strings.HasSuffix(s, "yes") {
		return true, nil // muted
	}
	return false, nil // unmuted
}

// MuteWithALSA will mute / unmute mic via alsa
func MuteWithALSA() error {
	// Try muting with ALSA (assuming 'Capture' as the control name)
	cmd := exec.Command("amixer", "set", "Capture", "toggle")
	return cmd.Run()
}

// MuteWithALSAEx will mute / unmute mic via alsa and return status
func MuteWithALSAEx() (bool, error) {
	// Toggle mute state
	cmd := exec.Command("amixer", "set", "Capture", "toggle")
	if err := cmd.Run(); err != nil {
		return false, err
	}

	// Query new state
	out, err := exec.Command("amixer", "get", "Capture").Output()
	if err != nil {
		return false, err
	}

	s := string(out)

	// Look for [on]/[off] in the output
	if strings.Contains(s, "[off]") {
		return true, nil // muted
	}
	if strings.Contains(s, "[on]") {
		return false, nil // unmuted
	}

	return false, fmt.Errorf("could not determine mute state from output: %s", s)
}

// GetAlsaMuteStatus will get mute status
func GetAlsaMuteStatus() (bool, error) {
	// Query new state
	out, err := exec.Command("amixer", "get", "Capture").Output()
	if err != nil {
		return false, err
	}

	s := string(out)

	// Look for [on]/[off] in the output
	if strings.Contains(s, "[off]") {
		return true, nil // muted
	}
	if strings.Contains(s, "[on]") {
		return false, nil // unmuted
	}
	return false, fmt.Errorf("could not determine mute state from output: %s", s)
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

// GetBcdDeviceHex returns the firmware version from bcdDevice as "1.18"
func GetBcdDeviceHex(path string) (string, error) {
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

	majorHex := bcdStr[:2]
	minorHex := bcdStr[2:]

	major, err := strconv.ParseInt(majorHex, 16, 0)
	if err != nil {
		return "", fmt.Errorf("failed to parse major version: %v", err)
	}

	minor, err := strconv.ParseInt(minorHex, 16, 0)
	if err != nil {
		return "", fmt.Errorf("failed to parse minor version: %v", err)
	}

	return fmt.Sprintf("%d.%02d", major, minor), nil
}

// generateRandomString generates a secure random string of the given length
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			continue
		}
		result[i] = charset[n.Int64()]
	}
	return string(result)
}

// GenerateRandomMD5 generates a random MD5 string using secure random bytes
func GenerateRandomMD5() string {
	randomBytes := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, randomBytes); err != nil {
		return generateRandomString(32) // fall back to normal string
	}
	hash := md5.Sum(randomBytes)
	return hex.EncodeToString(hash[:])
}
