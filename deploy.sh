#!/bin/bash

# Deployment script for Insightsim backend on AWS EC2
# Usage: ./deploy.sh [options]

set -e  # Exit on error

# Configuration
EC2_HOST="${EC2_HOST:-}"
EC2_USER="${EC2_USER:-ubuntu}"
EC2_KEY="${EC2_KEY:-~/.ssh/id_rsa}"
APP_NAME="insightsim"
APP_DIR="/opt/insightsim"
SERVICE_NAME="insightsim"
DB_PATH="/opt/insightsim/data/insightsim.db"
PORT="${PORT:-8080}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
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

usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Deploy Insightsim backend and frontend to AWS EC2 instance.

Options:
    -h, --host HOST          EC2 instance hostname or IP (required)
    -u, --user USER          SSH user (default: ubuntu)
    -k, --key KEY            SSH private key path (default: ~/.ssh/id_rsa)
    -p, --port PORT          Application port (default: 8080)
    --skip-build             Skip building the application locally
    --skip-upload            Skip uploading files to EC2
    --skip-service           Skip creating systemd service
    --help                   Show this help message

Environment Variables:
    EC2_HOST                EC2 instance hostname or IP
    EC2_USER                SSH user
    EC2_KEY                 SSH private key path
    PORT                    Application port

Examples:
    $0 --host ec2-1-2-3-4.compute-1.amazonaws.com
    $0 -h 1.2.3.4 -u ec2-user -k ~/.ssh/my-key.pem
    EC2_HOST=1.2.3.4 $0

EOF
}

# Parse arguments
SKIP_BUILD=false
SKIP_UPLOAD=false
SKIP_SERVICE=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--host)
            EC2_HOST="$2"
            shift 2
            ;;
        -u|--user)
            EC2_USER="$2"
            shift 2
            ;;
        -k|--key)
            EC2_KEY="$2"
            shift 2
            ;;
        -p|--port)
            PORT="$2"
            shift 2
            ;;
        --skip-build)
            SKIP_BUILD=true
            shift
            ;;
        --skip-upload)
            SKIP_UPLOAD=true
            shift
            ;;
        --skip-service)
            SKIP_SERVICE=true
            shift
            ;;
        --help)
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

# Validate required parameters
if [ -z "$EC2_HOST" ]; then
    log_error "EC2_HOST is required. Set it via -h/--host option or EC2_HOST environment variable."
    usage
    exit 1
fi

# Expand tilde in key path
EC2_KEY="${EC2_KEY/#\~/$HOME}"

# Check if key file exists
if [ ! -f "$EC2_KEY" ]; then
    log_error "SSH key file not found: $EC2_KEY"
    exit 1
fi

# Check SSH connection
log_info "Testing SSH connection to $EC2_USER@$EC2_HOST..."
if ! ssh -i "$EC2_KEY" -o StrictHostKeyChecking=no -o ConnectTimeout=10 "$EC2_USER@$EC2_HOST" "echo 'Connection successful'" > /dev/null 2>&1; then
    log_error "Failed to connect to EC2 instance. Please check your credentials and network."
    exit 1
fi
log_info "SSH connection successful"

# Build backend application locally
if [ "$SKIP_BUILD" = false ]; then
    log_info "Building backend application..."
    if ! (cd backend && go build -o "../$APP_NAME" ./cmd/server); then
        log_error "Failed to build backend application"
        exit 1
    fi
    log_info "Backend build successful"
    
    # Build frontend if available
    if [ -d "frontend" ]; then
        log_info "Building frontend application..."
        if ! command -v node &> /dev/null; then
            log_warn "Node.js is not installed. Frontend will not be built."
            log_warn "Install Node.js to enable frontend deployment: https://nodejs.org/"
        elif ! command -v npm &> /dev/null; then
            log_warn "npm is not installed. Frontend will not be built."
        else
            log_info "Installing frontend dependencies..."
            (cd frontend && npm install)
            log_info "Frontend dependencies installed"
            # Note: We don't build frontend for production here, we'll run dev server on EC2
            # If you want production build, uncomment: (cd frontend && npm run build)
        fi
    else
        log_warn "Frontend directory not found, skipping frontend build"
    fi
