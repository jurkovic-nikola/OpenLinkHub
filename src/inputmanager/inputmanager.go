package inputmanager

// Package: inputmanager
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/logger"
	"errors"
	"os"
	"slices"
	"sort"
	"syscall"
	"unsafe"
)

type KeyAssignment struct {
	Name           string `json:"name"`
	Default        bool   `json:"default"`
	ActionType     uint8  `json:"actionType"`
	ActionCommand  uint16 `json:"actionCommand"`
	ActionHold     bool   `json:"actionHold"`
	ButtonIndex    int    `json:"buttonIndex"`
	IsMacro        bool   `json:"isMacro"`
	ModifierKey    uint8  `json:"modifierKey"`
	RetainOriginal bool   `json:"retainOriginal"`
	ToggleDelay    uint16 `json:"toggleDelay"`
	ProfileSwitch  bool   `json:"profileSwitch"`
	OnRelease      bool   `json:"onRelease"`
	IsTilt         bool   `json:"isTilt"`
	TiltToggle     bool   `json:"tiltToggle"`
	TiltIndex      int    `json:"tiltIndex"`
}

type InputAction struct {
	Name        string // Key name
	CommandCode uint16 // Key code
	Media       bool   // Key can control media playback
	Mouse       bool
	Controller  bool
	Scroll      bool
}

type uInputUserDev struct {
	Name      [80]byte
	ID        inputID
	FFEffects uint32
	AbsMax    [64]int32
	AbsMin    [64]int32
	AbsFuzz   [64]int32
	AbsFlat   [64]int32
}

type inputID struct {
	BusType uint16
	Vendor  uint16
	Product uint16
	Version uint16
}

const (
	UiDevCreate         = 0x5501
	UiDevDestroy        = 0x5502
	UiSetEvbit          = 0x40045564
	UiSetKeybit         = 0x40045565
	UiSetRelbit         = 0x40045566
	EvKey        uint16 = 1
	EvSyn        uint16 = 0
	EvRel        uint16 = 2
	RelX                = 0x00
	RelY                = 0x01
	RelWheel            = 0x08
	RelHWheel           = 0x06
)

