package lcd

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/rgb"
	"encoding/json"
	"image"
	"os"

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

	// Nothing is decoded here. Every cached frame holds a full size RGBA
	// canvas, and only the background named in the profile can ever be drawn,
	// so walking the image folder tied the resident set to how many images the
	// user happens to keep. Most panels never select the animation mode at all,
	// so even the one background is only worth decoding once something asks to
	// render it.
	pruneAnimationCache(animation.Background)
}

// buildAnimationFrames composites every frame of an image onto its own canvas,
// ready to be copied and annotated at render time. Returns nil when the image
// has no decoded frames to work from.
func buildAnimationFrames(fileName string) []AnimationFrames {
	paletted := decodePalettedFrames(fileName)
	if paletted == nil {
		return nil
	}
	// Delays live with the encoded frames, which are retained.
	var delays []Frames
	if img := GetLcdImage(fileName); img != nil {
		delays = img.Buffer
	}

	imageBuffer := make([]AnimationFrames, len(paletted))
	for i, pf := range paletted {
		var delay float64
		if i < len(delays) {
			delay = delays[i].Delay
		}
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
		}
	}
	return imageBuffer
}

// pruneAnimationCache drops every cached animation except keep, without loading
// anything. Used when the selected background changes: the old one is dead
// weight immediately, while the new one is only worth decoding if something
// actually renders it.
func pruneAnimationCache(keep string) {
	mutex.Lock()
	defer mutex.Unlock()

	if animation.Images == nil {
		animation.Images = make(map[string][]AnimationFrames)
		return
	}
	for name := range animation.Images {
		if name != keep {
			delete(animation.Images, name)
		}
	}
}

// ensureAnimationLoaded caches the frames for the named image and drops every
// other entry. Only the background named in the profile is ever rendered, so
// holding more than one costs a full canvas per frame for nothing.
func ensureAnimationLoaded(fileName string) bool {
	mutex.Lock()
	if animation.Images == nil {
		animation.Images = make(map[string][]AnimationFrames)
	}
	_, ok := animation.Images[fileName]
	mutex.Unlock()

	var frames []AnimationFrames
	if !ok {
		if fileName == "" {
			return false
		}
		// Built outside the lock: decoding is slow and the render path takes
		// the same mutex.
		if frames = buildAnimationFrames(fileName); frames == nil {
			return false
		}
	}

	mutex.Lock()
	defer mutex.Unlock()
	if frames != nil {
		animation.Images[fileName] = frames
	}
	for name := range animation.Images {
		if name != fileName {
			delete(animation.Images, name)
		}
	}
	return true
}

// LoadAnimation will load animation based on filename
func LoadAnimation(fileName string) uint8 {
	// Called from the upload handler, so it races the render path. The cache is
	// now pruned as well as extended, and an unguarded read against a delete is
	// a fatal concurrent map access rather than a stale answer.
	mutex.Lock()
	initialised := animation.Images != nil
	_, cached := animation.Images[fileName]
	mutex.Unlock()

	if !initialised {
		return 0
	}

	if cached {
		return 2
	}

	// Validate only. Uploading an image says nothing about whether it will ever
	// be drawn, and caching it here would retain a canvas per frame on the
	// strength of that guess. The render path decodes what it actually needs.
	// The upload handler has already decoded the file to check it, so knowing
	// the image registered with frames is enough.
	if img := GetLcdImage(fileName); img == nil || img.Frames < 1 {
		return 0
	}
	return 1
}

// GetAnimation will return Animation object
func GetAnimation() *Animation {
	return animation
}

// SaveAnimation will save animation profile
func SaveAnimation(value *Animation) uint8 {
	animation = value
	profile := config.GetConfig().ConfigPath + "/database/lcd/animation.json"

	if err := common.SaveJsonData(profile, animation); err != nil {
		logger.Log(logger.Fields{"error": err, "location": profile}).Error("Unable to write lcd profile data")
		return 0
	}

	// The background may have just changed, which makes the previously cached
	// one dead weight. Drop it; the render path decodes the new one when it
	// first needs it.
	pruneAnimationCache(animation.Background)
	return 1
}
