# Pulse

Real-time team messaging platform with end-to-end encryption, built as a learning project for Go and Tauri.

## Tech Stack

| Layer     | Technology              |
|-----------|-------------------------|
| Backend   | Go 1.25 (stdlib router) |
| Database  | PostgreSQL 17           |
| Cache     | Redis 7                 |
| WebSocket | nhooyr.io/websocket     |
| Frontend  | Tauri (WIP)             |

## Architecture

Pulse backend follows a **layered architecture** pattern:

```
Transport (HTTP handlers, WebSocket)
    ↓
Service (business logic)
    ↓
Repository (data access)
    ↓
PostgreSQL / Redis
```

Each layer only talks to the one below it. Domain types are shared across layers but contain no logic.

## Project Structure

```
pulse/
├── backend/
│   ├── cmd/server/          # Application entrypoint
│   ├── internal/
│   │   ├── config/          # Environment configuration
│   │   ├── database/        # Database connection
│   │   ├── domain/          # Domain types (User, Workspace, Channel...)
│   │   ├── repository/      # Data access interfaces + Postgres implementation
│   │   ├── service/         # Business logic
│   │   └── transport/
│   │       ├── http/        # HTTP handlers & middleware
│   │       └── ws/          # WebSocket hub & client management
│   ├── migrations/          # Goose SQL migrations
│   ├── Dockerfile           # Multi-stage production build
│   └── docker-compose.yml   # Dev-only (DB services)
├── frontend/                # Tauri app (WIP)
├── docker-compose.yml       # Full-stack deployment
└── .env.example             # Environment variables template
```

## Prerequisites

- **Go** 1.25+
- **Docker** & **Docker Compose**
- **Goose** (for running migrations locally): `go install github.com/pressly/goose/v3/cmd/goose@latest`

## Local Development

Start database services (from `backend/`):

```bash
cd backend
docker compose up -d
```

Run migrations:

```bash
cd backend
goose -dir migrations postgres \
  "postgres://pulse:pulse_dev_password@localhost:5432/pulse?sslmode=disable" up
```

Start the server:

```bash
cd backend
go run ./cmd/server
```

The API is available at `http://localhost:8080`.

Optional: start pgAdmin for database inspection:

```bash
cd backend
docker compose --profile dev up -d
# pgAdmin at http://localhost:5050
```

## Docker Deployment

Run the full stack with a single command from the project root:

```bash
# Copy and configure environment
cp .env.example .env
# Edit .env — at minimum change JWT_SECRET for production

# Build and start all services
docker compose up -d --build

# Check status
docker compose ps

# View backend logs
docker compose logs -f backend
```

This starts PostgreSQL, Redis, and the backend. Migrations run automatically on startup.

To include pgAdmin:

```bash
docker compose --profile dev up -d --build
```

To stop everything:

```bash
docker compose down
```

## API Endpoints

### Auth
| Method | Endpoint                  | Auth | Description        |
|--------|---------------------------|------|--------------------|
| POST   | `/api/v1/auth/register`   | No   | Register a user    |
| POST   | `/api/v1/auth/login`      | No   | Login              |

### Workspaces
| Method | Endpoint                                      | Auth | Description        |
|--------|-----------------------------------------------|------|--------------------|
| POST   | `/api/v1/workspaces`                          | Yes  | Create workspace   |
| GET    | `/api/v1/workspaces`                          | Yes  | List workspaces    |
| GET    | `/api/v1/workspaces/{id}`                     | Yes  | Get workspace      |
| PATCH  | `/api/v1/workspaces/{id}`                     | Yes  | Update workspace   |
| DELETE | `/api/v1/workspaces/{id}`                     | Yes  | Delete workspace   |
| POST   | `/api/v1/workspaces/{id}/members`             | Yes  | Add member         |
| DELETE | `/api/v1/workspaces/{id}/members/{uid}`       | Yes  | Remove member      |
| GET    | `/api/v1/workspaces/{id}/members`             | Yes  | List members       |

