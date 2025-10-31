package devices

import (
	"OpenLinkHub/src/cluster"
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/devices/cc"
	"OpenLinkHub/src/devices/ccxt"
	"OpenLinkHub/src/devices/cduo"
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
	"OpenLinkHub/src/devices/k55coretkl"
	"OpenLinkHub/src/devices/k55pro"
	"OpenLinkHub/src/devices/k55proXT"
	"OpenLinkHub/src/devices/k57rgbWU"
	"OpenLinkHub/src/devices/k60rgbpro"
	"OpenLinkHub/src/devices/k65plusWU"
	"OpenLinkHub/src/devices/k65plusWdongle"
	"OpenLinkHub/src/devices/k65pm"
	"OpenLinkHub/src/devices/k68rgb"
	"OpenLinkHub/src/devices/k70core"
	"OpenLinkHub/src/devices/k70coretkl"
	"OpenLinkHub/src/devices/k70coretklWU"
	"OpenLinkHub/src/devices/k70lux"
	"OpenLinkHub/src/devices/k70luxrgb"
	"OpenLinkHub/src/devices/k70max"
	"OpenLinkHub/src/devices/k70mk2"
	"OpenLinkHub/src/devices/k70pmWU"
	"OpenLinkHub/src/devices/k70pro"
	"OpenLinkHub/src/devices/k70protkl"
	"OpenLinkHub/src/devices/k70rgbRF"
	"OpenLinkHub/src/devices/k70rgbtklcs"
	"OpenLinkHub/src/devices/k95platinum"
	"OpenLinkHub/src/devices/k95platinumXT"
	"OpenLinkHub/src/devices/katarpro"
	"OpenLinkHub/src/devices/katarproW"
	"OpenLinkHub/src/devices/katarproxt"
	"OpenLinkHub/src/devices/lncore"
	"OpenLinkHub/src/devices/lnpro"
	"OpenLinkHub/src/devices/lsh"
	"OpenLinkHub/src/devices/lt100"
	"OpenLinkHub/src/devices/m55"
	"OpenLinkHub/src/devices/m55rgbpro"
	"OpenLinkHub/src/devices/m65rgbelite"
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
	"OpenLinkHub/src/devices/psudongle"
	"OpenLinkHub/src/devices/psuhid"
	"OpenLinkHub/src/devices/sabreprocs"
	"OpenLinkHub/src/devices/sabrergbpro"
	"OpenLinkHub/src/devices/sabrergbproWU"
	"OpenLinkHub/src/devices/scimitar"
	"OpenLinkHub/src/devices/scimitarSEWU"
	"OpenLinkHub/src/devices/scimitarWU"
	"OpenLinkHub/src/devices/scimitarprorgb"
	"OpenLinkHub/src/devices/scimitarrgb"
	"OpenLinkHub/src/devices/scimitarrgbelite"
	"OpenLinkHub/src/devices/scufdongle"
	"OpenLinkHub/src/devices/scufenvisionproWU"
	"OpenLinkHub/src/devices/slipstream"
	"OpenLinkHub/src/devices/st100"
	"OpenLinkHub/src/devices/strafergbmk2"
	"OpenLinkHub/src/devices/virtuosoSEWU"
	"OpenLinkHub/src/devices/virtuosoWU"
	"OpenLinkHub/src/devices/virtuosomaxdongle"
	"OpenLinkHub/src/devices/virtuosorgbXTWU"
	"OpenLinkHub/src/devices/voidV2dongle"
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
type Device struct {
	ProductId uint16
	Path      string
	DevPath   string
	Serial    string
}

type Product struct {
	InterfaceId      int
	UsagePage        uint16
	Name             string
	DeviceRegister   deviceRegister
	DeviceRegisterEx deviceRegisterEx
}

var (
	mutex               sync.Mutex
	cls                 *cluster.Device
	expectedPermissions = []os.FileMode{os.FileMode(0600), os.FileMode(0660)}
	vendorId            = uint16(6940)  // Corsair
	scufVendorId        = uint16(11925) // Scuf
	interfaceId         = 0
	devices             = make(map[string]*common.Device)
	deviceList          = make(map[string]Device)
	legacyDevices       = []uint16{3080, 3081, 3082, 3090, 3091, 3093, 7168}
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

// GetSupportedDevices will return list of supported devices
func GetSupportedDevices() interface{} {
	type product struct {
		ProductId uint16
		Name      string
		Enabled   bool
	}

	var products []product
	for key, val := range deviceRegisterMap {
		if key > 0 {
			p := product{
				ProductId: key,
				Name:      val.Name,
				Enabled:   true,
			}

			if slices.Contains(config.GetConfig().Exclude, key) {
				p.Enabled = false
			}
			products = append(products, p)
		}
	}
	return products
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
			device.ProductType == common.ProductTypeCCXT ||
			device.ProductType == common.ProductTypeCPro ||
			device.ProductType == common.ProductTypeElite ||
			device.ProductType == common.ProductTypeHydro ||
			device.ProductType == common.ProductTypePlatinum {
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
			device.ProductType == common.ProductTypeHydro ||
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
func GetProducts() map[string]Device {
	return deviceList
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
func InitManual(productId uint16, key string) {
	var device = Device{
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

		if val, ok := deviceRegisterMap[info.ProductID]; ok {
			if val.UsagePage > 0 {
				if info.UsagePage == val.UsagePage {
					interfaceId = info.InterfaceNbr
				}
			} else {
				interfaceId = val.InterfaceId
			}
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

			if len(key) == 0 {
				key = info.SerialNbr
				if len(key) == 0 {
					key = strconv.Itoa(int(info.ProductID))
				}
			}

			if interfaceId == 1 || interfaceId == 3 || interfaceId == 4 {
				key = info.Path
			}
			device = Device{
				ProductId: info.ProductID,
				Path:      info.Path,
				Serial:    key,
			}
		}
		return nil
	})

	// Enumerate all Corsair devices
	err := hid.Enumerate(vendorId, productId, enum)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": vendorId}).Fatal("Unable to enumerate devices")
	}

	// Enumerate all Scuf devices
	err = hid.Enumerate(scufVendorId, productId, enum)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": vendorId}).Fatal("Unable to enumerate devices")
	}

	if device.ProductId > 0 && len(device.Path) > 0 {
		initializeDevice(productId, device.Serial, device.Path)
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

		if val, ok := deviceRegisterMap[info.ProductID]; ok {
			if val.UsagePage > 0 {
				if info.UsagePage == val.UsagePage {
					interfaceId = info.InterfaceNbr
				}
			} else {
				interfaceId = val.InterfaceId
			}
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

			key := info.SerialNbr
			if len(key) == 0 {
				// Devices with no serial, make serial based of productId
				key = strconv.Itoa(int(info.ProductID))
			}

			if interfaceId == 1 || interfaceId == 3 || interfaceId == 4 {
				key = info.Path
			}

			deviceList[key] = Device{
				ProductId: info.ProductID,
				Path:      info.Path,
				DevPath:   p,
				Serial:    info.SerialNbr,
			}
		}
		return nil
	})

	// Enumerate all Corsair devices
	err := hid.Enumerate(vendorId, hid.ProductIDAny, enum)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": vendorId}).Fatal("Unable to enumerate devices")
	}

	// Enumerate all Scuf devices
	err = hid.Enumerate(scufVendorId, hid.ProductIDAny, enum)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": vendorId}).Fatal("Unable to enumerate devices")
	}

	// Memory
	if config.GetConfig().Memory {
		sm, err := smbus.GetSmBus()
		if err == nil {
			if len(sm.Path) > 0 {
				deviceList[sm.Path] = Device{
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
			deviceList[device.SerialNbr] = Device{
				ProductId: device.ProductID,
				Path:      device.Path,
			}
		}
	}

	// USB-HID
	for key, product := range deviceList {
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
var deviceRegisterMap = map[uint16]Product{
	0:     {0, 0, "Memory", memory.Init, nil},                            // Memory
	3135:  {0, 0, "iCUE LINK SYSTEM HUB", lsh.Init, nil},                 // iCUE Link System Hub
	3122:  {0, 0, "iCUE COMMANDER Core", cc.Init, nil},                   // iCUE COMMANDER Core
	3100:  {0, 0, "iCUE COMMANDER Core", cc.Init, nil},                   // iCUE COMMANDER Core
	3114:  {0, 0, "iCUE COMMANDER CORE XT", ccxt.Init, nil},              // iCUE COMMANDER CORE XT
	3158:  {0, 0, "iCUE COMMANDER DUO", cduo.Init, nil},                  // iCUE COMMANDER DUO (USB)
	3090:  {0, 0, "H150i PLATINUM", platinum.Init, nil},                  // H150i PLATINUM
	3091:  {0, 0, "H115i PLATINUM", platinum.Init, nil},                  // H115i PLATINUM
	3093:  {0, 0, "H100i PLATINUM", platinum.Init, nil},                  // H100i PLATINUM
	3080:  {0, 0, "H80i HYDRO", hydro.Init, nil},                         // H80i HYDRO
	3081:  {0, 0, "H100i HYDRO", hydro.Init, nil},                        // H100i HYDRO
	3082:  {0, 0, "H115i HYDRO", hydro.Init, nil},                        // H115i HYDRO
	3125:  {0, 0, "iCUE H100i ELITE RGB", elite.Init, nil},               // iCUE H100i ELITE RGB
	3126:  {0, 0, "iCUE H115i ELITE RGB", elite.Init, nil},               // iCUE H115i ELITE RGB
	3127:  {0, 0, "iCUE H150i ELITE RGB", elite.Init, nil},               // iCUE H150i ELITE RGB
	3136:  {0, 0, "iCUE H100i ELITE RGB White", elite.Init, nil},         // iCUE H100i ELITE RGB White
	3137:  {0, 0, "iCUE H150i ELITE RGB White", elite.Init, nil},         // iCUE H150i ELITE RGB White
	3104:  {0, 0, "iCUE H100i RGB PRO XT", elite.Init, nil},              // iCUE H100i RGB PRO XT
	3105:  {0, 0, "iCUE H115i RGB PRO XT", elite.Init, nil},              // iCUE H115i RGB PRO XT
	3106:  {0, 0, "iCUE H150i RGB PRO XT", elite.Init, nil},              // iCUE H150i RGB PRO XT
	3095:  {0, 0, "H115i RGB PLATINUM", elite.Init, nil},                 // H115i RGB PLATINUM
	3096:  {0, 0, "H100i RGB PLATINUM", elite.Init, nil},                 // H100i RGB PLATINUM
	3097:  {0, 0, "H100i RGB PLATINUM SE", elite.Init, nil},              // H100i RGB PLATINUM SE
	3098:  {0, 0, "LIGHTING NODE CORE", lncore.Init, nil},                // Lighting Node CORE
	3083:  {0, 0, "LIGHTING NODE PRO", lnpro.Init, nil},                  // Lighting Node Pro
	3088:  {0, 0, "COMMANDER PRO", cpro.Init, nil},                       // Commander Pro
	7424:  {0, 0, "COMMANDER PRO 1000D", cpro.Init, nil},                 // Obsidian 1000D Hub (Commander Pro)
	3138:  {0, 0, "XC7 ELITE LCD", xc7.Init, nil},                        // XC7 ELITE LCD CPU Water Block
	2612:  {0, 0, "ST100", st100.Init, nil},                              // ST100 LED Driver
	7067:  {1, 0, "MM700 RGB", mm700.Init, nil},                          // MM700 RGB Gaming Mousepad
	7113:  {1, 0, "MM700 3XL RGB", mm700.Init, nil},                      // MM700 3XL RGB Gaming Mousepad
	6971:  {0, 0, "MM800 RGB POLARIS", mm800.Init, nil},                  // MM800 RGB POLARIS
	3107:  {0, 0, "LT100", lt100.Init, nil},                              // LT100 Smart Lighting Tower
	7198:  {0, 0, "HX1000i", psuhid.Init, nil},                           // HX1000i Power Supply
	7203:  {0, 0, "HX1200i", psuhid.Init, nil},                           // HX1200i Power Supply
	7199:  {0, 0, "HX1500i", psuhid.Init, nil},                           // HX1500i Power Supply
	7173:  {0, 0, "HX750i", psuhid.Init, nil},                            // HX750i Power Supply
	7174:  {0, 0, "HX850i", psuhid.Init, nil},                            // HX850i Power Supply
	7175:  {0, 0, "HX1000i", psuhid.Init, nil},                           // HX1000i Power Supply
	7176:  {0, 0, "HX1200i", psuhid.Init, nil},                           // HX1200i Power Supply
	7181:  {0, 0, "RM1000i", psuhid.Init, nil},                           // RM1000i Power Supply
	7180:  {0, 0, "RM850i", psuhid.Init, nil},                            // RM850i Power Supply
	7207:  {0, 0, "HX1200i", psuhid.Init, nil},                           // HX1200i Power Supply
	7054:  {0, 0, "iCUE NEXUS", nexus.Init, nil},                         // iCUE NEXUS
	7127:  {1, 0, "K65 PRO MINI", k65pm.Init, nil},                       // K65 PRO MINI
	7094:  {1, 0, "K70 PPO MINI", k70pmWU.Init, nil},                     // K70 PPO MINI
	7165:  {1, 0, "K70 CORE RGB", k70core.Init, nil},                     // K70 CORE RGB
	11009: {1, 0, "K70 CORE TKL", k70coretkl.Init, nil},                  // K70 CORE TKL
	11010: {1, 0, "K70 CORE TKL", k70coretklWU.Init, nil},                // K70 CORE TKL WIRELESS
	11028: {1, 0, "K70 PRO TKL", k70protkl.Init, nil},                    // K70 PRO TKL WIRELESS
	7097:  {1, 0, "K70 RGB TKL CS", k70rgbtklcs.Init, nil},               // K70 RGB TKL
	7027:  {1, 0, "K70 RGB TKL", k70rgbtklcs.Init, nil},                  // K70 RGB TKL
	6973:  {1, 0, "K55 RGB", k55.Init, nil},                              // K55 RGB
	7166:  {1, 0, "K55 CORE RGB", k55core.Init, nil},                     // K55 CORE RGB
	11040: {1, 0, "K55 CORE TKL RGB", k55coretkl.Init, nil},              // K55 CORE RGB
	7076:  {1, 0, "K55 PRO RGB", k55pro.Init, nil},                       // K55 PRO RGB
	7073:  {1, 0, "K55 RGB PRO XT", k55proXT.Init, nil},                  // K55 RGB PRO XT
	7022:  {1, 0, "K57 RGB WIRELESS", k57rgbWU.Init, nil},                // K57 RGB WIRELESS
	7072:  {1, 0, "K60 RGB PRO", k60rgbpro.Init, nil},                    // K60 RGB PRO
	7104:  {1, 0, "K70 MAX", k70max.Init, nil},                           // K70 MAX
	7110:  {1, 0, "K70 PRO", k70pro.Init, nil},                           // K70 PRO
	7091:  {1, 0, "K70 PRO", k70pro.Init, nil},                           // K70 PRO
	7124:  {1, 0, "K70 PRO", k70pro.Init, nil},                           // K70 PRO
	6966:  {1, 0, "K70 LUX", k70lux.Init, nil},                           // K70 LUX
	6963:  {1, 0, "K70 LUX RGB", k70luxrgb.Init, nil},                    // K70 LUX RGB
	6968:  {1, 0, "K70 RGB RAPIDFIRE", k70rgbRF.Init, nil},               // K70 RGB RAPIDFIRE
	6985:  {1, 0, "K70 RGB MK2", k70mk2.Init, nil},                       // K70 RGB MK2
	6997:  {1, 0, "K70 RGB MK2", k70mk2.Init, nil},                       // K70 RGB MK2
	7019:  {1, 0, "K70 RGB MK2", k70mk2.Init, nil},                       // K70 RGB MK2
	6984:  {1, 0, "STRAFE RGB MK2", strafergbmk2.Init, nil},              // STRAFE RGB MK2
	11024: {1, 0, "K65 PLUS WIRELESS", k65plusWU.Init, nil},              // K65 PLUS WIRELESS USB
	11025: {1, 0, "K65 PLUS WIRELESS", k65plusWU.Init, nil},              // K65 PLUS WIRELESS USB
	6957:  {1, 0, "K95 PLATINUM", k95platinum.Init, nil},                 // K95 PLATINUM
	7049:  {1, 0, "K95 PLATINUM XT", k95platinumXT.Init, nil},            // K95 PLATINUM XT
	6991:  {1, 0, "K68 RGB", k68rgb.Init, nil},                           // K68 RGB
	7083:  {1, 0, "K100 AIR", k100airWU.Init, nil},                       // K100 AIR USB
	7036:  {1, 0, "K100", k100.Init, nil},                                // K100
	7109:  {1, 0, "K100", k100.Init, nil},                                // K100
	7037:  {1, 0, "K100", k100.Init, nil},                                // K100
	11012: {1, 0, "MAKR 75", makr75WU.Init, nil},                         // MAKR 75
	7059:  {1, 0, "KATAR PRO", katarpro.Init, nil},                       // KATAR PRO Gaming Mouse
	7084:  {1, 0, "KATAR PRO XT", katarproxt.Init, nil},                  // KATAR PRO XT Gaming Mouse
	7005:  {1, 0, "IRONCLAW RGB", ironclaw.Init, nil},                    // IRONCLAW RGB Gaming Mouse
	6987:  {1, 0, "DARK CORE RGB SE", darkcorergbseWU.Init, nil},         // DARK CORE RGB SE
	6988:  {1, 0, "IRONCLAW RGB WIRELESS", ironclawWU.Init, nil},         // IRONCLAW RGB WIRELESS Gaming Mouse
	7096:  {1, 0, "NIGHTSABRE WIRELESS", nightsabreWU.Init, nil},         // NIGHTSABRE WIRELESS Mouse
	7139:  {1, 0, "SCIMITAR RGB ELITE", scimitar.Init, nil},              // SCIMITAR RGB ELITE
	6974:  {1, 0, "SCIMITAR PRO RGB", scimitarprorgb.Init, nil},          // SCIMITAR PRO RGB
	6942:  {1, 0, "SCIMITAR RGB", scimitarrgb.Init, nil},                 // SCIMITAR RGB
	7051:  {1, 0, "SCIMITAR RGB ELITE", scimitarrgbelite.Init, nil},      // SCIMITAR RGB ELITE
	7131:  {1, 0, "SCIMITAR RGB ELITE WIRELESS", scimitarWU.Init, nil},   // SCIMITAR RGB ELITE WIRELESS
	11042: {1, 0, "SCIMITAR ELITE WIRELESS SE", scimitarSEWU.Init, nil},  // SCIMITAR ELITE WIRELESS SE
	11011: {1, 0, "M55", m55.Init, nil},                                  // M55 Gaming Mouse
	7024:  {1, 0, "M55 RGB PRO", m55rgbpro.Init, nil},                    // M55 RGB PRO Gaming Mouse
	7060:  {1, 0, "KATAR PRO WIRELESS", katarproW.Init, nil},             // KATAR PRO Wireless Gaming Dongle
	7038:  {1, 0, "DARK CORE RGB PRO SE", darkcorergbproseWU.Init, nil},  // DARK CORE RGB PRO SE Gaming Mouse
	7040:  {1, 0, "DARK CORE RGB PRO", darkcorergbproWU.Init, nil},       // DARK CORE RGB PRO Gaming Mouse
	7152:  {1, 0, "M75", m75.Init, nil},                                  // M75 Gaming Mouse
	11016: {1, 0, "M75 WIRELESS", m75WU.Init, nil},                       // M75 WIRELESS Gaming Mouse
	7154:  {1, 0, "M75 AIR WIRELESS", m75AirWU.Init, nil},                // M75 AIR WIRELESS Gaming Mouse
	7070:  {1, 0, "M65 RGB ULTRA", m65rgbultra.Init, nil},                // M65 RGB ULTRA Gaming Mouse
	7093:  {1, 0, "M65 RGB ULTRA WIRELESS", m65rgbultraWU.Init, nil},     // M65 RGB ULTRA WIRELESS Gaming Mouse
	7126:  {1, 0, "M65 RGB ULTRA WIRELESS", m65rgbultraWU.Init, nil},     // M65 RGB ULTRA WIRELESS Gaming Mouse
	7002:  {1, 0, "M65 RGB ELITE", m65rgbelite.Init, nil},                // M65 RGB ELITE Gaming Mouse
	7029:  {1, 0, "HARPOON RGB PRO", harpoonrgbpro.Init, nil},            // HARPOON RGB PRO Gaming Mouse
	7006:  {1, 0, "HARPOON", harpoonWU.Init, nil},                        // HARPOON Gaming Mouse
	7004:  {1, 0, "NIGHTSWORD RGB", nightswordrgb.Init, nil},             // NIGHTSWORD RGB Gaming Mouse
	7064:  {1, 0, "SABRE RGB PRO WIRELESS", sabrergbproWU.Init, nil},     // SABRE RGB PRO WIRELESS Gaming Mouse
	7033:  {1, 0, "SABRE RGB PRO", sabrergbpro.Init, nil},                // SABRE RGB PRO
	7034:  {1, 0, "SABRE PRO CS", sabreprocs.Init, nil},                  // SABRE PRO CS
	7090:  {1, 0, "DARKSTAR RGB WIRELESS", darkstarWU.Init, nil},         // DARKSTAR RGB WIRELESS Gaming Mouse
	2658:  {3, 0, "VIRTUOSO RGB WIRELESS XT", virtuosorgbXTWU.Init, nil}, // VIRTUOSO RGB WIRELESS XT
	2627:  {3, 0, "VIRTUOSO", virtuosoWU.Init, nil},                      // VIRTUOSO USB Gaming Headset
	2696:  {3, 0, "HS80 RGB USB", hs80rgb.Init, nil},                     // HS80 RGB USB Gaming Headset
	7132:  {1, 0, "SLIPSTREAM WIRELESS", nil, slipstream.Init},           // SLIPSTREAM WIRELESS USB Receiver
	7078:  {1, 0, "SLIPSTREAM WIRELESS", nil, slipstream.Init},           // SLIPSTREAM WIRELESS USB Receiver
	11008: {1, 0, "SLIPSTREAM WIRELESS", nil, slipstream.Init},           // SLIPSTREAM WIRELESS USB Receiver
	10754: {4, 0, "VIRTUOSO MAX WIRELESS", nil, virtuosomaxdongle.Init},  // VIRTUOSO MAX WIRELESS
	2711:  {4, 0, "HS80 MAX WIRELESS", nil, hs80maxdongle.Init},          // HS80 MAX WIRELESS
	6993:  {1, 0, "DARK CORE RGB SE", nil, darkcorergbsesongle.Init},     // DARK CORE RGB SE Wireless USB Receiver
	2660:  {3, 0, "HEADSET DONGLE", nil, headsetdongle.Init},             // Headset dongle
	2667:  {3, 0, "HEADSET DONGLE", nil, headsetdongle.Init},             // Headset dongle
	2628:  {3, 0, "HEADSET DONGLE", nil, headsetdongle.Init},             // Headset dongle
	2622:  {3, 65346, "HEADSET DONGLE", nil, headsetdongle.Init},         // Headset dongle
	11015: {1, 0, "K65 PLUS WIRELESS", nil, k65plusWdongle.Init},         // K65 PLUS WIRELESS
	2621:  {3, 65346, "VIRTUOSO SE", virtuosoSEWU.Init, nil},             // CORSAIR VIRTUOSO SE USB Gaming Headset
	10760: {4, 0, "VOID WIRELESS V2", nil, voidV2dongle.Init},            // VOID WIRELESS V2
	7168:  {0, 0, "CORSAIR LINK TM USB DONGLE", psudongle.Init, nil},     // CORSAIR LINK TM USB DONGLE
	17229: {4, 0, "SCUF ENVISION PRO", scufenvisionproWU.Init, nil},      // SCUF Envision Pro Controller
	17230: {4, 0, "SCUF PC Controller Dongle", nil, scufdongle.Init},     // SCUF Gaming SCUF PC Controller Dongle
}

// initializeDevice will initialize a device
func initializeDevice(productId uint16, key, productPath string) {
	callback, ok := deviceRegisterMap[productId]
	if ok {
		if callback.DeviceRegister != nil {
			go func(vid, pid uint16, serial, path string, cb deviceRegister) {
				dev := cb(vid, pid, serial, path)
				addDevice(dev)
			}(vendorId, productId, key, productPath, callback.DeviceRegister)
		}

		if callback.DeviceRegisterEx != nil {
			go func(vid, pid uint16, serial, path string, cb deviceRegisterEx) {
				dev := cb(vid, pid, serial, path, addDevice)
				addDevice(dev)
			}(vendorId, productId, key, productPath, callback.DeviceRegisterEx)
		}
	}
}
