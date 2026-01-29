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
        PORT="${PORT:-$(jq -r '.server.port // "8888"' "$CONFIG_FILE")}"
    # Fallback to grep/sed if jq not available
    elif command -v python3 &> /dev/null; then
        DB_PATH="${DB_PATH:-$(python3 -c "import json; f=open('$CONFIG_FILE'); d=json.load(f); print(d.get('database', {}).get('path', 'insightsim.db'))" 2>/dev/null || echo "insightsim.db")}"
        PORT="${PORT:-$(python3 -c "import json; f=open('$CONFIG_FILE'); d=json.load(f); print(d.get('server', {}).get('port', '8888'))" 2>/dev/null || echo "8888")}"
    else
        # Simple grep fallback (less reliable)
        DB_PATH="${DB_PATH:-$(grep -o '"path"[[:space:]]*:[[:space:]]*"[^"]*"' "$CONFIG_FILE" | head -1 | sed 's/.*"path"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/' || echo "insightsim.db")}"
        PORT="${PORT:-$(grep -o '"port"[[:space:]]*:[[:space:]]*"[^"]*"' "$CONFIG_FILE" | head -1 | sed 's/.*"port"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/' || echo "8888")}"
    fi
else
    # Defaults if config file doesn't exist
    DB_PATH="${DB_PATH:-insightsim.db}"
    PORT="${PORT:-8888}"
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

Start Insightsim backend and frontend servers.

Options:
    -c, --config FILE        Config file path (default: config.json)
    -p, --port PORT          Server port (overrides config)
    -d, --db PATH            Database file path (overrides config)
    -b, --no-build           Skip building the application
    -r, --run-only           Run without building (use existing binary)
    -s, --stop               Stop all PM2 processes (backend and frontend)
    -R, --restart            Restart all PM2 processes
    -l, --logs               Show combined PM2 logs (from out.log)
    -h, --help               Show this help message

Environment Variables:
    CONFIG_FILE              Config file path (default: config.json)
    PORT                     Server port (overrides config)
    DB_PATH                  Database file path (overrides config)
    BUILD_ON_START           Build before start (default: true)

Examples:
    $0                       # Start backend and frontend with PM2
    $0 -p 3000                # Start backend on port 3000
    $0 --no-build             # Start without building backend
    $0 --stop                 # Stop all PM2 processes
    $0 --restart              # Restart all PM2 processes
    $0 --logs                 # View combined logs from out.log
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
                    PORT="${PORT:-$(jq -r '.server.port // "8888"' "$CONFIG_FILE")}"
                elif command -v python3 &> /dev/null; then
                    DB_PATH="${DB_PATH:-$(python3 -c "import json; f=open('$CONFIG_FILE'); d=json.load(f); print(d.get('database', {}).get('path', 'insightsim.db'))" 2>/dev/null || echo "insightsim.db")}"
                    PORT="${PORT:-$(python3 -c "import json; f=open('$CONFIG_FILE'); d=json.load(f); print(d.get('server', {}).get('port', '8888'))" 2>/dev/null || echo "8888")}"
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
BACKEND_PM2_NAME="${APP_NAME}-backend"
FRONTEND_PM2_NAME="${APP_NAME}-frontend"

if [ "$ACTION" = "stop" ]; then
    STOPPED_ANY=false
    if pm2 describe "$BACKEND_PM2_NAME" > /dev/null 2>&1; then
        log_info "Stopping backend ($BACKEND_PM2_NAME)..."
        pm2 stop "$BACKEND_PM2_NAME"
        pm2 delete "$BACKEND_PM2_NAME"
        log_info "Backend stopped successfully"
        STOPPED_ANY=true
    else
        log_warn "Backend process $BACKEND_PM2_NAME not found in PM2"
    fi
    
    if pm2 describe "$FRONTEND_PM2_NAME" > /dev/null 2>&1; then
        log_info "Stopping frontend ($FRONTEND_PM2_NAME)..."
        pm2 stop "$FRONTEND_PM2_NAME"
        pm2 delete "$FRONTEND_PM2_NAME"
        log_info "Frontend stopped successfully"
        STOPPED_ANY=true
    else
        log_warn "Frontend process $FRONTEND_PM2_NAME not found in PM2"
    fi
    
    if [ "$STOPPED_ANY" = false ]; then
        log_warn "No processes found to stop"
    fi
    exit 0
fi

if [ "$ACTION" = "restart" ]; then
    RESTARTED_ANY=false
    if pm2 describe "$BACKEND_PM2_NAME" > /dev/null 2>&1; then
        log_info "Restarting backend ($BACKEND_PM2_NAME)..."
        pm2 restart "$BACKEND_PM2_NAME"
        log_info "Backend restarted successfully"
        RESTARTED_ANY=true
    else
        log_error "Backend process $BACKEND_PM2_NAME not found in PM2. Use './start.sh' to start it first."
    fi
    
    if pm2 describe "$FRONTEND_PM2_NAME" > /dev/null 2>&1; then
        log_info "Restarting frontend ($FRONTEND_PM2_NAME)..."
        pm2 restart "$FRONTEND_PM2_NAME"
        log_info "Frontend restarted successfully"
        RESTARTED_ANY=true
    else
        log_warn "Frontend process $FRONTEND_PM2_NAME not found in PM2"
    fi
    
    if [ "$RESTARTED_ANY" = true ]; then
        pm2 status
        exit 0
    else
        exit 1
    fi
fi

if [ "$ACTION" = "logs" ]; then
    HAS_PROCESSES=false
    if pm2 describe "$BACKEND_PM2_NAME" > /dev/null 2>&1 || pm2 describe "$FRONTEND_PM2_NAME" > /dev/null 2>&1; then
        HAS_PROCESSES=true
    fi
    
    if [ "$HAS_PROCESSES" = true ]; then
        log_info "Showing logs from all processes (combined in out.log):"
        pm2 logs --lines 50
    else
        log_warn "No processes found in PM2"
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
    if (cd backend && go build -o "../$BINARY_NAME" ./cmd/server); then
        log_info "Build successful: $BINARY_NAME"
    else
        log_error "Build failed"
        exit 1
    fi
elif [ "$BINARY_EXISTS" = false ]; then
    log_error "Binary not found. Please build first or remove --no-build flag."
    log_info "Run: cd backend && go build -o ../$BINARY_NAME ./cmd/server"
    exit 1
else
    log_info "Skipping build (using existing binary)"
fi

# Note: Port checking and process killing is now done later, after PM2 process cleanup

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

# Start backend server
log_step "Starting backend server with PM2..."
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

# Check if PM2 processes already exist and kill them
BACKEND_PM2_NAME="${APP_NAME}-backend"
FRONTEND_PM2_NAME="${APP_NAME}-frontend"

log_step "Checking for existing processes..."

# Stop/delete from current user's PM2
if pm2 describe "$BACKEND_PM2_NAME" > /dev/null 2>&1; then
    log_warn "PM2 process '$BACKEND_PM2_NAME' already exists"
    log_info "Stopping and deleting existing backend process..."
    pm2 stop "$BACKEND_PM2_NAME" 2>/dev/null || true
    pm2 delete "$BACKEND_PM2_NAME" 2>/dev/null || true
fi
if pm2 describe "$FRONTEND_PM2_NAME" > /dev/null 2>&1; then
    log_warn "PM2 process '$FRONTEND_PM2_NAME' already exists"
    log_info "Stopping and deleting existing frontend process..."
    pm2 stop "$FRONTEND_PM2_NAME" 2>/dev/null || true
    pm2 delete "$FRONTEND_PM2_NAME" 2>/dev/null || true
fi

# Also stop/delete from root's PM2 (e.g. leftover from previous deploy with sudo)
# This prevents PM2 from immediately restarting the process when we kill by port
if [ "$(id -u)" != "0" ] && command -v sudo &> /dev/null; then
    if sudo pm2 describe "$BACKEND_PM2_NAME" > /dev/null 2>&1; then
        log_warn "Found '$BACKEND_PM2_NAME' in root's PM2 (from previous deploy?)"
        log_info "Stopping and deleting from root's PM2..."
        sudo pm2 stop "$BACKEND_PM2_NAME" 2>/dev/null || true
        sudo pm2 delete "$BACKEND_PM2_NAME" 2>/dev/null || true
    fi
    if sudo pm2 describe "$FRONTEND_PM2_NAME" > /dev/null 2>&1; then
        log_warn "Found '$FRONTEND_PM2_NAME' in root's PM2 (from previous deploy?)"
        log_info "Stopping and deleting from root's PM2..."
        sudo pm2 stop "$FRONTEND_PM2_NAME" 2>/dev/null || true
        sudo pm2 delete "$FRONTEND_PM2_NAME" 2>/dev/null || true
    fi
fi
log_info "Waiting for processes to release ports..."
sleep 3

# Also check for processes running on the backend port and kill them (all PIDs)
if command -v lsof &> /dev/null; then
    BACKEND_PIDS=$(lsof -Pi :$PORT -sTCP:LISTEN -t 2>/dev/null) || true
    if [ -n "$BACKEND_PIDS" ]; then
        log_warn "Found process(es) on backend port $PORT: $BACKEND_PIDS"
        for pid in $BACKEND_PIDS; do
            [ -n "$pid" ] || continue
            log_info "Killing process $pid..."
            kill -9 "$pid" 2>/dev/null || true
        done
        sleep 2
    fi
elif command -v netstat &> /dev/null; then
    for BACKEND_PID in $(netstat -tlnp 2>/dev/null | grep ":$PORT " | awk '{print $7}' | cut -d'/' -f1); do
        [ -n "$BACKEND_PID" ] && [ "$BACKEND_PID" != "-" ] || continue
        log_warn "Found process $BACKEND_PID on backend port $PORT"
        log_info "Killing process $BACKEND_PID..."
        kill -9 "$BACKEND_PID" 2>/dev/null || true
    done
    sleep 2
fi
# Fallback: use fuser to kill all processes on the port (Linux)
if command -v fuser &> /dev/null; then
    if fuser -n tcp "$PORT" &>/dev/null; then
        log_info "Freeing port $PORT with fuser..."
        fuser -k "$PORT/tcp" 2>/dev/null || true
        sleep 2
    fi
fi

# Check for frontend process on port 8086 (default Next.js port)
FRONTEND_PORT=8086
if command -v lsof &> /dev/null; then
    FRONTEND_PIDS=$(lsof -Pi :$FRONTEND_PORT -sTCP:LISTEN -t 2>/dev/null) || true
    if [ -n "$FRONTEND_PIDS" ]; then
        log_warn "Found process(es) on frontend port $FRONTEND_PORT: $FRONTEND_PIDS"
        for pid in $FRONTEND_PIDS; do
            [ -n "$pid" ] || continue
            log_info "Killing process $pid..."
            kill -9 "$pid" 2>/dev/null || true
        done
        sleep 2
    fi
elif command -v netstat &> /dev/null; then
    for FRONTEND_PID in $(netstat -tlnp 2>/dev/null | grep ":$FRONTEND_PORT " | awk '{print $7}' | cut -d'/' -f1); do
        [ -n "$FRONTEND_PID" ] && [ "$FRONTEND_PID" != "-" ] || continue
        log_warn "Found process $FRONTEND_PID on frontend port $FRONTEND_PORT"
        log_info "Killing process $FRONTEND_PID..."
        kill -9 "$FRONTEND_PID" 2>/dev/null || true
    done
    sleep 2
fi
if command -v fuser &> /dev/null; then
    if fuser -n tcp "$FRONTEND_PORT" &>/dev/null; then
        log_info "Freeing port $FRONTEND_PORT with fuser..."
        fuser -k "$FRONTEND_PORT/tcp" 2>/dev/null || true
        sleep 2
    fi
fi

log_info "Process cleanup completed"

# Verify ports are now free (retry once after 2s - kernel may delay releasing port)
log_step "Verifying ports are available..."
BACKEND_PORT_FREE=false
for attempt in 1 2; do
    if ! command -v lsof &> /dev/null || ! lsof -Pi :$PORT -sTCP:LISTEN -t >/dev/null 2>&1; then
        BACKEND_PORT_FREE=true
        break
    fi
    if [ "$attempt" = "2" ]; then
        log_error "Port $PORT is still in use after cleanup. Try: sudo pm2 list && sudo pm2 delete insightsim-backend; sudo lsof -i :$PORT; sudo fuser -k $PORT/tcp"
        exit 1
    fi
    log_warn "Port $PORT still in use, waiting 2s..."
    sleep 2
done
[ "$BACKEND_PORT_FREE" = true ] && log_info "Backend port $PORT is available"

# Start backend with PM2
log_info "Starting backend ($BACKEND_PM2_NAME) with PM2..."
pm2 start "$BINARY_ABS" \
    --name "$BACKEND_PM2_NAME" \
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

# Wait a bit for backend to start
sleep 2

# Check if Node.js and npm are available for frontend
log_step "Checking frontend prerequisites..."
if ! command -v node &> /dev/null; then
    log_warn "Node.js is not installed. Frontend will not be started."
    log_warn "Install Node.js to enable frontend: https://nodejs.org/"
    FRONTEND_AVAILABLE=false
else
    log_info "Node.js version: $(node --version)"
    if ! command -v npm &> /dev/null; then
        log_warn "npm is not installed. Frontend will not be started."
        FRONTEND_AVAILABLE=false
    else
        log_info "npm version: $(npm --version)"
        FRONTEND_AVAILABLE=true
    fi
fi

# Start frontend if available
if [ "$FRONTEND_AVAILABLE" = true ] && [ -d "frontend" ]; then
    log_step "Starting frontend with PM2..."
    
    # Check if frontend dependencies are installed
    if [ ! -d "frontend/node_modules" ]; then
        log_info "Installing frontend dependencies..."
        (cd frontend && npm install)
    fi
    
    FRONTEND_DIR="$WORK_DIR/frontend"
    
    # Start frontend with PM2
    log_info "Starting frontend ($FRONTEND_PM2_NAME) with PM2..."
    pm2 start npm \
        --name "$FRONTEND_PM2_NAME" \
        --cwd "$FRONTEND_DIR" \
        --log "$WORK_DIR/out.log" \
        --error "$WORK_DIR/out.log" \
        --output "$WORK_DIR/out.log" \
        --merge-logs \
        --time \
        -- run dev
    
    log_info "Frontend started successfully!"
else
    if [ "$FRONTEND_AVAILABLE" = false ]; then
        log_warn "Skipping frontend start (Node.js/npm not available)"
    elif [ ! -d "frontend" ]; then
        log_warn "Frontend directory not found, skipping frontend start"
    fi
fi

# Configure PM2 to write output to out.log and save
pm2 save --force > /dev/null 2>&1 || true

log_info ""
log_info "Services started with PM2!"
log_info ""
log_info "Useful commands:"
log_info "  View logs: pm2 logs (shows all processes)"
log_info "  View logs (file): tail -f out.log"
log_info "  Status: pm2 status"
log_info "  Stop backend: pm2 stop $BACKEND_PM2_NAME or ./start.sh --stop"
log_info "  Restart backend: pm2 restart $BACKEND_PM2_NAME or ./start.sh --restart"
if [ "$FRONTEND_AVAILABLE" = true ] && [ -d "frontend" ]; then
    log_info "  Stop frontend: pm2 stop $FRONTEND_PM2_NAME"
    log_info "  Restart frontend: pm2 restart $FRONTEND_PM2_NAME"
fi
log_info "  Delete all: pm2 delete all"
log_info ""
log_info "All logs are combined in: out.log"
log_info ""

# Show initial status
sleep 2
pm2 status

# Verify processes are running
BACKEND_RUNNING=false
FRONTEND_RUNNING=false

if pm2 describe "$BACKEND_PM2_NAME" > /dev/null 2>&1; then
    BACKEND_RUNNING=true
fi

if pm2 describe "$FRONTEND_PM2_NAME" > /dev/null 2>&1; then
    FRONTEND_RUNNING=true
fi

log_info ""
if [ "$BACKEND_RUNNING" = true ]; then
    log_info "✓ Backend started successfully and is running in background"
fi
if [ "$FRONTEND_RUNNING" = true ]; then
    log_info "✓ Frontend started successfully and is running in background"
fi
if [ "$BACKEND_RUNNING" = true ] || [ "$FRONTEND_RUNNING" = true ]; then
    log_info "✓ Processes will continue running even if this script exits"
    log_info "✓ PM2 will automatically restart processes if they crash"
    log_info ""
    log_info "Processes are alive and running. You can safely close this terminal."
    log_info ""
    
    # Exit successfully - processes will continue running in PM2
    exit 0
else
    log_error "Failed to start processes with PM2"
    exit 1
fi
