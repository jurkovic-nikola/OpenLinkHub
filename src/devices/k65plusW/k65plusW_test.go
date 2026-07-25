package k65plusW

import "testing"

func TestControlDialPressDetection(t *testing.T) {
	raw := make([]byte, 18)
	if isControlDialPress(raw) {
		t.Fatal("zero-value report was detected as a control-dial press")
	}
	raw[17] = 0x02
	if !isControlDialPress(raw) {
		t.Fatal("control-dial press report was not detected")
	}
	if isControlDialPress(raw[:17]) {
		t.Fatal("short report was detected as a control-dial press")
	}
}

func TestControlDialPressActions(t *testing.T) {
	tests := []struct {
		controlDial int
		want        controlDialPressAction
	}{
		{controlDial: 1, want: controlDialPressMute},
		{controlDial: 2, want: controlDialPressBrightness},
		{controlDial: 3, want: controlDialPressCtrlEnd},
		{controlDial: 4, want: controlDialPressZoomReset},
		{controlDial: 5, want: controlDialPressNone},
		{controlDial: 6, want: controlDialPressMediaPlayPause},
		{controlDial: 7, want: controlDialPressHorizontalScrollReset},
		{controlDial: 99, want: controlDialPressNone},
	}

	for _, test := range tests {
		if got := controlDialAction(test.controlDial); got != test.want {
			t.Errorf("controlDialAction(%d) = %d, want %d", test.controlDial, got, test.want)
		}
	}
}
