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
    -s, --stop               Stop the PM2 process
    -R, --restart            Restart the PM2 process
    -l, --logs               Show PM2 logs
    -h, --help               Show this help message

Environment Variables:
    CONFIG_FILE              Config file path (default: config.json)
    PORT                     Server port (overrides config)
    DB_PATH                  Database file path (overrides config)
    BUILD_ON_START           Build before start (default: true)

Examples:
    $0                       # Start server with PM2
    $0 -p 3000                # Start on port 3000
    $0 --no-build             # Start without building
    $0 --stop                 # Stop PM2 process
    $0 --restart              # Restart PM2 process
    $0 --logs                 # View logs
    PORT=3000 DB_PATH=dev.db $0

EOF
}

# Parse arguments
SKIP_BUILD=false
RUN_ONLY=false
ACTION="start"

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
        -s|--stop)
            ACTION="stop"
            shift
            ;;
        -R|--restart)
            ACTION="restart"
            shift
            ;;
        -l|--logs)
            ACTION="logs"
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

# Handle actions
if [ "$ACTION" = "stop" ]; then
    if pm2 describe "$APP_NAME" > /dev/null 2>&1; then
        log_info "Stopping $APP_NAME..."
        pm2 stop "$APP_NAME"
        pm2 delete "$APP_NAME"
        log_info "Stopped successfully"
    else
        log_warn "Process $APP_NAME not found in PM2"
    fi
    exit 0
fi

if [ "$ACTION" = "restart" ]; then
    if pm2 describe "$APP_NAME" > /dev/null 2>&1; then
        log_info "Restarting $APP_NAME..."
        pm2 restart "$APP_NAME"
        log_info "Restarted successfully"
        pm2 status "$APP_NAME"
    else
        log_error "Process $APP_NAME not found in PM2. Use './start.sh' to start it first."
        exit 1
    fi
    exit 0
fi

if [ "$ACTION" = "logs" ]; then
    if pm2 describe "$APP_NAME" > /dev/null 2>&1; then
        pm2 logs "$APP_NAME"
    else
        log_warn "Process $APP_NAME not found in PM2"
        if [ -f "out.log" ]; then
            log_info "Showing out.log file:"
            tail -f out.log
        else
            log_error "No logs found"
        fi
    fi
    exit 0
fi

# Check if Go is installed
log_step "Checking prerequisites..."
if ! command -v go &> /dev/null; then
    log_error "Go is not installed. Please install Go 1.19+ first."
    exit 1
fi
log_info "Go version: $(go version)"

# Check if PM2 is installed
if ! command -v pm2 &> /dev/null; then
    log_error "PM2 is not installed. Please install PM2 first."
    log_info "Install with: npm install -g pm2"
    exit 1
fi
log_info "PM2 version: $(pm2 --version)"

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

# Check if port is already in use and kill the process
log_step "Checking if port $PORT is available..."
if command -v lsof &> /dev/null; then
    PID=$(lsof -Pi :$PORT -sTCP:LISTEN -t 2>/dev/null)
    if [ -n "$PID" ]; then
        log_warn "Port $PORT is already in use by process $PID"
        log_info "Killing process $PID to free up port $PORT..."
        kill -9 "$PID" 2>/dev/null || true
        sleep 1
        
        # Verify port is now free
        if lsof -Pi :$PORT -sTCP:LISTEN -t >/dev/null 2>&1; then
            log_error "Failed to kill process on port $PORT"
            log_info "You can manually kill it: kill -9 $PID"
            exit 1
        else
            log_info "Port $PORT is now available"
        fi
    else
        log_info "Port $PORT is available"
    fi
elif command -v netstat &> /dev/null; then
    # Fallback for systems without lsof (some Linux distros)
    PID=$(netstat -tlnp 2>/dev/null | grep ":$PORT " | awk '{print $7}' | cut -d'/' -f1 | head -1)
    if [ -n "$PID" ] && [ "$PID" != "-" ]; then
        log_warn "Port $PORT is already in use by process $PID"
        log_info "Killing process $PID to free up port $PORT..."
        kill -9 "$PID" 2>/dev/null || true
        sleep 1
        log_info "Port $PORT should now be available"
    else
        log_info "Port $PORT is available"
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
log_step "Starting server with PM2..."
log_info "Database: $DB_PATH"
log_info "Port: $PORT"
log_info "Config: $CONFIG_FILE"
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

# Get absolute path for binary
BINARY_ABS=$(cd "$(dirname "$BINARY")" && pwd)/$(basename "$BINARY")
WORK_DIR=$(pwd)

# Check if PM2 process already exists
if pm2 describe "$APP_NAME" > /dev/null 2>&1; then
    log_warn "PM2 process '$APP_NAME' already exists"
    log_info "Stopping existing process..."
    pm2 stop "$APP_NAME" 2>/dev/null || true
    pm2 delete "$APP_NAME" 2>/dev/null || true
fi

# Start with PM2
log_info "Starting $APP_NAME with PM2..."
pm2 start "$BINARY_ABS" \
    --name "$APP_NAME" \
    --cwd "$WORK_DIR" \
    --log "$WORK_DIR/out.log" \
    --error "$WORK_DIR/out.log" \
    --output "$WORK_DIR/out.log" \
    --merge-logs \
    --time \
    -- \
    -config "$CONFIG_FILE" \
    -db "$DB_PATH" \
    -port "$PORT"

# Configure PM2 to write output to out.log and save
pm2 save --force > /dev/null 2>&1 || true

log_info ""
log_info "Server started with PM2!"
log_info ""
log_info "Useful commands:"
log_info "  View logs: pm2 logs $APP_NAME"
log_info "  View logs (file): tail -f out.log"
log_info "  Status: pm2 status"
log_info "  Stop: pm2 stop $APP_NAME or ./start.sh --stop"
log_info "  Restart: pm2 restart $APP_NAME or ./start.sh --restart"
log_info "  Delete: pm2 delete $APP_NAME"
log_info ""
log_info "Output will be written to: out.log"
log_info ""

# Show initial status
sleep 2
pm2 status "$APP_NAME"

# Verify process is running
if pm2 describe "$APP_NAME" > /dev/null 2>&1; then
    log_info ""
    log_info "✓ Process started successfully and is running in background"
    log_info "✓ Process will continue running even if this script exits"
    log_info "✓ PM2 will automatically restart the process if it crashes"
    log_info ""
    log_info "Process is alive and running. You can safely close this terminal."
    log_info ""
    
    # Exit successfully - process will continue running in PM2
    exit 0
else
    log_error "Failed to start process with PM2"
    exit 1
fi
