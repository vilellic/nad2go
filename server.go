package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"unicode"
)

const defaultSerialPort = "/dev/cu.PL2303G-USBtoUART2140"
const defaultHTTPPort = "8099"

type ControlResponse struct {
	Command  string `json:"command"`
	Response string `json:"response"`
	Error    string `json:"error,omitempty"`
}

type StatusResponse struct {
	PortPath  string `json:"portPath"`
	Connected bool   `json:"connected"`
}

func main() {
	serialPath := os.Getenv("SERIAL_PORT")
	if serialPath == "" {
		serialPath = defaultSerialPort
	}

	httpPort := os.Getenv("PORT")
	if httpPort == "" {
		httpPort = defaultHTTPPort
	}

	log.Printf("Opening serial port: %s", serialPath)
	if err := OpenSerialPort(serialPath); err != nil {
		log.Printf("Warning: could not open serial port: %v", err)
		log.Println("Server will start anyway; commands will fail until port is available")
	}

	http.HandleFunc("/control", handleControl)
	http.HandleFunc("/status", handleStatus)
	http.HandleFunc("/ir-commands", handleIRCommands)

	log.Printf("nad2go server starting on :%s", httpPort)
	log.Fatal(http.ListenAndServe(":"+httpPort, nil))
}

func handleControl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cmd := r.URL.Query().Get("cmd")
	op := r.URL.Query().Get("op")
	value := r.URL.Query().Get("value")

	if cmd == "" {
		http.Error(w, "Missing required parameter: cmd", http.StatusBadRequest)
		return
	}
	if op == "" {
		http.Error(w, "Missing required parameter: op", http.StatusBadRequest)
		return
	}
	if op != "=" && op != "+" && op != "-" && op != "?" {
		http.Error(w, "Invalid op: must be =, +, -, or ?", http.StatusBadRequest)
		return
	}

	// For Main.IR with a non-numeric value, look up the IR command map
	if cmd == "Main.IR" && value != "" && !isNumeric(value) {
		code, ok := LookupIR(value)
		if !ok {
			resp := ControlResponse{
				Error: fmt.Sprintf("Unknown IR command: %s", value),
			}
			writeJSON(w, http.StatusBadRequest, resp)
			return
		}
		log.Printf("IR lookup: %q -> %d", value, code)
		value = strconv.Itoa(code)
	}

	// Build the RS232 command string
	command := cmd + op + value
	log.Printf("Executing: %s", command)

	response, err := SendCommand(command)

	resp := ControlResponse{
		Command: command,
	}
	if err != nil {
		resp.Error = err.Error()
		resp.Response = ""
		writeJSON(w, http.StatusInternalServerError, resp)
		return
	}

	resp.Response = response
	writeJSON(w, http.StatusOK, resp)
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	resp := StatusResponse{
		PortPath:  GetPortPath(),
		Connected: IsPortOpen(),
	}
	writeJSON(w, http.StatusOK, resp)
}

func handleIRCommands(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, irCommands)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func isNumeric(s string) bool {
	for _, c := range s {
		if !unicode.IsDigit(c) {
			return false
		}
	}
	return len(s) > 0
}
