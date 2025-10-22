package psudongle

// Package: CORSAIR DONGLE PSUs
// This is the primary package for CORSAIR DONGLE PSUs.
// All device actions are controlled from this package.
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/dashboard"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/serial"
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Devices struct {
	ChannelId          int     `json:"channelId"`
	Type               byte    `json:"type"`
	Model              byte    `json:"-"`
	DeviceId           string  `json:"deviceId"`
	Name               string  `json:"name"`
	Description        string  `json:"description"`
	Profile            string  `json:"profile"`
	Label              string  `json:"label"`
	Rpm                float64 `json:"rpm"`
	Temperature        float64 `json:"temperature"`
	Volts              float64 `json:"volts"`
	Amps               float64 `json:"amps"`
	Watts              float64 `json:"watts"`
	TemperatureString  string  `json:"temperatureString"`
	HasSpeed           bool
	HasTemps           bool
	HasWatts           bool
	HasVolts           bool
	HasAmps            bool
	IsTemperatureProbe bool
	IsPowerProbe       bool
	ContainsPump       bool
	Output             bool
	Rail               bool
}

type DeviceProfile struct {
	Active  bool
	Path    string
	Product string
	Serial  string
	FanMode int
}

type Device struct {
	ProductId     uint16
	dev           *serial.Device
	Manufacturer  string                    `json:"manufacturer"`
	Product       string                    `json:"product"`
	Serial        string                    `json:"serial"`
	Firmware      string                    `json:"firmware"`
	Devices       map[int]*Devices          `json:"devices"`
	UserProfiles  map[string]*DeviceProfile `json:"userProfiles"`
	DeviceProfile *DeviceProfile
	Template      string
	CpuTemp       float32
	GpuTemp       float32
	FanModes      map[int]string
	InputVoltage  float64
	Path          string
	instance      *common.Device
}

var (
	pwd                   = ""
	dataRail12V           = byte(0x00)
	dataRail5V            = byte(0x01)
	dataRail3V            = byte(0x02)
	cmdSetFanMode         = byte(0xf0)
	cmdSetFanSpeed        = byte(0x3b)
	cmdProduct            = byte(0x9a)
	cmdGetFanSpeed        = byte(0x90)
	cmdOutputtPower       = byte(0xee)
	cmdRailGetAmps        = byte(0x8c)
	cmdRailGetVolts       = byte(0x8b)
	cmdRailGetWatts       = byte(0x96)
	cmdSwitchRail         = byte(0x00)
	dataFanModeDefault    = []byte{0x00}
	dataFanModeManual     = []byte{0x01}
	cmdTempSensors        = []byte{0x8d, 0x8e} // VRM, PSU temp
	cmdInputVoltage       = byte(0x88)
	mutex                 sync.Mutex
	timer                 = &time.Ticker{}
	autoRefreshChan       = make(chan bool)
	deviceRefreshInterval = 1000
	readBufferSize        = 256
	temperatureChannels   = 2
)

// encodeTable is used to encode data towards the device
var encodeTable = [16]uint8{
	0x55, 0x56, 0x59, 0x5A, 0x65, 0x66, 0x69, 0x6A, 0x95, 0x96, 0x99, 0x9A, 0xA5, 0xA6, 0xA9, 0xAA,
}

// decodeTable is used to decode data from the device
var decodeTable = [256]uint8{
	0x30, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x10, 0x20, 0x21, 0x00, 0x12, 0x22, 0x23, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x14, 0x24, 0x25, 0x00, 0x16, 0x26, 0x27, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x18, 0x28, 0x29, 0x00, 0x1A, 0x2A, 0x2B, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x1C, 0x2C, 0x2D, 0x00, 0x1E, 0x2E, 0x2F, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
}

