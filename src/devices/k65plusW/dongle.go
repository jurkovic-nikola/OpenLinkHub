package k65plusW

import (
	"OpenLinkHub/src/inputmanager"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/stats"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/sstallion/go-hid"
)

// getDeviceSerial will get device serial
func (d *Device) getDeviceSerial() {
	serial, err := d.read(cmdGetDevices)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to get device serial")
		return
	}
	serialLen := serial[10]
	serial = serial[11 : 11+serialLen]
	d.Serial = string(serial)
}

// getDongleFirmware will return a dongle firmware version out as string
func (d *Device) getDongleFirmware() {
	fw, err := d.transfer(
		cmdGetFirmware,
		nil,
		byte(cmdDongle),
	)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to write to a device")
	}

	v1, v2, v3 := int(fw[3]), int(fw[4]), int(binary.LittleEndian.Uint16(fw[5:7]))
	d.DongleFirmware = fmt.Sprintf("%d.%d.%d", v1, v2, v3)
}

// read will read data from given endpoint
func (d *Device) read(endpoint []byte) ([]byte, error) {
	buf := make([]byte, 64)
	_, err := d.transfer(endpoint, nil, byte(cmdDongle))
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to write to a device dongle")
		return buf, err
	}

	_, err = d.transfer(dataTypeGetData, nil, byte(cmdDongle))
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to write to a device dongle")
		return buf, err
	}

	buf, err = d.transfer(cmdRead, nil, byte(cmdDongle))
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to write to a device dongle")
		return buf, err
	}

	_, err = d.transfer(cmdCloseDingleEndpoint, nil, byte(cmdDongle))
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to write to a device dongle")
		return buf, err
	}
	return buf, nil
}

// transfer will send data to a device and retrieve device output
func (d *Device) transfer(endpoint, buffer []byte, command byte) ([]byte, error) {
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
			logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to enumerate devices")
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
						stats.UpdateBatteryStats(d.Serial, d.Product, d.BatteryLevel, 0)
					}
				}

				// Connection status
				if data[2] == 0x36 {
					switch data[4] {
					case 0x02:
						{
							if d.Connected == false {
								d.setConnectionStatus(true)
							}
						}
					case 0x00:
						{
							if d.Connected == true {
								d.setConnectionStatus(false)
							}
						}
					}
				}

				// FN color change
				functionKey := data[17] == 0x04
				if functionKey != d.FunctionKey {
					d.FunctionKey = functionKey
				}

				var modifierKey uint8 = 0
				modifierIndex := data[d.getModifierPosition()]
				if modifierIndex > 0 {
					modifierKey = d.getModifierKey(modifierIndex)
				}

				if data[1] == 0x02 {
					d.triggerKeyAssignment(data, functionKey, modifierKey)
				} else if data[1] == 0x05 {
					// Knob
					value := data[4]
					switch d.DeviceProfile.ControlDial {
					case 1:
						{
							switch value {
							case 1:
								inputmanager.InputControlKeyboard(inputmanager.VolumeUp, false)
								break
							case 255:
								inputmanager.InputControlKeyboard(inputmanager.VolumeDown, false)
								break
							}
						}
					case 2:
						{
							switch value {
							case 1:
								if d.DeviceProfile.BrightnessLevel+100 > 1000 {
									d.DeviceProfile.BrightnessLevel = 1000
								} else {
									d.DeviceProfile.BrightnessLevel += 100
								}
							case 255:
								if d.DeviceProfile.BrightnessLevel < 100 {
									d.DeviceProfile.BrightnessLevel = 0
								} else {
									d.DeviceProfile.BrightnessLevel -= 100
								}
							}
							d.saveDeviceProfile()
							d.setBrightnessLevel()
						}
					case 3:
						{
							switch value {
							case 1:
								inputmanager.InputControlScroll(false)
							case 255:
								inputmanager.InputControlScroll(true)
							}
						}
					case 4:
						{
							switch value {
							case 1:
								inputmanager.InputControlZoom(true)
							case 255:
								inputmanager.InputControlZoom(false)
							}
						}
					case 5:
						{
							switch value {
							case 1:
								inputmanager.InputControlKeyboard(inputmanager.KeyScreenBrightnessUp, false)
							case 255:
								inputmanager.InputControlKeyboard(inputmanager.KeyScreenBrightnessDown, false)
							}
						}
					}
				}
			}
		}
	}()
}

// checkIfAlive will check initial keyboard status on initialization
func (d *Device) checkIfAlive() {
	msg, err := d.transfer([]byte{0x12}, nil, byte(cmdKeyboard))
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to a device")
	}
	if msg != nil && len(msg) > 3 {
		if msg[2] == 0x00 {
			d.Connected = true
		}
	}
}

// keepAlive will keep a device alive
func (d *Device) keepAlive() {
	if d.Exit {
		return
	}
	_, err := d.transfer(cmdKeepAlive, nil, byte(cmdDongle))
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to a device")
	}
	if d.Exit {
		return
	}
	_, err = d.transfer(cmdKeepAlive, nil, byte(cmdKeyboard))
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to a device")
	}
}
