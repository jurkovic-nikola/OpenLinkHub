package katarproW

// Package: CORSAIR KATAR PRO WIRELESS Gaming Mouse.
// This is the primary package for CORSAIR KATAR PRO WIRELESS Gaming Mouse.
// All device actions are controlled from this package.
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/inputmanager"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/macro"
	"OpenLinkHub/src/rgb"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/sstallion/go-hid"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// DeviceProfile struct contains all device profile
type DeviceProfile struct {
	Active             bool
	Path               string
	Product            string
	Serial             string
	Brightness         uint8
	RGBProfile         string
	BrightnessSlider   *uint8
	Label              string
	Profile            int
	PollingRate        int
	Profiles           map[int]DPIProfile
	SleepMode          int
	ButtonOptimization int
}

type DPIProfile struct {
	Name        string `json:"name"`
	Value       uint16
	PackerIndex int
	ColorIndex  map[int][]int
	Color       *rgb.Color
}

type Device struct {
	Debug                 bool
	dev                   *hid.Device
	listener              *hid.Device
	Manufacturer          string `json:"manufacturer"`
	Product               string `json:"product"`
	Serial                string `json:"serial"`
	Firmware              string `json:"firmware"`
	activeRgb             *rgb.ActiveRGB
	UserProfiles          map[string]*DeviceProfile `json:"userProfiles"`
	Devices               map[int]string            `json:"devices"`
	DeviceProfile         *DeviceProfile
	OriginalProfile       *DeviceProfile
	Template              string
	VendorId              uint16
	ProductId             uint16
	SlipstreamId          uint16
	Brightness            map[int]string
	PollingRates          map[int]string
	SwitchModes           map[int]string
	KeyAssignmentTypes    map[int]string
	LEDChannels           int
	ChangeableLedChannels int
	CpuTemp               float32
	GpuTemp               float32
	Layouts               []string
	Rgb                   *rgb.RGB
	Endpoint              byte
	SleepModes            map[int]string
	Connected             bool
	mutex                 sync.Mutex
	timerKeepAlive        *time.Ticker
	timerSleep            *time.Ticker
	keepAliveChan         chan struct{}
	sleepChan             chan struct{}
	Exit                  bool
	KeyAssignment         map[int]inputmanager.KeyAssignment
	InputActions          map[uint8]inputmanager.InputAction
	PressLoop             bool
	keyAssignmentFile     string
	BatteryLevel          uint16
}

var (
	pwd                       = ""
	cmdSoftwareMode           = []byte{0x01, 0x03, 0x00, 0x02}
	cmdHardwareMode           = []byte{0x01, 0x03, 0x00, 0x01}
	cmdGetFirmware            = []byte{0x02, 0x13}
	cmdWriteColor             = []byte{0x06, 0x00}
	cmdOpenEndpoint           = []byte{0x0d, 0x00, 0x01}
	cmdOpenSleepWriteEndpoint = []byte{0x01, 0x0d, 0x00, 0x01}
	cmdHeartbeat              = []byte{0x12}
	cmdDongle                 = byte(0x08)
	cmdMouse                  = byte(0x09)
	cmdSetDpi                 = map[int][]byte{0: {0x01, 0x20, 0x00}}
	cmdSleep                  = map[int][]byte{0: {0x01, 0x37, 0x00}, 1: {0x01, 0x0e, 0x00}}
	cmdSetPollingRate         = []byte{0x01, 0x01, 0x00}
	cmdButtonOptimization     = []byte{0x01, 0xb0, 0x00}
	cmdOpenWriteEndpoint      = []byte{0x0d, 0x01, 0x02}
	cmdCloseEndpoint          = []byte{0x05, 0x01, 0x01}
	cmdWrite                  = []byte{0x06, 0x01}
	cmdBatteryLevel           = []byte{0x02, 0x0f}
	bufferSize                = 64
	bufferSizeWrite           = bufferSize + 1
	headerSize                = 2
	headerWriteSize           = 4
	keyAmount                 = 6
	minDpiValue               = 200
	maxDpiValue               = 10000
	deviceKeepAlive           = 10000
	transferTimeout           = 1000
)

func Init(vendorId, productId uint16, key string) *Device {
	// Set global working directory
	pwd = config.GetConfig().ConfigPath

	// Open device, return if failure
	dev, err := hid.OpenPath(key)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": vendorId, "productId": productId}).Error("Unable to open HID device")
		return nil
	}

	// Init new struct with HID device
	d := &Device{
		dev:            dev,
		Template:       "katarproW.html",
		Firmware:       "",
		VendorId:       vendorId,
		ProductId:      productId,
		sleepChan:      make(chan struct{}),
		keepAliveChan:  make(chan struct{}),
		timerSleep:     &time.Ticker{},
		timerKeepAlive: &time.Ticker{},
		Brightness: map[int]string{
			0: "RGB Profile",
			1: "33 %",
			2: "66 %",
			3: "100 %",
		},
		PollingRates: map[int]string{
			0: "Not Set",
			1: "125 Hz / 8 msec",
			2: "250 Hu / 4 msec",
			3: "500 Hz / 2 msec",
			4: "1000 Hz / 1 msec",
			5: "2000 Hz / 0.5 msec",
		},
		Product: "KATAR PRO WIRELESS",
		SleepModes: map[int]string{
			1:  "1 minute",
			5:  "5 minutes",
			15: "15 minutes",
		},
		LEDChannels:           1,
		ChangeableLedChannels: 0,
		SwitchModes: map[int]string{
			0: "Disabled",
			1: "Enabled",
		},
		KeyAssignmentTypes: map[int]string{
			0:  "None",
			1:  "Media Keys",
			2:  "DPI",
			3:  "Keyboard",
			10: "Macro",
		},
		InputActions:      inputmanager.GetInputActions(),
		keyAssignmentFile: "/database/key-assignments/katarpro.json",
	}

	d.getDebugMode()          // Debug
	d.getManufacturer()       // Manufacturer
	d.getSerial()             // Serial
	d.setDongleSoftwareMode() // Switch to software mode
	d.loadDeviceProfiles()    // Load all device profiles
	d.saveDeviceProfile()     // Save profile
	d.setKeepAlive()          // Keepalive
	d.backendListener()       // Control listener
	d.loadKeyAssignments()    // Key Assignments
	d.checkIfAlive()          // Initial setup
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device successfully initialized")
	return d
}

