package lcd

// Package: LCD Controller
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"LumenForge/src/common"
	"LumenForge/src/config"
	"LumenForge/src/dashboard"
	"LumenForge/src/logger"
	"LumenForge/src/rgb"
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"github.com/sstallion/go-hid"
	"golang.org/x/image/bmp"
	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	_ "golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/webp"
	_ "golang.org/x/image/webp"
)

const (
	DisplayLiquid         uint8 = 0
	DisplayPump           uint8 = 1
	DisplayCPU            uint8 = 2
	DisplayGPU            uint8 = 3
	DisplayAllInOne       uint8 = 4
	DisplayLiquidCPU      uint8 = 5
	DisplayCpuGpuTemp     uint8 = 6
	DisplayCpuGpuLoad     uint8 = 7
	DisplayCpuGpuLoadTemp uint8 = 8
	DisplayTime           uint8 = 9
	DisplayImage          uint8 = 10
	DisplayArc            uint8 = 100
	DisplayDoubleArc      uint8 = 101
	DisplayAnimation      uint8 = 102
)

const (
	ImageFormatJpg  = 0
	ImageFormatBmp  = 1
	ImageFormatWebp = 2
	ImageFormatGif  = 3
)

var (
	location      = ""
	images        = ""
	shippedImages = ""
	fontLocation  = ""
	mutex         sync.Mutex
	uploadMutex   sync.Mutex
	imgWidth             = 480
	imgHeight            = 480
	lcdDevices           = map[string]uint16{}
	vendorId      uint16 = 6940 // Corsair
	lcdSensors           = map[uint8]string{
		0: "CPU Temp",
		1: "GPU Temp",
		2: "Liquid Temp",
		3: "CPU Load",
		4: "GPU Load",
	}
	sensorTextCache = make(map[uint8]string)
	lcdPresent      = false
)

type ImageData struct {
	Name           string
	Frames         int
	Buffer         []Frames
	PalettedFrames []*image.Paletted
}

type Frames struct {
	Buffer []byte  `json:"-"`
	Delay  float64 `json:"-"`
}

type AnimationFrames struct {
	Delay  float64
	Canvas *image.RGBA
	RGBA   *image.RGBA
}
type LCD struct {
	image     image.Image
	font      *truetype.Font
	fontBytes []byte
	sfntFont  *opentype.Font
	Devices   []Device
	ImageData []ImageData
}

type Device struct {
	Lcd       *hid.Device
	ProductId uint16
	VendorId  uint16
	Product   string
	Serial    string
	AIO       bool
}

var lcd LCD

// Init will initialize LCD data
func Init() {
	paths := config.GetPaths()
	location = filepath.Join(paths.ShippedLCDRoot, "background.jpg")
	images = paths.MutableLCDUploadRoot
	shippedImages = paths.ShippedLCDMediaRoot
	fontLocation = filepath.Join(paths.StaticAssetRoot, "fonts", "teko.ttf")

	lcdDevices = make(map[string]uint16)
	lcdPresent = false

	checkForLcd()
	if !lcdPresent {
		logger.Log(logger.Fields{}).Info("No valid LCD devices found")
		return
	}

	// Open image
	file, e := os.Open(location)
	if e != nil {
		logger.Log(logger.Fields{"error": e, "location": location}).Error("Unable to open LCD image")
		return
	}
	defer func(file *os.File) {
		closeErr := file.Close()
		if closeErr != nil {
			logger.Log(logger.Fields{"error": closeErr, "location": location}).Error("Unable to close LCD image")
		}
	}(file)

	// Decode the image
	img, e := jpeg.Decode(file)
	if e != nil {
		logger.Log(logger.Fields{"error": e, "location": location}).Error("Unable to decode LCD image")
		return
	}

	// Load LCD font
	fontBytes, e := os.ReadFile(fontLocation) // Provide the path to your .ttf font file
	if e != nil {
		logger.Log(logger.Fields{"error": e, "location": fontLocation}).Error("Unable to get LCD font")
		return
	}
	fontParsed, e := freetype.ParseFont(fontBytes)
	if e != nil {
		logger.Log(logger.Fields{"error": e, "location": fontLocation}).Error("Unable to parse LCD font")
	}

	sfntFont, e := opentype.Parse(fontBytes)
	if e != nil {
		logger.Log(logger.Fields{"error": e, "location": fontLocation}).Error("Unable to parse LCD font")
	}

	lcdData := &LCD{
		image:     img,
		font:      fontParsed,
		fontBytes: fontBytes,
		sfntFont:  sfntFont,
	}
	lcd = *lcdData
	loadLcdImages()
	loadLcdDevices()

	// Single Arc
	InitArc()

	// Double arc
	InitDoubleArc()

	// Animations
	InitAnimation()

	for i := range lcdSensors {
		sensorTextCache[i] = strings.ToUpper(lcdSensors[i])
	}
}

// Reconnect will reconnect to all available LCD devices
func Reconnect() {
	for key, device := range lcd.Devices {
		lcdPanel, e := hid.Open(vendorId, device.ProductId, device.Serial)
		if e != nil {
			logger.Log(logger.Fields{"error": e, "vendorId": vendorId, "productId": device.ProductId}).Error("Unable to reconnect LCD HID device")
			continue
		}
		lcd.Devices[key].Lcd = lcdPanel
	}
}

// GetLcdImages will return all lcd images
func GetLcdImages() []ImageData {
	return lcd.ImageData
}

// GetLcdImage will return image data based on image name
func GetLcdImage(image string) *ImageData {
	for _, value := range lcd.ImageData {
		if value.Name == image {
			return &value
		}
	}
	return nil
}

// GetLcdBySerial will return HID device by serial number
func GetLcdBySerial(serial string) *hid.Device {
	for _, device := range lcd.Devices {
		if device.Serial == serial {
			return device.Lcd
		}
	}
	return nil
}

// GetLcdByProductId will return HID device by product id
func GetLcdByProductId(productId uint16) *hid.Device {
	for _, device := range lcd.Devices {
		if device.ProductId == productId {
			return device.Lcd
		}
	}
	return nil
}

// GetNonAIOLCDSerials will return serial numbers of XD5 pumps
func GetNonAIOLCDSerials() []string {
	var serials []string
	for _, device := range lcd.Devices {
		if device.AIO {
			continue
		}
		serials = append(serials, device.Serial)
	}
	return serials
}

// GetAioLCDSerial will return serial number of AIO LCD pumps
func GetAioLCDSerial() string {
	for _, device := range lcd.Devices {
		if device.AIO {
			return device.Serial
		}
	}
	return ""
}

// GetAioLCDData will return data of AIO LCD pumps
func GetAioLCDData() *Device {
	for _, device := range lcd.Devices {
		if device.AIO {
			return &device
		}
	}
	return nil
}

// GetNonAioLCDData will return data of XD5 LCD pumps
func GetNonAioLCDData() []Device {
	var devices []Device
	for _, device := range lcd.Devices {
		if !device.AIO {
			devices = append(devices, device)
		}
	}
	return devices
}

// GetLcdAmount will return amount of available LCDs
func GetLcdAmount() int {
	return len(lcd.Devices)
}

// GetLcdDevices will return all LCD devices
func GetLcdDevices() []Device {
	return lcd.Devices
}

// GetLcdSensors will return list of LCD sensors
func GetLcdSensors() map[uint8]string {
	return lcdSensors
}

// generateColor will generate color.RGBA based on red, green and blue
func generateColor(c rgb.Color) color.RGBA {
	return color.RGBA{R: uint8(c.Red), G: uint8(c.Green), B: uint8(c.Blue), A: 255}
}

