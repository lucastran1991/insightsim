# Deployment Guide

Hướng dẫn deploy Insightsim backend lên AWS EC2 instance.

## Prerequisites

1. **AWS EC2 Instance** đã được tạo và running
2. **SSH Key** để kết nối đến EC2 instance
3. **Go** đã được cài đặt trên máy local (để build application)
4. **Security Group** của EC2 instance đã mở port 8080 (hoặc port bạn muốn sử dụng)

## Quick Start

### 1. Cấu hình Security Group

Đảm bảo EC2 Security Group đã mở port cho application:

```bash
# Mở port 8080 từ bất kỳ đâu (hoặc chỉ từ IP của bạn)
# Inbound Rules:
#   Type: Custom TCP
#   Port: 8080
#   Source: 0.0.0.0/0 (hoặc IP cụ thể của bạn)
```

### 2. Deploy Application

```bash
# Cách 1: Sử dụng command line options
./deploy.sh --host ec2-1-2-3-4.compute-1.amazonaws.com

# Cách 2: Sử dụng environment variables
export EC2_HOST="ec2-1-2-3-4.compute-1.amazonaws.com"
export EC2_USER="ubuntu"
export EC2_KEY="~/.ssh/my-key.pem"
export PORT="8080"
./deploy.sh

# Cách 3: Kết hợp cả hai
./deploy.sh -h 1.2.3.4 -u ec2-user -k ~/.ssh/my-key.pem -p 8080
```

## Deployment Script Options

```bash
./deploy.sh [OPTIONS]

Options:
    -h, --host HOST          EC2 instance hostname or IP (required)
    -u, --user USER          SSH user (default: ubuntu)
    -k, --key KEY            SSH private key path (default: ~/.ssh/id_rsa)
    -p, --port PORT          Application port (default: 8080)
    --skip-build             Skip building the application locally
    --skip-upload            Skip uploading files to EC2
    --skip-service           Skip creating systemd service
    --help                   Show help message
```

## Deployment Process

Script sẽ thực hiện các bước sau:

1. **Build Application**: Build Go binary trên máy local
2. **Create Package**: Tạo deployment package với binary và các files cần thiết
3. **Upload**: Upload package lên EC2 instance
4. **Setup**: 
   - Extract package
   - Copy files vào `/opt/insightsim`
   - Tạo systemd service
   - Start service

## Application Structure trên EC2

```
/opt/insightsim/
├── insightsim          # Application binary
├── data/
│   └── insightsim.db   # SQLite database
├── logs/               # Log files (nếu có)
└── raw_data/           # Raw data files
```

## Systemd Service

Application được chạy như một systemd service với tên `insightsim`.

### Service Commands

```bash
# Check status
ssh user@ec2-host 'sudo systemctl status insightsim'

# View logs
ssh user@ec2-host 'sudo journalctl -u insightsim -f'

# Restart service
ssh user@ec2-host 'sudo systemctl restart insightsim'

# Stop service
ssh user@ec2-host 'sudo systemctl stop insightsim'

# Start service
ssh user@ec2-host 'sudo systemctl start insightsim'
```

### Service Configuration

Service file được tạo tại `/etc/systemd/system/insightsim.service`:

- **Auto-restart**: Service sẽ tự động restart nếu crash
- **Restart delay**: 10 seconds
- **Logs**: Logs được gửi đến systemd journal

## Verification

Sau khi deploy, kiểm tra service:

```bash
# 1. Check service status
ssh user@ec2-host 'sudo systemctl status insightsim'

# 2. Check if application is listening
ssh user@ec2-host 'sudo netstat -tlnp | grep 8080'

# 3. Test API endpoint
curl http://EC2_HOST:8080/health
```

## Troubleshooting

### Connection Issues

```bash
# Test SSH connection
ssh -i ~/.ssh/key.pem user@ec2-host

# Check if port is accessible
telnet EC2_HOST 8080
```

### Service Not Starting

```bash
# Check service logs
ssh user@ec2-host 'sudo journalctl -u insightsim -n 50'

# Check if port is already in use
ssh user@ec2-host 'sudo lsof -i :8080'

# Check file permissions
ssh user@ec2-host 'ls -la /opt/insightsim/'
```

### Database Issues

```bash
# Check database file
ssh user@ec2-host 'ls -lh /opt/insightsim/data/insightsim.db'

# Check database permissions
ssh user@ec2-host 'sudo chown root:root /opt/insightsim/data/insightsim.db'
```

## Manual Deployment

Nếu script tự động không hoạt động, bạn có thể deploy thủ công:

```bash
# 1. Build locally
go build -o insightsim ./cmd/server

# 2. Copy to EC2
scp -i ~/.ssh/key.pem insightsim user@ec2-host:/tmp/

# 3. SSH to EC2
ssh -i ~/.ssh/key.pem user@ec2-host

# 4. Setup on EC2
sudo mkdir -p /opt/insightsim/data
sudo cp /tmp/insightsim /opt/insightsim/
sudo chmod +x /opt/insightsim/insightsim

# 5. Create systemd service (copy from deploy.sh)
sudo nano /etc/systemd/system/insightsim.service

# 6. Start service
sudo systemctl daemon-reload
sudo systemctl enable insightsim
sudo systemctl start insightsim
```

## Updating Application

Để update application:

```bash
# Simply run deploy script again
./deploy.sh -h EC2_HOST

# Service will be automatically restarted
```

## Environment Variables

Bạn có thể set environment variables trong systemd service file:

```bash
# Edit service file
sudo nano /etc/systemd/system/insightsim.service

# Add under [Service] section:
Environment="PORT=8080"
Environment="DB_PATH=/opt/insightsim/data/insightsim.db"

# Reload and restart
sudo systemctl daemon-reload
sudo systemctl restart insightsim
```

## Security Considerations

1. **Firewall**: Chỉ mở port cần thiết trong Security Group
2. **SSH Key**: Bảo vệ SSH private key, không commit vào git
3. **Database**: Backup database thường xuyên
4. **Logs**: Monitor logs để phát hiện issues
5. **Updates**: Keep EC2 instance và application updated

## Backup

```bash
# Backup database
ssh user@ec2-host 'sudo cp /opt/insightsim/data/insightsim.db /opt/insightsim/data/insightsim.db.backup'

# Download backup
scp -i ~/.ssh/key.pem user@ec2-host:/opt/insightsim/data/insightsim.db.backup ./
```

## Monitoring

```bash
# Monitor service
watch -n 5 'ssh user@ec2-host "sudo systemctl status insightsim --no-pager"'

# Monitor logs
ssh user@ec2-host 'sudo journalctl -u insightsim -f'

# Monitor resource usage
ssh user@ec2-host 'top -p $(pgrep insightsim)'
```