// GetRgbProfiles will return RGB profiles for a target device
func (d *Device) GetRgbProfiles() interface{} {
	return d.Rgb
}

// Stop will stop all device operations and switch a device back to hardware mode
func (d *Device) Stop() {
	d.Exit = true
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Stopping device...")

	d.timerSleep.Stop()
	d.timerKeepAlive.Stop()
	var once sync.Once
	go func() {
		once.Do(func() {
			if d.keepAliveChan != nil {
				close(d.keepAliveChan)
			}
			if d.sleepChan != nil {
				close(d.sleepChan)
			}
		})
	}()

	d.setHardwareMode()
	if d.dev != nil {
		err := d.dev.Close()
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to close HID device")
		}
	}
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device stopped")
}

func (d *Device) checkIfAlive() {
	msg, err := d.transferToDevice(cmdMouse, cmdHeartbeat, nil, "checkIfAlive")
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Warn("Unable to perform initial mouse init. Device is either offline or in sleep mode")
		return
	}

	if len(msg) > 0 && msg[1] == 0x12 {
		d.setupMouse(true)
		d.setSleepTimer()
	}
}

// GetDeviceTemplate will return device template name
func (d *Device) GetDeviceTemplate() string {
	return d.Template
}

// getManufacturer will return device manufacturer
func (d *Device) getDebugMode() {
	d.Debug = config.GetConfig().Debug
}

// getSerial will return device serial number
func (d *Device) getSerial() {
	serial, err := d.dev.GetSerialNbr()
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to get device serial number")
	}
	d.Serial = serial
}

// getManufacturer will return device manufacturer
func (d *Device) getManufacturer() {
	manufacturer, err := d.dev.GetMfrStr()
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to get manufacturer")
	}
	d.Manufacturer = manufacturer
}

// setHardwareMode will switch a device to hardware mode
func (d *Device) setHardwareMode() {
	if d.Connected {
		_, err := d.transfer(cmdMouse, cmdHardwareMode, nil, "setHardwareMode")
		if err != nil {
			logger.Log(logger.Fields{"error": err, "caller": "setHardwareMode"}).Error("Unable to change mouse device mode")
		}
	}

	_, err := d.transfer(cmdDongle, cmdHardwareMode, nil, "setHardwareMode")
	if err != nil {
		logger.Log(logger.Fields{"error": err, "caller": "setHardwareMode"}).Error("Unable to change dongle device mode")
	}
}

func (d *Device) setMouseHardwareMode() {
	_, err := d.transfer(cmdMouse, cmdHardwareMode, nil, "setMouseHardwareMode")
	if err != nil {
		logger.Log(logger.Fields{"error": err, "caller": "setMouseHardwareMode"}).Error("Unable to change mouse device mode")
	}
}

// setDongleSoftwareMode will switch a device to software mode
func (d *Device) setDongleSoftwareMode() {
	_, err := d.transfer(cmdDongle, cmdSoftwareMode, nil, "setDongleSoftwareMode")
	if err != nil {
		logger.Log(logger.Fields{"error": err, "caller": "setDongleSoftwareMode"}).Error("Unable to change dongle device mode")
	}
}

// getBatterLevel will return initial battery level
func (d *Device) getBatterLevel() {
	batteryLevel, err := d.transfer(cmdMouse, cmdBatteryLevel, nil, "getBatterLevel")
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to get battery level")
	}
	d.BatteryLevel = binary.LittleEndian.Uint16(batteryLevel[3:5]) / 10
}

// setSoftwareMode will switch a device to software mode
func (d *Device) setSoftwareMode() {
	_, err := d.transfer(cmdMouse, cmdSoftwareMode, nil, "setSoftwareMode")
	if err != nil {
		logger.Log(logger.Fields{"error": err, "caller": "setSoftwareMode"}).Error("Unable to change mouse device mode")
	}
}

// getDeviceFirmware will return a device firmware version out as string
func (d *Device) getDeviceFirmware() {
	fw, err := d.transfer(cmdMouse, cmdGetFirmware, nil, "getDeviceFirmware")
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to write to a device")
	}

	v1, v2, v3 := int(fw[3]), int(fw[4]), int(binary.LittleEndian.Uint16(fw[5:7]))
	d.Firmware = fmt.Sprintf("%d.%d.%d", v1, v2, v3)
}

