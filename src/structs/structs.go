package structs

import (
	"sync"
)

type Header struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type TemperatureCurve struct {
	Id         uint8   `json:"id"`
	Min        float32 `json:"min"`
	Max        float32 `json:"max"`
	Mode       uint8   `json:"mode"`
	Fans       uint16  `json:"fans"`
	Pump       uint16  `json:"pump"`
	ChannelIds []uint8 `json:"channelIds"`
	Color      Color   `json:"color"`
}

type Color struct {
	Red        float64 `json:"red"`
	Green      float64 `json:"green"`
	Blue       float64 `json:"blue"`
	Brightness float64 `json:"brightness"`
}

type Speed struct {
	Mode  uint8  `json:"mode"`
	Value uint16 `json:"value"`
}

type ChannelIdData struct {
	Color Color `json:"color"`
	Speed Speed `json:"speed"`
}

type RGBModes struct {
	Speed      uint8   `json:"speed"`
	Brightness float64 `json:"brightness"`
}

type Configuration struct {
	VendorId                     string                `json:"vendorId"`
	ProductId                    string                `json:"productId"`
	ListenPort                   int                   `json:"listenPort"`
	ListenAddress                string                `json:"listenAddress"`
	PullingIntervalMs            int                   `json:"pullingIntervalMs"`
	TemperaturePullingIntervalMs int                   `json:"temperaturePullingIntervalMs"`
	DefaultFanValue              int                   `json:"defaultFanValue"`
	DefaultPumpValue             int                   `json:"defaultPumpValue"`
	Standalone                   bool                  `json:"standalone"`
	CPUSensorChip                string                `json:"cpuSensorChip"`
	CPUPackageIdent              string                `json:"cpuPackageIdent"`
	Serial                       string                `json:"serial"`
	Headers                      []Header              `json:"headers"`
	TemperatureCurves            []TemperatureCurve    `json:"temperatureCurve"`
	DefaultColor                 Color                 `json:"defaultColor"`
	UseCustomChannelIdColor      bool                  `json:"useCustomChannelIdColor"`
	UseCustomChannelIdSpeed      bool                  `json:"useCustomChannelIdSpeed"`
	CustomChannelIdData          map[int]ChannelIdData `json:"customChannelIdData"`
	RGBMode                      string                `json:"rgbMode"`
	RGBModes                     map[string]RGBModes   `json:"rgbModes"`
}

// Device primary struct for a Corsair iCUE Link device
type Device struct {
	Manufacturer string             `json:"manufacturer"`
	Product      string             `json:"product"`
	Serial       string             `json:"serial"`
	Firmware     string             `json:"firmware"`
	Standalone   bool               `json:"standalone"`
	Devices      map[int]LinkDevice `json:"devices"`
}

// LinkDevice contains information about devices connected to an iCUE Link
type LinkDevice struct {
	ChannelId    int     `json:"channelId"`
	Type         byte    `json:"type"`
	DeviceId     string  `json:"deviceId"`
	Name         string  `json:"name"`
	DefaultValue byte    `json:"-"`
	Rpm          int16   `json:"rpm"`
	Temperature  float32 `json:"temperature"`
	LedChannels  uint8   `json:"-"`
	ContainsPump bool    `json:"-"`
}

// DeviceList contains definition of supported devices
type DeviceList struct {
	DeviceId    byte
	Model       byte
	Name        string
	LedChannels uint8
}

// SpeedSensor contains data about device RPM information
type SpeedSensor struct {
	ChannelId int
	Status    byte
	Rpm       int16
}

// TemperatureSensor contains data about device temperature information
type TemperatureSensor struct {
	ChannelId   int
	Status      byte
	Temperature float32
}

// Devices contains list of connected devices
type Devices struct {
	ChannelId   int
	DeviceId    string
	DeviceType  byte
	DeviceModel byte
}

// DeviceData contains response from a device
type DeviceData struct {
	Data []byte
}

// Payload contains data from a client about device speed change
type Payload struct {
	ChannelId int    `json:"channelId"`
	Mode      uint8  `json:"mode"`
	Value     uint16 `json:"value"`
	Color     Color  `json:"color"`
	Code      int
	Message   string
}

// Response contains data what is sent back to a client
type Response struct {
	sync.Mutex
	Code    int         `json:"code"`
	Message string      `json:"message,omitempty"`
	Device  interface{} `json:"device,omitempty"`
	Devices interface{} `json:"devices,omitempty"`
}

type HSL struct {
	H, S, L float64
}

// DeviceMonitor struct contains the shared variable and synchronization primitives
type DeviceMonitor struct {
	Status byte
	Lock   sync.Mutex
	Cond   *sync.Cond
}
