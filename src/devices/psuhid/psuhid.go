package psuhid

// Package: CORSAIR HID PSUs
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/dashboard"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/metrics"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sstallion/go-hid"
)

type Devices struct {
	ChannelId            int     `json:"channelId"`
	Type                 byte    `json:"type"`
	Model                byte    `json:"-"`
	DeviceId             string  `json:"deviceId"`
	Name                 string  `json:"name"`
	Description          string  `json:"description"`
	Profile              string  `json:"profile"`
	Label                string  `json:"label"`
	Rpm                  int16   `json:"rpm"`
	Volts                float32 `json:"volts"`
	Amps                 float32 `json:"amps"`
	Watts                float32 `json:"watts"`
	Temperature          float32 `json:"temperature"`
	TemperatureString    string  `json:"temperatureString"`
	VrmTemperature       float32 `json:"vrmTemperature"`
	VrmTemperatureString string  `json:"vrmTemperatureString"`
	PsuTemperature       float32 `json:"psuTemperature"`
	PsuTemperatureString string  `json:"psuTemperatureString"`
	HasSpeed             bool
	HasTemps             bool
	HasWatts             bool
	HasVolts             bool
	HasAmps              bool
	IsTemperatureProbe   bool
	IsPowerProbe         bool
	ContainsPump         bool
	Output               bool
	Rail                 bool
	IsPSU                bool
	MainPSU              bool
}

type DeviceProfile struct {
	Active  bool
	Path    string
	Product string
	Serial  string
	FanMode int
	RgbOff  bool
}

type Device struct {
	dev             *hid.Device
	deviceKey       string
	mutex           sync.Mutex
	timer           *time.Ticker
	autoRefreshChan chan bool
	refreshMutex    sync.Mutex    // timer/autoRefreshChan: Stop, StopDirty and claimPort can race
	refreshDone     chan struct{} // closed when the refresh loop has returned
	refreshStopped  bool          // stopped for good: a retired Device must not start its loop later
	vendorId        uint16
	productId       uint16
	usbPath         string                    // USB port; a re-enumerated PSU comes back on the same one
	lost            bool                      // the HID handle died (PSU dropped off USB); guarded by mutex
	closed          bool                      // dev was closed; hidapi frees it, so close it once. Guarded by mutex
	fanMutex        sync.Mutex                // fan mode changes: the API, and the restore after a reopen
	Manufacturer    string                    `json:"manufacturer"`
	Product         string                    `json:"product"`
	Serial          string                    `json:"serial"`
	Firmware        string                    `json:"firmware"`
	Devices         map[int]*Devices          `json:"devices"`
	UserProfiles    map[string]*DeviceProfile `json:"userProfiles"`
	DeviceProfile   *DeviceProfile
	Template        string
	CpuTemp         float32
	GpuTemp         float32
	FanModes        map[int]string
	InputVoltage    float32
	instance        *common.Device
	IsPSU           bool
}

var (
	pwd                   = ""
	hidMutex              sync.Mutex             // psuhid's enumerate/open: hidapi shares a global error string
	byPort                = map[string]*Device{} // the live Device on each USB port; guarded by hidMutex
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
	deviceRefreshInterval = 1000
	readTimeout           = 1000 * time.Millisecond
	bufferSize            = 64
	readBufferSize        = 64
	bufferSizeWrite       = bufferSize + 1
	temperatureChannels   = 2
)

func Init(vendorId, productId uint16, key, path string) *common.Device {
	// Set global working directory
	pwd = config.GetConfig().ConfigPath

	// Open device, return if failure
	hidMutex.Lock()
	dev, err := hid.OpenPath(path)
	hidMutex.Unlock()
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": vendorId, "productId": productId, "path": path}).Error("Unable to open HID device")
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
		IsPSU:     true,
		deviceKey: key,
		vendorId:  vendorId,
		productId: productId,
	}
	d.usbPath, _ = common.GetShortUSBDevPath(filepath.Base(path))

	// Bootstrap
	d.getManufacturer()    // Manufacturer
	d.getProductData()     // Product data
	d.loadDeviceProfiles() // Load all device profiles
	d.getInputVoltage()    // Input voltage
	d.getDevices()         // Get devices
	d.saveDeviceProfile()  // Save device profile
	d.updateFanMode()      // Fan speed
	d.claimPort()          // Retire a replaced Device still polling this port
	d.setAutoRefresh()     // Auto refresh
	d.createDevice()       // Device register
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device successfully initialized")

	return d.instance
}

