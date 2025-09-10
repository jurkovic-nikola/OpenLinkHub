package devices

import (
	"OpenLinkHub/src/cluster"
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/devices/cc"
	"OpenLinkHub/src/devices/ccxt"
	"OpenLinkHub/src/devices/cpro"
	"OpenLinkHub/src/devices/darkcorergbproWU"
	"OpenLinkHub/src/devices/darkcorergbproseWU"
	"OpenLinkHub/src/devices/darkcorergbseWU"
	"OpenLinkHub/src/devices/darkcorergbsesongle"
	"OpenLinkHub/src/devices/darkstarWU"
	"OpenLinkHub/src/devices/elite"
	"OpenLinkHub/src/devices/harpoonWU"
	"OpenLinkHub/src/devices/harpoonrgbpro"
	"OpenLinkHub/src/devices/headsetdongle"
	"OpenLinkHub/src/devices/hs80maxdongle"
	"OpenLinkHub/src/devices/hs80rgb"
	"OpenLinkHub/src/devices/hydro"
	"OpenLinkHub/src/devices/ironclaw"
	"OpenLinkHub/src/devices/ironclawWU"
	"OpenLinkHub/src/devices/k100"
	"OpenLinkHub/src/devices/k100airWU"
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
	"OpenLinkHub/src/devices/k70coretklWU"
	"OpenLinkHub/src/devices/k70max"
	"OpenLinkHub/src/devices/k70mk2"
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
	"OpenLinkHub/src/devices/m55rgbpro"
	"OpenLinkHub/src/devices/m65rgbultra"
	"OpenLinkHub/src/devices/m65rgbultraWU"
	"OpenLinkHub/src/devices/m75"
	"OpenLinkHub/src/devices/m75AirWU"
	"OpenLinkHub/src/devices/m75WU"
	"OpenLinkHub/src/devices/makr75WU"
	"OpenLinkHub/src/devices/memory"
	"OpenLinkHub/src/devices/mm700"
	"OpenLinkHub/src/devices/mm800"
	"OpenLinkHub/src/devices/nexus"
	"OpenLinkHub/src/devices/nightsabreWU"
	"OpenLinkHub/src/devices/nightswordrgb"
	"OpenLinkHub/src/devices/platinum"
	"OpenLinkHub/src/devices/psuhid"
	"OpenLinkHub/src/devices/sabrergbpro"
	"OpenLinkHub/src/devices/sabrergbproWU"
	"OpenLinkHub/src/devices/scimitar"
	"OpenLinkHub/src/devices/scimitarSEWU"
	"OpenLinkHub/src/devices/scimitarWU"
	"OpenLinkHub/src/devices/scimitarprorgb"
	"OpenLinkHub/src/devices/scimitarrgb"
	"OpenLinkHub/src/devices/scimitarrgbelite"
	"OpenLinkHub/src/devices/slipstream"
	"OpenLinkHub/src/devices/st100"
	"OpenLinkHub/src/devices/virtuosomaxdongle"
	"OpenLinkHub/src/devices/virtuosorgbXTWU"
	"OpenLinkHub/src/devices/xc7"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/metrics"
	"OpenLinkHub/src/smbus"
	"OpenLinkHub/src/usb"
	"github.com/sstallion/go-hid"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"sync"
)