### Invites
| Method | Endpoint                                      | Auth | Description        |
|--------|-----------------------------------------------|------|--------------------|
| POST   | `/api/v1/workspaces/{id}/invites`             | Yes  | Create invite      |
| GET    | `/api/v1/workspaces/{id}/invites`             | Yes  | List invites       |
| DELETE | `/api/v1/workspaces/{id}/invites/{inviteId}`  | Yes  | Revoke invite      |
| GET    | `/api/v1/invites/{token}`                     | No   | Get invite info    |
| POST   | `/api/v1/invites/{token}/accept`              | Yes  | Accept invite      |

### Channels
| Method | Endpoint                                  | Auth | Description        |
|--------|-------------------------------------------|------|--------------------|
| POST   | `/api/v1/workspaces/{wid}/channels`       | Yes  | Create channel     |
| GET    | `/api/v1/workspaces/{wid}/channels`       | Yes  | List channels      |
| GET    | `/api/v1/channels/{id}`                   | Yes  | Get channel        |
| PATCH  | `/api/v1/channels/{id}`                   | Yes  | Update channel     |
| DELETE | `/api/v1/channels/{id}`                   | Yes  | Archive channel    |
| POST   | `/api/v1/channels/{id}/join`              | Yes  | Join channel       |
| POST   | `/api/v1/channels/{id}/members`           | Yes  | Add member         |
| DELETE | `/api/v1/channels/{id}/members/{uid}`     | Yes  | Remove member      |
| GET    | `/api/v1/channels/{id}/members`           | Yes  | List members       |

### Messages
| Method | Endpoint                              | Auth | Description        |
|--------|---------------------------------------|------|--------------------|
| POST   | `/api/v1/channels/{id}/messages`      | Yes  | Send message       |
| GET    | `/api/v1/channels/{id}/messages`      | Yes  | List messages      |
| PATCH  | `/api/v1/messages/{id}`               | Yes  | Edit message       |
| DELETE | `/api/v1/messages/{id}`               | Yes  | Delete message     |

### Direct Messages
| Method | Endpoint                                      | Auth | Description              |
|--------|-----------------------------------------------|------|--------------------------|
| POST   | `/api/v1/dm/conversations`                    | Yes  | Get or create DM         |
| GET    | `/api/v1/dm/conversations`                    | Yes  | List DM conversations    |
| POST   | `/api/v1/dm/conversations/{id}/messages`      | Yes  | Send DM                  |
| GET    | `/api/v1/dm/conversations/{id}/messages`      | Yes  | List DM messages         |
| PATCH  | `/api/v1/dm/messages/{id}`                    | Yes  | Edit DM                  |
| DELETE | `/api/v1/dm/messages/{id}`                    | Yes  | Delete DM                |

### Pulsemates (Friends)
| Method | Endpoint                                      | Auth | Description              |
|--------|-----------------------------------------------|------|--------------------------|
| POST   | `/api/v1/pulsemates/requests`                 | Yes  | Send friend request      |
| GET    | `/api/v1/pulsemates`                          | Yes  | List friends             |
| GET    | `/api/v1/pulsemates/requests/incoming`        | Yes  | Incoming requests        |
| GET    | `/api/v1/pulsemates/requests/outgoing`        | Yes  | Outgoing requests        |
| POST   | `/api/v1/pulsemates/requests/{id}/accept`     | Yes  | Accept request           |
| POST   | `/api/v1/pulsemates/requests/{id}/reject`     | Yes  | Reject request           |
| DELETE | `/api/v1/pulsemates/requests/{id}`            | Yes  | Cancel request           |
| DELETE | `/api/v1/pulsemates/{userId}`                 | Yes  | Remove friend            |

### WebSocket
| Endpoint | Auth            | Description              |
|----------|-----------------|--------------------------|
| `/ws`    | Query param JWT | Real-time events         |

### Health
| Method | Endpoint   | Description    |
|--------|------------|----------------|
| GET    | `/health`  | Health check   |

## Features

- [x] User registration & JWT authentication
- [x] Workspace CRUD with member management
- [x] Invite links for workspace joining
- [x] Channel CRUD with member management
- [x] Real-time messaging via WebSocket
- [x] Message editing & deletion
- [x] Direct messages
- [x] Pulsemates (friend system)
- [ ] End-to-end encryption
- [ ] Tauri desktop client
- [ ] File uploads
- [ ] Message reactions
- [ ] Thread replies

## License

[GPL-3.0](LICENSE)