const (
	vendorId                uint16 = 5840
	productId               uint16 = 5175
	None                    uint16 = 0
	VolumeUp                uint16 = 1
	VolumeDown              uint16 = 2
	VolumeMute              uint16 = 3
	MediaStop               uint16 = 4
	MediaPrev               uint16 = 5
	MediaPlayPause          uint16 = 6
	MediaNext               uint16 = 7
	Number1                 uint16 = 8
	Number2                 uint16 = 9
	Number3                 uint16 = 10
	Number4                 uint16 = 11
	Number5                 uint16 = 12
	Number6                 uint16 = 13
	Number7                 uint16 = 14
	Number8                 uint16 = 15
	Number9                 uint16 = 16
	Number0                 uint16 = 17
	KeyMinus                uint16 = 18
	KeyEqual                uint16 = 19
	KeyQ                    uint16 = 20
	KeyW                    uint16 = 21
	KeyE                    uint16 = 22
	KeyR                    uint16 = 23
	KeyT                    uint16 = 24
	KeyY                    uint16 = 25
	KeyU                    uint16 = 26
	KeyI                    uint16 = 27
	KeyO                    uint16 = 28
	KeyP                    uint16 = 29
	KeyA                    uint16 = 30
	KeyS                    uint16 = 31
	KeyD                    uint16 = 32
	KeyF                    uint16 = 33
	KeyG                    uint16 = 34
	KeyH                    uint16 = 35
	KeyJ                    uint16 = 36
	KeyK                    uint16 = 37
	KeyL                    uint16 = 38
	KeyZ                    uint16 = 39
	KeyX                    uint16 = 40
	KeyC                    uint16 = 41
	KeyV                    uint16 = 42
	KeyB                    uint16 = 43
	KeyN                    uint16 = 44
	KeyM                    uint16 = 45
	KeyF1                   uint16 = 46
	KeyF2                   uint16 = 47
	KeyF3                   uint16 = 48
	KeyF4                   uint16 = 49
	KeyF5                   uint16 = 50
	KeyF6                   uint16 = 51
	KeyF7                   uint16 = 52
	KeyF8                   uint16 = 53
	KeyF9                   uint16 = 54
	KeyF10                  uint16 = 55
	KeyF11                  uint16 = 56
	KeyF12                  uint16 = 57
	KeyBack                 uint16 = 58
	KeyTab                  uint16 = 59
	KeyEsc                  uint16 = 60
	KeyTilde                uint16 = 61
	KeyLeftSquare           uint16 = 62
	KeyRightSquare          uint16 = 63
	KeyBackslash            uint16 = 64
	KeyCapslock             uint16 = 65
	KeySemicolon            uint16 = 66
	KeySingleQuote          uint16 = 67
	KeyEnter                uint16 = 68
	KeyLeftShift            uint16 = 69
	KeyComma                uint16 = 70
	KeyDot                  uint16 = 71
	KeySlash                uint16 = 72
	KeyRightShift           uint16 = 73
	KeyUp                   uint16 = 74
	KeyLeftCtrl             uint16 = 75
	KeyWindowsKey           uint16 = 76
	KeyLeftAlt              uint16 = 77
	KeySpace                uint16 = 78
	KeyRightAlt             uint16 = 79
	KeyMenu                 uint16 = 80
	KeyRightCtrl            uint16 = 81
	KeyLeft                 uint16 = 82
	KeyDown                 uint16 = 83
	KeyRight                uint16 = 84
	KeyIns                  uint16 = 85
	KeyHome                 uint16 = 86
	KeyPgUp                 uint16 = 87
	KeyDel                  uint16 = 88
	KeyEnd                  uint16 = 89
	KeyPgDn                 uint16 = 90
	BtnLeft                 uint16 = 91
	BtnRight                uint16 = 92
	BtnMiddle               uint16 = 93
	BtnBack                 uint16 = 94
	BtnForward              uint16 = 95
	Kp1                     uint16 = 96
	Kp2                     uint16 = 97
	Kp3                     uint16 = 98
	Kp4                     uint16 = 99
	Kp5                     uint16 = 100
	Kp6                     uint16 = 101
	Kp7                     uint16 = 102
	Kp8                     uint16 = 103
	Kp9                     uint16 = 104
	Kp0                     uint16 = 105
	KpDot                   uint16 = 106
	KpPlus                  uint16 = 107
	KpMinus                 uint16 = 108
	KpMultiply              uint16 = 109
	KpDivide                uint16 = 110
	KpEnter                 uint16 = 111
	KeyF13                  uint16 = 112
	KeyF14                  uint16 = 113
	KeyF15                  uint16 = 114
	KeyF16                  uint16 = 115
	KeyF17                  uint16 = 116
	KeyF18                  uint16 = 117
	KeyF19                  uint16 = 118
	KeyF20                  uint16 = 119
	KeyF21                  uint16 = 120
	KeyF22                  uint16 = 121
	KeyF23                  uint16 = 122
	KeyF24                  uint16 = 123
	KeyScreenBrightnessDown uint16 = 124
	KeyScreenBrightnessUp   uint16 = 125
	KeyPrtSc                uint16 = 126
	KeyPause                uint16 = 127
	KeyControllerSouth      uint16 = 128
	KeyControllerEast       uint16 = 129
	KeyControllerNorth      uint16 = 130
	KeyControllerWest       uint16 = 131
	KeyControllerTL         uint16 = 132
	KeyControllerTR         uint16 = 133
	KeyControllerTL2        uint16 = 134
	KeyControllerTR2        uint16 = 135
	KeyControllerSelect     uint16 = 136
	KeyControllerStart      uint16 = 137
	KeyControllerMode       uint16 = 138
	KeyControllerThumbL     uint16 = 139
	KeyControllerThumbR     uint16 = 140
	KeyControllerDpadUp     uint16 = 141
	KeyControllerDpadDown   uint16 = 142
	KeyControllerDpadLeft   uint16 = 143
	KeyControllerDpadRight  uint16 = 144
)

