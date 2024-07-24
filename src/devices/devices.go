package devices

import (
	"OpenLinkHub/src/devices/cc"
	"OpenLinkHub/src/devices/ccxt"
	"OpenLinkHub/src/devices/elite"
	"OpenLinkHub/src/devices/linksystemhub"
	"OpenLinkHub/src/logger"
	"fmt"
	"github.com/sstallion/go-hid"
	"strconv"
)

const (
	productTypeLinkHub = 0
	productTypeCC      = 1
	productTypeCCXT    = 2
	productTypeElite   = 3
)

type Device struct {
	ProductType   uint8
	Product       string
	Serial        string
	Firmware      string
	LinkSystemHub *linksystemhub.Device `json:"linkSystemHub,omitempty"`
	CC            *cc.Device            `json:"cc,omitempty"`
	CCXT          *ccxt.Device          `json:"ccxt,omitempty"`
	Elite         *elite.Device         `json:"elite,omitempty"`
}

var (
	vendorId    uint16 = 6940 // Corsair
	interfaceId        = 0
	devices            = make(map[string]*Device)
	products           = make(map[string]uint16)
)

// Stop will stop all active devices
func Stop() {
	for _, device := range devices {
		switch device.ProductType {
		case productTypeLinkHub:
			{
				if device.LinkSystemHub != nil {
					device.LinkSystemHub.Stop()
				}
			}
		case productTypeCC:
			{
				if device.CC != nil {
					device.CC.Stop()
				}
			}
		case productTypeCCXT:
			{
				if device.CCXT != nil {
					device.CCXT.Stop()
				}
			}
		case productTypeElite:
			{
				if device.Elite != nil {
					device.Elite.Stop()
				}
			}
		}
	}
	err := hid.Exit()
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to exit HID interface")
	}
}

// UpdateExternalHubStatus will enable or disable device external-LED hub
func UpdateExternalHubStatus(deviceId string, status bool) int {
	if device, ok := devices[deviceId]; ok {
		switch device.ProductType {
		case productTypeCCXT:
			{
				if device.CCXT != nil {
					device.CCXT.UpdateExternalHubStatus(status)
					return 1
				}
			}
		}
	}
	return 0
}

// UpdateExternalHubDeviceType will update a device type connected to an external-LED hub
func UpdateExternalHubDeviceType(deviceId string, deviceType int) int {
	if device, ok := devices[deviceId]; ok {
		switch device.ProductType {
		case productTypeCCXT:
			{
				if device.CCXT != nil {
					return device.CCXT.UpdateExternalHubDeviceType(deviceType)
				}
			}
		}
	}
	return 0
}

// UpdateExternalHubDeviceAmount will update a device amount connected to an external-LED hub
func UpdateExternalHubDeviceAmount(deviceId string, deviceType int) int {
	if device, ok := devices[deviceId]; ok {
		switch device.ProductType {
		case productTypeCCXT:
			{
				if device.CCXT != nil {
					return device.CCXT.UpdateExternalHubDeviceAmount(deviceType)
				}
			}
		}
	}
	return 0
}

// UpdateSpeedProfile will update device speeds with a given serial number
func UpdateSpeedProfile(deviceId string, channelId int, profile string) {
	if device, ok := devices[deviceId]; ok {
		switch device.ProductType {
		case productTypeLinkHub:
			{
				if device.LinkSystemHub != nil {
					device.LinkSystemHub.UpdateSpeedProfile(channelId, profile)
				}
			}
		case productTypeCC:
			{
				if device.CC != nil {
					device.CC.UpdateSpeedProfile(channelId, profile)
				}
			}
		case productTypeCCXT:
			{
				if device.CCXT != nil {
					device.CCXT.UpdateSpeedProfile(channelId, profile)
				}
			}
		case productTypeElite:
			{
				if device.Elite != nil {
					device.Elite.UpdateSpeedProfile(channelId, profile)
				}
			}
		}
	}
}

// UpdateManualSpeed will update device speeds with a given serial number
func UpdateManualSpeed(deviceId string, channelId int, value uint16) uint8 {
	if device, ok := devices[deviceId]; ok {
		switch device.ProductType {
		case productTypeLinkHub:
			{
				if device.LinkSystemHub != nil {
					return device.LinkSystemHub.UpdateDeviceSpeed(channelId, value)
				}
			}
		case productTypeCC:
			{
				if device.CC != nil {
					return device.CC.UpdateDeviceSpeed(channelId, value)
				}
			}
		case productTypeCCXT:
			{
				if device.CCXT != nil {
					return device.CCXT.UpdateDeviceSpeed(channelId, value)
				}
			}
		case productTypeElite:
			{
				if device.Elite != nil {
					return device.Elite.UpdateDeviceSpeed(channelId, value)
				}
			}
		}
	}
	return 0
}

