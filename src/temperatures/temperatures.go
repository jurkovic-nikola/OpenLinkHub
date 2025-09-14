package temperatures

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/dashboard"
	"OpenLinkHub/src/logger"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
)

const (
	SensorTypeCPU                = 0
	SensorTypeGPU                = 1
	SensorTypeLiquidTemperature  = 2
	SensorTypeStorage            = 3
	SensorTypeTemperatureProbe   = 4
	SensorTypeCpuGpu             = 5
	SensorTypeExternalHwMon      = 6
	SensorTypeExternalExecutable = 7
	SensorTypeMultiGPU           = 8
	SensorTypeGlobalTemperature  = 9
)

type UpdateData struct {
	Fans uint16 `json:"fans"`
	Pump uint16 `json:"pump"`
}

type Point struct {
	X float32 `json:"x"` // Temperature
	Y float32 `json:"y"` // Speed
}

type PointData struct {
	Sensor uint8   `json:"sensor"`
	Point  []Point `json:"points"`
}
type TemperatureProfile struct {
	Id   int     `json:"id"`
	Min  float32 `json:"min"`
	Max  float32 `json:"max"`
	Mode uint8   `json:"mode"`
	Fans uint16  `json:"fans"`
	Pump uint16  `json:"pump"`
}

type Temperatures struct {
	Profiles map[string]TemperatureProfileData `json:"profiles"`
}

type TemperatureProfileData struct {
	Sensor             uint8                `json:"sensor"`
	ZeroRpm            bool                 `json:"zeroRpm"`
	Profiles           []TemperatureProfile `json:"profiles"`
	Points             map[uint8][]Point    `json:"points"`
	Device             string               `json:"device"`
	HwmonDevice        string               `json:"hwmonDevice"`
	TemperatureInputId string               `json:"temperatureInputId"`
	ChannelId          int                  `json:"channelId"`
	Linear             bool                 `json:"linear"`
	GPUIndex           uint8                `json:"gpuIndex"`
	Hidden             bool
}

type StorageTemperatures struct {
	Key               string
	Model             string
	Temperature       float32
	TemperatureString string
}

type MemoryTemperatures struct {
	Temperature float32
}

type HwMonSensor struct {
	HwmonName  string
	SensorName string
	InputName  string
	Label      string
	TempC      float64
	Path       string
}

type NewTemperatureProfile struct {
	Profile            string
	DeviceId           string
	Static             bool
	ZeroRpm            bool
	Linear             bool
	Sensor             uint8
	ChannelId          int
	HwmonDevice        string
	TemperatureInputId string
	GpuIndex           uint8
}