// drawCircle will draw filled circle
func drawCircle(img *image.RGBA, centerX, centerY, radius float64, col color.Color) {
	minX := int(centerX - radius)
	maxX := int(centerX + radius)
	minY := int(centerY - radius)
	maxY := int(centerY + radius)

	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			dx := float64(x) - centerX
			dy := float64(y) - centerY
			if dx*dx+dy*dy <= radius*radius {
				if x >= 0 && y >= 0 && x < img.Bounds().Dx() && y < img.Bounds().Dy() {
					img.Set(x, y, col)
				}
			}
		}
	}
}

// drawSmoothArcGradient will draw arc with color gradient
func drawSmoothArcGradient(img *image.RGBA, centerX, centerY, innerR, outerR float64, startAngle, endAngle float64, cStart, cEnd color.RGBA) {
	step := 0.01
	angleRange := endAngle - startAngle

	for angle := startAngle; angle <= endAngle; angle += step {
		t := (angle - startAngle) / angleRange
		col := interpolateColor(cStart, cEnd, t)

		radius := (innerR + outerR) / 2
		x := centerX + radius*math.Cos(angle)
		y := centerY + radius*math.Sin(angle)
		drawCircle(img, x, y, (outerR-innerR)/2, col)
	}
}

func calculateStringXY(fontSize float64, value string) (int, int) {
	opts := opentype.FaceOptions{Size: fontSize, DPI: 72, Hinting: 0}
	fontFace, err := opentype.NewFace(lcd.sfntFont, &opts)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to process font face")
	}

	bounds, _ := font.BoundString(fontFace, value)
	textWidth := (bounds.Max.X - bounds.Min.X).Ceil()
	textHeight := (bounds.Max.Y - bounds.Min.Y).Ceil()

	x := (imgWidth - textWidth) / 2
	y := (imgHeight+textHeight)/2 - 10
	return x, y
}

// sensorMaximumValue will return sensor maximum value
func sensorMaximumValue(sensor uint8) int {
	switch sensor {
	case 0:
		return 100
	case 1:
		return 90
	case 2:
		return 60
	case 3, 4:
		return 100
	default:
		return 100
	}
}

// isSensorTemperature will check if given sensor is temperature one
func isSensorTemperature(sensor uint8) bool {
	if sensor == 0 || sensor == 1 || sensor == 2 {
		return true
	}
	return false
}

// drawArcOutline will draw small outline
func drawArcOutline(img *image.RGBA, centerX, centerY, innerR, outerR float64, startAngle, endAngle float64, col color.RGBA) {
	step := 0.01
	radius := (innerR + outerR) / 2
	thickness := 3.0 // very thin

	for angle := startAngle; angle <= endAngle; angle += step {
		x := centerX + radius*math.Cos(angle)
		y := centerY + radius*math.Sin(angle)
		drawCircle(img, x, y, thickness, col)
	}
}

// interpolateColor will interpolate between two colors
func interpolateColor(c1, c2 color.RGBA, t float64) color.RGBA {
	return color.RGBA{
		R: uint8(float64(c1.R)*(1-t) + float64(c2.R)*t),
		G: uint8(float64(c1.G)*(1-t) + float64(c2.G)*t),
		B: uint8(float64(c1.B)*(1-t) + float64(c2.B)*t),
		A: 255,
	}
}

// PerformImageUpload will handle image upload
func PerformImageUpload(w http.ResponseWriter, r *http.Request) {
	const maxUploadSize = 5 * 1024 * 1024 // 5MB

	if r.Method != http.MethodPost {
		http.Error(w, "Use POST to upload image file", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		logger.Log(logger.Fields{"error": err}).Error("File too large or invalid upload")
		http.Error(w, "File too large or invalid upload", http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile("animationFile")
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Failed to read uploaded file")
		http.Error(w, "Failed to read uploaded file: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer func(file multipart.File) {
		if cerr := file.Close(); cerr != nil {
			logger.Log(logger.Fields{"error": cerr}).Error("Failed to close file")
		}
	}(file)

	ext := strings.ToLower(filepath.Ext(handler.Filename))
	name := strings.TrimSuffix(filepath.Base(handler.Filename), filepath.Ext(handler.Filename))
	if !common.AlphanumericRegex.MatchString(name) {
		http.Error(w, "Invalid filename. Only letters and numbers allowed", http.StatusBadRequest)
		return
	}

	type decoderFn func(r io.Reader) error
	specs := map[string]struct {
		mime   string
		decode decoderFn
		format uint8
	}{
		".gif": {
			mime: "image/gif",
			decode: func(r io.Reader) error {
				_, err := gif.Decode(r)
				return err
			},
			format: ImageFormatGif,
		},
		".jpg": {
			mime: "image/jpeg",
			decode: func(r io.Reader) error {
				_, err := jpeg.Decode(r)
				return err
			},
			format: ImageFormatJpg,
		},
		".jpeg": {
			mime: "image/jpeg",
			decode: func(r io.Reader) error {
				_, err := jpeg.Decode(r)
				return err
			},
			format: ImageFormatJpg,
		},
		".webp": {
			mime: "image/webp",
			decode: func(r io.Reader) error {
				_, err := webp.Decode(r)
				return err
			},
			format: ImageFormatWebp,
		},
		".bmp": {
			mime: "image/bmp",
			decode: func(r io.Reader) error {
				_, err := bmp.Decode(r)
				return err
			},
			format: ImageFormatBmp,
		},
	}

	spec, ok := specs[ext]
	if !ok {
		http.Error(w, "Invalid file type. Only .gif, .jpg, .jpeg, .webp, .bmp allowed", http.StatusBadRequest)
		return
	}

	header := make([]byte, 512)
	if _, err := file.Read(header); err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to inspect file")
		http.Error(w, "Unable to inspect file", http.StatusBadRequest)
		return
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Seek failed")
		http.Error(w, "Unable to inspect file", http.StatusBadRequest)
		return
	}

	detected := http.DetectContentType(header)
	if detected != spec.mime {
		logger.Log(logger.Fields{"detected": detected, "expected": spec.mime}).Error("MIME mismatch")
		http.Error(w, "Invalid file content for extension", http.StatusBadRequest)
		return
	}

	if err := spec.decode(file); err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Corrupted or invalid image")
		http.Error(w, "Corrupted or invalid image", http.StatusBadRequest)
		return
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Seek failed")
		http.Error(w, "Unable to save file", http.StatusBadRequest)
		return
	}

	filename := name + ext
	savePath, err := lcdUploadPath(images, filename)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "filename": filename}).Error("Invalid LCD upload destination")
		http.Error(w, "Invalid upload destination", http.StatusBadRequest)
		return
	}

	tempPath, err := writeMutableLCDUploadTemp(images, name, file)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Failed to stage uploaded file")
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	if err = transactMutableLCDUpload(
		images,
		name,
		tempPath,
		savePath,
		spec.format,
		defaultLCDUploadTransactionOps(),
	); err != nil {
		logger.Log(logger.Fields{"error": err, "location": savePath}).Error("Failed to activate LCD upload")
		http.Error(w, "Failed to activate image", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":  1,
		"message": "Image uploaded successfully",
		"name":    name,
		"format":  spec.format,
	})
}

type lcdUploadTransactionOps struct {
	rename         func(string, string) error
	restore        func(string, string) error
	removeAll      func(string) error
	beforeActivate func() error
}

type lcdFileSnapshot struct {
	path          string
	mode          os.FileMode
	data          []byte
	symlinkTarget string
}

