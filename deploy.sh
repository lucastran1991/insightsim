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
ECOSYSTEM_FILE="ecosystem.config.js"
DB_PATH="/opt/insightsim/data/insightsim.db"
PORT="${PORT:-8080}"
FRONTEND_PORT="${FRONTEND_PORT:-8086}"

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

Deploy Insightsim backend and frontend to AWS EC2 instance using PM2.

Options:
    -h, --host HOST          EC2 instance hostname or IP (required)
    -u, --user USER          SSH user (default: ubuntu)
    -k, --key KEY            SSH private key path (default: ~/.ssh/id_rsa)
    -p, --port PORT          Application port (default: 8080)
    --skip-build             Skip building the application locally
    --skip-upload            Skip uploading files to EC2
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
        log_info "Building frontend application (Production)..."
        if ! command -v node &> /dev/null; then
            log_warn "Node.js is not installed. Frontend will not be built."
        elif ! command -v npm &> /dev/null; then
            log_warn "npm is not installed. Frontend will not be built."
        else
            log_info "Installing frontend dependencies..."
            (cd frontend && npm install)
            log_info "Building frontend..."
            if ! (cd frontend && npm run build); then
                log_error "Failed to build frontend"
                exit 1
            fi
            log_info "Frontend production build successful"
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
# We don't need all source code on server, just binary and assets
# But keeping config is good
cp config.json "$TEMP_DIR/" 2>/dev/null || cp config.json.example "$TEMP_DIR/config.json" 2>/dev/null || log_warn "config.json not found, will use defaults"

# Copy frontend if available
if [ -d "frontend" ]; then
    log_info "Including frontend in deployment package..."
    # Copy only necessary files for Next.js production run
    mkdir -p "$TEMP_DIR/frontend"
    cp frontend/package.json "$TEMP_DIR/frontend/"
    cp frontend/package-lock.json "$TEMP_DIR/frontend/"
    cp -r frontend/.next "$TEMP_DIR/frontend/"
    cp -r frontend/public "$TEMP_DIR/frontend/" 2>/dev/null || true
    cp frontend/next.config.js "$TEMP_DIR/frontend/" 2>/dev/null || true
fi

# Create Ecosystem file for PM2
cat > "$TEMP_DIR/$ECOSYSTEM_FILE" << EOF
module.exports = {
  apps: [
    {
      name: '${APP_NAME}-backend',
      script: './${APP_NAME}',
      args: '-db ${DB_PATH} -port ${PORT}',
      cwd: '${APP_DIR}',
      env: {
        PORT: '${PORT}'
      },
      log_date_format: 'YYYY-MM-DD HH:mm:ss Z',
      error_file: '${APP_DIR}/logs/backend-error.log',
      out_file: '${APP_DIR}/logs/backend-out.log',
      merge_logs: true
    },
    {
      name: '${APP_NAME}-frontend',
      script: 'npm',
      args: 'start',
      cwd: '${APP_DIR}/frontend',
      env: {
        PORT: '${FRONTEND_PORT}',
        NEXT_PUBLIC_API_URL: 'http://localhost:${PORT}'
      },
      log_date_format: 'YYYY-MM-DD HH:mm:ss Z',
      error_file: '${APP_DIR}/logs/frontend-error.log',
      out_file: '${APP_DIR}/logs/frontend-out.log',
      merge_logs: true
    }
  ]
};
EOF

# Create setup script
cat > "$TEMP_DIR/setup.sh" << SETUPEOF
#!/bin/bash
set -e

APP_DIR="/opt/insightsim"
ECOSYSTEM_FILE="ecosystem.config.js"

# Create directories
sudo mkdir -p "\$APP_DIR/data"
sudo mkdir -p "\$APP_DIR/logs"
sudo mkdir -p "\$APP_DIR/frontend"

# Change ownership to current user to allow PM2 to run without sudo issues
sudo chown -R \$USER:\$USER "\$APP_DIR"

# Copy files
cp insightsim "\$APP_DIR/"
chmod +x "\$APP_DIR/insightsim"
cp $ECOSYSTEM_FILE "\$APP_DIR/"

if [ -d "raw_data" ]; then
    cp -r raw_data "\$APP_DIR/"
fi

if [ -d "frontend" ]; then
    cp -r frontend/* "\$APP_DIR/frontend/"
    cp frontend/.next "\$APP_DIR/frontend/" -r 2>/dev/null || true
fi

# Install Node.js & PM2 if missing
if ! command -v node &> /dev/null; then
    echo "Installing Node.js..."
    curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -
    sudo apt-get install -y nodejs
fi

if ! command -v pm2 &> /dev/null; then
    echo "Installing PM2..."
    sudo npm install -g pm2
fi

# Install frontend production dependencies
if [ -d "\$APP_DIR/frontend" ]; then
    echo "Installing frontend production dependencies..."
    (cd "\$APP_DIR/frontend" && npm install --production)
fi

# Cleanup old systemd services if they exist
echo "Cleaning up old systemd services..."
for svc in insightsim insightsim-backend insightsim-frontend; do
    if sudo systemctl is-active --quiet \$svc; then
        sudo systemctl stop \$svc
        sudo systemctl disable \$svc
    fi
    if [ -f "/etc/systemd/system/\$svc.service" ]; then
        sudo rm "/etc/systemd/system/\$svc.service"
    fi
done
sudo systemctl daemon-reload

# Start processes with PM2
echo "Starting applications with PM2..."
cd "\$APP_DIR"
pm2 start \$ECOSYSTEM_FILE

# Save PM2 list and generate startup script
pm2 save
# Note: startup command usually requires sudo and manual copy/paste in interactive shell
# We try to run it automatically
sudo env PATH=\$PATH:/usr/bin /usr/lib/node_modules/pm2/bin/pm2 startup systemd -u \$USER --hp \$HOME || true

echo "Setup completed successfully!"
SETUPEOF

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
    cd /tmp
    tar -xzf deploy.tar.gz
    rm deploy.tar.gz
    chmod +x setup.sh
    ./setup.sh
    rm -rf setup.sh insightsim raw_data go.mod go.sum frontend ecosystem.config.js
EOF

log_info "Deployment completed successfully!"
log_info "You can monitor your app with: ssh $EC2_USER@$EC2_HOST 'pm2 status'"
log_info "Logs are available at: ssh $EC2_USER@$EC2_HOST 'pm2 logs'"

# Cleanup local temp files
rm -rf "$TEMP_DIR"