var (
	i2cPrefix         = "i2c"
	temperatureOffset = 0
	pwd               = ""
	location          = ""
	profiles          = map[string]TemperatureProfileData{}
	memoryTemperature = map[int]MemoryTemperatures{}
	mutex             sync.Mutex
	temperatures      *Temperatures
	cpuPackages       = []string{"k10temp", "zenpower", "coretemp"}
	defaultTempFile   = "temp1_input"

	// Defaults
	profileQuiet = TemperatureProfileData{
		Sensor: 0,
		Profiles: []TemperatureProfile{
			{Id: 1, Min: 0, Max: 30, Mode: 0, Fans: 30, Pump: 50},
			{Id: 2, Min: 30, Max: 40, Mode: 0, Fans: 40, Pump: 50},
			{Id: 3, Min: 40, Max: 50, Mode: 0, Fans: 40, Pump: 50},
			{Id: 4, Min: 50, Max: 60, Mode: 0, Fans: 40, Pump: 50},
			{Id: 5, Min: 60, Max: 70, Mode: 0, Fans: 40, Pump: 50},
			{Id: 6, Min: 70, Max: 80, Mode: 0, Fans: 70, Pump: 80},
			{Id: 7, Min: 80, Max: 90, Mode: 0, Fans: 90, Pump: 90},
			{Id: 8, Min: 90, Max: 200, Mode: 0, Fans: 100, Pump: 100},
		},
		Hidden: false,
	}

	profileNormal = TemperatureProfileData{
		Sensor: 0,
		Profiles: []TemperatureProfile{
			{Id: 1, Min: 0, Max: 30, Mode: 0, Fans: 30, Pump: 70},
			{Id: 2, Min: 30, Max: 40, Mode: 0, Fans: 40, Pump: 70},
			{Id: 3, Min: 40, Max: 50, Mode: 0, Fans: 40, Pump: 70},
			{Id: 4, Min: 50, Max: 60, Mode: 0, Fans: 45, Pump: 70},
			{Id: 5, Min: 60, Max: 70, Mode: 0, Fans: 55, Pump: 70},
			{Id: 6, Min: 70, Max: 80, Mode: 0, Fans: 70, Pump: 80},
			{Id: 7, Min: 80, Max: 90, Mode: 0, Fans: 90, Pump: 90},
			{Id: 8, Min: 90, Max: 200, Mode: 0, Fans: 100, Pump: 100},
		},
		Hidden: false,
	}

	profilePerformance = TemperatureProfileData{
		Sensor: 0,
		Profiles: []TemperatureProfile{
			{Id: 1, Min: 0, Max: 30, Mode: 0, Fans: 70, Pump: 70},
			{Id: 2, Min: 30, Max: 40, Mode: 0, Fans: 70, Pump: 70},
			{Id: 3, Min: 40, Max: 50, Mode: 0, Fans: 70, Pump: 70},
			{Id: 4, Min: 50, Max: 60, Mode: 0, Fans: 70, Pump: 70},
			{Id: 5, Min: 60, Max: 70, Mode: 0, Fans: 70, Pump: 70},
			{Id: 6, Min: 70, Max: 80, Mode: 0, Fans: 70, Pump: 80},
			{Id: 7, Min: 80, Max: 90, Mode: 0, Fans: 90, Pump: 90},
			{Id: 8, Min: 90, Max: 200, Mode: 0, Fans: 100, Pump: 100},
		},
		Hidden: false,
	}

	// Static
	profileStatic = TemperatureProfileData{
		Sensor: 0,
		Profiles: []TemperatureProfile{
			{Id: 1, Min: 0, Max: 200, Mode: 0, Fans: 70, Pump: 70},
		},
		Hidden: false,
	}

	// Linear Liquid
	profileLinearLiquid = TemperatureProfileData{
		Sensor: 0,
		Profiles: []TemperatureProfile{
			{Id: 1, Min: 0, Max: 60, Mode: 0, Fans: 0, Pump: 0},
		},
		Hidden: false,
	}

	// AIO Liquid Temperature
	profileLiquidTemperature = TemperatureProfileData{
		Sensor: 2,
		Profiles: []TemperatureProfile{
			{Id: 1, Min: 0, Max: 30, Mode: 0, Fans: 30, Pump: 50},
			{Id: 2, Min: 30, Max: 32, Mode: 0, Fans: 30, Pump: 50},
			{Id: 3, Min: 32, Max: 34, Mode: 0, Fans: 40, Pump: 50},
			{Id: 4, Min: 34, Max: 36, Mode: 0, Fans: 40, Pump: 50},
			{Id: 5, Min: 36, Max: 38, Mode: 0, Fans: 40, Pump: 60},
			{Id: 6, Min: 38, Max: 40, Mode: 0, Fans: 40, Pump: 60},
			{Id: 7, Min: 40, Max: 42, Mode: 0, Fans: 50, Pump: 60},
			{Id: 8, Min: 42, Max: 44, Mode: 0, Fans: 60, Pump: 70},
			{Id: 9, Min: 44, Max: 46, Mode: 0, Fans: 70, Pump: 80},
			{Id: 10, Min: 46, Max: 48, Mode: 0, Fans: 80, Pump: 90},
			{Id: 11, Min: 48, Max: 50, Mode: 0, Fans: 90, Pump: 90},
			{Id: 12, Min: 50, Max: 60, Mode: 0, Fans: 100, Pump: 100}, // Critical
		},
		Hidden: false,
	}

	// Storage temperature profile
	profileStorageTemperature = TemperatureProfileData{
		Sensor: 3,
		Profiles: []TemperatureProfile{
			{Id: 1, Min: 0, Max: 30, Mode: 0, Fans: 40, Pump: 70},
			{Id: 2, Min: 30, Max: 35, Mode: 0, Fans: 40, Pump: 70},
			{Id: 3, Min: 35, Max: 40, Mode: 0, Fans: 40, Pump: 70},
			{Id: 4, Min: 40, Max: 45, Mode: 0, Fans: 50, Pump: 70},
			{Id: 5, Min: 45, Max: 50, Mode: 0, Fans: 60, Pump: 70},
			{Id: 6, Min: 50, Max: 55, Mode: 0, Fans: 70, Pump: 70},
			{Id: 7, Min: 55, Max: 60, Mode: 0, Fans: 80, Pump: 70},
			{Id: 8, Min: 60, Max: 65, Mode: 0, Fans: 90, Pump: 70},
			{Id: 9, Min: 65, Max: 70, Mode: 0, Fans: 100, Pump: 70},
		},
		Hidden: false,
	}

	// Static
	aioCriticalTemperature = TemperatureProfileData{
		Sensor: 2,
		Profiles: []TemperatureProfile{
			{Id: 1, Min: 0, Max: 65, Mode: 0, Fans: 100, Pump: 100},
		},
		Hidden: true,
	}

	// Temperature probes
	profileProbeTemperature = TemperatureProfileData{
		Sensor: 2,
		Profiles: []TemperatureProfile{
			{Id: 1, Min: 0, Max: 20, Mode: 0, Fans: 30, Pump: 50},
			{Id: 2, Min: 20, Max: 25, Mode: 0, Fans: 35, Pump: 50},
			{Id: 3, Min: 25, Max: 30, Mode: 0, Fans: 40, Pump: 50},
			{Id: 4, Min: 30, Max: 35, Mode: 0, Fans: 45, Pump: 50},
			{Id: 5, Min: 35, Max: 40, Mode: 0, Fans: 50, Pump: 60},
			{Id: 6, Min: 40, Max: 45, Mode: 0, Fans: 60, Pump: 60},
			{Id: 7, Min: 45, Max: 50, Mode: 0, Fans: 70, Pump: 70},
			{Id: 8, Min: 50, Max: 55, Mode: 0, Fans: 80, Pump: 80},
			{Id: 9, Min: 55, Max: 60, Mode: 0, Fans: 100, Pump: 100},
		},
		Hidden: false,
	}
)

