# ws-relay-go

Small Go WebSocket relay server for room-based apps.

## What it does

- JSON-based protocol with arbitrary `payload` relaying
- 3-digit room tokens
- Host-managed room config via `set_config`
- Up to 4 members per room by default
- Docker + GHCR deploy flow for a small VPS

## Configuration

| Env var | Default | Description |
| --- | --- | --- |
| `PORT` | `5001` | HTTP port for the server |
| `MAX_ROOM_SIZE` | `4` | Max members per room, minimum is `2` |

## Routes

| Path | Use |
| --- | --- |
| `/ws` | WebSocket endpoint |
| `/status` | JSON status with rooms, active connections, room size, and total capacity |
| `/` | Basic `OK` response |

Example `/status` response:

```json
{
  "name": "ws-relay-go",
  "status": "ok",
  "rooms": 2,
  "active_connections": 5,
  "max_room_size": 4,
  "total_capacity": 8
}
```

## Run locally

```bash
go run .
```

Custom port / room size:

```bash
PORT=8080 MAX_ROOM_SIZE=2 go run .
```

## Protocol

Every client message is JSON with a `type` field.

### Host a room

```json
{ "type": "host", "config": "<any string>" }
```

Response:

```json
{ "type": "token", "token": "472" }
```

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

Broadcast to others:

```json
{ "type": "message", "from": "<sender id>", "payload": { "action": "move", "x": 3, "y": 1 } }
```

### Update room config

Host-only message:

```json
{ "type": "set_config", "config": "<new config>" }
```

Broadcast:

```json
{ "type": "config_updated", "config": "<new config>" }
```

### Disconnect

```json
{ "type": "disconnect" }
```

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

```bash
docker build -t ws-relay-go .
docker run --rm -p 5001:5001 ws-relay-go
```

## GitHub deploy

Add these repository secrets:

- `SSH_KEY`
- `HOST`
- `USER`
- `PORT`

What it does:

- builds and pushes `ghcr.io/<owner>/<repo>:latest`
- SSHes into your server
- installs Docker if needed
- pulls the latest image
- runs the container on `${PORT}:5001`

If the GHCR package is private after first publish, change its visibility to public in GitHub.

## Reverse proxy

Point your proxy to the app port on the server, for example `127.0.0.1:5001`.

nginx for `/ws`:

```nginx
location /ws {
    proxy_pass         http://127.0.0.1:5001/ws;
    proxy_http_version 1.1;
    proxy_set_header   Upgrade    $http_upgrade;
    proxy_set_header   Connection $connection_upgrade;
    proxy_set_header   Host       $host;
    proxy_set_header   X-Real-IP  $remote_addr;
    proxy_read_timeout 3600s;
    proxy_send_timeout 3600s;
}
```

nginx for `/status`:

```nginx
location /status {
    proxy_pass http://127.0.0.1:5001/status;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
}
```

Apache needs `mod_proxy`, `mod_proxy_http`, `mod_proxy_wstunnel`, and usually `mod_headers`.

Apache for `/ws`:

```apache
ProxyPass        /ws  ws://127.0.0.1:5001/ws
ProxyPassReverse /ws  ws://127.0.0.1:5001/ws
```

Apache for `/status`:

```apache
ProxyPass        /status  http://127.0.0.1:5001/status
ProxyPassReverse /status  http://127.0.0.1:5001/status
```
