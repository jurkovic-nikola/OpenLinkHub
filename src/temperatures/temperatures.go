package temperatures

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/logger"
	"encoding/json"
	"fmt"
	"github.com/NVIDIA/go-nvml/pkg/nvml"
	"github.com/ssimunic/gosensors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

const (
	SensorTypeCPU = 0
	SensorTypeGPU = 1
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
	Sensor   uint8                `json:"sensor"`
	ZeroRpm  bool                 `json:"zeroRpm"`
	Profiles []TemperatureProfile `json:"profiles"`
}

var (
	pwd, _       = os.Getwd()
	location     = pwd + "/database/temperatures/"
	profiles     = map[string]TemperatureProfileData{}
	mutex        sync.Mutex
	temperatures *Temperatures

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
	}

	profileStatic = TemperatureProfileData{
		Sensor: 0,
		Profiles: []TemperatureProfile{
			{Id: 1, Min: 0, Max: 200, Mode: 0, Fans: 70, Pump: 70},
		},
	}
)

// Init will initialize temperature data
func Init() {
	// Load any custom profile user created
	LoadUserProfiles(profiles)

	// Append default profiles
	profiles["Quiet"] = profileQuiet
	profiles["Normal"] = profileNormal
	profiles["Performance"] = profilePerformance

	temperatures = &Temperatures{
		Profiles: profiles,
	}
}

// AddTemperatureProfile will save new temperature profile
func AddTemperatureProfile(profile string, static, zeroRpm bool, sensor uint8) bool {
	mutex.Lock()
	defer mutex.Unlock()
	pf := profileNormal

	if _, ok := temperatures.Profiles[profile]; !ok {
		if static {
			pf = profileStatic
			saveProfileToDisk(profile, pf)
		}
		pf.Sensor = sensor
		pf.ZeroRpm = zeroRpm
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
		logger.Log(logger.Fields{"error": err}).Error("Unable to read content of a string to JSON")
		return 0
	}

	// Get profile
	profileList := GetTemperatureProfile(profile)
	if profileList == nil {
		logger.Log(logger.Fields{"error": err}).Warn("Non-existing profile")
		return 0
	}

	// Loop thru profile values
	for key := range profileList.Profiles {
		// Extract payload data by given key
		if payloadValue, ok := payload[profileList.Profiles[key].Id]; ok {
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
			logger.Log(logger.Fields{"error": err, "location": profileLocation}).Warn("Unable to delete speed profile")
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
	mutex.Lock()
	defer mutex.Unlock()
	files, err := os.ReadDir(location)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": location}).Fatal("Unable to read content of a folder")
	}

	for _, fi := range files {
		if fi.IsDir() {
			continue // Exclude folders if any
		}

		// Define a full path of filename
		profileLocation := location + fi.Name()

		if !common.IsValidExtension(profileLocation, ".json") {
			continue
		}

		profileName := strings.Split(fi.Name(), ".")[0]

		fmt.Println("[Temperatures] Loading profile:", profileLocation)
		file, err := os.Open(profileLocation)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "location": profileLocation}).Fatal("Unable to read temperature profile")
		}

		// Decode and create profile
		var profile TemperatureProfileData
		reader := json.NewDecoder(file)
		if err = reader.Decode(&profile); err != nil {
			logger.Log(logger.Fields{"error": err, "location": profileLocation}).Fatal("Unable to read temperature profile")
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
		logger.Log(logger.Fields{"error": err, "location": location}).Fatal("Unable to convert to json format")
	}

	// Create profile filename
	file, fileErr := os.Create(profileLocation)
	if fileErr != nil {
		logger.Log(logger.Fields{"error": err, "location": location}).Fatal("Unable to create new filename")
	}

	// Write JSON buffer to file
	_, err = file.Write(buffer)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": profile}).Fatal("Unable to write data")
		return
	}

	// Close file
	err = file.Close()
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": location}).Fatal("Unable to close file handle")
	}

	// Add profile to the list
	temperatures.Profiles[profile] = values
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
	ret := nvml.Init()
	if ret != nvml.SUCCESS {
		logger.Log(logger.Fields{"err": nvml.ErrorString(ret)}).Warn("Unable to initialize new nvml")
		return 0
	}
	defer func() {
		ret := nvml.Shutdown()
		if ret != nvml.SUCCESS {
			return
		}
	}()

	count, ret := nvml.DeviceGetCount()
	if ret != nvml.SUCCESS {
		logger.Log(logger.Fields{"err": nvml.ErrorString(ret)}).Warn("Unable to get device count")
		return 0
	}

	for i := 0; i < count; i++ {
		device, ret := nvml.DeviceGetHandleByIndex(i)
		if ret != nvml.SUCCESS {
			logger.Log(logger.Fields{"index": i, "device": nvml.ErrorString(ret)}).Warn("Unable to get device")
			return 0
		}

		ts := nvml.TemperatureSensors(0)
		temperature, err := device.GetTemperature(ts)
		if ret != nvml.SUCCESS {
			logger.Log(logger.Fields{"err": err}).Warn("Unable to get device temperature")
			continue
		} else {
			return float32(temperature)
		}
	}
	return 0
}

// GetCpuTemperature will return CPU temperature
func GetCpuTemperature() float32 {
	sensors, err := gosensors.NewFromSystem()
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Warn("Unable to find sensors. You are probably missing lm-sensors package!")
		return 0
	}

	for chip := range sensors.Chips {
		if chip == config.GetConfig().CPUSensorChip {
			if val, ok := sensors.Chips[chip][config.GetConfig().CPUPackageIdent]; ok {
				val := val[1 : len(val)-3]
				value, err := strconv.ParseFloat(val, 32)
				if err != nil {
					return 0
				}
				return float32(value)
			}
		}
	}
	return 0
}
