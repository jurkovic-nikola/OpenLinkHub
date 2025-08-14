package devices

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/devices/cc"
	"OpenLinkHub/src/devices/ccxt"
	"OpenLinkHub/src/devices/cpro"
	"OpenLinkHub/src/devices/darkcorergbproW"
	"OpenLinkHub/src/devices/darkcorergbproWU"
	"OpenLinkHub/src/devices/darkcorergbproseW"
	"OpenLinkHub/src/devices/darkcorergbproseWU"
	"OpenLinkHub/src/devices/darkstarW"
	"OpenLinkHub/src/devices/darkstarWU"
	"OpenLinkHub/src/devices/elite"
	"OpenLinkHub/src/devices/harpoonW"
	"OpenLinkHub/src/devices/harpoonWU"
	"OpenLinkHub/src/devices/harpoonrgbpro"
	"OpenLinkHub/src/devices/headsetdongle"
	"OpenLinkHub/src/devices/hs80maxW"
	"OpenLinkHub/src/devices/hs80maxdongle"
	"OpenLinkHub/src/devices/hs80rgb"
	"OpenLinkHub/src/devices/hs80rgbW"
	"OpenLinkHub/src/devices/ironclaw"
	"OpenLinkHub/src/devices/ironclawW"
	"OpenLinkHub/src/devices/ironclawWU"
	"OpenLinkHub/src/devices/k100"
	"OpenLinkHub/src/devices/k100air"
	"OpenLinkHub/src/devices/k100airW"
	"OpenLinkHub/src/devices/k55"
	"OpenLinkHub/src/devices/k55core"
	"OpenLinkHub/src/devices/k55pro"
	"OpenLinkHub/src/devices/k55proXT"
	"OpenLinkHub/src/devices/k60rgbpro"
	"OpenLinkHub/src/devices/k65plusW"
	"OpenLinkHub/src/devices/k65plusWU"
	"OpenLinkHub/src/devices/k65pm"
	"OpenLinkHub/src/devices/k70core"
	"OpenLinkHub/src/devices/k70coretkl"
	"OpenLinkHub/src/devices/k70coretklW"
	"OpenLinkHub/src/devices/k70coretklWU"
	"OpenLinkHub/src/devices/k70mk2"
	"OpenLinkHub/src/devices/k70pmW"
	"OpenLinkHub/src/devices/k70pmWU"
	"OpenLinkHub/src/devices/k70pro"
	"OpenLinkHub/src/devices/k70protkl"
	"OpenLinkHub/src/devices/k70rgbtklcs"
	"OpenLinkHub/src/devices/k95platinum"
	"OpenLinkHub/src/devices/katarpro"
	"OpenLinkHub/src/devices/katarproW"
	"OpenLinkHub/src/devices/katarproxt"
	"OpenLinkHub/src/devices/lncore"
	"OpenLinkHub/src/devices/lnpro"
	"OpenLinkHub/src/devices/lsh"
	"OpenLinkHub/src/devices/lt100"
	"OpenLinkHub/src/devices/m55"
	"OpenLinkHub/src/devices/m55W"
	"OpenLinkHub/src/devices/m55rgbpro"
	"OpenLinkHub/src/devices/m65rgbultra"
	"OpenLinkHub/src/devices/m65rgbultraW"
	"OpenLinkHub/src/devices/m65rgbultraWU"
	"OpenLinkHub/src/devices/m75"
	"OpenLinkHub/src/devices/m75AirW"
	"OpenLinkHub/src/devices/m75AirWU"
	"OpenLinkHub/src/devices/m75W"
	"OpenLinkHub/src/devices/m75WU"
	"OpenLinkHub/src/devices/memory"
	"OpenLinkHub/src/devices/mm700"
	"OpenLinkHub/src/devices/mm800"
	"OpenLinkHub/src/devices/nexus"
	"OpenLinkHub/src/devices/nightsabreW"
	"OpenLinkHub/src/devices/nightsabreWU"
	"OpenLinkHub/src/devices/nightswordrgb"
	"OpenLinkHub/src/devices/platinum"
	"OpenLinkHub/src/devices/psuhid"
	"OpenLinkHub/src/devices/sabrergbpro"
	"OpenLinkHub/src/devices/sabrergbproW"
	"OpenLinkHub/src/devices/sabrergbproWU"
	"OpenLinkHub/src/devices/scimitar"
	"OpenLinkHub/src/devices/scimitarSEW"
	"OpenLinkHub/src/devices/scimitarSEWU"
	"OpenLinkHub/src/devices/scimitarW"
	"OpenLinkHub/src/devices/scimitarWU"
	"OpenLinkHub/src/devices/scimitarrgbelite"
	"OpenLinkHub/src/devices/slipstream"
	"OpenLinkHub/src/devices/st100"
	"OpenLinkHub/src/devices/virtuosomaxW"
	"OpenLinkHub/src/devices/virtuosomaxdongle"
	"OpenLinkHub/src/devices/virtuosorgbXTW"
	"OpenLinkHub/src/devices/virtuosorgbXTWU"
	"OpenLinkHub/src/devices/xc7"
	"OpenLinkHub/src/inputmanager"
	"OpenLinkHub/src/led"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/metrics"
	"OpenLinkHub/src/rgb"
	"OpenLinkHub/src/smbus"
	"OpenLinkHub/src/usb"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"

	"OpenLinkHub/src/devices/scimitarprorgb"
	"github.com/sstallion/go-hid"
)

const (
	productTypeLinkHub              = 0
	productTypeCC                   = 1
	productTypeCCXT                 = 2
	productTypeElite                = 3
	productTypeLNCore               = 4
	productTypeLnPro                = 5
	productTypeCPro                 = 6
	productTypeXC7                  = 7
	productTypeMemory               = 8
	productTypeNexus                = 9
	productTypePlatinum             = 10
	productTypeK65PM                = 101
	productTypeK70Core              = 102
	productTypeK55Core              = 103
	productTypeK70Pro               = 104
	productTypeK65Plus              = 105
	productTypeK65PlusW             = 106
	productTypeK100Air              = 107
	productTypeK100AirW             = 108
	productTypeK100                 = 109
	productTypeK70MK2               = 110
	productTypeK70CoreTkl           = 111
	productTypeK70CoreTklWU         = 112
	productTypeK70CoreTklW          = 113
	productTypeK70ProTkl            = 114
	productTypeK70RgbTkl            = 115
	productTypeK55Pro               = 116
	productTypeK55ProXT             = 117
	productTypeK55                  = 118
	productTypeK95Platinum          = 119
	productTypeK60RgbPro            = 120
	productTypeK70PMW               = 121
	productTypeK70PMWU              = 122
	productTypeKatarPro             = 201
	productTypeIronClawRgb          = 202
	productTypeIronClawRgbW         = 203
	productTypeIronClawRgbWU        = 204
	productTypeNightsabreW          = 205
	productTypeNightsabreWU         = 206
	productTypeScimitarRgbElite     = 207
	productTypeScimitarRgbEliteW    = 208
	productTypeScimitarRgbEliteWU   = 209
	productTypeM55                  = 210
	productTypeM55W                 = 211
	productTypeM55RgbPro            = 212
	productTypeKatarProW            = 213
	productTypeDarkCoreRgbProSEW    = 214
	productTypeDarkCoreRgbProSEWU   = 215
	productTypeDarkCoreRgbProW      = 216
	productTypeDarkCoreRgbProWU     = 217
	productTypeM75                  = 218
	productTypeM75AirW              = 219
	productTypeM75AirWU             = 220
	productTypeM75W                 = 221
	productTypeM75WU                = 222
	productTypeM65RgbUltra          = 223
	productTypeHarpoonRgbPro        = 224
	productTypeHarpoonRgbW          = 225
	productTypeHarpoonRgbWU         = 226
	productTypeKatarProXT           = 227
	productTypeDarkstarWU           = 228
	productTypeDarkstarW            = 229
	productTypeScimitarRgbEliteSEW  = 230
	productTypeScimitarRgbEliteSEWU = 231
	productTypeM65RgbUltraW         = 232
	productTypeM65RgbUltraWU        = 233
	productTypeSabreRgbProWU        = 234
	productTypeSabreRgbProW         = 235
	productTypeNightswordRgb        = 236
	productTypeSabreRgbPro          = 237
	productTypeScimitarProRgb       = 238
	productTypeVirtuosoXTW          = 300
	productTypeVirtuosoXTWU         = 301
	productTypeVirtuosoMAXW         = 302
	productTypeHS80RGBW             = 303
	productTypeHS80MAXW             = 304
	productTypeHS80RGB              = 305
	productTypeST100                = 401
	productTypeMM700                = 402
	productTypeLT100                = 403
	productTypeMM800                = 404
	productTypePSUHid               = 501
)

