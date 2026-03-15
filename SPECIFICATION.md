# IMAP MCP Server Specification

## Overview
A read-only MCP server that provides IMAP email access over SSE (Server-Sent Events) for consumption by MCP clients.

## Architecture

### Components
1. **MCP Server** (Go) - Core server implementing MCP protocol over SSE
2. **IMAP Client** - Connects to remote IMAP servers
3. **SSE Transport** - Handles streaming connections over HTTP

### Technology Stack
- **Language**: Go
- **MCP Protocol**: Model Context Protocol
- **Transport**: SSE (Server-Sent Events)
- **Container**: Podman-compatible Containerfile
- **IMAP**: Standard IMAP4 (RFC 3501)

## API Endpoints

### HTTP/SSE Endpoints
- `POST /sse` - Establish SSE connection and receive events
- `POST /message` - Send messages to server (via JSON-RPC)
- `GET /health` - Health check endpoint

## MCP Tools

### 1. `imap_list_emails`
List emails in a mailbox with optional filtering.

**Parameters**:
- `mailbox` (string, default: "INBOX") - Mailbox name
- `limit` (integer, default: 50) - Maximum emails to return
- `start_index` (integer, default: 0) - Pagination offset

**Returns**: Array of email summaries (ID, subject, sender, date, preview)

### 2. `imap_read_email`
Fetch full email content by UID.

**Parameters**:
- `uid` (integer) - Email UID
- `mailbox` (string, default: "INBOX") - Mailbox name

**Returns**: Complete email (headers, body, attachments metadata)

### 3. `imap_search_emails`
Search emails by criteria.

**Parameters**:
- `query` (string) - Search criteria (FROM, SUBJECT, etc.)
- `mailbox` (string, default: "INBOX") - Mailbox name
- `limit` (integer, default: 50) - Maximum results

**Returns**: Array of matching email UIDs and summaries

### 4. `imap_list_mailboxes`
List available mailboxes/folders.

**Parameters**:
- `pattern` (string, optional) - Filter by pattern (e.g., "%")

**Returns**: Array of mailbox names with hierarchy info

## Configuration

### Environment Variables
```bash
IMAP_HOST       # IMAP server address (required)
IMAP_PORT       # IMAP port, default: 993 (SSL)
IMAP_USERNAME   # IMAP username (required)
IMAP_PASSWORD   # IMAP password (required)
IMAP_MAILBOX    # Default mailbox, default: INBOX
SERVER_PORT     # Server HTTP port, default: 8080
```

### Security Considerations
- Password stored in environment variables (never in code)
- IMAP connections use SSL/TLS (port 993)
- No write operations (read-only)
- Connection timeout limits

## Container Structure

```
s-container-imapsmcp/
├── Containerfile      # Podman build definition
├── specification.md   # This document
├── cmd/
│   └── server/
│       └── main.go    # Application entry point
├── internal/
│   ├── server/
│   │   └── server.go  # HTTP/SSE server implementation
│   ├── imap/
│   │   └── client.go  # IMAP client wrapper
│   └── mcp/
│       └── handlers.go # MCP tool handlers
├── go.mod
├── go.sum
└── README.md
```

## Containerfile (Podman)

```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server ./cmd/server

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /build/server .
EXPOSE 8080
ENV SERVER_PORT=8080
CMD ["./server"]
```

## SSE Event Format

### Server → Client Events
```json
// connection:established
{"type": "connection", "status": "established"}

// tool_result
{"type": "result", "tool": "imap_list_emails", "data": {...}}

// error
{"type": "error", "message": "IMAP connection failed"}
```

## MCP Protocol Integration

The server implements the MCP protocol over SSE:
- JSON-RPC 2.0 messages
- Tool registration and discovery
- Request/response pattern over SSE stream

## Implementation Plan

### Phase 1: Core Infrastructure
1. Set up Go module and project structure
2. Implement IMAP client wrapper with connection pooling
3. Create basic HTTP/SSE server skeleton

### Phase 2: MCP Integration
1. Implement MCP protocol handlers
2. Register tools with MCP server
3. Handle JSON-RPC requests

### Phase 3: Tool Implementations
1. `imap_list_emails` - Fetch email UIDs and summaries
2. `imap_read_email` - Fetch full email content
3. `imap_search_emails` - IMAP SEARCH command wrapper
4. `imap_list_mailboxes` - LIST command wrapper

### Phase 4: Containerization
1. Create Containerfile
2. Test with podman build/run
3. Add health checks

### Phase 5: Testing & Documentation
1. Unit tests for IMAP client
2. Integration tests with test IMAP server
3. README with usage examples
