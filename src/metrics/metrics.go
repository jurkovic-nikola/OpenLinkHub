package metrics

// Package: metrics
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/systeminfo"
	"OpenLinkHub/src/temperatures"
	"fmt"
	"net/http"
	"strings"
	"sync"
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

type DeviceMetric struct {
	Temperature float64
	RPM         int16
}

type StorageTemp struct {
	HwmonDevice string
	Model       string
	Temperature float64
}

type DefaultTemp struct {
	Model       string
	Temperature float64
}

// PsuChannel is one telemetry channel of a PSU (the PSU itself, its fan, a temperature
// sensor, total power out or a rail). Only the Has* readings are exported.
type PsuChannel struct {
	Product     string
	Serial      string
	ChannelId   string
	Name        string
	Watts       float64
	Volts       float64
	Amps        float64
	Temperature float64
	Rpm         int16
	HasWatts    bool
	HasVolts    bool
	HasAmps     bool
	HasTemps    bool
	HasSpeed    bool
}

var (
	mu sync.RWMutex

	productMetrics = make(map[string]Header)      // key: serial
	deviceMetrics  = make(map[string]Header)      // key: serial:channel
	storageMetrics = make(map[string]StorageTemp) // key: hwmonDevice
	defaultMetrics = make(map[string]DefaultTemp) // key: model
	psuMetrics     = make(map[string]PsuChannel)  // key: serial:channel
)

// Init initializes internal maps (optional in Go, but for symmetry)
func Init() {
	productMetrics = make(map[string]Header)
	deviceMetrics = make(map[string]Header)
	storageMetrics = make(map[string]StorageTemp)
	defaultMetrics = make(map[string]DefaultTemp)
	psuMetrics = make(map[string]PsuChannel)
}

// PopulateDefault adds default temperature metrics (e.g., CPU, GPU)
func PopulateDefault() {
	cpu := systeminfo.GetInfo().CPU.Model
	mu.Lock()
	defaultMetrics[cpu] = DefaultTemp{
		Model:       cpu,
		Temperature: float64(temperatures.GetCpuTemperature()),
	}
	for key, val := range systeminfo.GetInfo().GPU {
		defaultMetrics[val.Model] = DefaultTemp{
			Model:       val.Model,
			Temperature: float64(temperatures.GetGpuTemperatureIndex(key)),
		}
	}
	mu.Unlock()
}

// PopulateStorage fills in temperature for storage devices
func PopulateStorage() {
	mu.Lock()
	for _, storage := range temperatures.GetStorageTemperatures() {
		storageMetrics[storage.Key] = StorageTemp{
			HwmonDevice: storage.Key,
			Model:       storage.Model,
			Temperature: float64(storage.Temperature),
		}
	}
	mu.Unlock()
}

// Populate fills in product and device temperature/speed info
func Populate(header *Header) {
	key := header.Serial + ":" + header.ChannelId

	mu.Lock()
	productMetrics[header.Serial] = *header
	deviceMetrics[key] = *header
	mu.Unlock()
}

// PopulatePsu fills in one PSU telemetry channel
func PopulatePsu(channel *PsuChannel) {
	mu.Lock()
	psuMetrics[channel.Serial+":"+channel.ChannelId] = *channel
	mu.Unlock()
}

// GetPsuMetrics return PSU metrics
func GetPsuMetrics() map[string]PsuChannel {
	mu.RLock()
	defer mu.RUnlock()
	cp := make(map[string]PsuChannel, len(psuMetrics))
	for k, v := range psuMetrics {
		cp[k] = v
	}
	return cp
}

// GetProductMetrics return product info
func GetProductMetrics() map[string]Header {
	mu.RLock()
	defer mu.RUnlock()
	cp := make(map[string]Header, len(productMetrics))
	for k, v := range productMetrics {
		cp[k] = v
	}
	return cp
}

// GetDeviceMetrics return device metrics
func GetDeviceMetrics() map[string]Header {
	mu.RLock()
	defer mu.RUnlock()
	cp := make(map[string]Header, len(deviceMetrics))
	for k, v := range deviceMetrics {
		cp[k] = v
	}
	return cp
}

// GetStorageMetrics return storage metrics
func GetStorageMetrics() map[string]StorageTemp {
	mu.RLock()
	defer mu.RUnlock()
	cp := make(map[string]StorageTemp, len(storageMetrics))
	for k, v := range storageMetrics {
		cp[k] = v
	}
	return cp
}

// GetDefaultMetrics return default metrics
func GetDefaultMetrics() map[string]DefaultTemp {
	mu.RLock()
	defer mu.RUnlock()
	cp := make(map[string]DefaultTemp, len(defaultMetrics))
	for k, v := range defaultMetrics {
		cp[k] = v
	}
	return cp
}