type lcdLiveStateSnapshot struct {
	imageData       []ImageData
	animationWasNil bool
	animationImages map[string][]AnimationFrames
}

func defaultLCDUploadTransactionOps() lcdUploadTransactionOps {
	return lcdUploadTransactionOps{
		rename:    os.Rename,
		restore:   os.Rename,
		removeAll: os.RemoveAll,
		beforeActivate: func() error {
			return nil
		},
	}
}

func writeMutableLCDUploadTemp(root, baseName string, source io.Reader) (tempPath string, err error) {
	rootInfo, err := os.Lstat(root)
	if err != nil {
		return "", fmt.Errorf("inspect mutable LCD root: %w", err)
	}
	if rootInfo.Mode()&os.ModeSymlink != 0 || !rootInfo.IsDir() {
		return "", fmt.Errorf("mutable LCD root is not a real directory: %s", root)
	}

	tempFile, err := os.CreateTemp(root, "."+baseName+"-upload-*")
	if err != nil {
		return "", fmt.Errorf("create staged LCD upload: %w", err)
	}
	tempPath = tempFile.Name()
	defer func() {
		if err != nil {
			_ = tempFile.Close()
			if removeErr := os.Remove(tempPath); removeErr != nil && !os.IsNotExist(removeErr) {
				logger.Log(logger.Fields{"error": removeErr, "location": tempPath}).Error("Failed to remove staged LCD upload")
			}
		}
	}()

	if err = tempFile.Chmod(0o600); err != nil {
		return "", fmt.Errorf("secure staged LCD upload: %w", err)
	}
	if _, err = io.Copy(tempFile, source); err != nil {
		return "", fmt.Errorf("write staged LCD upload: %w", err)
	}
	if err = tempFile.Sync(); err != nil {
		return "", fmt.Errorf("sync staged LCD upload: %w", err)
	}
	if err = tempFile.Close(); err != nil {
		return "", fmt.Errorf("close staged LCD upload: %w", err)
	}
	return tempPath, nil
}

func transactMutableLCDUpload(
	root, baseName, tempPath, destination string,
	format uint8,
	ops lcdUploadTransactionOps,
) (returnErr error) {
	uploadMutex.Lock()
	defer uploadMutex.Unlock()

	ops = normalizeLCDUploadTransactionOps(ops)

	cleanRoot := filepath.Clean(root)
	cleanTempPath := filepath.Clean(tempPath)
	cleanDestination := filepath.Clean(destination)
	if !strings.HasPrefix(filepath.Base(cleanTempPath), "."+baseName+"-upload-") {
		return fmt.Errorf("invalid staged LCD upload %q", filepath.Base(cleanTempPath))
	}
	defer func() {
		if removeErr := os.Remove(cleanTempPath); removeErr != nil && !os.IsNotExist(removeErr) {
			if returnErr == nil {
				returnErr = fmt.Errorf("remove staged LCD upload: %w", removeErr)
			} else {
				returnErr = fmt.Errorf("%w (remove staged LCD upload: %v)", returnErr, removeErr)
			}
		}
	}()

	if filepath.Dir(cleanTempPath) != cleanRoot || filepath.Dir(cleanDestination) != cleanRoot {
		return fmt.Errorf("LCD upload transaction escapes mutable root %q", cleanRoot)
	}
	destinationName := filepath.Base(cleanDestination)
	if strings.TrimSuffix(destinationName, filepath.Ext(destinationName)) != baseName ||
		!supportedLCDImageExtension(filepath.Ext(destinationName)) {
		return fmt.Errorf("invalid LCD upload destination %q", destinationName)
	}

	rootInfo, err := os.Lstat(cleanRoot)
	if err != nil {
		return fmt.Errorf("inspect mutable LCD root: %w", err)
	}
	if rootInfo.Mode()&os.ModeSymlink != 0 || !rootInfo.IsDir() {
		return fmt.Errorf("mutable LCD root is not a real directory: %s", cleanRoot)
	}
	tempInfo, err := os.Lstat(cleanTempPath)
	if err != nil {
		return fmt.Errorf("inspect staged LCD upload: %w", err)
	}
	if !tempInfo.Mode().IsRegular() {
		return fmt.Errorf("staged LCD upload is not a regular file: %s", cleanTempPath)
	}

	imageData, err := decodeLCDImage(cleanTempPath, baseName, format)
	if err != nil {
		_ = os.Remove(cleanTempPath)
		return fmt.Errorf("validate staged LCD upload: %w", err)
	}

	var animationFrames []AnimationFrames
	if format == ImageFormatGif {
		mutex.Lock()
		animationFrames, _ = buildAnimationFramesFrom(imageData)
		mutex.Unlock()
		if animationFrames == nil {
			_ = os.Remove(cleanTempPath)
			return fmt.Errorf("build LCD animation %q", baseName)
		}
	}

	snapshots, err := inspectMutableLCDFiles(cleanRoot, baseName)
	if err != nil {
		_ = os.Remove(cleanTempPath)
		return err
	}

	rollbackDirectory, err := os.MkdirTemp(cleanRoot, "."+baseName+"-rollback-*")
	if err != nil {
		_ = os.Remove(cleanTempPath)
		return fmt.Errorf("create LCD upload rollback directory: %w", err)
	}
	if err = os.Chmod(rollbackDirectory, 0o700); err != nil {
		_ = os.Remove(cleanTempPath)
		_ = os.RemoveAll(rollbackDirectory)
		return fmt.Errorf("secure LCD upload rollback directory: %w", err)
	}

	newDestinationInstalled := false
	liveStateInstalled := false
	var liveSnapshot lcdLiveStateSnapshot
	rollback := func(originalErr error) error {
		if liveStateInstalled {
			mutex.Lock()
			restoreLCDLiveStateLocked(liveSnapshot)
			mutex.Unlock()
		}
		if rollbackErr := rollbackMutableLCDFiles(
			cleanDestination,
			cleanTempPath,
			rollbackDirectory,
			snapshots,
			newDestinationInstalled,
			ops.restore,
		); rollbackErr != nil {
			logger.Log(logger.Fields{
				"error":         rollbackErr,
				"originalError": originalErr,
				"location":      cleanDestination,
			}).Error("Failed to completely roll back LCD upload")
			return fmt.Errorf("%w (LCD upload rollback failed: %v)", originalErr, rollbackErr)
		}
		return originalErr
	}

	if hasLCDSnapshot(snapshots, cleanDestination) {
		if err = ops.rename(cleanDestination, filepath.Join(rollbackDirectory, destinationName)); err != nil {
			return rollback(fmt.Errorf("preserve existing LCD destination: %w", err))
		}
	}
	if err = ops.rename(cleanTempPath, cleanDestination); err != nil {
		return rollback(fmt.Errorf("install staged LCD upload: %w", err))
	}
	newDestinationInstalled = true

	if err = ops.beforeActivate(); err != nil {
		return rollback(fmt.Errorf("activate LCD upload: %w", err))
	}

	mutex.Lock()
	liveSnapshot = snapshotLCDLiveStateLocked()
	installLCDLiveStateLocked(imageData, animationFrames, format == ImageFormatGif)
	mutex.Unlock()
	liveStateInstalled = true

	for _, snapshot := range snapshots {
		if snapshot.path == cleanDestination {
			continue
		}
		if err = ops.rename(snapshot.path, filepath.Join(rollbackDirectory, filepath.Base(snapshot.path))); err != nil {
			return rollback(fmt.Errorf("preserve obsolete LCD sibling %q: %w", filepath.Base(snapshot.path), err))
		}
	}

	if err = ops.removeAll(rollbackDirectory); err != nil {
		return rollback(fmt.Errorf("remove obsolete LCD media: %w", err))
	}
	return nil
}

