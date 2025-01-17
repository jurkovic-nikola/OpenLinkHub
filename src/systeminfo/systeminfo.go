package systeminfo

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/dashboard"
	"OpenLinkHub/src/logger"
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type CpuData struct {
	Model   string
	Cores   int
	Threads int
}

type GpuData struct {
	Model string
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
	GPU         *GpuData
	Kernel      *KernelData
	Storage     *[]StorageData
	Motherboard *MotherboardData
}

var (
	info      *SystemInfo
	prevTotal = 0
	prevIdle  = 0
)

// Init will initialize and store system info
func Init() {
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
		if strings.Contains(line, "VGA compatible controller") && strings.Contains(line, "NVIDIA") {
			// NVIDIA
			si.GPU = &GpuData{Model: GetNVIDIAGpuModel()}
			return
		} else if strings.Contains(line, "VGA compatible controller") && strings.Contains(line, "Advanced Micro Devices") {
			// AMD Models for now just use first one
			models, err := GetAMDGpuModels()
			if err == nil && len(models) > 0 {
				si.GPU = &GpuData{Model: models[0]}
			}
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
func GetNVIDIAGpuModel() string {
	model := ""
	cmd := exec.Command("nvidia-smi", "--query-gpu=gpu_name", "--format=csv,noheader,nounits")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	model = strings.TrimSpace(string(output))
	return model
}

func GetAMDGpuModels() ([]string, error) {
	var data map[string]map[string]interface{}
	cmd := exec.Command("rocm-smi", "--showallinfo", "--json")
	jsonOutput, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("error executing rocm-smi: %v", err)
	}

	err = json.Unmarshal(jsonOutput, &data)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling JSON: %v", err)
	}

	var models []string
	for key, value := range data {
		if strings.HasPrefix(key, "card") {
			if deviceName, ok := value["Device Name"].(string); ok {
				models = append(models, deviceName)
			}
		}
	}

	return models, nil
}

// GetNVIDIAUtilization will return NVIDIA gpu utilization
func getNVIDIAUtilization() int {
	cmd := exec.Command("nvidia-smi", "--query-gpu=utilization.gpu", "--format=csv,noheader,nounits")
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

// getCpuUtilizationData will return cpu utilization data
func getCpuUtilizationData() (idle, total uint64) {
	contents, err := os.ReadFile("/proc/stat")
	if err != nil {
		return
	}
	lines := strings.Split(string(contents), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if fields[0] == "cpu" {
			numFields := len(fields)
			for i := 1; i < numFields; i++ {
				val, err := strconv.ParseUint(fields[i], 10, 64)
				if err != nil {
					logger.Log(logger.Fields{"error": err, "line": i}).Error("Unable to parse cpu stats line")
				}
				total += val // tally up all the numbers to get total ticks
				if i == 4 {  // idle is the 5th field in the cpu line
					idle = val
				}
			}
			return
		}
	}
	return
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

// getAMDUtilization fetches the GPU utilization using rocm-smi
func getAMDUtilization() (float64, error) {
	// Execute the rocm-smi command to get utilization
	cmd := exec.Command("rocm-smi", "--showuse")

	// Run the command and capture the output
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Failed to get AMD GPU utilization")
		return 0, err
	}

	// Parse the output to find GPU utilization
	output := out.String()
	utilization, err := parseAMDUtilization(output)
	if err != nil {
		return 0, err
	}

	return utilization, nil
}

func parseAMDUtilization(output string) (float64, error) {
	// Example line: "GPU[0] : 35.0%"
	// Find lines that contain the utilization information
	re := regexp.MustCompile(`GPU\[(\d+)\]\s*:\s*GPU use \(%\):\s*(\d+)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) < 3 {
		return 0, fmt.Errorf("failed to parse GPU utilization from output")
	}

	// Convert utilization to float
	utilizationStr := strings.TrimSpace(matches[2])
	utilization, err := strconv.ParseFloat(utilizationStr, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse utilization value: %v", err)
	}

	return utilization, nil
}

func GetGPUUtilization() int {
	utilization := 0
	if strings.Contains(strings.ToLower(info.GPU.Model), "nvidia") {
		// NVIDIA
		utilization = getNVIDIAUtilization()
	} else {
		// AMD
		util, err := getAMDUtilization()
		if err == nil {
			utilization = int(util)
		}
	}

	return utilization
}
