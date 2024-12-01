package psuhid

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/dashboard"
	"OpenLinkHub/src/logger"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sstallion/go-hid"
	"math"
	"os"
	"regexp"
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
	Rpm                int16   `json:"rpm"`
	Temperature        float32 `json:"temperature"`
	Volts              float32 `json:"volts"`
	Amps               float32 `json:"amps"`
	Watts              float32 `json:"watts"`
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
	dev           *hid.Device
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
	InputVoltage  float32
}

var (
	pwd                   = ""
	cmdInit               = byte(0xfe)
	cmdRead               = byte(0x03)
	cmdWrite              = byte(0x02)
	cmdSetFanMode         = byte(0xf0)
	dataFanModeDefault    = byte(0x00)
	dataFanModeManual     = byte(0x01)
	dataTempSensors       = []byte{0x8d, 0x8e}
	dataFanSpeed          = byte(0x90)
	dataSetFanSpeed       = byte(0x3b)
	cmdSwitch             = byte(0x00)
	data12V               = byte(0x00)
	data5V                = byte(0x01)
	data3V                = byte(0x02)
	dataGetAmps           = byte(0x8c)
	dataGetVolts          = byte(0x8b)
	dataGetWatts          = byte(0x96)
	dataPowerOut          = byte(0xee)
	dataInputVoltage      = byte(0x88)
	mutex                 sync.Mutex
	timer                 = &time.Ticker{}
	authRefreshChan       = make(chan bool)
	deviceRefreshInterval = 1000
	bufferSize            = 64
	readBufferSize        = 64
	bufferSizeWrite       = bufferSize + 1
	temperatureChannels   = 2
)

func Init(vendorId, productId uint16, serial string) *Device {
	// Set global working directory
	pwd = config.GetConfig().ConfigPath

	// Open device, return if failure
	dev, err := hid.Open(vendorId, productId, serial)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": vendorId, "productId": productId, "serial": serial}).Error("Unable to open HID device")
		return nil
	}

	// Init new struct with HID device
	d := &Device{
		dev:      dev,
		Template: "psuhid.html",
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

	// Bootstrap
	d.getManufacturer()    // Manufacturer
	d.getProductData()     // Product data
	d.loadDeviceProfiles() // Load all device profiles
	d.getInputVoltage()    // Input voltage
	d.getDevices()         // Get devices
	d.saveDeviceProfile()  // Save device profile
	d.updateFanMode()      // Fan speed
	d.setAutoRefresh()
	return d
}

// Stop will stop all device operations and switch a device back to hardware mode
func (d *Device) Stop() {
	logger.Log(logger.Fields{"serial": d.Serial}).Info("Stopping device...")
	timer.Stop()
	authRefreshChan <- true
	d.setFanToDefault()
	if d.dev != nil {
		err := d.dev.Close()
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to close HID device")
		}
	}
}

// GetDeviceTemplate will return device template name
func (d *Device) GetDeviceTemplate() string {
	return d.Template
}

// getInputVoltage will get input voltage type, either 230 or 115 volts
func (d *Device) getInputVoltage() {
	d.init()
	buf := d.createPacket(cmdRead, dataInputVoltage, 0)
	output, err := d.transfer(buf)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to init to PSU device")
	}
	inputVoltage := common.FromLinear11(output)
	d.InputVoltage = inputVoltage
}

// setFanToDefault will set fan mode to default value
func (d *Device) setFanToDefault() {
	d.init()
	buf := d.createPacket(cmdWrite, cmdSetFanMode, dataFanModeDefault)
	_, err := d.transfer(buf)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to set fam to default value")
	}
}

// createPacket will create a new packet and return byte slice
func (d *Device) createPacket(mode, command, data byte) []byte {
	buf := make([]byte, bufferSizeWrite)
	buf[1] = mode
	buf[2] = command
	if data > 0 {
		buf[3] = data
	}
	return buf
}

// getManufacturer will return device manufacturer
func (d *Device) getManufacturer() {
	manufacturer, err := d.dev.GetMfrStr()
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to get manufacturer")
	}
	d.Manufacturer = manufacturer
}

// getProductData will get product data
func (d *Device) getProductData() {
	packet := d.createPacket(cmdInit, cmdRead, 0)
	output, err := d.transfer(packet)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to init to PSU device")
	}

	product := string(output[2:])
	product = strings.ReplaceAll(product, "\x00", "")
	d.Product = product

	hash := md5.Sum([]byte(product))
	serial := hex.EncodeToString(hash[:])
	d.Serial = serial
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
	var buf []byte

	if d.DeviceProfile == nil {
		return
	}
	if d.DeviceProfile.FanMode == 0 {
		buf = d.createPacket(cmdWrite, cmdSetFanMode, dataFanModeDefault)
		_, err := d.transfer(buf)
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to set fam value")
			return
		}
	} else {
		fanSpeed := d.DeviceProfile.FanMode * 10
		if fanSpeed < 40 {
			fanSpeed = 40
		}

		if fanSpeed > 100 {
			fanSpeed = 100
		}

		buf = d.createPacket(cmdWrite, cmdSetFanMode, dataFanModeManual)
		_, err := d.transfer(buf)
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to set fam value")
			return
		}
		buf = d.createPacket(cmdWrite, dataSetFanSpeed, byte(fanSpeed))
		_, err = d.transfer(buf)
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to set fam value")
			return
		}
	}

}

