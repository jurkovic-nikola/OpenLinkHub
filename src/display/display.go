package display

// Package: display
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/logger"
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

var (
	location = ""
	displays []common.Display
	mutex    sync.RWMutex
)

// Init will create and load display configuration
func Init() {
	location = config.GetConfig().ConfigPath + "/display.json"

	if !common.FileExists(location) {
		monitors := getScreenBounds()
		if err := common.SaveJsonData(location, monitors); err != nil {
			logger.Log(logger.Fields{"error": err, "location": location}).Error("Unable to save dashboard data")
			return
		}
	}

	file, err := os.Open(location)
	defer func(file *os.File) {
		err = file.Close()
		if err != nil {
			logger.Log(logger.Fields{"error": err, "file": location}).Warn("Failed to close file.")
		}
	}(file)

	if err != nil {
		logger.Log(logger.Fields{"error": err, "file": location}).Warn("Failed to open display file.")
		return
	}

	if err := json.NewDecoder(file).Decode(&displays); err != nil {
		logger.Log(logger.Fields{
			"error": err,
			"file":  location,
		}).Error("Failed to decode display config file")
	}
}

// GetDisplays will return list of displays
func GetDisplays() []common.Display {
	mutex.RLock()
	defer mutex.RUnlock()

	result := make([]common.Display, len(displays))
	copy(result, displays)

	return result
}

// GetScreenResolution will return combined screen resolution based on display placement.
func GetScreenResolution() common.Display {
	mutex.RLock()
	defer mutex.RUnlock()

	var width = 0
	var height = 0

	if len(displays) == 0 {
		return common.Display{
			Width:  1920,
			Height: 1080,
		}
	}

	width = displays[0].Width
	height = displays[0].Height

	for i := 1; i < len(displays); i++ {
		display := displays[i]

		if display.Left {
			width += display.Width

			if display.Height > height {
				height = display.Height
			}
		}

		if display.Top {
			height += display.Height

			if display.Width > width {
				width = display.Width
			}
		}
	}

	return common.Display{
		Width:  width,
		Height: height,
	}
}

// UpdateDisplay will update a display by index and save changes
func UpdateDisplay(display *common.Display) uint8 {
	mutex.Lock()
	defer mutex.Unlock()

	for i := range displays {
		if displays[i].Index == display.Index {
			displays[i].Width = display.Width
			displays[i].Height = display.Height
			displays[i].Left = display.Left
			displays[i].Top = display.Top

			if err := common.SaveJsonData(location, displays); err != nil {
				logger.Log(logger.Fields{
					"error":    err,
					"location": location,
					"index":    display.Index,
				}).Error("Unable to save display data")
				return 0
			}

			logger.Log(logger.Fields{
				"index":  display.Index,
				"width":  display.Width,
				"height": display.Height,
				"left":   display.Left,
				"top":    display.Top,
			}).Info("Display updated")
			return 1
		}
	}

	logger.Log(logger.Fields{"index": display.Index}).Warn("Display not found")
	return 0
}

// getScreenBounds will return list of connected displays
func getScreenBounds() []common.Display {
	var monitors []common.Display

	pattern := "/sys/class/drm/card*-*/modes"
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		logger.Log(logger.Fields{"error": err}).Warn("Could not find DRM modes")
		return []common.Display{
			{
				Name:   "fallback",
				Width:  1920,
				Height: 1080,
			},
		}
	}

	i := 0
	for _, modesFile := range matches {
		left := false
		connectorDir := filepath.Dir(modesFile)
		connectorName := filepath.Base(connectorDir)

		statusPath := filepath.Join(connectorDir, "status")
		statusBytes, err := os.ReadFile(statusPath)
		if err != nil {
			logger.Log(logger.Fields{"connector": connectorName, "error": err}).Warn("Could not read status")
			continue
		}

		if strings.TrimSpace(string(statusBytes)) != "connected" {
			continue
		}

		f, err := os.Open(modesFile)
		if err != nil {
			logger.Log(logger.Fields{"connector": connectorName, "error": err}).Warn("Could not open DRM file")
			continue
		}

		scanner := bufio.NewScanner(f)

		if scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			parts := strings.SplitN(line, "x", 2)

			if len(parts) == 2 {
				w, errW := strconv.Atoi(parts[0])
				h, errH := strconv.Atoi(parts[1])
				if errW == nil && errH == nil && w > 0 && h > 0 {
					i++
					if i > 1 {
						left = true
					}
					monitors = append(monitors, common.Display{
						Index:  i,
						Name:   connectorName,
						Width:  w,
						Height: h,
						Left:   left,
					})
				}
			}
		}

		if err := scanner.Err(); err != nil {
			logger.Log(logger.Fields{"connector": connectorName, "error": err}).Warn("Error reading DRM file")
		}

		err = f.Close()
		if err != nil {
			return nil
		}
	}

	if len(monitors) == 0 {
		logger.Log(logger.Fields{}).Warn("Could not parse any connected monitors, falling back to 1920x1080")
		return []common.Display{
			{Name: "fallback", Width: 1920, Height: 1080},
		}
	}

	logger.Log(logger.Fields{"monitors": monitors}).Info("Detected monitor bounds")
	return monitors
}
