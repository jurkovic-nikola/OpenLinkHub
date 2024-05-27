package common

import (
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/device/brightness"
	"OpenLinkHub/src/structs"
	"math/rand"
	"sort"
	"strconv"
	"unsafe"
)

var (
	MinimumPumpPercent = 50
	PercentMin         = 0
	PercentMax         = 100
)

func ContainsPump(t byte) bool {
	return t == 0x07 || t == 0x0c || t == 0x0e
}

func GetDevice(deviceId byte, deviceModel byte) *structs.DeviceList {
	for _, device := range config.GetDevices() {
		if device.DeviceId == deviceId && device.Model == deviceModel {
			return &device
		}
	}
	return nil
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

// FClamp function restricts the value within the specified range [min, max].
func FClamp(value, min, max float64) float64 {
	if value < min {
		return min
	} else if value > max {
		return max
	}
	return value
}

// SetDefaultChannelData will setup default channel speed
func SetDefaultChannelData(device *structs.DeviceList) byte {
	value := config.GetConfig().DefaultFanValue
	if device.ContainsPump {
		value = config.GetConfig().DefaultPumpValue
		if value < MinimumPumpPercent {
			value = MinimumPumpPercent
		}
	}
	return byte(Clamp(value, PercentMin, PercentMax))
}

// SetColor will generate byte output for RGB data
func SetColor(data map[int][]byte) []byte {
	buffer := make([]byte, len(data)*3)
	i := 0

	// We need to sort array due to the nature of RGB.
	// R G B needs to be applied in the same way it was created.
	keys := make([]int, 0)
	for k := range data {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	for _, k := range keys {
		buffer[i] = data[k][0]   // r
		buffer[i+1] = data[k][1] // g
		buffer[i+2] = data[k][2] // b
		i += 3                   // Move to the next place
	}

	return buffer
}

// SetSpeed will generate byte output for speed RPM data
func SetSpeed(data map[int][]byte, mode uint8) []byte {
	buffer := make([]byte, len(data)*4+1)
	buffer[0] = byte(len(data))
	i := 1
	for channel, speed := range data {
		v := 2
		buffer[i] = byte(channel)
		buffer[i+1] = mode // Either percent mode or RPM mode
		for value := range speed {
			buffer[i+v] = speed[value]
			v++
		}
		i += 4 // Move to the next place
	}
	return buffer
}

// IntToByteArray will covert integer to byte array
func IntToByteArray(num uint16) []byte {
	size := 2
	buffer := make([]byte, size)
	for i := 0; i < size; i++ {
		byt := *(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(&num)) + uintptr(i)))
		buffer[i] = byt
	}
	return buffer
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

// ConvertHexToUint16 takes a hexadecimal string and converts it to an uint16.
func ConvertHexToUint16(hexStr string) (uint16, error) {
	value, err := strconv.ParseUint(hexStr, 16, 16)
	if err != nil {
		return 0, err // Return the zero value of uint16 and the error
	}
	return uint16(value), nil
}

// GenerateRandomColor will generate random color with provided bts as brightness
func GenerateRandomColor(bts float64) *structs.Color {
	r := rand.Intn(256) // Random value between 0 and 255
	g := rand.Intn(256) // Random value between 0 and 255
	b := rand.Intn(256) // Random value between 0 and 255

	color := &structs.Color{
		Red:        float64(r),
		Green:      float64(g),
		Blue:       float64(b),
		Brightness: bts,
	}
	return brightness.ModifyBrightness(*color)
}