// Init will initialize temperature data
func Init() {
	pwd = config.GetConfig().ConfigPath
	location = pwd + "/database/temperatures/"

	// Load any custom profile user created
	LoadUserProfiles(profiles)

	// Append default profiles
	profiles["Quiet"] = profileQuiet
	profiles["Normal"] = profileNormal
	profiles["Performance"] = profilePerformance
	profiles["aioCriticalTemperature"] = aioCriticalTemperature

	// Upgrade existing profiles to graph data
	upgradeGraphProfiles()

	// Setup
	temperatures = &Temperatures{
		Profiles: profiles,
	}

	if config.GetConfig().TemperatureOffset != 0 {
		temperatureOffset = config.GetConfig().TemperatureOffset
	}
	memoryTemperature = make(map[int]MemoryTemperatures)

	if len(config.GetConfig().CpuTempFile) > 0 {
		defaultTempFile = config.GetConfig().CpuTempFile
	}
}

// upgradeGraphProfiles will perform initial graph calculation
func upgradeGraphProfiles() {
	for name, profile := range profiles {
		if profile.Points == nil {
			pump := make([]Point, 0)
			fans := make([]Point, 0)
			data := make(map[uint8][]Point)
			if len(profile.Profiles) == 1 {
				for _, profileData := range profile.Profiles {
					point := Point{}
					// Pump
					point.X = profileData.Min
					point.Y = float32(profileData.Pump)
					pump = append(pump, point)
					point.X = profileData.Max
					pump = append(pump, point)

					// Fans
					point.X = profileData.Min
					point.Y = float32(profileData.Fans)
					fans = append(fans, point)
					point.X = profileData.Max
					fans = append(fans, point)
				}
			} else {
				for i, profileData := range profile.Profiles {
					point := Point{}
					if i == 0 {
						point.X = profileData.Min
						point.Y = float32(profileData.Pump)
					} else {
						point.X = profileData.Max
						point.Y = float32(profileData.Pump)
					}
					pump = append(pump, point)

					// Fan values
					point.Y = float32(profileData.Fans)
					fans = append(fans, point)
				}
			}

			data[0] = pump
			data[1] = fans
			profile.Points = data
			profiles[name] = profile
		}
	}
}

// SetMemoryTemperature will update memory temperature
func SetMemoryTemperature(channelId int, temperature float32) {
	memoryTemperature[channelId] = MemoryTemperatures{
		Temperature: temperature,
	}
}

// GetMemoryTemperature will return memory temperature
func GetMemoryTemperature(channelId int) float32 {
	if val, ok := memoryTemperature[channelId]; ok {
		return val.Temperature
	}
	return 0
}