type deviceRegister func(vid, pid uint16, serial, path string) *common.Device
type deviceRegisterEx func(vid, pid uint16, serial, path string, callback func(device *common.Device)) *common.Device

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
	mutex               sync.Mutex
	expectedPermissions        = []os.FileMode{os.FileMode(0600), os.FileMode(0660)}
	vendorId            uint16 = 6940 // Corsair
	interfaceId                = 0
	devices                    = make(map[string]*common.Device)
	products                   = make(map[string]Product)
	keyboards                  = []uint16{7127, 7165, 7166, 7110, 7083, 11024, 11025, 11015, 7109, 7091, 7124, 7036, 7037, 6985, 6997, 7019, 11009, 11010, 11028, 7097, 7027, 7076, 7073, 6973, 6957, 7072, 7094, 7104, 11012}
	mouses                     = []uint16{7059, 7005, 6988, 7096, 7139, 7131, 11011, 7024, 7038, 7040, 7152, 7154, 11016, 7070, 7029, 7006, 7084, 7090, 11042, 7093, 7126, 7163, 7064, 7051, 7004, 7033, 6974, 6942, 6987, 6993}
	pads                       = []uint16{7067, 7113}
	headsets                   = []uint16{2658, 2660, 2667, 2696}
	headsets2                  = []uint16{10754, 2711}
	dongles                    = []uint16{7132, 7078, 11008, 7060}
	legacyDevices              = []uint16{3080, 3081, 3082, 3090, 3091, 3093}
	cls                 *cluster.Device
)

// Stop will stop all active devices
func Stop() {
	// Stop all cluster operations
	cls.Stop()

	for _, device := range devices {
		CallDeviceMethod(device.Serial, "Stop")
		delete(devices, device.Serial)
	}
	err := hid.Exit()
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to exit HID interface")
	}
}

// StopDirty will stop the device without closing the file handles. Used when device is unplugged
func StopDirty(deviceId string, productId uint16) {
	device, ok := devices[deviceId]
	if !ok {
		device, ok = devices[strconv.Itoa(int(productId))]
		if !ok {
			return
		}
	}

	res := CallDeviceMethod(device.Serial, "StopDirty")
	if res != nil {
		val := res[0]
		uintResult := val.Uint()
		if uint8(uintResult) == 2 { // USB only devices, remove them from the device list
			deleteDevice(device.Serial)
		}
	}
}

// GetRgbProfiles will return a list of all RGB profiles for every device
func GetRgbProfiles() map[string]interface{} {
	profiles := make(map[string]interface{}, len(devices))
	for _, device := range devices {
		res := CallDeviceMethod(device.Serial, "GetRgbProfiles")
		if res != nil {
			val := res[0]
			profiles[device.Serial] = val.Interface()
		}
	}
	return profiles
}

// ScheduleDeviceBrightness will change device brightness level based on scheduler
func ScheduleDeviceBrightness(mode uint8) {
	for _, device := range GetDevices() {
		CallDeviceMethod(device.Serial, "SchedulerBrightness", mode)
	}
}

// UpdateGlobalRgbProfile will update device RGB profile
func UpdateGlobalRgbProfile(profile string) uint8 {
	channelId := -1
	for _, device := range devices {
		CallDeviceMethod(device.Serial, "UpdateRgbProfile", channelId, profile)
	}
	return 1
}

// ResetSpeedProfiles will reset the speed profile on each available device
func ResetSpeedProfiles(profile string) {
	for _, device := range devices {
		if device.ProductType == common.ProductTypeLinkHub ||
			device.ProductType == common.ProductTypeCC ||
			device.ProductType == common.ProductTypeCCXT {
			CallDeviceMethod(device.Serial, "ResetSpeedProfiles", profile)
		}
	}
}

// GetDevicesLedData will return led data for all devices
func GetDevicesLedData() interface{} {
	var leds []interface{}
	for _, device := range devices {
		res := CallDeviceMethod(device.Serial, "GetDeviceLedData")
		if res != nil && len(res) > 0 {
			val := res[0]
			leds = append(leds, val.Interface())
		}
	}
	return leds
}

// GetTemperatureProbes will return a list of temperature probes
func GetTemperatureProbes() interface{} {
	var probes []interface{}
	for _, device := range devices {
		if device.ProductType == common.ProductTypeLinkHub ||
			device.ProductType == common.ProductTypeCC ||
			device.ProductType == common.ProductTypeCCXT ||
			device.ProductType == common.ProductTypeMemory ||
			device.ProductType == common.ProductTypeCPro ||
			device.ProductType == common.ProductTypeElite ||
			device.ProductType == common.ProductTypeXC7 ||
			device.ProductType == common.ProductTypePlatinum {
			res := CallDeviceMethod(device.Serial, "GetTemperatureProbes")
			if res != nil && len(res) > 0 {
				val := res[0]
				probes = append(probes, val.Interface())
			}
		}
	}
	return probes
}

