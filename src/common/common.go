package common

import (
	"math"
	"os"
	"strconv"
	"unsafe"
)

// FileExists will check if given filename exists
func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return err == nil
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

func InBetween(i, min, max float32) bool {
	if (i >= min) && (i <= max) {
		return true
	} else {
		return false
	}
}

// FractionOfByte will return fraction of given value
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
