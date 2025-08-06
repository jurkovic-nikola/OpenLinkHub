package systeminfo

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/dashboard"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/temperatures"
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type AmdGfxActivity struct {
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
}

type AmdGpuUsage struct {
	GfxActivity AmdGfxActivity `json:"gfx_activity"`
}

type AmdGPUInfo struct {
	GPU   int         `json:"gpu"`
	Usage AmdGpuUsage `json:"usage"`
}

type CpuData struct {
	Model   string
	Cores   int
	Threads int
}

type GpuData struct {
	Index             int
	Model             string
	Temperature       float32
	TemperatureString string
}

type StorageData struct {
	Model             string
	Temperature       float32
	TemperatureString string
	Key               string
}

type KernelData struct {
	OsType       string
	Architecture string
}

type MotherboardData struct {
	Model    string
	BIOS     string
	BIOSDate string
}

type SystemInfo struct {
	CPU         *CpuData
	GPU         map[int]GpuData
	Kernel      *KernelData
	Storage     *[]StorageData
	Motherboard *MotherboardData
}

type Asic struct {
	MarketName string `json:"market_name"`
	VendorID   string `json:"vendor_id"`
	VendorName string `json:"vendor_name"`
}

type AMDGPUInfo struct {
	GPU  int  `json:"gpu"`
	Asic Asic `json:"asic"`
}

var (
	info      *SystemInfo
	prevTotal = 0
	prevIdle  = 0
	gpuIndex  = 0
	amdsmi    = "amd-smi"
)

// Init will initialize and store system info
func Init() {
	gpuIndex = config.GetConfig().AMDGpuIndex
	if len(config.GetConfig().AMDSmiPath) > 0 {
		amdsmi = config.GetConfig().AMDSmiPath
	}

	info = &SystemInfo{}
	info.getCpuData()
	info.getKernelData()
	info.getGpuData()
	info.GetStorageData()
	info.GetBoardData()
}

// GetInfo will return currently stored system info
func GetInfo() *SystemInfo {
	return info
}

// getKernelData will return Kernel data
func (si *SystemInfo) getKernelData() {
	f, err := os.ReadFile("/proc/sys/kernel/ostype")
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to read kernel ostype")
		return
	}

	cmd := exec.Command("uname", "-m")
	var out bytes.Buffer
	cmd.Stdout = &out

	err = cmd.Run()
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to read kernel architecture")
		return
	}

	si.Kernel = &KernelData{
		OsType:       strings.TrimSpace(string(f)),
		Architecture: out.String(),
	}
}

// getGpuData will return GPU data
func (si *SystemInfo) getGpuData() {
	cmd := exec.Command("lspci")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Error running lspci")
		return
	}

	// Parse the output to find NVIDIA GPUs
	scanner := bufio.NewScanner(&out)
	for scanner.Scan() {
		line := scanner.Text()
		gpus := make(map[int]GpuData)
		if strings.Contains(line, "VGA compatible controller") && strings.Contains(line, "NVIDIA") && config.GetConfig().DefaultNvidiaGPU != -1 {
			// NVIDIA
			for key := range config.GetConfig().NvidiaGpuIndex {
				gpuModel := GetNVIDIAGpuModel(key)
				if len(gpuModel) > 0 {
					temp := temperatures.GetGpuTemperatureIndex(key)
					model := &GpuData{
						Index:             key,
						Model:             gpuModel,
						Temperature:       temp,
						TemperatureString: dashboard.GetDashboard().TemperatureToString(temp),
					}
					gpus[key] = *model
				}
			}
			si.GPU = gpus
			return
		} else if strings.Contains(line, "VGA compatible controller") && strings.Contains(line, "Advanced Micro Devices") {
			temp := temperatures.GetAMDGpuTemperature()
			model := &GpuData{
				Index:             0,
				Model:             GetAMDGpuModel(),
				Temperature:       temp,
				TemperatureString: dashboard.GetDashboard().TemperatureToString(temp),
			}
			gpus[0] = *model
			si.GPU = gpus
			return
		} else {
			si.GPU = nil
		}
	}

	if err = scanner.Err(); err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Error reading lspci data")
	} else {
		fmt.Println("No compatible VGA controllers found")
	}
}

// getCpuData will return CPU data
func (si *SystemInfo) getCpuData() {
	cores := 0
	threads := 0
	model := ""
	// Open cpuinfo
	f, err := os.Open("/proc/cpuinfo")
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to get CPU info")
		return
	}

	// Close it
	defer func(f *os.File) {
		err = f.Close()
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to close file handle")
		}
	}(f)

	// Scan it
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "model name") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 && len(model) == 0 {
				model = strings.TrimSpace(parts[1])
			}
		}

		if strings.Contains(line, "cpu cores") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 && cores == 0 {
				cores, err = strconv.Atoi(strings.TrimSpace(parts[1]))
				if err != nil {
					logger.Log(logger.Fields{"error": err}).Error("Unable to process CPU cores")
				}
			}
		}

		if strings.Contains(line, "processor") {
			threads++
		}
	}

	if err = scanner.Err(); err != nil {
		return
	}

	si.CPU = &CpuData{
		Model:   model,
		Cores:   cores,
		Threads: threads,
	}
}

