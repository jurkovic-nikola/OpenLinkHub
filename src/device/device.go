package device

import (
	"OpenICUELinkHub/src/config"
	"OpenICUELinkHub/src/device/brightness"
	"OpenICUELinkHub/src/device/comm"
	"OpenICUELinkHub/src/device/common"
	"OpenICUELinkHub/src/device/opcodes"
	"OpenICUELinkHub/src/logger"
	"OpenICUELinkHub/src/structs"
	"encoding/binary"
	"fmt"
	"github.com/sstallion/go-hid"
	"os"
	"sort"
	"sync"
	"time"
)

var (
	device        *structs.Device
	AF            chan bool
	deviceMonitor *structs.DeviceMonitor
	currentColors map[int]*structs.Color
)

// GetDevice will return structs.Device
func GetDevice() *structs.Device {
	return device
}

// Stop will send the device back to hardware mode, usually when the program exits
func Stop() {
	AF <- true
	SetDeviceMode(device.Handle, opcodes.GetOpcode(opcodes.OpcodeHardwareMode))
	err := device.Handle.Close()
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to close HID interface")
	}
	if err := hid.Exit(); err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to exit HID interface")
	}
}

// Init will initialize a device and prepare a device for receiving commands
func Init() {
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

	dev, err := hid.Open(
		vendorId,
		productId,
		config.GetConfig().Serial,
	)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": "", "deviceId": ""}).Fatal("Unable to open HID device")
	}

	manufacturer, err := dev.GetMfrStr()
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": "", "deviceId": ""}).Fatal("Unable to get manufacturer")
	}

	product, err := dev.GetProductStr()
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": "", "deviceId": ""}).Fatal("Unable to get product")
	}

	serial, err := dev.GetSerialNbr()
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": "", "deviceId": ""}).Fatal("Unable to get device serial number")
	}

	device = &structs.Device{
		Handle:       dev,
		Manufacturer: manufacturer,
		Product:      product,
		Serial:       serial,
		Firmware:     GetDeviceFirmware(dev),
		Standalone:   config.GetConfig().Standalone,
	}

	// Activate software mode on device
	SetDeviceMode(dev, opcodes.GetOpcode(opcodes.OpcodeSoftwareMode))

	// Init all channels
	device.Devices = InitDevices(dev)

	// Default channel data
	ChannelsDefault(dev, device.Devices)

	// Device color
	SetDeviceColor(0, nil)

	// Speed and temp refresh
	AF = SetAutoRefresh(dev)

	// Monitor for device
	deviceMonitor = NewDeviceMonitor()

	logger.Log(logger.Fields{"device": device}).Info("Device successfully initialized")
}

// InitDevices will retrieve all available devices from a device
func InitDevices(dev *hid.Device) map[int]structs.LinkDevice {
	deviceList := make(map[int]structs.LinkDevice)
	var devices []structs.Devices

	response := comm.Read(
		dev,
		opcodes.GetOpcode(opcodes.OpcodeGetDevices),
		opcodes.GetOpcode(opcodes.OpcodeDevices),
	)

	channel := response.Data[6]
	index := response.Data[7:]
	position := 0

	for i := 1; i <= int(channel); i++ {
		deviceIdLen := index[position+7]
		if deviceIdLen == 0 {
			position += 8
			continue
		}
		deviceTypeModel := index[position : position+8]
		deviceId := index[position+8 : position+8+int(deviceIdLen)]

		hubDevice := &structs.Devices{
			ChannelId:   i,
			DeviceId:    string(deviceId),
			DeviceType:  deviceTypeModel[2],
			DeviceModel: deviceTypeModel[3],
		}

		devices = append(devices, *hubDevice)
		position += 8 + int(deviceIdLen)
	}

	if len(devices) < 1 {
		fmt.Println(fmt.Sprintf("[INFO] %s %s (%s)", device.Manufacturer, device.Product, device.Firmware))
		fmt.Println(fmt.Sprintf("[ERROR] Detected %d iCUE Link devices. Exit...", len(devices)))
		Stop()
		os.Exit(0)
	}

	for _, hubDevice := range devices {
		match := common.GetDevice(hubDevice.DeviceType, hubDevice.DeviceModel)
		if match == nil {
			continue
		}
		hubDeviceInfo := structs.LinkDevice{
			ChannelId:    hubDevice.ChannelId,
			Type:         hubDevice.DeviceType,
			DeviceId:     hubDevice.DeviceId,
			Name:         match.Name,
			DefaultValue: common.SetDefaultChannelData(hubDevice.DeviceType),
			Rpm:          0,
			Temperature:  0,
			LedChannels:  match.LedChannels,
			ContainsPump: common.ContainsPump(hubDevice.DeviceType),
		}
		deviceList[hubDevice.ChannelId] = hubDeviceInfo
	}

	currentColors = make(map[int]*structs.Color, len(deviceList))
	return deviceList
}

