# pb - bastepin

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
debug = false
serve_path = "/p/"
upload_path = "./pastes/"
```

### Command-line flags

```bash
./pb --help

Usage:
  -b, --bind           address:port to run the server on (default: 0.0.0.0:3001)
  -c, --config         Path to a configuration file (default: config.toml)
  -s, --serve-path     Path to serve pastes from (default: /p/)
  -u, --upload-path    Path to store uploaded pastes (default: ./pastes/)
  --debug              Enable debug mode
```

Command-line flags override config file values.
