package openrgb

import (
	"LumenForge/src/config"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"testing"
	"time"
)

func withFakeSDKServer(t *testing.T, handler func(net.Conn)) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	previousAddress := sdkAddress
	previousDial := dialContext
	done := make(chan struct{})
	if err == nil {
		sdkAddress = func() (string, error) { return listener.Addr().String(), nil }
		go func() {
			defer close(done)
			conn, acceptErr := listener.Accept()
			if acceptErr != nil {
				return
			}
			defer conn.Close()
			handler(conn)
		}()
	} else {
		clientConn, serverConn := net.Pipe()
		sdkAddress = func() (string, error) { return "pipe", nil }
		dialContext = func(context.Context, string, string) (net.Conn, error) {
			return clientConn, nil
		}
		go func() {
			defer close(done)
			defer serverConn.Close()
			handler(serverConn)
		}()
	}

	t.Cleanup(func() {
		sdkAddress = previousAddress
		dialContext = previousDial
		if listener != nil {
			_ = listener.Close()
		}
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Error("fake OpenRGB server did not stop")
		}
	})
}

func readRequest(t *testing.T, conn net.Conn) []byte {
	t.Helper()
	header := make([]byte, 16)
	if _, err := io.ReadFull(conn, header); err != nil {
		t.Errorf("read request: %v", err)
	}
	return header
}

func responseHeader(magic string, controllerID, opcode, size uint32) []byte {
	header := make([]byte, 16)
	copy(header[:4], magic)
	binary.LittleEndian.PutUint32(header[4:8], controllerID)
	binary.LittleEndian.PutUint32(header[8:12], opcode)
	binary.LittleEndian.PutUint32(header[12:16], size)
	return header
}

func writeControllerCount(conn net.Conn, count uint32) error {
	if _, err := conn.Write(responseHeader("ORGB", 0, opcodeRequestControllerCount, 4)); err != nil {
		return err
	}
	payload := make([]byte, 4)
	binary.LittleEndian.PutUint32(payload, count)
	_, err := conn.Write(payload)
	return err
}

func controllerPayload(name, vendor, description, version, serial, location string) []byte {
	payload := new(bytes.Buffer)
	_ = binary.Write(payload, binary.LittleEndian, uint32(0))
	_ = binary.Write(payload, binary.LittleEndian, int32(0))
	for _, value := range []string{name, vendor, description, version, serial, location} {
		encoded := append([]byte(value), 0)
		_ = binary.Write(payload, binary.LittleEndian, uint16(len(encoded)))
		_, _ = payload.Write(encoded)
	}
	_ = binary.Write(payload, binary.LittleEndian, uint16(0))
	_ = binary.Write(payload, binary.LittleEndian, int32(-1))
	return payload.Bytes()
}

