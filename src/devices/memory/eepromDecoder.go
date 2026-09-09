package memory

// Package: memory
// File: eepromDecoder.go
// Description: This file provides the basic framework and functionality to decode EEPROM data but extracts only the SKU/Part Number from the EEPROM data of DDR5 memory modules.
// Author: PabloGS
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/logger"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const (
	spd5118DriverPath  = "/sys/bus/i2c/drivers/spd5118"
	ddr5SPDBaseAddress = 0x50
	ddr5SPDLastAddress = 0x57
)

// RAMModule holds the decoded information from the EEPROM data of a RAM module.
// The struct can be extended to include more attributes as needed.
type RAMModule struct {
	// Hardware metadata
	EEPROMPath      string // Path to the SPD EEPROM
	TemperaturePath string // Path to the SPD5118 temperature input, when available
	SKU             string // SKU is the part number or identifier for the RAM module
	I2CBus          int    // I2C bus exposing the SPD5118 device
	I2CAddress      int    // SPD5118 I2C address (normally 0x50-0x57)
	Slot            int    // Logical memory slot derived from the SPD5118 address
}

// parseSKUInfo reads the byte range the SKU/Part Number is normally found at
// and filters out non-printable ASCII characters.
func parseSKUInfo(m *RAMModule, spd []byte) {
	if len(spd) >= 0x021B {
		partBytes := spd[0x0209:0x021B]
		for _, b := range partBytes {
			if b >= 32 && b <= 126 {
				m.SKU += string(b)
			}
		}
	}
}

// parseSPDModule parses the SPD data from the EEPROM file.
func parseSPDModule(path string, temperaturePath string, bus int, address int, spd []byte) RAMModule {
	m := RAMModule{
		EEPROMPath:      path,
		TemperaturePath: temperaturePath,
		I2CBus:          bus,
		I2CAddress:      address,
		Slot:            -1,
	}

	if address >= ddr5SPDBaseAddress && address <= ddr5SPDLastAddress {
		m.Slot = address - ddr5SPDBaseAddress
	}

	parseSKUInfo(&m, spd)
	return m
}

// getSPD5118TemperatureFile returns the first temperature input exposed for an
// SPD5118 device. The driver normally exposes temp1_input, but use a glob to
// avoid depending on a particular hwmon directory number.
func getSPD5118TemperatureFile(devicePath string) string {
	hwmonFolders, err := filepath.Glob(filepath.Join(devicePath, "hwmon", "hwmon*"))
	if err != nil {
		return ""
	}
	sort.Strings(hwmonFolders)

	for _, hwmonFolder := range hwmonFolders {
		temps, err := filepath.Glob(filepath.Join(hwmonFolder, "temp*_input"))
		if err != nil {
			continue
		}
		sort.Strings(temps)
		if len(temps) > 0 {
			return temps[0]
		}
	}
	return ""
}

// findEEPROMs returns SPD5118 devices belonging to the configured memory I2C
// bus. Motherboards are free to expose populated DIMMs at any addresses in the
// standard 0x50-0x57 range, so discovery must not assume a fixed starting
// address or contiguous population.
func findEEPROMs(bus int) ([]RAMModule, error) {
	entries, err := os.ReadDir(spd5118DriverPath)
	if err != nil {
		return nil, err
	}

	busPrefix := fmt.Sprintf("%d-", bus)
	var modules []RAMModule

	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), busPrefix) {
			continue
		}

		parts := strings.SplitN(entry.Name(), "-", 2)
		if len(parts) != 2 {
			continue
		}

		address, err := strconv.ParseInt(parts[1], 16, 32)
		if err != nil {
			logger.Log(logger.Fields{"device": entry.Name(), "error": err}).Warn("Unable to parse SPD5118 I2C address")
			continue
		}

		devicePath := filepath.Join(spd5118DriverPath, entry.Name())
		eepromPath := filepath.Join(devicePath, "eeprom")
		if _, err := os.Stat(eepromPath); err != nil {
			continue
		}

		data, err := os.ReadFile(eepromPath)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "path": eepromPath}).Error("Failed to read eeprom data")
			continue
		}

		module := parseSPDModule(
			eepromPath,
			getSPD5118TemperatureFile(devicePath),
			bus,
			int(address),
			data,
		)
		modules = append(modules, module)
	}

	if len(modules) == 0 {
		return nil, fmt.Errorf("no EEPROMs found on I2C bus %d", bus)
	}

	sort.Slice(modules, func(i, j int) bool {
		return modules[i].I2CAddress < modules[j].I2CAddress
	})
	return modules, nil
}

// NewMemoryModules finds and decodes DDR5 memory modules on the configured I2C
// bus. Keeping discovery scoped to the memory bus prevents unrelated SPD5118
// devices elsewhere in the system from being attached to this memory device.
func NewMemoryModules(bus int) []RAMModule {
	modules, err := findEEPROMs(bus)
	if err != nil {
		return nil
	}
	return modules
}

/*
// Print decoded SPD information to console. Not intended for production use, but useful for debugging.
func PrintModuleSPDInfo(m RAMModule) {
	fmt.Println("EEPROM Path:         ", m.EEPROMPath)
	fmt.Println("Temperature Path:    ", m.TemperaturePath)
	fmt.Println("SKU:                 ", m.SKU)
	fmt.Println("I2C Bus:             ", m.I2CBus)
	fmt.Printf("I2C Address:         0x%02x\n", m.I2CAddress)
	fmt.Println("Slot:                ", m.Slot)
}
// Iterates over all memory modules found in the system and prints their SPD information.
func PrintAllModules(modules []RAMModule) {
	for _, m := range modules {
		fmt.Println("--------------------------------------------------")
		fmt.Println("Memory Module SPD Information:")
		fmt.Println("--------------------------------------------------")
		PrintModuleSPDInfo(m)
	}
	fmt.Println("--------------------------------------------------")
}
*/
