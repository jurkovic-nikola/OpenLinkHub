package systeminfo

import (
	"OpenLinkHub/src/logger"
	"bufio"
	"bytes"
	"fmt"
	"github.com/NVIDIA/go-nvml/pkg/nvml"
	"os"
	"os/exec"
	"path/filepath"
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
	Model       string
	Temperature float32
	Key         string
}

type KernelData struct {
	OsType       string
	Architecture string
}

type SystemInfo struct {
	CPU     *CpuData
	GPU     *GpuData
	Kernel  *KernelData
	Storage *[]StorageData
}

var info *SystemInfo

// Init will initialize and store system info
func Init() {
	info = &SystemInfo{}
	info.getCpuData()
	info.getKernelData()
	info.getGpuData()
	info.GetStorageData()
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
	ret := nvml.Init()
	if ret != nvml.SUCCESS {
		logger.Log(logger.Fields{"err": nvml.ErrorString(ret)}).Warn("Unable to initialize new nvml")
		return ""
	}
	defer func() {
		ret = nvml.Shutdown()
		if ret != nvml.SUCCESS {
			return
		}
	}()

	count, ret := nvml.DeviceGetCount()
	if ret != nvml.SUCCESS {
		logger.Log(logger.Fields{"err": nvml.ErrorString(ret)}).Warn("Unable to get device count")
		return ""
	}

	for i := 0; i < count; i++ {
		device, ret := nvml.DeviceGetHandleByIndex(i)
		if ret != nvml.SUCCESS {
			logger.Log(logger.Fields{"index": i, "error": nvml.ErrorString(ret)}).Warn("Unable to get device")
			return ""
		}

		model, ret = device.GetName()
		if ret != nvml.SUCCESS {
			logger.Log(logger.Fields{"err": nvml.ErrorString(ret)}).Warn("Unable to get GPU model")
			continue
		} else {
			return model
		}
	}
	return ""
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
