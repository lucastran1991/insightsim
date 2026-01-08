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

Deploy Insightsim backend to AWS EC2 instance.

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

# Build application locally
if [ "$SKIP_BUILD" = false ]; then
    log_info "Building application..."
    if ! go build -o "$APP_NAME" ./cmd/server; then
        log_error "Failed to build application"
        exit 1
    fi
    log_info "Build successful"
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
cp go.mod go.sum "$TEMP_DIR/" 2>/dev/null || log_warn "go.mod/go.sum not found, skipping"
cp -r cmd "$TEMP_DIR/" 2>/dev/null || log_warn "cmd directory not found, skipping"
cp -r internal "$TEMP_DIR/" 2>/dev/null || log_warn "internal directory not found, skipping"

# Create deployment script
cat > "$TEMP_DIR/setup.sh" << 'SETUP_EOF'
#!/bin/bash
set -e

APP_NAME="insightsim"
APP_DIR="/opt/insightsim"
SERVICE_NAME="insightsim"
DB_PATH="/opt/insightsim/data/insightsim.db"
PORT="${PORT:-8080}"

# Create application directory
sudo mkdir -p "$APP_DIR/data"
sudo mkdir -p "$APP_DIR/logs"

# Copy application
sudo cp "$APP_NAME" "$APP_DIR/"
sudo chmod +x "$APP_DIR/$APP_NAME"

# Copy raw_data if exists
if [ -d "raw_data" ]; then
    sudo cp -r raw_data "$APP_DIR/"
fi

# Create systemd service file
sudo tee "/etc/systemd/system/${SERVICE_NAME}.service" > /dev/null << EOF
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
StandardOutput=journal
StandardError=journal
SyslogIdentifier=$SERVICE_NAME

# Environment variables
Environment="PORT=$PORT"

[Install]
WantedBy=multi-user.target
EOF

# Reload systemd
sudo systemctl daemon-reload

# Enable and start service
sudo systemctl enable "$SERVICE_NAME"
sudo systemctl restart "$SERVICE_NAME"

echo "Deployment completed successfully"
echo "Service status:"
sudo systemctl status "$SERVICE_NAME" --no-pager -l || true
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
    rm -rf /tmp/setup.sh /tmp/$APP_NAME /tmp/raw_data /tmp/go.mod /tmp/go.sum 2>/dev/null || true
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
log_info "Service: $SERVICE_NAME"
log_info "Port: $PORT"
log_info "Database: $DB_PATH"
log_info ""
log_info "Useful commands:"
log_info "  Check status: ssh $EC2_USER@$EC2_HOST 'sudo systemctl status $SERVICE_NAME'"
log_info "  View logs: ssh $EC2_USER@$EC2_HOST 'sudo journalctl -u $SERVICE_NAME -f'"
log_info "  Restart: ssh $EC2_USER@$EC2_HOST 'sudo systemctl restart $SERVICE_NAME'"
log_info "  Stop: ssh $EC2_USER@$EC2_HOST 'sudo systemctl stop $SERVICE_NAME'"
log_info "=========================================="
