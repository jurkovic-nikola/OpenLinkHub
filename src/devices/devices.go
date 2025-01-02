package devices

import (
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/devices/cc"
	"OpenLinkHub/src/devices/ccxt"
	"OpenLinkHub/src/devices/cpro"
	"OpenLinkHub/src/devices/elite"
	"OpenLinkHub/src/devices/ironclaw"
	"OpenLinkHub/src/devices/ironclawW"
	"OpenLinkHub/src/devices/ironclawWU"
	"OpenLinkHub/src/devices/k100"
	"OpenLinkHub/src/devices/k100air"
	"OpenLinkHub/src/devices/k100airW"
	"OpenLinkHub/src/devices/k55core"
	"OpenLinkHub/src/devices/k65plus"
	"OpenLinkHub/src/devices/k65plusW"
	"OpenLinkHub/src/devices/k65pm"
	"OpenLinkHub/src/devices/k70core"
	"OpenLinkHub/src/devices/k70pro"
	"OpenLinkHub/src/devices/katarpro"
	"OpenLinkHub/src/devices/katarproW"
	"OpenLinkHub/src/devices/lncore"
	"OpenLinkHub/src/devices/lnpro"
	"OpenLinkHub/src/devices/lsh"
	"OpenLinkHub/src/devices/lt100"
	"OpenLinkHub/src/devices/m55"
	"OpenLinkHub/src/devices/m55W"
	"OpenLinkHub/src/devices/m55rgbpro"
	"OpenLinkHub/src/devices/memory"
	"OpenLinkHub/src/devices/mm700"
	"OpenLinkHub/src/devices/nightsabreW"
	"OpenLinkHub/src/devices/nightsabreWU"
	"OpenLinkHub/src/devices/psuhid"
	"OpenLinkHub/src/devices/scimitar"
	"OpenLinkHub/src/devices/scimitarW"
	"OpenLinkHub/src/devices/scimitarWU"
	"OpenLinkHub/src/devices/slipstream"
	"OpenLinkHub/src/devices/st100"
	"OpenLinkHub/src/devices/xc7"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/metrics"
	"OpenLinkHub/src/rgb"
	"OpenLinkHub/src/smbus"
	"github.com/sstallion/go-hid"
	"os"
	"reflect"
	"slices"
	"strconv"
)

const (
	productTypeLinkHub            = 0
	productTypeCC                 = 1
	productTypeCCXT               = 2
	productTypeElite              = 3
	productTypeLNCore             = 4
	productTypeLnPro              = 5
	productTypeCPro               = 6
	productTypeXC7                = 7
	productTypeMemory             = 8
	productTypeK65PM              = 101
	productTypeK70Core            = 102
	productTypeK55Core            = 103
	productTypeK70Pro             = 104
	productTypeK65Plus            = 105
	productTypeK65PlusW           = 106
	productTypeK100Air            = 107
	productTypeK100AirW           = 108
	productTypeK100               = 109
	productTypeKatarPro           = 201
	productTypeIronClawRgb        = 202
	productTypeIronClawRgbW       = 203
	productTypeIronClawRgbWU      = 204
	productTypeNightsabreW        = 205
	productTypeNightsabreWU       = 206
	productTypeScimitarRgbElite   = 207
	productTypeScimitarRgbEliteW  = 208
	productTypeScimitarRgbEliteWU = 209
	productTypeM55                = 210
	productTypeM55W               = 211
	productTypeM55RgbPro          = 212
	productTypeKatarProW          = 213
	productTypeST100              = 401
	productTypeMM700              = 402
	productTypeLT100              = 403
	productTypePSUHid             = 501
)

type AIOData struct {
	Rpm         int16
	Temperature float32
	Serial      string
}

type Device struct {
	ProductType uint16
	Product     string
	Serial      string
	Firmware    string
	Image       string
	GetDevice   interface{}
	Instance    interface{}
	Hidden      bool
}

type Product struct {
	ProductId uint16
	Path      string
}

var (
	expectedPermissions        = []int{0600, 0660}
	vendorId            uint16 = 6940 // Corsair
	interfaceId                = 0
	devices                    = make(map[string]*Device, 0)
	products                   = make(map[string]Product, 0)
	keyboards                  = []uint16{7127, 7165, 7166, 7110, 7083, 11024, 11015, 7109, 7091, 7036, 7037}
	mouses                     = []uint16{7059, 7005, 6988, 7096, 7139, 7131, 11011, 7024}
	pads                       = []uint16{7067}
	dongles                    = []uint16{7132, 7078, 11008, 7060}
)

// Stop will stop all active devices
func Stop() {
	for _, device := range devices {
		methodName := "Stop"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found")
			continue
		} else {
			method.Call(nil)
		}
		delete(devices, device.Serial)
	}
	err := hid.Exit()
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to exit HID interface")
	}
}

// GetDeviceTemplate will return device template
func GetDeviceTemplate(device interface{}) string {
	if device == nil {
		return "404.html"
	}
	methodName := "GetDeviceTemplate"
	method := reflect.ValueOf(device).MethodByName(methodName)
	if !method.IsValid() {
		logger.Log(logger.Fields{"method": methodName}).Warn("Method not found or method is not supported for this device type")
		return ""
	} else {
		results := method.Call(nil)
		if len(results) > 0 {
			val := results[0]
			return val.String()
		}
	}
	return ""
}

// UpdateMiscColor will process a POST request from a client for misc color change
func UpdateMiscColor(deviceId string, keyId, keyOptions int, color rgb.Color) uint8 {
	if device, ok := devices[deviceId]; ok {
		methodName := "UpdateDeviceColor"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found or method is not supported for this device type")
			return 0
		} else {
			var reflectArgs []reflect.Value
			reflectArgs = append(reflectArgs, reflect.ValueOf(keyId))
			reflectArgs = append(reflectArgs, reflect.ValueOf(keyOptions))
			reflectArgs = append(reflectArgs, reflect.ValueOf(color))
			results := method.Call(reflectArgs)
			if len(results) > 0 {
				val := results[0]
				uintResult := val.Uint()
				return uint8(uintResult)
			}
		}
	}
	return 0
}