// setButtonOptimization will change Button Response Optimization mode
func (d *Device) setButtonOptimization() {
	if d.DeviceProfile == nil {
		return
	}

	if d.DeviceProfile.ButtonOptimization < 0 || d.DeviceProfile.ButtonOptimization > 1 {
		return
	}

	buf := make([]byte, 1)
	buf[0] = byte(d.DeviceProfile.ButtonOptimization)
	_, _ = d.transfer(cmdMouse, cmdButtonOptimization, buf, "setButtonOptimization")
}

// initLeds will initialize LED endpoint
func (d *Device) initLeds() {
	_, err := d.transfer(cmdMouse, cmdOpenEndpoint, nil, "initLeds")
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to init led endpoint")
	}
}

// toggleExit will change Exit value
func (d *Device) toggleExit() {
	if d.Exit {
		d.Exit = false
	}
}

// Close will close all timers and channels before restart
func (d *Device) Close() {
	d.Exit = true
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Stopping device...")
	if d.activeRgb != nil {
		d.activeRgb.Exit <- true // Exit current RGB mode
		d.activeRgb = nil
	}
	d.Connected = false
	time.Sleep(500 * time.Millisecond)
}

// Restart will re-init device
func (d *Device) Restart() {
	if d.dev != nil {
		err := d.dev.Close()
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to close HID device")
		}
	}
	d.dev = nil

	interfaceId := 1
	path := ""
	enum := hid.EnumFunc(func(info *hid.DeviceInfo) error {
		if info.InterfaceNbr == interfaceId {
			path = info.Path
		}
		return nil
	})
	err := hid.Enumerate(d.VendorId, d.ProductId, enum)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": d.VendorId}).Fatal("Unable to enumerate devices")
	}

	dev, err := hid.OpenPath(path)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": d.VendorId, "productId": d.ProductId, "caller": "Restart()"}).Error("Unable to open HID device")
		return
	}

	d.dev = dev
	d.setDongleSoftwareMode() // Switch to software mode

	msg, err := d.transferToDevice(cmdMouse, cmdHeartbeat, nil, "checkIfAlive")
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Warn("Unable to perform initial mouse init. Device is either offline or in sleep mode")
		return
	}

	if len(msg) > 0 && msg[1] == 0x12 {
		d.Connected = true       // Mark as connected
		d.setMouseHardwareMode() // Hardware mode
		d.setSoftwareMode()      // Switch to software mode
		d.getDeviceFirmware()    // Firmware
		d.initLeds()             // Init LED ports
		d.toggleExit()           // Remove Exit flag
		d.toggleDPI()            // DPI
		d.setSleepTimer()        // Sleep timer
		d.backendListener()      // Control listener
		d.setupKeyAssignment()   // Setup key assignments
	}
}

// UpdatePollingRate will set device polling rate
func (d *Device) UpdatePollingRate(pullingRate int) uint8 {
	if !d.Connected {
		return 0
	}

	if _, ok := d.PollingRates[pullingRate]; ok {
		// Check if the mouse is alive and connected. If mouse is not alive, block any change.
		msg, err := d.transferToDevice(cmdMouse, cmdHeartbeat, nil, "checkIfAlive")
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Warn("Unable to perform initial mouse check. Device is either offline or in sleep mode")
			return 0
		}
		if len(msg) > 0 && msg[1] == 0x12 {
			// Mouse has to be connected, since polling change is done on dongle and mouse.
			// Changing the polling rate either on dongle or mouse only will break the connection.
			if d.DeviceProfile == nil {
				return 0
			}

			d.DeviceProfile.PollingRate = pullingRate
			d.saveDeviceProfile()

			d.Close()
			buf := make([]byte, 1)
			buf[0] = byte(pullingRate)
			_, err = d.transfer(cmdMouse, cmdSetPollingRate, buf, "UpdatePollingRate")
			if err != nil {
				logger.Log(logger.Fields{"error": err, "vendorId": d.VendorId}).Error("Unable to set mouse polling rate")
				return 0
			}
			_, err = d.transfer(cmdDongle, cmdSetPollingRate, buf, "UpdatePollingRate")
			if err != nil {
				logger.Log(logger.Fields{"error": err, "vendorId": d.VendorId}).Error("Unable to set mouse dongle polling rate")
				return 0
			}

			time.Sleep(5000 * time.Millisecond)
			d.Restart()
			return 1
		}
	}
	return 0
}

// ChangeDeviceProfile will change device profile
func (d *Device) ChangeDeviceProfile(profileName string) uint8 {
	if !d.Connected {
		return 0
	}

	if profile, ok := d.UserProfiles[profileName]; ok {
		currentProfile := d.DeviceProfile
		currentProfile.Active = false
		d.DeviceProfile = currentProfile
		d.saveDeviceProfile()

		// RGB reset
		if d.activeRgb != nil {
			d.activeRgb.Exit <- true // Exit current RGB mode
			d.activeRgb = nil
		}

		newProfile := profile
		newProfile.Active = true
		d.DeviceProfile = newProfile
		d.saveDeviceProfile()
		d.setDeviceColor()
		return 1
	}
	return 0
}

