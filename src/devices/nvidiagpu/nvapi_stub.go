//go:build !linux || !cgo

package nvidiagpu

import "fmt"

func detectNativeGPUs() ([]nativeGPU, error) {
	return nil, fmt.Errorf("native NVIDIA GPU RGB support requires Linux with cgo")
}

func setNativeZone(_ uintptr, _ int, _, _, _, _ uint8, _ bool) error {
	return fmt.Errorf("native NVIDIA GPU RGB support requires Linux with cgo")
}
