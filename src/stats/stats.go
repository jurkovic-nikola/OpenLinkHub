package stats

// Package: stats
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import "sync"

type Device struct {
	Device            string
	TemperatureString string
	Temperature       float32
	Speed             string
	Label             string
}

type BatteryStats struct {
	Device     string
	Level      uint16
	DeviceType uint8
}

type DeviceList struct {
	Devices map[int]Device
}

type Stats struct {
	Stats map[string]DeviceList
}

var (
	stats             = map[string]DeviceList{}
	statsMutex        sync.RWMutex
	batteryStats      = map[string]BatteryStats{}
	batteryStatsMutex sync.RWMutex
)

func Init() {
	stats = make(map[string]DeviceList)
	batteryStats = make(map[string]BatteryStats)
}

// UpdateBatteryStats will update battery stats
func UpdateBatteryStats(serial, device string, level uint16, deviceType uint8) {
	batteryStatsMutex.Lock()
	defer batteryStatsMutex.Unlock()

	if data, ok := batteryStats[serial]; ok {
		data.Level = level
		data.DeviceType = deviceType
		data.Device = device
		batteryStats[serial] = data
	} else {
		batteryStats[serial] = BatteryStats{
			Device:     device,
			Level:      level,
			DeviceType: deviceType,
		}
	}
}

// UpdateDeviceStats will update device stats
func UpdateDeviceStats(serial, name, temp, speed, label string, channelId int, temperature float32) {
	statsMutex.Lock()
	defer statsMutex.Unlock()

	if data, ok := stats[serial]; ok {
		data.Devices[channelId] = Device{
			Device:            name,
			TemperatureString: temp,
			Temperature:       temperature,
			Speed:             speed,
		}
		stats[serial] = data
	} else {
		stats[serial] = DeviceList{
			Devices: map[int]Device{
				channelId: {
					Device:            name,
					TemperatureString: temp,
					Temperature:       temperature,
					Speed:             speed,
				},
			},
		}
	}
}

// GetDeviceTemperature will return temperature for given device and channel
func GetDeviceTemperature(serial string, channelId int) float32 {
	statsMutex.RLock()
	defer statsMutex.RUnlock()

	if data, ok := stats[serial]; ok {
		if value, found := data.Devices[channelId]; found {
			return value.Temperature
		}
	}
	return 0
}

// GetAIOStats will return AIO stats
func GetAIOStats() map[string]DeviceList {
	//return stats
	statsMutex.RLock()
	defer statsMutex.RUnlock()

	cp := make(map[string]DeviceList, len(stats))
	for key, value := range stats {
		device := make(map[int]Device, len(value.Devices))
		for dk, dv := range value.Devices {
			device[dk] = dv
		}
		cp[key] = DeviceList{Devices: device}
	}
	return cp
}

// GetAIOData will return AIO data
func GetAIOData(serial string, channelId int) *Device {
	statsMutex.RLock()
	defer statsMutex.RUnlock()

	if value, ok := stats[serial]; ok {
		if data, found := value.Devices[channelId]; found {
			return &data
		}
	}
	return nil
}

// GetBatteryStats will return battery stats
func GetBatteryStats() map[string]BatteryStats {
	batteryStatsMutex.RLock()
	defer batteryStatsMutex.RUnlock()

	cp := make(map[string]BatteryStats, len(batteryStats))
	for key, value := range batteryStats {
		cp[key] = value
	}
	return cp
}
