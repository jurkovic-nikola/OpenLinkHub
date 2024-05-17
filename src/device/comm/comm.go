package comm

import (
	"OpenICUELinkHub/src/device/common"
	"OpenICUELinkHub/src/device/opcodes"
	"OpenICUELinkHub/src/logger"
	"OpenICUELinkHub/src/structs"
	"bytes"
	"context"
	"encoding/binary"
	"github.com/sstallion/go-hid"
	"sync"
	"time"
)

var (
	mutex                   = sync.Mutex{}
	BufferSize              = 512
	HeaderSize              = 3
	HeaderWriteSize         = 4
	BufferSizeWrite         = BufferSize + 1
	TransferTimeout         = 500
	MaxBufferSizePerRequest = 508
)

const (
	EndpointTypeDefault uint8 = 0
	EndpointTypeColor   uint8 = 1
)

// Read will read data from a device and return data as structs.DeviceResponse
func Read(dev *hid.Device, endpoint, bufferType []byte) structs.DeviceData {
	mutex.Lock()
	defer mutex.Unlock()

	// Endpoint data
	var buffer []byte

	// Close specified endpoint
	_, err := Transfer(dev, opcodes.GetOpcode(opcodes.OpcodeCloseEndpoint), endpoint, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to close endpoint")
	}

	// Open endpoint
	_, err = Transfer(dev, opcodes.GetOpcode(opcodes.OpcodeOpenEndpoint), endpoint, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to open endpoint")
	}

	// Read data from endpoint
	buffer, err = Transfer(dev, opcodes.GetOpcode(opcodes.OpcodeRead), endpoint, bufferType)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to read endpoint")
	}

	// Close specified endpoint
	_, err = Transfer(dev, opcodes.GetOpcode(opcodes.OpcodeCloseEndpoint), endpoint, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to close endpoint")
	}

	return structs.DeviceData{
		Data: buffer,
	}
}

// Write will write data to the device with specific endpoint
func Write(dev *hid.Device, endpoint, bufferType, data []byte, endpointType uint8) int {
	mutex.Lock()
	defer mutex.Unlock()

	// Buffer
	buffer := make([]byte, len(bufferType)+len(data)+HeaderWriteSize)
	binary.LittleEndian.PutUint16(buffer[0:2], uint16(len(data)+2))
	copy(buffer[HeaderWriteSize:HeaderWriteSize+len(bufferType)], bufferType)
	copy(buffer[HeaderWriteSize+len(bufferType):], data)

	// Close endpoint
	_, err := Transfer(dev, opcodes.GetOpcode(opcodes.OpcodeCloseEndpoint), endpoint, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to close endpoint")
		return 0
	}

	// What action are we doing?
	switch endpointType {
	case EndpointTypeDefault: // Speed control
		{
			// Open endpoint
			_, err = Transfer(dev, opcodes.GetOpcode(opcodes.OpcodeOpenEndpoint), endpoint, nil)
			if err != nil {
				logger.Log(logger.Fields{"error": err}).Fatal("Unable to open endpoint")
				return 0
			}

			// Send it
			_, err = Transfer(dev, opcodes.GetOpcode(opcodes.OpcodeWrite), buffer, nil)
			if err != nil {
				logger.Log(logger.Fields{"error": err}).Fatal("Unable to write to endpoint")
				return 0
			}
		}
	case EndpointTypeColor: // RGB
		{
			// Open endpoint
			_, err = Transfer(dev, opcodes.GetOpcode(opcodes.OpcodeOpenColorEndpoint), endpoint, nil)
			if err != nil {
				logger.Log(logger.Fields{"error": err}).Fatal("Unable to open endpoint")
				return 0
			}

			// Process buffer and create a chunked array if needed
			writeColorEp := opcodes.GetOpcode(opcodes.OpcodeWriteColor)
			colorEp := make([]byte, len(writeColorEp))
			copy(colorEp, writeColorEp)

			chunks := common.ProcessMultiChunkPacket(buffer, MaxBufferSizePerRequest)
			for i, chunk := range chunks {
				// Next color endpoint based on number of chunks
				colorEp[0] = colorEp[0] + byte(i)
				// Send it
				_, err = Transfer(dev, colorEp, chunk, nil)
				if err != nil {
					logger.Log(logger.Fields{"error": err}).Fatal("Unable to write to endpoint")
					return 0
				}

			}
		}
	}

	// Close endpoint
	_, err = Transfer(dev, opcodes.GetOpcode(opcodes.OpcodeCloseEndpoint), endpoint, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to close endpoint")
		return 0
	}

	// OK
	return 1
}

// Transfer will send data to a device and retrieve device output
func Transfer(dev *hid.Device, endpoint, buffer, bufferType []byte) ([]byte, error) {
	bufferW := make([]byte, BufferSizeWrite)
	bufferW[2] = 0x01
	endpointHeaderPosition := bufferW[HeaderSize : HeaderSize+len(endpoint)]
	copy(endpointHeaderPosition, endpoint)
	if len(buffer) > 0 {
		copy(bufferW[HeaderSize+len(endpoint):HeaderSize+len(endpoint)+len(buffer)], buffer)
	}

	bufferR := make([]byte, BufferSize)

	// Send command to a device
	if _, err := dev.Write(bufferW); err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to write to a device")
	}

	// Get data from a device
	if _, err := dev.Read(bufferR); err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to read data from device")
	}

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
