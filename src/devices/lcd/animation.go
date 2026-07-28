package lcd

import (
	"LumenForge/src/common"
	"LumenForge/src/config"
	"LumenForge/src/logger"
	"LumenForge/src/rgb"
	"encoding/json"
	"image"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/image/draw"
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
	animationProfile := filepath.Join(config.GetPaths().MutableLCDRoot, "animation.json")
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

	animationData := make(map[string][]AnimationFrames)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, imageData := range lcd.ImageData {
		if imageData.PalettedFrames == nil {
			continue
		}
		wg.Add(1)

		go func(fileName string) {
			defer wg.Done()

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
		}(imageData.Name)
	}
	wg.Wait()
	animation.Images = animationData
}

// LoadAnimation will load animation based on filename
func LoadAnimation(fileName string) uint8 {
	mutex.Lock()
	if animation.Images == nil {
		mutex.Unlock()
		return 0
	}

	if _, ok := animation.Images[fileName]; ok {
		mutex.Unlock()
		return 2
	}
	mutex.Unlock()

	imageBuffer, ok := buildAnimationFrames(fileName)
	if !ok {
		return 0
	}

	mutex.Lock()
	defer mutex.Unlock()
	if animation.Images == nil {
		return 0
	}
	if _, ok = animation.Images[fileName]; ok {
		return 2
	}
	animation.Images[fileName] = imageBuffer
	return 1
}

func refreshAnimationCache(fileName string, animated bool) bool {
	mutex.Lock()
	defer mutex.Unlock()
	delete(animation.Images, fileName)
	if !animated {
		return true
	}
	imageBuffer, ok := buildAnimationFramesFrom(getPalettedFrames(fileName))
	if !ok {
		return false
	}
	if animation.Images == nil {
		animation.Images = make(map[string][]AnimationFrames)
	}
	animation.Images[fileName] = imageBuffer
	return true
}

func buildAnimationFrames(fileName string) ([]AnimationFrames, bool) {
	return buildAnimationFramesFrom(GetPalettedFrames(fileName))
}

func buildAnimationFramesFrom(palettedFrames ImageData) ([]AnimationFrames, bool) {
	if palettedFrames.PalettedFrames == nil {
		return nil, false
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
	return imageBuffer, true
}

// GetPalettedFrames will return
func GetPalettedFrames(fileName string) ImageData {
	mutex.Lock()
	defer mutex.Unlock()
	return getPalettedFrames(fileName)
}

func getPalettedFrames(fileName string) ImageData {
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
	profile := filepath.Join(config.GetPaths().MutableLCDRoot, "animation.json")

	if err := common.SaveJsonData(profile, animation); err != nil {
		logger.Log(logger.Fields{"error": err, "location": profile}).Error("Unable to write lcd profile data")
		return 0
	}
	return 1
}