// UpdateKeyboardColor will process POST request from a client for keyboard color change
func UpdateKeyboardColor(deviceId string, keyId, keyOptions int, color rgb.Color) uint8 {
	if device, ok := devices[deviceId]; ok {
		methodName := "UpdateDeviceColor"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found or method is not supported for this device type")
			return 0
		} else {
			var reflectArgs []reflect.Value
			reflectArgs = append(reflectArgs, reflect.ValueOf(keyId))
			reflectArgs = append(reflectArgs, reflect.ValueOf(keyOptions))
			reflectArgs = append(reflectArgs, reflect.ValueOf(color))
			results := method.Call(reflectArgs)
			if len(results) > 0 {
				val := results[0]
				uintResult := val.Uint()
				return uint8(uintResult)
			}
		}
	}
	return 0
}

// UpdateARGBDevice will process POST request from a client for ARGB 3-pin devices
func UpdateARGBDevice(deviceId string, portId, deviceType int) uint8 {
	if device, ok := devices[deviceId]; ok {
		methodName := "UpdateARGBDevice"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found or method is not supported for this device type")
			return 0
		} else {
			var reflectArgs []reflect.Value
			reflectArgs = append(reflectArgs, reflect.ValueOf(portId))
			reflectArgs = append(reflectArgs, reflect.ValueOf(deviceType))
			results := method.Call(reflectArgs)
			if len(results) > 0 {
				val := results[0]
				uintResult := val.Uint()
				return uint8(uintResult)
			}
		}
	}
	return 0
}

// UpdateExternalHubDeviceType will update a device type connected to an external-LED hub
func UpdateExternalHubDeviceType(deviceId string, portId, deviceType int) uint8 {
	if device, ok := devices[deviceId]; ok {
		methodName := "UpdateExternalHubDeviceType"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found or method is not supported for this device type")
			return 0
		} else {
			var reflectArgs []reflect.Value
			reflectArgs = append(reflectArgs, reflect.ValueOf(portId))
			reflectArgs = append(reflectArgs, reflect.ValueOf(deviceType))
			results := method.Call(reflectArgs)
			if len(results) > 0 {
				val := results[0]
				uintResult := val.Uint()
				return uint8(uintResult)
			}
		}
	}
	return 0
}

// UpdatePsuFanMode will update a device fan mode
func UpdatePsuFanMode(deviceId string, fanMode int) uint8 {
	if device, ok := devices[deviceId]; ok {
		methodName := "UpdatePsuFan"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found or method is not supported for this device type")
			return 0
		} else {
			var reflectArgs []reflect.Value
			reflectArgs = append(reflectArgs, reflect.ValueOf(fanMode))
			results := method.Call(reflectArgs)
			if len(results) > 0 {
				val := results[0]
				uintResult := val.Uint()
				return uint8(uintResult)
			}
		}
	}
	return 0
}

// SaveMouseDPI will save mouse DPI values
func SaveMouseDPI(deviceId string, stages map[int]uint16) uint8 {
	if device, ok := devices[deviceId]; ok {
		methodName := "SaveMouseDPI"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found or method is not supported for this device type")
			return 0
		} else {
			var reflectArgs []reflect.Value
			reflectArgs = append(reflectArgs, reflect.ValueOf(stages))
			results := method.Call(reflectArgs)
			if len(results) > 0 {
				val := results[0]
				uintResult := val.Uint()
				return uint8(uintResult)
			}
		}
	}
	return 0
}

// SaveMouseZoneColors will save mouse zone colors
func SaveMouseZoneColors(deviceId string, dpi rgb.Color, zones map[int]rgb.Color) uint8 {
	if device, ok := devices[deviceId]; ok {
		methodName := "SaveMouseZoneColors"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found or method is not supported for this device type")
			return 0
		} else {
			var reflectArgs []reflect.Value
			reflectArgs = append(reflectArgs, reflect.ValueOf(dpi))
			reflectArgs = append(reflectArgs, reflect.ValueOf(zones))
			results := method.Call(reflectArgs)
			if len(results) > 0 {
				val := results[0]
				uintResult := val.Uint()
				return uint8(uintResult)
			}
		}
	}
	return 0
}

// SaveMouseDpiColors will save mouse DPI colors
func SaveMouseDpiColors(deviceId string, dpi rgb.Color, zones map[int]rgb.Color) uint8 {
	if device, ok := devices[deviceId]; ok {
		methodName := "SaveMouseDpiColors"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found or method is not supported for this device type")
			return 0
		} else {
			var reflectArgs []reflect.Value
			reflectArgs = append(reflectArgs, reflect.ValueOf(dpi))
			reflectArgs = append(reflectArgs, reflect.ValueOf(zones))
			results := method.Call(reflectArgs)
			if len(results) > 0 {
				val := results[0]
				uintResult := val.Uint()
				return uint8(uintResult)
			}
		}
	}
	return 0
}

// UpdateExternalHubDeviceAmount will update a device amount connected to an external-LED hub
func UpdateExternalHubDeviceAmount(deviceId string, portId, deviceType int) uint8 {
	if device, ok := devices[deviceId]; ok {
		methodName := "UpdateExternalHubDeviceAmount"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found or method is not supported for this device type")
			return 0
		} else {
			var reflectArgs []reflect.Value
			reflectArgs = append(reflectArgs, reflect.ValueOf(portId))
			reflectArgs = append(reflectArgs, reflect.ValueOf(deviceType))
			results := method.Call(reflectArgs)
			if len(results) > 0 {
				val := results[0]
				uintResult := val.Uint()
				return uint8(uintResult)
			}
		}
	}
	return 0
}

// UpdateDeviceMetrics will update device metrics
func UpdateDeviceMetrics() {
	// Default
	metrics.PopulateDefault()

	// Storage
	metrics.PopulateStorage()

	// Devices
	for _, device := range devices {
		if device.ProductType == productTypeLinkHub ||
			device.ProductType == productTypeCC ||
			device.ProductType == productTypeElite ||
			device.ProductType == productTypeCPro ||
			device.ProductType == productTypeCCXT {
			methodName := "UpdateDeviceMetrics"
			method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
			if !method.IsValid() {
				logger.Log(logger.Fields{"method": methodName}).Warn("Method not found or method is not supported for this device type")
				continue
			} else {
				method.Call(nil)
			}
		}
	}
}

