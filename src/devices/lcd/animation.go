package lcd

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/rgb"
	"encoding/json"
	"golang.org/x/image/draw"
	"image"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

type Animation struct {
	Id             int                          `json:"id"`
	Name           string                       `json:"name"`
	Background     string                       `json:"background"`
	Margin         float64                      `json:"margin"`
	Workers        int                          `json:"workers"`
	FrameDelay     int                          `json:"frameDelay"`
	SeparatorColor rgb.Color                    `json:"separatorColor"`
	Sensors        map[int]Sensors              `json:"sensors"`
	Images         map[string][]AnimationFrames `json:"-"`
}

type Sensors struct {
	Name      string    `json:"name"`
	Sensor    uint8     `json:"sensor"`
	TextColor rgb.Color `json:"textColor"`
	Enabled   bool      `json:"enabled"`
}

var (
	animation = new(Animation)
)

func InitAnimation() {
	animationProfile := config.GetConfig().ConfigPath + "/database/lcd/animation.json"
	if common.FileExists(animationProfile) {
		file, err := os.Open(animationProfile)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "location": animationProfile}).Error("Unable to load animation profile")
			return
		}
		if err = json.NewDecoder(file).Decode(&animation); err != nil {
			logger.Log(logger.Fields{"error": err, "location": animationProfile}).Error("Unable to decode animation profile")
			return
		}
	} else {
		// Initial setup
		data := &Animation{
			Id:         102,
			Name:       "Animation",
			Background: "concentric",
			Margin:     60,
			Workers:    4,
			FrameDelay: 0,
			SeparatorColor: rgb.Color{
				Red:        0,
				Green:      255,
				Blue:       255,
				Brightness: 0,
				Hex:        "#00ffff",
			},
			Sensors: map[int]Sensors{
				0: {
					Name:   "Sensor 1",
					Sensor: 0,
					TextColor: rgb.Color{
						Red:        255,
						Green:      255,
						Blue:       0,
						Brightness: 0,
						Hex:        "#ffff00",
					},
					Enabled: true,
				},
				1: {
					Name:   "Sensor 2",
					Sensor: 2,
					TextColor: rgb.Color{
						Red:        0,
						Green:      255,
						Blue:       255,
						Brightness: 0,
						Hex:        "#00ffff",
					},
					Enabled: true,
				},
				2: {
					Name:   "Sensor 3",
					Sensor: 1,
					TextColor: rgb.Color{
						Red:        0,
						Green:      255,
						Blue:       0,
						Brightness: 0,
						Hex:        "#00ff00",
					},
					Enabled: true,
				},
			},
		}
		animation = data
		if SaveAnimation(data) == 0 {
			logger.Log(logger.Fields{}).Warn("Unable to save animation profile. LCD will have default values")
		}
	}

	animationsFolder := pwd + "/database/lcd/images/"
	files, err := os.ReadDir(animationsFolder)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": animationsFolder}).Error("Unable to read content of a folder")
		return
	}

	animationData := make(map[string][]AnimationFrames)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, fi := range files {
		wg.Add(1)

		go func(fi os.DirEntry) {
			defer wg.Done()

			imagePath := animationsFolder + fi.Name()

			filenameFull := filepath.Base(imagePath)
			fileName := strings.TrimSuffix(filenameFull, filepath.Ext(filenameFull))

			// Validate file name
			if m, _ := regexp.MatchString("^[a-zA-Z0-9]+$", fileName); !m {
				logger.Log(logger.Fields{"location": animationsFolder, "image": imagePath}).Warn("Image name can only have letters and numbers. Please rename your image")
				return
			}

			if strings.ToLower(filepath.Ext(imagePath)) != ".gif" {
				return
			}

			palettedFrames := GetPalettedFrames(fileName)
			if palettedFrames.PalettedFrames == nil {
				return
			}

			imageBuffer := make([]AnimationFrames, len(palettedFrames.PalettedFrames))
			for i, pf := range palettedFrames.PalettedFrames {
				delay := palettedFrames.Buffer[i].Delay
				if delay == 0 {
					if animation.FrameDelay > 0 {
						delay = float64(animation.FrameDelay)
					}
				}

				canvas := image.NewRGBA(pf.Bounds())
				draw.Draw(canvas, canvas.Bounds(), pf, image.Point{}, draw.Over)

				imageBuffer[i] = AnimationFrames{
					Delay:  delay,
					Canvas: canvas,
					RGBA:   image.NewRGBA(canvas.Bounds()),
				}
			}

			mu.Lock()
			animationData[fileName] = imageBuffer
			mu.Unlock()
		}(fi)
	}
	wg.Wait()
	animation.Images = animationData
}

// GetPalettedFrames will return
func GetPalettedFrames(fileName string) ImageData {
	for _, val := range lcd.ImageData {
		if val.Name == fileName {
			return val
		}
	}
	return ImageData{}
}

// GetAnimation will return Animation object
func GetAnimation() *Animation {
	return animation
}

// SaveAnimation will save animation profile
func SaveAnimation(value *Animation) uint8 {
	animation = value
	profile := config.GetConfig().ConfigPath + "/database/lcd/animation.json"

	buffer, err := json.MarshalIndent(animation, "", "    ")
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to convert to json format")
		return 0
	}

	file, fileErr := os.Create(profile)
	if fileErr != nil {
		logger.Log(logger.Fields{"error": err, "location": profile}).Error("Unable to create animation profile")
		return 0
	}

	_, err = file.Write(buffer)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": profile}).Error("Unable to write data")
		return 0
	}

	err = file.Close()
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": profile}).Error("Unable to close file handle")
	}
	return 1
}