// Handler serves metrics in Prometheus exposition format.
func Handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var b strings.Builder

	// Device info
	b.WriteString("# HELP openlinkhub Product information\n")
	b.WriteString("# TYPE openlinkhub gauge\n")
	for _, p := range GetProductMetrics() {
		b.WriteString(fmt.Sprintf(`openlinkhub{product="%s",serial="%s",firmware="%s"} 1`+"\n",
			p.Product, p.Serial, p.Firmware))
	}

	// Temperature data
	b.WriteString("# HELP openlinkhub_temperature Current temperature of devices.\n")
	b.WriteString("# TYPE openlinkhub_temperature gauge\n")
	for _, d := range GetDeviceMetrics() {
		if d.Temperature > 0 {
			b.WriteString(fmt.Sprintf(`openlinkhub_temperature{serial="%s",channelId="%s",name="%s",description="%s",profile="%s",label="%s",rgb="%s",aio="%s",pump="%s",probe="%s",led="%s"} %.2f`+"\n",
				d.Serial, d.ChannelId, d.Name, d.Description, d.Profile, d.Label, d.RGB, d.AIO, d.ContainsPump, d.TemperatureProbe, d.LedChannels, d.Temperature))
		}
	}

	// RPM data
	b.WriteString("# HELP openlinkhub_speed Current speed (RPM) of devices.\n")
	b.WriteString("# TYPE openlinkhub_speed gauge\n")
	for _, d := range GetDeviceMetrics() {
		if d.Rpm > 0 {
			b.WriteString(fmt.Sprintf(`openlinkhub_speed{serial="%s",channelId="%s",name="%s",description="%s",profile="%s",label="%s",rgb="%s",aio="%s",pump="%s",probe="%s",led="%s"} %d`+"\n",
				d.Serial, d.ChannelId, d.Name, d.Description, d.Profile, d.Label, d.RGB, d.AIO, d.ContainsPump, d.TemperatureProbe, d.LedChannels, d.Rpm))
		}
	}

	// Storage temps
	b.WriteString("# HELP openlinkhub_storage_temp Current temperature of storage devices.\n")
	b.WriteString("# TYPE openlinkhub_storage_temp gauge\n")
	for _, s := range GetStorageMetrics() {
		b.WriteString(fmt.Sprintf(`openlinkhub_storage_temp{hwmonDevice="%s",model="%s"} %.2f`+"\n",
			s.HwmonDevice, s.Model, s.Temperature))
	}

	// Default temps
	b.WriteString("# HELP openlinkhub_default_temp Current temperature of default devices.\n")
	b.WriteString("# TYPE openlinkhub_default_temp gauge\n")
	for _, d := range GetDefaultMetrics() {
		b.WriteString(fmt.Sprintf(`openlinkhub_default_temp{model="%s"} %.2f`+"\n",
			d.Model, d.Temperature))
	}

	// PSU telemetry
	psu := GetPsuMetrics()
	psuLabels := func(p PsuChannel) string {
		return fmt.Sprintf(`serial="%s",product="%s",channelId="%s",name="%s"`, p.Serial, p.Product, p.ChannelId, p.Name)
	}
	b.WriteString("# HELP openlinkhub_psu_power_watts PSU power (input, total output or per rail).\n")
	b.WriteString("# TYPE openlinkhub_psu_power_watts gauge\n")
	for _, p := range psu {
		if p.HasWatts {
			b.WriteString(fmt.Sprintf("openlinkhub_psu_power_watts{%s} %.2f\n", psuLabels(p), p.Watts))
		}
	}
	b.WriteString("# HELP openlinkhub_psu_voltage_volts PSU rail voltage.\n")
	b.WriteString("# TYPE openlinkhub_psu_voltage_volts gauge\n")
	for _, p := range psu {
		if p.HasVolts && p.Volts > 0 {
			b.WriteString(fmt.Sprintf("openlinkhub_psu_voltage_volts{%s} %.2f\n", psuLabels(p), p.Volts))
		}
	}
	b.WriteString("# HELP openlinkhub_psu_current_amps PSU rail current.\n")
	b.WriteString("# TYPE openlinkhub_psu_current_amps gauge\n")
	for _, p := range psu {
		if p.HasAmps && p.Volts > 0 {
			b.WriteString(fmt.Sprintf("openlinkhub_psu_current_amps{%s} %.2f\n", psuLabels(p), p.Amps))
		}
	}
	b.WriteString("# HELP openlinkhub_psu_temperature_celsius PSU temperature sensor.\n")
	b.WriteString("# TYPE openlinkhub_psu_temperature_celsius gauge\n")
	for _, p := range psu {
		if p.HasTemps && p.Temperature > 0 {
			b.WriteString(fmt.Sprintf("openlinkhub_psu_temperature_celsius{%s} %.2f\n", psuLabels(p), p.Temperature))
		}
	}
	b.WriteString("# HELP openlinkhub_psu_fan_rpm PSU fan speed (0 while the fan is stopped).\n")
	b.WriteString("# TYPE openlinkhub_psu_fan_rpm gauge\n")
	for _, p := range psu {
		if p.HasSpeed {
			b.WriteString(fmt.Sprintf("openlinkhub_psu_fan_rpm{%s} %d\n", psuLabels(p), p.Rpm))
		}
	}

	// Send it
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	_, err := w.Write([]byte(b.String()))
	if err != nil {
		http.Error(w, "Failed to generate metrics", http.StatusInternalServerError)
		return
	}
}