// UpdateRgbProfile will update device RGB profile
func UpdateRgbProfile(deviceId string, channelId int, profile string) {
	if device, ok := devices[deviceId]; ok {
		switch device.ProductType {
		case productTypeLinkHub:
			{
				if device.LinkSystemHub != nil {
					device.LinkSystemHub.UpdateRgbProfile(channelId, profile)
				}
			}
		case productTypeCC:
			{
				if device.CC != nil {
					device.CC.UpdateRgbProfile(channelId, profile)
				}
			}
		case productTypeCCXT:
			{
				if device.CCXT != nil {
					device.CCXT.UpdateRgbProfile(channelId, profile)
				}
			}
		case productTypeElite:
			{
				if device.Elite != nil {
					device.Elite.UpdateRgbProfile(channelId, profile)
				}
			}
		}

	}
}

// ResetSpeedProfiles will reset the speed profile on each available device
func ResetSpeedProfiles(profile string) {
	for _, device := range devices {
		switch device.ProductType {
		case productTypeLinkHub:
			{
				if device.LinkSystemHub != nil {
					device.LinkSystemHub.ResetSpeedProfiles(profile)
				}
			}
		case productTypeCC:
			{
				if device.CC != nil {
					device.CC.ResetSpeedProfiles(profile)
				}
			}
		case productTypeCCXT:
			{
				if device.CCXT != nil {
					device.CCXT.ResetSpeedProfiles(profile)
				}
			}
		}
	}
}

// GetDevices will return all available devices
func GetDevices() map[string]*Device {
	return devices
}

// GetDevice will return a device by device serial
func GetDevice(deviceId string) interface{} {
	if device, ok := devices[deviceId]; ok {
		switch device.ProductType {
		case productTypeLinkHub:
			{
				return device.LinkSystemHub
			}
		case productTypeCC:
			{
				return device.CC
			}
		case productTypeCCXT:
			{
				return device.CCXT
			}
		case productTypeElite:
			{
				return device.Elite
			}
		}
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
		// We only need interface 0
		if info.InterfaceNbr == interfaceId {
			products[info.SerialNbr] = info.ProductID
		}
		return nil
	})

	// Enumerate all Corsair devices
	err := hid.Enumerate(vendorId, hid.ProductIDAny, enum)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": vendorId}).Fatal("Unable to enumerate devices")
	}

	if len(products) == 0 {
		fmt.Println("Found 0 compatible devices. Exit")
		logger.Log(logger.Fields{"vendor": vendorId}).Fatal("No compatible devices")
	}

	for serial, productId := range products {
		switch productId {
		case 3135: // CORSAIR iCUE Link System Hub
			{
				go func(vendorId, productId uint16, serialId string) {
					dev := linksystemhub.Init(vendorId, productId, serialId)
					if dev == nil {
						return
					}
					devices[dev.Serial] = &Device{
						LinkSystemHub: dev,
						ProductType:   productTypeLinkHub,
						Product:       dev.Product,
						Serial:        dev.Serial,
						Firmware:      dev.Firmware,
					}
				}(vendorId, productId, serial)
			}

		case 3122, 3100: // CORSAIR iCUE COMMANDER Core
			{
				go func(vendorId, productId uint16, serialId string) {
					dev := cc.Init(vendorId, productId, serialId)
					if dev == nil {
						return
					}
					devices[dev.Serial] = &Device{
						CC:          dev,
						ProductType: productTypeCC,
						Product:     dev.Product,
						Serial:      dev.Serial,
						Firmware:    dev.Firmware,
					}
				}(vendorId, productId, serial)
			}
		case 3114: // CORSAIR iCUE COMMANDER CORE XT
			{
				go func(vendorId, productId uint16, serialId string) {
					dev := ccxt.Init(vendorId, productId, serialId)
					if dev == nil {
						return
					}
					devices[dev.Serial] = &Device{
						CCXT:        dev,
						ProductType: productTypeCCXT,
						Product:     dev.Product,
						Serial:      dev.Serial,
						Firmware:    dev.Firmware,
					}
				}(vendorId, productId, serial)
			}
		case 3125, 3126, 3127: // CORSAIR iCUE H100i,H115i,H150i ELITE RGB
			{
				go func(vendorId, productId uint16) {
					dev := elite.Init(vendorId, productId)
					if dev == nil {
						return
					}
					devices[strconv.Itoa(int(productId))] = &Device{
						Elite:       dev,
						ProductType: productTypeElite,
						Product:     dev.Product,
						Serial:      dev.Serial,
						Firmware:    dev.Firmware,
					}
				}(vendorId, productId)
			}
		default:
			logger.Log(logger.Fields{"vendor": vendorId, "product": productId, "serial": serial}).Warn("Unsupported device detected. Please open a new feature request for your device on OpenLinkHub repository")
			continue
		}
	}
}