// AddTemperatureProfile will save new temperature profile
func AddTemperatureProfile(newTemperatureProfile *NewTemperatureProfile) bool {
	mutex.Lock()
	defer mutex.Unlock()

	if _, ok := temperatures.Profiles[newTemperatureProfile.Profile]; !ok {
		pf := TemperatureProfileData{}
		if newTemperatureProfile.Static || newTemperatureProfile.Linear {
			pf = profileStatic
			pf.Sensor = newTemperatureProfile.Sensor
			if newTemperatureProfile.Sensor == 2 {
				pf.Profiles[0].Max = 60
			}
			if newTemperatureProfile.Linear {
				pf = profileLinearLiquid
				pf.Linear = newTemperatureProfile.Linear
			}
			if len(newTemperatureProfile.DeviceId) > 0 {
				pf.Device = newTemperatureProfile.DeviceId
			}
			if newTemperatureProfile.Sensor == 4 || newTemperatureProfile.Sensor == 9 {
				pf.ChannelId = newTemperatureProfile.ChannelId
			}
			if pf.Points == nil {
				pump := make([]Point, 0)
				fans := make([]Point, 0)
				data := make(map[uint8][]Point)
				for _, profileData := range pf.Profiles {
					point := Point{}
					// Pump
					point.X = profileData.Min
					point.Y = float32(profileData.Pump)
					pump = append(pump, point)
					point.X = profileData.Max
					pump = append(pump, point)

					// Fans
					point.X = profileData.Min
					point.Y = float32(profileData.Fans)
					fans = append(fans, point)
					point.X = profileData.Max
					fans = append(fans, point)
				}

				data[0] = pump
				data[1] = fans
				pf.Points = data
			}

			err := saveProfileToDisk(newTemperatureProfile.Profile, pf)
			if err != nil {
				return false
			}
			return true
		}

		if newTemperatureProfile.Sensor == 3 && len(newTemperatureProfile.DeviceId) < 1 {
			return false
		}

		if newTemperatureProfile.Sensor == 4 && len(newTemperatureProfile.DeviceId) < 1 {
			return false
		}

		if newTemperatureProfile.Sensor == 4 && newTemperatureProfile.ChannelId < 1 {
			return false
		}

		switch newTemperatureProfile.Sensor {
		case SensorTypeCPU:
			{
				pf = profileNormal
			}
		case SensorTypeGPU:
			{
				pf = profileNormal
			}
		case SensorTypeLiquidTemperature:
			{
				pf = profileLiquidTemperature
			}
		case SensorTypeStorage:
			{
				pf = profileStorageTemperature
			}
		case SensorTypeTemperatureProbe:
			{
				pf = profileProbeTemperature
				if strings.HasPrefix(newTemperatureProfile.DeviceId, i2cPrefix) {
					pf = profileNormal
				}
			}
		case SensorTypeCpuGpu:
			{
				pf = profileNormal
			}
		case SensorTypeExternalHwMon:
			{
				pf = profileNormal
			}
		case SensorTypeExternalExecutable:
			{
				pf = profileNormal
			}
		case SensorTypeMultiGPU:
			{
				pf = profileNormal
			}
		case SensorTypeGlobalTemperature:
			{
				pf = profileNormal
			}
		}

		if len(newTemperatureProfile.DeviceId) > 0 {
			pf.Device = newTemperatureProfile.DeviceId
		}

		if newTemperatureProfile.Sensor == 4 || newTemperatureProfile.Sensor == 9 {
			pf.ChannelId = newTemperatureProfile.ChannelId
		}

		if len(newTemperatureProfile.HwmonDevice) > 0 {
			pf.HwmonDevice = newTemperatureProfile.HwmonDevice
			pf.TemperatureInputId = newTemperatureProfile.TemperatureInputId
		}

		pf.Sensor = newTemperatureProfile.Sensor
		pf.ZeroRpm = newTemperatureProfile.ZeroRpm
		pf.Linear = newTemperatureProfile.Linear
		pf.GPUIndex = newTemperatureProfile.GpuIndex

		if pf.Points == nil {
			pump := make([]Point, 0)
			fans := make([]Point, 0)
			data := make(map[uint8][]Point)
			for i, profileData := range pf.Profiles {
				point := Point{}
				if i == 0 {
					point.X = profileData.Min
					point.Y = float32(profileData.Pump)
				} else {
					point.X = profileData.Max
					point.Y = float32(profileData.Pump)
				}
				pump = append(pump, point)

				// Fan values
				point.Y = float32(profileData.Fans)
				fans = append(fans, point)
			}

			data[0] = pump
			data[1] = fans
			pf.Points = data
		}

		err := saveProfileToDisk(newTemperatureProfile.Profile, pf)
		if err != nil {
			return false
		}
		return true
	} else {
		return false
	}
}

