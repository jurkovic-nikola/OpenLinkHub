package opcodes

var (
	CmdOpenEndpoint         = []byte{0x0d, 0x01}
	CmdOpenColorEndpoint    = []byte{0x0d, 0x00}
	CmdCloseEndpoint        = []byte{0x05, 0x01, 0x01}
	CmdGetFirmware          = []byte{0x02, 0x13}
	CmdSoftwareMode         = []byte{0x01, 0x03, 0x00, 0x02}
	CmdHardwareMode         = []byte{0x01, 0x03, 0x00, 0x01}
	CmdWrite                = []byte{0x06, 0x01}
	CmdWriteColor           = []byte{0x06, 0x00}
	CmdRead                 = []byte{0x08, 0x01}
	CmdGetDeviceMode        = []byte{0x01, 0x08, 0x01}
	ModeGetDevices          = []byte{0x36}
	ModeGetTemperatures     = []byte{0x21}
	ModeGetSpeeds           = []byte{0x17}
	ModeSetSpeed            = []byte{0x18}
	ModeSetColor            = []byte{0x22}
	DataTypeGetDevices      = []byte{0x21, 0x00}
	DataTypeGetTemperatures = []byte{0x10, 0x00}
	DataTypeGetSpeeds       = []byte{0x25, 0x00}
	DataTypeSetSpeed        = []byte{0x07, 0x00}
	DataTypeSetColor        = []byte{0x12, 0x00}
)