var (
	evKey                   uint16 = 0x01
	evSyn                   uint16 = 0x00
	evAbs                   uint16 = 0x03
	keyVolumeUp             uint16 = 0x73
	keyVolumeDown           uint16 = 0x72
	keyVolumeMute           uint16 = 0x71
	keyMediaStop            uint16 = 0xA6
	keyMediaPrev            uint16 = 0xA5
	keyMediaPlay            uint16 = 0xA4
	keyMediaNext            uint16 = 0xA3
	keyNumber1              uint16 = 0x2
	keyNumber2              uint16 = 0x3
	keyNumber3              uint16 = 0x4
	keyNumber4              uint16 = 0x5
	keyNumber5              uint16 = 0x6
	keyNumber6              uint16 = 0x7
	keyNumber7              uint16 = 0x8
	keyNumber8              uint16 = 0x9
	keyNumber9              uint16 = 0xA
	keyNumber0              uint16 = 0xB
	keyEsc                  uint16 = 0x1
	keyF1                   uint16 = 0x3B
	keyF2                   uint16 = 0x3C
	keyF3                   uint16 = 0x3D
	keyF4                   uint16 = 0x3E
	keyF5                   uint16 = 0x3F
	keyF6                   uint16 = 0x40
	keyF7                   uint16 = 0x41
	keyF8                   uint16 = 0x42
	keyF9                   uint16 = 0x43
	keyF10                  uint16 = 0x44
	keyF11                  uint16 = 0x57
	keyF12                  uint16 = 0x58
	keyF13                  uint16 = 0xB7
	keyF14                  uint16 = 0xB8
	keyF15                  uint16 = 0xB9
	keyF16                  uint16 = 0xBA
	keyF17                  uint16 = 0xBB
	keyF18                  uint16 = 0xBC
	keyF19                  uint16 = 0xBD
	keyF20                  uint16 = 0xBE
	keyF21                  uint16 = 0xBF
	keyF22                  uint16 = 0xC0
	keyF23                  uint16 = 0xC1
	keyF24                  uint16 = 0xC2
	keyTilde                uint16 = 0x29
	keyMinus                uint16 = 0xC
	keyEqual                uint16 = 0xD
	keyBack                 uint16 = 0xE
	keyTab                  uint16 = 0xF
	keyQ                    uint16 = 0x10
	keyW                    uint16 = 0x11
	keyE                    uint16 = 0x12
	keyR                    uint16 = 0x13
	keyT                    uint16 = 0x14
	keyY                    uint16 = 0x15
	keyU                    uint16 = 0x16
	keyI                    uint16 = 0x17
	keyO                    uint16 = 0x18
	keyP                    uint16 = 0x19
	keyLeftSquare           uint16 = 0x1A
	keyRightSquare          uint16 = 0x1B
	keyBackslash            uint16 = 0x2B
	keyCapslock             uint16 = 0x3A
	keyA                    uint16 = 0x1E
	keyS                    uint16 = 0x1F
	keyD                    uint16 = 0x20
	keyF                    uint16 = 0x21
	keyG                    uint16 = 0x22
	keyH                    uint16 = 0x23
	keyJ                    uint16 = 0x24
	keyK                    uint16 = 0x25
	keyL                    uint16 = 0x26
	keySemicolon            uint16 = 0x27
	keySingleQuote          uint16 = 0x28
	keyEnter                uint16 = 0x1C
	keyLeftShift            uint16 = 0x2A
	keyZ                    uint16 = 0x2C
	keyX                    uint16 = 0x2D
	keyC                    uint16 = 0x2E
	keyV                    uint16 = 0x2F
	keyB                    uint16 = 0x30
	keyN                    uint16 = 0x31
	keyM                    uint16 = 0x32
	keyComma                uint16 = 0x33
	keyDot                  uint16 = 0x34
	keySlash                uint16 = 0x35
	keyRightShift           uint16 = 0x36
	keyUp                   uint16 = 0x67
	keyLeftCtrl             uint16 = 0x1D
	keyWindowsKey           uint16 = 0x7D
	keyLeftAlt              uint16 = 0x38
	keySpace                uint16 = 0x39
	keyRightAlt             uint16 = 0x64
	keyMenu                 uint16 = 0x7F
	keyRightCtrl            uint16 = 0x61
	keyLeft                 uint16 = 0x69
	keyDown                 uint16 = 0x6C
	keyRight                uint16 = 0x6A
	keyIns                  uint16 = 0x6E
	keyHome                 uint16 = 0x66
	keyPgUp                 uint16 = 0x68
	keyDel                  uint16 = 0x6F
	keyEnd                  uint16 = 0x6B
	keyPgDn                 uint16 = 0x6D
	btnLeft                 uint16 = 0x110
	btnRight                uint16 = 0x111
	btnMiddle               uint16 = 0x112
	btnForward              uint16 = 0x114
	btnBack                 uint16 = 0x113
	keyKp1                  uint16 = 0x4F
	keyKp2                  uint16 = 0x50
	keyKp3                  uint16 = 0x51
	keyKp4                  uint16 = 0x4B
	keyKp5                  uint16 = 0x4C
	keyKp6                  uint16 = 0x4D
	keyKp7                  uint16 = 0x47
	keyKp8                  uint16 = 0x48
	keyKp9                  uint16 = 0x49
	keyKp0                  uint16 = 0x52
	keyKpDot                uint16 = 0x53
	keyKpPlus               uint16 = 0x4E
	keyKpMinus              uint16 = 0x4A
	keyKpMultiply           uint16 = 0x37
	keyKpDivide             uint16 = 0x62
	keyKpEnter              uint16 = 0x60
	keyScreenBrightnessDown uint16 = 0xE0
	keyScreenBrightnessUp   uint16 = 0xE1
	keyPrtSc                uint16 = 0x63
	keyPause                uint16 = 0x77
	evRel                   uint16 = 0x02
	relWheel                uint16 = 0x08
	relHWheel               uint16 = 0x06
	btnControllerSouth      uint16 = 0x130 // A, (XBox) / Cross (PS)
	btnControllerEast       uint16 = 0x131 // B, Circle
	btnControllerNorth      uint16 = 0x133 // Y. Triangle
	btnControllerWest       uint16 = 0x134 // X. Square
	btnControllerTL         uint16 = 0x136 // L1
	btnControllerTR         uint16 = 0x137 // R1
	btnControllerTL2        uint16 = 0x138 // L2 (can also be EV_ABS axis)
	btnControllerTR2        uint16 = 0x139 // R2
	btnControllerSelect     uint16 = 0x13A // Select / Share
	btnControllerStart      uint16 = 0x13B // Start / Options
	btnControllerMode       uint16 = 0x13C // Mode / PS button
	btnControllerThumbL     uint16 = 0x13D // Left stick click
	btnControllerThumbR     uint16 = 0x13E // Right stick click
	btnControllerDpadUp     uint16 = 0x220
	btnControllerDpadDown   uint16 = 0x221
	btnControllerDpadLeft   uint16 = 0x222
	btnControllerDpadRight  uint16 = 0x223
	inputActions            map[uint16]InputAction
	virtualKeyboardPointer  uintptr
	virtualMousePointer     uintptr
	virtualKeyboardFile     *os.File
	virtualMouseFile        *os.File
)