// UpdateDeviceMetrics will update device metrics
func UpdateDeviceMetrics() {
	metrics.PopulateDefault()
	metrics.PopulateStorage()

	for _, device := range devices {
		CallDeviceMethod(device.Serial, "UpdateDeviceMetrics")
	}
}

// deleteDevice will remove device from device list
func deleteDevice(serial string) {
	mutex.Lock()
	defer mutex.Unlock()
	delete(devices, serial)
}

// addDevice will add device to device list
func addDevice(device *common.Device) {
	if device == nil {
		return
	}

	mutex.Lock()
	defer mutex.Unlock()
	devices[device.Serial] = device
}

// CallDeviceMethod will call device method based on method name and arguments
func CallDeviceMethod(deviceId string, methodName string, args ...interface{}) []reflect.Value {
	mutex.Lock()
	defer mutex.Unlock()

	device, ok := devices[deviceId]
	if !ok {
		logger.Log(logger.Fields{"deviceId": deviceId}).Warn("Device not found")
		return nil
	}

	method := reflect.ValueOf(GetDevice(device.Serial)).MethodByName(methodName)
	if !method.IsValid() {
		logger.Log(logger.Fields{"method": methodName, "device": device.Product}).Warn("Method not found or not supported for this device type")
		return nil
	}

	reflectArgs := make([]reflect.Value, len(args))
	for i, a := range args {
		reflectArgs[i] = reflect.ValueOf(a)
	}

	return method.Call(reflectArgs)
}

// GetProducts will return all available products
func GetProducts() map[string]Product {
	return products
}