// SaveDeviceProfile will save keyboard profile
func SaveDeviceProfile(deviceId, profileName string, new bool) uint8 {
	if device, ok := devices[deviceId]; ok {
		methodName := "SaveDeviceProfile"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found or method is not supported for this device type")
			return 0
		} else {
			var reflectArgs []reflect.Value
			reflectArgs = append(reflectArgs, reflect.ValueOf(profileName))
			reflectArgs = append(reflectArgs, reflect.ValueOf(new))
			results := method.Call(reflectArgs)
			if len(results) > 0 {
				val := results[0]
				uintResult := val.Uint()
				return uint8(uintResult)
			}
		}
	}
	return 0
}

// ChangeKeyboardLayout will change keyboard layout
func ChangeKeyboardLayout(deviceId, layout string) uint8 {
	if device, ok := devices[deviceId]; ok {
		methodName := "ChangeKeyboardLayout"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found or method is not supported for this device type")
			return 0
		} else {
			var reflectArgs []reflect.Value
			reflectArgs = append(reflectArgs, reflect.ValueOf(layout))
			results := method.Call(reflectArgs)
			if len(results) > 0 {
				val := results[0]
				uintResult := val.Uint()
				return uint8(uintResult)
			}
		}
	}
	return 0
}

// ChangeKeyboardControlDial will change keyboard control dial function
func ChangeKeyboardControlDial(deviceId string, controlDial int) uint8 {
	if device, ok := devices[deviceId]; ok {
		methodName := "UpdateControlDial"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found or method is not supported for this device type")
			return 0
		} else {
			var reflectArgs []reflect.Value
			reflectArgs = append(reflectArgs, reflect.ValueOf(controlDial))
			results := method.Call(reflectArgs)
			if len(results) > 0 {
				val := results[0]
				uintResult := val.Uint()
				return uint8(uintResult)
			}
		}
	}
	return 0
}

// ChangeDeviceSleepMode will change device sleep mode
func ChangeDeviceSleepMode(deviceId string, sleepMode int) uint8 {
	if device, ok := devices[deviceId]; ok {
		methodName := "UpdateSleepTimer"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found or method is not supported for this device type")
			return 0
		} else {
			var reflectArgs []reflect.Value
			reflectArgs = append(reflectArgs, reflect.ValueOf(sleepMode))
			results := method.Call(reflectArgs)
			if len(results) > 0 {
				val := results[0]
				uintResult := val.Uint()
				return uint8(uintResult)
			}
		}
	}
	return 0
}

// ChangeKeyboardProfile will change keyboard profile
func ChangeKeyboardProfile(deviceId, profileName string) uint8 {
	if device, ok := devices[deviceId]; ok {
		methodName := "UpdateKeyboardProfile"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found or method is not supported for this device type")
			return 0
		} else {
			var reflectArgs []reflect.Value
			reflectArgs = append(reflectArgs, reflect.ValueOf(profileName))
			results := method.Call(reflectArgs)
			if len(results) > 0 {
				val := results[0]
				uintResult := val.Uint()
				return uint8(uintResult)
			}
		}
	}
	return 0
}

// DeleteKeyboardProfile will save keyboard profile
func DeleteKeyboardProfile(deviceId, profileName string) uint8 {
	if device, ok := devices[deviceId]; ok {
		methodName := "DeleteKeyboardProfile"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found or method is not supported for this device type")
			return 0
		} else {
			var reflectArgs []reflect.Value
			reflectArgs = append(reflectArgs, reflect.ValueOf(profileName))
			results := method.Call(reflectArgs)
			if len(results) > 0 {
				val := results[0]
				uintResult := val.Uint()
				return uint8(uintResult)
			}
		}
	}
	return 0
}

// SaveUserProfile will save new device user profile
func SaveUserProfile(deviceId, profileName string) uint8 {
	if device, ok := devices[deviceId]; ok {
		methodName := "SaveUserProfile"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found or method is not supported for this device type")
			return 0
		} else {
			var reflectArgs []reflect.Value
			reflectArgs = append(reflectArgs, reflect.ValueOf(profileName))
			results := method.Call(reflectArgs)
			if len(results) > 0 {
				val := results[0]
				uintResult := val.Uint()
				return uint8(uintResult)
			}
		}
	}
	return 0
}

// UpdateDevicePosition will change device position
func UpdateDevicePosition(deviceId string, position, direction int) uint8 {
	if device, ok := devices[deviceId]; ok {
		methodName := "UpdateDevicePosition"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found or method is not supported for this device type")
			return 0
		} else {
			var reflectArgs []reflect.Value
			reflectArgs = append(reflectArgs, reflect.ValueOf(position))
			reflectArgs = append(reflectArgs, reflect.ValueOf(direction))
			results := method.Call(reflectArgs)
			if len(results) > 0 {
				val := results[0]
				uintResult := val.Uint()
				return uint8(uintResult)
			}
		}
	}
	return 0
}

// ScheduleDeviceBrightness will change device brightness level based on scheduler
func ScheduleDeviceBrightness(mode uint8) {
	for _, device := range GetDevices() {
		methodName := "SchedulerBrightness"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found or method is not supported for this device type")
			continue
		} else {
			var reflectArgs []reflect.Value
			reflectArgs = append(reflectArgs, reflect.ValueOf(mode))
			method.Call(reflectArgs)
		}
	}
}

// ChangeDeviceBrightness will change device brightness level
func ChangeDeviceBrightness(deviceId string, value uint8) uint8 {
	if device, ok := devices[deviceId]; ok {
		methodName := "ChangeDeviceBrightness"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found or method is not supported for this device type")
			return 0
		} else {
			var reflectArgs []reflect.Value
			reflectArgs = append(reflectArgs, reflect.ValueOf(value))
			results := method.Call(reflectArgs)
			if len(results) > 0 {
				val := results[0]
				uintResult := val.Uint()
				return uint8(uintResult)
			}
		}
	}
	return 0
}

// ChangeDeviceBrightnessGradual will change device brightness level via defined number from 0-100
func ChangeDeviceBrightnessGradual(deviceId string, value uint8) uint8 {
	if device, ok := devices[deviceId]; ok {
		methodName := "ChangeDeviceBrightnessValue"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found or method is not supported for this device type")
			return 0
		} else {
			var reflectArgs []reflect.Value
			reflectArgs = append(reflectArgs, reflect.ValueOf(value))
			results := method.Call(reflectArgs)
			if len(results) > 0 {
				val := results[0]
				uintResult := val.Uint()
				return uint8(uintResult)
			}
		}
	}
	return 0
}