func normalizeLCDUploadTransactionOps(ops lcdUploadTransactionOps) lcdUploadTransactionOps {
	defaults := defaultLCDUploadTransactionOps()
	if ops.rename == nil {
		ops.rename = defaults.rename
	}
	if ops.restore == nil {
		ops.restore = defaults.restore
	}
	if ops.removeAll == nil {
		ops.removeAll = defaults.removeAll
	}
	if ops.beforeActivate == nil {
		ops.beforeActivate = defaults.beforeActivate
	}
	return ops
}

func inspectMutableLCDFiles(root, baseName string) ([]lcdFileSnapshot, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read mutable LCD root: %w", err)
	}

	snapshots := make([]lcdFileSnapshot, 0, len(entries))
	for _, entry := range entries {
		entryName := entry.Name()
		if !supportedLCDImageExtension(filepath.Ext(entryName)) ||
			strings.TrimSuffix(entryName, filepath.Ext(entryName)) != baseName {
			continue
		}
		entryPath := filepath.Join(root, entryName)
		info, statErr := os.Lstat(entryPath)
		if statErr != nil {
			return nil, fmt.Errorf("inspect LCD media %q: %w", entryName, statErr)
		}

		snapshot := lcdFileSnapshot{path: entryPath, mode: info.Mode()}
		switch {
		case info.Mode().IsRegular():
			snapshot.data, err = os.ReadFile(entryPath)
			if err != nil {
				return nil, fmt.Errorf("snapshot LCD media %q: %w", entryName, err)
			}
		case info.Mode()&os.ModeSymlink != 0:
			snapshot.symlinkTarget, err = os.Readlink(entryPath)
			if err != nil {
				return nil, fmt.Errorf("snapshot LCD media symlink %q: %w", entryName, err)
			}
		default:
			return nil, fmt.Errorf("same-name LCD sibling is not a regular file or symlink: %s", entryName)
		}
		snapshots = append(snapshots, snapshot)
	}
	return snapshots, nil
}

func hasLCDSnapshot(snapshots []lcdFileSnapshot, path string) bool {
	for _, snapshot := range snapshots {
		if snapshot.path == path {
			return true
		}
	}
	return false
}

func snapshotLCDLiveStateLocked() lcdLiveStateSnapshot {
	snapshot := lcdLiveStateSnapshot{
		imageData:       slices.Clone(lcd.ImageData),
		animationWasNil: animation == nil,
	}
	if animation != nil && animation.Images != nil {
		snapshot.animationImages = make(map[string][]AnimationFrames, len(animation.Images))
		for name, frames := range animation.Images {
			snapshot.animationImages[name] = frames
		}
	}
	return snapshot
}

func installLCDLiveStateLocked(imageData ImageData, animationFrames []AnimationFrames, animated bool) {
	replaced := false
	for index := range lcd.ImageData {
		if lcd.ImageData[index].Name == imageData.Name {
			lcd.ImageData[index] = imageData
			replaced = true
			break
		}
	}
	if !replaced {
		lcd.ImageData = append(lcd.ImageData, imageData)
	}

	if animation == nil {
		animation = new(Animation)
	}
	if animation.Images == nil {
		animation.Images = make(map[string][]AnimationFrames)
	}
	delete(animation.Images, imageData.Name)
	if animated {
		animation.Images[imageData.Name] = animationFrames
	}
}

func restoreLCDLiveStateLocked(snapshot lcdLiveStateSnapshot) {
	lcd.ImageData = snapshot.imageData
	if snapshot.animationWasNil {
		animation = nil
		return
	}
	if animation == nil {
		animation = new(Animation)
	}
	animation.Images = snapshot.animationImages
}

func rollbackMutableLCDFiles(
	destination, tempPath, rollbackDirectory string,
	snapshots []lcdFileSnapshot,
	newDestinationInstalled bool,
	restore func(string, string) error,
) error {
	var rollbackErrors []string
	if newDestinationInstalled {
		if err := os.Remove(destination); err != nil && !os.IsNotExist(err) {
			rollbackErrors = append(rollbackErrors, fmt.Sprintf("remove attempted destination: %v", err))
		}
	}

	for _, snapshot := range snapshots {
		if _, err := os.Lstat(snapshot.path); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			rollbackErrors = append(rollbackErrors, fmt.Sprintf("inspect %s: %v", snapshot.path, err))
			continue
		}

		backupPath := filepath.Join(rollbackDirectory, filepath.Base(snapshot.path))
		if _, err := os.Lstat(backupPath); err == nil {
			if renameErr := restore(backupPath, snapshot.path); renameErr != nil {
				rollbackErrors = append(rollbackErrors, fmt.Sprintf("restore %s: %v", snapshot.path, renameErr))
				if snapshotErr := restoreLCDFileSnapshot(snapshot); snapshotErr != nil {
					rollbackErrors = append(rollbackErrors, snapshotErr.Error())
				}
			}
			continue
		} else if !os.IsNotExist(err) {
			rollbackErrors = append(rollbackErrors, fmt.Sprintf("inspect rollback file %s: %v", backupPath, err))
			continue
		}

		if err := restoreLCDFileSnapshot(snapshot); err != nil {
			rollbackErrors = append(rollbackErrors, err.Error())
		}
	}

	if err := os.Remove(tempPath); err != nil && !os.IsNotExist(err) {
		rollbackErrors = append(rollbackErrors, fmt.Sprintf("remove staged upload: %v", err))
	}
	if err := os.RemoveAll(rollbackDirectory); err != nil {
		rollbackErrors = append(rollbackErrors, fmt.Sprintf("remove rollback directory: %v", err))
	}
	if len(rollbackErrors) > 0 {
		return fmt.Errorf("%s", strings.Join(rollbackErrors, "; "))
	}
	return nil
}

func restoreLCDFileSnapshot(snapshot lcdFileSnapshot) error {
	if snapshot.mode&os.ModeSymlink != 0 {
		if err := os.Symlink(snapshot.symlinkTarget, snapshot.path); err != nil {
			return fmt.Errorf("restore LCD symlink %s: %w", snapshot.path, err)
		}
		return nil
	}

	file, err := os.OpenFile(snapshot.path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, snapshot.mode.Perm())
	if err != nil {
		return fmt.Errorf("recreate LCD media %s: %w", snapshot.path, err)
	}
	if _, err = file.Write(snapshot.data); err != nil {
		_ = file.Close()
		return fmt.Errorf("restore LCD media %s: %w", snapshot.path, err)
	}
	if err = file.Sync(); err != nil {
		_ = file.Close()
		return fmt.Errorf("sync restored LCD media %s: %w", snapshot.path, err)
	}
	if err = file.Close(); err != nil {
		return fmt.Errorf("close restored LCD media %s: %w", snapshot.path, err)
	}
	if err = os.Chmod(snapshot.path, snapshot.mode.Perm()); err != nil {
		return fmt.Errorf("restore LCD media mode %s: %w", snapshot.path, err)
	}
	return nil
}

