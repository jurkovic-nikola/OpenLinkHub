package localnetwork

import (
	"net"
	"strconv"
)

const IPv4Loopback = "127.0.0.1"

// Address returns a TCP address restricted to LumenForge's local-only IPv4
// loopback interface.
func Address(port int) string {
	return net.JoinHostPort(IPv4Loopback, strconv.Itoa(port))
}
