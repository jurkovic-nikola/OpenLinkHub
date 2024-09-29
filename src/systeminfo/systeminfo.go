package systeminfo

import (
	"OpenLinkHub/src/logger"
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
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
	Model       string
	Temperature float32
	Key         string
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

var info *SystemInfo

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
			// AMD
			// To-Do: Find proper AMD GPU model
			si.GPU = &GpuData{Model: "AMD Compatible GPU"}
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
				Key:         entry.Name(),
				Temperature: temperature,
				Model:       model,
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
	idle0, total0 := getCpuUtilizationData()
	time.Sleep(100 * time.Millisecond)
	idle1, total1 := getCpuUtilizationData()

	idleTicks := float64(idle1 - idle0)
	totalTicks := float64(total1 - total0)
	cpuUsage := 100 * (totalTicks - idleTicks) / totalTicks
	return cpuUsage
}

// getAMDUtilization fetches the GPU utilization using rocm-smi
func getAMDUtilization() (float64, error) {
	// Execute the rocm-smi command to get utilization
	cmd := exec.Command("rocm-smi", "--show-utilization")

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

// parseUtilization parses the rocm-smi output to find GPU utilization
func parseAMDUtilization(output string) (float64, error) {
	// Example line: "GPU[0] : 35.0%"
	// Find lines that contain the utilization information
	re := regexp.MustCompile(`GPU\[\d+\]\s*:\s*(\d+\.\d+)%`)
	matches := re.FindStringSubmatch(output)
	if len(matches) < 2 {
		return 0, fmt.Errorf("failed to parse GPU utilization from output")
	}

	// Convert utilization to float
	utilizationStr := strings.TrimSpace(matches[1])
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
