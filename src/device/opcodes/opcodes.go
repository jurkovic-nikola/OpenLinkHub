package opcodes

const (
	OpcodeOpenEndpoint      uint8 = 0
	OpcodeOpenColorEndpoint uint8 = 1
	OpcodeCloseEndpoint     uint8 = 2
	OpcodeGetFirmware       uint8 = 3
	OpcodeSoftwareMode      uint8 = 4
	OpcodeHardwareMode      uint8 = 5
	OpcodeWrite             uint8 = 6
	OpcodeWriteColor        uint8 = 7
	OpcodeRead              uint8 = 8
	OpcodeGetDevices        uint8 = 9
	OpcodeDevices           uint8 = 10
	OpcodeGetTemperatures   uint8 = 11
	OpcodeTemperatures      uint8 = 12
	OpcodeGetSpeeds         uint8 = 13
	OpcodeSpeeds            uint8 = 14
	OpcodeSetSpeed          uint8 = 15
	OpcodeSpeed             uint8 = 16
	OpcodeSetColor          uint8 = 17
	OpcodeColor             uint8 = 18
	OpcodeGetDeviceMode     uint8 = 19
)

var opcodes = map[uint8][]byte{
	OpcodeOpenEndpoint:      {0x0d, 0x01},
	OpcodeOpenColorEndpoint: {0x0d, 0x00},
	OpcodeCloseEndpoint:     {0x05, 0x01, 0x01},
	OpcodeGetFirmware:       {0x02, 0x13},
	OpcodeSoftwareMode:      {0x01, 0x03, 0x00, 0x02},
	OpcodeHardwareMode:      {0x01, 0x03, 0x00, 0x01},
	OpcodeWrite:             {0x06, 0x01},
	OpcodeWriteColor:        {0x06, 0x00},
	OpcodeRead:              {0x08, 0x01},
	OpcodeGetDevices:        {0x36},
	OpcodeDevices:           {0x21, 0x00},
	OpcodeGetTemperatures:   {0x21},
	OpcodeTemperatures:      {0x10, 0x00},
	OpcodeGetSpeeds:         {0x17},
	OpcodeSpeeds:            {0x25, 0x00},
	OpcodeSetSpeed:          {0x18},
	OpcodeSpeed:             {0x07, 0x00},
	OpcodeSetColor:          {0x22},
	OpcodeColor:             {0x12, 0x00},
	OpcodeGetDeviceMode:     {0x01, 0x08, 0x01},
}

func GetOpcode(opcode uint8) []byte {
	val, ok := opcodes[opcode]
	if ok {
		return val
	}
	return nil
}