// createDevice will create new device register object
func (d *Device) createDevice() {
	d.instance = &common.Device{
		ProductType: common.ProductTypePSUHid,
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
	logger.Log(logger.Fields{"serial": d.Serial}).Info("Stopping device...")
	d.stopAutoRefresh()
	d.setFanToDefault()
	d.mutex.Lock()
	defer d.mutex.Unlock()
	if err := d.closeDev(); err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to close HID device")
	}
	d.lost = true // nothing may use the closed handle again
}

// closeDev closes the HID handle once, whoever gets there first. Caller holds mutex.
func (d *Device) closeDev() error {
	if d.dev == nil || d.closed {
		return nil
	}
	d.closed = true
	return d.dev.Close()
}

// StopDirty will stop device in a dirty way
func (d *Device) StopDirty() uint8 {
	logger.Log(logger.Fields{"serial": d.Serial}).Info("Stopping device (dirty)...")
	d.stopAutoRefresh()
	return 1
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
	product = strings.Replace(product, "CORSAIR ", "", -1)
	product = strings.Replace(product, " PSU", "", -1)
	product = strings.Replace(product, "Power Supply", "", -1)

	d.Product = product

	seed := product
	if i := strings.Index(d.deviceKey, "-"); i >= 0 {
		seed = product + d.deviceKey[i:]
	}

	hash := md5.Sum([]byte(seed))
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

	d.fanMutex.Lock()
	defer d.fanMutex.Unlock()
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
	var devices = make(map[int]*Devices)

	// PSU Object
	device := &Devices{
		ChannelId: m,
		DeviceId:  fmt.Sprintf("%s-%v", "Psu", m),
		Name:      d.Product,
		Rpm:       0,
		Label:     "PSU",
		MainPSU:   true,
		HasSpeed:  false,
	}
	devices[m] = device
	m++

	// Fan
	device = &Devices{
		ChannelId: m,
		DeviceId:  fmt.Sprintf("%s-%v", "Fan", m),
		Name:      fmt.Sprintf("Fan %d", 1),
		Rpm:       0,
		HasSpeed:  true,
		Label:     "PSU Fan",
	}
	devices[m] = device
	m++

	// Temperature probes
	for i := 0; i < temperatureChannels; i++ {
		switch i {
		case 0:
			device = &Devices{
				ChannelId:          m,
				DeviceId:           fmt.Sprintf("%s-%v", "Probe", i),
				Name:               "VRM Temperature",
				Temperature:        0,
				Description:        "Probe",
				HasTemps:           true,
				IsTemperatureProbe: true,
				Label:              "Probe",
			}
		case 1:
			device = &Devices{
				ChannelId:          m,
				DeviceId:           fmt.Sprintf("%s-%v", "Probe", i),
				Name:               "PSU Temperature",
				Temperature:        0,
				Description:        "Probe",
				HasTemps:           true,
				IsTemperatureProbe: true,
				Label:              "Probe",
			}
		}
		devices[m] = device
		m++
	}

	// Power Out
	device = &Devices{
		ChannelId:    m,
		DeviceId:     fmt.Sprintf("%s-%v", "PowerOut", m),
		Name:         "Power Out",
		Description:  "Output Power",
		IsPowerProbe: true,
		Label:        "Output Power",
		Output:       true,
		IsPSU:        true,
	}
	devices[m] = device
	m++

	// 12V Current
	device = &Devices{
		ChannelId:    m,
		DeviceId:     fmt.Sprintf("%s-%v", "12vrail", m),
		Name:         "12V Rail",
		Description:  "Output Power",
		IsPowerProbe: true,
		Label:        "12V Rail Stats",
		Rail:         true,
		IsPSU:        true,
	}
	devices[m] = device
	m++

	// 5V Current
	device = &Devices{
		ChannelId:    m,
		DeviceId:     fmt.Sprintf("%s-%v", "5vrail", m),
		Name:         "5V Rail",
		Description:  "Output Power",
		IsPowerProbe: true,
		Label:        "5V Rail Stats",
		Rail:         true,
		IsPSU:        true,
	}
	devices[m] = device
	m++

	// 3V Current
	device = &Devices{
		ChannelId:    m,
		DeviceId:     fmt.Sprintf("%s-%v", "3vrail", m),
		Name:         "3V Rail",
		Description:  "Output Power",
		IsPowerProbe: true,
		Label:        "3V Rail Stats",
		Rail:         true,
		IsPSU:        true,
	}
	devices[m] = device
	m++

	d.Devices = devices
	return len(devices)
}

