package lcd

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLCDUploadPathCannotEscapeMutableRoot(t *testing.T) {
	root := t.TempDir()
	path, err := lcdUploadPath(root, "image.gif")
	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join(root, "image.gif") {
		t.Fatalf("upload path = %q", path)
	}

	for _, filename := range []string{"../escape.gif", "nested/escape.gif", ""} {
		if _, err = lcdUploadPath(root, filename); err == nil {
			t.Errorf("lcdUploadPath accepted %q", filename)
		}
	}
}

func TestBundledLCDImageLoadsFromReadOnlyApplicationRoot(t *testing.T) {
	applicationRoot := t.TempDir()
	mediaRoot := filepath.Join(applicationRoot, "database", "lcd", "images")
	if err := os.MkdirAll(mediaRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	imagePath := filepath.Join(mediaRoot, "bundled.gif")
	file, err := os.OpenFile(imagePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o444)
	if err != nil {
		t.Fatal(err)
	}
	pixel := image.NewRGBA(image.Rect(0, 0, 1, 1))
	pixel.Set(0, 0, color.RGBA{R: 255, A: 255})
	if err = gif.Encode(file, pixel, nil); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err = file.Close(); err != nil {
		t.Fatal(err)
	}
	if err = os.Chmod(applicationRoot, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(applicationRoot, 0o755)
	})

	original := lcd.ImageData
	lcd.ImageData = nil
	t.Cleanup(func() {
		lcd.ImageData = original
	})
	loadLcdImagesFrom(mediaRoot)

	if len(lcd.ImageData) != 1 || lcd.ImageData[0].Name != "bundled" {
		t.Fatalf("bundled image data = %#v", lcd.ImageData)
	}
}

func TestLCDLiveMediaReplacementRefreshesAnimationCache(t *testing.T) {
	root := t.TempDir()
	bundledRoot := filepath.Join(root, "bundled")
	mutableRoot := filepath.Join(root, "mutable")
	for _, directory := range []string{bundledRoot, mutableRoot} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
	}

	writeTestGIF(t, filepath.Join(bundledRoot, "shared.gif"), color.RGBA{R: 255, A: 255})
	writeTestGIF(t, filepath.Join(mutableRoot, "shared.gif"), color.RGBA{B: 255, A: 255})
	writeTestGIF(t, filepath.Join(mutableRoot, "unique.gif"), color.RGBA{G: 255, A: 255})

	mutex.Lock()
	originalImageData := lcd.ImageData
	originalAnimation := animation
	originalWidth := imgWidth
	originalHeight := imgHeight
	lcd.ImageData = nil
	animation = &Animation{Images: make(map[string][]AnimationFrames)}
	imgWidth = 2
	imgHeight = 2
	mutex.Unlock()
	t.Cleanup(func() {
		mutex.Lock()
		lcd.ImageData = originalImageData
		animation = originalAnimation
		imgWidth = originalWidth
		imgHeight = originalHeight
		mutex.Unlock()
	})

	loadLcdImagesFrom(bundledRoot)
	if !refreshAnimationCache("shared", true) {
		t.Fatal("failed to load bundled animation")
	}
	bundledFrames, ok := animationFramesForTest("shared")
	if !ok {
		t.Fatal("bundled animation cache entry is missing")
	}
	if len(bundledFrames) != 1 {
		t.Fatalf("bundled animation frame count = %d", len(bundledFrames))
	}
	bundledColor := bundledFrames[0].Canvas.RGBAAt(0, 0)

	loadLcdImagesFrom(mutableRoot)
	if len(lcd.ImageData) != 2 {
		t.Fatalf("image count after same-name and unique-name loads = %d", len(lcd.ImageData))
	}
	mutableImage := GetPalettedFrames("shared")
	if mutableImage.PalettedFrames == nil {
		t.Fatal("mutable same-name GIF did not replace bundled image data")
	}
	mutableColor := color.RGBAModel.Convert(mutableImage.PalettedFrames[0].At(0, 0)).(color.RGBA)
	if mutableColor.B <= mutableColor.R {
		t.Fatalf("mutable image color = %#v, want blue replacement", mutableColor)
	}

	if !refreshAnimationCache("shared", true) {
		t.Fatal("same-name GIF animation refresh reported a conflict or failure")
	}
	mutableFrames, ok := animationFramesForTest("shared")
	if !ok {
		t.Fatal("mutable animation cache entry is missing")
	}
	if len(mutableFrames) != 1 {
		t.Fatalf("mutable animation frame count = %d", len(mutableFrames))
	}
	mutableAnimationColor := mutableFrames[0].Canvas.RGBAAt(0, 0)
	if mutableAnimationColor == bundledColor || mutableAnimationColor.B <= mutableAnimationColor.R {
		t.Fatalf("animation color = %#v, bundled color = %#v", mutableAnimationColor, bundledColor)
	}

	if !refreshAnimationCache("unique", true) {
		t.Fatal("unique-name animation refresh failed")
	}
	uniqueFrames, ok := animationFramesForTest("unique")
	if !ok || len(uniqueFrames) != 1 {
		t.Fatal("unique-name animation was not added")
	}

	staticPath := filepath.Join(mutableRoot, "shared.jpg")
	writeTestJPEG(t, staticPath, color.RGBA{G: 255, B: 255, A: 255})
	if !loadImage(staticPath, ImageFormatJpg) {
		t.Fatal("failed to load same-name non-GIF replacement")
	}
	if !refreshAnimationCache("shared", false) {
		t.Fatal("failed to remove stale animation for non-GIF replacement")
	}
	if frames, exists := animationFramesForTest("shared"); exists {
		t.Fatalf("stale same-name animation remains after non-GIF replacement: %#v", frames)
	}
	if imageData := GetPalettedFrames("shared"); imageData.PalettedFrames != nil {
		t.Fatal("same-name non-GIF did not replace GIF image data")
	}
}