// GetDevice will return a device by device serial
func GetDevice(deviceId string) interface{} {
	if device, ok := devices[deviceId]; ok {
		return device.Instance
	}
	return nil
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
					Path:      sm.Path,
				}
			}
		} else {
			logger.Log(logger.Fields{"error": err}).Warn("No valid I2C devices found")
		}
	}

	// Create dummy cluster object before any other object
	cls = cluster.Init()
	devices["cluster"] = &common.Device{
		ProductType: common.ProductTypeCluster,
		Product:     "Cluster",
		Serial:      "cluster",
		Hidden:      true,
		Instance:    cls,
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

// deviceRegisterMap hold map of supported devices and their initialization call
var deviceRegisterMap = map[uint16]deviceRegister{
	0:     memory.Init,             // Memory
	3135:  lsh.Init,                // iCUE Link System Hub
	3122:  cc.Init,                 // iCUE COMMANDER Core
	3100:  cc.Init,                 // iCUE COMMANDER Core
	3114:  ccxt.Init,               // iCUE COMMANDER CORE XT
	3090:  platinum.Init,           // H150i Platinum
	3091:  platinum.Init,           // H115i Platinum
	3093:  platinum.Init,           // H100i Platinum
	3080:  hydro.Init,              // H80i Hydro
	3081:  hydro.Init,              // H100i Hydro
	3082:  hydro.Init,              // H115i Hydro
	3125:  elite.Init,              // iCUE H100i ELITE RGB
	3126:  elite.Init,              // iCUE H115i ELITE RGB
	3127:  elite.Init,              // iCUE H150i ELITE RGB
	3136:  elite.Init,              // iCUE H100i ELITE RGB White
	3137:  elite.Init,              // iCUE H150i ELITE RGB White
	3104:  elite.Init,              // iCUE H100i RGB PRO XT
	3105:  elite.Init,              // iCUE H115i RGB PRO XT
	3106:  elite.Init,              // iCUE H150i RGB PRO XT
	3095:  elite.Init,              // H115i RGB PLATINUM
	3096:  elite.Init,              // H100i RGB PLATINUM
	3097:  elite.Init,              // H100i RGB PLATINUM SE
	3098:  lncore.Init,             // Lighting Node CORE
	3083:  lnpro.Init,              // Lighting Node Pro
	3088:  cpro.Init,               // Commander Pro
	3138:  xc7.Init,                // XC7 ELITE LCD CPU Water Block
	2612:  st100.Init,              // ST100 LED Driver
	7067:  mm700.Init,              // MM700 RGB Gaming Mousepad
	7113:  mm700.Init,              // MM700 3XL RGB Gaming Mousepad
	6971:  mm800.Init,              // MM800 RGB POLARIS
	3107:  lt100.Init,              // LT100 Smart Lighting Tower
	7198:  psuhid.Init,             // HX1000i Power Supply
	7203:  psuhid.Init,             // HX1200i Power Supply
	7199:  psuhid.Init,             // HX1500i Power Supply
	7173:  psuhid.Init,             // HX750i Power Supply
	7174:  psuhid.Init,             // HX850i Power Supply
	7175:  psuhid.Init,             // HX1000i Power Supply
	7176:  psuhid.Init,             // HX1200i Power Supply
	7181:  psuhid.Init,             // RM1000i Power Supply
	7180:  psuhid.Init,             // RM850i Power Supply
	7207:  psuhid.Init,             // HX1200i Power Supply
	7054:  nexus.Init,              // iCUE NEXUS
	7127:  k65pm.Init,              // K65 PRO MINI
	7094:  k70pmWU.Init,            // K70 PPO MINI
	7165:  k70core.Init,            // K70 CORE RGB
	11009: k70coretkl.Init,         // K70 CORE TKL
	11010: k70coretklWU.Init,       // K70 CORE TKL WIRELESS
	11028: k70protkl.Init,          // K70 PRO TKL WIRELESS
	7097:  k70rgbtklcs.Init,        // K70 RGB TKL
	7027:  k70rgbtklcs.Init,        // K70 RGB TKL
	6973:  k55.Init,                // K55 RGB
	7166:  k55core.Init,            // K55 CORE RGB
	7076:  k55pro.Init,             // K55 PRO RGB
	7073:  k55proXT.Init,           // K55 RGB PRO XT
	7072:  k60rgbpro.Init,          // K60 RGB PRO
	7104:  k70max.Init,             // K70 MAX
	7110:  k70pro.Init,             // K70 PRO
	7091:  k70pro.Init,             // K70 PRO
	7124:  k70pro.Init,             // K70 PRO
	6985:  k70mk2.Init,             // K70 RGB MK2
	6997:  k70mk2.Init,             // K70 RGB MK2
	7019:  k70mk2.Init,             // K70 RGB MK2
	11024: k65plusWU.Init,          // K65 PLUS WIRELESS USB
	11025: k65plusWU.Init,          // K65 PLUS WIRELESS USB
	11015: k65plusW.Init,           // K65 PLUS WIRELESS
	6957:  k95platinum.Init,        // K95 PLATINUM
	7083:  k100airWU.Init,          // K100 AIR USB
	7036:  k100.Init,               // K100
	7109:  k100.Init,               // K100
	7037:  k100.Init,               // K100
	11012: makr75WU.Init,           // MAKR 75
	7059:  katarpro.Init,           // KATAR PRO Gaming Mouse
	7084:  katarproxt.Init,         // KATAR PRO XT Gaming Mouse
	7005:  ironclaw.Init,           // IRONCLAW RGB Gaming Mouse
	6987:  darkcorergbseWU.Init,    // DARK CORE RGB SE
	6988:  ironclawWU.Init,         // IRONCLAW RGB WIRELESS Gaming Mouse
	7096:  nightsabreWU.Init,       // NIGHTSABRE WIRELESS Mouse
	7139:  scimitar.Init,           // SCIMITAR RGB ELITE
	6974:  scimitarprorgb.Init,     // SCIMITAR PRO RGB
	6942:  scimitarrgb.Init,        // SCIMITAR RGB
	7051:  scimitarrgbelite.Init,   // SCIMITAR RGB ELITE
	7131:  scimitarWU.Init,         // SCIMITAR RGB ELITE WIRELESS
	11042: scimitarSEWU.Init,       // SCIMITAR ELITE WIRELESS SE
	11011: m55.Init,                // M55 Gaming Mouse
	7024:  m55rgbpro.Init,          // M55 RGB PRO Gaming Mouse
	7060:  katarproW.Init,          // KATAR PRO Wireless Gaming Dongle
	7038:  darkcorergbproseWU.Init, // DARK CORE RGB PRO SE Gaming Mouse
	7040:  darkcorergbproWU.Init,   // DARK CORE RGB PRO Gaming Mouse
	7152:  m75.Init,                // M75 Gaming Mouse
	11016: m75WU.Init,              // M75 WIRELESS Gaming Mouse
	7154:  m75AirWU.Init,           // M75 AIR WIRELESS Gaming Mouse
	7070:  m65rgbultra.Init,        // M65 RGB ULTRA Gaming Mouse
	7093:  m65rgbultraWU.Init,      // M65 RGB ULTRA WIRELESS Gaming Mouse
	7126:  m65rgbultraWU.Init,      // M65 RGB ULTRA WIRELESS Gaming Mouse
	7029:  harpoonrgbpro.Init,      // HARPOON RGB PRO Gaming Mouse
	7006:  harpoonWU.Init,          // HARPOON Gaming Mouse
	7004:  nightswordrgb.Init,      // NIGHTSWORD RGB Gaming Mouse
	7064:  sabrergbproWU.Init,      // SABRE RGB PRO WIRELESS Gaming Mouse
	7033:  sabrergbpro.Init,        // SABRE RGB PRO
	7090:  darkstarWU.Init,         // DARKSTAR RGB WIRELESS Gaming Mouse
	2658:  virtuosorgbXTWU.Init,    // VIRTUOSO RGB WIRELESS XT
	2696:  hs80rgb.Init,            // HS80 RGB USB Gaming Headset
}

// deviceRegisterMapEx hold map of supported devices and their initialization call with callback function
var deviceRegisterMapEx = map[uint16]deviceRegisterEx{
	7132:  slipstream.Init,          // SLIPSTREAM WIRELESS USB Receiver
	7078:  slipstream.Init,          // SLIPSTREAM WIRELESS USB Receiver
	11008: slipstream.Init,          // SLIPSTREAM WIRELESS USB Receiver
	10754: virtuosomaxdongle.Init,   // VIRTUOSO MAX WIRELESS
	2711:  hs80maxdongle.Init,       // HS80 MAX WIRELESS
	6993:  darkcorergbsesongle.Init, // DARK CORE RGB SE Wireless USB Receiver
	2660:  headsetdongle.Init,       // Headset dongle
	2667:  headsetdongle.Init,       // Headset dongle
}

// initializeDevice will initialize a device
func initializeDevice(productId uint16, key, productPath string) {
	callback, ok := deviceRegisterMap[productId]
	if ok {
		go func(vid, pid uint16, serial, path string, cb deviceRegister) {
			dev := cb(vid, pid, serial, path)
			addDevice(dev)
		}(vendorId, productId, key, productPath, callback)
	} else {
		// Used for initialization of devices with callback function
		callbackEx, valid := deviceRegisterMapEx[productId]
		if valid {
			go func(vid, pid uint16, serial, path string, cb deviceRegisterEx) {
				dev := cb(vid, pid, serial, path, addDevice)
				addDevice(dev)
			}(vendorId, productId, key, productPath, callbackEx)
		}
	}
}