type AIOData struct {
	Rpm         int16
	Temperature float32
	Serial      string
}
type Product struct {
	ProductId uint16
	Path      string
	DevPath   string
	Serial    string
}

type ProductEX struct {
	ProductId uint16
	Serial    string
	Path      string
}

var (
	expectedPermissions        = []os.FileMode{os.FileMode(0600), os.FileMode(0660)}
	vendorId            uint16 = 6940 // Corsair
	interfaceId                = 0
	devices                    = make(map[string]*common.Device)
	products                   = make(map[string]Product)
	keyboards                  = []uint16{7127, 7165, 7166, 7110, 7083, 11024, 11025, 11015, 7109, 7091, 7124, 7036, 7037, 6985, 6997, 7019, 11009, 11010, 11028, 7097, 7027, 7076, 7073, 6973, 6957, 7072, 7094}
	mouses                     = []uint16{7059, 7005, 6988, 7096, 7139, 7131, 11011, 7024, 7038, 7040, 7152, 7154, 11016, 7070, 7029, 7006, 7084, 7090, 11042, 7093, 7126, 7163, 7064, 7051, 7004, 7033, 6974}
	pads                       = []uint16{7067, 7113}
	headsets                   = []uint16{2658, 2660, 2667, 2696}
	headsets2                  = []uint16{10754, 2711}
	dongles                    = []uint16{7132, 7078, 11008, 7060}
	legacyDevices              = []uint16{3090, 3091, 3093}
)

// isUSBConnected will check if a USB device is connected
func isUSBConnected(productId uint16) bool {
	for _, value := range products {
		if value.ProductId == productId {
			return true
		}
	}
	return false
}

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

// StopDirty will stop the device without closing the file handles. Used when device is unplugged
func StopDirty(deviceId string) {
	if device, ok := devices[deviceId]; ok {
		methodName := "StopDirty"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found")
		} else {
			results := method.Call(nil)
			if len(results) > 0 {
				val := results[0]
				uintResult := val.Uint()
				if uint8(uintResult) == 2 { // USB only devices, remove them from the device list
					delete(devices, deviceId)
				}
			}
		}
	}
}

// GetRgbProfiles will return a list of all RGB profiles for every device
func GetRgbProfiles() map[string]interface{} {
	profiles := make(map[string]interface{}, len(devices))
	for _, device := range devices {
		methodName := "GetRgbProfiles"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName, "device": device.Product}).Warn("Method not found or method is not supported for this device type")
			continue
		} else {
			results := method.Call(nil)
			if len(results) > 0 {
				val := results[0]
				profiles[device.Serial] = val.Interface()
			}
		}
	}
	return profiles
}

// GetRgbProfile will return a list of RGB profiles for a target device
func GetRgbProfile(deviceId string) interface{} {
	if device, ok := devices[deviceId]; ok {
		methodName := "GetRgbProfiles"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName, "device": device.Product}).Warn("Method not found or method is not supported for this device type")
			return nil
		} else {
			results := method.Call(nil)
			if len(results) > 0 {
				val := results[0]
				return val.Interface()
			}
		}
	}
	return nil
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

