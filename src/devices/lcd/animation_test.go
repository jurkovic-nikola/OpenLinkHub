package lcd

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestPerformImageUploadRegistersGIF(t *testing.T) {
	restoreAnimationTestState(t)
	images = t.TempDir()
	imgWidth, imgHeight = 2, 2
	animation = &Animation{Images: make(map[string][]AnimationFrames)}
	lcd.ImageData = nil

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)
	part, err := writer.CreateFormFile("animationFile", "uploaded.gif")
	if err != nil {
		t.Fatal(err)
	}
	data := testGIFBytes(t)
	for len(data) < 512 {
		data = append(data, 0)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/lcd/upload", &requestBody)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()
	PerformImageUpload(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("upload returned HTTP %d: %s", response.Code, response.Body.String())
	}
	if _, ok := animation.Images["uploaded"]; !ok {
		t.Fatal("uploaded GIF was not registered in the animation catalog")
	}
}

func TestLoadAnimationRegistersUploadedImage(t *testing.T) {
	restoreAnimationTestState(t)
	animation = &Animation{Images: make(map[string][]AnimationFrames)}
	lcd.ImageData = []ImageData{{Name: "uploaded", Frames: 1}}

	if status := LoadAnimation("uploaded"); status != 1 {
		t.Fatalf("LoadAnimation returned %d, want 1", status)
	}
	if _, ok := animation.Images["uploaded"]; !ok {
		t.Fatal("uploaded image was not registered in the animation catalog")
	}
}

func TestLoadAnimationInvalidatesReplacedImage(t *testing.T) {
	restoreAnimationTestState(t)
	animation = &Animation{
		Images: map[string][]AnimationFrames{
			"uploaded": {{Canvas: image.NewRGBA(image.Rect(0, 0, 1, 1))}},
		},
	}
	lcd.ImageData = []ImageData{{Name: "uploaded", Frames: 1}}

	if status := LoadAnimation("uploaded"); status != 1 {
		t.Fatalf("LoadAnimation returned %d, want 1", status)
	}
	if frames := animation.Images["uploaded"]; frames != nil {
		t.Fatalf("replaced image retained %d stale decoded frames", len(frames))
	}
}

func TestLoadAnimationCatalogFindsGIFsWithoutDecoding(t *testing.T) {
	restoreAnimationTestState(t)
	images = t.TempDir()
	animation = &Animation{}

	writeTestGIF(t, filepath.Join(images, "first.gif"))
	if err := os.WriteFile(filepath.Join(images, "not-an-animation.jpg"), []byte("ignored"), 0o600); err != nil {
		t.Fatal(err)
	}

	loadAnimationCatalog()
	if _, ok := animation.Images["first"]; !ok {
		t.Fatal("GIF on disk was not registered in the animation catalog")
	}
	if frames := animation.Images["first"]; frames != nil {
		t.Fatalf("catalog load decoded %d frames eagerly", len(frames))
	}
	if _, ok := animation.Images["not-an-animation"]; ok {
		t.Fatal("non-GIF image was registered as an animation")
	}
}

func TestEnsureAnimationLoadedDecodesCatalogEntry(t *testing.T) {
	restoreAnimationTestState(t)
	images = t.TempDir()
	imgWidth, imgHeight = 2, 2
	animation = &Animation{Images: map[string][]AnimationFrames{"pulse": nil}}
	lcd.ImageData = []ImageData{{
		Name:   "pulse",
		Frames: 1,
		Buffer: []Frames{{Delay: 10}},
	}}
	writeTestGIF(t, filepath.Join(images, "pulse.gif"))

	if !ensureAnimationLoaded("pulse") {
		t.Fatal("ensureAnimationLoaded returned false")
	}
	if frames := animation.Images["pulse"]; len(frames) != 1 {
		t.Fatalf("decoded %d frames, want 1", len(frames))
	}
}

func writeTestGIF(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, testGIFBytes(t), 0o600); err != nil {
		t.Fatal(err)
	}
}

func testGIFBytes(t *testing.T) []byte {
	t.Helper()
	palette := color.Palette{color.Black, color.White}
	frame := image.NewPaletted(image.Rect(0, 0, 2, 2), palette)
	frame.SetColorIndex(0, 0, 1)

	var data bytes.Buffer
	if err := gif.EncodeAll(&data, &gif.GIF{
		Image: []*image.Paletted{frame},
		Delay: []int{1},
		Config: image.Config{
			ColorModel: palette,
			Width:      2,
			Height:     2,
		},
	}); err != nil {
		t.Fatal(err)
	}
	return data.Bytes()
}

func restoreAnimationTestState(t *testing.T) {
	t.Helper()
	previousAnimation := animation
	previousLCD := lcd
	previousImages := images
	previousWidth, previousHeight := imgWidth, imgHeight
	t.Cleanup(func() {
		animation = previousAnimation
		lcd = previousLCD
		images = previousImages
		imgWidth, imgHeight = previousWidth, previousHeight
	})
}
