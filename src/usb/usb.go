package usb

// Package: usb
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/common"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

var (
	usbDirOut              = 0x00
	usbTypeVendor          = 0x40
	usbRecipDevice         = 0x00
	usbdevfsControl        = 0xC0105500
	iocNrbits              = 8
	iocTypebits            = 8
	iocSizebits            = 14
	iocNrshift             = 0
	iocTypeshift           = iocNrshift + iocNrbits
	iocSizeshift           = iocTypeshift + iocTypebits
	iocDirshift            = iocSizeshift + iocSizebits
	iocWrite               = 1
	iocRead                = 2
	iocReadwrite           = iocRead | iocWrite
	endpointBulkOut        = 0x01
	endpointBulkIn         = 0x81 // Example bulk IN endpoint
	corsairVendorId uint16 = 6940
)

var devices []DeviceStruct

type DeviceStruct struct {
	Name         string
	VendorId     uint16
	ProductID    uint16
	SerialNbr    string
	Path         string
	InterfaceNbr int
	Manufacturer string
}

type Device struct {
	devicePath   string
	file         *os.File
	dev          *DeviceStruct
	blocking     bool
	timeout      int
	UsbdevfsBulk uintptr
}

// usbCtrlSetup matches the layout expected by the USB kernel interface
type usbCtrlSetup struct {
	RequestType uint8
	Request     uint8
	Value       uint16
	Index       uint16
	Length      uint16
	Data        uintptr
}

type usbdevfsBulkTransfer struct {
	Ep      uint32  // Endpoint number
	Len     uint32  // Length of data
	Timeout uint32  // Timeout in milliseconds
	Data    uintptr // Pointer to data buffer
}

func Init(deviceList []uint16) int {
	basePath := "/sys/bus/usb/devices"
	_ = filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		vendorFile := filepath.Join(path, "idVendor")
		productFile := filepath.Join(path, "idProduct")

		vendorData, err1 := os.ReadFile(vendorFile)
		productData, err2 := os.ReadFile(productFile)

		if err1 == nil && err2 == nil {
			vid := strings.TrimSpace(string(vendorData))
			pid := strings.TrimSpace(string(productData))

			// Convert VID / PID to uint16
			productId := common.PidVidToUint16(pid)
			vendorId := common.PidVidToUint16(vid)

			// Corsair vendor and legacy device list
			if vendorId == corsairVendorId && slices.Contains(deviceList, productId) {
				serial := ""
				deviceName := ""
				serialFile := filepath.Join(path, "serial")
				serialData, errSerial := os.ReadFile(serialFile)
				if errSerial == nil {
					serial = strings.TrimSpace(string(serialData))
				}

				// Devices without serial, we use PID as a serial to be unique
				if len(serial) == 0 {
					serial = strconv.Itoa(int(productId))
				}

				// Remove _ and . in serial number
				serial = strings.Replace(serial, "_", "", -1)
				serial = strings.Replace(serial, ".", "", -1)

				// Optional: Get product (device) name
				productNameFile := filepath.Join(path, "product")
				productNameData, errProduct := os.ReadFile(productNameFile)
				if errProduct == nil {
					deviceName = strings.TrimSpace(string(productNameData))
				}

				// Get bus and device number
				busnumPath := filepath.Join(path, "busnum")
				devnumPath := filepath.Join(path, "devnum")

				busnumData, errBus := os.ReadFile(busnumPath)
				devnumData, errDev := os.ReadFile(devnumPath)

				if errBus == nil && errDev == nil {
					bus := fmt.Sprintf("%03s", strings.TrimSpace(string(busnumData)))
					device := fmt.Sprintf("%03s", strings.TrimSpace(string(devnumData)))
					devicePath := fmt.Sprintf("/dev/bus/usb/%s/%s", bus, device)
					dev := DeviceStruct{
						Name:         deviceName,
						VendorId:     vendorId,
						ProductID:    productId,
						SerialNbr:    serial,
						Path:         devicePath,
						InterfaceNbr: 0,
						Manufacturer: "Corsair",
					}
					devices = append(devices, dev)
				}
			}
		}
		return nil
	})
	return len(devices)
}

// ioc calculates the ioctl request number
func ioc(dir, typ, nr, size uintptr) uintptr {
	return (dir << iocDirshift) |
		(typ << iocTypeshift) |
		(nr << iocNrshift) |
		(size << iocSizeshift)
}