// setDeviceColor will activate and set device RGB
func (d *Device) setDeviceColor() {
	buf := make([]byte, d.LEDChannels*3)

	if d.DeviceProfile == nil {
		logger.Log(logger.Fields{"serial": d.Serial}).Error("Unable to set color. DeviceProfile is null!")
		return
	}

	// DPI
	dpiColor := d.DeviceProfile.Profiles[d.DeviceProfile.Profile].Color
	dpiColor.Brightness = rgb.GetBrightnessValueFloat(*d.DeviceProfile.BrightnessSlider)
	dpiColor = rgb.ModifyBrightness(*dpiColor)

	dpiLeds := d.DeviceProfile.Profiles[d.DeviceProfile.Profile]
	for i := 0; i < len(dpiLeds.ColorIndex); i++ {
		dpiColorIndexRange := dpiLeds.ColorIndex[i]
		for key, dpiColorIndex := range dpiColorIndexRange {
			switch key {
			case 0: // Red
				buf[dpiColorIndex] = byte(dpiColor.Red)
			case 1: // Green
				buf[dpiColorIndex] = byte(dpiColor.Green)
			case 2: // Blue
				buf[dpiColorIndex] = byte(dpiColor.Blue)
			}
		}
	}
	d.writeColor(buf)

	time.Sleep(1000 * time.Millisecond)
	for i := 0; i < len(dpiLeds.ColorIndex); i++ {
		dpiColorIndexRange := dpiLeds.ColorIndex[i]
		for key, dpiColorIndex := range dpiColorIndexRange {
			switch key {
			case 0: // Red
				buf[dpiColorIndex] = byte(0)
			case 1: // Green
				buf[dpiColorIndex] = byte(0)
			case 2: // Blue
				buf[dpiColorIndex] = byte(0)
			}
		}
	}
	d.writeColor(buf)
	return
}

// setupKeyAssignment will setup mouse keys
func (d *Device) setupKeyAssignment() {
	// Prevent modifications if key amount does not match the expected key amount
	definedKeyAmount := len(d.KeyAssignment)
	if definedKeyAmount < keyAmount || definedKeyAmount > keyAmount {
		logger.Log(logger.Fields{"vendorId": d.VendorId, "keys": definedKeyAmount, "expected": keyAmount}).Warn("Expected key amount does not match the expected key amount.")
		return
	}

	keys := make([]int, 0)
	for k := range d.KeyAssignment {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	buf := make([]byte, keyAmount)
	i := 0
	for _, k := range keys {
		value := d.KeyAssignment[k]
		if value.Default {
			buf[i] = byte(1)
		} else {
			buf[i] = byte(0)
		}
		i++
	}
	d.writeKeyAssignmentData(buf)
}

// triggerKeyAssignment will trigger key assignment if defined
func (d *Device) triggerKeyAssignment(value byte) {
	if value == 0 {
		d.PressLoop = false
	}

	if val, ok := d.KeyAssignment[int(value)]; ok {
		if value == 0x20 && val.Default {
			d.ModifyDpi()
			return
		}

		if val.Default {
			return
		}

		if val.ActionHold {
			d.PressLoop = val.ActionHold
			go func() {
				for {
					if !d.PressLoop {
						return
					}
					switch val.ActionType {
					case 1, 3:
						inputmanager.InputControl(val.ActionCommand, d.Serial)
						break
					case 2:
						d.ModifyDpi()
						break
					case 10: // Macro
						macroProfile := macro.GetProfile(int(val.ActionCommand))
						if macroProfile == nil {
							logger.Log(logger.Fields{"serial": d.Serial}).Error("Invalid macro profile")
							return
						}
						for i := 0; i < len(macroProfile.Actions); i++ {
							if v, valid := macroProfile.Actions[i]; valid {
								switch v.ActionType {
								case 1, 3:
									inputmanager.InputControl(v.ActionCommand, d.Serial)
									break
								case 5:
									if v.ActionDelay > 0 {
										time.Sleep(time.Duration(v.ActionDelay) * time.Millisecond)
									}
									break
								}
							}
						}
						break
					}
					time.Sleep(20 * time.Millisecond)
				}
			}()
		} else {
			switch val.ActionType {
			case 1, 3:
				inputmanager.InputControl(val.ActionCommand, d.Serial)
				break
			case 2:
				d.ModifyDpi()
				break
			case 10: // Macro
				macroProfile := macro.GetProfile(int(val.ActionCommand))
				if macroProfile == nil {
					logger.Log(logger.Fields{"serial": d.Serial}).Error("Invalid macro profile")
					return
				}
				for i := 0; i < len(macroProfile.Actions); i++ {
					if v, valid := macroProfile.Actions[i]; valid {
						switch v.ActionType {
						case 1, 3:
							inputmanager.InputControl(v.ActionCommand, d.Serial)
							break
						case 5:
							if v.ActionDelay > 0 {
								time.Sleep(time.Duration(v.ActionDelay) * time.Millisecond)
							}
							break
						}
					}
				}
				break
			}
		}
	}
}

// writeColor will write data to the device with a specific endpoint.
func (d *Device) writeColor(data []byte) {
	if d.Exit {
		return
	}
	buffer := make([]byte, len(data)+headerWriteSize)
	binary.LittleEndian.PutUint16(buffer[0:2], uint16(len(data)))
	copy(buffer[headerWriteSize:], data)

	_, err := d.transfer(cmdMouse, cmdWriteColor, buffer, "writeColor")
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to color endpoint")
	}
}

