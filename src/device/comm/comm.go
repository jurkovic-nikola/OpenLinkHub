package comm

import (
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/device/common"
	"OpenLinkHub/src/device/opcodes"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/structs"
	"bytes"
	"context"
	"encoding/binary"
	"github.com/sstallion/go-hid"
	"sync"
	"time"
)

var (
	dev                     *hid.Device
	mutex                   sync.Mutex
	BufferSize              = 512
	HeaderSize              = 3
	HeaderWriteSize         = 4
	BufferSizeWrite         = BufferSize + 1
	TransferTimeout         = 500
	MaxBufferSizePerRequest = 508
)

// Close will attempt to close a USB HID device and terminate in case of failure.
func Close() {
	if dev != nil {
		err := dev.Close()
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Fatal("Unable to close HID device")
		}

		if err = hid.Exit(); err != nil {
			logger.Log(logger.Fields{"error": err}).Fatal("Unable to exit HID interface")
		}
	}
}

// Open will attempt to open a new USB HID device and terminate in case of failure.
func Open() {
	if err := hid.Init(); err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to initialize HID interface")
	}

	vendorId, err := common.ConvertHexToUint16(config.GetConfig().VendorId)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to parse vendorId")
	}

	productId, err := common.ConvertHexToUint16(config.GetConfig().ProductId)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to parse productId")
	}

	dev, err = hid.Open(
		vendorId,
		productId,
		config.GetConfig().Serial,
	)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": "", "deviceId": ""}).Fatal("Unable to open HID device")
	}
}

// GetManufacturer will return device manufacturer
func GetManufacturer() string {
	manufacturer, err := dev.GetMfrStr()
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": "", "deviceId": ""}).Fatal("Unable to get manufacturer")
	}
	return manufacturer
}

// GetProduct will return device name
func GetProduct() string {
	product, err := dev.GetProductStr()
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": "", "deviceId": ""}).Fatal("Unable to get product")
	}
	return product
}

// GetSerial will return device serial number
func GetSerial() string {
	serial, err := dev.GetSerialNbr()
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": "", "deviceId": ""}).Fatal("Unable to get device serial number")
	}
	return serial
}

// Read will read data from a device and return data as structs.DeviceResponse
func Read(endpoint, bufferType []byte) structs.DeviceData {
	// Endpoint data
	var buffer []byte

	// Close specified endpoint
	_, err := Transfer(opcodes.CmdCloseEndpoint, endpoint, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to close endpoint")
	}

	// Open endpoint
	_, err = Transfer(opcodes.CmdOpenEndpoint, endpoint, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to open endpoint")
	}

	// Read data from endpoint
	buffer, err = Transfer(opcodes.CmdRead, endpoint, bufferType)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to read endpoint")
	}

	// Close specified endpoint
	_, err = Transfer(opcodes.CmdCloseEndpoint, endpoint, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to close endpoint")
	}

	return structs.DeviceData{
		Data: buffer,
	}
}

// WriteColor will write data to the device with a specific endpoint.
// WriteColor does not require endpoint closing and opening like normal Write requires.
// Endpoint is open only once. Once the endpoint is open, color can be sent continuously.
func WriteColor(bufferType, data []byte) int {
	// Buffer
	buffer := make([]byte, len(bufferType)+len(data)+HeaderWriteSize)
	binary.LittleEndian.PutUint16(buffer[0:2], uint16(len(data)+2))
	copy(buffer[HeaderWriteSize:HeaderWriteSize+len(bufferType)], bufferType)
	copy(buffer[HeaderWriteSize+len(bufferType):], data)

	// Process buffer and create a chunked array if needed
	writeColorEp := opcodes.CmdWriteColor
	colorEp := make([]byte, len(writeColorEp))
	copy(colorEp, writeColorEp)

	chunks := common.ProcessMultiChunkPacket(buffer, MaxBufferSizePerRequest)
	for i, chunk := range chunks {
		// Next color endpoint based on number of chunks
		colorEp[0] = colorEp[0] + byte(i)
		// Send it
		_, err := Transfer(colorEp, chunk, nil)
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Fatal("Unable to write to endpoint")
			return 0
		}
	}
	// OK
	return 1
}

// Write will write data to the device with specific endpoint
func Write(endpoint, bufferType, data []byte) int {
	// Buffer
	buffer := make([]byte, len(bufferType)+len(data)+HeaderWriteSize)
	binary.LittleEndian.PutUint16(buffer[0:2], uint16(len(data)+2))
	copy(buffer[HeaderWriteSize:HeaderWriteSize+len(bufferType)], bufferType)
	copy(buffer[HeaderWriteSize+len(bufferType):], data)

	// Close endpoint
	_, err := Transfer(opcodes.CmdCloseEndpoint, endpoint, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to close endpoint")
		return 0
	}

	// Open endpoint
	_, err = Transfer(opcodes.CmdOpenEndpoint, endpoint, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to open endpoint")
		return 0
	}

	// Send it
	_, err = Transfer(opcodes.CmdWrite, buffer, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to write to endpoint")
		return 0
	}

	// Close endpoint
	_, err = Transfer(opcodes.CmdCloseEndpoint, endpoint, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to close endpoint")
		return 0
	}

	// OK
	return 1
}

// Transfer will send data to a device and retrieve device output
func Transfer(endpoint, buffer, bufferType []byte) ([]byte, error) {
	mutex.Lock()
	defer mutex.Unlock()

	// Create write buffer
	bufferW := make([]byte, BufferSizeWrite)
	bufferW[2] = 0x01
	endpointHeaderPosition := bufferW[HeaderSize : HeaderSize+len(endpoint)]
	copy(endpointHeaderPosition, endpoint)
	if len(buffer) > 0 {
		copy(bufferW[HeaderSize+len(endpoint):HeaderSize+len(endpoint)+len(buffer)], buffer)
	}

	// Create read buffer
	bufferR := make([]byte, BufferSize)

	// Send command to a device
	if _, err := dev.Write(bufferW); err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to write to a device")
	}

	// Get data from a device
	if _, err := dev.Read(bufferR); err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to read data from device")
	}

	// Read remaining data from a device
	if len(bufferType) == 2 {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(TransferTimeout)*time.Millisecond)
		defer cancel()

		for ctx.Err() != nil && !ResponseMatch(bufferR, bufferType) {
			if _, err := dev.Read(bufferR); err != nil {
				logger.Log(logger.Fields{"error": err}).Error("Unable to read data from device")
			}
		}
		if ctx.Err() != nil {
			logger.Log(logger.Fields{"error": ctx.Err()}).Error("Unable to read data from device")
		}
	}

	return bufferR, nil
}

// ResponseMatch will check if two byte arrays match
func ResponseMatch(response, expected []byte) bool {
	responseBuffer := response[5:7]
	return bytes.Equal(responseBuffer, expected)
}