// GetNVIDIAGpuModel will return NVIDIA gpu model
func GetNVIDIAGpuModel(index int) string {
	model := ""
	cmd := exec.Command("nvidia-smi", "-i", strconv.Itoa(index), "--query-gpu=gpu_name", "--format=csv,noheader,nounits")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	model = strings.TrimSpace(string(output))
	return model
}

func GetAMDGpuModel() string {
	cmd := exec.Command(amdsmi, "static", "-g", strconv.Itoa(gpuIndex), "--asic", "--json")
	jsonOutput, err := cmd.Output()
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to process amd-smi")
		return ""
	}

	var gpuInfo []AMDGPUInfo
	err = json.Unmarshal(jsonOutput, &gpuInfo)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to unmarshal JSON data")
		return ""
	}
	return gpuInfo[0].Asic.MarketName
}

// GetNVIDIAUtilization will return NVIDIA gpu utilization
func getNVIDIAUtilization(index int) int {
	cmd := exec.Command("nvidia-smi", "-i", strconv.Itoa(index), "--query-gpu=utilization.gpu", "--format=csv,noheader,nounits")
	output, err := cmd.Output()
	if err != nil {
		return 0
	}
	utilization := strings.TrimSpace(string(output))
	util, e := strconv.Atoi(utilization)
	if e != nil {
		logger.Log(logger.Fields{"error": e}).Error("Unable to convert GPU utilization")
		return 0
	}
	return util
}

// GetStorageData will return storage information
func (si *SystemInfo) GetStorageData() {
	hwmonDir := "/sys/class/hwmon"
	entries, err := os.ReadDir(hwmonDir)
	if err != nil {
		logger.Log(logger.Fields{"dir": hwmonDir, "error": err}).Error("Unable to read hwmon directory")
		return
	}

	var storageList []StorageData

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
				logger.Log(logger.Fields{"dir": entry.Name(), "file": tempFile, "error": e}).Error("Unable to read nvme temperature file")
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

			storage := StorageData{
				Key:               entry.Name(),
				Temperature:       temperature,
				TemperatureString: dashboard.GetDashboard().TemperatureToString(temperature),
				Model:             model,
			}
			storageList = append(storageList, storage)
		}
	}

	si.Storage = &storageList
}

// GetBoardData will return motherboard details
func (si *SystemInfo) GetBoardData() {
	board := &MotherboardData{}

	// Motherboard model
	f, err := os.ReadFile("/sys/class/dmi/id/product_name")
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to read kernel ostype")
		return
	}
	board.Model = strings.TrimSpace(string(f))

	// BIOS version
	f, err = os.ReadFile("/sys/class/dmi/id/bios_version")
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to read kernel ostype")
		return
	}
	board.BIOS = strings.TrimSpace(string(f))

	// BIOS release date
	f, err = os.ReadFile("/sys/class/dmi/id/bios_date")
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to read kernel ostype")
		return
	}
	board.BIOSDate = strings.TrimSpace(string(f))
	si.Motherboard = board
}

// GetCpuUtilization will return CPU utilization
func GetCpuUtilization() float64 {
	file, err := os.Open("/proc/stat")
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Failed to open /proc/stat")
		return 0
	}
	defer func(file *os.File) {
		err = file.Close()
		if err != nil {

		}
	}(file)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "cpu") {
			fields := strings.Fields(line)
			var (
				user    = common.Atoi(fields[1])
				nice    = common.Atoi(fields[2])
				system  = common.Atoi(fields[3])
				idle    = common.Atoi(fields[4])
				iowait  = common.Atoi(fields[5])
				irq     = common.Atoi(fields[6])
				softirq = common.Atoi(fields[7])
			)

			total := user + nice + system + idle + iowait + irq + softirq
			idleTime := idle + iowait

			totalDiff := total - prevTotal
			idleDiff := idleTime - prevIdle

			prevTotal = total
			prevIdle = idleTime

			cpuUsage := float64(totalDiff-idleDiff) / float64(totalDiff) * 100
			return cpuUsage
		}
	}
	return 0
}

// getAMDUtilization fetches the GPU utilization using amd-smi
func getAMDUtilization() float64 {
	cmd := exec.Command(amdsmi, "metric", "-g", strconv.Itoa(gpuIndex), "-u", "--json")
	jsonOutput, err := cmd.Output()
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to process amd-smi")
		return 0
	}

	var gpus []AmdGPUInfo
	err = json.Unmarshal(jsonOutput, &gpus)
	if err != nil {
		fmt.Println(err)
		logger.Log(logger.Fields{"error": err}).Error("Unable to unmarshal JSON data")
		return 0
	}

	if len(gpus) > 0 {
		return gpus[0].Usage.GfxActivity.Value
	}
	return 0
}

func GetGPUUtilization() int {
	utilization := 0
	if info.GPU != nil {
		index := config.GetConfig().DefaultNvidiaGPU
		if strings.Contains(strings.ToLower(info.GPU[index].Model), "nvidia") {
			// NVIDIA
			utilization = getNVIDIAUtilization(index)
		} else {
			// AMD
			util := getAMDUtilization()
			utilization = int(util)
		}
	}
	return utilization
}
