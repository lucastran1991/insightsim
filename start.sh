#!/bin/bash

# Start script for Insightsim backend
# Usage: ./start.sh [options]

set -e  # Exit on error

# Configuration
APP_NAME="insightsim"
BINARY_NAME="server"
CONFIG_FILE="${CONFIG_FILE:-config.json}"
BUILD_ON_START="${BUILD_ON_START:-true}"

# Load config from config.json if exists
if [ -f "$CONFIG_FILE" ]; then
    # Try to use jq if available
    if command -v jq &> /dev/null; then
        DB_PATH="${DB_PATH:-$(jq -r '.database.path // "insightsim.db"' "$CONFIG_FILE")}"
        PORT="${PORT:-$(jq -r '.server.port // "8080"' "$CONFIG_FILE")}"
    # Fallback to grep/sed if jq not available
    elif command -v python3 &> /dev/null; then
        DB_PATH="${DB_PATH:-$(python3 -c "import json; f=open('$CONFIG_FILE'); d=json.load(f); print(d.get('database', {}).get('path', 'insightsim.db'))" 2>/dev/null || echo "insightsim.db")}"
        PORT="${PORT:-$(python3 -c "import json; f=open('$CONFIG_FILE'); d=json.load(f); print(d.get('server', {}).get('port', '8080'))" 2>/dev/null || echo "8080")}"
    else
        # Simple grep fallback (less reliable)
        DB_PATH="${DB_PATH:-$(grep -o '"path"[[:space:]]*:[[:space:]]*"[^"]*"' "$CONFIG_FILE" | head -1 | sed 's/.*"path"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/' || echo "insightsim.db")}"
        PORT="${PORT:-$(grep -o '"port"[[:space:]]*:[[:space:]]*"[^"]*"' "$CONFIG_FILE" | head -1 | sed 's/.*"port"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/' || echo "8080")}"
    fi
else
    # Defaults if config file doesn't exist
    DB_PATH="${DB_PATH:-insightsim.db}"
    PORT="${PORT:-8080}"
fi

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Start Insightsim backend server.

Options:
    -c, --config FILE        Config file path (default: config.json)
    -p, --port PORT          Server port (overrides config)
    -d, --db PATH            Database file path (overrides config)
    -b, --no-build           Skip building the application
    -r, --run-only           Run without building (use existing binary)
    -h, --help               Show this help message

Environment Variables:
    CONFIG_FILE              Config file path (default: config.json)
    PORT                     Server port (overrides config)
    DB_PATH                  Database file path (overrides config)
    BUILD_ON_START           Build before start (default: true)

Examples:
    $0
    $0 -p 3000
    $0 -d myapp.db -p 8080
    $0 --no-build
    PORT=3000 DB_PATH=dev.db $0

EOF
}

# Parse arguments
SKIP_BUILD=false
RUN_ONLY=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -c|--config)
            CONFIG_FILE="$2"
            shift 2
            # Reload config after changing config file
            if [ -f "$CONFIG_FILE" ]; then
                if command -v jq &> /dev/null; then
                    DB_PATH="${DB_PATH:-$(jq -r '.database.path // "insightsim.db"' "$CONFIG_FILE")}"
                    PORT="${PORT:-$(jq -r '.server.port // "8080"' "$CONFIG_FILE")}"
                elif command -v python3 &> /dev/null; then
                    DB_PATH="${DB_PATH:-$(python3 -c "import json; f=open('$CONFIG_FILE'); d=json.load(f); print(d.get('database', {}).get('path', 'insightsim.db'))" 2>/dev/null || echo "insightsim.db")}"
                    PORT="${PORT:-$(python3 -c "import json; f=open('$CONFIG_FILE'); d=json.load(f); print(d.get('server', {}).get('port', '8080'))" 2>/dev/null || echo "8080")}"
                fi
            fi
            ;;
        -p|--port)
            PORT="$2"
            shift 2
            ;;
        -d|--db)
            DB_PATH="$2"
            shift 2
            ;;
        -b|--no-build)
            SKIP_BUILD=true
            shift
            ;;
        -r|--run-only)
            RUN_ONLY=true
            SKIP_BUILD=true
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Check if Go is installed
log_step "Checking prerequisites..."
if ! command -v go &> /dev/null; then
    log_error "Go is not installed. Please install Go 1.19+ first."
    exit 1
fi
log_info "Go version: $(go version)"

# Check if binary exists
BINARY_EXISTS=false
if [ -f "$BINARY_NAME" ] || [ -f "$APP_NAME" ]; then
    BINARY_EXISTS=true
fi

# Build application
if [ "$SKIP_BUILD" = false ]; then
    log_step "Building application..."
    if go build -o "$BINARY_NAME" ./cmd/server; then
        log_info "Build successful: $BINARY_NAME"
    else
        log_error "Build failed"
        exit 1
    fi
elif [ "$BINARY_EXISTS" = false ]; then
    log_error "Binary not found. Please build first or remove --no-build flag."
    log_info "Run: go build -o $BINARY_NAME ./cmd/server"
    exit 1
else
    log_info "Skipping build (using existing binary)"
fi

# Check if port is already in use
log_step "Checking if port $PORT is available..."
if command -v lsof &> /dev/null; then
    if lsof -Pi :$PORT -sTCP:LISTEN -t >/dev/null 2>&1; then
        log_warn "Port $PORT is already in use"
        log_info "You can:"
        log_info "  1. Use a different port: $0 -p 3000"
        log_info "  2. Kill the process using port $PORT"
        log_info "  3. Check what's using it: lsof -i :$PORT"
        exit 1
    fi
fi

# Create database directory if needed
DB_DIR=$(dirname "$DB_PATH")
if [ "$DB_DIR" != "." ] && [ ! -d "$DB_DIR" ]; then
    log_info "Creating database directory: $DB_DIR"
    mkdir -p "$DB_DIR"
fi

# Check if raw_data directory exists
if [ ! -d "raw_data" ]; then
    log_warn "raw_data directory not found. Some APIs may not work."
fi

# Start server
log_step "Starting server..."
log_info "Database: $DB_PATH"
log_info "Port: $PORT"
log_info ""
log_info "Server will start in the foreground."
log_info "Press Ctrl+C to stop the server."
log_info ""

# Determine which binary to use
if [ -f "$BINARY_NAME" ]; then
    BINARY="./$BINARY_NAME"
elif [ -f "$APP_NAME" ]; then
    BINARY="./$APP_NAME"
else
    log_error "Binary not found"
    exit 1
fi

# Start the server with config file
exec "$BINARY" -config "$CONFIG_FILE" -db "$DB_PATH" -port "$PORT"
