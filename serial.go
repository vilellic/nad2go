package main

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"go.bug.st/serial"
)

const (
	baudRate    = 115200
	readTimeout = 50 * time.Millisecond
)

var (
	serialPort serial.Port
	serialMu   sync.Mutex
	portPath   string
)

// OpenSerialPort opens the serial port at the configured path.
func OpenSerialPort(path string) error {
	serialMu.Lock()
	defer serialMu.Unlock()

	portPath = path
	return openPort()
}

// openPort opens the serial port (must be called with serialMu held).
func openPort() error {
	if serialPort != nil {
		serialPort.Close()
		serialPort = nil
	}

	mode := &serial.Mode{
		BaudRate: baudRate,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}

	port, err := serial.Open(portPath, mode)
	if err != nil {
		return fmt.Errorf("failed to open serial port %s: %w", portPath, err)
	}

	port.SetReadTimeout(readTimeout)
	serialPort = port
	log.Printf("Serial port %s opened", portPath)
	return nil
}

// SendCommand sends an RS232 command to the NAD receiver and returns the response.
// The command is wrapped with CR characters: \r<command>\r
// The mutex ensures only one command is in flight at a time.
func SendCommand(command string) (string, error) {
	serialMu.Lock()
	defer serialMu.Unlock()

	if serialPort == nil {
		if err := openPort(); err != nil {
			return "", err
		}
	}

	// Build the full command with CR wrapping
	fullCmd := "\r" + command + "\r"
	log.Printf("TX: %q", fullCmd)

	_, err := serialPort.Write([]byte(fullCmd))
	if err != nil {
		log.Printf("Write failed, attempting reconnect: %v", err)
		// Try to reconnect once
		if reconnErr := openPort(); reconnErr != nil {
			return "", fmt.Errorf("write failed and reconnect failed: %w", reconnErr)
		}
		_, err = serialPort.Write([]byte(fullCmd))
		if err != nil {
			return "", fmt.Errorf("write failed after reconnect: %w", err)
		}
	}

	// Read response
	response, err := readResponse()
	if err != nil {
		return "", err
	}

	log.Printf("RX: %q", response)
	return response, nil
}

// readResponse reads from the serial port until no more data arrives.
func readResponse() (string, error) {
	var result strings.Builder
	buf := make([]byte, 256)

	for {
		n, err := serialPort.Read(buf)
		if n > 0 {
			result.Write(buf[:n])
		}
		if err != nil {
			// Timeout is expected — means no more data
			break
		}
		if n == 0 {
			break
		}
	}

	// Trim CR/LF from response
	return strings.TrimSpace(result.String()), nil
}

// IsPortOpen returns whether the serial port is currently open.
func IsPortOpen() bool {
	serialMu.Lock()
	defer serialMu.Unlock()
	return serialPort != nil
}

// GetPortPath returns the configured serial port path.
func GetPortPath() string {
	return portPath
}
