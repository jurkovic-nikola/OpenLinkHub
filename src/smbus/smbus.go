package smbus

// Package: smbus
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/logger"
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"
	"unsafe"
)

const (
	i2cSlave            = 0x0703
	i2cSmbus            = 0x0720
	i2cRead      uint8  = 1
	i2cWrite     uint8  = 0
	i2cByteData  uint32 = 2
	i2cWordData  uint32 = 3
	i2cBlockData uint32 = 5
	i2csmBusMax  uint32 = 32
)

type SmBus struct {
	Path string
}

type i2cCommand struct {
	mode    uint8
	command uint8
	length  uint32
	pointer unsafe.Pointer
}

type Connection struct {
	File *os.File
}

// GetSmBus will return selected SMBus
func GetSmBus() (*SmBus, error) {
	var devices []string
	dir := "/dev/"

	files, err := os.ReadDir(dir)
	if err != nil {
		logger.Log(logger.Fields{"dir": dir, "error": err}).Error("Unable to enumerate directory")
		return nil, err
	}

	for _, file := range files {
		if strings.HasPrefix(file.Name(), "i2c-") {
			devices = append(devices, file.Name())
		}
	}

	if len(devices) == 0 {
		logger.Log(logger.Fields{}).Warn("Found 0 i2c devices. Memory will not work.")
		return nil, errors.New("no devices")
	}

	for _, device := range devices {
		if device == config.GetConfig().MemorySmBus {
			dev := fmt.Sprintf("%s%s", dir, device)
			f, err := Open(dev)
			if err != nil {
				logger.Log(logger.Fields{"device": dev, "error": err}).Error("Unable to open i2c device")
				continue
			}

			err = f.File.Close()
			if err != nil {
				logger.Log(logger.Fields{"device": dev, "error": err}).Error("Unable to close i2c device")
			}
			return &SmBus{Path: dev}, nil
		}
	}
	return nil, errors.New("no devices")
}

// ioctl is a linux implementation of ioctl
func ioctl(fd, cmd, arg uintptr) (err error) {
	_, _, e1 := syscall.Syscall(syscall.SYS_IOCTL, fd, cmd, arg)
	if e1 != 0 {
		err = e1
	}
	return
}

// Open will open specified i2c device
func Open(device string) (*Connection, error) {
	f, err := os.OpenFile(device, os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	return &Connection{File: f}, nil
}

// WriteRegister will write byte to a register
func WriteRegister(f *os.File, address, register, value uint8) error {
	if err := ioctl(f.Fd(), i2cSlave, uintptr(address)); err != nil {
		return err
	}

	var cmd = i2cCommand{
		mode:    i2cWrite,
		command: register,
		length:  i2cByteData,
		pointer: unsafe.Pointer(&value),
	}
	ptr := unsafe.Pointer(&cmd)
	return ioctl(f.Fd(), i2cSmbus, uintptr(ptr))
}

// WriteBlockData will write a byte array to register
func WriteBlockData(f *os.File, address, register uint8, buf []byte) error {
	if len(buf) > int(i2csmBusMax) {
		return errors.New("buffer is too long for this type")
	}

	if err := ioctl(f.Fd(), i2cSlave, uintptr(address)); err != nil {
		return err
	}

	data := make([]byte, 1+len(buf), i2csmBusMax+2)
	data[0] = byte(len(buf))
	copy(data[1:], buf)

	cmd := i2cCommand{
		mode:    i2cWrite,
		command: register,
		length:  i2cBlockData,
		pointer: unsafe.Pointer(&data[0]),
	}
	ptr := unsafe.Pointer(&cmd)
	return ioctl(f.Fd(), i2cSmbus, uintptr(ptr))
}

// ReadRegister will read byte from a register
func ReadRegister(f *os.File, address, register uint8) (uint8, error) {
	if err := ioctl(f.Fd(), i2cSlave, uintptr(address)); err != nil {
		return 0, err
	}

	var v uint8
	var cmd = i2cCommand{
		mode:    i2cRead,
		command: register,
		length:  i2cByteData,
		pointer: unsafe.Pointer(&v),
	}
	ptr := unsafe.Pointer(&cmd)
	err := ioctl(f.Fd(), i2cSmbus, uintptr(ptr))
	return v, err
}

// ReadWord reads a 2-bytes word from a designated register.
func ReadWord(f *os.File, addr, reg uint8) (uint16, error) {
	if err := ioctl(f.Fd(), i2cSlave, uintptr(addr)); err != nil {
		return 0, err
	}
	var v uint16
	cmd := i2cCommand{
		mode:    i2cRead,
		command: reg,
		length:  i2cWordData,
		pointer: unsafe.Pointer(&v),
	}
	ptr := unsafe.Pointer(&cmd)
	err := ioctl(f.Fd(), i2cSmbus, uintptr(ptr))
	return v, err
}
