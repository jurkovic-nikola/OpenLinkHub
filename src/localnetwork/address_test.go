package localnetwork

import "testing"

func TestAddressUsesIPv4LoopbackAndConfiguredPort(t *testing.T) {
	tests := []struct {
		port int
		want string
	}{
		{port: 27003, want: "127.0.0.1:27003"},
		{port: 6742, want: "127.0.0.1:6742"},
		{port: 28080, want: "127.0.0.1:28080"},
	}

	for _, test := range tests {
		if got := Address(test.port); got != test.want {
			t.Errorf("Address(%d) = %q, want %q", test.port, got, test.want)
		}
	}
}