// UpdateTemperatureProfile will update temperature profile with given JSON string
func UpdateTemperatureProfile(profile string, values string) int {
	i := 0
	var payload map[int]UpdateData
	err := json.Unmarshal([]byte(values), &payload)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "caller": "UpdateTemperatureProfile()"}).Error("Unable to read content of a string to JSON")
		return 0
	}

	// Get profile
	profileList := GetTemperatureProfile(profile)
	if profileList == nil {
		logger.Log(logger.Fields{"error": err, "caller": "UpdateTemperatureProfile()"}).Warn("Non-existing profile")
		return 0
	}

	// Loop thru profile values
	for key := range profileList.Profiles {
		// Extract payload data by given key
		if payloadValue, ok := payload[profileList.Profiles[key].Id]; ok {
			if payloadValue.Pump < 20 {
				payloadValue.Pump = 50
			}
			// Payload contains our key, update values
			if profiles[profile].Profiles[key].Pump != payloadValue.Pump || profiles[profile].Profiles[key].Fans != payloadValue.Fans {
				// Update if original is different from new
				profiles[profile].Profiles[key].Pump = payloadValue.Pump
				profiles[profile].Profiles[key].Fans = payloadValue.Fans
				i++
			}
		}
	}

	// Persistent save
	err = saveProfileToDisk(profile, profiles[profile])
	if err != nil {
		logger.Log(logger.Fields{"error": err, "caller": "UpdateTemperatureProfile()"}).Error("Unable to save profile to disk")
		return 0
	}
	return i
}

// UpdateTemperatureProfileGraph will update temperature profile with given JSON string
func UpdateTemperatureProfileGraph(profile string, value TemperatureProfileData) uint8 {
	err := saveProfileToDisk(profile, value)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "caller": "UpdateTemperatureProfile()"}).Error("Unable to save profile to disk")
		return 0
	}
	return 1
}

// DeleteTemperatureProfile will delete temperature profile
func DeleteTemperatureProfile(profile string) {
	mutex.Lock()
	defer mutex.Unlock()

	if _, ok := temperatures.Profiles[profile]; ok {
		profileLocation := location + profile + ".json"
		err := os.Remove(profileLocation)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "location": profileLocation, "caller": "DeleteTemperatureProfile()"}).Warn("Unable to delete speed profile")
		} else {
			delete(temperatures.Profiles, profile)
		}
	}
}

// GetTemperatureProfile will return structs.TemperatureProfile for given profile name
func GetTemperatureProfile(profile string) *TemperatureProfileData {
	mutex.Lock()
	defer mutex.Unlock()

	if value, ok := temperatures.Profiles[profile]; ok {
		return &value
	}
	return nil
}

// GetTemperatureGraph will return temperature graph in X,Y slice for given profile name
func GetTemperatureGraph(profile string) map[int]PointData {
	mutex.Lock()
	defer mutex.Unlock()

	result := make(map[int]PointData, 2)

	if value, ok := temperatures.Profiles[profile]; ok {
		result[0] = PointData{Sensor: value.Sensor, Point: value.Points[0]} // 0 - Pump
		result[1] = PointData{Sensor: value.Sensor, Point: value.Points[1]} // 1 - Fans
	}
	return result
}

// GetTemperatureProfiles will return map of structs.TemperatureProfile
func GetTemperatureProfiles() map[string]TemperatureProfileData {
	mutex.Lock()
	defer mutex.Unlock()
	return temperatures.Profiles
}

// getHwMonDirectoryByDeviceName return hwmon path for given device
func getHwMonDirectoryByDeviceName(deviceName string) string {
	basePath := "/sys/class/hwmon/"
	hwmonEntries, err := os.ReadDir(basePath)
	if err != nil {
		fmt.Printf("Error reading hwmon dir: %v\n", err)
		return ""
	}

	for _, entry := range hwmonEntries {
		hwmonDirName := entry.Name()
		hwmonPath := filepath.Join(basePath, hwmonDirName)

		// Read sensor chip name
		nameBytes, err := os.ReadFile(filepath.Join(hwmonPath, "name"))
		if err != nil {
			continue
		}
		sensorName := strings.TrimSpace(string(nameBytes))
		if sensorName == deviceName {
			return hwmonPath
		}
	}
	return ""
}