func TestCleanupMutableLCDSiblingsCrossExtensionAndContainment(t *testing.T) {
	root := t.TempDir()
	outsideRoot := t.TempDir()
	bundledRoot := t.TempDir()
	nestedRoot := filepath.Join(root, "nested")
	if err := os.MkdirAll(nestedRoot, 0o700); err != nil {
		t.Fatal(err)
	}

	keepPath := filepath.Join(root, "shared.gif")
	for _, path := range []string{
		keepPath,
		filepath.Join(root, "shared.jpg"),
		filepath.Join(root, "shared.jpeg"),
		filepath.Join(root, "shared.bmp"),
		filepath.Join(root, "shared.webp"),
		filepath.Join(root, "different.jpg"),
		filepath.Join(nestedRoot, "shared.jpg"),
		filepath.Join(outsideRoot, "shared.jpg"),
		filepath.Join(bundledRoot, "shared.jpg"),
	} {
		if err := os.WriteFile(path, []byte("image"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	outsideDirectory := filepath.Join(outsideRoot, "directory")
	if err := os.Mkdir(outsideDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	symlinkPath := filepath.Join(root, "shared.WEBP")
	if err := os.Symlink(outsideDirectory, symlinkPath); err != nil {
		t.Fatal(err)
	}

	if err := cleanupMutableLCDSiblings(root, "shared", keepPath); err != nil {
		t.Fatal(err)
	}

	for _, removed := range []string{
		filepath.Join(root, "shared.jpg"),
		filepath.Join(root, "shared.jpeg"),
		filepath.Join(root, "shared.bmp"),
		filepath.Join(root, "shared.webp"),
		symlinkPath,
	} {
		if _, err := os.Lstat(removed); !os.IsNotExist(err) {
			t.Errorf("obsolete same-name sibling remains at %q: %v", removed, err)
		}
	}
	for _, preserved := range []string{
		keepPath,
		filepath.Join(root, "different.jpg"),
		filepath.Join(nestedRoot, "shared.jpg"),
		filepath.Join(outsideRoot, "shared.jpg"),
		filepath.Join(bundledRoot, "shared.jpg"),
		outsideDirectory,
	} {
		if _, err := os.Lstat(preserved); err != nil {
			t.Errorf("cleanup removed or changed preserved path %q: %v", preserved, err)
		}
	}
}

func TestCleanupMutableLCDSiblingsJPEGAliases(t *testing.T) {
	for _, test := range []struct {
		name       string
		keepName   string
		removeName string
	}{
		{name: "jpg replaces jpeg", keepName: "shared.jpg", removeName: "shared.jpeg"},
		{name: "jpeg replaces jpg", keepName: "shared.jpeg", removeName: "shared.jpg"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			keepPath := filepath.Join(root, test.keepName)
			removePath := filepath.Join(root, test.removeName)
			for _, path := range []string{keepPath, removePath} {
				if err := os.WriteFile(path, []byte("image"), 0o600); err != nil {
					t.Fatal(err)
				}
			}

			if err := cleanupMutableLCDSiblings(root, "shared", keepPath); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(keepPath); err != nil {
				t.Fatalf("replacement %q was removed: %v", test.keepName, err)
			}
			if _, err := os.Stat(removePath); !os.IsNotExist(err) {
				t.Fatalf("obsolete alias %q remains: %v", test.removeName, err)
			}
		})
	}
}

func TestCleanupMutableLCDSiblingsKeepsRestartStateConsistent(t *testing.T) {
	root := t.TempDir()
	oldPath := filepath.Join(root, "shared.jpg")
	replacementPath := filepath.Join(root, "shared.gif")
	writeTestJPEG(t, oldPath, color.RGBA{R: 255, A: 255})
	writeTestGIF(t, replacementPath, color.RGBA{B: 255, A: 255})
	if err := os.Chmod(replacementPath, 0o600); err != nil {
		t.Fatal(err)
	}

	if err := cleanupMutableLCDSiblings(root, "shared", replacementPath); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf("old cross-extension sibling remains: %v", err)
	}
	if info, err := os.Stat(replacementPath); err != nil {
		t.Fatal(err)
	} else if mode := info.Mode().Perm(); mode != 0o600 {
		t.Fatalf("replacement mode = %#o, want 0600", mode)
	}

	mutex.Lock()
	originalImageData := lcd.ImageData
	originalAnimation := animation
	originalWidth := imgWidth
	originalHeight := imgHeight
	lcd.ImageData = nil
	animation = &Animation{Images: make(map[string][]AnimationFrames)}
	imgWidth = 2
	imgHeight = 2
	mutex.Unlock()
	t.Cleanup(func() {
		mutex.Lock()
		lcd.ImageData = originalImageData
		animation = originalAnimation
		imgWidth = originalWidth
		imgHeight = originalHeight
		mutex.Unlock()
	})

	loadLcdImagesFrom(root)
	if imageData := GetPalettedFrames("shared"); imageData.PalettedFrames == nil {
		t.Fatal("replacement GIF was not selected after simulated restart")
	}
	if !refreshAnimationCache("shared", true) {
		t.Fatal("replacement GIF animation did not rebuild after simulated restart")
	}
	if frames, ok := animationFramesForTest("shared"); !ok || len(frames) != 1 {
		t.Fatalf("replacement animation after restart = %#v, present=%v", frames, ok)
	}
}

func TestTransactionalLCDUploadCreatesNewMedia(t *testing.T) {
	root := t.TempDir()
	destination := filepath.Join(root, "fresh.gif")
	tempPath := stageTestLCDUpload(t, root, "fresh", color.RGBA{G: 255, A: 255})
	preserveLCDTransactionState(t, nil, nil)

	if err := transactMutableLCDUpload(
		root,
		"fresh",
		tempPath,
		destination,
		ImageFormatGif,
		defaultLCDUploadTransactionOps(),
	); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(destination)
	if err != nil {
		t.Fatal(err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Fatalf("destination mode = %#o, want 0600", mode)
	}
	if imageData := GetPalettedFrames("fresh"); imageData.PalettedFrames == nil {
		t.Fatal("new upload was not installed in live image state")
	}
	if frames, ok := animationFramesForTest("fresh"); !ok || len(frames) != 1 {
		t.Fatalf("new upload animation = %#v, present=%v", frames, ok)
	}
	assertNoLCDTransactionArtifacts(t, root)
}

func TestTransactionalLCDUploadOverwritesExistingDestination(t *testing.T) {
	root := t.TempDir()
	destination := filepath.Join(root, "shared.gif")
	writeTestGIF(t, destination, color.RGBA{R: 255, A: 255})
	originalBytes, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	tempPath := stageTestLCDUpload(t, root, "shared", color.RGBA{B: 255, A: 255})
	preserveLCDTransactionState(t, []ImageData{{Name: "shared", Frames: 7}}, map[string][]AnimationFrames{
		"shared": {{Delay: 99}},
	})

	if err = transactMutableLCDUpload(
		root,
		"shared",
		tempPath,
		destination,
		ImageFormatGif,
		defaultLCDUploadTransactionOps(),
	); err != nil {
		t.Fatal(err)
	}

	replacementBytes, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(replacementBytes, originalBytes) {
		t.Fatal("successful overwrite retained the previous destination bytes")
	}
	if imageData := GetPalettedFrames("shared"); imageData.PalettedFrames == nil || imageData.Frames != 1 {
		t.Fatalf("live image after overwrite = %#v", imageData)
	}
	if frames, ok := animationFramesForTest("shared"); !ok || len(frames) != 1 || frames[0].Delay == 99 {
		t.Fatalf("animation after overwrite = %#v, present=%v", frames, ok)
	}
	assertNoLCDTransactionArtifacts(t, root)
}

func TestTransactionalLCDUploadActivationFailureRestoresDestinationAndLiveState(t *testing.T) {
	root := t.TempDir()
	destination := filepath.Join(root, "shared.gif")
	writeTestGIF(t, destination, color.RGBA{R: 255, A: 255})
	originalBytes, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	tempPath := stageTestLCDUpload(t, root, "shared", color.RGBA{B: 255, A: 255})
	preserveLCDTransactionState(t, []ImageData{{Name: "shared", Frames: 7}}, map[string][]AnimationFrames{
		"shared": {{Delay: 99}},
	})

	ops := defaultLCDUploadTransactionOps()
	ops.beforeActivate = func() error {
		return errors.New("injected activation failure")
	}
	err = transactMutableLCDUpload(root, "shared", tempPath, destination, ImageFormatGif, ops)
	if err == nil || !strings.Contains(err.Error(), "injected activation failure") {
		t.Fatalf("activation failure = %v", err)
	}

	restoredBytes, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(restoredBytes, originalBytes) {
		t.Fatal("activation failure changed the original destination")
	}
	assertLCDLiveStateUnchanged(t, "shared")
	assertNoLCDTransactionArtifacts(t, root)
}

func TestTransactionalLCDUploadSiblingCleanupFailureRollsBack(t *testing.T) {
	root := t.TempDir()
	destination := filepath.Join(root, "shared.gif")
	sibling := filepath.Join(root, "shared.jpg")
	writeTestGIF(t, destination, color.RGBA{R: 255, A: 255})
	writeTestJPEG(t, sibling, color.RGBA{G: 255, A: 255})
	originalDestination, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	originalSibling, err := os.ReadFile(sibling)
	if err != nil {
		t.Fatal(err)
	}
	tempPath := stageTestLCDUpload(t, root, "shared", color.RGBA{B: 255, A: 255})
	preserveLCDTransactionState(t, []ImageData{{Name: "shared", Frames: 7}}, map[string][]AnimationFrames{
		"shared": {{Delay: 99}},
	})

	ops := defaultLCDUploadTransactionOps()
	ops.rename = func(oldPath, newPath string) error {
		if oldPath == sibling {
			return errors.New("injected sibling cleanup failure")
		}
		return os.Rename(oldPath, newPath)
	}
	err = transactMutableLCDUpload(root, "shared", tempPath, destination, ImageFormatGif, ops)
	if err == nil || !strings.Contains(err.Error(), "injected sibling cleanup failure") {
		t.Fatalf("sibling cleanup failure = %v", err)
	}

	restoredDestination, readErr := os.ReadFile(destination)
	if readErr != nil {
		t.Fatal(readErr)
	}
	restoredSibling, readErr := os.ReadFile(sibling)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(restoredDestination, originalDestination) {
		t.Fatal("cleanup failure changed the original destination")
	}
	if !bytes.Equal(restoredSibling, originalSibling) {
		t.Fatal("cleanup failure changed the obsolete sibling")
	}
	assertLCDLiveStateUnchanged(t, "shared")
	assertNoLCDTransactionArtifacts(t, root)
}

func TestTransactionalLCDUploadCrossExtensionCleanupRunsAfterActivation(t *testing.T) {
	root := t.TempDir()
	destination := filepath.Join(root, "shared.gif")
	sibling := filepath.Join(root, "shared.jpg")
	writeTestJPEG(t, sibling, color.RGBA{R: 255, A: 255})
	tempPath := stageTestLCDUpload(t, root, "shared", color.RGBA{B: 255, A: 255})
	preserveLCDTransactionState(t, []ImageData{{Name: "shared", Frames: 7}}, map[string][]AnimationFrames{
		"shared": {{Delay: 99}},
	})

	activationObserved := false
	ops := defaultLCDUploadTransactionOps()
	ops.beforeActivate = func() error {
		activationObserved = true
		if _, err := os.Stat(sibling); err != nil {
			return errors.New("obsolete sibling removed before activation")
		}
		if imageData := GetPalettedFrames("shared"); imageData.Frames != 7 {
			return errors.New("live state changed before activation")
		}
		return nil
	}
	if err := transactMutableLCDUpload(root, "shared", tempPath, destination, ImageFormatGif, ops); err != nil {
		t.Fatal(err)
	}
	if !activationObserved {
		t.Fatal("activation seam was not reached")
	}
	if _, err := os.Stat(sibling); !os.IsNotExist(err) {
		t.Fatalf("obsolete sibling remains after successful activation: %v", err)
	}
	if _, err := os.Stat(destination); err != nil {
		t.Fatal(err)
	}
	assertNoLCDTransactionArtifacts(t, root)
}

func writeTestGIF(t *testing.T, path string, frameColors ...color.RGBA) {
	t.Helper()
	frames := make([]*image.Paletted, len(frameColors))
	for index, frameColor := range frameColors {
		frame := image.NewPaletted(image.Rect(0, 0, 2, 2), color.Palette{
			color.RGBA{A: 255},
			frameColor,
		})
		for pixel := range frame.Pix {
			frame.Pix[pixel] = 1
		}
		frames[index] = frame
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = gif.EncodeAll(file, &gif.GIF{
		Image: frames,
		Delay: make([]int, len(frames)),
	}); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err = file.Close(); err != nil {
		t.Fatal(err)
	}
}

func writeTestJPEG(t *testing.T, path string, fill color.RGBA) {
	t.Helper()
	pixels := image.NewRGBA(image.Rect(0, 0, 2, 2))
	for x := 0; x < 2; x++ {
		for y := 0; y < 2; y++ {
			pixels.SetRGBA(x, y, fill)
		}
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = jpeg.Encode(file, pixels, nil); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err = file.Close(); err != nil {
		t.Fatal(err)
	}
}

func animationFramesForTest(name string) ([]AnimationFrames, bool) {
	mutex.Lock()
	defer mutex.Unlock()
	frames, ok := animation.Images[name]
	return frames, ok
}

func stageTestLCDUpload(t *testing.T, root, baseName string, fill color.RGBA) string {
	t.Helper()
	file, err := os.CreateTemp(root, "."+baseName+"-upload-*")
	if err != nil {
		t.Fatal(err)
	}
	tempPath := file.Name()
	if err = file.Close(); err != nil {
		t.Fatal(err)
	}
	writeTestGIF(t, tempPath, fill)
	if err = os.Chmod(tempPath, 0o600); err != nil {
		t.Fatal(err)
	}
	return tempPath
}

func preserveLCDTransactionState(
	t *testing.T,
	imageData []ImageData,
	animationImages map[string][]AnimationFrames,
) {
	t.Helper()
	mutex.Lock()
	originalImageData := lcd.ImageData
	originalAnimation := animation
	originalWidth := imgWidth
	originalHeight := imgHeight
	lcd.ImageData = imageData
	animation = &Animation{Images: animationImages}
	imgWidth = 2
	imgHeight = 2
	mutex.Unlock()
	t.Cleanup(func() {
		mutex.Lock()
		lcd.ImageData = originalImageData
		animation = originalAnimation
		imgWidth = originalWidth
		imgHeight = originalHeight
		mutex.Unlock()
	})
}

func assertLCDLiveStateUnchanged(t *testing.T, name string) {
	t.Helper()
	if imageData := GetPalettedFrames(name); imageData.Frames != 7 {
		t.Fatalf("live image changed after failed transaction: %#v", imageData)
	}
	if frames, ok := animationFramesForTest(name); !ok || len(frames) != 1 || frames[0].Delay != 99 {
		t.Fatalf("live animation changed after failed transaction: %#v, present=%v", frames, ok)
	}
}

func assertNoLCDTransactionArtifacts(t *testing.T, root string) {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), "-upload-") || strings.Contains(entry.Name(), "-rollback-") {
			t.Errorf("LCD transaction artifact remains: %s", entry.Name())
		}
	}
}
