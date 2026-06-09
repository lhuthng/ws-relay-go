# ws-relay-go

`ws-relay-go` is a lightweight WebSocket relay server for small rooms. It uses a simple JSON protocol, supports up to 4 members per room by default, and is designed to be easy to run on a small VPS.

## Highlights

- JSON-based protocol with arbitrary `payload` relaying
- Room tokens generated per active host session
- Host-managed room config via `set_config`
- Ping/pong keepalive for long-lived connections
- Health endpoint at `/healthz`
- GitHub Actions deploy workflow for SSH-based server rollout

## Configuration

| Env var | Default | Description |
| --- | --- | --- |
| `PORT` | `5001` | HTTP port for the server |
| `MAX_ROOM_SIZE` | `4` | Maximum members per room, minimum accepted value is `2` |

## Local development

```bash
go run .
```

Or build a binary:

```bash
go build -o server .
./server
```

Custom settings:

```bash
PORT=8080 MAX_ROOM_SIZE=2 go run .
```

## HTTP endpoints

| Path | Purpose |
| --- | --- |
| `/ws` | WebSocket endpoint |
| `/healthz` | Health check endpoint |
| `/` | Basic liveness response |

## Protocol

Every client message is JSON and must include a `type` field.

### Host a room

```json
{ "type": "host", "config": "<any string>" }
```

Server response:

```json
{ "type": "token", "token": "472" }
```

The host receives a unique 3-digit token for the room. The `config` value is stored and returned to joiners.

### Join a room

```json
{ "type": "join", "token": "472" }
```

Successful response to the joiner:

```json
{ "type": "joined", "id": "a1b2c3d4", "config": "<host config>", "members": 2 }
```

Broadcast to existing members:

```json
{ "type": "member_joined", "from": "a1b2c3d4", "members": 2 }
```

Rejected responses:

```json
{ "type": "rejected", "reason": "not_found" }
{ "type": "rejected", "reason": "room_full" }
```

### Relay a message

```json
{ "type": "message", "payload": { "action": "move", "x": 3, "y": 1 } }
```

Broadcast to everyone else in the room:

```json
{ "type": "message", "from": "<sender id>", "payload": { "action": "move", "x": 3, "y": 1 } }
```

`payload` may be any valid JSON value.

### Update room config

Host-only message:

```json
{ "type": "set_config", "config": "<new config>" }
```

Broadcast to other members:

```json
{ "type": "config_updated", "config": "<new config>" }
```

### Disconnect

```json
{ "type": "disconnect" }
```

Clients may also disconnect by closing the socket.

If a non-host leaves:

```json
{ "type": "member_left", "from": "<id>", "members": 2 }
```

If the host leaves:

```json
{ "type": "room_closed", "reason": "host_left" }
```

### Error responses

```json
{ "type": "error", "reason": "already_in_room" }
{ "type": "error", "reason": "not_in_room" }
{ "type": "error", "reason": "not_host" }
{ "type": "error", "reason": "invalid_json" }
{ "type": "error", "reason": "unknown_type" }
```

## Message reference

| Direction | `type` | Description |
| --- | --- | --- |
| client -> server | `host` | Create a room |
| client -> server | `join` | Join a room by token |
| client -> server | `message` | Relay a payload to other room members |
| client -> server | `set_config` | Update room config as host |
| client -> server | `disconnect` | Gracefully close the connection |
| server -> client | `token` | Assigned room token |
| server -> client | `joined` | Join confirmation with state |
| server -> client | `member_joined` | Another member joined |
| server -> client | `message` | Relayed payload from another member |
| server -> client | `config_updated` | Room config changed |
| server -> client | `member_left` | A non-host member disconnected |
| server -> client | `room_closed` | Host disconnected and room closed |
| server -> client | `rejected` | Join failed |
| server -> client | `error` | Protocol or permission error |

## Docker

Build and run:

```bash
docker build -t ws-relay-go .
docker run --rm -p 5001:5001 ws-relay-go
```

## GitHub Actions deployment

The workflow at [`.github/workflows/deploy.yml`](/Volumes/SSD/Documents SSD/ws-relay-go/.github/workflows/deploy.yml) does two things on every push to `main`:

1. Runs `go test ./...`
2. Builds a Linux binary, uploads it over SSH, and restarts a systemd service on your server

Set these GitHub repository secrets:

- `SSH_KEY`: private SSH key used by GitHub Actions
- `HOST`: server hostname or IP
- `USER`: SSH username

The workflow currently assumes:

- Binary install path: `/usr/local/bin/ws-relay-go`
- Service name: `ws-relay-go`
- The SSH user can run `sudo install` and `sudo systemctl restart ws-relay-go`

If your server uses a different binary path or service name, edit the `REMOTE_PATH` and `SERVICE_NAME` values in [`.github/workflows/deploy.yml`](/Volumes/SSD/Documents SSD/ws-relay-go/.github/workflows/deploy.yml).

## Manual deployment

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o ws-relay-go .
scp ws-relay-go user@your-server:/tmp/ws-relay-go
ssh user@your-server "sudo install -m 0755 /tmp/ws-relay-go /usr/local/bin/ws-relay-go && sudo systemctl restart ws-relay-go"
```
