package sabrev2prodongle

// Package: SABRE V2 PRO Wireless Dongle
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/devices/m55W"
	"OpenLinkHub/src/devices/sabrev2proW"
	"OpenLinkHub/src/inputmanager"
	"OpenLinkHub/src/logger"
	"github.com/sstallion/go-hid"
	"os"
	"strconv"
	"sync"
	"time"
)

type Devices struct {
	Type      byte   `json:"type"`
	Endpoint  byte   `json:"endpoint"`
	Serial    string `json:"serial"`
	VendorId  uint16 `json:"deviceId"`
	ProductId uint16 `json:"productId"`
}

type Device struct {
	dev            *hid.Device
	listener       *hid.Device
	mouse          *os.File
	Manufacturer   string `json:"manufacturer"`
	Product        string `json:"product"`
	Serial         string `json:"serial"`
	Firmware       string `json:"firmware"`
	ProductId      uint16
	VendorId       uint16
	Devices        map[int]*Devices `json:"devices"`
	PairedDevices  map[uint16]any
	SharedDevices  func(device *common.Device)
	DeviceList     map[string]*common.Device
	Template       string
	Debug          bool
	Exit           bool
	timerKeepAlive *time.Ticker
	timerSleep     *time.Ticker
	keepAliveChan  chan struct{}
	sleepChan      chan struct{}
	mutex          sync.Mutex
	deviceLock     sync.Mutex
	instance       *common.Device
}

const (
	EvKey      = 0x01
	BtnLeft    = 0x110
	BtnRight   = 0x111
	BtnMiddle  = 0x112
	BtnBack    = 0x113 // 275
	BtnForward = 0x114 // 276
)

var (
	bufferSize      = 64
	bufferSizeWrite = bufferSize + 1
	headerSize      = 2
	deviceKeepAlive = 2000
	cmdReadDongle   = []byte{0x03}
	cmdRead         = []byte{0x04}
	transferTimeout = 200
	mouseProductId  = uint16(11048)
)

func Init(vendorId, productId uint16, _, path string, callback func(device *common.Device)) *common.Device {
	// Open device, return if failure
	dev, err := hid.OpenPath(path)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": vendorId, "productId": productId, "path": path}).Error("Unable to open HID device")
		return nil
	}

	// Init new struct with HID device
	d := &Device{
		dev:            dev,
		VendorId:       vendorId,
		ProductId:      productId,
		PairedDevices:  make(map[uint16]any),
		DeviceList:     make(map[string]*common.Device),
		Template:       "slipstream.html",
		sleepChan:      make(chan struct{}),
		keepAliveChan:  make(chan struct{}),
		timerSleep:     &time.Ticker{},
		timerKeepAlive: &time.Ticker{},
		SharedDevices:  callback,
	}

	d.getDebugMode()    // Debug
	d.getManufacturer() // Manufacturer
	d.getProduct()      // Product
	d.getSerial()       // Serial
	d.getDevices()      // Get devices
	d.addDevices()      // Add devices
	d.monitorDevice()   // Monitor device
	d.mouseListener()   // Mouse listener
	d.createDevice()    // Device register
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device successfully initialized")

	return d.instance
}

// createDevice will create new device register object
func (d *Device) createDevice() {
	d.instance = &common.Device{
		ProductType: common.ProductTypeSlipstream,
		Product:     "SLIPSTREAM",
		Serial:      d.Serial,
		Firmware:    d.Firmware,
		Image:       "icon-dongle.svg",
		Instance:    d,
		Hidden:      true,
	}
}

// addDevices will add list of available devices
func (d *Device) addDevices() {
	for _, value := range d.Devices {
		switch value.ProductId {
		case 11048: // SABRE V2 PRO
			{
				dev := sabrev2proW.Init(
					value.VendorId,
					d.ProductId,
					value.ProductId,
					d.dev,
					value.Endpoint,
					value.Serial,
				)

				object := &common.Device{
					ProductType: common.ProductTypeSabreV2Pro,
					Product:     "SABRE V2 PRO",
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-mouse.svg",
					Instance:    dev,
					DeviceType:  common.DeviceTypeMouse,
				}
				d.SharedDevices(object)
				d.AddPairedDevice(value.ProductId, dev, object)
			}
		default:
			logger.Log(logger.Fields{"productId": value.ProductId}).Warn("Unsupported device detected")
		}
	}
}