func Init(vendorId, productId uint16, _, _ string) *common.Device {
	// Set global working directory
	pwd = config.GetConfig().ConfigPath

	devs, err := common.FindTtyByUsbId(vendorId, productId)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": vendorId, "productId": productId}).Warn("Unable to find TTY device")
		return nil
	}

	if len(devs) == 0 {
		logger.Log(logger.Fields{"vendorId": vendorId, "productId": productId}).Warn("Unable to find TTY device")
		return nil
	}

	cfg := &serial.Config{Name: devs[0], Baud: 115200}
	dev, err := serial.Open(cfg)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": vendorId, "productId": productId, "path": devs[0]}).Error("Unable to open TTY device")
		return nil
	}

	// Init new struct with HID device
	d := &Device{
		Path:      devs[0],
		ProductId: productId,
		dev:       dev,
		Template:  "psudongle.html",
		FanModes: map[int]string{
			0:  "Default",
			4:  "40 %",
			5:  "50 %",
			6:  "60 %",
			7:  "70 %",
			8:  "80 %",
			9:  "90 %",
			10: "100 %",
		},
	}

	if d.setupDevice() == 0 {
		logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Error("Failed to setup TTY device")
		return nil
	}

	// Bootstrap
	d.getManufacturer()    // Manufacturer
	d.getSerial()          // Serial
	d.getProduct()         // Product data
	d.loadDeviceProfiles() // Load all device profiles
	d.getInputVoltage()    // Input voltage
	d.getDevices()         // Get devices
	d.saveDeviceProfile()  // Save device profile
	d.updateFanMode()      // Fan speed
	d.setAutoRefresh()     // Auto refresh
	d.createDevice()       // Device register
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device successfully initialized")

	return d.instance
}

// createDevice will create new device register object
func (d *Device) createDevice() {
	d.instance = &common.Device{
		ProductType: common.ProductTypePSUDongle,
		Product:     d.Product,
		Serial:      d.Serial,
		Firmware:    d.Firmware,
		Image:       "icon-psu.svg",
		Instance:    d,
		GetDevice:   d,
	}
}

// Stop will stop all device operations and switch a device back to hardware mode
func (d *Device) Stop() {
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Stopping device...")
	timer.Stop()
	autoRefreshChan <- true
	d.setFanToDefault()
	if d.dev != nil {
		err := d.dev.Close()
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to close TTY device")
		}
	}
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device stopped")
}

// StopDirty will stop device in a dirty way
func (d *Device) StopDirty() uint8 {
	logger.Log(logger.Fields{"serial": d.Serial}).Info("Stopping device (dirty)...")
	timer.Stop()
	autoRefreshChan <- true
	return 1
}

// GetDeviceTemplate will return device template name
func (d *Device) GetDeviceTemplate() string {
	return d.Template
}

// getSerial will set the device serial number from PID
func (d *Device) getSerial() {
	d.Serial = strconv.Itoa(int(d.ProductId))
}

// getInputVoltage will get input voltage type, either 230 or 115 volts
func (d *Device) getInputVoltage() {
	inputVoltage := d.Read(cmdInputVoltage)
	d.InputVoltage = d.Byte2Float(inputVoltage)
}

// setFanToDefault will set fan mode to default value
func (d *Device) setFanToDefault() {
	d.Write(cmdSetFanMode, dataFanModeDefault)
}

// getManufacturer will return device manufacturer
func (d *Device) getManufacturer() {
	d.Manufacturer = "Corsair"
}

// getProduct will get product data
func (d *Device) getProduct() {
	product := d.Read(cmdProduct)
	product = bytes.Trim(product, "\x00")
	d.Product = string(product)
}

// UpdatePsuFan will update PSU fan speed
func (d *Device) UpdatePsuFan(mode int) uint8 {
	if d.DeviceProfile == nil {
		return 0
	}

	if mode > 10 || mode < 0 {
		return 0
	}

	d.DeviceProfile.FanMode = mode
	d.saveDeviceProfile()
	d.updateFanMode()
	return 1
}

// updateFanMode will update fan operation mode
func (d *Device) updateFanMode() {
	if d.DeviceProfile == nil {
		return
	}
	if d.DeviceProfile.FanMode == 0 {
		d.Write(cmdSetFanMode, dataFanModeDefault)
	} else {
		fanSpeed := d.DeviceProfile.FanMode * 10
		if fanSpeed < 40 {
			fanSpeed = 40
		}

		if fanSpeed > 100 {
			fanSpeed = 100
		}

		// Switch to manual mode
		d.Write(cmdSetFanMode, dataFanModeManual)

		// Set fan percent
		d.Write(cmdSetFanSpeed, []byte{byte(fanSpeed)})
	}
}