else
    log_warn "Skipping build step"
fi

# Create deployment package
log_info "Creating deployment package..."
TEMP_DIR=$(mktemp -d)
DEPLOY_PACKAGE="$TEMP_DIR/deploy.tar.gz"

# Copy necessary files
cp "$APP_NAME" "$TEMP_DIR/"
cp -r raw_data "$TEMP_DIR/" 2>/dev/null || log_warn "raw_data directory not found, skipping"
cp backend/go.mod backend/go.sum "$TEMP_DIR/" 2>/dev/null || log_warn "go.mod/go.sum not found, skipping"
cp -r backend/cmd "$TEMP_DIR/" 2>/dev/null || log_warn "cmd directory not found, skipping"
cp -r backend/internal "$TEMP_DIR/" 2>/dev/null || log_warn "internal directory not found, skipping"
cp config.json "$TEMP_DIR/" 2>/dev/null || cp config.json.example "$TEMP_DIR/config.json" 2>/dev/null || log_warn "config.json not found, will use defaults"

# Copy frontend if available
if [ -d "frontend" ]; then
    log_info "Including frontend in deployment package..."
    cp -r frontend "$TEMP_DIR/" 2>/dev/null || log_warn "Failed to copy frontend directory"
fi

# Create deployment script
cat > "$TEMP_DIR/setup.sh" << 'SETUP_EOF'
#!/bin/bash
set -e

APP_NAME="insightsim"
APP_DIR="/opt/insightsim"
BACKEND_SERVICE_NAME="insightsim-backend"
FRONTEND_SERVICE_NAME="insightsim-frontend"
DB_PATH="/opt/insightsim/data/insightsim.db"
LOG_FILE="/opt/insightsim/logs/out.log"
PORT="${PORT:-8080}"
FRONTEND_PORT="${FRONTEND_PORT:-8086}"

# Create application directory
sudo mkdir -p "$APP_DIR/data"
sudo mkdir -p "$APP_DIR/logs"
sudo mkdir -p "$APP_DIR/frontend"

# Copy application
sudo cp "$APP_NAME" "$APP_DIR/"
sudo chmod +x "$APP_DIR/$APP_NAME"

# Copy raw_data if exists
if [ -d "raw_data" ]; then
    sudo cp -r raw_data "$APP_DIR/"
fi