type inputEvent struct {
	Time  syscall.Timeval
	Type  uint16
	Code  uint16
	Value int32
}

// buildInputActions will fill inputActions map with InputAction data
func buildInputActions() {
	inputActions = make(map[uint16]InputAction)

	// Placeholder
	inputActions[None] = InputAction{Name: "None"}

	// Media
	inputActions[VolumeUp] = InputAction{Name: "Volume Up", CommandCode: keyVolumeUp, Media: true}
	inputActions[VolumeDown] = InputAction{Name: "Volume Down", CommandCode: keyVolumeDown, Media: true}
	inputActions[VolumeMute] = InputAction{Name: "Mute", CommandCode: keyVolumeMute, Media: true}
	inputActions[MediaStop] = InputAction{Name: "Stop", CommandCode: keyMediaStop, Media: true}
	inputActions[MediaPrev] = InputAction{Name: "Previous", CommandCode: keyMediaPrev, Media: true}
	inputActions[MediaPlayPause] = InputAction{Name: "Play", CommandCode: keyMediaPlay, Media: true}
	inputActions[MediaNext] = InputAction{Name: "Next", CommandCode: keyMediaNext, Media: true}

	// Numbers
	inputActions[Number0] = InputAction{Name: "Number 0", CommandCode: keyNumber0}
	inputActions[Number1] = InputAction{Name: "Number 1", CommandCode: keyNumber1}
	inputActions[Number2] = InputAction{Name: "Number 2", CommandCode: keyNumber2}
	inputActions[Number3] = InputAction{Name: "Number 3", CommandCode: keyNumber3}
	inputActions[Number4] = InputAction{Name: "Number 4", CommandCode: keyNumber4}
	inputActions[Number5] = InputAction{Name: "Number 5", CommandCode: keyNumber5}
	inputActions[Number6] = InputAction{Name: "Number 6", CommandCode: keyNumber6}
	inputActions[Number7] = InputAction{Name: "Number 7", CommandCode: keyNumber7}
	inputActions[Number8] = InputAction{Name: "Number 8", CommandCode: keyNumber8}
	inputActions[Number9] = InputAction{Name: "Number 9", CommandCode: keyNumber9}

	// Numpad
	inputActions[Kp1] = InputAction{Name: "Numpad 1", CommandCode: keyKp1}
	inputActions[Kp2] = InputAction{Name: "Numpad 2", CommandCode: keyKp2}
	inputActions[Kp3] = InputAction{Name: "Numpad 3", CommandCode: keyKp3}
	inputActions[Kp4] = InputAction{Name: "Numpad 4", CommandCode: keyKp4}
	inputActions[Kp5] = InputAction{Name: "Numpad 5", CommandCode: keyKp5}
	inputActions[Kp6] = InputAction{Name: "Numpad 6", CommandCode: keyKp6}
	inputActions[Kp7] = InputAction{Name: "Numpad 7", CommandCode: keyKp7}
	inputActions[Kp8] = InputAction{Name: "Numpad 8", CommandCode: keyKp8}
	inputActions[Kp9] = InputAction{Name: "Numpad 9", CommandCode: keyKp9}
	inputActions[Kp0] = InputAction{Name: "Numpad 0", CommandCode: keyKp0}
	inputActions[KpEnter] = InputAction{Name: "Numpad Enter", CommandCode: keyKpEnter}
	inputActions[KpDot] = InputAction{Name: "Numpad Dot (.)", CommandCode: keyKpDot}
	inputActions[KpPlus] = InputAction{Name: "Numpad Plus (+)", CommandCode: keyKpPlus}
	inputActions[KpMinus] = InputAction{Name: "Numpad Minus (-)", CommandCode: keyKpMinus}
	inputActions[KpMultiply] = InputAction{Name: "Numpad Multiply (*)", CommandCode: keyKpMultiply}
	inputActions[KpDivide] = InputAction{Name: "Numpad Divide (/)", CommandCode: keyKpDivide}

	// Keys
	inputActions[KeyMinus] = InputAction{Name: "Minus (-)", CommandCode: keyMinus}
	inputActions[KeyEqual] = InputAction{Name: "Equals (=)", CommandCode: keyEqual}
	inputActions[KeyQ] = InputAction{Name: "Q", CommandCode: keyQ}
	inputActions[KeyW] = InputAction{Name: "W", CommandCode: keyW}
	inputActions[KeyE] = InputAction{Name: "E", CommandCode: keyE}
	inputActions[KeyR] = InputAction{Name: "R", CommandCode: keyR}
	inputActions[KeyT] = InputAction{Name: "T", CommandCode: keyT}
	inputActions[KeyY] = InputAction{Name: "Y", CommandCode: keyY}
	inputActions[KeyU] = InputAction{Name: "U", CommandCode: keyU}
	inputActions[KeyI] = InputAction{Name: "I", CommandCode: keyI}
	inputActions[KeyO] = InputAction{Name: "O", CommandCode: keyO}
	inputActions[KeyP] = InputAction{Name: "P", CommandCode: keyP}
	inputActions[KeyA] = InputAction{Name: "A", CommandCode: keyA}
	inputActions[KeyS] = InputAction{Name: "S", CommandCode: keyS}
	inputActions[KeyD] = InputAction{Name: "D", CommandCode: keyD}
	inputActions[KeyF] = InputAction{Name: "F", CommandCode: keyF}
	inputActions[KeyG] = InputAction{Name: "G", CommandCode: keyG}
	inputActions[KeyH] = InputAction{Name: "H", CommandCode: keyH}
	inputActions[KeyJ] = InputAction{Name: "J", CommandCode: keyJ}
	inputActions[KeyK] = InputAction{Name: "K", CommandCode: keyK}
	inputActions[KeyL] = InputAction{Name: "L", CommandCode: keyL}
	inputActions[KeyZ] = InputAction{Name: "Z", CommandCode: keyZ}
	inputActions[KeyX] = InputAction{Name: "X", CommandCode: keyX}
	inputActions[KeyC] = InputAction{Name: "C", CommandCode: keyC}
	inputActions[KeyV] = InputAction{Name: "V", CommandCode: keyV}
	inputActions[KeyB] = InputAction{Name: "B", CommandCode: keyB}
	inputActions[KeyN] = InputAction{Name: "N", CommandCode: keyN}
	inputActions[KeyM] = InputAction{Name: "M", CommandCode: keyM}
	inputActions[KeyF1] = InputAction{Name: "F1", CommandCode: keyF1}
	inputActions[KeyF2] = InputAction{Name: "F2", CommandCode: keyF2}
	inputActions[KeyF3] = InputAction{Name: "F3", CommandCode: keyF3}
	inputActions[KeyF4] = InputAction{Name: "F4", CommandCode: keyF4}
	inputActions[KeyF5] = InputAction{Name: "F5", CommandCode: keyF5}
	inputActions[KeyF6] = InputAction{Name: "F6", CommandCode: keyF6}
	inputActions[KeyF7] = InputAction{Name: "F7", CommandCode: keyF7}
	inputActions[KeyF8] = InputAction{Name: "F8", CommandCode: keyF8}
	inputActions[KeyF9] = InputAction{Name: "F9", CommandCode: keyF9}
	inputActions[KeyF10] = InputAction{Name: "F10", CommandCode: keyF10}
	inputActions[KeyF11] = InputAction{Name: "F11", CommandCode: keyF11}
	inputActions[KeyF12] = InputAction{Name: "F12", CommandCode: keyF12}
	inputActions[KeyF13] = InputAction{Name: "F13", CommandCode: keyF13}
	inputActions[KeyF14] = InputAction{Name: "F14", CommandCode: keyF14}
	inputActions[KeyF15] = InputAction{Name: "F15", CommandCode: keyF15}
	inputActions[KeyF16] = InputAction{Name: "F16", CommandCode: keyF16}
	inputActions[KeyF17] = InputAction{Name: "F17", CommandCode: keyF17}
	inputActions[KeyF18] = InputAction{Name: "F18", CommandCode: keyF18}
	inputActions[KeyF19] = InputAction{Name: "F19", CommandCode: keyF19}
	inputActions[KeyF20] = InputAction{Name: "F20", CommandCode: keyF20}
	inputActions[KeyF21] = InputAction{Name: "F21", CommandCode: keyF21}
	inputActions[KeyF22] = InputAction{Name: "F22", CommandCode: keyF22}
	inputActions[KeyF23] = InputAction{Name: "F23", CommandCode: keyF23}
	inputActions[KeyF24] = InputAction{Name: "F24", CommandCode: keyF24}
	inputActions[KeyBack] = InputAction{Name: "Back", CommandCode: keyBack}
	inputActions[KeyTab] = InputAction{Name: "Tab", CommandCode: keyTab}
	inputActions[KeyEsc] = InputAction{Name: "Esc", CommandCode: keyEsc}
	inputActions[KeyTilde] = InputAction{Name: "Tilde (`)", CommandCode: keyTilde}
	inputActions[KeyLeftSquare] = InputAction{Name: "Left Square [{", CommandCode: keyLeftSquare}
	inputActions[KeyRightSquare] = InputAction{Name: "Right Square }]", CommandCode: keyRightSquare}
	inputActions[KeyBackslash] = InputAction{Name: "Backslash (\\)", CommandCode: keyBackslash}
	inputActions[KeyCapslock] = InputAction{Name: "Capslock", CommandCode: keyCapslock}
	inputActions[KeySemicolon] = InputAction{Name: "Semicolon (;)", CommandCode: keySemicolon}
	inputActions[KeySingleQuote] = InputAction{Name: "Single Quote (')", CommandCode: keySingleQuote}
	inputActions[KeyEnter] = InputAction{Name: "Enter", CommandCode: keyEnter}
	inputActions[KeyLeftShift] = InputAction{Name: "Left Shift", CommandCode: keyLeftShift}
	inputActions[KeyComma] = InputAction{Name: "Comma (,)", CommandCode: keyComma}
	inputActions[KeyDot] = InputAction{Name: "Dot (.)", CommandCode: keyDot}
	inputActions[KeySlash] = InputAction{Name: "Slash (/)", CommandCode: keySlash}
	inputActions[KeyRightShift] = InputAction{Name: "Right Shift", CommandCode: keyRightShift}
	inputActions[KeyUp] = InputAction{Name: "Up", CommandCode: keyUp}
	inputActions[KeyLeftCtrl] = InputAction{Name: "Left Ctrl", CommandCode: keyLeftCtrl}
	inputActions[KeyWindowsKey] = InputAction{Name: "Windows Key", CommandCode: keyWindowsKey}
	inputActions[KeyLeftAlt] = InputAction{Name: "Left Alt", CommandCode: keyLeftAlt}
	inputActions[KeySpace] = InputAction{Name: "Space", CommandCode: keySpace}
	inputActions[KeyRightAlt] = InputAction{Name: "Right Alt", CommandCode: keyRightAlt}
	inputActions[KeyMenu] = InputAction{Name: "Menu", CommandCode: keyMenu}
	inputActions[KeyRightCtrl] = InputAction{Name: "Right Ctrl", CommandCode: keyRightCtrl}
	inputActions[KeyLeft] = InputAction{Name: "Left", CommandCode: keyLeft}
	inputActions[KeyDown] = InputAction{Name: "Down", CommandCode: keyDown}
	inputActions[KeyRight] = InputAction{Name: "Right", CommandCode: keyRight}
	inputActions[KeyIns] = InputAction{Name: "Insert", CommandCode: keyIns}
	inputActions[KeyHome] = InputAction{Name: "Home", CommandCode: keyHome}
	inputActions[KeyPgUp] = InputAction{Name: "Pg Up", CommandCode: keyPgUp}
	inputActions[KeyDel] = InputAction{Name: "Delete", CommandCode: keyDel}
	inputActions[KeyEnd] = InputAction{Name: "End", CommandCode: keyEnd}
	inputActions[KeyPgDn] = InputAction{Name: "Pg Dn", CommandCode: keyPgDn}
	inputActions[KeyScreenBrightnessDown] = InputAction{Name: "Screen Brightness (-)", CommandCode: keyScreenBrightnessDown}
	inputActions[KeyScreenBrightnessUp] = InputAction{Name: "Screen Brightness (+)", CommandCode: keyScreenBrightnessUp}
	inputActions[KeyPrtSc] = InputAction{Name: "PrtSc", CommandCode: keyPrtSc}
	inputActions[KeyPause] = InputAction{Name: "Pause", CommandCode: keyPause}

	// Mouse
	inputActions[BtnLeft] = InputAction{Name: "(Mouse) Left Click", CommandCode: btnLeft, Mouse: true}
	inputActions[BtnRight] = InputAction{Name: "(Mouse) Right Click", CommandCode: btnRight, Mouse: true}
	inputActions[BtnMiddle] = InputAction{Name: "(Mouse) Middle Click", CommandCode: btnMiddle, Mouse: true}
	inputActions[BtnBack] = InputAction{Name: "(Mouse) Back", CommandCode: btnBack, Mouse: true}
	inputActions[BtnForward] = InputAction{Name: "(Mouse) Forward", CommandCode: btnForward, Mouse: true}

	// Controller
	inputActions[KeyControllerSouth] = InputAction{Name: "(Controller) South (A)", CommandCode: btnControllerSouth, Controller: true}
	inputActions[KeyControllerEast] = InputAction{Name: "(Controller) East (B)", CommandCode: btnControllerEast, Controller: true}
	inputActions[KeyControllerNorth] = InputAction{Name: "(Controller) North (Y)", CommandCode: btnControllerNorth, Controller: true}
	inputActions[KeyControllerWest] = InputAction{Name: "(Controller) West (X)", CommandCode: btnControllerWest, Controller: true}
	inputActions[KeyControllerTL] = InputAction{Name: "(Controller) L1", CommandCode: btnControllerTL, Controller: true}
	inputActions[KeyControllerTR] = InputAction{Name: "(Controller) R1", CommandCode: btnControllerTR, Controller: true}
	inputActions[KeyControllerTL2] = InputAction{Name: "(Controller) L2", CommandCode: btnControllerTL2, Controller: true}
	inputActions[KeyControllerTR2] = InputAction{Name: "(Controller) R2", CommandCode: btnControllerTR2, Controller: true}
	inputActions[KeyControllerSelect] = InputAction{Name: "(Controller) Select", CommandCode: btnControllerSelect, Controller: true}
	inputActions[KeyControllerStart] = InputAction{Name: "(Controller) Start", CommandCode: btnControllerStart, Controller: true}
	inputActions[KeyControllerMode] = InputAction{Name: "(Controller) Mode", CommandCode: btnControllerMode, Controller: true}
	inputActions[KeyControllerThumbL] = InputAction{Name: "(Controller) Thumb L Press", CommandCode: btnControllerThumbL, Controller: true}
	inputActions[KeyControllerThumbR] = InputAction{Name: "(Controller) Thumb R Press", CommandCode: btnControllerThumbR, Controller: true}
	inputActions[KeyControllerDpadUp] = InputAction{Name: "(Controller) D-Pad Up", CommandCode: btnControllerDpadUp, Controller: true}
	inputActions[KeyControllerDpadDown] = InputAction{Name: "(Controller) D-Pad Down", CommandCode: btnControllerDpadDown, Controller: true}
	inputActions[KeyControllerDpadLeft] = InputAction{Name: "(Controller) D-Pad Left", CommandCode: btnControllerDpadLeft, Controller: true}
	inputActions[KeyControllerDpadRight] = InputAction{Name: "(Controller) D-Pad Right", CommandCode: btnControllerDpadRight, Controller: true}
}

