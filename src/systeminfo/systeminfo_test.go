package systeminfo

import "testing"

func TestParseAMDGPUModel(t *testing.T) {
	model, err := parseAMDGPUModel([]byte(`{"gpu_data":[{"gpu":0,"asic":{"market_name":"Example Radeon"}}]}`))
	if err != nil {
		t.Fatalf("parseAMDGPUModel() error = %v", err)
	}
	if model != "Example Radeon" {
		t.Fatalf("parseAMDGPUModel() = %q, want %q", model, "Example Radeon")
	}
}

func TestParseAMDUtilization(t *testing.T) {
	utilization, err := parseAMDUtilization([]byte(`{"gpu_data":[{"gpu":0,"usage":{"gfx_activity":{"value":42.5,"unit":"%"}}}]}`))
	if err != nil {
		t.Fatalf("parseAMDUtilization() error = %v", err)
	}
	if utilization != 42.5 {
		t.Fatalf("parseAMDUtilization() = %v, want 42.5", utilization)
	}
}

func TestParseAMDSMIDataRejectsMissingOrInvalidGPUData(t *testing.T) {
	tests := []struct {
		name  string
		parse func([]byte) error
	}{
		{
			name: "empty model data",
			parse: func(data []byte) error {
				_, err := parseAMDGPUModel(data)
				return err
			},
		},
		{
			name: "empty usage data",
			parse: func(data []byte) error {
				_, err := parseAMDUtilization(data)
				return err
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.parse([]byte(`{"gpu_data":[]}`)); err == nil {
				t.Fatal("parser returned nil error for empty gpu_data")
			}
			if err := test.parse([]byte(`{`)); err == nil {
				t.Fatal("parser returned nil error for invalid JSON")
			}
		})
	}
}
