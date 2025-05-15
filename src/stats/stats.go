package stats

type Device struct {
	Device            string
	TemperatureString string
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
	stats        = map[string]DeviceList{}
	batteryStats = map[string]BatteryStats{}
)

func Init() {
	stats = make(map[string]DeviceList)
	batteryStats = make(map[string]BatteryStats)
}

// UpdateBatteryStats will update battery stats
func UpdateBatteryStats(serial, device string, level uint16, deviceType uint8) {
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

// UpdateAIOStats will update AIO stats
func UpdateAIOStats(serial, name, temp, speed, label string, channelId int) {
	if data, ok := stats[serial]; ok {
		data.Devices[channelId] = Device{
			Device:            name,
			TemperatureString: temp,
			Speed:             speed,
		}
		stats[serial] = data
	} else {
		stats[serial] = DeviceList{
			Devices: map[int]Device{
				channelId: {
					Device:            name,
					TemperatureString: temp,
					Speed:             speed,
				},
			},
		}
	}
}

// GetAIOStats will return AIO stats
func GetAIOStats() map[string]DeviceList {
	return stats
}

// GetAIOData will return AIO data
func GetAIOData(serial string, channelId int) *Device {
	if value, ok := stats[serial]; ok {
		if data, found := value.Devices[channelId]; found {
			return &data
		}
	}
	return nil
}

// GetBatteryStats will return battery stats
func GetBatteryStats() map[string]BatteryStats {
	return batteryStats
}