// charToKey mapping
var charToKey = map[rune]uint16{
	// letters (lowercase)
	'a': keyA, 'b': keyB, 'c': keyC, 'd': keyD, 'e': keyE,
	'f': keyF, 'g': keyG, 'h': keyH, 'i': keyI, 'j': keyJ,
	'k': keyK, 'l': keyL, 'm': keyM, 'n': keyN, 'o': keyO,
	'p': keyP, 'q': keyQ, 'r': keyR, 's': keyS, 't': keyT,
	'u': keyU, 'v': keyV, 'w': keyW, 'x': keyX, 'y': keyY,
	'z': keyZ,

	// numbers (top row)
	'0': keyNumber0, '1': keyNumber1, '2': keyNumber2,
	'3': keyNumber3, '4': keyNumber4, '5': keyNumber5,
	'6': keyNumber6, '7': keyNumber7, '8': keyNumber8, '9': keyNumber9,

	// whitespace / control
	' ':  keySpace,
	'\n': keyEnter,
	'\r': keyEnter,
	'\t': keyTab,
	'\b': keyBack,

	// common punctuation
	'-':  keyMinus,
	'=':  keyEqual,
	',':  keyComma,
	'.':  keyDot,
	'/':  keySlash,
	';':  keySemicolon,
	'\'': keySingleQuote,
	'[':  keyLeftSquare,
	']':  keyRightSquare,
	'\\': keyBackslash,
	'`':  keyTilde,
}