func cleanupMutableLCDSiblings(root, baseName, keepPath string) error {
	cleanRoot := filepath.Clean(root)
	cleanKeepPath := filepath.Clean(keepPath)
	if filepath.Dir(cleanKeepPath) != cleanRoot {
		return fmt.Errorf("LCD upload destination %q is outside mutable root %q", cleanKeepPath, cleanRoot)
	}
	keepName := filepath.Base(cleanKeepPath)
	keepExtension := strings.ToLower(filepath.Ext(keepName))
	if strings.TrimSuffix(keepName, filepath.Ext(keepName)) != baseName ||
		!supportedLCDImageExtension(keepExtension) {
		return fmt.Errorf("invalid LCD upload destination %q", keepName)
	}

	rootInfo, err := os.Lstat(cleanRoot)
	if err != nil {
		return fmt.Errorf("inspect mutable LCD root: %w", err)
	}
	if rootInfo.Mode()&os.ModeSymlink != 0 || !rootInfo.IsDir() {
		return fmt.Errorf("mutable LCD root is not a real directory: %s", cleanRoot)
	}
	keepInfo, err := os.Lstat(cleanKeepPath)
	if err != nil {
		return fmt.Errorf("inspect uploaded LCD image: %w", err)
	}
	if !keepInfo.Mode().IsRegular() {
		return fmt.Errorf("uploaded LCD image is not a regular file: %s", cleanKeepPath)
	}

	entries, err := os.ReadDir(cleanRoot)
	if err != nil {
		return fmt.Errorf("read mutable LCD root: %w", err)
	}
	for _, entry := range entries {
		entryName := entry.Name()
		entryExtension := strings.ToLower(filepath.Ext(entryName))
		if !supportedLCDImageExtension(entryExtension) ||
			strings.TrimSuffix(entryName, filepath.Ext(entryName)) != baseName {
			continue
		}
		entryPath := filepath.Join(cleanRoot, entryName)
		if filepath.Clean(entryPath) == cleanKeepPath {
			continue
		}
		if entry.Type()&os.ModeSymlink == 0 && entry.IsDir() {
			return fmt.Errorf("same-name LCD sibling is a directory: %s", entryName)
		}
		if err = os.Remove(entryPath); err != nil {
			return fmt.Errorf("remove obsolete LCD sibling %q: %w", entryName, err)
		}
	}
	return nil
}

func supportedLCDImageExtension(extension string) bool {
	switch strings.ToLower(extension) {
	case ".gif", ".jpg", ".jpeg", ".webp", ".bmp":
		return true
	default:
		return false
	}
}

func lcdUploadPath(root, filename string) (string, error) {
	if filename == "" || filepath.Base(filename) != filename {
		return "", fmt.Errorf("invalid LCD upload filename %q", filename)
	}
	destination := filepath.Join(root, filename)
	relative, err := filepath.Rel(root, destination)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("LCD upload escapes mutable media root")
	}
	return destination, nil
}

// GetCustomLcdProfiles will return list of LCD profiles currently supported by the product. This list is defined
// manually and its updated how new modes are created.
func GetCustomLcdProfiles() map[uint8]interface{} {
	profiles := make(map[uint8]interface{})
	profiles[DisplayArc] = GetArc()
	profiles[DisplayDoubleArc] = GetDoubleArc()
	profiles[DisplayAnimation] = GetAnimation()
	return profiles
}

// GenerateDoubleArcScreenImage handles generation of double arc screen image
func GenerateDoubleArcScreenImage(values []float32) []byte {
	arcImage := image.NewRGBA(image.Rect(0, 0, imgWidth, imgHeight))
	bg := generateColor(doubleRrc.Background)
	draw.Draw(arcImage, arcImage.Bounds(), &image.Uniform{C: bg}, image.Point{}, draw.Src)

	c := freetype.NewContext()
	c.SetDPI(72)
	c.SetFont(lcd.font)
	c.SetClip(arcImage.Bounds())
	c.SetDst(arcImage)
	c.SetSrc(image.NewUniform(color.RGBA{R: 255, G: 255, B: 253, A: 255}))

	// Common radius math
	outerRadius := float64(imgWidth)/2 - doubleRrc.Margin
	innerRadius := outerRadius - doubleRrc.Thickness
	centerY := float64(imgHeight) / 2

	// Border
	borderColor := generateColor(doubleRrc.BorderColor)

	// Left arc
	leftArc := doubleRrc.Arcs[0]
	leftColStart := generateColor(leftArc.StartColor)
	leftColEnd := generateColor(leftArc.EndColor)
	leftCenterX := doubleRrc.Margin + outerRadius
	leftMax := sensorMaximumValue(leftArc.Sensor)
	leftValue := values[leftArc.Sensor]
	if leftValue > float32(leftMax) {
		leftValue = float32(leftMax)
	}

	leftArcStart := math.Pi/2 + doubleRrc.GapRadians/2
	leftArcEnd := leftArcStart + float64(leftValue)/float64(leftMax)*(math.Pi-doubleRrc.GapRadians)
	leftBorderEnd := leftArcStart + float64(leftMax)/float64(leftMax)*(math.Pi-doubleRrc.GapRadians)

	// Border
	drawArcOutline(arcImage, leftCenterX, centerY, innerRadius, outerRadius, leftArcStart, leftBorderEnd, borderColor)

	// Arc
	drawSmoothArcGradient(arcImage, leftCenterX, centerY, innerRadius, outerRadius, leftArcStart, leftArcEnd, leftColStart, leftColEnd)

	if isSensorTemperature(leftArc.Sensor) {
		v := dashboard.GetDashboard().TemperatureToString(leftValue)
		x, y := calculateStringXY(100, v)
		drawColorString(x, y-80, 100, v, arcImage, leftArc.TextColor)
	} else {
		v := fmt.Sprintf("%.1f %s", leftValue, "%")
		x, y := calculateStringXY(100, v)
		drawColorString(x, y-80, 100, v, arcImage, leftArc.TextColor)
	}

	separator := "-------------------------------------------"
	x, y := calculateStringXY(20, separator)
	drawColorString(x, y, 20, separator, arcImage, doubleRrc.SeparatorColor)

	// Right Arc
	rightArc := doubleRrc.Arcs[1]
	rightColStart := generateColor(rightArc.EndColor) // Reversed
	rightColEnd := generateColor(rightArc.StartColor) // Reversed
	rightCenterX := float64(imgWidth) - doubleRrc.Margin - outerRadius
	rightMax := sensorMaximumValue(rightArc.Sensor)
	rightValue := values[rightArc.Sensor]
	if rightValue > float32(rightMax) {
		rightValue = float32(rightMax)
	}
	rightArcEnd := math.Pi/2 - doubleRrc.GapRadians/2
	rightArcStart := rightArcEnd - float64(rightValue)/float64(rightMax)*(math.Pi-doubleRrc.GapRadians)
	rightBorderStart := rightArcEnd - float64(rightMax)/float64(rightMax)*(math.Pi-doubleRrc.GapRadians)

	// Border
	drawArcOutline(arcImage, rightCenterX, centerY, innerRadius, outerRadius, rightBorderStart, rightArcEnd, borderColor)

	// Arc
	drawSmoothArcGradient(arcImage, rightCenterX, centerY, innerRadius, outerRadius, rightArcStart, rightArcEnd, rightColStart, rightColEnd)

	// Text
	if isSensorTemperature(rightArc.Sensor) {
		v := dashboard.GetDashboard().TemperatureToString(rightValue)
		x, y = calculateStringXY(100, v)
		drawColorString(x, y+80, 100, v, arcImage, rightArc.TextColor)
	} else {
		v := fmt.Sprintf("%.1f %s", rightValue, "%")
		x, y = calculateStringXY(100, v)
		drawColorString(x, y+80, 100, v, arcImage, rightArc.TextColor)
	}

	var b bytes.Buffer
	err := jpeg.Encode(&b, arcImage, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to encode LCD image")
		return nil
	}
	return b.Bytes()
}