// SetDeviceColor will set device color
func SetDeviceColor(channelId int, customColor *structs.Color) {
	var i uint8 = 0
	var m = 0
	buf := map[int][]byte{}

	if customColor != nil {
		color := brightness.ModifyBrightness(
			*customColor,
			customColor.Brightness,
		)
		if channelId == 0 {
			// All channels
			for _, linkDevice := range device.Devices {
				LedChannels := linkDevice.LedChannels
				if LedChannels > 0 {
					for i = 0; i < LedChannels; i++ {
						buf[m] = []byte{
							byte(color.Red),
							byte(color.Green),
							byte(color.Blue),
						}
						m++
					}
				}
			}
		} else {
			// Change color on a specified channel
			currentColors[channelId] = color

			keys := make([]int, 0)
			for k := range currentColors {
				keys = append(keys, k)
			}
			sort.Ints(keys)

			// Loop through ordered keys
			for _, k := range keys {
				val, ok := device.Devices[k]
				if ok {
					LedChannels := val.LedChannels

					// Generate color bytes for each channel & led
					for i = 0; i < LedChannels; i++ {
						buf[m] = []byte{
							byte(currentColors[k].Red),
							byte(currentColors[k].Green),
							byte(currentColors[k].Blue),
						}
						m++
					}

				}
			}
		}

		// Send it!
		data := common.SetColor(buf)
		comm.Write(
			device.Handle,
			opcodes.GetOpcode(opcodes.OpcodeSetColor),
			opcodes.GetOpcode(opcodes.OpcodeColor),
			data,
			comm.EndpointTypeColor,
		)
		return
	}

	if config.GetConfig().UseCustomChannelIdColor {
		// Custom colors from configuration
		customChannelIdData := config.GetConfig().CustomChannelIdData

		// Check if anything is defined
		if len(customChannelIdData) < 1 {
			logger.Log(logger.Fields{}).Warn("Unable to find any custom channel in config.json")
			return
		}

		// We need to sort map keys properly before processing to avoid
		// colors being applied to different devices then its defined ones.
		keys := make([]int, 0)
		for k := range customChannelIdData {
			keys = append(keys, k)
		}
		sort.Ints(keys)

		// Loop through ordered keys
		for _, k := range keys {
			// Check if channelId actually exists
			val, ok := device.Devices[k]
			if ok {
				// Get number of LEDs per channel
				LedChannels := val.LedChannels
				if LedChannels > 0 {
					// Get color for specified channelId
					color := customChannelIdData[k]

					// Generate color based on config file
					deviceColor := brightness.ModifyBrightness(
						color.Color,
						color.Color.Brightness,
					)

					// Add current colors
					currentColors[k] = deviceColor

					// Generate color bytes for each channel & led
					for i = 0; i < LedChannels; i++ {
						buf[m] = []byte{
							byte(deviceColor.Red),
							byte(deviceColor.Green),
							byte(deviceColor.Blue),
						}
						m++
					}
				}
			} else {
				logger.Log(logger.Fields{"channelId": k}).Warn("Unable to find custom channel in config.json")
				continue
			}
		}
	} else {
		// default color on all devices
		color := brightness.ModifyBrightness(
			config.GetConfig().DefaultColor,
			config.GetConfig().DefaultColor.Brightness,
		)
		for _, linkDevice := range device.Devices {
			// Add current colors
			currentColors[linkDevice.ChannelId] = color

			// Get LED channels
			LedChannels := linkDevice.LedChannels
			if LedChannels > 0 {
				for i = 0; i < LedChannels; i++ {
					buf[m] = []byte{
						byte(color.Red),
						byte(color.Green),
						byte(color.Blue),
					}
					m++
				}
			}
		}
	}

	// Send it!
	data := common.SetColor(buf)
	comm.Write(
		device.Handle,
		opcodes.GetOpcode(opcodes.OpcodeSetColor),
		opcodes.GetOpcode(opcodes.OpcodeColor),
		data,
		comm.EndpointTypeColor,
	)
}

