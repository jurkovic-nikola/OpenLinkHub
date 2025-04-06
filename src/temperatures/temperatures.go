package temperatures

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/dashboard"
	"OpenLinkHub/src/logger"
	"encoding/json"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

const (
	SensorTypeCPU               = 0
	SensorTypeGPU               = 1
	SensorTypeLiquidTemperature = 2
	SensorTypeStorage           = 3
	SensorTypeTemperatureProbe  = 4
	SensorTypeCpuGpu            = 5
)

type UpdateData struct {
	Fans uint16 `json:"fans"`
	Pump uint16 `json:"pump"`
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
	Sensor    uint8                `json:"sensor"`
	ZeroRpm   bool                 `json:"zeroRpm"`
	Profiles  []TemperatureProfile `json:"profiles"`
	Device    string               `json:"device"`
	ChannelId int                  `json:"channelId"`
	Linear    bool                 `json:"linear"`
	Hidden    bool
}

type StorageTemperatures struct {
	Key               string
	Model             string
	Temperature       float32
	TemperatureString string
}

var (
	temperatureOffset = 0
	pwd               = ""
	location          = ""
	profiles          = map[string]TemperatureProfileData{}
	mutex             sync.Mutex
	temperatures      *Temperatures
	cpuPackages       = []string{"k10temp", "zenpower", "coretemp"}
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

	temperatures = &Temperatures{
		Profiles: profiles,
	}

	if config.GetConfig().TemperatureOffset != 0 {
		temperatureOffset = config.GetConfig().TemperatureOffset
	}
}

// AddTemperatureProfile will save new temperature profile
func AddTemperatureProfile(profile, deviceId string, static, zeroRpm, linear bool, sensor uint8, channelId int) bool {
	mutex.Lock()
	defer mutex.Unlock()

	if _, ok := temperatures.Profiles[profile]; !ok {
		pf := TemperatureProfileData{}
		if static || linear {
			pf = profileStatic

			if linear {
				pf = profileLinearLiquid
				pf.Linear = linear
				pf.Sensor = sensor
			}
			saveProfileToDisk(profile, pf)
			return true
		}

		if sensor == 3 && len(deviceId) < 1 {
			return false
		}

		if sensor == 4 && len(deviceId) < 1 {
			return false
		}

		if sensor == 4 && channelId < 1 {
			return false
		}

		switch sensor {
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
			}
		case SensorTypeCpuGpu:
			{
				pf = profileNormal
			}
		}

		if len(deviceId) > 0 {
			pf.Device = deviceId
		}

		if sensor == 4 {
			pf.ChannelId = channelId
		}

		pf.Sensor = sensor
		pf.ZeroRpm = zeroRpm
		pf.Linear = linear
		saveProfileToDisk(profile, pf)
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
	saveProfileToDisk(profile, profiles[profile])
	return i
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

// GetTemperatureProfiles will return map of structs.TemperatureProfile
func GetTemperatureProfiles() map[string]TemperatureProfileData {
	mutex.Lock()
	defer mutex.Unlock()
	return temperatures.Profiles
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
		profiles[profileName] = profile
	}
}

// saveProfileToDisk will save profile to the disk
func saveProfileToDisk(profile string, values TemperatureProfileData) {
	profileLocation := location + profile + ".json"

	// Convert to JSON
	buffer, err := json.Marshal(values)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": location, "caller": "saveProfileToDisk()"}).Error("Unable to convert to json format")
	}

	// Create profile filename
	file, fileErr := os.Create(profileLocation)
	if fileErr != nil {
		logger.Log(logger.Fields{"error": err, "location": location, "caller": "saveProfileToDisk()"}).Error("Unable to create new filename")
	}

	// Write JSON buffer to file
	_, err = file.Write(buffer)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": profile, "caller": "saveProfileToDisk()"}).Error("Unable to write data")
		return
	}

	// Close file
	err = file.Close()
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": location, "caller": "saveProfileToDisk()"}).Error("Unable to close file handle")
	}

	LoadUserProfiles(profiles)
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
func GetNVIDIAGpuTemperature() float32 {
	cmd := exec.Command("nvidia-smi", "--query-gpu=temperature.gpu", "--format=csv,noheader,nounits")
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
	temp := GetNVIDIAGpuTemperature()
	if temp == 0 {
		temp = GetAMDGpuTemperature()
	}
	return temp
}

// getHwMonTemperature will return temperature for given entry
func getHwMonTemperature(hwmonDir string, entry os.DirEntry) float32 {
	tempFile := filepath.Join(hwmonDir, entry.Name(), "temp1_input")
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
