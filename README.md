# nad2go

REST API server for controlling NAD amplifiers and receivers via RS232 serial port. Written in Go with zero external HTTP dependencies. Connects to the amplifier through a USB-to-serial adapter and exposes simple HTTP endpoints that can be called from Home Assistant, scripts, or any HTTP client.

Currently includes IR command mappings for the NAD C725BEE, but the RS232 protocol is shared across many NAD models — adapt the IR map for your unit.

## Features

- **Full RS232 control** — power, source, mute, speakers, tuner, sleep, tone defeat, and more
- **IR command passthrough** — send any remote control command via `Main.IR` using plain-text names (e.g. `volume_up`) or decimal codes
- **Fire-and-forget mode** — optional `nowait` parameter returns instantly without waiting for serial response, ideal for volume control
- **Auto-reconnect** — recovers gracefully if the serial connection drops
- **Home Assistant ready** — simple GET endpoints work directly with `rest_command`

## Requirements

- NAD amplifier/receiver with RS232 serial port
- USB-to-serial adapter (e.g. PL2303-based cable)
- Go 1.23+ (for building from source)

## Configuration

| Env Variable  | Default                            | Description         |
|---------------|------------------------------------|---------------------|
| `SERIAL_PORT` | `/dev/cu.PL2303G-USBtoUART2140`    | Serial port path    |
| `PORT`        | `8080`                             | HTTP listen port    |

## Endpoints

### GET /control

Send a command to the amplifier.

| Param    | Required | Description                                                          |
|----------|----------|----------------------------------------------------------------------|
| `cmd`    | yes      | RS232 command: `Main.Power`, `Main.IR`, `Main.Source`, `Main.Model`… |
| `op`     | yes      | Operator: `=` (set), `+` (increment), `-` (decrement), `?` (query)  |
| `value`  | no       | Value to set. For `Main.IR`, accepts plain-text names like `volume_up` |
| `nowait` | no       | Set to `true` for fire-and-forget (returns immediately)              |

**Response:**
```json
{"command": "Main.Power=On", "response": "Main.Power=On"}
```

**With nowait:**
```json
{"command": "Main.IR=136", "response": "ok (nowait)"}
```

### GET /status

Returns serial port connection status.
```json
{"portPath": "/dev/cu.PL2303G-USBtoUART2140", "connected": true}
```

### GET /ir-commands

Returns the full IR command name-to-decimal-code mapping as JSON. Useful for discovering available commands.

## RS232 Protocol

NAD RS232 commands follow the format `\r<Command>\r` (carriage return before and after).

| Command           | Values      | Description              |
|-------------------|-------------|--------------------------|
| `Main.Power`      | On / Off    | Power on/off             |
| `Main.Source`     | 1–6         | Input source             |
| `Main.Mute`       | On / Off    | Mute                     |
| `Main.SpeakerA`   | On / Off    | Speaker A output         |
| `Main.SpeakerB`   | On / Off    | Speaker B output         |
| `Main.Dimmer`     | On / Off    | Display dimmer           |
| `Main.Sleep`      | 0–90        | Sleep timer (minutes)    |
| `Main.ToneDefeat` | On / Off    | Tone defeat              |
| `Main.Model`      | *(query)*   | Model number             |
| `Main.Version`    | *(query)*   | Firmware version         |
| `Main.IR`         | decimal code| Send IR remote command   |

**Note:** Volume is not available as a direct RS232 command. Use `Main.IR` with `volume_up` (136) or `volume_down` (140).

## Usage Examples

```bash
# Query model number
curl "http://localhost:8080/control?cmd=Main.Model&op=?"

# Power on
curl "http://localhost:8080/control?cmd=Main.Power&op==&value=On"

# Volume up (fire-and-forget for speed)
curl "http://localhost:8080/control?cmd=Main.IR&op==&value=volume_up&nowait=true"

# Volume down
curl "http://localhost:8080/control?cmd=Main.IR&op==&value=volume_down&nowait=true"

# Speaker A on via IR
curl "http://localhost:8080/control?cmd=Main.IR&op==&value=speaker_a"

# Mute
curl "http://localhost:8080/control?cmd=Main.Mute&op==&value=On"

# Change source to input 3
curl "http://localhost:8080/control?cmd=Main.Source&op==&value=3"

# Increment source
curl "http://localhost:8080/control?cmd=Main.Source&op=+"

# Query FM frequency
curl "http://localhost:8080/control?cmd=Tuner.FM.Frequency&op=?"

# Use decimal IR code directly
curl "http://localhost:8080/control?cmd=Main.IR&op==&value=206"

# List available IR commands
curl "http://localhost:8080/ir-commands"
```

## Home Assistant

### rest_command configuration

```yaml
rest_command:
  nad_power_on:
    url: "http://nad2go:8080/control?cmd=Main.Power&op==&value=On"
  nad_power_off:
    url: "http://nad2go:8080/control?cmd=Main.Power&op==&value=Off"
  nad_volume_up:
    url: "http://nad2go:8080/control?cmd=Main.IR&op==&value=volume_up&nowait=true"
  nad_volume_down:
    url: "http://nad2go:8080/control?cmd=Main.IR&op==&value=volume_down&nowait=true"
  nad_mute:
    url: "http://nad2go:8080/control?cmd=Main.Mute&op==&value=On"
  nad_unmute:
    url: "http://nad2go:8080/control?cmd=Main.Mute&op==&value=Off"
  nad_source:
    url: "http://nad2go:8080/control?cmd=Main.Source&op==&value={{ source }}"
```

### Automation example

```yaml
automation:
  - alias: "Turn on amplifier when TV starts"
    trigger:
      - platform: state
        entity_id: media_player.living_room_tv
        to: "playing"
    action:
      - service: rest_command.nad_power_on
      - delay: "00:00:02"
      - service: rest_command.nad_source
        data:
          source: "1"
```

## Run

### Direct

```bash
SERIAL_PORT=/dev/cu.PL2303G-USBtoUART2140 go run .
```

### Build and run

```bash
go build -o nad2go .
SERIAL_PORT=/dev/cu.PL2303G-USBtoUART2140 ./nad2go
```

### Docker

```bash
docker compose up --build
```

The `docker-compose.yml` maps the serial device into the container. Edit the `devices` section to match your adapter path.

## Project Structure

| File                | Description                              |
|---------------------|------------------------------------------|
| `server.go`         | HTTP server, endpoints, main entrypoint  |
| `serial.go`         | Serial port communication, auto-reconnect|
| `irmap.go`          | IR command name → decimal code mapping   |
| `Dockerfile`        | Multi-stage alpine build                 |
| `docker-compose.yml`| Container config with serial passthrough |

## License

MIT
