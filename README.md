# Insightsim Backend

Golang backend service cung cấp HTTP APIs để load và query timeseries data từ JSON feed vào SQLite database.

## Prerequisites

- **Go 1.19+** đã được cài đặt
- **Git** để clone repository

## Quick Start

### 1. Clone Repository

```bash
git clone https://github.com/lucastran1991/insightsim.git
cd insightsim
```

### 2. Install Dependencies

```bash
go mod download
```

### 3. Build Application

```bash
go build -o server ./cmd/server
```

Hoặc build với tên khác:

```bash
go build -o insightsim ./cmd/server
```

### 4. Run Server

**Cách 1: Sử dụng start script (Khuyến nghị)**

```bash
./start.sh
```

Script sẽ tự động:
- Check prerequisites (Go installed)
- Build application nếu cần
- Check port availability
- Start server với options phù hợp

**Với custom options:**
```bash
./start.sh -p 3000                    # Custom port
./start.sh -d mydatabase.db           # Custom database
./start.sh -d myapp.db -p 3000        # Both
./start.sh --no-build                 # Skip build, use existing binary
```

**Cách 2: Sử dụng binary đã build**

```bash
./server
```

**Cách 3: Sử dụng Go run (development)**

```bash
go run ./cmd/server/main.go
```

**Cách 4: Với custom options**

```bash
./server -db mydatabase.db -port 3000
```

### 5. Verify Server is Running

```bash
# Test health endpoint
curl http://localhost:8080/health

# Should return: OK
```

## Start Script Options

```bash
./start.sh [OPTIONS]

Options:
  -p, --port PORT          Server port (default: 8080)
  -d, --db PATH            Database file path (default: insightsim.db)
  -b, --no-build           Skip building the application
  -r, --run-only           Run without building (use existing binary)
  -h, --help               Show help message

Environment Variables:
  PORT                     Server port
  DB_PATH                  Database file path
  BUILD_ON_START           Build before start (default: true)
```

## Server Binary Options

```bash
./server [OPTIONS]

Options:
  -db string     Path to SQLite database file (default: "insightsim.db")
  -port string   Server port (default: "8080")
```

**Examples:**

```bash
# Default settings (port 8080, database: insightsim.db)
./server

# Custom database path
./server -db /path/to/database.db

# Custom port
./server -port 3000

# Both custom database and port
./server -db myapp.db -port 3000
```

## Development Workflow

### Run với Auto-reload (using air)

Cài đặt [air](https://github.com/cosmtrek/air) để auto-reload khi code thay đổi:

```bash
# Install air
go install github.com/cosmtrek/air@latest

# Run với air
air
```

### Run với Hot Reload (using nodemon alternative)

Sử dụng [CompileDaemon](https://github.com/githubnemo/CompileDaemon):

```bash
# Install
go install github.com/githubnemo/CompileDaemon@latest

# Run
CompileDaemon -command="./server"
```

## Project Structure

```
insightsim/
├── cmd/
│   └── server/
│       └── main.go              # Application entry point
├── internal/
│   ├── models/                  # Data models
│   ├── database/                # Database layer
│   ├── handlers/                # HTTP handlers
│   └── services/                # Business logic
├── raw_data/                    # Raw data files
├── go.mod                       # Go dependencies
├── go.sum                       # Go dependencies checksum
├── deploy.sh                    # Deployment script
├── API.md                       # API documentation
├── DEPLOYMENT.md                # Deployment guide
└── README.md                    # This file
```

## API Endpoints

Sau khi server chạy, các API endpoints có sẵn:

- `GET /health` - Health check
- `POST /api/load` - Load data from raw_data folder
- `POST /api/generate-dummy` - Generate dummy data
- `GET /api/timeseriesdata/{start}/{end}?tags=...` - Query timeseries data

Xem chi tiết trong [API.md](API.md).

## Example Usage

### 1. Start Server

**Using start script:**
```bash
./start.sh
```

**Or using binary directly:**
```bash
./server -db insightsim.db -port 8080
```

Output:
```
Database initialized at: insightsim.db
Server starting on port 8080
API endpoints:
  POST /api/load
  POST /api/generate-dummy
  GET  /api/timeseriesdata/{start}/{end}?tags=<tag1,tag2>
  GET  /health
```

### 2. Load Data

```bash
curl -X POST "http://localhost:8080/api/load" \
  -H "Content-Type: application/json"
```

### 3. Generate Dummy Data

```bash
curl -X POST "http://localhost:8080/api/generate-dummy" \
  -H "Content-Type: application/json"
```

### 4. Query Data

```bash
curl "http://localhost:8080/api/timeseriesdata/2025-12-01T00:00:00/2026-01-31T23:59:59/?tags=RP447628.RPSYSFEDFR001A"
```

## Database

- **Type**: SQLite
- **Default Location**: `insightsim.db` (trong project root)
- **Auto-creation**: Database và tables được tạo tự động khi server start lần đầu

## Troubleshooting

### Port Already in Use

```bash
# Check what's using the port
lsof -i :8080

# Kill the process
kill -9 <PID>

# Or use different port
./server -port 3000
```

### Database Locked

```bash
# Stop server first, then check
# Make sure no other process is using the database
```

### Build Errors

```bash
# Clean and rebuild
go clean
go mod tidy
go build -o server ./cmd/server
```

### Missing Dependencies

```bash
# Download dependencies
go mod download

# Verify
go mod verify
```

## Development Tips

1. **Use different database for development:**
   ```bash
   ./server -db dev.db
   ```

2. **Check logs:**
   Server logs được output trực tiếp ra console

3. **Test APIs:**
   Sử dụng `curl` hoặc Postman để test APIs

4. **Database location:**
   Database file được tạo trong cùng thư mục với binary hoặc path bạn chỉ định

## Next Steps

- Xem [API.md](API.md) để biết chi tiết về các API endpoints
- Xem [DEPLOYMENT.md](DEPLOYMENT.md) để deploy lên AWS EC2
- Sử dụng `deploy.sh` để deploy tự động

## License

[Add your license here]