// LoadUserProfiles will load all user profiles
func LoadUserProfiles(profiles map[string]TemperatureProfileData) {
	files, err := os.ReadDir(location)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": location, "caller": "LoadUserProfiles()"}).Fatal("Unable to read content of a folder")
	}

	for _, fi := range files {
		if fi.IsDir() {
			continue // Exclude folders if any
		}

		// Define a full path of filename
		profileLocation := location + fi.Name()

		// Check if filename has .json extension
		if !common.IsValidExtension(profileLocation, ".json") {
			continue
		}

		profileName := strings.Split(fi.Name(), ".")[0]
		file, fe := os.Open(profileLocation)
		if fe != nil {
			logger.Log(logger.Fields{"error": fe, "location": profileLocation, "caller": "LoadUserProfiles()"}).Fatal("Unable to read temperature profile")
		}

		// Decode and create profile
		var profile TemperatureProfileData
		reader := json.NewDecoder(file)
		if fe = reader.Decode(&profile); fe != nil {
			logger.Log(logger.Fields{"error": fe, "location": profileLocation, "caller": "LoadUserProfiles()"}).Fatal("Unable to read temperature profile")
		}

		// Recalculate hwmon dynamic path
		if len(profile.HwmonDevice) > 0 && len(profile.TemperatureInputId) > 0 {
			path := getHwMonDirectoryByDeviceName(profile.HwmonDevice)
			deviceId := fmt.Sprintf("%s/%s", path, profile.TemperatureInputId)
			profile.Device = deviceId
		}
		profiles[profileName] = profile
	}
}

// saveProfileToDisk will save profile to the disk
func saveProfileToDisk(profile string, values TemperatureProfileData) error {
	profileLocation := location + profile + ".json"

	// Convert to JSON
	buffer, err := json.MarshalIndent(values, "", "    ")
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": location, "caller": "saveProfileToDisk()"}).Error("Unable to convert to json format")
		return err
	}

	// Create profile filename
	file, err := os.Create(profileLocation)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": location, "caller": "saveProfileToDisk()"}).Error("Unable to create new filename")
	}

	// Write JSON buffer to file
	_, err = file.Write(buffer)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": profile, "caller": "saveProfileToDisk()"}).Error("Unable to write data")
		return err
	}

	// Close file
	err = file.Close()
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": location, "caller": "saveProfileToDisk()"}).Error("Unable to close file handle")
	}

	LoadUserProfiles(profiles)
	return nil
}

// friendlyTempLabel will attempt to make friendly name of the sensor
func friendlyTempLabel(file string) string {
	num := file[4:strings.Index(file, "_")]
	return "Sensor " + num
}

// GetAMDGpuTemperature will return AMD GPU temperature
func GetAMDGpuTemperature() float32 {
	hwmonDir := "/sys/class/hwmon"
	entries, err := os.ReadDir(hwmonDir)
	if err != nil {
		return 0
	}

	for _, entry := range entries {
		nameFile := filepath.Join(hwmonDir, entry.Name(), "name")
		name, err := os.ReadFile(nameFile)
		if err != nil {
			continue
		}

		if strings.TrimSpace(string(name)) == "amdgpu" {
			tempFile := filepath.Join(hwmonDir, entry.Name(), "temp1_input")
			temp, err := os.ReadFile(tempFile)
			if err != nil {
				continue
			}

			tempStr := strings.TrimSpace(string(temp))
			tempValue, err := strconv.Atoi(tempStr)
			if err != nil {
				continue
			}
			tempCelsius := float32(tempValue) / 1000.0
			return tempCelsius
		}
	}
	return 0
}

// GetNVIDIAGpuTemperature will return NVIDIA gpu temperature
func GetNVIDIAGpuTemperature(gpuIndex int) float32 {
	if config.GetConfig().DefaultNvidiaGPU == -1 {
		return 0
	}

	cmd := exec.Command("nvidia-smi", "-i", strconv.Itoa(gpuIndex), "--query-gpu=temperature.gpu", "--format=csv,noheader,nounits")
	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	// Convert output to string and trim spaces
	tempStr := strings.TrimSpace(string(output))

	// Convert temperature to an integer
	temp, err := strconv.Atoi(tempStr)
	if err != nil {
		return 0
	}

	return float32(temp)
}

// GetGpuTemperature will return GPU temperature
func GetGpuTemperature() float32 {
	temp := GetNVIDIAGpuTemperature(config.GetConfig().DefaultNvidiaGPU)
	if temp == 0 {
		temp = GetAMDGpuTemperature()
	}
	return temp
}