// SaveMouseZoneColorsSniper will save mouse zone colors + sniper color if available
func SaveMouseZoneColorsSniper(deviceId string, dpi rgb.Color, zones map[int]rgb.Color, sniper rgb.Color) uint8 {
	if device, ok := devices[deviceId]; ok {
		methodName := "SaveMouseZoneColorsSniper"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found or method is not supported for this device type")
			return 0
		} else {
			var reflectArgs []reflect.Value
			reflectArgs = append(reflectArgs, reflect.ValueOf(dpi))
			reflectArgs = append(reflectArgs, reflect.ValueOf(zones))
			reflectArgs = append(reflectArgs, reflect.ValueOf(sniper))
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

// SaveHeadsetZoneColors will save headset zone colors
func SaveHeadsetZoneColors(deviceId string, zones map[int]rgb.Color) uint8 {
	if device, ok := devices[deviceId]; ok {
		methodName := "SaveHeadsetZoneColors"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found or method is not supported for this device type")
			return 0
		} else {
			var reflectArgs []reflect.Value
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

// ChangeDevicePollingRate will change device polling rate
func ChangeDevicePollingRate(deviceId string, pullingRate int) uint8 {
	if device, ok := devices[deviceId]; ok {
		methodName := "UpdatePollingRate"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found or method is not supported for this device type")
			return 0
		} else {
			var reflectArgs []reflect.Value
			reflectArgs = append(reflectArgs, reflect.ValueOf(pullingRate))
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

// ChangeDeviceAngleSnapping will change device angle snapping mode
func ChangeDeviceAngleSnapping(deviceId string, angleSnappingMode int) uint8 {
	if device, ok := devices[deviceId]; ok {
		methodName := "UpdateAngleSnapping"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found or method is not supported for this device type")
			return 0
		} else {
			var reflectArgs []reflect.Value
			reflectArgs = append(reflectArgs, reflect.ValueOf(angleSnappingMode))
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

// ChangeDeviceButtonOptimization will change device button optimization mode
func ChangeDeviceButtonOptimization(deviceId string, buttonOptimizationMode int) uint8 {
	if device, ok := devices[deviceId]; ok {
		methodName := "UpdateButtonOptimization"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found or method is not supported for this device type")
			return 0
		} else {
			var reflectArgs []reflect.Value
			reflectArgs = append(reflectArgs, reflect.ValueOf(buttonOptimizationMode))
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

// ChangeDeviceKeyAssignment will change device key assignment
func ChangeDeviceKeyAssignment(deviceId string, keyIndex int, keyAssignment inputmanager.KeyAssignment) uint8 {
	if device, ok := devices[deviceId]; ok {
		methodName := "UpdateDeviceKeyAssignment"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found or method is not supported for this device type")
			return 0
		} else {
			var reflectArgs []reflect.Value
			reflectArgs = append(reflectArgs, reflect.ValueOf(keyIndex))
			reflectArgs = append(reflectArgs, reflect.ValueOf(keyAssignment))
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

// ChangeDeviceMuteIndicator will change device mute indicator
func ChangeDeviceMuteIndicator(deviceId string, muteIndicator int) uint8 {
	if device, ok := devices[deviceId]; ok {
		methodName := "UpdateMuteIndicator"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found or method is not supported for this device type")
			return 0
		} else {
			var reflectArgs []reflect.Value
			reflectArgs = append(reflectArgs, reflect.ValueOf(muteIndicator))
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

// UpdateDeviceLcdProfile will update device LCD
func UpdateDeviceLcdProfile(deviceId string, profile string) uint8 {
	if device, ok := devices[deviceId]; ok {
		methodName := "UpdateDeviceLcdProfile"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found or method is not supported for this device type")
			return 0
		} else {
			var reflectArgs []reflect.Value
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

// UpdateSpeedProfileBulk will update device speeds with a given serial number and array of device ids
func UpdateSpeedProfileBulk(deviceId string, channelIds []int, profile string) uint8 {
	if device, ok := devices[deviceId]; ok {
		methodName := "UpdateSpeedProfileBulk"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found or method is not supported for this device type")
			return 0
		} else {
			var reflectArgs []reflect.Value
			reflectArgs = append(reflectArgs, reflect.ValueOf(channelIds))
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

// UpdateLinkAdapter will update device LINK adapter. This is used only for iCUE Link System Hub
func UpdateLinkAdapter(deviceId string, channelId int, stripId int) uint8 {
	if device, ok := devices[deviceId]; ok {
		methodName := "UpdateLinkAdapter"
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

// UpdateLinkAdapterRgbProfile will update LINK adapter RGB profile
func UpdateLinkAdapterRgbProfile(deviceId string, channelId, adapterId int, profile string) uint8 {
	if device, ok := devices[deviceId]; ok {
		methodName := "UpdateLinkAdapterRgbProfile"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found")
			return 0
		} else {
			var reflectArgs []reflect.Value
			reflectArgs = append(reflectArgs, reflect.ValueOf(channelId))
			reflectArgs = append(reflectArgs, reflect.ValueOf(adapterId))
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

// UpdateLinkAdapterRgbProfileBulk will update LINK adapter bulk RGB profile
func UpdateLinkAdapterRgbProfileBulk(deviceId string, channelId int, profile string) uint8 {
	if device, ok := devices[deviceId]; ok {
		methodName := "UpdateLinkAdapterRgbProfileBulk"
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

// UpdateRgbProfileBulk will update device RGB profile on bulk selected devices
func UpdateRgbProfileBulk(deviceId string, channelIds []int, profile string) uint8 {
	if device, ok := devices[deviceId]; ok {
		methodName := "UpdateRgbProfileBulk"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found")
			return 0
		} else {
			var reflectArgs []reflect.Value
			reflectArgs = append(reflectArgs, reflect.ValueOf(channelIds))
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

// UpdateRgbProfileData will update device RGB profile data
func UpdateRgbProfileData(deviceId, profileName string, profile rgb.Profile) uint8 {
	if device, ok := devices[deviceId]; ok {
		methodName := "UpdateRgbProfileData"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found")
			return 0
		} else {
			var reflectArgs []reflect.Value
			reflectArgs = append(reflectArgs, reflect.ValueOf(profileName))
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

// UpdateHardwareRgbProfile will update device hardware RGB profile
func UpdateHardwareRgbProfile(deviceId string, hardwareLight int) uint8 {
	if device, ok := devices[deviceId]; ok {
		methodName := "UpdateHardwareRgbProfile"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found")
			return 0
		} else {
			var reflectArgs []reflect.Value
			reflectArgs = append(reflectArgs, reflect.ValueOf(hardwareLight))
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

// GetDeviceLedData will return device led data
func GetDeviceLedData(deviceId string) interface{} {
	if device, ok := devices[deviceId]; ok {
		methodName := "GetDeviceLedData"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found")
			return 0
		} else {
			results := method.Call(nil)
			if len(results) > 0 {
				val := results[0]
				return val.Interface()
			}
		}
	}
	return 0
}

// GetDevicesLedData will return led data for all devices
func GetDevicesLedData() interface{} {
	var leds []interface{}
	for _, device := range devices {
		methodName := "GetDeviceLedData"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found")
			continue
		} else {
			results := method.Call(nil)
			if len(results) > 0 {
				val := results[0]
				res := val.Interface()
				leds = append(leds, res)
			}
		}
	}
	return leds
}

// UpdateDeviceLedData will update device led data
func UpdateDeviceLedData(deviceId string, ledProfile led.Device) uint8 {
	if device, ok := devices[deviceId]; ok {
		methodName := "UpdateDeviceLedData"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found")
			return 0
		} else {
			var reflectArgs []reflect.Value
			reflectArgs = append(reflectArgs, reflect.ValueOf(ledProfile))
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

// ProcessGetKeyboardKey will get keyboard key data
func ProcessGetKeyboardKey(deviceId string, keyId int) interface{} {
	if device, ok := devices[deviceId]; ok {
		methodName := "ProcessGetKeyboardKey"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found")
			return nil
		} else {
			var reflectArgs []reflect.Value
			reflectArgs = append(reflectArgs, reflect.ValueOf(keyId))
			results := method.Call(reflectArgs)
			if len(results) > 0 {
				val := results[0]
				return val.Interface()
			}
		}
	}
	return nil
}

// ProcessGetKeyAssignmentTypes will get keyboard key assignment types
func ProcessGetKeyAssignmentTypes(deviceId string) interface{} {
	if device, ok := devices[deviceId]; ok {
		methodName := "ProcessGetKeyAssignmentTypes"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found")
			return nil
		} else {
			results := method.Call(nil)
			if len(results) > 0 {
				val := results[0]
				return val.Interface()
			}
		}
	}
	return nil
}

// ProcessGetKeyAssignmentModifiers will get keyboard key assignment modifiers
func ProcessGetKeyAssignmentModifiers(deviceId string) interface{} {
	if device, ok := devices[deviceId]; ok {
		methodName := "ProcessGetKeyAssignmentModifiers"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found")
			return nil
		} else {
			results := method.Call(nil)
			if len(results) > 0 {
				val := results[0]
				return val.Interface()
			}
		}
	}
	return nil
}

// ProcessGetKeyboardPerformance will get keyboard performance
func ProcessGetKeyboardPerformance(deviceId string) interface{} {
	if device, ok := devices[deviceId]; ok {
		methodName := "ProcessGetKeyboardPerformance"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found")
			return nil
		} else {
			results := method.Call(nil)
			if len(results) > 0 {
				val := results[0]
				return val.Interface()
			}
		}
	}
	return nil
}

// ProcessSetKeyboardPerformance will set keyboard performance
func ProcessSetKeyboardPerformance(deviceId string, performance common.KeyboardPerformanceData) uint8 {
	if device, ok := devices[deviceId]; ok {
		methodName := "ProcessSetKeyboardPerformance"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found")
			return 0
		} else {
			var reflectArgs []reflect.Value
			reflectArgs = append(reflectArgs, reflect.ValueOf(performance))
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

// GetDevices will return all available devices
func GetDevices() map[string]*common.Device {
	return devices
}

// GetDevicesEx will return all available devices with partial data
func GetDevicesEx() map[string]*common.Device {
	out := make(map[string]*common.Device)
	for _, device := range devices {
		out[device.Serial] = &common.Device{
			ProductType: 0,
			Product:     device.Product,
			Serial:      device.Serial,
			GetDevice:   device.GetDevice,
			Hidden:      device.Hidden,
		}
	}
	return out
}

// GetProducts will return all available products
func GetProducts() map[string]Product {
	return products
}

// GetTemperatureProbes will return a list of temperature probes
func GetTemperatureProbes() interface{} {
	var probes []interface{}
	for _, device := range devices {
		if device.ProductType == productTypeLinkHub ||
			device.ProductType == productTypeCC ||
			device.ProductType == productTypeCCXT ||
			device.ProductType == productTypeMemory ||
			device.ProductType == productTypeCPro ||
			device.ProductType == productTypeElite ||
			device.ProductType == productTypeXC7 ||
			device.ProductType == productTypePlatinum {
			methodName := "GetTemperatureProbes"
			method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
			if !method.IsValid() {
				logger.Log(logger.Fields{"method": methodName}).Warn("Method not found or method is not supported for this device type")
				continue
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

// ProcessGetRgbOverride will get rgb override data
func ProcessGetRgbOverride(deviceId string, channelId, subDeviceId int) interface{} {
	if device, ok := devices[deviceId]; ok {
		methodName := "ProcessGetRgbOverride"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found")
			return nil
		} else {
			var reflectArgs []reflect.Value
			reflectArgs = append(reflectArgs, reflect.ValueOf(channelId))
			reflectArgs = append(reflectArgs, reflect.ValueOf(subDeviceId))
			results := method.Call(reflectArgs)
			if len(results) > 0 {
				val := results[0]
				return val.Interface()
			}
		}
	}
	return nil
}

// ProcessGetLedData will get led data
func ProcessGetLedData(deviceId string, channelId, subDeviceId int) interface{} {
	if device, ok := devices[deviceId]; ok {
		methodName := "ProcessGetLedData"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found")
			return nil
		} else {
			var reflectArgs []reflect.Value
			reflectArgs = append(reflectArgs, reflect.ValueOf(channelId))
			reflectArgs = append(reflectArgs, reflect.ValueOf(subDeviceId))
			results := method.Call(reflectArgs)
			if len(results) > 0 {
				val := results[0]
				return val.Interface()
			}
		}
	}
	return nil
}

// ProcessSetLedData will set led data
func ProcessSetLedData(deviceId string, channelId, subDeviceId int, zoneColors map[int]rgb.Color, save bool) interface{} {
	if device, ok := devices[deviceId]; ok {
		methodName := "ProcessSetLedData"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found")
			return nil
		} else {
			var reflectArgs []reflect.Value
			reflectArgs = append(reflectArgs, reflect.ValueOf(channelId))
			reflectArgs = append(reflectArgs, reflect.ValueOf(subDeviceId))
			reflectArgs = append(reflectArgs, reflect.ValueOf(zoneColors))
			reflectArgs = append(reflectArgs, reflect.ValueOf(save))
			results := method.Call(reflectArgs)
			if len(results) > 0 {
				val := results[0]
				return val.Interface()
			}
		}
	}
	return nil
}

// ProcessSetOpenRgbIntegration will set OpenRGB integration
func ProcessSetOpenRgbIntegration(deviceId string, enabled bool) uint8 {
	if device, ok := devices[deviceId]; ok {
		methodName := "ProcessSetOpenRgbIntegration"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found")
			return 0
		} else {
			var reflectArgs []reflect.Value
			reflectArgs = append(reflectArgs, reflect.ValueOf(enabled))
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

// ProcessSetRgbOverride will set rgb override data
func ProcessSetRgbOverride(deviceId string, channelId, subDeviceId int, enabled bool, startColor, endColor rgb.Color, speed float64) uint8 {
	if device, ok := devices[deviceId]; ok {
		methodName := "ProcessSetRgbOverride"
		method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
		if !method.IsValid() {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found")
			return 0
		} else {
			var reflectArgs []reflect.Value
			reflectArgs = append(reflectArgs, reflect.ValueOf(channelId))
			reflectArgs = append(reflectArgs, reflect.ValueOf(subDeviceId))
			reflectArgs = append(reflectArgs, reflect.ValueOf(enabled))
			reflectArgs = append(reflectArgs, reflect.ValueOf(startColor))
			reflectArgs = append(reflectArgs, reflect.ValueOf(endColor))
			reflectArgs = append(reflectArgs, reflect.ValueOf(speed))
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

// GetDevice will return a device by device serial
func GetDevice(deviceId string) interface{} {
	if device, ok := devices[deviceId]; ok {
		return device.Instance
	}
	return nil
}

// InitManual will initialize device manually when plugged in
func InitManual(productId uint16, serial string) {
	var product = ProductEX{
		ProductId: 0,
		Path:      "",
		Serial:    "",
	}

	enum := hid.EnumFunc(func(info *hid.DeviceInfo) error {
		logger.Log(
			logger.Fields{
				"productId": info.ProductID,
				"interface": info.InterfaceNbr,
				"serial":    info.SerialNbr,
				"device":    info.ProductStr,
			},
		).Info("Processing device...")

		if slices.Contains(keyboards, info.ProductID) {
			interfaceId = 1 // Keyboard
		} else if slices.Contains(mouses, info.ProductID) {
			interfaceId = 1 // Mouse
		} else if slices.Contains(pads, info.ProductID) {
			interfaceId = 1 // Mousepad
		} else if slices.Contains(dongles, info.ProductID) {
			interfaceId = 1 // USB Dongle
		} else if slices.Contains(headsets, info.ProductID) {
			interfaceId = 3 // USB Headset
		} else if slices.Contains(headsets2, info.ProductID) {
			interfaceId = 4 // USB Headset
		} else {
			interfaceId = 0
		}

		if info.InterfaceNbr == interfaceId {
			devPath := info.Path
			if config.GetConfig().CheckDevicePermission {
				dev, err := os.Stat(devPath)
				if err != nil {
					logger.Log(logger.Fields{"error": err}).Error("Unable to get device stat info")
					return nil
				}
				filePerm := dev.Mode().Perm()
				if !slices.Contains(expectedPermissions, filePerm) {
					logger.Log(logger.Fields{"productId": info.ProductID, "path": info.Path, "device": info.ProductStr}).Warn("Invalid permissions")
					return nil
				}
			}

			if interfaceId == 1 || interfaceId == 3 || interfaceId == 4 {
				product = ProductEX{
					ProductId: info.ProductID,
					Path:      info.Path,
					Serial:    info.Path,
				}
			} else {
				if len(serial) == 0 {
					serial = info.SerialNbr
				}
				if len(serial) == 0 {
					// Devices with no serial, make serial based of productId
					serial = strconv.Itoa(int(info.ProductID))
				}
				product = ProductEX{
					ProductId: info.ProductID,
					Path:      info.Path,
					Serial:    serial,
				}
			}
		}

		return nil
	})

	// Enumerate all Corsair devices
	err := hid.Enumerate(vendorId, productId, enum)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": vendorId}).Fatal("Unable to enumerate devices")
	}

	if product.ProductId > 0 && len(product.Path) > 0 {
		initializeDevice(productId, product.Serial, product.Path)
	}
}

// Init will initialize all compatible Corsair devices in your system
func Init() {
	// Initialize general HID interface
	if err := hid.Init(); err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to initialize HID interface")
	}

	enum := hid.EnumFunc(func(info *hid.DeviceInfo) error {
		logger.Log(
			logger.Fields{
				"productId": info.ProductID,
				"interface": info.InterfaceNbr,
				"serial":    info.SerialNbr,
				"device":    info.ProductStr,
			},
		).Info("Processing device...")

		if slices.Contains(keyboards, info.ProductID) {
			interfaceId = 1 // Keyboard
		} else if slices.Contains(mouses, info.ProductID) {
			interfaceId = 1 // Mouse
		} else if slices.Contains(pads, info.ProductID) {
			interfaceId = 1 // Mousepad
		} else if slices.Contains(dongles, info.ProductID) {
			interfaceId = 1 // USB Dongle
		} else if slices.Contains(headsets, info.ProductID) {
			interfaceId = 3 // USB Headset
		} else if slices.Contains(headsets2, info.ProductID) {
			interfaceId = 4 // USB Headset
		} else {
			interfaceId = 0
		}

		if info.InterfaceNbr == interfaceId {
			devPath := info.Path
			if config.GetConfig().CheckDevicePermission {
				dev, err := os.Stat(devPath)
				if err != nil {
					logger.Log(logger.Fields{"error": err}).Error("Unable to get device stat info")
					return nil
				}
				filePerm := dev.Mode().Perm()
				if !slices.Contains(expectedPermissions, filePerm) {
					logger.Log(logger.Fields{"productId": info.ProductID, "path": info.Path, "device": info.ProductStr}).Warn("Invalid permissions")
					return nil
				}
			}
			base := filepath.Base(info.Path)
			p, _ := common.GetShortUSBDevPath(base)

			if interfaceId == 1 || interfaceId == 3 || interfaceId == 4 {
				products[info.Path] = Product{
					ProductId: info.ProductID,
					Path:      info.Path,
					DevPath:   p,
					Serial:    info.SerialNbr,
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
					DevPath:   p,
					Serial:    info.SerialNbr,
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

	// Memory
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

	// Legacy devices
	res := usb.Init(legacyDevices)
	if res != 0 {
		for _, device := range usb.GetDevices() {
			products[device.SerialNbr] = Product{
				ProductId: device.ProductID,
				Path:      device.Path,
			}
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
		initializeDevice(productId, key, productPath)
	}
}

// initializeDevice will initialize a device
func initializeDevice(productId uint16, key, productPath string) {
	switch productId {
	case 3135: // CORSAIR iCUE Link System Hub
		{
			go func(vendorId, productId uint16, serialId, productPath string) {
				dev := lsh.Init(vendorId, productId, serialId, productPath)
				if dev == nil {
					return
				}
				devices[dev.Serial] = &common.Device{
					ProductType: productTypeLinkHub,
					Product:     dev.Product,
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-device.svg",
					Instance:    dev,
				}
				devices[dev.Serial].GetDevice = GetDevice(dev.Serial)
			}(vendorId, productId, key, productPath)
		}
	case 3122, 3100: // CORSAIR iCUE COMMANDER Core
		{
			go func(vendorId, productId uint16, serialId, path string) {
				dev := cc.Init(vendorId, productId, serialId, path)
				if dev == nil {
					return
				}
				devices[dev.Serial] = &common.Device{
					ProductType: productTypeCC,
					Product:     dev.Product,
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-device.svg",
					Instance:    dev,
				}
				devices[dev.Serial].GetDevice = GetDevice(dev.Serial)
			}(vendorId, productId, key, productPath)
		}
	case 3114: // CORSAIR iCUE COMMANDER CORE XT
		{
			go func(vendorId, productId uint16, serialId string) {
				dev := ccxt.Init(vendorId, productId, serialId)
				if dev == nil {
					return
				}
				devices[dev.Serial] = &common.Device{
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
	case 3090, 3091, 3093:
		// Corsair H150i Platinum
		// Corsair H115i Platinum
		// Corsair H100i Platinum
		{
			go func(vendorId, productId uint16, path string) {
				dev := platinum.Init(vendorId, productId, path)
				if dev == nil {
					return
				}
				devices[dev.Serial] = &common.Device{
					ProductType: productTypePlatinum,
					Product:     dev.Product,
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-device.svg",
					Instance:    dev,
				}
				devices[dev.Serial].GetDevice = GetDevice(dev.Serial)
			}(vendorId, productId, productPath)
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
			go func(vendorId, productId uint16, path string) {
				dev := elite.Init(vendorId, productId, path)
				if dev == nil {
					return
				}
				devices[strconv.Itoa(int(productId))] = &common.Device{
					ProductType: productTypeElite,
					Product:     dev.Product,
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-device.svg",
					Instance:    dev,
				}
				devices[dev.Serial].GetDevice = GetDevice(dev.Serial)
			}(vendorId, productId, productPath)
		}
	case 3098: // CORSAIR Lighting Node CORE
		{
			go func(vendorId, productId uint16, serialId string) {
				dev := lncore.Init(vendorId, productId, serialId)
				if dev == nil {
					return
				}
				devices[dev.Serial] = &common.Device{
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
				devices[dev.Serial] = &common.Device{
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
				devices[dev.Serial] = &common.Device{
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
				devices[dev.Serial] = &common.Device{
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
				devices[dev.Serial] = &common.Device{
					ProductType: productTypeK65PM,
					Product:     dev.Product,
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-keyboard.svg",
					Instance:    dev,
				}
			}(vendorId, productId, key)
		}
	case 7094: // K70 Pro Mini
		{
			go func(vendorId, productId uint16, key string) {
				dev := k70pmWU.Init(vendorId, productId, key)
				if dev == nil {
					return
				}
				devices[dev.Serial] = &common.Device{
					ProductType: productTypeK70PMWU,
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
				devices[dev.Serial] = &common.Device{
					ProductType: productTypeK70Core,
					Product:     dev.Product,
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-keyboard.svg",
					Instance:    dev,
				}
			}(vendorId, productId, key)
		}
	case 11009: // K70 CORE TKL
		{
			go func(vendorId, productId uint16, key string) {
				dev := k70coretkl.Init(vendorId, productId, key)
				if dev == nil {
					return
				}
				devices[dev.Serial] = &common.Device{
					ProductType: productTypeK70CoreTkl,
					Product:     dev.Product,
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-keyboard.svg",
					Instance:    dev,
				}
			}(vendorId, productId, key)
		}
	case 11010: // K70 CORE TKL WIRELESS
		{
			go func(vendorId, productId uint16, key string) {
				dev := k70coretklWU.Init(vendorId, productId, key)
				if dev == nil {
					return
				}
				devices[dev.Serial] = &common.Device{
					ProductType: productTypeK70CoreTklWU,
					Product:     dev.Product,
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-keyboard.svg",
					Instance:    dev,
				}
			}(vendorId, productId, key)
		}
	case 11028: // K70 CORE TKL WIRELESS
		{
			go func(vendorId, productId uint16, key string) {
				dev := k70protkl.Init(vendorId, productId, key)
				if dev == nil {
					return
				}
				devices[dev.Serial] = &common.Device{
					ProductType: productTypeK70ProTkl,
					Product:     dev.Product,
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-keyboard.svg",
					Instance:    dev,
				}
			}(vendorId, productId, key)
		}
	case 7097, 7027: // K70 RGB TKL
		{
			go func(vendorId, productId uint16, key string) {
				dev := k70rgbtklcs.Init(vendorId, productId, key)
				if dev == nil {
					return
				}
				devices[dev.Serial] = &common.Device{
					ProductType: productTypeK70RgbTkl,
					Product:     dev.Product,
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-keyboard.svg",
					Instance:    dev,
				}
			}(vendorId, productId, key)
		}
	case 6973: // K55 RGB
		{
			go func(vendorId, productId uint16, key string) {
				dev := k55.Init(vendorId, productId, key)
				if dev == nil {
					return
				}
				devices[dev.Serial] = &common.Device{
					ProductType: productTypeK55,
					Product:     dev.Product,
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-keyboard.svg",
					Instance:    dev,
				}
			}(vendorId, productId, key)
		}
	case 6957: // K95 PLATINUM
		{
			go func(vendorId, productId uint16, key string) {
				dev := k95platinum.Init(vendorId, productId, key)
				if dev == nil {
					return
				}
				devices[dev.Serial] = &common.Device{
					ProductType: productTypeK95Platinum,
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
				devices[dev.Serial] = &common.Device{
					ProductType: productTypeK55Core,
					Product:     dev.Product,
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-keyboard.svg",
					Instance:    dev,
				}
			}(vendorId, productId, key)
		}
	case 7076: // K55 PRO RGB
		{
			go func(vendorId, productId uint16, key string) {
				dev := k55pro.Init(vendorId, productId, key)
				if dev == nil {
					return
				}
				devices[dev.Serial] = &common.Device{
					ProductType: productTypeK55Pro,
					Product:     dev.Product,
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-keyboard.svg",
					Instance:    dev,
				}
			}(vendorId, productId, key)
		}
	case 7073: // K55 RGB PRO XT
		{
			go func(vendorId, productId uint16, key string) {
				dev := k55proXT.Init(vendorId, productId, key)
				if dev == nil {
					return
				}
				devices[dev.Serial] = &common.Device{
					ProductType: productTypeK55ProXT,
					Product:     dev.Product,
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-keyboard.svg",
					Instance:    dev,
				}
			}(vendorId, productId, key)
		}
	case 7072: // K60 RGB PRO
		{
			go func(vendorId, productId uint16, key string) {
				dev := k60rgbpro.Init(vendorId, productId, key)
				if dev == nil {
					return
				}
				devices[dev.Serial] = &common.Device{
					ProductType: productTypeK60RgbPro,
					Product:     dev.Product,
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-keyboard.svg",
					Instance:    dev,
				}
			}(vendorId, productId, key)
		}
	case 7110, 7091, 7124: // K70 RGB PRO
		{
			go func(vendorId, productId uint16, key string) {
				dev := k70pro.Init(vendorId, productId, key)
				if dev == nil {
					return
				}
				devices[dev.Serial] = &common.Device{
					ProductType: productTypeK70Pro,
					Product:     dev.Product,
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-keyboard.svg",
					Instance:    dev,
				}
			}(vendorId, productId, key)
		}
	case 6985, 6997, 7019: // K70 RGB MK2
		{
			go func(vendorId, productId uint16, key string) {
				dev := k70mk2.Init(vendorId, productId, key)
				if dev == nil {
					return
				}
				devices[dev.Serial] = &common.Device{
					ProductType: productTypeK70MK2,
					Product:     dev.Product,
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-keyboard.svg",
					Instance:    dev,
				}
			}(vendorId, productId, key)
		}
	case 11024, 11025: // K65 PLUS WIRELESS USB
		{
			go func(vendorId, productId uint16, key string) {
				dev := k65plusWU.Init(vendorId, productId, key)
				if dev == nil {
					return
				}
				devices[dev.Serial] = &common.Device{
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
				devices[dev.Serial] = &common.Device{
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
				devices[dev.Serial] = &common.Device{
					ProductType: productTypeK100Air,
					Product:     dev.Product,
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-keyboard.svg",
					Instance:    dev,
				}
			}(vendorId, productId, key)
		}
	case 7036, 7109, 7037: // K100 RGB
		{
			go func(vendorId, productId uint16, key string) {
				dev := k100.Init(vendorId, productId, key)
				if dev == nil {
					return
				}
				devices[dev.Serial] = &common.Device{
					ProductType: productTypeK100,
					Product:     dev.Product,
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-keyboard.svg",
					Instance:    dev,
				}
			}(vendorId, productId, key)
		}
	case 2660, 2667: // Headset dongle
		{
			go func(vendorId, productId uint16, key string) {
				dev := headsetdongle.Init(vendorId, productId, key, devices)
				if dev == nil {
					return
				}
				devices[dev.Serial] = &common.Device{
					ProductType: productTypeIronClawRgbW,
					Product:     "HEADSET DONGLE",
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-dongle.svg",
					Instance:    dev,
					Hidden:      true,
				}
				for _, value := range dev.Devices {
					if isUSBConnected(value.ProductId) {
						continue
					}

					switch value.ProductId {
					case 2658:
						{
							d := virtuosorgbXTW.Init(
								value.VendorId,
								productId,
								value.ProductId,
								dev.GetDevice(),
								value.Endpoint,
								value.Serial,
							)
							devices[d.Serial] = &common.Device{
								ProductType: productTypeVirtuosoXTW,
								Product:     "VIRTUOSO RGB WIRELESS XT",
								Serial:      d.Serial,
								Firmware:    d.Firmware,
								Image:       "icon-headphone.svg",
								Instance:    d,
							}
							dev.AddPairedDevice(value.ProductId, d, devices[d.Serial])
						}
					case 2665:
						{
							d := hs80rgbW.Init(
								value.VendorId,
								productId,
								value.ProductId,
								dev.GetDevice(),
								value.Endpoint,
								value.Serial,
							)
							devices[d.Serial] = &common.Device{
								ProductType: productTypeHS80RGBW,
								Product:     "HS80 RGB WIRELESS",
								Serial:      d.Serial,
								Firmware:    d.Firmware,
								Image:       "icon-headphone.svg",
								Instance:    d,
							}
							dev.AddPairedDevice(value.ProductId, d, devices[d.Serial])
						}
					default:
						logger.Log(logger.Fields{"productId": value.ProductId}).Warn("Unsupported device detected")
					}
				}
				dev.InitAvailableDevices()
			}(vendorId, productId, key)
		}
	case 10754: // CORSAIR VIRTUOSO MAX WIRELESS
		{
			go func(vendorId, productId uint16, key string) {
				dev := virtuosomaxdongle.Init(vendorId, productId, key)
				if dev == nil {
					return
				}
				devices[dev.Serial] = &common.Device{
					ProductType: productTypeVirtuosoMAXW,
					Product:     "HEADSET DONGLE",
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-dongle.svg",
					Instance:    dev,
					Hidden:      true,
				}

				switch dev.Devices.ProductId {
				case 10752:
					{
						d := virtuosomaxW.Init(
							dev.Devices.VendorId,
							productId,
							dev.Devices.ProductId,
							dev.GetDevice(),
							dev.Devices.Endpoint,
							dev.Devices.Serial,
						)
						devices[d.Serial] = &common.Device{
							ProductType: productTypeVirtuosoMAXW,
							Product:     "VIRTUOSO MAX",
							Serial:      d.Serial,
							Firmware:    d.Firmware,
							Image:       "icon-headphone.svg",
							Instance:    d,
						}
						dev.AddPairedDevice(dev.Devices.ProductId, d)
					}
				default:
					logger.Log(logger.Fields{"productId": dev.Devices.ProductId}).Warn("Unsupported device detected")
				}
				dev.InitAvailableDevices()
			}(vendorId, productId, key)
		}
	case 2711: // CORSAIR HS80 MAX WIRELESS
		{
			go func(vendorId, productId uint16, key string) {
				dev := hs80maxdongle.Init(vendorId, productId, key)
				if dev == nil {
					return
				}
				devices[dev.Serial] = &common.Device{
					ProductType: productTypeHS80MAXW,
					Product:     "HEADSET DONGLE",
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-dongle.svg",
					Instance:    dev,
					Hidden:      true,
				}
				switch dev.Devices.ProductId {
				case 2710:
					{
						d := hs80maxW.Init(
							dev.Devices.VendorId,
							productId,
							dev.Devices.ProductId,
							dev.GetDevice(),
							dev.Devices.Endpoint,
							dev.Devices.Serial,
						)
						devices[d.Serial] = &common.Device{
							ProductType: productTypeHS80MAXW,
							Product:     "HS80 MAX WIRELESS",
							Serial:      d.Serial,
							Firmware:    d.Firmware,
							Image:       "icon-headphone.svg",
							Instance:    d,
						}
						dev.AddPairedDevice(dev.Devices.ProductId, d)
					}
				default:
					logger.Log(logger.Fields{"productId": dev.Devices.ProductId}).Warn("Unsupported device detected")
				}
				dev.InitAvailableDevices()
			}(vendorId, productId, key)
		}
	case 7132, 7078, 11008: // Corsair SLIPSTREAM WIRELESS USB Receiver
		{
			go func(vendorId, productId uint16, key string) {
				dev := slipstream.Init(vendorId, productId, key, devices)
				if dev == nil {
					return
				}
				devices[dev.Serial] = &common.Device{
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
					case 7163: // M55
						{
							d := m55W.Init(
								value.VendorId,
								productId,
								value.ProductId,
								dev.GetDevice(),
								value.Endpoint,
								value.Serial,
							)
							devices[d.Serial] = &common.Device{
								ProductType: productTypeM55W,
								Product:     "M55 WIRELESS",
								Serial:      d.Serial,
								Firmware:    d.Firmware,
								Image:       "icon-mouse.svg",
								Instance:    d,
							}
							dev.AddPairedDevice(value.ProductId, d, devices[d.Serial])
						}
					case 7131: // SCIMITAR
						{
							d := scimitarW.Init(
								value.VendorId,
								productId,
								value.ProductId,
								dev.GetDevice(),
								value.Endpoint,
								value.Serial,
							)
							devices[d.Serial] = &common.Device{
								ProductType: productTypeScimitarRgbEliteW,
								Product:     "SCIMITAR RGB ELITE",
								Serial:      d.Serial,
								Firmware:    d.Firmware,
								Image:       "icon-mouse.svg",
								Instance:    d,
							}
							dev.AddPairedDevice(value.ProductId, d, devices[d.Serial])
						}
					case 11042: // CORSAIR SCIMITAR ELITE WIRELESS SE
						{
							d := scimitarSEW.Init(
								value.VendorId,
								productId,
								value.ProductId,
								dev.GetDevice(),
								value.Endpoint,
								value.Serial,
							)
							devices[d.Serial] = &common.Device{
								ProductType: productTypeScimitarRgbEliteSEW,
								Product:     "SCIMITAR ELITE SE",
								Serial:      d.Serial,
								Firmware:    d.Firmware,
								Image:       "icon-mouse.svg",
								Instance:    d,
							}
							dev.AddPairedDevice(value.ProductId, d, devices[d.Serial])
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
							devices[d.Serial] = &common.Device{
								ProductType: productTypeNightsabreW,
								Product:     "NIGHTSABRE",
								Serial:      d.Serial,
								Firmware:    d.Firmware,
								Image:       "icon-mouse.svg",
								Instance:    d,
							}
							dev.AddPairedDevice(value.ProductId, d, devices[d.Serial])
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
							devices[d.Serial] = &common.Device{
								ProductType: productTypeK100AirW,
								Product:     "K100 AIR",
								Serial:      d.Serial,
								Firmware:    d.Firmware,
								Image:       "icon-keyboard.svg",
								Instance:    d,
							}
							dev.AddPairedDevice(value.ProductId, d, devices[d.Serial])
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
							devices[d.Serial] = &common.Device{
								ProductType: productTypeIronClawRgbW,
								Product:     "IRONCLAW RGB",
								Serial:      d.Serial,
								Firmware:    d.Firmware,
								Image:       "icon-mouse.svg",
								Instance:    d,
							}
							dev.AddPairedDevice(value.ProductId, d, devices[d.Serial])
						}
					case 7038: // DARK CORE RGB PRO SE WIRELESS
						{
							d := darkcorergbproseW.Init(
								value.VendorId,
								productId,
								value.ProductId,
								dev.GetDevice(),
								value.Endpoint,
								value.Serial,
							)
							devices[d.Serial] = &common.Device{
								ProductType: productTypeDarkCoreRgbProSEW,
								Product:     "DARK CORE RGB PRO SE",
								Serial:      d.Serial,
								Firmware:    d.Firmware,
								Image:       "icon-mouse.svg",
								Instance:    d,
							}
							dev.AddPairedDevice(value.ProductId, d, devices[d.Serial])
						}
					case 7040: // DARK CORE RGB PRO WIRELESS
						{
							d := darkcorergbproW.Init(
								value.VendorId,
								productId,
								value.ProductId,
								dev.GetDevice(),
								value.Endpoint,
								value.Serial,
							)
							devices[d.Serial] = &common.Device{
								ProductType: productTypeDarkCoreRgbProW,
								Product:     "DARK CORE RGB PRO",
								Serial:      d.Serial,
								Firmware:    d.Firmware,
								Image:       "icon-mouse.svg",
								Instance:    d,
							}
							dev.AddPairedDevice(value.ProductId, d, devices[d.Serial])
						}
					case 11016: // M75 WIRELESS
						{
							d := m75W.Init(
								value.VendorId,
								productId,
								value.ProductId,
								dev.GetDevice(),
								value.Endpoint,
								value.Serial,
							)
							devices[d.Serial] = &common.Device{
								ProductType: productTypeM75W,
								Product:     "M75 WIRELESS",
								Serial:      d.Serial,
								Firmware:    d.Firmware,
								Image:       "icon-mouse.svg",
								Instance:    d,
							}
							dev.AddPairedDevice(value.ProductId, d, devices[d.Serial])
						}
					case 7154: // M75 AIR WIRELESS
						{
							d := m75AirW.Init(
								value.VendorId,
								productId,
								value.ProductId,
								dev.GetDevice(),
								value.Endpoint,
								value.Serial,
							)
							devices[d.Serial] = &common.Device{
								ProductType: productTypeM75AirW,
								Product:     "M75 AIR WIRELESS",
								Serial:      d.Serial,
								Firmware:    d.Firmware,
								Image:       "icon-mouse.svg",
								Instance:    d,
							}
							dev.AddPairedDevice(value.ProductId, d, devices[d.Serial])
						}
					case 7006: // HARPOON RGB WIRELESS
						{
							d := harpoonW.Init(
								value.VendorId,
								productId,
								value.ProductId,
								dev.GetDevice(),
								value.Endpoint,
								value.Serial,
							)
							devices[d.Serial] = &common.Device{
								ProductType: productTypeHarpoonRgbW,
								Product:     "HARPOON RGB",
								Serial:      d.Serial,
								Firmware:    d.Firmware,
								Image:       "icon-mouse.svg",
								Instance:    d,
							}
							dev.AddPairedDevice(value.ProductId, d, devices[d.Serial])
						}
					case 7090: // CORSAIR DARKSTAR RGB WIRELESS Gaming Mouse
						{
							d := darkstarW.Init(
								value.VendorId,
								productId,
								value.ProductId,
								dev.GetDevice(),
								value.Endpoint,
								value.Serial,
							)
							devices[d.Serial] = &common.Device{
								ProductType: productTypeDarkstarW,
								Product:     "DARKSTAR",
								Serial:      d.Serial,
								Firmware:    d.Firmware,
								Image:       "icon-mouse.svg",
								Instance:    d,
							}
							dev.AddPairedDevice(value.ProductId, d, devices[d.Serial])
						}
					case 7093, 7126: // M65 RGB ULTRA WIRELESS Gaming Mouse
						{
							d := m65rgbultraW.Init(
								value.VendorId,
								productId,
								value.ProductId,
								dev.GetDevice(),
								value.Endpoint,
								value.Serial,
							)
							devices[d.Serial] = &common.Device{
								ProductType: productTypeM65RgbUltraW,
								Product:     "M65 RGB ULTRA",
								Serial:      d.Serial,
								Firmware:    d.Firmware,
								Image:       "icon-mouse.svg",
								Instance:    d,
							}
							dev.AddPairedDevice(value.ProductId, d, devices[d.Serial])
						}
					case 11010: // K70 CORE TKL WIRELESS
						{
							d := k70coretklW.Init(
								value.VendorId,
								productId,
								value.ProductId,
								dev.GetDevice(),
								value.Endpoint,
								value.Serial,
								dev.Serial,
							)
							devices[d.Serial] = &common.Device{
								ProductType: productTypeK70CoreTklW,
								Product:     "K70 CORE TKL",
								Serial:      d.Serial,
								Firmware:    d.Firmware,
								Image:       "icon-keyboard.svg",
								Instance:    d,
							}
							dev.AddPairedDevice(value.ProductId, d, devices[d.Serial])
						}
					case 7064: // CORSAIR SABRE RGB PRO WIRELESS Gaming Mouse
						{
							d := sabrergbproW.Init(
								value.VendorId,
								productId,
								value.ProductId,
								dev.GetDevice(),
								value.Endpoint,
								value.Serial,
							)
							devices[d.Serial] = &common.Device{
								ProductType: productTypeSabreRgbProW,
								Product:     "SABRE RGB PRO",
								Serial:      d.Serial,
								Firmware:    d.Firmware,
								Image:       "icon-mouse.svg",
								Instance:    d,
							}
							dev.AddPairedDevice(value.ProductId, d, devices[d.Serial])
						}
					case 7094: // K70 PRO MINI
						{
							d := k70pmW.Init(
								value.VendorId,
								productId,
								value.ProductId,
								dev.GetDevice(),
								value.Endpoint,
								value.Serial,
							)
							devices[d.Serial] = &common.Device{
								ProductType: productTypeK70PMW,
								Product:     "K70 PRO MINI",
								Serial:      d.Serial,
								Firmware:    d.Firmware,
								Image:       "icon-keyboard.svg",
								Instance:    d,
							}
							dev.AddPairedDevice(value.ProductId, d, devices[d.Serial])
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
				devices[dev.Serial] = &common.Device{
					ProductType: productTypeST100,
					Product:     dev.Product,
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-headphone-stand.svg",
					Instance:    dev,
				}
			}(vendorId, productId, key)
		}
	case 7067, 7113: // Corsair MM700 RGB Gaming Mousepad
		{
			go func(vendorId, productId uint16, key string) {
				dev := mm700.Init(vendorId, productId, key)
				if dev == nil {
					return
				}
				devices[dev.Serial] = &common.Device{
					ProductType: productTypeMM700,
					Product:     dev.Product,
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-mousepad.svg",
					Instance:    dev,
				}
			}(vendorId, productId, key)
		}
	case 6971: // Corsair Gaming MM800 RGB POLARIS
		{
			go func(vendorId, productId uint16, serialId string) {
				dev := mm800.Init(vendorId, productId, serialId)
				if dev == nil {
					return
				}
				devices[dev.Serial] = &common.Device{
					ProductType: productTypeMM800,
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
				devices[dev.Serial] = &common.Device{
					ProductType: productTypeLT100,
					Product:     dev.Product,
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-towers.svg",
					Instance:    dev,
				}
			}(vendorId, productId, key, productPath)
		}
	case 7198, 7203, 7199, 7173, 7174, 7175, 7176, 7181, 7180, 7207:
		// Corsair HX1000i Power Supply
		// Corsair HX1200i Power Supply
		// Corsair HX1500i Power Supply
		// Corsair HX750i Power Supply
		// Corsair HX850i Power Supply
		// Corsair HX1000i Power Supply
		// Corsair HX1200i Power Supply
		// Corsair RM1000i Power Supply
		// Corsair RM850i Power Supply
		// Corsair HX1200i Power Supply
		{
			go func(vendorId, productId uint16, key string) {
				dev := psuhid.Init(vendorId, productId, key)
				if dev == nil {
					return
				}
				devices[dev.Serial] = &common.Device{
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
				devices[dev.Serial] = &common.Device{
					ProductType: productTypeKatarPro,
					Product:     dev.Product,
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-mouse.svg",
					Instance:    dev,
				}
			}(vendorId, productId, key)
		}
	case 7084: // Corsair KATAR PRO XT Gaming Mouse
		{
			go func(vendorId, productId uint16, key string) {
				dev := katarproxt.Init(vendorId, productId, key)
				if dev == nil {
					return
				}
				devices[dev.Serial] = &common.Device{
					ProductType: productTypeKatarProXT,
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
				devices[dev.Serial] = &common.Device{
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
				devices[dev.Serial] = &common.Device{
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
				devices[dev.Serial] = &common.Device{
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
				devices[dev.Serial] = &common.Device{
					ProductType: productTypeScimitarRgbElite,
					Product:     dev.Product,
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-mouse.svg",
					Instance:    dev,
				}
			}(vendorId, productId, key)
		}
	case 6974:
		{
			go func(vendorId, productId uint16, key string) {
				dev := scimitarprorgb.Init(vendorId, productId, key)
				if dev == nil {
					return
				}
				devices[dev.Serial] = &common.Device{
					ProductType: productTypeScimitarProRgb,
					Product:     dev.Product,
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-mouse.svg",
					Instance:    dev,
				}
			}(vendorId, productId, key)
		}
	case 7051: // CORSAIR SCIMITAR RGB ELITE
		{
			go func(vendorId, productId uint16, key string) {
				dev := scimitarrgbelite.Init(vendorId, productId, key)
				if dev == nil {
					return
				}
				devices[dev.Serial] = &common.Device{
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
				devices[dev.Serial] = &common.Device{
					ProductType: productTypeScimitarRgbEliteWU,
					Product:     dev.Product,
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-mouse.svg",
					Instance:    dev,
				}
			}(vendorId, productId, key)
		}
	case 11042: // CORSAIR SCIMITAR ELITE WIRELESS SE
		{
			go func(vendorId, productId uint16, key string) {
				dev := scimitarSEWU.Init(vendorId, productId, key)
				if dev == nil {
					return
				}
				devices[dev.Serial] = &common.Device{
					ProductType: productTypeScimitarRgbEliteSEWU,
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
				devices[dev.Serial] = &common.Device{
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
				devices[dev.Serial] = &common.Device{
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
				if dev == nil {
					return
				}
				devices[dev.Serial] = &common.Device{
					ProductType: productTypeKatarProW,
					Product:     "KATAR PRO",
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-mouse.svg",
					Instance:    dev,
				}
			}(vendorId, productId, key)
		}
	case 7038: // CORSAIR DARK CORE RGB PRO SE Gaming Mouse
		{
			go func(vendorId, productId uint16, key string) {
				dev := darkcorergbproseWU.Init(vendorId, productId, key)
				if dev == nil {
					return
				}
				devices[dev.Serial] = &common.Device{
					ProductType: productTypeDarkCoreRgbProSEWU,
					Product:     dev.Product,
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-mouse.svg",
					Instance:    dev,
				}
			}(vendorId, productId, key)
		}
	case 7040: // CORSAIR DARK CORE RGB PRO Gaming Mouse
		{
			go func(vendorId, productId uint16, key string) {
				dev := darkcorergbproWU.Init(vendorId, productId, key)
				if dev == nil {
					return
				}
				devices[dev.Serial] = &common.Device{
					ProductType: productTypeDarkCoreRgbProWU,
					Product:     dev.Product,
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-mouse.svg",
					Instance:    dev,
				}
			}(vendorId, productId, key)
		}
	case 7152: // CORSAIR M75 Gaming Mouse
		{
			go func(vendorId, productId uint16, key string) {
				dev := m75.Init(vendorId, productId, key)
				if dev == nil {
					return
				}
				devices[dev.Serial] = &common.Device{
					ProductType: productTypeM75,
					Product:     dev.Product,
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-mouse.svg",
					Instance:    dev,
				}
			}(vendorId, productId, key)
		}
	case 11016: // CORSAIR M75 WIRELESS Gaming Mouse
		{
			go func(vendorId, productId uint16, key string) {
				dev := m75WU.Init(vendorId, productId, key)
				if dev == nil {
					return
				}
				devices[dev.Serial] = &common.Device{
					ProductType: productTypeM75WU,
					Product:     dev.Product,
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-mouse.svg",
					Instance:    dev,
				}
			}(vendorId, productId, key)
		}
	case 7154: // CORSAIR M75 AIR WIRELESS Gaming Mouse
		{
			go func(vendorId, productId uint16, key string) {
				dev := m75AirWU.Init(vendorId, productId, key)
				if dev == nil {
					return
				}
				devices[dev.Serial] = &common.Device{
					ProductType: productTypeM75AirWU,
					Product:     dev.Product,
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-mouse.svg",
					Instance:    dev,
				}
			}(vendorId, productId, key)
		}
	case 7070: // CORSAIR M65 RGB ULTRA Gaming Mouse
		{
			go func(vendorId, productId uint16, key string) {
				dev := m65rgbultra.Init(vendorId, productId, key)
				if dev == nil {
					return
				}
				devices[dev.Serial] = &common.Device{
					ProductType: productTypeM65RgbUltra,
					Product:     dev.Product,
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-mouse.svg",
					Instance:    dev,
				}
			}(vendorId, productId, key)
		}
	case 7093, 7126: // CORSAIR M65 RGB ULTRA WIRELESS Gaming Mouse
		{
			go func(vendorId, productId uint16, key string) {
				dev := m65rgbultraWU.Init(vendorId, productId, key)
				if dev == nil {
					return
				}
				devices[dev.Serial] = &common.Device{
					ProductType: productTypeM65RgbUltraWU,
					Product:     dev.Product,
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-mouse.svg",
					Instance:    dev,
				}
			}(vendorId, productId, key)
		}
	case 7029: // CORSAIR HARPOON RGB PRO Gaming Mouse
		{
			go func(vendorId, productId uint16, key string) {
				dev := harpoonrgbpro.Init(vendorId, productId, key)
				if dev == nil {
					return
				}
				devices[dev.Serial] = &common.Device{
					ProductType: productTypeHarpoonRgbPro,
					Product:     dev.Product,
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-mouse.svg",
					Instance:    dev,
				}
			}(vendorId, productId, key)
		}
	case 7006: // CORSAIR HARPOON Gaming Mouse
		{
			go func(vendorId, productId uint16, key string) {
				dev := harpoonWU.Init(vendorId, productId, key)
				if dev == nil {
					return
				}
				devices[dev.Serial] = &common.Device{
					ProductType: productTypeHarpoonRgbWU,
					Product:     dev.Product,
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-mouse.svg",
					Instance:    dev,
				}
			}(vendorId, productId, key)
		}
	case 7004: // CORSAIR NIGHTSWORD RGB Gaming Mouse
		{
			go func(vendorId, productId uint16, key string) {
				dev := nightswordrgb.Init(vendorId, productId, key)
				if dev == nil {
					return
				}
				devices[dev.Serial] = &common.Device{
					ProductType: productTypeNightswordRgb,
					Product:     dev.Product,
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-mouse.svg",
					Instance:    dev,
				}
			}(vendorId, productId, key)
		}
	case 7064: // CORSAIR SABRE RGB PRO WIRELESS Gaming Mouse
		{
			go func(vendorId, productId uint16, key string) {
				dev := sabrergbproWU.Init(vendorId, productId, key)
				if dev == nil {
					return
				}
				devices[dev.Serial] = &common.Device{
					ProductType: productTypeSabreRgbProWU,
					Product:     dev.Product,
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-mouse.svg",
					Instance:    dev,
				}
			}(vendorId, productId, key)
		}
	case 7033: // CORSAIR SABRE RGB PRO
		{
			go func(vendorId, productId uint16, key string) {
				dev := sabrergbpro.Init(vendorId, productId, key)
				if dev == nil {
					return
				}
				devices[dev.Serial] = &common.Device{
					ProductType: productTypeSabreRgbPro,
					Product:     dev.Product,
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-mouse.svg",
					Instance:    dev,
				}
			}(vendorId, productId, key)
		}
	case 7090: // CORSAIR DARKSTAR RGB WIRELESS Gaming Mouse
		{
			go func(vendorId, productId uint16, key string) {
				dev := darkstarWU.Init(vendorId, productId, key)
				if dev == nil {
					return
				}
				devices[dev.Serial] = &common.Device{
					ProductType: productTypeDarkstarWU,
					Product:     dev.Product,
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-mouse.svg",
					Instance:    dev,
				}
			}(vendorId, productId, key)
		}
	case 2658: // VIRTUOSO RGB WIRELESS XT
		{
			go func(vendorId, productId uint16, key string) {
				dev := virtuosorgbXTWU.Init(vendorId, productId, key)
				if dev == nil {
					return
				}
				devices[dev.Serial] = &common.Device{
					ProductType: productTypeVirtuosoXTWU,
					Product:     dev.Product,
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-headphone.svg",
					Instance:    dev,
				}
			}(vendorId, productId, key)
		}
	case 2696: // Corsair HS80 RGB USB Gaming Headset
		{
			go func(vendorId, productId uint16, path string) {
				dev := hs80rgb.Init(vendorId, productId, path)
				if dev == nil {
					return
				}
				devices[dev.Serial] = &common.Device{
					ProductType: productTypeHS80RGB,
					Product:     dev.Product,
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-headphone.svg",
					Instance:    dev,
				}
			}(vendorId, productId, productPath)
		}
	case 7054: // CORSAIR iCUE NEXUS
		{
			go func(vendorId, productId uint16, serialId string) {
				dev := nexus.Init(vendorId, productId, serialId)
				if dev == nil {
					return
				}
				devices[dev.Serial] = &common.Device{
					ProductType: productTypeNexus,
					Product:     dev.Product,
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-device.svg",
					Instance:    dev,
				}
				devices[dev.Serial].GetDevice = GetDevice(dev.Serial)
			}(vendorId, productId, key)
		}
	case 0: // Memory
		{
			go func(serialId string) {
				dev := memory.Init(serialId, "Memory")
				if dev != nil {
					devices[dev.Serial] = &common.Device{
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
		return
	}
}
