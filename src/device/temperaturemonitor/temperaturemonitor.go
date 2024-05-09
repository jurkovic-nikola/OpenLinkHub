package temperaturemonitor

import (
	"OpenICUELinkHub/src/config"
	"OpenICUELinkHub/src/device"
	"OpenICUELinkHub/src/logger"
	"OpenICUELinkHub/src/structs"
	"github.com/ssimunic/gosensors"
	"strconv"
	"time"
)

var (
	temperatureCurves []structs.TemperatureCurve
	quit              chan bool
)

func GetCpuTemperature() float32 {
	if config.GetConfig().Standalone {
		sensors, err := gosensors.NewFromSystem()
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Warn("Unable to find sensors. You are probably missing lm-sensors package!")
			return 0
		}

		for chip := range sensors.Chips {
			if chip == config.GetConfig().CPUSensorChip {
				if val, ok := sensors.Chips[chip][config.GetConfig().CPUPackageIdent]; ok {
					val := val[1 : len(val)-3]
					value, err := strconv.ParseFloat(val, 32)
					if err != nil {
						return 0
					}
					return float32(value)
				}
			}
		}
	}

	return 0
}

// Init will initialize CPU temperature monitor
func Init() {
	if config.GetConfig().Standalone {
		var currentCurve uint8 = 0
		temperatureCurves = config.GetConfig().TemperatureCurves
		ticker := time.NewTicker(time.Duration(config.GetConfig().TemperaturePullingIntervalMs) * time.Millisecond)
		quit = make(chan bool)
		go func() {
			for {
				select {
				case <-ticker.C:
					temp := GetCpuTemperature()
					if temp > 0 {
						for i := 0; i < len(temperatureCurves); i++ {
							curve := temperatureCurves[i]
							if InBetween(temp, curve.Min, curve.Max) {
								if currentCurve != curve.Id { // Change device speed only if a curve changes
									currentCurve = curve.Id
									// Limits for minimum and maximum pump operations
									if curve.Pump < 50 {
										curve.Pump = 50
									}

									if curve.Pump > 100 {
										curve.Pump = 100
									}

									if device.GetDevice() == nil {
										return
									}

									// Custom color is defined on a temperature curve.
									// This is when a certain temperature is reached, and the user needs to know that.
									// You can also use this as different lightning for the temperature range of a CPU.
									if (curve.Color == structs.Color{}) {
										// No defined color, go back to default
										device.SetDeviceColor(nil)
									} else {
										// Color is defined, override everything else
										color := &structs.Color{
											Red:        curve.Color.Red,
											Green:      curve.Color.Green,
											Blue:       curve.Color.Blue,
											Brightness: curve.Color.Brightness,
										}
										device.SetDeviceColor(color)
									}

									if len(curve.ChannelIds) > 0 {
										// Custom IDs
										for ch := range curve.ChannelIds {
											if dev, ok := device.GetDevice().Devices[int(curve.ChannelIds[ch])]; ok {
												if dev.ContainsPump {
													device.SetDeviceSpeed(dev.ChannelId, curve.Pump, 0)
												} else {
													device.SetDeviceSpeed(dev.ChannelId, curve.Fans, curve.Mode)
												}
											}
										}
									} else {
										// All channels
										for _, dev := range device.GetDevice().Devices {
											if dev.ContainsPump { // AIO is always in percent mode
												device.SetDeviceSpeed(dev.ChannelId, curve.Pump, 0)
											} else {
												device.SetDeviceSpeed(dev.ChannelId, curve.Fans, curve.Mode)
											}
										}
									}
								}
							}
						}
					}
				case <-quit:
					ticker.Stop()
					return
				}
			}
		}()
	} else {
		logger.Log(logger.Fields{}).Info("Temperature monitor is inactive due to standalone being set to false")
	}
}

func Stop() {
	if config.GetConfig().Standalone {
		quit <- true
	}
}
func InBetween(i, min, max float32) bool {
	if (i >= min) && (i <= max) {
		return true
	} else {
		return false
	}
}