// ChangeUserProfile will change device user profile
func ChangeUserProfile(deviceId, profileName string) uint8 {
	if device, ok := devices[deviceId]; ok {
		methodName := "ChangeDeviceProfile"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found or method is not supported for this device type")
			return 0
		} else {
			var reflectArgs []reflect.Value
			reflectArgs = append(reflectArgs, reflect.ValueOf(profileName))
			results := method.Call(reflectArgs)
			if len(results) > 0 {
				val := results[0]
				uintResult := val.Uint()
				return uint8(uintResult)
			}
		}
	}
	return 0
}

// UpdateDeviceLcd will update device LCD
func UpdateDeviceLcd(deviceId string, channelId int, mode uint8) uint8 {
	if device, ok := devices[deviceId]; ok {
		methodName := "UpdateDeviceLcd"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found or method is not supported for this device type")
			return 0
		} else {
			var reflectArgs []reflect.Value
			reflectArgs = append(reflectArgs, reflect.ValueOf(channelId))
			reflectArgs = append(reflectArgs, reflect.ValueOf(mode))
			results := method.Call(reflectArgs)
			if len(results) > 0 {
				val := results[0]
				uintResult := val.Uint()
				return uint8(uintResult)
			}
		}
	}
	return 0
}

// ChangeDeviceLcd will change device LCD
func ChangeDeviceLcd(deviceId string, channelId int, lcdSerial string) uint8 {
	if device, ok := devices[deviceId]; ok {
		methodName := "ChangeDeviceLcd"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found or method is not supported for this device type")
			return 0
		} else {
			var reflectArgs []reflect.Value
			reflectArgs = append(reflectArgs, reflect.ValueOf(channelId))
			reflectArgs = append(reflectArgs, reflect.ValueOf(lcdSerial))
			results := method.Call(reflectArgs)
			if len(results) > 0 {
				val := results[0]
				uintResult := val.Uint()
				return uint8(uintResult)
			}
		}
	}
	return 0
}

// UpdateDeviceLcdRotation will update device LCD rotation
func UpdateDeviceLcdRotation(deviceId string, channelId int, rotation uint8) uint8 {
	if device, ok := devices[deviceId]; ok {
		methodName := "UpdateDeviceLcdRotation"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found or method is not supported for this device type")
			return 0
		} else {
			var reflectArgs []reflect.Value
			reflectArgs = append(reflectArgs, reflect.ValueOf(channelId))
			reflectArgs = append(reflectArgs, reflect.ValueOf(rotation))
			results := method.Call(reflectArgs)
			if len(results) > 0 {
				val := results[0]
				uintResult := val.Uint()
				return uint8(uintResult)
			}
		}
	}
	return 0
}

// UpdateDeviceLcdImage will update device LCD image
func UpdateDeviceLcdImage(deviceId string, channelId int, image string) uint8 {
	if device, ok := devices[deviceId]; ok {
		methodName := "UpdateDeviceLcdImage"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found or method is not supported for this device type")
			return 0
		} else {
			var reflectArgs []reflect.Value
			reflectArgs = append(reflectArgs, reflect.ValueOf(channelId))
			reflectArgs = append(reflectArgs, reflect.ValueOf(image))
			results := method.Call(reflectArgs)
			if len(results) > 0 {
				val := results[0]
				uintResult := val.Uint()
				return uint8(uintResult)
			}
		}
	}
	return 0
}

// UpdateDeviceLabel will set / update device label
func UpdateDeviceLabel(deviceId string, channelId int, label string, deviceType int) uint8 {
	if device, ok := devices[deviceId]; ok {
		methodName := ""
		if deviceType == 0 {
			methodName = "UpdateDeviceLabel"
		} else {
			methodName = "UpdateRGBDeviceLabel"
		}
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found or method is not supported for this device type")
			return 0
		} else {
			var reflectArgs []reflect.Value
			reflectArgs = append(reflectArgs, reflect.ValueOf(channelId))
			reflectArgs = append(reflectArgs, reflect.ValueOf(label))
			results := method.Call(reflectArgs)
			if len(results) > 0 {
				val := results[0]
				uintResult := val.Uint()
				return uint8(uintResult)
			}
		}
	}
	return 0
}

// UpdateSpeedProfile will update device speeds with a given serial number
func UpdateSpeedProfile(deviceId string, channelId int, profile string) uint8 {
	if device, ok := devices[deviceId]; ok {
		methodName := "UpdateSpeedProfile"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found or method is not supported for this device type")
			return 0
		} else {
			var reflectArgs []reflect.Value
			reflectArgs = append(reflectArgs, reflect.ValueOf(channelId))
			reflectArgs = append(reflectArgs, reflect.ValueOf(profile))
			results := method.Call(reflectArgs)
			if len(results) > 0 {
				val := results[0]
				uintResult := val.Uint()
				return uint8(uintResult)
			}
		}
	}
	return 0
}

// UpdateManualSpeed will update device speeds with a given serial number
func UpdateManualSpeed(deviceId string, channelId int, value uint16) uint8 {
	if device, ok := devices[deviceId]; ok {
		methodName := "UpdateDeviceSpeed"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found or method is not supported for this device type")
			return 0
		} else {
			var reflectArgs []reflect.Value
			reflectArgs = append(reflectArgs, reflect.ValueOf(channelId))
			reflectArgs = append(reflectArgs, reflect.ValueOf(value))
			results := method.Call(reflectArgs)
			if len(results) > 0 {
				val := results[0]
				uintResult := val.Uint()
				return uint8(uintResult)
			}
		}
	}
	return 0
}

// UpdateRgbStrip will update device RGB strip
func UpdateRgbStrip(deviceId string, channelId int, stripId int) uint8 {
	if device, ok := devices[deviceId]; ok {
		methodName := "UpdateExternalAdapter"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found or method is not supported for this device type")
			return 0
		} else {
			var reflectArgs []reflect.Value
			reflectArgs = append(reflectArgs, reflect.ValueOf(channelId))
			reflectArgs = append(reflectArgs, reflect.ValueOf(stripId))
			results := method.Call(reflectArgs)
			if len(results) > 0 {
				val := results[0]
				uintResult := val.Uint()
				return uint8(uintResult)
			}
		}
	}
	return 0
}