// getDevices will generate list of devices
func (d *Device) getDevices() int {
	m := 0
	var devices = make(map[int]*Devices)

	// Fan
	device := &Devices{ChannelId: m, DeviceId: fmt.Sprintf("%s-%v", "Fan", m), Name: fmt.Sprintf("Fan %d", 1), Rpm: 0, HasSpeed: true, Label: "PSU Fan"}
	devices[m] = device
	m++

	// Temperature probes
	for i := 0; i < temperatureChannels; i++ {
		switch i {
		case 0:
			device = &Devices{ChannelId: m, DeviceId: fmt.Sprintf("%s-%v", "Probe", i), Name: "VRM Temperature", Temperature: 0, Description: "Probe", HasTemps: true, IsTemperatureProbe: true, Label: "Probe"}
		case 1:
			device = &Devices{ChannelId: m, DeviceId: fmt.Sprintf("%s-%v", "Probe", i), Name: "PSU Temperature", Temperature: 0, Description: "Probe", HasTemps: true, IsTemperatureProbe: true, Label: "Probe"}
		}
		devices[m] = device
		m++
	}

	// Power Out
	device = &Devices{ChannelId: m, DeviceId: fmt.Sprintf("%s-%v", "PowerOut", m), Name: "Power Out", Description: "Output Power", IsPowerProbe: true, Label: "Output Power", Output: true}
	devices[m] = device
	m++

	// 12V Rail
	device = &Devices{ChannelId: m, DeviceId: fmt.Sprintf("%s-%v", "12vrail", m), Name: "12V Rail", Description: "Output Power", IsPowerProbe: true, Label: "12V Rail Stats", Rail: true}
	devices[m] = device
	m++

	// 5V Rail
	device = &Devices{ChannelId: m, DeviceId: fmt.Sprintf("%s-%v", "5vrail", m), Name: "5V Rail", Description: "Output Power", IsPowerProbe: true, Label: "5V Rail Stats", Rail: true}
	devices[m] = device
	m++

	// 3V Rail
	device = &Devices{ChannelId: m, DeviceId: fmt.Sprintf("%s-%v", "3vrail", m), Name: "3V Rail", Description: "Output Power", IsPowerProbe: true, Label: "3V Rail Stats", Rail: true}
	devices[m] = device
	m++

	d.Devices = devices
	return len(devices)
}

// getDeviceData will fetch device data, from temperatures, fan speed, etc...
func (d *Device) getDeviceData() {
	m := 0

	// Fan
	fanRpm := d.Read(cmdGetFanSpeed)
	if _, ok := d.Devices[m]; ok {
		d.Devices[m].Rpm = d.Byte2Float(fanRpm)
	}
	m++

	// Temps
	for i := 0; i < temperatureChannels; i++ {
		output := d.Read(cmdTempSensors[i])
		temp := d.Byte2Float(output)
		if _, ok := d.Devices[m]; ok {
			if temp > 1 {
				d.Devices[m].Temperature = temp
				d.Devices[m].TemperatureString = dashboard.GetDashboard().TemperatureToString(float32(temp))
			}
		}
		m++
	}

	// Power Out
	powerOut := d.Read(cmdOutputtPower)
	if _, ok := d.Devices[m]; ok {
		powerOutW := d.Byte2Float(powerOut)
		d.Devices[m].Watts = powerOutW
		d.Devices[m].HasWatts = true
	}
	m++

	rails := []byte{dataRail12V, dataRail5V, dataRail3V}
	for _, rail := range rails {
		// Set rail
		d.Write(cmdSwitchRail, []byte{rail})

		// Get amps
		output := d.Read(cmdRailGetAmps)
		railAmps := d.Byte2Float(output)

		// Get volts
		output = d.Read(cmdRailGetVolts)
		railVolts := d.Byte2Float(output)

		// Get watts
		output = d.Read(cmdRailGetWatts)
		railWatts := d.Byte2Float(output)

		// Update object
		if _, ok := d.Devices[m]; ok {
			d.Devices[m].Watts = railWatts
			d.Devices[m].Volts = math.Floor(railVolts*100) / 100
			d.Devices[m].Amps = math.Floor(railAmps*100) / 100
			d.Devices[m].HasWatts = true
			d.Devices[m].HasAmps = true
			d.Devices[m].HasVolts = true
		}
		m++
	}
}

