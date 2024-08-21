package metrics

import (
	"OpenLinkHub/src/systeminfo"
	"OpenLinkHub/src/temperatures"
	"github.com/prometheus/client_golang/prometheus"
)

type Header struct {
	Product          string
	Serial           string
	Firmware         string
	ChannelId        string
	Name             string
	Description      string
	Profile          string
	Label            string
	RGB              string
	AIO              string
	ContainsPump     string
	TemperatureProbe string
	LedChannels      string
	HwmonDevice      string
	Temperature      float64
	Rpm              int16
}

var (
	productGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "openlinkhub",
			Help: "Product information",
		},
		[]string{"product", "serial", "firmware"},
	)

	temperatureGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "openlinkhub_temperature",
			Help: "Current temperature of devices.",
		},
		[]string{"serial", "channelId", "name", "description", "profile", "label", "rgb", "aio", "pump", "probe", "led"},
	)

	rpmGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "openlinkhub_speed",
			Help: "Current speed (RPM) of devices.",
		},
		[]string{"serial", "channelId", "name", "description", "profile", "label", "rgb", "aio", "pump", "probe", "led"},
	)

	storageTempGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "openlinkhub_storage_temp",
			Help: "Current temperature of storage devices.",
		},
		[]string{"hwmonDevice", "model"},
	)

	defaultTempGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "openlinkhub_default_temp",
			Help: "Current temperature of storage devices.",
		},
		[]string{"model"},
	)
)

// Init will initialize metric headers
func Init() {
	prometheus.MustRegister(productGauge)
	prometheus.MustRegister(temperatureGauge)
	prometheus.MustRegister(rpmGauge)
	prometheus.MustRegister(storageTempGauge)
	prometheus.MustRegister(defaultTempGauge)
}

// PopulateDefault will populate default metrics like CPU, GPU...
func PopulateDefault() {
	defaultTempGauge.WithLabelValues(
		systeminfo.GetInfo().CPU.Model,
	).Set(float64(temperatures.GetCpuTemperature()))

	defaultTempGauge.WithLabelValues(
		systeminfo.GetInfo().GPU.Model,
	).Set(float64(temperatures.GetGpuTemperature()))
}

// PopulateStorage will populate storage device metrics
func PopulateStorage() {
	for _, storageTemp := range temperatures.GetStorageTemperatures() {
		storageTempGauge.WithLabelValues(
			storageTemp.Key,
			storageTemp.Model,
		).Set(float64(storageTemp.Temperature))
	}
}

// Populate will populate device metrics
func Populate(header *Header) {
	// Product info
	productGauge.WithLabelValues(
		header.Product,
		header.Serial,
		header.Firmware,
	).Set(1)

	// Temperature values
	temperatureGauge.WithLabelValues(
		header.Serial,
		header.ChannelId,
		header.Name,
		header.Description,
		header.Profile,
		header.Label,
		header.RGB,
		header.AIO,
		header.ContainsPump,
		header.TemperatureProbe,
		header.LedChannels,
	).Set(header.Temperature)

	// Speed values
	rpmGauge.WithLabelValues(
		header.Serial,
		header.ChannelId,
		header.Name,
		header.Description,
		header.Profile,
		header.Label,
		header.RGB,
		header.AIO,
		header.ContainsPump,
		header.TemperatureProbe,
		header.LedChannels,
	).Set(float64(header.Rpm))
}