// getDeviceData will fetch device data, from temperatures, fan speed, etc...
// claimPort makes d the live Device on its USB port. Hotplug re-adds a PSU as a new Device
// but its unplug never reaches the old one (StopDirty looks it up by USB serial, which a PSU
// lacks), so the old one would reopen the port too and poll the PSU alongside the new one.
func (d *Device) claimPort() {
	if d.usbPath == "" {
		return
	}
	hidMutex.Lock()
	old := byPort[d.usbPath]
	byPort[d.usbPath] = d
	hidMutex.Unlock()
	if old == nil || old == d {
		return
	}
	old.stopAutoRefresh() // outside hidMutex: its refresh may be waiting on it in reopen
	old.mutex.Lock()
	_ = old.closeDev()
	old.lost = true
	old.mutex.Unlock()
}

func (d *Device) getDeviceData() {
	if d.isLost() && !d.reopen() {
		metrics.Remove(d.Serial) // Gone: no series rather than 0 W
		return
	}
	m := 0

	// Main PSU
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
				switch i {
				case 0:
					d.Devices[m].VrmTemperature = temp
					d.Devices[m].VrmTemperatureString = dashboard.GetDashboard().TemperatureToString(temp)
				case 1:
					d.Devices[m].PsuTemperature = temp
					d.Devices[m].PsuTemperatureString = dashboard.GetDashboard().TemperatureToString(temp)
				}
			}
		}
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
	powerOutWatts = common.FromLinear11(output)
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

	if d.isLost() {
		metrics.Remove(d.Serial) // Died mid-refresh: these readings are zeroed buffers
		return
	}
	d.updateDeviceMetrics()
}

// isLost reports whether the HID handle died and the PSU must be reopened
func (d *Device) isLost() bool {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	return d.lost
}

// reopen opens the PSU again once it is back on its USB port. A PSU that drops off USB comes
// back on a new hidraw node; without this its old handle read 0 until a restart, and the
// hotplug monitor can't help where no uevents arrive (e.g. a container).
func (d *Device) reopen() bool {
	if d.usbPath == "" {
		return false // no port: an identical PSU still present would be taken for this one
	}
	hidMutex.Lock()
	if byPort[d.usbPath] != d { // retired: another Device owns this port now
		hidMutex.Unlock()
		return false
	}
	var paths []string
	_ = hid.Enumerate(d.vendorId, d.productId, func(info *hid.DeviceInfo) error {
		usbPath, _ := common.GetShortUSBDevPath(filepath.Base(info.Path))
		if usbPath == d.usbPath {
			paths = append(paths, info.Path)
		}
		return nil
	})
	var dev *hid.Device
	var err error
	if len(paths) == 1 {
		dev, err = hid.OpenPath(paths[0])
	}
	hidMutex.Unlock()
	if dev == nil || err != nil {
		return false
	}

	d.mutex.Lock()
	_ = d.closeDev()
	d.dev = dev
	d.closed = false
	d.lost = false
	d.mutex.Unlock()
	logger.Log(logger.Fields{"serial": d.Serial, "path": paths[0]}).Info("PSU reconnected, HID device reopened")
	d.fanMutex.Lock()
	d.updateFanMode() // The PSU came back in its default fan mode
	d.fanMutex.Unlock()
	return true
}

