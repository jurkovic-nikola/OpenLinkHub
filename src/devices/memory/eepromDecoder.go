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
	"strconv"
	"strings"
)

const (
	hwmonRoot = "/sys/class/hwmon"
)

// RAMModule holds the decoded information from the EEPROM data of a RAM module.
// The struct can be extended to include more attributes as needed.
type RAMModule struct {
	// Hardware metadata
	EEPROMPath   string // Path to the EEPROM within hwmon device directory
	SKU          string // SKU is the part number or identifier for the RAM module
	I2CAddress   uint8  // I2C address of the SPD hub (e.g., 0x51, 0x53)
	ColorIndex   int    // Index into colorAddresses array (SPD address - 0x50)
}

// parseSKUInfo Reads the byte range the SKU/Part Number is normally found at and filters out non-printable ASCII characters
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

// extractI2CAddress extracts the I2C address from the EEPROM path
// Path format: /sys/class/hwmon/hwmonX/device/eeprom
// Device symlink points to: ../../i2c-1/1-0051 (where 0051 is the hex address)
func extractI2CAddress(eepromPath string) (uint8, int) {
	// Resolve the device symlink to get the real path
	devicePath := filepath.Dir(eepromPath)
	realPath, err := filepath.EvalSymlinks(devicePath)
	if err != nil {
		return 0, -1
	}

	// Extract the I2C address from the path (e.g., "1-0051" -> 0x51)
	base := filepath.Base(realPath)
	parts := strings.Split(base, "-")
	if len(parts) != 2 {
		return 0, -1
	}

	// Parse the hex address (e.g., "0051" -> 0x51)
	addr, err := strconv.ParseUint(parts[1], 16, 8)
	if err != nil {
		return 0, -1
	}

	// Calculate color index: SPD address 0x50-0x57 maps to color index 0-7
	colorIndex := int(addr) - 0x50
	if colorIndex < 0 || colorIndex > 7 {
		colorIndex = -1
	}

	return uint8(addr), colorIndex
}

// parseSPDModule Parse the SPD data from the EEPROM file.
func parseSPDModule(path string, spd []byte) RAMModule {
	var m RAMModule
	m.EEPROMPath = path
	m.I2CAddress, m.ColorIndex = extractI2CAddress(path)
	parseSKUInfo(&m, spd)
	return m
}

// findEEPROMs traverse the hwmon directory structure to find EEPROMs
func findEEPROMs() ([]string, error) {
	var paths []string

	entries, err := os.ReadDir(hwmonRoot)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		namePath := filepath.Join(hwmonRoot, entry.Name(), "name")
		nameBytes, err := os.ReadFile(namePath)
		if err != nil {
			continue
		}
		if strings.TrimSpace(string(nameBytes)) == "spd5118" {
			eepromPath := filepath.Join(hwmonRoot, entry.Name(), "device", "eeprom")
			paths = append(paths, eepromPath)
		}
	}

	if len(paths) == 0 {
		return nil, fmt.Errorf("no EEPROMs found")
	}
	return paths, nil
}

// decodeEEPROMs reads the EEPROM files and decodes the SPD data into RAMModule structs.
func decodeEEPROMs(paths []string) []RAMModule {
	var modules []RAMModule
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "path": path}).Error("Failed to read eeprom data")
			continue
		}
		module := parseSPDModule(path, data)
		modules = append(modules, module)
	}
	return modules
}

// NewMemoryModules finds and decodes all memory modules in the system.
func NewMemoryModules() []RAMModule {
	paths, err := findEEPROMs()
	if err != nil {
		// If no EEPROMs are found, return an empty slice and the error
		return nil
	}
	return decodeEEPROMs(paths)
}

/*
// Print decoded SPD information to console. Not intended for production use, but useful for debugging.
func PrintModuleSPDInfo(m RAMModule) {
	fmt.Println("EEPROM Path:         ", m.EEPROMPath)
	fmt.Println("SKU:                 ", m.SKU)
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