// StopDirty will stop devices in a dirty way
func (d *Device) StopDirty() uint8 {
	d.Exit = true
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Stopping device (dirty)...")

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

	for _, value := range d.PairedDevices {
		if dev, found := value.(*m55W.Device); found {
			if dev.Connected {
				dev.StopDirty()
			}
		}
	}
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device stopped")
	return 2
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

	for _, value := range d.PairedDevices {
		if dev, found := value.(*m55W.Device); found {
			if dev.Connected {
				dev.StopInternal()
			}
		}
	}

	time.Sleep(500 * time.Millisecond)
	if d.dev != nil {
		err := d.dev.Close()
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to close HID device")
		}
	}
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device stopped")
}

// getManufacturer will return device manufacturer
func (d *Device) getDebugMode() {
	d.Debug = config.GetConfig().Debug
}

// GetDeviceTemplate will return device template name
func (d *Device) GetDeviceTemplate() string {
	return d.Template
}

// AddPairedDevice will add a paired device
func (d *Device) AddPairedDevice(productId uint16, device any, dev *common.Device) {
	d.PairedDevices[productId] = device
	d.DeviceList[dev.Serial] = dev
}

// setDeviceOnlineByProductId will online device by given productId
func (d *Device) setDeviceOnlineByProductId(productId uint16) {
	if dev, ok := d.PairedDevices[productId]; ok {
		if device, found := dev.(*sabrev2proW.Device); found {
			if !device.Connected {
				device.Connect()
				d.SharedDevices(d.DeviceList[device.Serial])
			}
		}
	}
}

// setDeviceOnlineByProductId will offline device by given productId
func (d *Device) setDeviceOfflineByProductId(productId uint16) {
	if dev, ok := d.PairedDevices[productId]; ok {
		if device, found := dev.(*sabrev2proW.Device); found {
			if device.Connected {
				device.SetConnected(false)
			}
		}
	}
}

// setDeviceBatteryByProductId will set device battery by given productId
func (d *Device) setDeviceBatteryByProductId(productId, batteryLevel uint16) {
	if dev, ok := d.PairedDevices[productId]; ok {
		if device, found := dev.(*sabrev2proW.Device); found {
			if device.Connected {
				device.ModifyBatteryLevel(batteryLevel)
			}
		}
	}
}

// GetDevice will return HID device
func (d *Device) GetDevice() *hid.Device {
	return d.dev
}

// getDevices will get a list of paired devices
func (d *Device) getDevices() {
	var devices = make(map[int]*Devices)

	devices[0] = &Devices{
		Type:      1,
		Endpoint:  0x08,
		Serial:    strconv.Itoa(int(mouseProductId)),
		VendorId:  d.VendorId,
		ProductId: mouseProductId,
	}
	d.Devices = devices
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

// getProduct will return device name
func (d *Device) getProduct() {
	product, err := d.dev.GetProductStr()
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to get product")
	}
	d.Product = product
}

// monitorDevice will refresh device data
func (d *Device) monitorDevice() {
	d.timerKeepAlive = time.NewTicker(time.Duration(deviceKeepAlive) * time.Millisecond)
	go func() {
		for {
			select {
			case <-d.timerKeepAlive.C:
				{
					d.deviceLock.Lock()
					if d.Exit {
						return
					}

					buf := make([]byte, 15)
					buf[14] = 0x4a
					status, err := d.transfer(cmdReadDongle, buf)
					if err != nil {
						logger.Log(logger.Fields{"error": err}).Error("Unable to get battery level")
					}

					if status[1] == 0x03 {
						switch status[6] {
						case 1:
							d.setDeviceOnlineByProductId(mouseProductId)
						case 0:
							d.setDeviceOfflineByProductId(mouseProductId)
						}
					}
					time.Sleep(time.Duration(transferTimeout) * time.Millisecond)
					if dev, ok := d.PairedDevices[mouseProductId]; ok {
						if device, found := dev.(*sabrev2proW.Device); found {
							if device.Connected {
								b := make([]byte, 15)
								b[14] = 0x49
								batteryStatus, err2 := d.transfer(cmdRead, b)
								if err2 != nil {
									logger.Log(logger.Fields{"error": err}).Error("Unable to get battery level")
								}
								d.setDeviceBatteryByProductId(mouseProductId, uint16(batteryStatus[6]))
							}
						}
					}
					d.deviceLock.Unlock()
				}
			case <-d.keepAliveChan:
				return
			}
		}
	}()
}

