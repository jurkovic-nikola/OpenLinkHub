package device

import (
	"OpenICUELinkHub/src/config"
	"OpenICUELinkHub/src/device/brightness"
	"OpenICUELinkHub/src/device/comm"
	"OpenICUELinkHub/src/device/common"
	"OpenICUELinkHub/src/device/opcodes"
	"OpenICUELinkHub/src/device/rgb"
	"OpenICUELinkHub/src/device/rgb/circle"
	"OpenICUELinkHub/src/device/rgb/colorpulse"
	"OpenICUELinkHub/src/device/rgb/colorshift"
	"OpenICUELinkHub/src/device/rgb/colorwarp"
	"OpenICUELinkHub/src/device/rgb/flickering"
	"OpenICUELinkHub/src/device/rgb/rainbow"
	"OpenICUELinkHub/src/device/rgb/spinner"
	"OpenICUELinkHub/src/device/rgb/watercolor"
	"OpenICUELinkHub/src/logger"
	"OpenICUELinkHub/src/structs"
	"encoding/binary"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"
)

var (
	ticker          *time.Ticker
	rgbChan         chan bool
	authRefreshChan chan bool
	device          *structs.Device
	deviceMonitor   *structs.DeviceMonitor
	currentColors   map[int]*structs.Color
	startTime       = time.Now()
	rgbSpeed        = 40
)

// GetDevice will return structs.Device
func GetDevice() *structs.Device {
	return device
}

// Stop will send the device back to hardware mode, usually when the program exits
func Stop() {
	authRefreshChan <- true
	setDeviceMode(opcodes.CmdHardwareMode)
	comm.Close()
}

// Init will initialize a device and prepare a device for receiving commands
func Init() {
	comm.Open()

	manufacturer := comm.GetManufacturer()
	product := comm.GetProduct()
	serial := comm.GetSerial()

	device = &structs.Device{
		Manufacturer: manufacturer,
		Product:      product,
		Serial:       serial,
		Firmware:     getDeviceFirmware(),
		Standalone:   config.GetConfig().Standalone,
	}

	// Activate software mode on device
	setDeviceMode(opcodes.CmdSoftwareMode)

	// Init all channels
	device.Devices = initDevices()

	// Default channel data
	channelsDefault(device.Devices)

	// Initialize RGB endpoint
	initColorEndpoint()

	// Device color
	SetDeviceColor(0, nil)

	// Speed and temp refresh
	setAutoRefresh()

	// Monitor for device
	deviceMonitor = newDeviceMonitor()

	logger.Log(logger.Fields{"device": device}).Info("Device successfully initialized")
}

// initColorEndpoint will initialize color endpoint for RGB
func initColorEndpoint() {
	// Close any RGB endpoint
	_, err := comm.Transfer(opcodes.CmdCloseEndpoint, opcodes.ModeSetColor, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to close endpoint")
	}

	// Open RGB endpoint
	_, err = comm.Transfer(opcodes.CmdOpenColorEndpoint, opcodes.ModeSetColor, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to open endpoint")
	}
}

// initDevices will retrieve all available devices from a device
func initDevices() map[int]structs.LinkDevice {
	deviceList := make(map[int]structs.LinkDevice)
	var devices []structs.Devices

	response := comm.Read(opcodes.ModeGetDevices, opcodes.DataTypeGetDevices)
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
			Model:        hubDevice.DeviceModel,
			DeviceId:     hubDevice.DeviceId,
			Name:         match.Name,
			DefaultValue: common.SetDefaultChannelData(match),
			Rpm:          0,
			Temperature:  0,
			LedChannels:  match.LedChannels,
			ContainsPump: match.ContainsPump,
		}
		deviceList[hubDevice.ChannelId] = hubDeviceInfo
	}

	currentColors = make(map[int]*structs.Color, len(deviceList))
	return deviceList
}

// SetDeviceColor will set device color
func SetDeviceColor(channelId int, customColor *structs.Color) {
	if rgb.IsGRBEnabled() {
		setDeviceRGBMode()
		return
	}

	var i uint8 = 0
	var m = 0
	buf := map[int][]byte{}

	if customColor != nil {
		color := brightness.ModifyBrightness(*customColor)
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
		comm.WriteColor(opcodes.DataTypeSetColor, data)
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
					deviceColor := brightness.ModifyBrightness(color.Color)

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
		color := brightness.ModifyBrightness(config.GetConfig().DefaultColor)
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
	comm.WriteColor(opcodes.DataTypeSetColor, data)
}

// SetDeviceSpeed will set device speed based on client input
func SetDeviceSpeed(channelId int, value uint16, mode uint8) int {
	speed := common.IntToByteArray(value)
	channelSpeeds := map[int][]byte{}
	channelSpeeds[channelId] = speed
	data := common.SetSpeed(channelSpeeds, mode)
	return comm.Write(opcodes.ModeSetSpeed, opcodes.DataTypeSetSpeed, data)
}

// channelsDefault will initialize all channels default power when the program starts
func channelsDefault(linkDevices map[int]structs.LinkDevice) {
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
		comm.Write(opcodes.ModeSetSpeed, opcodes.DataTypeSetSpeed, data)
	}
}