// toggleDPI will change DPI mode
func (d *Device) toggleDPI() {
	if d.Exit {
		return
	}
	if d.DeviceProfile != nil {
		profile := d.DeviceProfile.Profiles[d.DeviceProfile.Profile]
		value := profile.Value

		// Send DPI packet
		if value < uint16(minDpiValue) {
			value = uint16(minDpiValue)
		}
		if value > uint16(maxDpiValue) {
			value = uint16(maxDpiValue)
		}

		buf := make([]byte, 2)
		binary.LittleEndian.PutUint16(buf[0:2], value)
		for i := 0; i <= 1; i++ {
			_, err := d.transfer(cmdMouse, cmdSetDpi[i], buf, "toggleDPI")
			if err != nil {
				logger.Log(logger.Fields{"error": err, "vendorId": d.VendorId}).Error("Unable to set dpi")
			}
		}

		if d.activeRgb != nil {
			d.activeRgb.Exit <- true // Exit current RGB mode
			d.activeRgb = nil
		}
		d.setDeviceColor() // Restart RGB
	}
}

func (d *Device) ModifyDpi() {
	if d.Exit || !d.Connected {
		return
	}

	if d.DeviceProfile.Profile >= 2 {
		d.DeviceProfile.Profile = 0
	} else {
		d.DeviceProfile.Profile++
	}
	d.saveDeviceProfile()
	d.toggleDPI()
}

// saveDeviceProfile will save device profile for persistent configuration
func (d *Device) saveDeviceProfile() {
	var defaultBrightness = uint8(100)
	profilePath := pwd + "/database/profiles/" + d.Serial + ".json"

	deviceProfile := &DeviceProfile{
		Product:          d.Product,
		Serial:           d.Serial,
		Path:             profilePath,
		BrightnessSlider: &defaultBrightness,
	}

	// First save, assign saved profile to a device
	if d.DeviceProfile == nil {
		// RGB, Label
		deviceProfile.RGBProfile = "mouse"
		deviceProfile.Label = "Mouse"
		deviceProfile.Active = true
		deviceProfile.Profiles = map[int]DPIProfile{
			0: {
				Name:        "Stage 1",
				Value:       400,
				PackerIndex: 1,
				ColorIndex: map[int][]int{
					0: {0, 1, 2},
				},
				Color: &rgb.Color{
					Red:        255,
					Green:      0,
					Blue:       0,
					Brightness: 1,
					Hex:        fmt.Sprintf("#%02x%02x%02x", 255, 0, 0),
				},
			},
			1: {
				Name:        "Stage 2",
				Value:       800,
				PackerIndex: 2,
				ColorIndex: map[int][]int{
					0: {0, 1, 2},
				},
				Color: &rgb.Color{
					Red:        255,
					Green:      255,
					Blue:       255,
					Brightness: 1,
					Hex:        fmt.Sprintf("#%02x%02x%02x", 255, 255, 255),
				},
			},
			2: {
				Name:        "Stage 3",
				Value:       1200,
				PackerIndex: 3,
				ColorIndex: map[int][]int{
					0: {0, 1, 2},
				},
				Color: &rgb.Color{
					Red:        0,
					Green:      255,
					Blue:       0,
					Brightness: 1,
					Hex:        fmt.Sprintf("#%02x%02x%02x", 0, 255, 0),
				},
			},
		}
		deviceProfile.Profile = 1
		deviceProfile.SleepMode = 5
		deviceProfile.PollingRate = 4
	} else {
		if d.DeviceProfile.BrightnessSlider == nil {
			deviceProfile.BrightnessSlider = &defaultBrightness
			d.DeviceProfile.BrightnessSlider = &defaultBrightness
		} else {
			deviceProfile.BrightnessSlider = d.DeviceProfile.BrightnessSlider
		}
		deviceProfile.Active = d.DeviceProfile.Active
		deviceProfile.Brightness = d.DeviceProfile.Brightness
		deviceProfile.RGBProfile = d.DeviceProfile.RGBProfile
		deviceProfile.Label = d.DeviceProfile.Label
		deviceProfile.Profiles = d.DeviceProfile.Profiles
		deviceProfile.Profile = d.DeviceProfile.Profile
		deviceProfile.SleepMode = d.DeviceProfile.SleepMode
		deviceProfile.PollingRate = d.DeviceProfile.PollingRate
		deviceProfile.ButtonOptimization = d.DeviceProfile.ButtonOptimization

		if len(d.DeviceProfile.Path) < 1 {
			deviceProfile.Path = profilePath
			d.DeviceProfile.Path = profilePath
		} else {
			deviceProfile.Path = d.DeviceProfile.Path
		}
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

// UpdateButtonOptimization will update button response optimization mode
func (d *Device) UpdateButtonOptimization(buttonOptimizationMode int) uint8 {
	if d.DeviceProfile == nil {
		return 0
	}

	if d.DeviceProfile.ButtonOptimization == buttonOptimizationMode {
		return 0
	}

	d.DeviceProfile.ButtonOptimization = buttonOptimizationMode
	d.saveDeviceProfile()
	d.setButtonOptimization()
	return 1
}

// UpdateDeviceKeyAssignment will update device key assignments
func (d *Device) UpdateDeviceKeyAssignment(keyIndex int, keyAssignment inputmanager.KeyAssignment) uint8 {
	if val, ok := d.KeyAssignment[keyIndex]; ok {
		val.Default = keyAssignment.Default
		val.ActionHold = keyAssignment.ActionHold
		val.ActionType = keyAssignment.ActionType
		val.ActionCommand = keyAssignment.ActionCommand
		val.IsMacro = keyAssignment.IsMacro
		d.KeyAssignment[keyIndex] = val
		d.saveKeyAssignments()
		d.setupKeyAssignment()
		return 1
	}
	return 0
}

func (d *Device) saveKeyAssignments() {
	keyAssignmentsFile := pwd + d.keyAssignmentFile
	if common.FileExists(keyAssignmentsFile) {

	}
	// Convert to JSON
	buffer, err := json.MarshalIndent(d.KeyAssignment, "", "    ")
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to convert to json format")
		return
	}

	// Create profile filename
	file, err := os.Create(keyAssignmentsFile)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": keyAssignmentsFile}).Error("Unable to create new device profile")
		return
	}

	// Write JSON buffer to file
	_, err = file.Write(buffer)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": keyAssignmentsFile}).Error("Unable to write data")
		return
	}

	// Close file
	err = file.Close()
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": keyAssignmentsFile}).Error("Unable to close file handle")
	}
}

