package common

import (
	"OpenICUELinkHub/src/config"
	"OpenICUELinkHub/src/structs"
	"sort"
	"strconv"
	"unsafe"
)

var (
	MinimumPumpPercent = 50
	PercentMin         = 0
	PercentMax         = 100
	devices            = []structs.DeviceList{
		{0x01, 0x00, "QX Fan", 34},        // Fan
		{0x13, 0x00, "RX Fan", 0},         // Fan No LEDs
		{0x0f, 0x00, "RX RGB Fan", 8},     // Fan
		{0x07, 0x02, "H150i", 20},         // AIO Black
		{0x07, 0x05, "H150i", 20},         // AIO White
		{0x07, 0x01, "H115i", 20},         // AIO
		{0x07, 0x03, "H170i", 20},         // AIO
		{0x07, 0x00, "H100i", 20},         // AIO Black
		{0x07, 0x04, "H100i", 20},         // AIO White
		{0x09, 0x00, "XC7 Elite", 24},     // CPU Block Stealth Gray
		{0x09, 0x01, "XC7 Elite", 24},     // CPU Block White
		{0x0d, 0x00, "XG7", 16},           // GPU Block
		{0x0c, 0x00, "XD5 Elite", 22},     // Pump reservoir Stealth Gray
		{0x0c, 0x01, "XD5 Elite", 22},     // Pump reservoir White (?)
		{0x0e, 0x00, "XD5 Elite LCD", 22}, // Pump reservoir Stealth Gray
		{0x0e, 0x01, "XD5 Elite LCD", 22}, // Pump reservoir White (?)
	}
)

func ContainsPump(t byte) bool {
	return t == 0x07 || t == 0x0c || t == 0x0e
}

func GetDevice(deviceId byte, deviceModel byte) *structs.DeviceList {
	for _, device := range devices {
		if device.DeviceId == deviceId && device.Model == deviceModel {
			return &device
		}
	}
	return nil
}

// Clamp will check if value is in specified range
func Clamp(value, min, max int) int {
	if value < min {
		return min
	} else if value > max {
		return max
	}

	return value
}

// SetDefaultChannelData will setup default channel speed
func SetDefaultChannelData(deviceType byte) byte {
	value := config.GetConfig().DefaultFanValue
	if ContainsPump(deviceType) {
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