// GenerateAnimationScreenImage handles generation of animation screen image
func GenerateAnimationScreenImage(values []float32) []Frames {
	mutex.Lock()
	background := animation.Background
	val, ok := animation.Images[background]
	sensors := animation.Sensors
	separatorColor := animation.SeparatorColor
	margin := int(animation.Margin)
	mutex.Unlock()

	if !ok {
		return nil
	}

	z := 0
	for _, sensor := range sensors {
		if sensor.Enabled {
			z++
		}
	}

	jpegOptions := jpeg.Options{Quality: 90}

	imageBuffer := make([]Frames, len(val))
	var wg sync.WaitGroup
	wg.Add(len(val))
	sem := make(chan struct{}, animation.Workers)

	for i := 0; i < len(val); i++ {
		sem <- struct{}{}

		i := i
		canvasSource := val[i].Canvas
		canvasRGBA := val[i].RGBA
		delay := val[i].Delay

		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			canvas := canvasRGBA
			copy(canvas.Pix, canvasSource.Pix)

			total := z * 125
			paddingStart := -total / 2
			paddingStart += margin
			padding := paddingStart

			for m := 0; m < len(sensors); m++ {
				sensor := sensors[m]
				if sensor.Enabled {
					sensorMax := sensorMaximumValue(sensor.Sensor)
					sensorValue := values[sensor.Sensor]
					if sensorValue > float32(sensorMax) {
						sensorValue = float32(sensorMax)
					}
					if isSensorTemperature(sensor.Sensor) {
						v := dashboard.GetDashboard().TemperatureToString(sensorValue)
						x, y := calculateStringXY(80, v)
						drawColorString(x, y+padding, 80, v, canvas, sensor.TextColor)
					} else {
						v := fmt.Sprintf("%.1f %%", sensorValue)
						x, y := calculateStringXY(80, v)
						drawColorString(x, y+padding, 80, v, canvas, sensor.TextColor)
					}

					sensorText := sensorTextCache[sensor.Sensor]
					x, y := calculateStringXY(40, sensorText)
					drawColorString(x, y+padding+55, 40, sensorText, canvas, sensor.TextColor)

					if m != len(sensors)-1 {
						separator := "-------------------------------------------"
						x, y = calculateStringXY(20, separator)
						drawColorString(x, y+padding+88, 20, separator, canvas, separatorColor)
					}
					padding += 125
				}
			}

			var buf bytes.Buffer
			err := jpeg.Encode(&buf, canvas, &jpegOptions)
			if err == nil {
				imageBuffer[i] = Frames{
					Buffer: buf.Bytes(),
					Delay:  delay,
				}
			}
		}()
	}

	wg.Wait()
	return imageBuffer
}

// GenerateArcScreenImage handles generation or arc screen image
func GenerateArcScreenImage(arcType, sensor, value int) []byte {
	mutex.Lock()
	defer mutex.Unlock()

	// Prevent over 100
	if (arcType == 0 || arcType == 1) && value > 100 {
		value = 100
	}

	// Prevent over 60
	if arcType == 2 && value > 60 {
		value = 60
	}

	bg := generateColor(arc.Background)
	arcStartColor := generateColor(arc.StartColor)
	arcEndColor := generateColor(arc.EndColor)
	arcThickness := arc.Thickness
	maxValue := sensorMaximumValue(arc.Sensor)
	img := image.NewRGBA(image.Rect(0, 0, imgWidth, imgHeight))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: bg}, image.Point{}, draw.Src)
	centerX, centerY := float64(imgWidth)/2, float64(imgHeight)/2
	outerRadius := float64(imgWidth)/2 - arc.Margin
	innerRadius := outerRadius - arcThickness
	arcStart := math.Pi/2 + arc.GapRadians/2
	arcEnd := arcStart + float64(value)/float64(maxValue)*((math.Pi*2)-arc.GapRadians)

	// Border
	borderAngle := startAngle + 1*2*math.Pi
	borderColor := generateColor(arc.BorderColor)

	// Draw the arc
	drawArcOutline(img, centerX, centerY, innerRadius, outerRadius, startAngle, borderAngle, borderColor)
	if value > 0 {
		drawSmoothArcGradient(img, centerX, centerY, innerRadius, outerRadius, arcStart, arcEnd, arcStartColor, arcEndColor)
	}
	c := freetype.NewContext()
	c.SetDPI(72)
	c.SetFont(lcd.font)
	c.SetClip(img.Bounds())
	c.SetDst(img)
	c.SetSrc(image.NewUniform(color.RGBA{R: 255, G: 255, B: 253, A: 255}))

	// Text
	if isSensorTemperature(uint8(sensor)) {
		// Value
		v := dashboard.GetDashboard().Temperature(float32(value))
		x, y := calculateStringXY(250, v[0])
		drawColorString(x, y, 250, v[0], img, arc.TextColor)

		// Unit
		unit := fmt.Sprintf("[ %s ]", v[1])
		x, y = calculateStringXY(40, unit)
		drawColorString(x, y+120, 40, unit, img, arc.TextColor)
	} else {
		// Value
		x, y := calculateStringXY(280, strconv.Itoa(value))
		drawColorString(x, y, 280, strconv.Itoa(value), img, arc.TextColor)

		// Unit
		x, y = calculateStringXY(40, "[ % ]")
		drawColorString(x, y+120, 40, "[ % ]", img, arc.TextColor)
	}

	var b bytes.Buffer
	err := jpeg.Encode(&b, img, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to encode LCD image")
		return nil
	}
	return b.Bytes()
}