// loadKeyAssignments will load custom key assignments
func (d *Device) loadKeyAssignments() {
	if d.DeviceProfile == nil {
		return
	}
	keyAssignmentsFile := pwd + d.keyAssignmentFile
	if common.FileExists(keyAssignmentsFile) {
		file, err := os.Open(keyAssignmentsFile)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial, "location": keyAssignmentsFile}).Warn("Unable to load JSON file")
			return
		}

		if err = json.NewDecoder(file).Decode(&d.KeyAssignment); err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial, "location": keyAssignmentsFile}).Warn("Unable to decode key assignments JSON")
			return
		}

		// Prevent left click modifications
		if !d.KeyAssignment[1].Default {
			logger.Log(logger.Fields{"serial": d.Serial, "value": d.KeyAssignment[1].Default, "expectedValue": 1}).Warn("Restoring left button to original value")
			var val = d.KeyAssignment[1]
			val.Default = true
			d.KeyAssignment[1] = val
		}

		err = file.Close()
		if err != nil {
			logger.Log(logger.Fields{"location": keyAssignmentsFile, "serial": d.Serial}).Warn("Failed to close file handle")
		}
	} else {
		var keyAssignment = map[int]inputmanager.KeyAssignment{
			32: {
				Name:          "DPI Button",
				Default:       true,
				ActionType:    0,
				ActionCommand: 0,
				ActionHold:    false,
			},
			16: {
				Name:          "Back Button",
				Default:       true,
				ActionType:    0,
				ActionCommand: 0,
				ActionHold:    false,
			},
			8: {
				Name:          "Forward Button",
				Default:       true,
				ActionType:    0,
				ActionCommand: 0,
				ActionHold:    false,
			},
			4: {
				Name:          "Middle Button",
				Default:       true,
				ActionType:    0,
				ActionCommand: 0,
				ActionHold:    false,
			},
			2: {
				Name:          "Right Button",
				Default:       true,
				ActionType:    0,
				ActionCommand: 0,
				ActionHold:    false,
			},
			1: {
				Name:          "Left Button",
				Default:       true,
				ActionType:    0,
				ActionCommand: 0,
				ActionHold:    false,
			},
		}

		// Convert to JSON
		buffer, err := json.MarshalIndent(keyAssignment, "", "    ")
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to convert to json format")
			return
		}

		file, err := os.Create(keyAssignmentsFile)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "location": keyAssignmentsFile}).Error("Unable to create new key assignment file")
			return
		}

		_, err = file.Write(buffer)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "location": keyAssignmentsFile}).Error("Unable to write data tp key assignment file")
			return
		}

		err = file.Close()
		if err != nil {
			logger.Log(logger.Fields{"error": err, "location": keyAssignmentsFile}).Error("Unable to close key assignment file")
		}
		d.KeyAssignment = keyAssignment
	}
}

