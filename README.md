# IMAP MCP Server

A read-only MCP server that provides IMAP email access over SSE (Server-Sent Events) for consumption by MCP clients.

## Features

- List emails in mailboxes with pagination
- Read full email content by UID
- Search emails by criteria (FROM, SUBJECT, TO)
- List available mailboxes/folders
- MCP protocol over SSE transport
- Read-only IMAP access (no write operations)

## Configuration

Set the following environment variables:

```bash
IMAP_HOST       # IMAP server address (required)
IMAP_PORT       # IMAP port, default: 993 (SSL)
SERVER_PORT     # Server HTTP port, default: 2757
```

Create a Podman secret with your IMAP credentials in `user:password` format:

```bash
echo 'your-email@example.com:yourpassword' | podman secret create imap_userpass -
```

Note: Passwords can contain colons (`:`). The secret is parsed by splitting on the first colon only.

Create a Podman secret with your IMAP credentials in `user:password` format:

```bash
echo 'your-email@example.com:yourpassword' | podman secret create imap_userpass -
```

Note: Passwords can contain colons (`:`). The secret is parsed by splitting on the first colon only.

## Building

### Using Make

```bash
make container-build
```

## Running

### Create Podman secret

First, create the secret:

```bash
echo 'your-email@example.com:yourpassword' | podman secret create imap_userpass -
```

### Using Make

The Make target will check for the required secret and fail if it doesn't exist:

```bash
IMAP_HOST='example.com' make container-run
```

This will:
1. Verify the `imap_userpass` secret exists
2. Run the container with the secret mounted at `/run/secrets/imap_userpass`
3. Expose port 2757

## API Endpoints

### Health Check

```bash
GET /health
```

Response:
```json
{"status": "healthy"}
```

### SSE Connection

```bash
POST /sse
```

Establishes an SSE connection for receiving events.

### MCP Messages

```bash
POST /message
```

Send JSON-RPC 2.0 messages to the server.

## MCP Tools

### imap_list_emails

List emails in a mailbox with optional filtering.

**Parameters:**
- `mailbox` (string, default: "INBOX") - Mailbox name
- `limit` (integer, default: 50) - Maximum emails to return
- `start_index` (integer, default: 0) - Pagination offset

### imap_read_email

Fetch full email content by UID.

**Parameters:**
- `uid` (integer, required) - Email UID
- `mailbox` (string, default: "INBOX") - Mailbox name

### imap_search_emails

Search emails by criteria.

**Parameters:**
- `query` (string, required) - Search criteria (FROM, SUBJECT, TO, etc.)
- `mailbox` (string, default: "INBOX") - Mailbox name
- `limit` (integer, default: 50) - Maximum results

### imap_list_mailboxes

List available mailboxes/folders.

**Parameters:**
- `pattern` (string, optional) - Filter by pattern (e.g., "%")

## Security Considerations

- Credentials stored securely as Podman secrets (never in environment variables)
- Secret file read from `/run/secrets/imap_userpass` in `user:password` format
- Passwords can contain colons - only the first colon is used as delimiter
- IMAP connections use SSL/TLS (port 993)
- No write operations (read-only)
- Connection timeout limits

## License

MIT