// GetGpuTemperatureIndex will return GPU temperature via device index
func GetGpuTemperatureIndex(index int) float32 {
	temp := GetNVIDIAGpuTemperature(index)
	if temp == 0 {
		temp = GetAMDGpuTemperature()
	}
	return temp
}

// getHwMonTemperature will return temperature for given entry
func getHwMonTemperature(hwmonDir string, entry os.DirEntry) float32 {
	tempFile := filepath.Join(hwmonDir, entry.Name(), defaultTempFile)
	temp, err := os.ReadFile(tempFile)
	if err != nil {
		return 0
	}

	tempStr := strings.TrimSpace(string(temp))
	tempValue, err := strconv.Atoi(tempStr)
	if err != nil {
		return 0
	}
	tempCelsius := float32(tempValue) / 1000.0
	temperature := float32(math.Floor(float64(tempCelsius*100)) / 100)
	if temperatureOffset != 0 {
		temperature = temperature + float32(temperatureOffset)
	}
	return temperature
}

// GetCpuTemperature will return CPU temperature
func GetCpuTemperature() float32 {
	hwmonDir := "/sys/class/hwmon"
	entries, err := os.ReadDir(hwmonDir)
	if err != nil {
		logger.Log(logger.Fields{"dir": hwmonDir, "error": err, "caller": "GetCpuTemperature()"}).Error("Unable to read hwmon directory")
		return 0
	}

	for _, entry := range entries {
		nameFile := filepath.Join(hwmonDir, entry.Name(), "name")
		name, e := os.ReadFile(nameFile)
		if e != nil {
			continue
		}
		cpuPackage := strings.TrimSpace(string(name))

		// Manual package definition
		if len(config.GetConfig().CPUSensorChip) > 0 {
			if cpuPackage == config.GetConfig().CPUSensorChip {
				temp := getHwMonTemperature(hwmonDir, entry)
				if temp == 0 {
					continue
				}
				return temp
			}
		} else {
			// Automatic package detection
			for _, val := range cpuPackages {
				if val == cpuPackage {
					temp := getHwMonTemperature(hwmonDir, entry)
					if temp == 0 {
						continue
					}
					return temp
				}
			}
		}
	}
	return 0
}

// GetStorageTemperatures will return storage temperatures
func GetStorageTemperatures() []StorageTemperatures {
	hwmonDir := "/sys/class/hwmon"
	entries, err := os.ReadDir(hwmonDir)
	if err != nil {
		logger.Log(logger.Fields{"dir": hwmonDir, "error": err}).Error("Unable to read hwmon directory")
		return nil
	}

	var storageList []StorageTemperatures

	for _, entry := range entries {
		nameFile := filepath.Join(hwmonDir, entry.Name(), "name")
		nameBytes, e := os.ReadFile(nameFile)
		if e != nil {
			continue
		}

		name := strings.TrimSpace(string(nameBytes))

		if string(name) == "nvme" || string(name) == "drivetemp" {
			var temperature float32 = 0.0
			tempFile := filepath.Join(hwmonDir, entry.Name(), "temp1_input")
			temp, e := os.ReadFile(tempFile)
			if e != nil {
				logger.Log(logger.Fields{"dir": entry.Name(), "file": tempFile, "error": e}).Error("Unable to read hwmon file")
				continue
			}

			// Convert the temperature from milli-Celsius to Celsius
			tempMilliC, e := strconv.Atoi(strings.TrimSpace(string(temp)))
			if e != nil {
				logger.Log(logger.Fields{"dir": entry.Name(), "file": tempFile, "error": e}).Error("Unable to read storage temperature file")
				continue
			}
			temperature = float32(tempMilliC / 1000)

			modelFile := filepath.Join(hwmonDir, entry.Name(), "device/model")
			deviceModel, e := os.ReadFile(modelFile)
			if e != nil {
				logger.Log(logger.Fields{"dir": entry.Name(), "file": tempFile, "error": e}).Error("Unable to read hwmon file")
				continue
			}

			model := strings.TrimSpace(string(deviceModel))

			storage := StorageTemperatures{
				Key:               entry.Name(),
				Temperature:       temperature,
				TemperatureString: dashboard.GetDashboard().TemperatureToString(temperature),
				Model:             model,
			}
			storageList = append(storageList, storage)
		}
	}
	return storageList
}

