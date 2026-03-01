package motherboards

// Package: motherboards
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/logger"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

type Headers struct {
	Id           int            `json:"id"`
	HeaderName   string         `json:"headerName"`
	HeaderInput  string         `json:"headerInput"`
	HeaderConfig string         `json:"headerConfig"`
	HeaderLabel  string         `json:"headerLabel"`
	HeaderModes  map[int]string `json:"headerModes"`
	HeaderValue  string         `json:"headerValue"`
}
type Motherboard struct {
	Name        string          `json:"name"`
	DisplayName string          `json:"displayName"`
	Chip        string          `json:"chip"`
	Interval    float32         `json:"interval"`
	Headers     map[int]Headers `json:"headers"`
}
type Motherboards struct {
	Entry        string        `json:"entry"`
	Motherboards []Motherboard `json:"motherboards"`
}

var (
	pwd         = ""
	boardName   = ""
	hwmonPath   = ""
	boardSerial = ""
	motherboard Motherboards
	mutex       sync.Mutex
)

func Init() {
	pwd = config.GetConfig().ConfigPath

	// Read actual board name from DMI
	dmiBoardNamePath := "/sys/class/dmi/id/board_name"
	if b, err := os.ReadFile(dmiBoardNamePath); err != nil {
		logger.Log(logger.Fields{"error": err, "location": dmiBoardNamePath}).Warn("Unable to read system board name")
	} else {
		boardName = strings.TrimSpace(string(b))
		sum := md5.Sum([]byte(boardName))
		boardSerial = hex.EncodeToString(sum[:])
	}

	location := pwd + "/database/motherboard/motherboard.json"

	file, fe := os.Open(location)
	if fe != nil {
		logger.Log(logger.Fields{"error": fe, "location": location}).Warn("Unable to open motherboard file")
		return
	}

	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			//
		}
	}(file)

	reader := json.NewDecoder(file)
	if err := reader.Decode(&motherboard); err != nil {
		logger.Log(logger.Fields{"error": err, "location": location}).Warn("Unable to decode motherboard file")
		return
	}

	m := GetMotherboard()
	if m != nil {
		hwmonPath = findHwmonByChip(motherboard.Entry, m.Chip)

		for k, v := range m.Headers {
			headerLabel := GetMotherboardHeaderLabel(v.Id)
			if headerLabel != "" {
				v.HeaderName = headerLabel
				m.Headers[k] = v
			}
		}
	}
}

// GetMotherboard will return motherboard by its name
func GetMotherboard() *Motherboard {
	for i := range motherboard.Motherboards {
		if motherboard.Motherboards[i].Name == boardName {
			return &motherboard.Motherboards[i]
		}
	}
	return nil
}

// GetMotherboardSerial will return motherboard serial
func GetMotherboardSerial() string {
	return boardSerial
}

// GetMotherboardPath will return motherboard hwmon path
func GetMotherboardPath() string {
	return hwmonPath
}

// SetMotherboardHeaderMode will set motherboard header mode
func SetMotherboardHeaderMode(header, mode int) uint8 {
	mutex.Lock()
	defer mutex.Unlock()

	if mode < 1 || mode > 2 {
		logger.Log(logger.Fields{"mode": mode, "header": header}).Warn("Invalid PWM header mode")
		return 0
	}

	m := GetMotherboard()
	if m == nil {
		return 0
	}

	if val, ok := m.Headers[header]; ok {
		pwmConfig := filepath.Join(hwmonPath, strings.TrimPrefix(val.HeaderConfig, "/"))
		if common.FileExists(pwmConfig) {
			err := os.WriteFile(pwmConfig, []byte(fmt.Sprintf("%d\n", mode)), 0)
			if err != nil {
				logger.Log(logger.Fields{"mode": mode, "header": header, "error": err}).Warn("Unable to set PWM header mode")
				return 0
			}
			return 1
		}
	}
	return 0
}

// GetMotherboardHeaderMode will return motherboard header mode
func GetMotherboardHeaderMode(header int) int {
	mutex.Lock()
	defer mutex.Unlock()

	m := GetMotherboard()
	if m == nil {
		return 0
	}

	val, ok := m.Headers[header]
	if !ok {
		return 0
	}

	pwmConfig := filepath.Join(hwmonPath, strings.TrimPrefix(val.HeaderConfig, "/"))

	b, err := os.ReadFile(pwmConfig)
	if err != nil {
		logger.Log(logger.Fields{"header": header, "path": pwmConfig, "error": err}).Warn("Unable to read header config value")
		return 0
	}

	s := strings.TrimSpace(string(b))
	n, err := strconv.Atoi(s)
	if err != nil {
		logger.Log(logger.Fields{"header": header, "path": pwmConfig, "raw": s, "error": err}).Warn("Unable to parse header config value")
		return 0
	}

	if n < 0 {
		return 0
	}

	if n > 2 {
		return 2
	}

	return n
}