// transfer will send data to a device and retrieve device output
func (d *Device) transfer(endpoint, buffer []byte) ([]byte, error) {
	// Packet control, mandatory for this device
	d.mutex.Lock()
	defer d.mutex.Unlock()

	// Create write buffer
	bufferW := make([]byte, bufferSizeWrite)
	bufferW[1] = 0x08
	endpointHeaderPosition := bufferW[headerSize : headerSize+len(endpoint)]
	copy(endpointHeaderPosition, endpoint)
	if len(buffer) > 0 {
		copy(bufferW[headerSize+len(endpoint):headerSize+len(endpoint)+len(buffer)], buffer)
	}

	reports := make([]byte, 1)
	err := d.dev.SetNonblock(true)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to SetNonblock")
	}

	for {
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
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to read data from device")
		return bufferR, err
	}
	return bufferR, nil
}

// transfer will send data to a device and retrieve device output
func (d *Device) transferTimeout(endpoint, buffer []byte) ([]byte, error) {
	// Packet control, mandatory for this device
	d.mutex.Lock()
	defer d.mutex.Unlock()

	// Create write buffer
	bufferW := make([]byte, bufferSizeWrite)
	bufferW[1] = 0x08
	endpointHeaderPosition := bufferW[headerSize : headerSize+len(endpoint)]
	copy(endpointHeaderPosition, endpoint)
	if len(buffer) > 0 {
		copy(bufferW[headerSize+len(endpoint):headerSize+len(endpoint)+len(buffer)], buffer)
	}

	reports := make([]byte, 1)
	err := d.dev.SetNonblock(true)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to SetNonblock")
	}

	for {
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
	if _, err := d.dev.ReadWithTimeout(bufferR, time.Duration(transferTimeout)*time.Millisecond); err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to read data from device")
		return bufferR, err
	}
	return bufferR, nil
}

func (d *Device) findDevice() *sabrev2proW.Device {
	for _, v := range d.PairedDevices {
		if dev, ok := v.(*sabrev2proW.Device); ok && dev != nil {
			return dev
		}
	}
	return nil
}

// mouseListener will listen for events from the control buttons
func (d *Device) mouseListener() {
	if inputmanager.GetVirtualMouse() == nil {
		logger.Log(logger.Fields{"serial": d.Serial}).Warn("Virtual mouse is not available. Key AssignmentS are blocked")
	}

	go func() {
		inputEvent := ""
		enum := hid.EnumFunc(func(info *hid.DeviceInfo) error {
			if info.InterfaceNbr == 0 {
				events, err := common.FindEventsByHidraw(info.Path)
				if err != nil {
					return err
				}
				if len(events) > 0 {
					inputEvent = events[0]
				}
			}
			return nil
		})

		err := hid.Enumerate(d.VendorId, d.ProductId, enum)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to enumerate devices")
		}

		if len(inputEvent) > 0 {
			f, err := os.OpenFile(inputEvent, os.O_RDONLY, 0)
			if err != nil {
				logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to open input event listener")
				return
			}
			err = inputmanager.IocTlInt(f.Fd(), inputmanager.EvIocGrab, 1)
			if err != nil {
				_ = f.Close()
				logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("EvIocGrab failed")
				return
			}
			d.mouse = f
		}

		var dev = d.findDevice()
		if d.mouse != nil {
			for {
				if dev == nil {
					continue
				}

				select {
				default:
					if d.Exit {
						err = d.mouse.Close()
						if err != nil {
							logger.Log(logger.Fields{"error": err, "vendorId": d.VendorId}).Error("Failed to close listener")
							return
						}
						return
					}

					ev, err := inputmanager.ReadEvent(d.mouse)
					if err != nil {
						return
					}

					// Buttons
					if ev.Type == EvKey && (ev.Code == BtnBack || ev.Code == BtnForward || ev.Code == BtnMiddle || ev.Code == BtnLeft || ev.Code == BtnRight) {
						var val byte = 0
						switch ev.Code {
						case 272:
							val = 1
						case 273:
							val = 2
						case 274:
							val = 4
						case 275:
							val = 8
						case 276:
							val = 16
						}

						if ev.Value == 1 {
							dev.TriggerKeyAssignment(val)
						} else {
							dev.TriggerKeyAssignment(0)
						}
					}

					// Mouse position
					if ev.Type == 0x02 {
						if ev.Code == 0x08 {
							direction := ev.Value == 0x01
							inputmanager.InputControlScroll(direction)
						} else {
							val := map[int]int32{}

							if ev.Code == 0 {
								val[0] = ev.Value
							}

							if ev.Code == 1 {
								val[1] = ev.Value
							}
							inputmanager.InputControlMove(val[0], val[1])
						}
					}
				}
			}
		}
	}()
}