// GetStorageTemperature will return storage temperature for specified hwmon sensor
func GetStorageTemperature(hwmonDeviceId string) float32 {
	hwmonDir := "/sys/class/hwmon"
	tempFile := filepath.Join(hwmonDir, hwmonDeviceId, "temp1_input")
	temp, e := os.ReadFile(tempFile)
	if e != nil {
		return 0
	}

	tempMilliC, e := strconv.Atoi(strings.TrimSpace(string(temp)))
	if e != nil {
		logger.Log(logger.Fields{"deviceId": hwmonDeviceId, "file": tempFile, "error": e}).Error("Unable to read storage temperature file")
		return 0
	}
	return float32(tempMilliC / 1000)
}

// GetHwMonTemperature will return hwmon temperature for specified hwmon sensor
func GetHwMonTemperature(hwmonDevice string) float32 {
	temp, e := os.ReadFile(hwmonDevice)
	if e != nil {
		return 0
	}

	tempMilliC, e := strconv.Atoi(strings.TrimSpace(string(temp)))
	if e != nil {
		logger.Log(logger.Fields{"file": hwmonDevice, "error": e}).Error("Unable to read hwmon temperature file")
		return 0
	}
	return float32(math.Round(float64(tempMilliC)/10.0) / 100.0)
}

// GetExternalBinaryTemperature will return temperature of external binary
func GetExternalBinaryTemperature(filePath string) float32 {
	cmd := exec.Command(filePath)
	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	temp := strings.TrimSpace(string(output))
	tempFloat, err := strconv.ParseFloat(temp, 64)
	if err != nil {
		logger.Log(logger.Fields{"file": filePath, "error": err}).Error("Unable to parse binary temperature output")
		return 0
	}

	return float32(math.Round(tempFloat*100) / 100)
}

// Interpolate will perform linear interpolation
func Interpolate(points []Point, inputTemp float32) float32 {
	if len(points) == 0 {
		return 0
	}

	// Sort points by temperature
	sort.Slice(points, func(i, j int) bool {
		return points[i].X < points[j].X
	})

	// Clamp below first point
	if inputTemp <= points[0].X {
		return points[0].Y
	}

	// Clamp above last point
	if inputTemp >= points[len(points)-1].X {
		return points[len(points)-1].Y
	}

	// Linear interpolation between two points
	for i := 0; i < len(points)-1; i++ {
		a := points[i]
		b := points[i+1]
		if inputTemp >= a.X && inputTemp <= b.X {
			ratio := (inputTemp - a.X) / (b.X - a.X)
			return a.Y + ratio*(b.Y-a.Y)
		}
	}
	return 0
}

// GetExternalHwMonSensors will parse and return all external hwmon sensors available in the system.
func GetExternalHwMonSensors() interface{} {
	basePath := "/sys/class/hwmon/"
	hwmonEntries, err := os.ReadDir(basePath)
	if err != nil {
		fmt.Printf("Error reading hwmon dir: %v\n", err)
		return nil
	}

	var sensors []HwMonSensor

	for _, entry := range hwmonEntries {
		hwmonDirName := entry.Name()
		hwmonPath := filepath.Join(basePath, hwmonDirName)

		info, err := os.Stat(hwmonPath)
		if err != nil || !info.IsDir() {
			continue
		}

		// Read sensor chip name
		nameBytes, err := os.ReadFile(filepath.Join(hwmonPath, "name"))
		if err != nil {
			continue
		}
		sensorName := strings.TrimSpace(string(nameBytes))

		files, err := os.ReadDir(hwmonPath)
		if err != nil {
			continue
		}

		for _, file := range files {
			fileName := file.Name()

			if strings.HasPrefix(fileName, "temp") && strings.HasSuffix(fileName, "_input") {
				fullPath := filepath.Join(hwmonPath, fileName)

				dataBytes, err := os.ReadFile(fullPath)
				if err != nil {
					continue
				}
				rawStr := strings.TrimSpace(string(dataBytes))
				rawMilliC, err := strconv.Atoi(rawStr)
				if err != nil {
					continue
				}
				tempC := float64(rawMilliC) / 1000.0

				// Try to read label
				labelFile := strings.Replace(fileName, "_input", "_label", 1)
				labelPath := filepath.Join(hwmonPath, labelFile)
				label := ""
				if labelBytes, err := os.ReadFile(labelPath); err == nil {
					label = strings.TrimSpace(string(labelBytes))
				} else {
					label = friendlyTempLabel(fileName)
				}

				sensors = append(sensors,
					HwMonSensor{
						HwmonName:  hwmonDirName,
						SensorName: sensorName,
						InputName:  fileName,
						Label:      label,
						TempC:      tempC,
						Path:       fullPath,
					},
				)
			}
		}
	}
	return sensors
}