// updateDeviceMetrics publishes every channel's latest readings to the metrics endpoint
func (d *Device) updateDeviceMetrics() {
	for _, device := range d.Devices {
		header := &metrics.Header{
			Product:     d.Product,
			Serial:      d.Serial,
			ChannelId:   strconv.Itoa(device.ChannelId),
			Name:        device.Name,
			Watts:       float64(device.Watts),
			Volts:       float64(device.Volts),
			Amps:        float64(device.Amps),
			Temperature: float64(device.Temperature),
			Rpm:         device.Rpm,
			HasWatts:    device.HasWatts,
			HasVolts:    device.HasVolts,
			HasAmps:     device.HasAmps,
			HasTemps:    device.HasTemps,
			HasSpeed:    device.HasSpeed,
			IsPSU:       true,
			MainPSU:     device.MainPSU,
		}
		metrics.Populate(header)
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

	if d.DeviceProfile == nil {
		deviceProfile.Active = true
		d.DeviceProfile = deviceProfile
	} else {
		deviceProfile.Active = d.DeviceProfile.Active
		deviceProfile.FanMode = d.DeviceProfile.FanMode
		deviceProfile.RgbOff = d.DeviceProfile.RgbOff
	}

	// Fix profile paths if folder database/ folder is moved
	filename := filepath.Base(deviceProfile.Path)
	path := fmt.Sprintf("%s/database/profiles/%s", pwd, filename)
	if deviceProfile.Path != path {
		logger.Log(logger.Fields{"original": deviceProfile.Path, "new": path}).Warn("Detected mismatching device profile path. Fixing paths...")
		deviceProfile.Path = path
	}

	// Save profile
	if err := common.SaveJsonData(deviceProfile.Path, deviceProfile); err != nil {
		logger.Log(logger.Fields{"error": err, "location": deviceProfile.Path}).Error("Unable to write device profile data")
		return
	}

	d.loadDeviceProfiles()
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
		if !common.AlphanumericDashRegex.MatchString(fileName) {
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
	d.refreshMutex.Lock()
	defer d.refreshMutex.Unlock()
	if d.refreshStopped {
		return
	}
	d.timer = time.NewTicker(time.Duration(deviceRefreshInterval) * time.Millisecond)
	d.autoRefreshChan = make(chan bool)
	d.refreshDone = make(chan struct{})
	go func(timer *time.Ticker, stop chan bool, done chan struct{}) {
		defer close(done)
		for {
			select {
			case <-timer.C:
				d.getDeviceData()
			case <-stop:
				timer.Stop()
				return
			}
		}
	}(d.timer, d.autoRefreshChan, d.refreshDone)
}

// stopAutoRefresh stops this device's refresh loop, if it was started
func (d *Device) stopAutoRefresh() {
	d.refreshMutex.Lock()
	timer, stop, done := d.timer, d.autoRefreshChan, d.refreshDone
	d.timer, d.autoRefreshChan = nil, nil
	d.refreshStopped = true
	d.refreshMutex.Unlock()
	if timer != nil {
		timer.Stop()
		close(stop) // not a send: a second caller would block forever once the loop has gone
	}
	if done != nil {
		<-done // every caller returns only once the loop is no longer using the handle
	}
}

// transfer will send data to a device and retrieve device output
func (d *Device) transfer(buffer []byte) ([]byte, error) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	bufferR := make([]byte, readBufferSize)
	if d.lost {
		return bufferR, errors.New("HID device lost")
	}

	if _, err := d.dev.Write(buffer); err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to a device")
		d.lost = true
		return bufferR, err
	}

	deadline := time.Now().Add(readTimeout)
	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			err := errors.New("no matching response before the read timeout")
			logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to read data from device")
			return make([]byte, 64), err
		}

		n, err := d.dev.ReadWithTimeout(bufferR, remaining)
		if err != nil {
			if strings.Contains(err.Error(), "Interrupted system call") {
				continue
			}
			logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to read data from device")
			d.lost = !errors.Is(err, hid.ErrTimeout) // a timeout is an unanswered request, not a dead handle
			return make([]byte, 64), err
		}
		if n == 0 {
			continue
		}

		if buffer[1] == bufferR[0] && buffer[2] == bufferR[1] {
			return bufferR, nil
		}
	}
}
