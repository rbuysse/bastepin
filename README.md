# pb - Pastebin Service

A modern, feature-rich pastebin service written in Go with user authentication and syntax highlighting.

## Features

- **User Authentication**: Register and login to manage your pastes
- **Private Pastes**: Create private pastes that only you can view
- **Unlisted Pastes**: Create pastes accessible via direct link but not listed publicly
- **Paste Expiration**: Set TTL for pastes (10 min, 1 hour, 1 day, 1 week, 30 days)
- **Syntax Highlighting**: Support for 15+ programming languages
- **Paste Editing**: Edit your own pastes after creation
- **Anonymous Pastes**: Create pastes without logging in (view-only)
- **Content Deduplication**: Identical pastes are automatically deduplicated
- **Search**: Full-text search through your own pastes
- **API Keys**: Generate API keys for programmatic access
- **Admin Panel**: User management for administrators
- **Modern UI**: Dark theme with clean, intuitive interface
- **Database-backed**: SQLite for reliable persistence and user management

## Quick Start

```bash
# Build and run
go build
./pb

# Or run directly
go run .
```

Docker
```bash
just docker-build
just docker-run
```

The server will start on `http://0.0.0.0:3001` by default.

## Configuration

Create a `config.toml` file (see `config.toml.example`):

```toml
bind = "0.0.0.0:3001"
database_path = "./pastes.db"
debug = false
serve_path = "/p/"
```

### Command-line flags

```bash
./pb --help

Usage:
  -b, --bind           address:port to run the server on (default: 0.0.0.0:3001)
  -c, --config         Path to a configuration file (default: config.toml)
  -d, --database       Path to SQLite database file (default: ./pastebin.db)
  -s, --serve-path     Path to serve pastes from (default: /p/)
  --debug              Enable debug mode
```

Command-line flags override config file values.

## Usage

### Creating a Paste

1. Visit the homepage
2. Optionally log in to enable private pastes and editing
3. Select a language from the dropdown
4. Paste or type your content
5. Check "Private paste" if you want to restrict access (requires login)
6. Click "Create Paste"

### Editing a Paste

1. Log in to your account
2. View one of your pastes
3. Click "Edit" button
4. Make your changes and click "Save Changes"

### API Usage

```bash
# Upload paste (legacy - plain text)
curl -X POST http://localhost:3001/upload -d "Your paste content"

# Upload paste (JSON API with options)
curl -X POST http://localhost:3001/upload \
  -H "Content-Type: application/json" \
  -d '{"content":"print(\"hello\")","language":"python","is_private":false}'

# View paste (raw)
curl http://localhost:3001/p/PASTE_ID?raw=1

# Upload with API key
curl -X POST http://localhost:3001/upload \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"content":"print(\"hello\")","language":"python","expires_in":60}'
```

## API Keys

Generate API keys in the web interface under "API Keys". Use them for programmatic access:

```bash
# Create a paste with API key
curl -X POST http://localhost:3001/upload \
  -H "Authorization: Bearer pb_your_api_key_here" \
  -H "Content-Type: application/json" \
  -d '{"title":"My Paste","content":"Hello World","language":"text","expires_in":1440}'
```

## Admin Panel

Users can be granted admin privileges by directly adding a record to the `admins` table:

```sql
INSERT INTO admins (user_id) VALUES (1);
```

Admins can access the admin panel at `/admin` to manage users.