// UpdateRgbProfile will update device RGB profile
func UpdateRgbProfile(deviceId string, channelId int, profile string) uint8 {
	if device, ok := devices[deviceId]; ok {
		methodName := "UpdateRgbProfile"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found")
			return 0
		} else {
			var reflectArgs []reflect.Value
			reflectArgs = append(reflectArgs, reflect.ValueOf(channelId))
			reflectArgs = append(reflectArgs, reflect.ValueOf(profile))
			results := method.Call(reflectArgs)
			if len(results) > 0 {
				val := results[0]
				uintResult := val.Uint()
				return uint8(uintResult)
			}
		}
	}
	return 0
}

// ResetSpeedProfiles will reset the speed profile on each available device
func ResetSpeedProfiles(profile string) {
	for _, device := range devices {
		if device.ProductType == productTypeLinkHub ||
			device.ProductType == productTypeCC ||
			device.ProductType == productTypeCCXT {
			methodName := "ResetSpeedProfiles"
			method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
			if !method.IsValid() {
				logger.Log(logger.Fields{"method": methodName}).Warn("Method not found")
				return
			} else {
				var reflectArgs []reflect.Value
				reflectArgs = append(reflectArgs, reflect.ValueOf(profile))
				method.Call(reflectArgs)
			}
		}
	}
}

// GetDevices will return all available devices
func GetDevices() map[string]*Device {
	return devices
}

// GetTemperatureProbes will return a list of temperature probes
func GetTemperatureProbes() interface{} {
	var probes []interface{}
	for _, device := range devices {
		if device.ProductType == productTypeLinkHub ||
			device.ProductType == productTypeCC ||
			device.ProductType == productTypeCCXT ||
			device.ProductType == productTypeCPro {
			methodName := "GetTemperatureProbes"
			method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
			if !method.IsValid() {
				logger.Log(logger.Fields{"method": methodName}).Warn("Method not found or method is not supported for this device type")
				return 0
			} else {
				results := method.Call(nil)
				if len(results) > 0 {
					val := results[0]
					res := val.Interface()
					probes = append(probes, res)
				}
			}
		}
	}
	return probes
}

// GetDevice will return a device by device serial
func GetDevice(deviceId string) interface{} {
	if device, ok := devices[deviceId]; ok {
		return device.Instance
	}
	return nil
}

