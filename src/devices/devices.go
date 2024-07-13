package devices

import (
	"OpenLinkHub/src/devices/linksystemhub"
	"OpenLinkHub/src/logger"
	"fmt"
	"github.com/sstallion/go-hid"
)

const (
	productTypeLinkHub = 0
)

type Device struct {
	ProductType   uint8
	Product       string
	Serial        string
	Firmware      string
	LinkSystemHub *linksystemhub.Device `json:"linkSystemHub,omitempty"`
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
		}
	}
	err := hid.Exit()
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to exit HID interface")
	}
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
		}
	}
}

// UpdateManualSpeed will update device speeds with a given serial number
func UpdateManualSpeed(deviceId string, channelId int, value uint16) {
	if device, ok := devices[deviceId]; ok {
		switch device.ProductType {
		case productTypeLinkHub:
			{
				if device.LinkSystemHub != nil {
					device.LinkSystemHub.UpdateDeviceSpeed(channelId, value)
				}
			}
		}
	}
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
		default:
			logger.Log(logger.Fields{"vendor": vendorId, "product": productId, "serial": serial}).Warn("Unsupported device detected. Please open a new feature request for your device on OpenLinkHub repository")
			continue
		}
	}
}