// SetDeviceSpeed will set device speed based on client input
func SetDeviceSpeed(channelId int, value uint16, mode uint8) int {
	speed := common.IntToByteArray(value)
	channelSpeeds := map[int][]byte{}
	channelSpeeds[channelId] = speed
	data := common.SetSpeed(channelSpeeds, mode)
	return comm.Write(
		device.Handle,
		opcodes.GetOpcode(opcodes.OpcodeSetSpeed),
		opcodes.GetOpcode(opcodes.OpcodeSpeed),
		data,
		comm.EndpointTypeDefault,
	)
}

// ChannelsDefault will initialize all channels default power when the program starts
func ChannelsDefault(dev *hid.Device, linkDevices map[int]structs.LinkDevice) {
	var data []byte
	channelDefaults := map[int][]byte{}

	// Custom speed defined in config
	if config.GetConfig().UseCustomChannelIdSpeed {
		customChannelIdData := config.GetConfig().CustomChannelIdData
		for linkDevice := range linkDevices {
			if speed, ok := customChannelIdData[linkDevice]; ok {
				SetDeviceSpeed(linkDevice, speed.Speed.Value, speed.Speed.Mode)
			} else {
				logger.Log(logger.Fields{"channelId": linkDevice}).Warn("Unable to find custom channel in config.json")
				continue
			}
		}
	} else {
		for linkDevice := range linkDevices {
			channelDefaults[linkDevice] = []byte{linkDevices[linkDevice].DefaultValue}
		}
		data = common.SetSpeed(channelDefaults, 0)
		comm.Write(
			dev,
			opcodes.GetOpcode(opcodes.OpcodeSetSpeed),
			opcodes.GetOpcode(opcodes.OpcodeSpeed),
			data,
			comm.EndpointTypeDefault,
		)
	}
}

// GetDeviceTemperature will retrieve all temperature sensors from devices
func GetDeviceTemperature() {
	response := comm.Read(
		device.Handle,
		opcodes.GetOpcode(opcodes.OpcodeGetTemperatures),
		opcodes.GetOpcode(opcodes.OpcodeTemperatures),
	).Data

	amount := response[6]
	sensorData := response[7:]
	sensors := make([]structs.TemperatureSensor, amount)

	for i, s := 0, 0; i < int(amount); i, s = i+1, s+3 {
		currentSensor := sensorData[s : s+3]
		status := currentSensor[0]
		var temperature float32
		if status == 0x00 {
			temp := int16(binary.LittleEndian.Uint16(currentSensor[1:3]))
			temperature = float32(temp) / 10.0
		}
		sensors[i] = structs.TemperatureSensor{
			ChannelId:   i,
			Status:      status,
			Temperature: temperature,
		}
	}

	for _, sensor := range sensors {
		if _, ok := device.Devices[sensor.ChannelId]; ok {
			if sensor.Status == 0x00 {
				temp := device.Devices[sensor.ChannelId]
				temp.Temperature = sensor.Temperature
				device.Devices[sensor.ChannelId] = temp
			}
		}
	}
}

