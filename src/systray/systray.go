package systray

import (
	"OpenLinkHub/src/dashboard"
	"OpenLinkHub/src/stats"
	"OpenLinkHub/src/temperatures"
)

type Systray struct {
	CpuTemp string      `json:"cpu_temp"`
	GpuTemp string      `json:"gpu_temp"`
	Battery interface{} `json:"battery"`
}

// Get will return base stats used in /api/systray call from external systray application
func Get() interface{} {
	cpuTempRaw, gpuTempRaw := temperatures.GetCpuTemperature(), temperatures.GetGpuTemperature()
	dash := dashboard.GetDashboard()
	systray := Systray{
		CpuTemp: dash.TemperatureToString(cpuTempRaw),
		GpuTemp: dash.TemperatureToString(gpuTempRaw),
		Battery: stats.GetBatteryStats(),
	}
	return systray
}