// shiftedToBase mappings with Shift
var shiftedToBase = map[rune]rune{
	// uppercase letters
	'A': 'a', 'B': 'b', 'C': 'c', 'D': 'd', 'E': 'e',
	'F': 'f', 'G': 'g', 'H': 'h', 'I': 'i', 'J': 'j',
	'K': 'k', 'L': 'l', 'M': 'm', 'N': 'n', 'O': 'o',
	'P': 'p', 'Q': 'q', 'R': 'r', 'S': 's', 'T': 't',
	'U': 'u', 'V': 'v', 'W': 'w', 'X': 'x', 'Y': 'y',
	'Z': 'z',

	// shifted numbers -> symbol
	'!': '1', '@': '2', '#': '3', '$': '4', '%': '5',
	'^': '6', '&': '7', '*': '8', '(': '9', ')': '0',

	// shifted punctuation
	'_': '-', '+': '=', '{': '[', '}': ']', '|': '\\',
	':': ';', '"': '\'', '<': ',', '>': '.', '?': '/', '~': '`',
}

// Init will fetch an input device
func Init() {
	buildInputActions()
	CreateVirtualKeyboard(productId)
	CreateVirtualMouse(productId)
}

// CreateVirtualKeyboard will create new keyboard based on given productId
func CreateVirtualKeyboard(productId uint16) {
	if virtualKeyboardFile == nil {
		err := createVirtualKeyboard(vendorId, productId)
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Failed to create virtual keyboard")
			return
		}
		logger.Log(logger.Fields{}).Info("Virtual keyboard successfully created")
	}
}

