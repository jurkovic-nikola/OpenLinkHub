package memory

// ENE SMBus memory RGB support is based on the same register protocol used by
// OpenRGB's ENESMBusController for DRAM controllers.

import (
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/dashboard"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/smbus"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

const (
	memoryProtocolCorsair = "corsair"
	memoryProtocolENE     = "ene"

	eneCommandSelectRegister = 0x00
	eneCommandWriteValue     = 0x01
	eneCommandWriteBlock     = 0x03
	eneCommandReadValue      = 0x81

	eneApplyValue = 0x01
	eneMaxBlock   = 3

	eneRegDeviceName      = 0x1000
	eneRegMicronCheck     = 0x1030
	eneRegConfigTable     = 0x1c00
	eneRegColorsDirect    = 0x8000
	eneRegColorsEffect    = 0x8010
	eneRegDirect          = 0x8020
	eneRegApply           = 0x80a0
	eneRegSlotIndex       = 0x80f8
	eneRegI2CAddress      = 0x80f9
	eneRegColorsDirectV2  = 0x8100
	eneRegColorsEffectV2  = 0x8160
	eneConfigLedCount     = 0x02
	eneDefaultLedChannels = 8
	eneMaxLedChannels     = 32
)

var eneRamAddresses = []byte{
	0x70,
	0x71,
	0x72,
	0x73,
	0x74,
	0x75,
	0x76,
	0x77,
	0x4f,
	0x66,
	0x67,
	0x39,
	0x3a,
	0x3b,
	0x3c,
	0x3d,
}

func (d *Device) getENEDevices(modules []RAMModule) map[int]*Devices {
	devices := make(map[int]*Devices)

	d.remapENEDevices()

	channelId := 0
	for _, address := range eneRamAddresses {
		if !d.testENEController(address) {
			continue
		}

		version := d.readENEString(address, eneRegDeviceName, 16)
		configTable := d.readENEConfigTable(address)
		ledChannels := eneLedCount(configTable)
		directRegister, effectRegister := eneRegistersForVersion(version, eneConfiguredLedCount(configTable))

		sku := ""
		var module *RAMModule
		if channelId < len(modules) {
			module = &modules[channelId]
			sku = strings.TrimSpace(module.SKU)
		}
		if sku == "" {
			sku = strings.TrimSpace(config.GetConfig().MemorySku)
		}

		device := &Devices{
			ChannelId:      channelId,
			DeviceId:       channelId,
			Sku:            sku,
			MemoryType:     d.RuntimeMemoryType,
			LedChannels:    uint8(ledChannels),
			Protocol:       memoryProtocolENE,
			Address:        address,
			Name:           eneDeviceName(sku, version),
			Label:          d.memoryLabel(channelId),
			RGB:            d.memoryRGBProfile(channelId),
			DirectRegister: directRegister,
			EffectRegister: effectRegister,
		}
		d.setENETemperature(device, module)

		if d.Debug {
			logger.Log(logger.Fields{"memoryDevice": device, "version": version, "configTable": fmt.Sprintf("% 2x", configTable)}).Info("ENE Memory DIMM Info - Device")
		}
		if d.SkuLine == "" {
			d.SkuLine = device.Name
		}
		devices[channelId] = device
		d.LEDChannels += ledChannels
		channelId++
		time.Sleep(1 * time.Millisecond)
	}

	return devices
}

func (d *Device) remapENEDevices() {
	if !d.eneAddressResponds(0x77) {
		return
	}

	addressListIndex := -1
	for slot := 0; slot < maximumRegisters; slot++ {
		if !d.eneAddressResponds(0x77) {
			return
		}

		for {
			addressListIndex++
			if addressListIndex >= len(eneRamAddresses) {
				return
			}
			if !d.eneAddressResponds(eneRamAddresses[addressListIndex]) {
				break
			}
		}

		address := eneRamAddresses[addressListIndex]
		if d.Debug {
			logger.Log(logger.Fields{"slot": slot, "address": address}).Info("Remapping ENE memory slot")
		}
		if err := d.eneWriteRegister(0x77, eneRegSlotIndex, byte(slot)); err != nil {
			logger.Log(logger.Fields{"slot": slot, "address": address, "error": err}).Warn("Unable to select ENE memory slot for remap")
			return
		}
		if err := d.eneWriteRegister(0x77, eneRegI2CAddress, address<<1); err != nil {
			logger.Log(logger.Fields{"slot": slot, "address": address, "error": err}).Warn("Unable to remap ENE memory slot")
			return
		}
		time.Sleep(1 * time.Millisecond)
	}
}

func (d *Device) eneAddressResponds(address byte) bool {
	if _, err := smbus.ReadByte(d.dev.File, address); err == nil {
		return true
	}
	if _, err := smbus.ReadRegister(d.dev.File, address, 0x00); err == nil {
		return true
	}
	return false
}

func (d *Device) testENEController(address byte) bool {
	if !d.eneAddressResponds(address) {
		return false
	}

	for register := byte(0xa0); register < 0xb0; register++ {
		value, err := smbus.ReadRegister(d.dev.File, address, register)
		if err != nil {
			return false
		}
		if value != register-0xa0 {
			return false
		}
	}

	value := d.readENERawString(address, eneRegMicronCheck, 16)
	if strings.TrimRight(string(value), "\x00") == "Micron" {
		return false
	}
	return true
}

func (d *Device) readENEConfigTable(address byte) []byte {
	configTable := make([]byte, 64)
	for i := range configTable {
		value, err := d.eneReadRegister(address, eneRegConfigTable+uint16(i))
		if err != nil {
			return configTable
		}
		configTable[i] = value
	}
	return configTable
}

func eneLedCount(configTable []byte) int {
	ledChannels := eneConfiguredLedCount(configTable)
	if ledChannels <= 0 {
		ledChannels = eneDefaultLedChannels
	}
	if ledChannels > eneMaxLedChannels {
		ledChannels = eneMaxLedChannels
	}
	return ledChannels
}

func eneConfiguredLedCount(configTable []byte) int {
	if len(configTable) > eneConfigLedCount {
		return int(configTable[eneConfigLedCount])
	}
	return 0
}

func eneRegistersForVersion(version string, configuredLedChannels int) (uint16, uint16) {
	if strings.HasPrefix(version, "AUDA0-E6K5") || configuredLedChannels > 5 {
		return eneRegColorsDirectV2, eneRegColorsEffectV2
	}
	return eneRegColorsDirect, eneRegColorsEffect
}

func eneDeviceName(sku, version string) string {
	if strings.HasPrefix(strings.ToUpper(sku), "F5") {
		return "G.SKILL Trident Z5 RGB"
	}
	if strings.HasPrefix(strings.ToUpper(sku), "F4") {
		return "G.SKILL Trident Z RGB"
	}
	if version != "" {
		return "ENE DRAM RGB"
	}
	return "G.SKILL / ENE DRAM RGB"
}

func (d *Device) memoryLabel(channelId int) string {
	label := "Set Label"
	if d.DeviceProfile == nil {
		logger.Log(logger.Fields{"serial": d.Serial}).Warn("DeviceProfile is not set, probably first startup")
		return label
	}
	if lb, ok := d.DeviceProfile.Labels[channelId]; ok && len(lb) > 0 {
		label = lb
	}
	return label
}

func (d *Device) memoryRGBProfile(channelId int) string {
	rgbProfile := "static"
	if d.DeviceProfile == nil {
		logger.Log(logger.Fields{"serial": d.Serial}).Warn("DeviceProfile is not set, probably first startup")
		return rgbProfile
	}
	if rp, ok := d.DeviceProfile.RGBProfiles[channelId]; ok {
		if d.GetRgbProfile(rp) != nil {
			return rp
		}
		logger.Log(logger.Fields{"serial": d.Serial, "profile": rp}).Warn("Tried to apply non-existing rgb profile")
	}
	return rgbProfile
}

func (d *Device) setENETemperature(device *Devices, module *RAMModule) {
	if !config.GetConfig().RamTempViaHwmon || module == nil || module.EEPROMPath == "" {
		return
	}

	hwmonPath := filepath.Join(filepath.Dir(filepath.Dir(module.EEPROMPath)), "temp1_input")
	hwmonTemp, err := d.getTemperature(hwmonPath)
	if err != nil {
		return
	}

	device.HwmonPath = hwmonPath
	device.Temperature = hwmonTemp
	device.TemperatureString = dashboard.GetDashboard().TemperatureToString(hwmonTemp)
	device.HasTemps = true
}

func (d *Device) readENEString(address byte, register uint16, size int) string {
	return strings.TrimRight(string(d.readENERawString(address, register, size)), "\x00")
}

func (d *Device) readENERawString(address byte, register uint16, size int) []byte {
	buffer := make([]byte, 0, size)
	for i := 0; i < size; i++ {
		value, err := d.eneReadRegister(address, register+uint16(i))
		if err != nil {
			return buffer
		}
		buffer = append(buffer, value)
	}
	return buffer
}

func (d *Device) eneReadRegister(address byte, register uint16) (byte, error) {
	if err := smbus.WriteWord(d.dev.File, address, eneCommandSelectRegister, eneSwapRegister(register)); err != nil {
		return 0, err
	}
	return smbus.ReadRegister(d.dev.File, address, eneCommandReadValue)
}

func (d *Device) eneWriteRegister(address byte, register uint16, value byte) error {
	if err := smbus.WriteWord(d.dev.File, address, eneCommandSelectRegister, eneSwapRegister(register)); err != nil {
		return err
	}
	return smbus.WriteRegister(d.dev.File, address, eneCommandWriteValue, value)
}

func (d *Device) eneWriteBlock(address byte, register uint16, data []byte) error {
	if len(data) == 0 {
		return nil
	}

	for offset := 0; offset < len(data); offset += eneMaxBlock {
		end := offset + eneMaxBlock
		if end > len(data) {
			end = len(data)
		}
		chunk := data[offset:end]
		if err := smbus.WriteWord(d.dev.File, address, eneCommandSelectRegister, eneSwapRegister(register+uint16(offset))); err != nil {
			return err
		}
		if err := smbus.WriteBlockDataUncached(d.dev.File, address, eneCommandWriteBlock, chunk); err != nil {
			for _, value := range chunk {
				if writeErr := smbus.WriteRegister(d.dev.File, address, eneCommandWriteValue, value); writeErr != nil {
					return writeErr
				}
			}
		}
	}
	return nil
}

func (d *Device) transferENE(buffer []byte, device *Devices) uint16 {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if device == nil || device.Address == 0 {
		return 0
	}

	if err := d.eneWriteRegister(device.Address, eneRegDirect, 1); err != nil {
		logger.Log(logger.Fields{"error": err, "address": device.Address}).Error("Unable to set ENE direct mode")
		return 0
	}

	colorBuffer := make([]byte, int(device.LedChannels)*3)
	for led := 0; led < int(device.LedChannels); led++ {
		src := led * 3
		dst := led * 3
		if src+2 >= len(buffer) {
			break
		}
		colorBuffer[dst] = buffer[src]
		colorBuffer[dst+1] = buffer[src+2]
		colorBuffer[dst+2] = buffer[src+1]
	}

	if d.Debug {
		logger.Log(logger.Fields{"colorPacket": fmt.Sprintf("% 2x", colorBuffer), "address": device.Address}).Info("ENE Memory Color")
	}
	if err := d.eneWriteBlock(device.Address, device.DirectRegister, colorBuffer); err != nil {
		logger.Log(logger.Fields{"error": err, "address": device.Address}).Error("Unable to write ENE memory color")
		return 0
	}
	if err := d.eneWriteRegister(device.Address, eneRegApply, eneApplyValue); err != nil {
		logger.Log(logger.Fields{"error": err, "address": device.Address}).Error("Unable to apply ENE memory color")
	}
	return 0
}

func eneSwapRegister(register uint16) uint16 {
	return ((register << 8) & 0xff00) | ((register >> 8) & 0x00ff)
}