// GetDeviceSpeed will retrieve all speed sensors from devices
func GetDeviceSpeed() {
	response := comm.Read(
		device.Handle,
		opcodes.GetOpcode(opcodes.OpcodeGetSpeeds),
		opcodes.GetOpcode(opcodes.OpcodeSpeeds),
	).Data

	amount := response[6]
	sensorData := response[7:]
	sensors := make([]structs.SpeedSensor, amount)

	for i := 0; i < int(amount); i++ {
		currentSensor := sensorData[i*3 : (i+1)*3]
		status := currentSensor[0]
		var rpmSpeed int16
		if status == 0x00 {
			rpmSpeed = int16(binary.LittleEndian.Uint16(currentSensor[1:3]))
		}
		sensors[i] = structs.SpeedSensor{
			ChannelId: i,
			Status:    status,
			Rpm:       rpmSpeed,
		}
	}

	for _, sensor := range sensors {
		if _, ok := device.Devices[sensor.ChannelId]; ok {
			if sensor.Status == 0x00 {
				temp := device.Devices[sensor.ChannelId]
				temp.Rpm = sensor.Rpm
				device.Devices[sensor.ChannelId] = temp
			}
		}
	}
}

// GetDeviceFirmware will return a device firmware version out as string
func GetDeviceFirmware(dev *hid.Device) string {
	fw, err := comm.Transfer(
		dev,
		opcodes.GetOpcode(opcodes.OpcodeGetFirmware),
		nil,
		nil,
	)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to write to a device")
	}

	v1, v2, v3 := int(fw[4]), int(fw[5]), int(binary.LittleEndian.Uint16(fw[6:8]))
	return fmt.Sprintf("%d.%d.%d", v1, v2, v3)
}

func GetDeviceMode(dev *hid.Device) byte {
	/*
		0 - Device is powered on and initialized
		1 - Device is powered on, and it's being initialized.
		This will be triggered when the machine wakes up from sleep.
	*/
	mode, err := comm.Transfer(
		dev,
		opcodes.GetOpcode(opcodes.OpcodeGetDeviceMode),
		nil,
		nil,
	)

	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to write to a device")
	}
	return mode[1]
}

// SetAutoRefresh will automatically refresh data from a device. We need to refresh device data constantly,
// since if a device is left without communication, it will automatically switch back to default hardware mode.
func SetAutoRefresh(dev *hid.Device) chan bool {
	ticker := time.NewTicker(time.Duration(config.GetConfig().PullingIntervalMs) * time.Millisecond)
	quit := make(chan bool)

	go func() {
		for {
			select {
			case <-ticker.C:
				SetDeviceStatus(GetDeviceMode(dev))
				GetDeviceSpeed()
				GetDeviceTemperature()
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
	return quit
}

// SetDeviceMode will switch a device to Hardware or Software mode
func SetDeviceMode(dev *hid.Device, mode []byte) {
	_, err := comm.Transfer(dev, mode, nil, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to change device mode")
	}
}

// NewDeviceMonitor initializes and returns a new Monitor
func NewDeviceMonitor() *structs.DeviceMonitor {
	m := &structs.DeviceMonitor{}
	m.Cond = sync.NewCond(&m.Lock)
	go WaitForDevice(func() {
		Stop()
		Init()
	})
	return m
}

// SetDeviceStatus sets the status and notifies a waiting goroutine if necessary
func SetDeviceStatus(val byte) {
	deviceMonitor.Lock.Lock()
	defer deviceMonitor.Lock.Unlock()
	deviceMonitor.Status = val
	deviceMonitor.Cond.Broadcast()
}

// WaitForDevice waits for the status to change from zero to one and back to zero before running the action
func WaitForDevice(action func()) {
	deviceMonitor.Lock.Lock()
	for deviceMonitor.Status != 1 {
		deviceMonitor.Cond.Wait()
	}
	deviceMonitor.Lock.Unlock()

	deviceMonitor.Lock.Lock()
	for deviceMonitor.Status != 0 {
		deviceMonitor.Cond.Wait()
	}
	deviceMonitor.Lock.Unlock()
	action()
}