// loadDeviceProfiles will load custom user profiles
func (d *Device) loadDeviceProfiles() {
	profileList := make(map[string]*DeviceProfile, 0)
	userProfileDirectory := pwd + "/database/profiles/"

	files, err := os.ReadDir(userProfileDirectory)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": userProfileDirectory, "serial": d.Serial}).Error("Unable to read content of a folder")
		return
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

// SaveMouseDpiColors will save mouse dpi colors
func (d *Device) SaveMouseDpiColors(dpi rgb.Color, dpiColors map[int]rgb.Color) uint8 {
	if !d.Connected {
		return 0
	}

	i := 0
	if d.DeviceProfile == nil {
		return 0
	}
	if dpi.Red > 255 ||
		dpi.Green > 255 ||
		dpi.Blue > 255 ||
		dpi.Red < 0 ||
		dpi.Green < 0 ||
		dpi.Blue < 0 {
		return 0
	}

	// Zone Colors
	for key, zone := range dpiColors {
		if zone.Red > 255 ||
			zone.Green > 255 ||
			zone.Blue > 255 ||
			zone.Red < 0 ||
			zone.Green < 0 ||
			zone.Blue < 0 {
			continue
		}
		if profileColor, ok := d.DeviceProfile.Profiles[key]; ok {
			profileColor.Color.Red = zone.Red
			profileColor.Color.Green = zone.Green
			profileColor.Color.Blue = zone.Blue
			profileColor.Color.Hex = fmt.Sprintf("#%02x%02x%02x", int(zone.Red), int(zone.Green), int(zone.Blue))
		}
		i++
	}

	if i > 0 {
		d.saveDeviceProfile()
		if d.activeRgb != nil {
			d.activeRgb.Exit <- true // Exit current RGB mode
			d.activeRgb = nil
		}
		d.setDeviceColor() // Restart RGB
		return 1
	}
	return 0
}

// SaveMouseDPI will save mouse DPI
func (d *Device) SaveMouseDPI(stages map[int]uint16) uint8 {
	if !d.Connected {
		return 0
	}

	i := 0
	if d.DeviceProfile == nil {
		return 0
	}

	if len(stages) == 0 {
		return 0
	}

	for key, stage := range stages {
		if _, ok := d.DeviceProfile.Profiles[key]; ok {
			profile := d.DeviceProfile.Profiles[key]
			if stage > uint16(maxDpiValue) {
				continue
			}
			if stage < uint16(minDpiValue) {
				continue
			}
			profile.Value = stage
			d.DeviceProfile.Profiles[key] = profile
			i++
		}
	}

	if i > 0 {
		d.saveDeviceProfile()
		d.toggleDPI()
		return 1
	}
	return 0
}

// UpdateSleepTimer will update device sleep timer
func (d *Device) UpdateSleepTimer(minutes int) uint8 {
	if !d.Connected {
		return 0
	}

	if d.DeviceProfile != nil {
		if minutes > 15 {
			return 0
		}
		d.DeviceProfile.SleepMode = minutes
		d.saveDeviceProfile()
		d.setSleepTimer()
		return 1
	}
	return 0
}

// setSleepTimer will set device sleep timer
func (d *Device) setSleepTimer() uint8 {
	if d.DeviceProfile != nil {
		if !d.Connected {
			return 0
		}

		_, err := d.transfer(cmdMouse, cmdOpenSleepWriteEndpoint, nil, "setSleepTimer")
		if err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Warn("Unable to change device sleep timer")
			return 0
		}

		buf := make([]byte, 4)
		for i := 0; i < 2; i++ {
			command := cmdSleep[i]
			if i == 0 {
				buf[0] = 0xa0
				buf[1] = 0xbb
				buf[2] = 0x0d
				buf[3] = 0x00
			} else {
				sleep := d.DeviceProfile.SleepMode * (60 * 1000)
				binary.LittleEndian.PutUint32(buf, uint32(sleep))
			}
			_, err = d.transfer(cmdMouse, command, buf, "setSleepTimer")
			if err != nil {
				logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Warn("Unable to change device sleep timer")
				continue
			}
		}
		return 1
	}
	return 0
}

// GetSleepMode will return current sleep mode
func (d *Device) GetSleepMode() int {
	if d.DeviceProfile != nil {
		return d.DeviceProfile.SleepMode
	}
	return 0
}

// setupMouse will perform mouse setup
func (d *Device) setupMouse(init bool) {
	if init {
		d.Connected = true       // Mark as connected
		d.setMouseHardwareMode() // Hardware mode
		d.setSoftwareMode()      // Switch to software mode
		d.getDeviceFirmware()    // Firmware
		d.getBatterLevel()       // Battery level
		d.initLeds()             // Init LED ports
		d.toggleDPI()            // DPI
		d.setSleepTimer()        // Sleep timer
		d.setupKeyAssignment()   // Setup key assignments
	} else {
		d.Connected = false
	}
}

// keepAlive will keep a device alive
func (d *Device) keepAlive() {
	if d.Exit {
		return
	}
	_, err := d.transfer(cmdDongle, cmdHeartbeat, nil, "keepAlive")
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to a device")
	}

	if d.Connected {
		_, err = d.transferToDevice(cmdMouse, cmdHeartbeat, nil, "keepAlive")
		if err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to a device")
		}
	}
}

// setKeepAlive will refresh device data
func (d *Device) setKeepAlive() {
	d.timerKeepAlive = time.NewTicker(time.Duration(deviceKeepAlive) * time.Millisecond)
	go func() {
		for {
			select {
			case <-d.timerKeepAlive.C:
				if d.Exit {
					return
				}
				d.keepAlive()
			case <-d.keepAliveChan:
				d.timerKeepAlive.Stop()
				return
			}
		}
	}()
}

// getListenerData will listen for keyboard events and return data on success or nil on failure.
// ReadWithTimeout is mandatory due to the nature of listening for events
func (d *Device) getListenerData() []byte {
	data := make([]byte, bufferSize)
	n, err := d.listener.ReadWithTimeout(data, 100*time.Millisecond)
	if err != nil || n == 0 {
		return nil
	}
	return data
}