// Init will initialize all compatible Corsair devices in your system
func Init() {
	// Initialize general HID interface
	if err := hid.Init(); err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to initialize HID interface")
	}

	enum := hid.EnumFunc(func(info *hid.DeviceInfo) error {
		match := 0
		if slices.Contains(keyboards, info.ProductID) {
			interfaceId = 1 // Keyboard
		} else if slices.Contains(mouses, info.ProductID) {
			interfaceId = 1 // Mouse
		} else if slices.Contains(pads, info.ProductID) {
			interfaceId = 1 // Mousepad
		} else if slices.Contains(dongles, info.ProductID) {
			interfaceId = 1 // USB Dongle
		} else {
			interfaceId = 0
		}
		if info.InterfaceNbr == interfaceId {
			devPath := info.Path
			dev, err := os.Stat(devPath)
			if err != nil {
				logger.Log(logger.Fields{"error": err}).Error("Unable to get device stat info")
				return nil
			}
			filePerm := dev.Mode().Perm()
			for _, expectedPermission := range expectedPermissions {
				if filePerm != os.FileMode(expectedPermission) {
					continue
				} else {
					match++
				}
			}
			
			if match == 0 {
				logger.Log(logger.Fields{"productId": info.ProductID, "path": info.Path, "device": info.ProductStr}).Warn("Invalid permissions")
				return nil
			}

			if interfaceId == 1 {
				products[info.Path] = Product{
					ProductId: info.ProductID,
					Path:      info.Path,
				}
			} else {
				serial := info.SerialNbr
				if len(serial) == 0 {
					// Devices with no serial, make serial based of productId
					serial = strconv.Itoa(int(info.ProductID))
				}
				products[serial] = Product{
					ProductId: info.ProductID,
					Path:      info.Path,
				}
			}
		}
		return nil
	})

	// Enumerate all Corsair devices
	err := hid.Enumerate(vendorId, hid.ProductIDAny, enum)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": vendorId}).Fatal("Unable to enumerate devices")
	}

	if config.GetConfig().Memory {
		sm, err := smbus.GetSmBus()
		if err == nil {
			if len(sm.Path) > 0 {
				products[sm.Path] = Product{
					ProductId: 0,
					Path:      "",
				}
			}
		} else {
			logger.Log(logger.Fields{"error": err}).Warn("No valid I2C devices found")
		}
	}

	// USB-HID
	for key, product := range products {
		productId := product.ProductId
		productPath := product.Path
		if slices.Contains(config.GetConfig().Exclude, productId) {
			logger.Log(logger.Fields{"productId": productId}).Warn("Product excluded via config.json")
			continue
		}
		switch product.ProductId {
		case 3135: // CORSAIR iCUE Link System Hub
			{
				go func(vendorId, productId uint16, serialId string) {
					dev := lsh.Init(vendorId, productId, serialId)
					if dev == nil {
						return
					}
					devices[dev.Serial] = &Device{
						ProductType: productTypeLinkHub,
						Product:     dev.Product,
						Serial:      dev.Serial,
						Firmware:    dev.Firmware,
						Image:       "icon-device.svg",
						Instance:    dev,
					}
					devices[dev.Serial].GetDevice = GetDevice(dev.Serial)
				}(vendorId, productId, key)
			}

		case 3122, 3100: // CORSAIR iCUE COMMANDER Core
			{
				go func(vendorId, productId uint16, serialId string) {
					dev := cc.Init(vendorId, productId, serialId)
					if dev == nil {
						return
					}
					devices[dev.Serial] = &Device{
						ProductType: productTypeCC,
						Product:     dev.Product,
						Serial:      dev.Serial,
						Firmware:    dev.Firmware,
						Image:       "icon-device.svg",
						Instance:    dev,
					}
					devices[dev.Serial].GetDevice = GetDevice(dev.Serial)
				}(vendorId, productId, key)
			}
		case 3114: // CORSAIR iCUE COMMANDER CORE XT
			{
				go func(vendorId, productId uint16, serialId string) {
					dev := ccxt.Init(vendorId, productId, serialId)
					if dev == nil {
						return
					}
					devices[dev.Serial] = &Device{
						ProductType: productTypeCCXT,
						Product:     dev.Product,
						Serial:      dev.Serial,
						Firmware:    dev.Firmware,
						Image:       "icon-device.svg",
						Instance:    dev,
					}
					devices[dev.Serial].GetDevice = GetDevice(dev.Serial)
				}(vendorId, productId, key)
			}
		case 3125, 3126, 3127, 3136, 3137, 3104, 3105, 3106, 3095, 3096, 3097:
			// iCUE H100i ELITE RGB
			// iCUE H115i ELITE RGB
			// iCUE H150i ELITE RGB
			// iCUE H100i ELITE RGB White
			// iCUE H150i ELITE RGB White
			// iCUE H100i RGB PRO XT
			// iCUE H115i RGB PRO XT
			// iCUE H150i RGB PRO XT
			// H115i RGB PLATINUM
			// H100i RGB PLATINUM
			// H100i RGB PLATINUM SE
			{
				go func(vendorId, productId uint16) {
					dev := elite.Init(vendorId, productId)
					if dev == nil {
						return
					}
					devices[strconv.Itoa(int(productId))] = &Device{
						ProductType: productTypeElite,
						Product:     dev.Product,
						Serial:      dev.Serial,
						Firmware:    dev.Firmware,
						Image:       "icon-device.svg",
						Instance:    dev,
					}
					devices[dev.Serial].GetDevice = GetDevice(dev.Serial)
				}(vendorId, productId)
			}
		case 3098: // CORSAIR Lighting Node CORE
			{
				go func(vendorId, productId uint16, serialId string) {
					dev := lncore.Init(vendorId, productId, serialId)
					if dev == nil {
						return
					}
					devices[dev.Serial] = &Device{
						ProductType: productTypeLNCore,
						Product:     dev.Product,
						Serial:      dev.Serial,
						Firmware:    dev.Firmware,
						Image:       "icon-device.svg",
						Instance:    dev,
					}
				}(vendorId, productId, key)
			}
		case 3083: // CORSAIR Lighting Node Pro
			{
				go func(vendorId, productId uint16, serialId string) {
					dev := lnpro.Init(vendorId, productId, serialId)
					if dev == nil {
						return
					}
					devices[dev.Serial] = &Device{
						ProductType: productTypeLnPro,
						Product:     dev.Product,
						Serial:      dev.Serial,
						Firmware:    dev.Firmware,
						Image:       "icon-device.svg",
						Instance:    dev,
					}
				}(vendorId, productId, key)
			}
		case 3088: // Corsair Commander Pro
			{
				go func(vendorId, productId uint16, serialId string) {
					dev := cpro.Init(vendorId, productId, serialId)
					if dev == nil {
						return
					}
					devices[dev.Serial] = &Device{
						ProductType: productTypeCPro,
						Product:     dev.Product,
						Serial:      dev.Serial,
						Firmware:    dev.Firmware,
						Image:       "icon-device.svg",
						Instance:    dev,
					}
					devices[dev.Serial].GetDevice = GetDevice(dev.Serial)
				}(vendorId, productId, key)
			}
		case 3138: // CORSAIR XC7 ELITE LCD CPU Water Block
			{
				go func(vendorId, productId uint16, serialId string) {
					dev := xc7.Init(vendorId, productId, serialId)
					if dev == nil {
						return
					}
					devices[dev.Serial] = &Device{
						ProductType: productTypeXC7,
						Product:     dev.Product,
						Serial:      dev.Serial,
						Firmware:    dev.Firmware,
						Image:       "icon-device.svg",
						Instance:    dev,
					}
					devices[dev.Serial].GetDevice = GetDevice(dev.Serial)
				}(vendorId, productId, key)
			}
		case 7127: // K65 Pro Mini
			{
				go func(vendorId, productId uint16, key string) {
					dev := k65pm.Init(vendorId, productId, key)
					if dev == nil {
						return
					}
					devices[dev.Serial] = &Device{
						ProductType: productTypeK65PM,
						Product:     dev.Product,
						Serial:      dev.Serial,
						Firmware:    dev.Firmware,
						Image:       "icon-keyboard.svg",
						Instance:    dev,
					}
				}(vendorId, productId, key)
			}
		case 7165: // K70 CORE RGB
			{
				go func(vendorId, productId uint16, key string) {
					dev := k70core.Init(vendorId, productId, key)
					if dev == nil {
						return
					}
					devices[dev.Serial] = &Device{
						ProductType: productTypeK70Core,
						Product:     dev.Product,
						Serial:      dev.Serial,
						Firmware:    dev.Firmware,
						Image:       "icon-keyboard.svg",
						Instance:    dev,
					}
				}(vendorId, productId, key)
			}
		case 7166: // K55 CORE RGB
			{
				go func(vendorId, productId uint16, key string) {
					dev := k55core.Init(vendorId, productId, key)
					if dev == nil {
						return
					}
					devices[dev.Serial] = &Device{
						ProductType: productTypeK55Core,
						Product:     dev.Product,
						Serial:      dev.Serial,
						Firmware:    dev.Firmware,
						Image:       "icon-keyboard.svg",
						Instance:    dev,
					}
				}(vendorId, productId, key)
			}
		case 7110, 7091: // K70 RGB PRO
			{
				go func(vendorId, productId uint16, key string) {
					dev := k70pro.Init(vendorId, productId, key)
					if dev == nil {
						return
					}
					devices[dev.Serial] = &Device{
						ProductType: productTypeK70Pro,
						Product:     dev.Product,
						Serial:      dev.Serial,
						Firmware:    dev.Firmware,
						Image:       "icon-keyboard.svg",
						Instance:    dev,
					}
				}(vendorId, productId, key)
			}
		case 11024: // K65 PLUS USB
			{
				go func(vendorId, productId uint16, key string) {
					dev := k65plus.Init(vendorId, productId, key)
					if dev == nil {
						return
					}
					devices[dev.Serial] = &Device{
						ProductType: productTypeK65Plus,
						Product:     dev.Product,
						Serial:      dev.Serial,
						Firmware:    dev.Firmware,
						Image:       "icon-keyboard.svg",
						Instance:    dev,
					}
				}(vendorId, productId, key)
			}
		case 11015: // K65 PLUS WIRELESS
			{
				go func(vendorId, productId uint16, key string) {
					dev := k65plusW.Init(vendorId, productId, key)
					if dev == nil {
						return
					}
					devices[dev.Serial] = &Device{
						ProductType: productTypeK65PlusW,
						Product:     dev.Product,
						Serial:      dev.Serial,
						Firmware:    dev.Firmware,
						Image:       "icon-keyboard.svg",
						Instance:    dev,
					}
				}(vendorId, productId, key)
			}
		case 7083: // K100 AIR USB
			{
				go func(vendorId, productId uint16, key string) {
					dev := k100air.Init(vendorId, productId, key)
					if dev == nil {
						return
					}
					devices[dev.Serial] = &Device{
						ProductType: productTypeK100Air,
						Product:     dev.Product,
						Serial:      dev.Serial,
						Firmware:    dev.Firmware,
						Image:       "icon-keyboard.svg",
						Instance:    dev,
					}
				}(vendorId, productId, key)
			}
		case 7109, 7036, 7037: // K100 RGB
			{
				go func(vendorId, productId uint16, key string) {
					dev := k100.Init(vendorId, productId, key)
					if dev == nil {
						return
					}
					devices[dev.Serial] = &Device{
						ProductType: productTypeK100,
						Product:     dev.Product,
						Serial:      dev.Serial,
						Firmware:    dev.Firmware,
						Image:       "icon-keyboard.svg",
						Instance:    dev,
					}
				}(vendorId, productId, key)
			}
		case 7132, 7078, 11008: // Corsair SLIPSTREAM WIRELESS USB Receiver
			{
				go func(vendorId, productId uint16, key string) {
					dev := slipstream.Init(vendorId, productId, key)
					devices[dev.Serial] = &Device{
						ProductType: productTypeIronClawRgbW,
						Product:     "SLIPSTREAM",
						Serial:      dev.Serial,
						Firmware:    dev.Firmware,
						Image:       "icon-dongle.svg",
						Instance:    dev,
						Hidden:      true,
					}
					for _, value := range dev.Devices {
						switch value.ProductId {
						case 7163:
							{
								d := m55W.Init(
									value.VendorId,
									productId,
									value.ProductId,
									dev.GetDevice(),
									value.Endpoint,
									value.Serial,
								)
								devices[d.Serial] = &Device{
									ProductType: productTypeM55W,
									Product:     "M55 WIRELESS",
									Serial:      d.Serial,
									Firmware:    d.Firmware,
									Image:       "icon-mouse.svg",
									Instance:    d,
								}
								dev.AddPairedDevice(value.ProductId, d)
							}
						case 7131:
							{
								d := scimitarW.Init(
									value.VendorId,
									productId,
									value.ProductId,
									dev.GetDevice(),
									value.Endpoint,
									value.Serial,
								)
								devices[d.Serial] = &Device{
									ProductType: productTypeScimitarRgbEliteW,
									Product:     "SCIMITAR RGB ELITE",
									Serial:      d.Serial,
									Firmware:    d.Firmware,
									Image:       "icon-mouse.svg",
									Instance:    d,
								}
								dev.AddPairedDevice(value.ProductId, d)
							}
						case 7096: // NIGHTSABRE
							{
								d := nightsabreW.Init(
									value.VendorId,
									productId,
									value.ProductId,
									dev.GetDevice(),
									value.Endpoint,
									value.Serial,
								)
								devices[d.Serial] = &Device{
									ProductType: productTypeNightsabreW,
									Product:     "NIGHTSABRE",
									Serial:      d.Serial,
									Firmware:    d.Firmware,
									Image:       "icon-mouse.svg",
									Instance:    d,
								}
								dev.AddPairedDevice(value.ProductId, d)
							}
						case 7083: // K100 AIR WIRELESS
							{
								d := k100airW.Init(
									value.VendorId,
									productId,
									value.ProductId,
									dev.GetDevice(),
									value.Endpoint,
									value.Serial,
								)
								devices[d.Serial] = &Device{
									ProductType: productTypeK100AirW,
									Product:     "K100 AIR",
									Serial:      d.Serial,
									Firmware:    d.Firmware,
									Image:       "icon-keyboard.svg",
									Instance:    d,
								}
								dev.AddPairedDevice(value.ProductId, d)
							}
						case 6988: // IRONCLAW RGB WIRELESS
							{
								d := ironclawW.Init(
									value.VendorId,
									productId,
									value.ProductId,
									dev.GetDevice(),
									value.Endpoint,
									value.Serial,
								)
								devices[d.Serial] = &Device{
									ProductType: productTypeIronClawRgbW,
									Product:     "IRONCLAW RGB",
									Serial:      d.Serial,
									Firmware:    d.Firmware,
									Image:       "icon-mouse.svg",
									Instance:    d,
								}
								dev.AddPairedDevice(value.ProductId, d)
							}
						default:
							logger.Log(logger.Fields{"productId": value.ProductId}).Warn("Unsupported device detected")
						}
					}
					dev.InitAvailableDevices()
				}(vendorId, productId, key)
			}
		case 2612: // Corsair ST100 LED Driver
			{
				go func(vendorId, productId uint16, key string) {
					dev := st100.Init(vendorId, productId, key)
					if dev == nil {
						return
					}
					devices[dev.Serial] = &Device{
						ProductType: productTypeST100,
						Product:     dev.Product,
						Serial:      dev.Serial,
						Firmware:    dev.Firmware,
						Image:       "icon-headphone.svg",
						Instance:    dev,
					}
				}(vendorId, productId, key)
			}
		case 7067: // Corsair MM700 RGB Gaming Mousepad
			{
				go func(vendorId, productId uint16, key string) {
					dev := mm700.Init(vendorId, productId, key)
					if dev == nil {
						return
					}
					devices[dev.Serial] = &Device{
						ProductType: productTypeMM700,
						Product:     dev.Product,
						Serial:      dev.Serial,
						Firmware:    dev.Firmware,
						Image:       "icon-mousepad.svg",
						Instance:    dev,
					}
				}(vendorId, productId, key)
			}
		case 3107: // Corsair iCUE LT100 Smart Lighting Tower
			{
				go func(vendorId, productId uint16, key, devicePath string) {
					dev := lt100.Init(vendorId, productId, key, devicePath)
					if dev == nil {
						return
					}
					devices[dev.Serial] = &Device{
						ProductType: productTypeLT100,
						Product:     dev.Product,
						Serial:      dev.Serial,
						Firmware:    dev.Firmware,
						Image:       "icon-towers.svg",
						Instance:    dev,
					}
				}(vendorId, productId, key, productPath)
			}
		case 7198, 7203, 7199, 7173, 7174, 7175, 7176, 7181, 7180:
			// Corsair HX1000i Power Supply
			// Corsair HX1200i Power Supply
			// Corsair HX1500i Power Supply
			// Corsair HX750i Power Supply
			// Corsair HX850i Power Supply
			// Corsair HX1000i Power Supply
			// Corsair HX1200i Power Supply
			// Corsair RM1000i Power Supply
			// Corsair RM850i Power Supply
			{
				go func(vendorId, productId uint16, key string) {
					dev := psuhid.Init(vendorId, productId, key)
					if dev == nil {
						return
					}
					devices[dev.Serial] = &Device{
						ProductType: productTypePSUHid,
						Product:     dev.Product,
						Serial:      dev.Serial,
						Firmware:    dev.Firmware,
						Image:       "icon-psu.svg",
						Instance:    dev,
					}
					devices[dev.Serial].GetDevice = GetDevice(dev.Serial)
				}(vendorId, productId, productPath)
			}
		case 7059: // Corsair KATAR PRO Gaming Mouse
			{
				go func(vendorId, productId uint16, key string) {
					dev := katarpro.Init(vendorId, productId, key)
					if dev == nil {
						return
					}
					devices[dev.Serial] = &Device{
						ProductType: productTypeKatarPro,
						Product:     dev.Product,
						Serial:      dev.Serial,
						Firmware:    dev.Firmware,
						Image:       "icon-mouse.svg",
						Instance:    dev,
					}
				}(vendorId, productId, key)
			}
		case 7005: // Corsair IRONCLAW RGB Gaming Mouse
			{
				go func(vendorId, productId uint16, key string) {
					dev := ironclaw.Init(vendorId, productId, key)
					if dev == nil {
						return
					}
					devices[dev.Serial] = &Device{
						ProductType: productTypeIronClawRgb,
						Product:     dev.Product,
						Serial:      dev.Serial,
						Firmware:    dev.Firmware,
						Image:       "icon-mouse.svg",
						Instance:    dev,
					}
				}(vendorId, productId, key)
			}
		case 6988: // Corsair IRONCLAW RGB WIRELESS Gaming Mouse
			{
				go func(vendorId, productId uint16, key string) {
					dev := ironclawWU.Init(vendorId, productId, key)
					if dev == nil {
						return
					}
					devices[dev.Serial] = &Device{
						ProductType: productTypeIronClawRgbWU,
						Product:     dev.Product,
						Serial:      dev.Serial,
						Firmware:    dev.Firmware,
						Image:       "icon-mouse.svg",
						Instance:    dev,
					}
				}(vendorId, productId, key)
			}
		case 7096: // Corsair NIGHTSABRE WIRELESS Mouse
			{
				go func(vendorId, productId uint16, key string) {
					dev := nightsabreWU.Init(vendorId, productId, key)
					if dev == nil {
						return
					}
					devices[dev.Serial] = &Device{
						ProductType: productTypeNightsabreWU,
						Product:     dev.Product,
						Serial:      dev.Serial,
						Firmware:    dev.Firmware,
						Image:       "icon-mouse.svg",
						Instance:    dev,
					}
				}(vendorId, productId, key)
			}
		case 7139: // CORSAIR SCIMITAR RGB ELITE
			{
				go func(vendorId, productId uint16, key string) {
					dev := scimitar.Init(vendorId, productId, key)
					if dev == nil {
						return
					}
					devices[dev.Serial] = &Device{
						ProductType: productTypeScimitarRgbElite,
						Product:     dev.Product,
						Serial:      dev.Serial,
						Firmware:    dev.Firmware,
						Image:       "icon-mouse.svg",
						Instance:    dev,
					}
				}(vendorId, productId, key)
			}
		case 7131: // CORSAIR SCIMITAR RGB ELITE WIRELESS
			{
				go func(vendorId, productId uint16, key string) {
					dev := scimitarWU.Init(vendorId, productId, key)
					if dev == nil {
						return
					}
					devices[dev.Serial] = &Device{
						ProductType: productTypeScimitarRgbEliteWU,
						Product:     dev.Product,
						Serial:      dev.Serial,
						Firmware:    dev.Firmware,
						Image:       "icon-mouse.svg",
						Instance:    dev,
					}
				}(vendorId, productId, key)
			}
		case 11011: // CORSAIR M55 Gaming Mouse
			{
				go func(vendorId, productId uint16, key string) {
					dev := m55.Init(vendorId, productId, key)
					if dev == nil {
						return
					}
					devices[dev.Serial] = &Device{
						ProductType: productTypeM55,
						Product:     dev.Product,
						Serial:      dev.Serial,
						Firmware:    dev.Firmware,
						Image:       "icon-mouse.svg",
						Instance:    dev,
					}
				}(vendorId, productId, key)
			}
		case 7024: // CORSAIR M55 RGB PRO Gaming Mouse
			{
				go func(vendorId, productId uint16, key string) {
					dev := m55rgbpro.Init(vendorId, productId, key)
					if dev == nil {
						return
					}
					devices[dev.Serial] = &Device{
						ProductType: productTypeM55RgbPro,
						Product:     dev.Product,
						Serial:      dev.Serial,
						Firmware:    dev.Firmware,
						Image:       "icon-mouse.svg",
						Instance:    dev,
					}
				}(vendorId, productId, key)
			}
		case 7060: // CORSAIR KATAR PRO Wireless Gaming Dongle
			{
				go func(vendorId, productId uint16, key string) {
					dev := katarproW.Init(vendorId, productId, key)
					devices[dev.Serial] = &Device{
						ProductType: productTypeKatarProW,
						Product:     "KATAR PRO WIRELESS",
						Serial:      dev.Serial,
						Firmware:    dev.Firmware,
						Image:       "icon-mouse.svg",
						Instance:    dev,
					}
				}(vendorId, productId, key)
			}
		case 0: // Memory
			{
				go func(serialId string) {
					dev := memory.Init(serialId, "Memory")
					if dev != nil {
						devices[dev.Serial] = &Device{
							ProductType: productTypeMemory,
							Product:     dev.Product,
							Serial:      dev.Serial,
							Firmware:    "0",
							Image:       "icon-ram.svg",
							Instance:    dev,
						}
						devices[dev.Serial].GetDevice = GetDevice(dev.Serial)
					}
				}(key)
			}
		default:
			continue
		}
	}
}