// saveDeviceProfile will save device profile for persistent configuration
func (d *Device) saveDeviceProfile() {
	profilePath := pwd + "/database/profiles/" + d.Serial + ".json"
	deviceProfile := &DeviceProfile{
		Product: d.Product,
		Serial:  d.Serial,
		Path:    profilePath,
	}

	// First save, assign saved profile to a device
	if d.DeviceProfile == nil {
		deviceProfile.Active = true
		d.DeviceProfile = deviceProfile
	} else {
		deviceProfile.Active = d.DeviceProfile.Active
		deviceProfile.FanMode = d.DeviceProfile.FanMode
	}

	// Convert to JSON
	buffer, err := json.MarshalIndent(deviceProfile, "", "    ")
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to convert to json format")
		return
	}

	// Create profile filename
	file, fileErr := os.Create(deviceProfile.Path)
	if fileErr != nil {
		logger.Log(logger.Fields{"error": fileErr, "location": deviceProfile.Path}).Error("Unable to create new device profile")
		return
	}

	// Write JSON buffer to file
	_, err = file.Write(buffer)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": deviceProfile.Path}).Error("Unable to write data")
		return
	}

	// Close file
	err = file.Close()
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": deviceProfile.Path}).Error("Unable to close file handle")
	}
	d.loadDeviceProfiles() // Reload
}

// loadDeviceProfiles will load custom user profiles
func (d *Device) loadDeviceProfiles() {
	profileList := make(map[string]*DeviceProfile)
	userProfileDirectory := pwd + "/database/profiles/"

	files, err := os.ReadDir(userProfileDirectory)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": userProfileDirectory, "serial": d.Serial}).Fatal("Unable to read content of a folder")
	}

	for _, fi := range files {
		pf := &DeviceProfile{}
		if fi.IsDir() {
			continue // Exclude folders if any
		}

		// Define a full path of filename
		profileLocation := userProfileDirectory + fi.Name()

		// Check if filename has .json extension
		if !common.IsValidExtension(profileLocation, ".json") {
			continue
		}

		fileName := strings.Split(fi.Name(), ".")[0]
		if m, _ := regexp.MatchString("^[a-zA-Z0-9-]+$", fileName); !m {
			continue
		}

		fileSerial := ""
		if strings.Contains(fileName, "-") {
			fileSerial = strings.Split(fileName, "-")[0]
		} else {
			fileSerial = fileName
		}

		if fileSerial != d.Serial {
			continue
		}

		file, err := os.Open(profileLocation)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial, "location": profileLocation}).Warn("Unable to load profile")
			continue
		}
		if err = json.NewDecoder(file).Decode(pf); err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial, "location": profileLocation}).Warn("Unable to decode profile")
			continue
		}
		err = file.Close()
		if err != nil {
			logger.Log(logger.Fields{"location": profileLocation, "serial": d.Serial}).Warn("Failed to close file handle")
		}

		if pf.Serial == d.Serial {
			if fileName == d.Serial {
				profileList["default"] = pf
			} else {
				name := strings.Split(fileName, "-")[1]
				profileList[name] = pf
			}
			logger.Log(logger.Fields{"location": profileLocation, "serial": d.Serial}).Info("Loaded custom user profile")
		}
	}
	d.UserProfiles = profileList
	d.getDeviceProfile()
}

// getDeviceProfile will load persistent device configuration
func (d *Device) getDeviceProfile() {
	if len(d.UserProfiles) == 0 {
		logger.Log(logger.Fields{"serial": d.Serial}).Warn("No profile found for device. Probably initial start")
	} else {
		for _, pf := range d.UserProfiles {
			if pf.Active {
				d.DeviceProfile = pf
			}
		}
	}
}