// GenerateScreenImage will generate LCD screen image with given value
func GenerateScreenImage(imageType uint8, value, value1, value2, value3 int) []byte {
	mutex.Lock()
	defer mutex.Unlock()

	rgba := image.NewRGBA(lcd.image.Bounds())
	draw.Draw(rgba, rgba.Bounds(), lcd.image, image.Point{}, draw.Src)

	c := freetype.NewContext()
	c.SetDPI(72)
	c.SetFont(lcd.font)
	c.SetClip(rgba.Bounds())
	c.SetDst(rgba)
	c.SetSrc(image.NewUniform(color.RGBA{R: 255, G: 255, B: 253, A: 255}))

	switch imageType {
	case DisplayLiquid:
		{
			x, y := calculateStringXY(40, "LIQUID TEMP")
			drawString(x, y-120, 40, "LIQUID TEMP", rgba)

			x, y = calculateStringXY(40, "[ °C ]")
			drawString(x, y+120, 40, "[ °C ]", rgba)

			x, y = calculateStringXY(240, strconv.Itoa(value))
			drawString(x, y, 240, strconv.Itoa(value), rgba)
		}
	case DisplayGPU:
		{
			x, y := calculateStringXY(40, "GPU TEMP")
			drawString(x, y-120, 40, "GPU TEMP", rgba)

			x, y = calculateStringXY(40, "[ °C ]")
			drawString(x, y+120, 40, "[ °C ]", rgba)

			x, y = calculateStringXY(240, strconv.Itoa(value))
			drawString(x, y, 240, strconv.Itoa(value), rgba)
		}
	case DisplayCPU:
		{
			x, y := calculateStringXY(40, "CPU TEMP")
			drawString(x, y-120, 40, "CPU TEMP", rgba)

			x, y = calculateStringXY(40, "[ °C ]")
			drawString(x, y+120, 40, "[ °C ]", rgba)

			x, y = calculateStringXY(240, strconv.Itoa(value))
			drawString(x, y, 240, strconv.Itoa(value), rgba)
		}
	case DisplayPump:
		{
			x, y := calculateStringXY(40, "PUMP SPEED")
			drawString(x, y-120, 40, "PUMP SPEED", rgba)

			x, y = calculateStringXY(40, "[ RPM ]")
			drawString(x, y+120, 40, "[ RPM ]", rgba)

			x, y = calculateStringXY(200, strconv.Itoa(value))
			drawString(x, y, 200, strconv.Itoa(value), rgba)
		}
	case DisplayAllInOne:
		{
			x, y := calculateStringXY(40, "LIQUID")
			drawString(x-80, y-110, 40, "LIQUID", rgba)

			x, y = calculateStringXY(40, "CPU")
			drawString(x+80, y-110, 40, "CPU", rgba)

			x, y = calculateStringXY(40, "PUMP")
			drawString(x, y+130, 40, "PUMP", rgba)

			x, y = calculateStringXY(100, strconv.Itoa(value))
			drawString(x-80, y-40, 100, strconv.Itoa(value), rgba)

			x, y = calculateStringXY(100, strconv.Itoa(value1))
			drawString(x+80, y-40, 100, strconv.Itoa(value1), rgba)

			x, y = calculateStringXY(100, strconv.Itoa(value2))
			drawString(x, y+60, 100, strconv.Itoa(value2), rgba)
		}
	case DisplayLiquidCPU:
		{
			drawString(120+int(c.PointToFixed(24)>>6), 110+int(c.PointToFixed(24)>>6), 40, "LIQUID", rgba)
			drawString(280+int(c.PointToFixed(24)>>6), 110+int(c.PointToFixed(24)>>6), 40, "CPU", rgba)
			drawString(190+int(c.PointToFixed(24)>>6), 350+int(c.PointToFixed(24)>>6), 40, "[ °C ]", rgba)
			drawString(95+int(c.PointToFixed(24)>>6), 270+int(c.PointToFixed(24)>>6), 160, strconv.Itoa(value), rgba)
			drawString(250+int(c.PointToFixed(24)>>6), 270+int(c.PointToFixed(24)>>6), 160, strconv.Itoa(value1), rgba)
		}
	case DisplayCpuGpuTemp:
		{
			drawString(120+int(c.PointToFixed(24)>>6), 120+int(c.PointToFixed(24)>>6), 40, "CPU", rgba)
			drawString(270+int(c.PointToFixed(24)>>6), 120+int(c.PointToFixed(24)>>6), 40, "GPU", rgba)
			drawString(190+int(c.PointToFixed(24)>>6), 350+int(c.PointToFixed(24)>>6), 40, "[ °C ]", rgba)
			drawString(90+int(c.PointToFixed(24)>>6), 270+int(c.PointToFixed(24)>>6), 160, strconv.Itoa(value), rgba)
			drawString(240+int(c.PointToFixed(24)>>6), 270+int(c.PointToFixed(24)>>6), 160, strconv.Itoa(value1), rgba)
		}
	case DisplayCpuGpuLoadTemp:
		{
			drawString(130+int(c.PointToFixed(24)>>6), 140+int(c.PointToFixed(24)>>6), 40, "CPU", rgba)
			drawString(270+int(c.PointToFixed(24)>>6), 140+int(c.PointToFixed(24)>>6), 40, "GPU", rgba)
			drawString(190+int(c.PointToFixed(24)>>6), 90+int(c.PointToFixed(24)>>6), 40, "[ °C ]", rgba)
			drawString(190+int(c.PointToFixed(24)>>6), 350+int(c.PointToFixed(24)>>6), 40, "[ % ]", rgba)
			drawString(120+int(c.PointToFixed(24)>>6), 220+int(c.PointToFixed(24)>>6), 80, fmt.Sprintf("%02d", value), rgba)
			drawString(260+int(c.PointToFixed(24)>>6), 220+int(c.PointToFixed(24)>>6), 80, fmt.Sprintf("%02d", value1), rgba)
			drawString(120+int(c.PointToFixed(24)>>6), 290+int(c.PointToFixed(24)>>6), 80, fmt.Sprintf("%02d", value2), rgba)
			drawString(260+int(c.PointToFixed(24)>>6), 290+int(c.PointToFixed(24)>>6), 80, fmt.Sprintf("%02d", value3), rgba)
		}
	case DisplayCpuGpuLoad:
		{
			drawString(120+int(c.PointToFixed(24)>>6), 120+int(c.PointToFixed(24)>>6), 40, "CPU", rgba)
			drawString(270+int(c.PointToFixed(24)>>6), 120+int(c.PointToFixed(24)>>6), 40, "GPU", rgba)
			drawString(190+int(c.PointToFixed(24)>>6), 350+int(c.PointToFixed(24)>>6), 40, "[ % ]", rgba)

			reduce := 0
			bounds, _ := font.BoundString(basicfont.Face7x13, strconv.Itoa(value))
			textWidth := (bounds.Max.X - bounds.Min.X).Floor()
			if value == 100 {
				reduce = 30
			}
			x := 100 + textWidth - reduce
			drawString(x, 270+int(c.PointToFixed(24)>>6), 160, fmt.Sprintf("%02d", value), rgba)

			bounds, _ = font.BoundString(basicfont.Face7x13, strconv.Itoa(value1))
			textWidth = (bounds.Max.X - bounds.Min.X).Floor()
			if value == 100 {
				reduce = 30
			}
			x = 240 + textWidth + 15 - reduce
			drawString(x, 270+int(c.PointToFixed(24)>>6), 160, fmt.Sprintf("%02d", value1), rgba)
		}
	case DisplayTime:
		{
			x, y := calculateStringXY(70, common.GetDate())
			drawString(x, y-50, 70, common.GetDate(), rgba)

			x, y = calculateStringXY(130, common.GetTime())
			drawString(x, y+50, 130, common.GetTime(), rgba)
		}
	}

	// Buff it and return
	buffer := new(bytes.Buffer)
	err := jpeg.Encode(buffer, rgba, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to encode LCD image")
		return nil
	}
	return buffer.Bytes()
}

/*
// drawString will create a new string for image
func drawString(x, y int, fontSite float64, c *freetype.Context, text string) *freetype.Context {
	c.SetFontSize(fontSite)
	pt := freetype.Pt(x, y)
	_, err := c.DrawString(text, pt)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to generate LCD image")
		return nil
	}
	return c
}
*/

// drawString will create a new string for image
func drawString(x, y int, fontSite float64, text string, rgba *image.RGBA) {
	pt := freetype.Pt(x, y)
	opts := opentype.FaceOptions{Size: fontSite, DPI: 72, Hinting: 0}
	fontFace, err := opentype.NewFace(lcd.sfntFont, &opts)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to process font face")
		return
	}
	d := &font.Drawer{
		Dst:  rgba,
		Src:  image.NewUniform(color.RGBA{R: 255, G: 255, B: 255, A: 255}),
		Face: fontFace, // Use the built-in font
		Dot:  pt,
	}
	d.DrawString(text)
}

// drawColorString will create a new string for image with color ability
func drawColorString(x, y int, fontSite float64, text string, rgba *image.RGBA, textColor rgb.Color) {
	pt := freetype.Pt(x, y)
	opts := opentype.FaceOptions{Size: fontSite, DPI: 72, Hinting: 0}
	fontFace, err := opentype.NewFace(lcd.sfntFont, &opts)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to process font face")
		return
	}
	d := &font.Drawer{
		Dst: rgba,
		Src: image.NewUniform(
			color.RGBA{
				R: uint8(textColor.Red),
				G: uint8(textColor.Green),
				B: uint8(textColor.Blue),
				A: 255,
			},
		),
		Face: fontFace, // Use the built-in font
		Dot:  pt,
	}
	d.DrawString(text)
}

