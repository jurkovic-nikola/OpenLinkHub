package stats

type Device struct {
	Device            string
	TemperatureString string
	Speed             string
	Label             string
}

type DeviceList struct {
	Devices map[int]Device
}

type Stats struct {
	Stats map[string]DeviceList
}

var (
	stats = map[string]DeviceList{}
)

func Init() {
	stats = make(map[string]DeviceList)
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
