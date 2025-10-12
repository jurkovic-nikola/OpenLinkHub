package serial

// Package: serial
// Package for low level communication with ttyUSB devices
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"golang.org/x/sys/unix"
	"os"
	"syscall"
	"time"
	"unsafe"
)

type Device struct {
	file *os.File
}

type Config struct {
	Name string
	Baud int
}

const (
	CS8    = 0x00000030
	CLOCAL = 0x00000800
	CREAD  = 0x00000100
	ICANON = 0x00000002
	ECHO   = 0x00000008
	IXON   = 0x00000400
	OPOST  = 0x00000001
	VMIN   = 6
	VTIME  = 5
)

var baudMap = map[int]uint32{
	50:      unix.B50,
	75:      unix.B75,
	110:     unix.B110,
	134:     unix.B134,
	150:     unix.B150,
	200:     unix.B200,
	300:     unix.B300,
	600:     unix.B600,
	1200:    unix.B1200,
	1800:    unix.B1800,
	2400:    unix.B2400,
	4800:    unix.B4800,
	9600:    unix.B9600,
	19200:   unix.B19200,
	38400:   unix.B38400,
	57600:   unix.B57600,
	115200:  unix.B115200,
	230400:  unix.B230400,
	460800:  unix.B460800,
	500000:  unix.B500000,
	576000:  unix.B576000,
	921600:  unix.B921600,
	1000000: unix.B1000000,
	1152000: unix.B1152000,
	1500000: unix.B1500000,
	2000000: unix.B2000000,
	2500000: unix.B2500000,
	3000000: unix.B3000000,
	3500000: unix.B3500000,
	4000000: unix.B4000000,
}

// Open opens the file handle
func Open(c *Config) (*Device, error) {
	fd, err := syscall.Open(c.Name, syscall.O_RDWR|syscall.O_NOCTTY|syscall.O_NONBLOCK, 0666)
	if err != nil {
		return nil, err
	}

	var term syscall.Termios
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), uintptr(syscall.TCGETS), uintptr(unsafe.Pointer(&term))); errno != 0 {
		err := syscall.Close(fd)
		if err != nil {
			return nil, err
		}
		return nil, errno
	}

	// Set baud rate
	speed, ok := baudMap[c.Baud]
	if !ok {
		speed = baudMap[9600]
	}

	term.Cflag = speed | CS8 | CLOCAL | CREAD
	term.Lflag &^= ICANON | ECHO
	term.Iflag &^= IXON
	term.Oflag &^= OPOST
	term.Cc[VMIN] = 1
	term.Cc[VTIME] = 0

	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), uintptr(syscall.TCSETS), uintptr(unsafe.Pointer(&term))); errno != 0 {
		err := syscall.Close(fd)
		if err != nil {
			return nil, err
		}
		return nil, errno
	}

	file := os.NewFile(uintptr(fd), c.Name)
	return &Device{file: file}, nil
}

// Close closes the file handle
func (d *Device) Close() error {
	return d.file.Close()
}

// ReadTimeout reads from the serial port with a timeout
func (d *Device) ReadTimeout(buf []byte, timeout time.Duration) (int, error) {
	if timeout > 0 {
		err := d.file.SetReadDeadline(time.Now().Add(timeout))
		if err != nil {
			return 0, err
		}
	} else {
		// No timeout
		_ = d.file.SetReadDeadline(time.Time{})
	}

	n, err := d.file.Read(buf)

	// Clear deadline after read
	_ = d.file.SetReadDeadline(time.Time{})
	return n, err
}

// Read reads from the serial port
func (d *Device) Read(buf []byte) (int, error) {
	return d.file.Read(buf)
}

// Write writes to the serial port
func (d *Device) Write(buf []byte) (int, error) {
	return d.file.Write(buf)
}