func TestDiscoverControllersValidTransactionSetsConnected(t *testing.T) {
	withFakeSDKServer(t, func(conn net.Conn) {
		readRequest(t, conn)
		if err := writeControllerCount(conn, 0); err != nil {
			t.Errorf("write response: %v", err)
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	controllers, err := DiscoverControllersContext(ctx)
	if err != nil {
		t.Fatalf("DiscoverControllersContext: %v", err)
	}
	if len(controllers) != 0 {
		t.Fatalf("got %d controllers, want 0", len(controllers))
	}
	state, statusErr := GetStatus()
	if state != StateConnected || statusErr != nil {
		t.Fatalf("status = %q, %v; want Connected, nil", state, statusErr)
	}
}

func TestStatusNeutralDiscoveryPreservesGlobalStatus(t *testing.T) {
	withFakeSDKServer(t, func(conn net.Conn) {
		readRequest(t, conn)
		if err := writeControllerCount(conn, 0); err != nil {
			t.Errorf("write response: %v", err)
		}
	})

	sentinel := errors.New("manager retry state")
	SetDisconnected(sentinel)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	controllers, err := DiscoverControllersStatusNeutralContext(ctx)
	if err != nil {
		t.Fatalf("DiscoverControllersStatusNeutralContext: %v", err)
	}
	if len(controllers) != 0 {
		t.Fatalf("got %d controllers, want 0", len(controllers))
	}
	state, statusErr := GetStatus()
	if state != StateOffline || statusErr != sentinel {
		t.Fatalf("status = %q, %v; want unchanged Offline sentinel", state, statusErr)
	}
}

func TestStatusNeutralDiscoveryFailurePreservesGlobalStatus(t *testing.T) {
	previousAddress := sdkAddress
	sdkAddress = func() (string, error) { return "", errors.New("injected address failure") }
	t.Cleanup(func() { sdkAddress = previousAddress })

	SetConnected()
	if _, err := DiscoverControllersStatusNeutralContext(context.Background()); err == nil {
		t.Fatal("expected status-neutral discovery failure")
	}
	state, statusErr := GetStatus()
	if state != StateConnected || statusErr != nil {
		t.Fatalf("status = %q, %v; want unchanged Connected status", state, statusErr)
	}
}

func TestDialAloneDoesNotMarkSDKConnected(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer serverConn.Close()
	previousAddress := sdkAddress
	previousDial := dialContext
	sdkAddress = func() (string, error) { return "pipe", nil }
	dialContext = func(context.Context, string, string) (net.Conn, error) { return clientConn, nil }
	t.Cleanup(func() {
		sdkAddress = previousAddress
		dialContext = previousDial
	})
	SetDisconnected(errors.New("not connected"))

	conn, err := dial(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	_ = conn.Close()
	state, _ := GetStatus()
	if state != StateOffline {
		t.Fatalf("status = %q after dial, want Offline", state)
	}
}

func TestSDKAddressUsesIPv4LoopbackAndConfiguredOpenRGBPort(t *testing.T) {
	tests := []struct {
		name          string
		listenAddress string
		openRGBPort   int
		want          string
	}{
		{name: "default port", listenAddress: "0.0.0.0", openRGBPort: 6742, want: "127.0.0.1:6742"},
		{name: "configured port", listenAddress: "192.168.1.50", openRGBPort: 6743, want: "127.0.0.1:6743"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			address, err := sdkAddressForConfig(config.Configuration{
				ListenAddress: test.listenAddress,
				OpenRGBPort:   test.openRGBPort,
			})
			if err != nil {
				t.Fatalf("sdkAddressForConfig() returned error: %v", err)
			}
			if address != test.want {
				t.Fatalf("sdkAddressForConfig() = %q, want %q", address, test.want)
			}
		})
	}
}

func TestDiscoverControllersParsesControllerMetadata(t *testing.T) {
	payload := controllerPayload("Test Controller", "Test Vendor", "Description", "1.0", "external-serial", "usb:1")
	withFakeSDKServer(t, func(conn net.Conn) {
		readRequest(t, conn)
		if err := writeControllerCount(conn, 1); err != nil {
			t.Errorf("write count: %v", err)
			return
		}
		readRequest(t, conn)
		if _, err := conn.Write(responseHeader("ORGB", 0, opcodeRequestControllerData, uint32(len(payload)))); err != nil {
			t.Errorf("write header: %v", err)
			return
		}
		if _, err := conn.Write(payload); err != nil {
			t.Errorf("write payload: %v", err)
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	controllers, err := DiscoverControllersContext(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(controllers) != 1 {
		t.Fatalf("controllers = %#v", controllers)
	}
	controller := controllers[0]
	if controller.ID != 0 || controller.Name != "Test Controller" || controller.Vendor != "Test Vendor" || controller.Serial != "external-serial" || controller.Location != "usb:1" {
		t.Fatalf("controller metadata = %#v", controller)
	}
}

func TestDiscoverControllersRejectsMalformedResponses(t *testing.T) {
	tests := []struct {
		name    string
		handler func(net.Conn)
	}{
		{
			name: "invalid magic",
			handler: func(conn net.Conn) {
				readRequest(t, conn)
				_, _ = conn.Write(responseHeader("NOPE", 0, opcodeRequestControllerCount, 0))
			},
		},
		{
			name: "truncated header",
			handler: func(conn net.Conn) {
				readRequest(t, conn)
				_, _ = conn.Write([]byte("ORGB"))
			},
		},
		{
			name: "truncated payload",
			handler: func(conn net.Conn) {
				readRequest(t, conn)
				_, _ = conn.Write(responseHeader("ORGB", 0, opcodeRequestControllerCount, 4))
				_, _ = conn.Write([]byte{1, 0})
			},
		},
		{
			name: "oversized payload",
			handler: func(conn net.Conn) {
				readRequest(t, conn)
				_, _ = conn.Write(responseHeader("ORGB", 0, opcodeRequestControllerCount, maxPayloadSize+1))
			},
		},
		{
			name: "excessive controller count",
			handler: func(conn net.Conn) {
				readRequest(t, conn)
				_ = writeControllerCount(conn, maxControllerCount+1)
			},
		},
		{
			name: "wrong controller ID",
			handler: func(conn net.Conn) {
				readRequest(t, conn)
				_, _ = conn.Write(responseHeader("ORGB", 9, opcodeRequestControllerCount, 0))
			},
		},
		{
			name: "wrong opcode",
			handler: func(conn net.Conn) {
				readRequest(t, conn)
				_, _ = conn.Write(responseHeader("ORGB", 0, opcodeRequestControllerData, 0))
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			withFakeSDKServer(t, test.handler)
			ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
			defer cancel()
			if _, err := DiscoverControllersContext(ctx); err == nil {
				t.Fatal("expected malformed response error")
			}
		})
	}
}

func TestDiscoverControllersRejectsMalformedControllerResponse(t *testing.T) {
	tests := []struct {
		name         string
		controllerID uint32
		opcode       uint32
	}{
		{name: "wrong controller ID", controllerID: 7, opcode: opcodeRequestControllerData},
		{name: "wrong opcode", controllerID: 0, opcode: opcodeRequestControllerCount},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			withFakeSDKServer(t, func(conn net.Conn) {
				readRequest(t, conn)
				if err := writeControllerCount(conn, 1); err != nil {
					t.Errorf("write count: %v", err)
					return
				}
				readRequest(t, conn)
				_, _ = conn.Write(responseHeader("ORGB", test.controllerID, test.opcode, 0))
			})
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if _, err := DiscoverControllersContext(ctx); err == nil {
				t.Fatal("expected malformed controller response error")
			}
		})
	}
}

func TestDiscoverControllersStalledResponseHonorsContext(t *testing.T) {
	withFakeSDKServer(t, func(conn net.Conn) {
		readRequest(t, conn)
		_, _ = io.Copy(io.Discard, conn)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	started := time.Now()
	_, err := DiscoverControllersContext(ctx)
	if err == nil {
		t.Fatal("expected stalled response error")
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("stalled discovery took %v", elapsed)
	}
}

func TestSendFrameContextCancellationStopsStalledWrite(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	previousAddress := sdkAddress
	previousDial := dialContext
	sdkAddress = func() (string, error) { return "pipe", nil }
	dialContext = func(context.Context, string, string) (net.Conn, error) { return clientConn, nil }
	t.Cleanup(func() {
		sdkAddress = previousAddress
		dialContext = previousDial
		_ = clientConn.Close()
		_ = serverConn.Close()
	})

	customModeRead := make(chan struct{})
	releaseServer := make(chan struct{})
	defer close(releaseServer)
	go func() {
		defer serverConn.Close()
		if _, err := io.ReadFull(serverConn, make([]byte, 16)); err == nil {
			close(customModeRead)
		}
		<-releaseServer
	}()

	statusMarker := errors.New("prior status")
	SetDisconnected(statusMarker)
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		result <- SendFrameContext(ctx, 1, []byte{1, 2, 3})
	}()
	select {
	case <-customModeRead:
	case <-time.After(time.Second):
		t.Fatal("custom-mode packet was not written")
	}
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("SendFrameContext error = %v, want context cancellation", err)
		}
	case <-time.After(time.Second):
		t.Fatal("SendFrameContext did not stop after cancellation")
	}
	state, statusErr := GetStatus()
	if state != StateOffline || statusErr != statusMarker {
		t.Fatalf("status = %q, %v; cancellation should not replace prior SDK status", state, statusErr)
	}
}