// Open opens a USB device
func Open(devicePath string) (*Device, error) {
	file, err := os.OpenFile(devicePath, syscall.O_RDWR, 0660)
	if err != nil {
		return nil, fmt.Errorf("failed to open device %s: %v", devicePath, err)
	}

	const sizeOfUsbdevfsBulkTransfer = unsafe.Sizeof(usbdevfsBulkTransfer{})

	return &Device{
		devicePath:   devicePath,
		file:         file,
		dev:          GetDevicesByPath(devicePath),
		UsbdevfsBulk: ioc(uintptr(iocReadwrite), 'U', 2, sizeOfUsbdevfsBulkTransfer),
	}, nil
}

// GetDevices will return all devices
func GetDevices() []DeviceStruct {
	return devices
}

// GetDevicesByPath will return a device by path
func GetDevicesByPath(path string) *DeviceStruct {
	for _, value := range devices {
		if value.Path == path {
			return &value
		}
	}
	return nil
}

// Close closes the USB device
func (h *Device) Close() error {
	return h.file.Close()
}

// GetSerialNbr returns serial number
func (h *Device) GetSerialNbr() string {
	return h.dev.SerialNbr
}

// GetMfrStr returns device vendor
func (h *Device) GetMfrStr() string {
	return h.dev.Manufacturer
}

// GetProductStr returns device name
func (h *Device) GetProductStr() string {
	product := strings.Replace(h.dev.Name, "CORSAIR ", "", -1)
	product = strings.Replace(product, "Corsair ", "", -1)
	product = strings.Replace(product, " CPU Water Block", "", -1)
	return product
}

// SetEndpoints sets device endpoint
func (h *Device) SetEndpoints(endpointOut, endpointIn int) {
	endpointBulkOut = endpointOut
	endpointBulkIn = endpointIn
}

// SetDeviceControl will send control packet to the device and return err if any.
func (h *Device) SetDeviceControl(request uint8, value, index, length uint16, data uintptr) error {
	ctrl := usbCtrlSetup{
		RequestType: uint8(usbTypeVendor | usbDirOut | usbRecipDevice),
		Request:     request,
		Value:       value,
		Index:       index,
		Length:      length,
		Data:        data,
	}
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, h.file.Fd(), uintptr(usbdevfsControl), uintptr(unsafe.Pointer(&ctrl)))
	if errno != 0 {
		return fmt.Errorf("failed to send control packet: %v", errno)
	}
	return nil
}

// Write writes data to the USB device via syscall
func (h *Device) Write(buffer []byte) error {
	bulkTransfer := usbdevfsBulkTransfer{
		Ep:      uint32(endpointBulkOut),
		Len:     uint32(len(buffer)),
		Timeout: 1000,
		Data:    uintptr(unsafe.Pointer(&buffer[0])),
	}
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, h.file.Fd(), h.UsbdevfsBulk, uintptr(unsafe.Pointer(&bulkTransfer)))
	if errno != 0 {
		return fmt.Errorf("failed to write to device %s: %v", h.devicePath, errno)
	}
	return nil
}

// Read reads data from the USB device via syscall
func (h *Device) Read(buffer []byte) error {
	bulkTransfer := usbdevfsBulkTransfer{
		Ep:      uint32(endpointBulkIn),
		Len:     uint32(len(buffer)),
		Timeout: 1000,                                // Timeout in milliseconds
		Data:    uintptr(unsafe.Pointer(&buffer[0])), // Pointer to the read buffer
	}
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, h.file.Fd(), h.UsbdevfsBulk, uintptr(unsafe.Pointer(&bulkTransfer)))
	if errno != 0 {
		return fmt.Errorf("failed to read from device %s: %v", h.devicePath, errno)
	}
	return nil
}

// ReadNonBlock performs a quick, timeout-based nonblocking read.
func (h *Device) ReadNonBlock(buffer []byte, timeoutMs uint32) error {
	bulkTransfer := usbdevfsBulkTransfer{
		Ep:      uint32(endpointBulkIn),
		Len:     uint32(len(buffer)),
		Timeout: timeoutMs,                           // Timeout in milliseconds
		Data:    uintptr(unsafe.Pointer(&buffer[0])), // pointer to buffer
	}

	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		h.file.Fd(),
		h.UsbdevfsBulk,
		uintptr(unsafe.Pointer(&bulkTransfer)),
	)
	if errno != 0 {
		if errors.Is(errno, syscall.ETIMEDOUT) {
			return nil
		}
		return fmt.Errorf("failed to read from device %s: %v", h.devicePath, errno)
	}
	return nil
}
