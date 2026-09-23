package lcd

import (
	"OpenLinkHub/src/config"
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/golang/freetype"
	"golang.org/x/image/font/opentype"
)

func setupAnimationProfileDir(t *testing.T) {
	t.Helper()
	t.Chdir(t.TempDir())
	if err := os.MkdirAll("database/lcd", 0700); err != nil {
		t.Fatal(err)
	}
	config.Init()
}

func TestSaveAnimationPreservesConcurrentUpload(t *testing.T) {
	restoreAnimationTestState(t)
	setupAnimationProfileDir(t)
	animation = &Animation{Background: "old", Images: map[string][]AnimationFrames{"old": nil}}
	lcd.ImageData = []ImageData{{Name: "old", Frames: 1}, {Name: "uploaded", Frames: 1}}
	// The real save handler obtains this snapshot before modifying fields.
	settings := GetAnimation()
	// An upload completes while that request is in flight.
	if LoadAnimation("uploaded") != 1 {
		t.Fatal("upload registration failed")
	}
	settings.Margin = 60
	if SaveAnimation(settings) != 1 {
		t.Fatal("save failed")
	}
	if _, ok := GetAnimation().Images["uploaded"]; !ok {
		t.Fatal("saving settings removed a successfully uploaded GIF from the dropdown")
	}
}

func TestSaveAnimationPreservesUploadInvalidation(t *testing.T) {
	restoreAnimationTestState(t)
	setupAnimationProfileDir(t)
	old := image.NewRGBA(image.Rect(0, 0, 2, 2))
	animation = &Animation{Background: "pulse", Images: map[string][]AnimationFrames{"pulse": {{Canvas: old}}}}
	lcd.ImageData = []ImageData{{Name: "pulse", Frames: 1}}
	settings := GetAnimation()
	if LoadAnimation("pulse") != 1 {
		t.Fatal("replacement registration failed")
	}
	if SaveAnimation(settings) != 1 {
		t.Fatal("save failed")
	}
	if len(animation.Images["pulse"]) != 0 {
		t.Fatal("settings save restored stale frames invalidated by the replacement upload")
	}
}

func writeColorGIF(t *testing.T, path string, c color.Color) {
	t.Helper()
	frame := image.NewPaletted(image.Rect(0, 0, 2, 2), color.Palette{c})
	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, &gif.GIF{Image: []*image.Paletted{frame}, Delay: []int{1}, Config: image.Config{ColorModel: frame.Palette, Width: 2, Height: 2}}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestDecodePalettedFramesDistinguishesFilenameCase(t *testing.T) {
	restoreAnimationTestState(t)
	images = t.TempDir()
	imgWidth, imgHeight = 2, 2
	animation = &Animation{}
	writeColorGIF(t, filepath.Join(images, "Pulse.GIF"), color.RGBA{R: 255, A: 255})
	writeColorGIF(t, filepath.Join(images, "pulse.gif"), color.RGBA{B: 255, A: 255})
	loadAnimationCatalog()
	if len(animation.Images) != 2 {
		t.Fatal("expected two independently selectable GIFs")
	}
	frames := decodePalettedFrames("pulse")
	if len(frames) != 1 {
		t.Fatal("decode failed")
	}
	r, _, b, _ := frames[0].At(0, 0).RGBA()
	if b <= r {
		t.Fatalf("selected blue pulse.gif but decoded red Pulse.gif: red=%d blue=%d", r, b)
	}
}

func TestThreeSensorAnimationBackgroundSwitch(t *testing.T) {
	restoreAnimationTestState(t)
	images = t.TempDir()
	imgWidth, imgHeight = 480, 480
	fontBytes, err := os.ReadFile("../../../static/fonts/teko.ttf")
	if err != nil {
		t.Fatal(err)
	}
	lcd.font, err = freetype.ParseFont(fontBytes)
	if err != nil {
		t.Fatal(err)
	}
	lcd.sfntFont, err = opentype.Parse(fontBytes)
	if err != nil {
		t.Fatal(err)
	}
	lcd.fontBytes = fontBytes
	animation = &Animation{Background: "red", Workers: 4, Margin: 60, Images: map[string][]AnimationFrames{}, Sensors: map[int]Sensors{
		0: {Enabled: true, Sensor: 0}, 1: {Enabled: true, Sensor: 2}, 2: {Enabled: true, Sensor: 1},
	}}
	writeColorGIF(t, filepath.Join(images, "red.gif"), color.RGBA{R: 255, A: 255})
	writeColorGIF(t, filepath.Join(images, "blue.gif"), color.RGBA{B: 255, A: 255})
	loadImage(filepath.Join(images, "red.gif"), ImageFormatGif)
	loadImage(filepath.Join(images, "blue.gif"), ImageFormatGif)
	loadAnimationCatalog()
	for _, name := range []string{"red", "blue"} {
		animation.Background = name
		pruneAnimationCache(name)
		frames := GenerateAnimationScreenImage([]float32{44, 39, 35})
		if len(frames) != 1 {
			t.Fatalf("%s: got %d frames", name, len(frames))
		}
		decoded, err := jpeg.Decode(bytes.NewReader(frames[0].Buffer))
		if err != nil {
			t.Fatal(err)
		}
		r, _, b, _ := decoded.At(0, 0).RGBA()
		if name == "red" && r <= b || name == "blue" && b <= r {
			t.Fatalf("wrong background for %s", name)
		}
	}
}

func TestConcurrentAnimationSaveUploadAndRender(t *testing.T) {
	restoreAnimationTestState(t)
	setupAnimationProfileDir(t)
	images = t.TempDir()
	imgWidth, imgHeight = 2, 2
	writeColorGIF(t, filepath.Join(images, "pulse.gif"), color.White)
	lcd.ImageData = []ImageData{{Name: "pulse", Frames: 1, Buffer: []Frames{{Delay: 10}}}}
	animation = &Animation{Background: "pulse", Workers: 1, Images: map[string][]AnimationFrames{"pulse": nil}}

	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		for i := 0; i < 32; i++ {
			settings := GetAnimation()
			settings.Workers = 1 + i%4
			settings.FrameDelay = i
			if SaveAnimation(settings) != 1 {
				t.Error("profile save failed")
				return
			}
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 32; i++ {
			if LoadAnimation("pulse") != 1 {
				t.Error("upload registration failed")
				return
			}
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 32; i++ {
			GenerateAnimationScreenImage(nil)
		}
	}()
	wg.Wait()
	if _, ok := GetAnimation().Images["pulse"]; !ok {
		t.Fatal("catalog entry lost")
	}
}
