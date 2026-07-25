package requests

import "testing"

func TestLinkAdapterResult(t *testing.T) {
	tests := []struct {
		name        string
		result      uint64
		wantMessage string
		wantStatus  int
	}{
		{name: "success", result: 1, wantMessage: "txtLinkAdapterUpdated", wantStatus: 1},
		{name: "missing Link adapter", result: 2, wantMessage: "txtUnableToChangeRgbStripNoLink", wantStatus: 0},
		{name: "generic failure", result: 0, wantMessage: "txtUnableToChangeRgbStrip", wantStatus: 0},
		{name: "unknown result", result: 99, wantMessage: "txtUnableToChangeRgbStrip", wantStatus: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			message, status := linkAdapterResult(test.result)
			if message != test.wantMessage || status != test.wantStatus {
				t.Fatalf("linkAdapterResult(%d) = (%q, %d), want (%q, %d)",
					test.result, message, status, test.wantMessage, test.wantStatus)
			}
		})
	}
}