// setAutoRefresh will refresh device data
func (d *Device) setAutoRefresh() {
	timer = time.NewTicker(time.Duration(deviceRefreshInterval) * time.Millisecond)
	autoRefreshChan = make(chan bool)
	go func() {
		for {
			select {
			case <-timer.C:
				d.getDeviceData()
			case <-autoRefreshChan:
				timer.Stop()
				return
			}
		}
	}()
}

// Byte2Float convert byte into a float
func (d *Device) Byte2Float(data []byte) float64 {
	if len(data) < 2 {
		return 0
	}

	p1 := int((data[1] >> 3) & 31)
	if p1 > 15 {
		p1 -= 32
	}

	p2 := (int(data[1])&7)*256 + int(data[0])
	if p2 > 1024 {
		p2 = -(65536 - (p2 | 63488))
	}

	return float64(p2) * math.Pow(2.0, float64(p1))
}

// setupDevice will init dongle and device
func (d *Device) setupDevice() uint8 {
	buf := make([]byte, 7)
	buf[0] = 0x11
	buf[1] = 0x02
	buf[2] = 0x64
	encoded := d.encode(buf)
	_, err := d.transfer(encoded)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to init device")
		return 0
	}
	return 1
}

// encode will encode data for device
func (d *Device) encode(data []byte) []byte {
	length := len(data)
	size := (length * 2) + 2
	buf := make([]byte, size)

	buf[0] = encodeTable[(0x00<<1)&0x0F] & 0xFC
	buf[size-1] = 0

	j := 1
	for i := 0; i < length; i++ {
		buf[j] = encodeTable[data[i]&0x0F]
		buf[j+1] = encodeTable[data[i]>>4]
		j += 2
	}

	return buf
}

// decode will decode data from device
func (d *Device) decode(data []byte) []byte {
	length := len(data)
	if length < 2 {
		return nil
	}

	size := length / 2
	if size <= 0 {
		return nil
	}

	if ((decodeTable[data[0]] & 0x0F) >> 1) != 7 {
		return nil
	}

	buf := make([]byte, size)
	j := 0
	for i := 1; i+1 < size; i += 2 {
		buf[j] = (decodeTable[data[i]] & 0x0F) | ((decodeTable[data[i+1]] & 0x0F) << 4)
		j++
	}

	return buf
}

func (d *Device) Read(command byte) []byte {
	var length = byte(0x02)
	if command == cmdProduct {
		length = 0x07
	}

	buf := make([]byte, 7)
	buf[0] = 0x13
	buf[1] = 0x03
	buf[2] = 0x06
	buf[3] = 0x01
	buf[4] = 0x07
	buf[5] = length
	buf[6] = command

	encoded := d.encode(buf)
	_, err := d.transfer(encoded)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to read data from device")
	}

	buf = make([]byte, 3)
	buf[0] = 0x08
	buf[1] = 0x07
	buf[2] = length

	encoded = d.encode(buf)
	buffer, err := d.transfer(encoded)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to read data from device")
	}
	return d.decode(buffer)
}

// Write will write data to the device and get output
func (d *Device) Write(command byte, data []byte) []byte {
	buf := make([]byte, len(data)+5)
	buf[0] = 0x13
	buf[1] = 0x01
	buf[2] = 0x04
	buf[3] = byte(len(data) + 1)
	buf[4] = command
	copy(buf[5:], data)

	encoded := d.encode(buf)
	buffer, err := d.transfer(encoded)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to a device")
		return nil
	}
	return d.decode(buffer)
}

// transfer will send data to a device and retrieve device output
func (d *Device) transfer(buffer []byte) ([]byte, error) {
	// Packet control, mandatory for this device
	mutex.Lock()
	defer mutex.Unlock()

	// Create read buffer
	bufferR := make([]byte, readBufferSize)

	// Send command to a device
	if _, err := d.dev.Write(buffer); err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to a device")
		return bufferR, err
	}

	// Get data from a device
	if _, err := d.dev.Read(bufferR); err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to read data from device")
		return bufferR, err
	}

	return bufferR, nil
}