func loadImage(imagePath string, format uint8) bool {
	filename := filepath.Base(imagePath)
	fileName := strings.TrimSuffix(filename, filepath.Ext(filename))
	imageData, err := decodeLCDImage(imagePath, fileName, format)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": images, "image": imagePath}).Warn("Unable to load image")
		return false
	}

	mutex.Lock()
	defer mutex.Unlock()
	installLCDImageDataLocked(imageData)
	return true
}

func decodeLCDImage(imagePath, fileName string, format uint8) (ImageData, error) {
	file, err := os.Open(imagePath)
	if err != nil {
		return ImageData{}, fmt.Errorf("open image: %w", err)
	}

	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			logger.Log(logger.Fields{"error": closeErr, "location": images, "image": imagePath}).Warn("Unable to close image")
		}
	}()

	imageBuffer := make([]Frames, 1)
	var paletted []*image.Paletted

	switch format {
	case ImageFormatJpg: // JPG, JPEG
		{
			var src image.Image
			var buffer bytes.Buffer

			src, err = jpeg.Decode(file)
			if err != nil {
				return ImageData{}, fmt.Errorf("decode JPEG: %w", err)
			}

			resized := common.ResizeImage(src, imgWidth, imgHeight)
			err = jpeg.Encode(&buffer, resized, nil)
			if err != nil {
				return ImageData{}, fmt.Errorf("encode JPEG frame: %w", err)
			}
			imageBuffer[0] = Frames{
				Buffer: buffer.Bytes(),
				Delay:  0,
			}
		}
		break
	case ImageFormatBmp: // BMP
		{
			var src image.Image
			var buffer bytes.Buffer

			src, err = bmp.Decode(file)
			if err != nil {
				return ImageData{}, fmt.Errorf("decode BMP: %w", err)
			}

			resized := common.ResizeImage(src, imgWidth, imgHeight)
			err = jpeg.Encode(&buffer, resized, nil)
			if err != nil {
				return ImageData{}, fmt.Errorf("encode BMP frame: %w", err)
			}
			imageBuffer[0] = Frames{
				Buffer: buffer.Bytes(),
				Delay:  0,
			}
		}
		break
	case ImageFormatWebp: // WEBP static
		{
			var src image.Image
			var buffer bytes.Buffer

			src, err = webp.Decode(file)
			if err != nil {
				return ImageData{}, fmt.Errorf("decode WEBP: %w", err)
			}

			resized := common.ResizeImage(src, imgWidth, imgHeight)
			err = jpeg.Encode(&buffer, resized, nil)
			if err != nil {
				return ImageData{}, fmt.Errorf("encode WEBP frame: %w", err)
			}
			imageBuffer[0] = Frames{
				Buffer: buffer.Bytes(),
				Delay:  0,
			}
		}
		break
	case ImageFormatGif: // Gif
		{
			var src *gif.GIF
			src, err = gif.DecodeAll(file)
			if err != nil {
				return ImageData{}, fmt.Errorf("decode GIF animation: %w", err)
			}
			imageBuffer = make([]Frames, len(src.Image))
			paletted = common.ResizeGifImage(src, imgWidth, imgHeight)
			for i, frame := range paletted {
				var buffer bytes.Buffer
				err = jpeg.Encode(&buffer, frame, nil)
				if err != nil {
					return ImageData{}, fmt.Errorf("encode GIF frame %d: %w", i, err)
				}
				imageBuffer[i] = Frames{
					Buffer: buffer.Bytes(),
					Delay:  float64(src.Delay[i]) * 10,
				}
			}
		}
		break
	default:
		return ImageData{}, fmt.Errorf("unsupported LCD image format %d", format)
	}

	return ImageData{
		Name:           fileName,
		Frames:         len(imageBuffer),
		Buffer:         imageBuffer,
		PalettedFrames: paletted,
	}, nil
}

func installLCDImageDataLocked(imageData ImageData) {
	for index := range lcd.ImageData {
		if lcd.ImageData[index].Name == imageData.Name {
			lcd.ImageData[index] = imageData
			return
		}
	}
	lcd.ImageData = append(lcd.ImageData, imageData)
}

// loadLcdImages will load all LCD images
func loadLcdImages() {
	loadLcdImagesFrom(shippedImages)
	loadLcdImagesFrom(images)
}

func loadLcdImagesFrom(directory string) {
	files, err := os.ReadDir(directory)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": directory}).Error("Unable to read content of a folder")
		return
	}
	for _, fi := range files {
		imagePath := filepath.Join(directory, fi.Name())

		// Process filename
		filename := filepath.Base(imagePath)
		fileName := strings.TrimSuffix(filename, filepath.Ext(filename))
		if !common.AlphanumericRegex.MatchString(fileName) {
			logger.Log(logger.Fields{"error": err, "location": directory, "image": imagePath}).Warn("Image name can only have letters, numbers, - and _. Please rename your image")
			continue
		}

		switch strings.ToLower(filepath.Ext(imagePath)) {
		case ".jpg":
			{
				loadImage(imagePath, ImageFormatJpg)
			}
			break
		case ".jpeg":
			{
				loadImage(imagePath, ImageFormatJpg)
			}
			break
		case ".bmp":
			{
				loadImage(imagePath, ImageFormatBmp)
			}
			break
		case ".webp":
			{
				loadImage(imagePath, ImageFormatWebp)
			}
			break
		case ".gif":
			{
				loadImage(imagePath, ImageFormatGif)
			}
			break
		default:
			logger.Log(logger.Fields{"error": err, "location": directory, "image": imagePath}).Warn("Invalid image extension")
			continue
		}
	}
}

// checkForLcd will check for LCD presence
func checkForLcd() {
	lcdProductIds := []uint16{3150, 3139, 3129, 3123, 3159, 3157, 3138}
	enum := hid.EnumFunc(func(info *hid.DeviceInfo) error {
		if info.InterfaceNbr == 0 {
			if slices.Contains(lcdProductIds, info.ProductID) {
				lcdPresent = true
			}
		}
		return nil
	})

	// Enumerate all Corsair devices
	err := hid.Enumerate(vendorId, hid.ProductIDAny, enum)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": vendorId}).Error("Unable to enumerate LCD devices")
		return
	}
}

// loadLcdDevices will load all available LCD devices
func loadLcdDevices() {
	lcdProductIds := []uint16{3150, 3139}

	enum := hid.EnumFunc(func(info *hid.DeviceInfo) error {
		if info.InterfaceNbr == 0 {
			if slices.Contains(lcdProductIds, info.ProductID) {
				lcdDevices[info.SerialNbr] = info.ProductID
			}
		}
		return nil
	})

	// Enumerate all Corsair devices
	err := hid.Enumerate(vendorId, hid.ProductIDAny, enum)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": vendorId}).Error("Unable to enumerate LCD devices")
		return
	}

	for serial, productId := range lcdDevices {
		logger.Log(logger.Fields{"serial": serial, "vendorId": vendorId, "productId": productId}).Info("Processing LCD device")
		lcdPanel, e := hid.Open(vendorId, productId, serial)
		if e != nil {
			logger.Log(logger.Fields{"error": e, "vendorId": vendorId, "productId": productId}).Error("Unable to open LCD HID device")
			continue
		}

		product := ""
		switch productId {
		case 3150:
			product = "iCUE LINK AIO LCD"
		case 3139:
			product = "iCUE LINK XD5 LCD"
		}
		device := &Device{
			Lcd:       lcdPanel,
			ProductId: productId,
			Product:   product,
			Serial:    serial,
			AIO:       productId == 3150,
			VendorId:  vendorId,
		}
		lcd.Devices = append(lcd.Devices, *device)
	}
}