// getDeviceTemperature will retrieve all temperature sensors from devices
func getDeviceTemperature() {
	response := comm.Read(
		opcodes.ModeGetTemperatures,
		opcodes.DataTypeGetTemperatures,
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

// getDeviceSpeed will retrieve all speed sensors from devices
func getDeviceSpeed() {
	response := comm.Read(
		opcodes.ModeGetSpeeds,
		opcodes.DataTypeGetSpeeds,
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

// getDeviceFirmware will return a device firmware version out as string
func getDeviceFirmware() string {
	fw, err := comm.Transfer(
		opcodes.CmdGetFirmware,
		nil,
		nil,
	)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to write to a device")
	}

	v1, v2, v3 := int(fw[4]), int(fw[5]), int(binary.LittleEndian.Uint16(fw[6:8]))
	return fmt.Sprintf("%d.%d.%d", v1, v2, v3)
}

// getDeviceMode will return current device mode
func getDeviceMode() byte {
	/*
		0 - Device is powered on and initialized
		1 - Device is powered on, and it's being initialized.
		This will be triggered when the machine wakes up from sleep.
	*/
	mode, err := comm.Transfer(opcodes.CmdGetDeviceMode, nil, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to write to a device")
	}
	return mode[1]
}

// SetAutoRefresh will automatically refresh data from a device. We need to refresh device data constantly,
// since if a device is left without communication, it will automatically switch back to default hardware mode.
func setAutoRefresh() {
	timer := time.NewTicker(time.Duration(config.GetConfig().PullingIntervalMs) * time.Millisecond)
	authRefreshChan = make(chan bool)

	go func() {
		for {
			select {
			case <-timer.C:
				getDeviceStatus(getDeviceMode())
				getDeviceSpeed()
				getDeviceTemperature()
			case <-authRefreshChan:
				timer.Stop()
				return
			}
		}
	}()
}

// setDeviceRGBMode will configure custom RGB mode based from service configuration
func setDeviceRGBMode() {
	rgbMode := rgb.GetRGBMode()
	if rgbMode == nil {
		logger.Log(logger.Fields{}).Info("Unable to find specified RGB mode. Check your configuration")
		return
	}

	// Get the number of LED channels we have
	ledChannels := 0
	for _, linkDevice := range device.Devices {
		ledChannels += int(linkDevice.LedChannels)
	}

	// Do we have any RGB component in the system?
	if ledChannels == 0 {
		logger.Log(logger.Fields{}).Info("No RGB compatible devices found")
		return
	}

	// RGB data
	rgbCustomColor := true
	rgbModeSpeed := rgbMode.Speed
	rgbModeSpeed = common.Clamp(rgbModeSpeed, 1, 10)

	rgbModeName := rgb.GetRGBModeName()
	rgbModeBrightness := rgbMode.Brightness
	rgbLoopDuration := time.Duration(rgbModeSpeed) * time.Second
	rgbStartColor := common.GenerateRandomColor(rgbModeBrightness)
	rgbEndColor := common.GenerateRandomColor(rgbModeBrightness)
	rgbSmoothness := rgbMode.Smoothness

	// Check if we have custom colors
	if (structs.Color{}) == rgbMode.StartColor || (structs.Color{}) == rgbMode.EndColor {
		rgbCustomColor = false
	}

	// Custom color is set, override values
	if rgbCustomColor {
		rgbStartColor = &rgbMode.StartColor
		rgbEndColor = &rgbMode.EndColor
	}

	rgbSmoothness = common.Clamp(rgbSmoothness, 1, 40)

	// Timer
	ticker = time.NewTicker(time.Duration(rgbSpeed) * time.Millisecond)
	rgbChan = make(chan bool)

	go func(lc, smoothness int, mode string, bts float64) {
		for {
			select {
			case <-ticker.C:
				elapsed := time.Since(startTime).Seconds() * float64(rgbModeSpeed)
				switch mode {
				case "rainbow":
					rainbow.Init(lc, elapsed, bts)
				case "watercolor":
					watercolor.Init(lc, elapsed, bts)
				case "colorpulse":
					colorpulse.Init(lc, smoothness, rgbLoopDuration, rgbStartColor, rgbEndColor, bts)
				case "colorshift":
					colorshift.Init(lc, smoothness, rgbCustomColor, rgbLoopDuration, rgbStartColor, rgbEndColor, bts)
				case "circle", "circleshift":
					circle.Init(lc, rgbLoopDuration, rgbStartColor, rgbEndColor, bts)
				case "flickering":
					flickering.Init(lc, rgbLoopDuration, rgbCustomColor, rgbStartColor, rgbEndColor, bts)
				case "colorwarp":
					colorwarp.Init(lc, smoothness, rgbLoopDuration, bts)
				case "snipper":
					spinner.Init(lc, rgbStartColor, rgbEndColor, bts)
				}
			case <-rgbChan:
				ticker.Stop()
			}
		}
	}(ledChannels, rgbSmoothness, rgbModeName, rgbModeBrightness)
}

// setDeviceMode will switch a device to Hardware or Software mode
func setDeviceMode(mode []byte) {
	_, err := comm.Transfer(mode, nil, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to change device mode")
	}
}

// newDeviceMonitor initializes and returns a new Monitor
func newDeviceMonitor() *structs.DeviceMonitor {
	m := &structs.DeviceMonitor{}
	m.Cond = sync.NewCond(&m.Lock)
	go waitForDevice(func() {
		Stop()
		Init()
	})
	return m
}

// getDeviceStatus sets the status and notifies a waiting goroutine if necessary
func getDeviceStatus(val byte) {
	deviceMonitor.Lock.Lock()
	defer deviceMonitor.Lock.Unlock()
	deviceMonitor.Status = val
	deviceMonitor.Cond.Broadcast()
}

// waitForDevice waits for the status to change from zero to one and back to zero before running the action
func waitForDevice(action func()) {
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