// getDevices will generate list of devices
func (d *Device) getDevices() int {
	m := 0
	var devices = make(map[int]*Devices, 0)

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

	// 12V Current
	device = &Devices{ChannelId: m, DeviceId: fmt.Sprintf("%s-%v", "12vrail", m), Name: "12V Rail", Description: "Output Power", IsPowerProbe: true, Label: "12V Rail Stats", Rail: true}
	devices[m] = device
	m++

	// 5V Current
	device = &Devices{ChannelId: m, DeviceId: fmt.Sprintf("%s-%v", "5vrail", m), Name: "5V Rail", Description: "Output Power", IsPowerProbe: true, Label: "5V Rail Stats", Rail: true}
	devices[m] = device
	m++

	// 3V Current
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
	d.init()
	buf := d.createPacket(cmdRead, dataFanSpeed, 0)
	output, err := d.transfer(buf)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to init to PSU device")
	}
	rpm := common.FromLinear11(output)
	if _, ok := d.Devices[m]; ok {
		d.Devices[m].Rpm = int16(rpm)
	}
	m++

	// Temps
	for i := 0; i < temperatureChannels; i++ {
		d.init()
		buf = d.createPacket(cmdRead, dataTempSensors[i], 0)
		output, err = d.transfer(buf)
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to init to PSU device")
		}
		temp := common.FromLinear11(output)
		if _, ok := d.Devices[m]; ok {
			if temp > 1 {
				d.Devices[m].Temperature = temp
				d.Devices[m].TemperatureString = dashboard.GetDashboard().TemperatureToString(temp)
			}
		}
		m++
	}

	// Power out
	d.init()
	buf = d.createPacket(cmdWrite, cmdSwitch, data12V)
	output, err = d.transfer(buf)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to init to PSU device")
	}

	d.init()
	buf = d.createPacket(cmdRead, dataPowerOut, 0)
	output, err = d.transfer(buf)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to init to PSU device")
	}
	powerOutWatts := common.FromLinear11(output)
	if _, ok := d.Devices[m]; ok {
		d.Devices[m].Watts = powerOutWatts
		d.Devices[m].HasWatts = true
	}
	m++

	rails := []byte{data12V, data5V, data3V}
	for _, rail := range rails {
		// Switch rail
		d.init()
		buf = d.createPacket(cmdWrite, cmdSwitch, rail)
		output, err = d.transfer(buf)
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to init to PSU device")
		}

		// Get amps
		d.init()
		buf = d.createPacket(cmdRead, dataGetAmps, 0)
		output, err = d.transfer(buf)
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to init to PSU device")
		}
		railAmps := common.FromLinear11(output)

		// Get volts
		d.init()
		buf = d.createPacket(cmdRead, dataGetVolts, 0)
		output, err = d.transfer(buf)
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to init to PSU device")
		}
		railVolts := common.FromLinear11(output)

		// Get watts
		d.init()
		buf = d.createPacket(cmdRead, dataGetWatts, 0)
		output, err = d.transfer(buf)
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to init to PSU device")
		}
		railWatts := common.FromLinear11(output)

		// Update object
		if _, ok := d.Devices[m]; ok {
			d.Devices[m].Watts = float32(math.Floor(float64(railWatts*100)) / 100)
			d.Devices[m].Volts = float32(math.Floor(float64(railVolts*100)) / 100)
			d.Devices[m].Amps = float32(math.Floor(float64(railAmps*100)) / 100)
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
		logger.Log(logger.Fields{"error": err, "location": deviceProfile.Path}).Error("Unable to create new device profile")
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
	profileList := make(map[string]*DeviceProfile, 0)
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

// init will init to a PSU device
func (d *Device) init() {
	buf := d.createPacket(cmdInit, cmdRead, 0)
	_, err := d.transfer(buf)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to write data to a device")
	}
}

// setAutoRefresh will refresh device data
func (d *Device) setAutoRefresh() {
	timer = time.NewTicker(time.Duration(deviceRefreshInterval) * time.Millisecond)
	authRefreshChan = make(chan bool)
	go func() {
		for {
			select {
			case <-timer.C:
				d.getDeviceData()
			case <-authRefreshChan:
				timer.Stop()
				return
			}
		}
	}()
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
		return nil, err
	}

	// Get data from a device
	if _, err := d.dev.Read(bufferR); err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to read data from device")
		return nil, err
	}

	if buffer[1] == bufferR[0] && buffer[2] == bufferR[1] {
		return bufferR, nil
	} else {
		err := errors.New("response does not match the request")
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Invalid response. Probably another software is monitoring this device")
		return make([]byte, 64), err
	}
}