// GetMotherboardHeaderLabel will return motherboard header label
func GetMotherboardHeaderLabel(header int) string {
	mutex.Lock()
	defer mutex.Unlock()

	m := GetMotherboard()
	if m == nil {
		return ""
	}

	val, ok := m.Headers[header]
	if !ok {
		return ""
	}

	headerLabel := filepath.Join(hwmonPath, strings.TrimPrefix(val.HeaderLabel, "/"))

	if common.FileExists(headerLabel) {
		b, err := os.ReadFile(headerLabel)
		if err != nil {
			logger.Log(logger.Fields{"header": header, "path": headerLabel, "error": err}).Warn("Unable to read header label value")
			return ""
		}

		return strings.TrimSpace(string(b))
	}
	return ""
}

// SetMotherboardHeaderValue will set motherboard header value
func SetMotherboardHeaderValue(header, value int) uint8 {
	mutex.Lock()
	defer mutex.Unlock()

	if value < 1 || value > 100 {
		logger.Log(logger.Fields{"value": value, "header": header}).Warn("Invalid PWM header value")
		return 0
	}

	m := GetMotherboard()
	if m == nil {
		return 0
	}

	if val, ok := m.Headers[header]; ok {
		pwmValue := filepath.Join(hwmonPath, strings.TrimPrefix(val.HeaderValue, "/"))
		valToByte := percentToByte(value)
		if common.FileExists(pwmValue) {
			err := os.WriteFile(pwmValue, []byte(fmt.Sprintf("%d\n", valToByte)), 0)
			if err != nil {
				logger.Log(logger.Fields{"value": valToByte, "header": header, "error": err}).Warn("Unable to set PWM header value")
				return 0
			}
			return 1
		}
	}
	return 0
}

// GetMotherboardHeaderValue will get motherboard header value
func GetMotherboardHeaderValue(header int) int16 {
	mutex.Lock()
	defer mutex.Unlock()

	m := GetMotherboard()
	if m == nil {
		return 0
	}

	val, ok := m.Headers[header]
	if !ok {
		return 0
	}

	inputValue := filepath.Join(hwmonPath, strings.TrimPrefix(val.HeaderInput, "/"))

	b, err := os.ReadFile(inputValue)
	if err != nil {
		logger.Log(logger.Fields{"header": header, "path": inputValue, "error": err}).Warn("Unable to read header input value")
		return 0
	}

	s := strings.TrimSpace(string(b))
	n, err := strconv.Atoi(s)
	if err != nil {
		logger.Log(logger.Fields{"header": header, "path": inputValue, "raw": s, "error": err}).Warn("Unable to parse header input value")
		return 0
	}

	if n < 0 {
		return 0
	}
	if n > 32767 {
		return 32767
	}
	return int16(n)
}

// findHwmonByChip scans base with chip and returns full hwmonX path of the given chip
func findHwmonByChip(base, chip string) string {
	base = strings.TrimSpace(base)
	chip = strings.TrimSpace(chip)

	if base == "" || chip == "" {
		logger.Log(logger.Fields{"base": base, "chip": chip}).Warn("base or chip is empty")
		return ""
	}

	entries, err := os.ReadDir(base)
	if err != nil {
		logger.Log(logger.Fields{"base": base, "chip": chip, "error": err}).Warn("read hwmon base failed")
		return ""
	}

	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), "hwmon") {
			continue
		}

		namePath := filepath.Join(base, e.Name(), "name")
		b, err := os.ReadFile(namePath)
		if err != nil {
			logger.Log(logger.Fields{"base": base, "chip": chip, "error": err}).Warn("read hwmon path failed")
			continue
		}

		if strings.TrimSpace(string(b)) == chip {
			return filepath.Join(base, e.Name())
		}
	}
	return ""
}

// percentToByte will convert percent into byte value
func percentToByte(p int) uint8 {
	if p < 0 {
		p = 0
	} else if p > 100 {
		p = 100
	}
	return uint8((p*255 + 50) / 100)
}