// CreateVirtualMouse will create new mouse based on given productId
func CreateVirtualMouse(productId uint16) {
	if virtualMouseFile == nil {
		err := createVirtualMouse(vendorId, productId)
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Failed to create virtual keyboard")
			return
		}
		logger.Log(logger.Fields{}).Info("Virtual mouse successfully created")
	}
}

// Stop will stop input manager and destroy virtual inputs
func Stop() {
	logger.Log(logger.Fields{}).Info("Stopping virtual keyboard and mouse")
	destroyVirtualKeyboard()
	destroyVirtualMouse()
}

// GetMediaKeys will return a map of InputAction for Media keys
func GetMediaKeys() map[uint16]InputAction {
	keys := make(map[uint16]InputAction)
	for key, value := range inputActions {
		if value.Media {
			keys[key] = value
		}
	}
	return keys
}

// GetControllerKeys will return a map of InputAction for Controller keys
func GetControllerKeys() map[uint16]InputAction {
	keys := make(map[uint16]InputAction)
	for key, value := range inputActions {
		if value.Controller {
			keys[key] = value
		}
	}
	return keys
}

// GetInputKeys will return a map of InputAction for non-media keys
func GetInputKeys() map[uint16]InputAction {
	keys := make(map[uint16]InputAction)
	for key, value := range inputActions {
		if value.Media {
			continue
		}
		keys[key] = value
	}
	return keys
}