# Copy frontend if exists
if [ -d "frontend" ]; then
    sudo cp -r frontend/* "$APP_DIR/frontend/"
    echo "Frontend files copied"
fi

# Create backend systemd service file
sudo tee "/etc/systemd/system/${BACKEND_SERVICE_NAME}.service" > /dev/null << EOF
[Unit]
Description=Insightsim Backend Service
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=$APP_DIR
ExecStart=$APP_DIR/$APP_NAME -db $DB_PATH -port $PORT
Restart=always
RestartSec=10
StandardOutput=append:$LOG_FILE
StandardError=append:$LOG_FILE
SyslogIdentifier=$BACKEND_SERVICE_NAME

# Environment variables
Environment="PORT=$PORT"

[Install]
WantedBy=multi-user.target
EOF

# Check if Node.js is installed for frontend
FRONTEND_AVAILABLE=false
if command -v node &> /dev/null && command -v npm &> /dev/null; then
    FRONTEND_AVAILABLE=true
    # Install PM2 globally if not available
    if ! command -v pm2 &> /dev/null; then
        echo "Installing PM2..."
        sudo npm install -g pm2
    fi
    
    # Install frontend dependencies
    if [ -d "$APP_DIR/frontend" ] && [ -f "$APP_DIR/frontend/package.json" ]; then
        echo "Installing frontend dependencies..."
        (cd "$APP_DIR/frontend" && sudo npm install)
    fi
    
    # Create frontend systemd service file
    sudo tee "/etc/systemd/system/${FRONTEND_SERVICE_NAME}.service" > /dev/null << EOF
[Unit]
Description=Insightsim Frontend Service
After=network.target ${BACKEND_SERVICE_NAME}.service
Requires=${BACKEND_SERVICE_NAME}.service

[Service]
Type=simple
User=root
WorkingDirectory=$APP_DIR/frontend
ExecStart=/usr/bin/npm run dev
Restart=always
RestartSec=10
StandardOutput=append:$LOG_FILE
StandardError=append:$LOG_FILE
SyslogIdentifier=$FRONTEND_SERVICE_NAME

# Environment variables
Environment="PORT=$FRONTEND_PORT"
Environment="NEXT_PUBLIC_API_URL=http://localhost:$PORT"

[Install]
WantedBy=multi-user.target
EOF
else
    echo "WARNING: Node.js/npm not found. Frontend service will not be created."
    echo "WARNING: Install Node.js to enable frontend: curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash - && sudo apt-get install -y nodejs"
fi

# Stop existing services if they exist (before reloading systemd)
echo "Stopping existing services if they exist..."
if sudo systemctl is-active --quiet "$BACKEND_SERVICE_NAME" 2>/dev/null; then
    echo "Stopping existing backend service..."
    sudo systemctl stop "$BACKEND_SERVICE_NAME" 2>/dev/null || true
    sleep 1
fi

if sudo systemctl is-active --quiet "$FRONTEND_SERVICE_NAME" 2>/dev/null; then
    echo "Stopping existing frontend service..."
    sudo systemctl stop "$FRONTEND_SERVICE_NAME" 2>/dev/null || true
    sleep 1
fi

# Also kill any processes that might be running on the ports
echo "Checking for processes on ports..."
if command -v lsof &> /dev/null; then
    # Kill backend port
    BACKEND_PID=$(sudo lsof -Pi :$PORT -sTCP:LISTEN -t 2>/dev/null || true)
    if [ -n "$BACKEND_PID" ]; then
        echo "Killing process $BACKEND_PID on backend port $PORT..."
        sudo kill -9 "$BACKEND_PID" 2>/dev/null || true
        sleep 1
    fi
    
    # Kill frontend port
    FRONTEND_PID=$(sudo lsof -Pi :$FRONTEND_PORT -sTCP:LISTEN -t 2>/dev/null || true)
    if [ -n "$FRONTEND_PID" ]; then
        echo "Killing process $FRONTEND_PID on frontend port $FRONTEND_PORT..."
        sudo kill -9 "$FRONTEND_PID" 2>/dev/null || true
        sleep 1
    fi
elif command -v netstat &> /dev/null; then
    # Fallback for systems without lsof
    BACKEND_PID=$(sudo netstat -tlnp 2>/dev/null | grep ":$PORT " | awk '{print $7}' | cut -d'/' -f1 | head -1 || true)
    if [ -n "$BACKEND_PID" ] && [ "$BACKEND_PID" != "-" ]; then
        echo "Killing process $BACKEND_PID on backend port $PORT..."
        sudo kill -9 "$BACKEND_PID" 2>/dev/null || true
        sleep 1
    fi
    
    FRONTEND_PID=$(sudo netstat -tlnp 2>/dev/null | grep ":$FRONTEND_PORT " | awk '{print $7}' | cut -d'/' -f1 | head -1 || true)
    if [ -n "$FRONTEND_PID" ] && [ "$FRONTEND_PID" != "-" ]; then
        echo "Killing process $FRONTEND_PID on frontend port $FRONTEND_PORT..."
        sudo kill -9 "$FRONTEND_PID" 2>/dev/null || true
        sleep 1
    fi
fi

# Reload systemd
sudo systemctl daemon-reload

# Enable and start backend service
sudo systemctl enable "$BACKEND_SERVICE_NAME"
sudo systemctl restart "$BACKEND_SERVICE_NAME"

# Wait a bit for backend to start
sleep 3

# Enable and start frontend service if available
if [ "$FRONTEND_AVAILABLE" = true ] && [ -d "$APP_DIR/frontend" ]; then
    sudo systemctl enable "$FRONTEND_SERVICE_NAME"
    sudo systemctl restart "$FRONTEND_SERVICE_NAME"
    echo "Frontend service started"
fi

echo "Deployment completed successfully"
echo ""
echo "Backend service status:"
sudo systemctl status "$BACKEND_SERVICE_NAME" --no-pager -l | head -10 || true

if [ "$FRONTEND_AVAILABLE" = true ]; then
    echo ""
    echo "Frontend service status:"
    sudo systemctl status "$FRONTEND_SERVICE_NAME" --no-pager -l | head -10 || true
fi

echo ""
echo "Combined logs are available at: $LOG_FILE"
echo "View logs with: sudo tail -f $LOG_FILE"
SETUP_EOF

chmod +x "$TEMP_DIR/setup.sh"

# Create tar.gz package
tar -czf "$DEPLOY_PACKAGE" -C "$TEMP_DIR" .
log_info "Deployment package created: $DEPLOY_PACKAGE"

# Upload to EC2
if [ "$SKIP_UPLOAD" = false ]; then
    log_info "Uploading files to EC2 instance..."
    scp -i "$EC2_KEY" -o StrictHostKeyChecking=no "$DEPLOY_PACKAGE" "$EC2_USER@$EC2_HOST:/tmp/deploy.tar.gz"
    log_info "Upload completed"
else
    log_warn "Skipping upload step"
fi

# Extract and setup on EC2
log_info "Setting up application on EC2..."
ssh -i "$EC2_KEY" -o StrictHostKeyChecking=no "$EC2_USER@$EC2_HOST" << EOF
    set -e
    
    # Extract package
    cd /tmp
    tar -xzf deploy.tar.gz
    rm deploy.tar.gz
    
    # Export PORT variable
    export PORT=$PORT
    
    # Run setup script
    chmod +x setup.sh
    sudo ./setup.sh
    
    # Cleanup
    rm -rf /tmp/setup.sh /tmp/$APP_NAME /tmp/raw_data /tmp/go.mod /tmp/go.sum /tmp/frontend 2>/dev/null || true
EOF

log_info "Deployment completed successfully!"

# Show service status
log_info "Checking service status..."
ssh -i "$EC2_KEY" -o StrictHostKeyChecking=no "$EC2_USER@$EC2_HOST" "sudo systemctl status $SERVICE_NAME --no-pager -l | head -20" || true

# Cleanup local temp files
rm -rf "$TEMP_DIR"

log_info ""
log_info "=========================================="
log_info "Deployment Summary:"
log_info "=========================================="
log_info "EC2 Host: $EC2_HOST"
log_info "Application: $APP_NAME"
log_info "Backend Service: ${APP_NAME}-backend"
log_info "Frontend Service: ${APP_NAME}-frontend"
log_info "Backend Port: $PORT"
log_info "Frontend Port: 8086 (default)"
log_info "Database: $DB_PATH"
log_info "Log File: /opt/insightsim/logs/out.log"
log_info ""
log_info "Useful commands:"
log_info "  Check backend status: ssh $EC2_USER@$EC2_HOST 'sudo systemctl status ${APP_NAME}-backend'"
log_info "  Check frontend status: ssh $EC2_USER@$EC2_HOST 'sudo systemctl status ${APP_NAME}-frontend'"
log_info "  View combined logs: ssh $EC2_USER@$EC2_HOST 'sudo tail -f /opt/insightsim/logs/out.log'"
log_info "  View backend logs: ssh $EC2_USER@$EC2_HOST 'sudo journalctl -u ${APP_NAME}-backend -f'"
log_info "  View frontend logs: ssh $EC2_USER@$EC2_HOST 'sudo journalctl -u ${APP_NAME}-frontend -f'"
log_info "  Restart backend: ssh $EC2_USER@$EC2_HOST 'sudo systemctl restart ${APP_NAME}-backend'"
log_info "  Restart frontend: ssh $EC2_USER@$EC2_HOST 'sudo systemctl restart ${APP_NAME}-frontend'"
log_info "  Stop all: ssh $EC2_USER@$EC2_HOST 'sudo systemctl stop ${APP_NAME}-backend ${APP_NAME}-frontend'"
log_info "=========================================="