// backendListener will listen for events from the device
func (d *Device) backendListener() {
	go func() {
		enum := hid.EnumFunc(func(info *hid.DeviceInfo) error {
			if info.InterfaceNbr == 2 {
				listener, err := hid.OpenPath(info.Path)
				if err != nil {
					return err
				}
				d.listener = listener
			}
			return nil
		})

		err := hid.Enumerate(d.VendorId, d.ProductId, enum)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "vendorId": d.VendorId}).Error("Unable to enumerate devices")
		}

		for {
			select {
			default:
				if d.Exit {
					err = d.listener.Close()
					if err != nil {
						logger.Log(logger.Fields{"error": err, "vendorId": d.VendorId}).Error("Failed to close listener")
						return
					}
					return
				}

				data := d.getListenerData()
				if len(data) == 0 || data == nil {
					continue
				}

				// Battery
				if data[2] == 0x0f {
					val := binary.LittleEndian.Uint16(data[4:6])
					if val > 0 {
						d.BatteryLevel = val / 10
					}
				}

				if data[1] == 0x01 && data[2] == 0x36 {
					value := data[4]
					switch value {
					case 0x02:
						{
							if d.Connected == false {
								// Mouse needs to initialize
								time.Sleep(time.Duration(transferTimeout) * time.Millisecond)

								// Setup mouse
								d.setupMouse(true)
							}
						}
					case 0x00:
						{
							// Turned off or sleep mode
							d.setupMouse(false)
						}
					}
				} else {
					switch data[0] {
					case 1, 2: // Mouse
						{
							if data[1] == 0x02 {
								d.triggerKeyAssignment(data[2])
							}
						}
					}
				}
			}
		}
	}()
}

// writeKeyAssignmentData will write key assignment to the device.
func (d *Device) writeKeyAssignmentData(data []byte) {
	if d.Exit {
		return
	}

	// Open endpoint
	_, err := d.transfer(cmdMouse, cmdOpenWriteEndpoint, nil, "writeKeyAssignmentData")
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": d.VendorId}).Error("Unable to open write endpoint")
		return
	}

	// Send data
	buffer := make([]byte, len(data)+headerWriteSize)
	binary.LittleEndian.PutUint16(buffer[0:2], uint16(len(data)))
	copy(buffer[headerWriteSize:], data)
	_, err = d.transfer(cmdMouse, cmdWrite, buffer, "writeKeyAssignmentData")
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to data endpoint")
	}

	// Close endpoint
	_, err = d.transfer(cmdMouse, cmdCloseEndpoint, nil, "writeKeyAssignmentData")
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": d.VendorId}).Error("Unable to close endpoint")
		return
	}
}

// transfer will send data to a device and retrieve device output
func (d *Device) transfer(command byte, endpoint, buffer []byte, caller string) ([]byte, error) {
	// Packet control, mandatory for this device
	d.mutex.Lock()
	defer d.mutex.Unlock()

	// Create write buffer
	bufferW := make([]byte, bufferSizeWrite)
	bufferW[1] = command
	endpointHeaderPosition := bufferW[headerSize : headerSize+len(endpoint)]
	copy(endpointHeaderPosition, endpoint)
	if len(buffer) > 0 {
		copy(bufferW[headerSize+len(endpoint):headerSize+len(endpoint)+len(buffer)], buffer)
	}

	reports := make([]byte, bufferSize)
	err := d.dev.SetNonblock(true)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to SetNonblock")
	}

	for {
		if d.Exit {
			break
		}

		n, err := d.dev.Read(reports)
		if err != nil {
			if n < 0 {
				//
			}
			if err == hid.ErrTimeout || n == 0 {
				break
			}
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	err = d.dev.SetNonblock(false)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to SetNonblock")
	}

	// Create read buffer
	bufferR := make([]byte, bufferSize)

	// Send command to a device
	if _, err := d.dev.Write(bufferW); err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to a device")
		return bufferR, err
	}

	// Get data from a device
	if _, err := d.dev.Read(bufferR); err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial, "caller": caller}).Error("Unable to read data from device")
		return bufferR, err
	}
	return bufferR, nil
}

// transfer will send data to a device and retrieve device output
func (d *Device) transferToDevice(command byte, endpoint, buffer []byte, caller string) ([]byte, error) {
	// Packet control, mandatory for this device
	d.mutex.Lock()
	defer d.mutex.Unlock()

	// Create write buffer
	bufferW := make([]byte, bufferSizeWrite)
	bufferW[1] = command
	endpointHeaderPosition := bufferW[headerSize : headerSize+len(endpoint)]
	copy(endpointHeaderPosition, endpoint)
	if len(buffer) > 0 {
		copy(bufferW[headerSize+len(endpoint):headerSize+len(endpoint)+len(buffer)], buffer)
	}

	// Create read buffer
	bufferR := make([]byte, bufferSize)

	// Send command to a device
	if _, err := d.dev.Write(bufferW); err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to a device")
		return bufferR, err
	}

	// Get data from a device
	if _, err := d.dev.ReadWithTimeout(bufferR, time.Duration(transferTimeout)*time.Millisecond); err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial, "caller": caller}).Error("Unable to read data from device")
		return bufferR, err
	}
	return bufferR, nil
}