// GetMouseButtons will return a map of InputAction for mouse buttons
func GetMouseButtons() map[uint16]InputAction {
	keys := make(map[uint16]InputAction)
	for key, value := range inputActions {
		if value.Mouse {
			keys[key] = value
		}
	}
	return keys
}

func GetKeyName(keyIndex uint16) string {
	if key, ok := inputActions[keyIndex]; ok {
		return key.Name
	}
	return ""
}

// GetInputActions will return a map of InputAction
func GetInputActions() map[uint16]InputAction {
	return inputActions
}

// FindKeyAssignment will find nearest KeyAssignment by input value and given offset
func FindKeyAssignment(keyAssignment map[int]KeyAssignment, input uint32, offset []uint32) uint32 {
	keys := make([]int, 0)
	for k := range keyAssignment {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	if slices.Contains(keys, int(input)) {
		return input
	} else {
		for _, k := range keys {
			for _, value := range offset {
				if k == int(input-value) {
					return uint32(k)
				}
			}
		}
	}
	return 0
}

// getInputAction will return InputAction based on actionType
func getInputAction(actionType uint16) *InputAction {
	if action, ok := inputActions[actionType]; ok {
		return &action
	}
	return nil
}

// createInputEvent will create a list of input events
func createInputEvent(code uint16, hold bool) []inputEvent {
	// Create an input event for key press
	keyPress := inputEvent{
		Type:  evKey,
		Code:  code,
		Value: 1, // Key press
	}

	// Synchronization event
	syncEvent := inputEvent{
		Type:  evSyn,
		Code:  0,
		Value: 0,
	}

	events := []inputEvent{keyPress, syncEvent}

	// Only release if hold is false
	if !hold {
		keyRelease := inputEvent{
			Type:  evKey,
			Code:  code,
			Value: 0,
		}
		events = append(events, keyRelease, syncEvent)
	}
	return events
}

func createInputEventHold(code uint16, press bool) []inputEvent {
	// Synchronization event
	syncEvent := inputEvent{
		Type:  evSyn,
		Code:  0,
		Value: 0,
	}

	var events []inputEvent

	// Only release if hold is false
	if press {
		// Create an input event for key press
		keyPress := inputEvent{
			Type:  evKey,
			Code:  code,
			Value: 1, // Key press
		}
		events = []inputEvent{keyPress, syncEvent}
	} else {
		// Create an input event for key release
		keyRelease := inputEvent{
			Type:  evKey,
			Code:  code,
			Value: 0,
		}
		events = []inputEvent{keyRelease, syncEvent}
	}
	return events
}

// writeVirtualEvent will send event to virtual keyboard device
func writeVirtualEvent(f *os.File, event *inputEvent) error {
	if f == nil {
		logger.Log(logger.Fields{}).Error("No valid virtual inputs")
		return errors.New("no valid virtual inputs")
	}

	buf := (*(*[unsafe.Sizeof(*event)]byte)(unsafe.Pointer(event)))[:]
	_, err := f.Write(buf)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Failed to send virtual inputs event")
		return err
	}
	return nil
}
