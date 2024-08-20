package metrics

import (
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
)

// Init will initialize metric headers
func Init() {
	prometheus.MustRegister(productGauge)
	prometheus.MustRegister(temperatureGauge)
	prometheus.MustRegister(rpmGauge)
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
