# Insightsim API Documentation

Golang backend service cung cấp HTTP APIs để load và query timeseries data từ JSON feed vào SQLite database.

## Table of Contents

- [Overview](#overview)
- [Base URL](#base-url)
- [Endpoints](#endpoints)
  - [Health Check](#health-check)
  - [Load Data](#load-data)
  - [Generate Dummy Data](#generate-dummy-data)
  - [Query Timeseries Data](#query-timeseries-data)
- [Data Format](#data-format)
- [Error Handling](#error-handling)
- [Examples](#examples)
- [Running the Server](#running-the-server)
  - [Local Development](#local-development)
  - [Deployment to AWS EC2](#deployment-to-aws-ec2)

## Overview

Service này cho phép:
- Load dữ liệu từ JSON feed vào SQLite database
- Query timeseries data với filtering theo date range và tags
- Trả về data theo format giống input JSON

## Base URL

```
http://localhost:8080
```

## Endpoints

### Health Check

Kiểm tra trạng thái server.

**Endpoint:** `GET /health`

**Request:**
```bash
curl http://localhost:8080/health
```

**Response:**
```
OK
```

**Status Codes:**
- `200 OK` - Server đang hoạt động

---

### Load Data

Load dữ liệu từ tất cả JSON files trong folder `raw_data/` vào database.

**Endpoint:** `POST /api/load`

**Request:**

API này không cần parameters. Nó sẽ tự động scan folder `raw_data/` và load tất cả các file `.json` trong đó.

**Request Example:**
```bash
curl -X POST "http://localhost:8080/api/load" \
  -H "Content-Type: application/json"
```

**Response:**

**Success (200 OK):**
```json
{
  "success": true,
  "message": "Data loaded successfully",
  "count": 1438,
  "files_count": 1
}
```

**Error (500):**
```json
{
  "success": false,
  "message": "Error message here"
}
```

**Response Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `success` | boolean | Trạng thái thành công hay không |
| `message` | string | Thông báo kết quả |
| `count` | integer | Tổng số lượng records đã load/update (chỉ có khi success) |
| `files_count` | integer | Số lượng JSON files đã xử lý (chỉ có khi success) |

**Status Codes:**
- `200 OK` - Load thành công
- `500 Internal Server Error` - Lỗi server (folder không tồn tại, parse lỗi, database error)

**Duplicate Handling:**

Khi có duplicate records (cùng tag và timestamp):
- **Nếu quality mới > quality cũ**: Cập nhật record với giá trị và quality mới
- **Nếu quality mới = quality cũ**: Cập nhật record (override)
- **Nếu quality mới < quality cũ**: Bỏ qua, giữ nguyên record cũ

**Notes:**
- API tự động scan folder `raw_data/` trong project root
- Chỉ xử lý các file có extension `.json` (case-insensitive)
- Mỗi file được xử lý trong một transaction riêng
- Nếu một file lỗi, toàn bộ quá trình sẽ dừng và trả về lỗi

---

### Generate Dummy Data

Generate dummy timeseries data cho tất cả tags từ file `raw_data/tag_list.json`.

**Endpoint:** `POST /api/generate-dummy`

**Request:**

API này không cần parameters. Nó sẽ tự động đọc tags từ `raw_data/tag_list.json` và generate data.

**Request Example:**
```bash
curl -X POST "http://localhost:8080/api/generate-dummy" \
  -H "Content-Type: application/json"
```

**Response:**

**Success (200 OK):**
```json
{
  "success": true,
  "message": "Dummy data generated successfully",
  "count": 17856000,
  "tags_count": 200
}
```

**Note:** Số lượng records trong response là ước tính. Với time range từ 2025-12-01 đến 2026-01-31 (2 tháng), mỗi tag sẽ có khoảng 89,280 records (1 record/phút).

**Error (500):**
```json
{
  "success": false,
  "message": "Error message here"
}
```

**Response Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `success` | boolean | Trạng thái thành công hay không |
| `message` | string | Thông báo kết quả |
| `count` | integer | Tổng số lượng records đã generate (chỉ có khi success) |
| `tags_count` | integer | Số lượng tags đã xử lý (chỉ có khi success) |

**Status Codes:**
- `200 OK` - Generate thành công
- `500 Internal Server Error` - Lỗi server (file không tồn tại, parse lỗi, database error)

**Data Specifications:**

- **Time Range**: 2025-12-01 00:00:00 đến 2026-01-31 23:59:59
- **Frequency**: 1 record mỗi phút cho mỗi tag
- **Value Generation**:
  - Base value: Random từ 1000-5000 cho mỗi tag
  - Value thay đổi giữa 2 record liên tiếp: < 30% (random -30% đến +30%)
  - Value được đảm bảo không âm
- **Quality**: Fixed value = 3

**Duplicate Handling:**

Tương tự như Load API, khi có duplicate records:
- Nếu quality mới >= quality cũ: Cập nhật record
- Nếu quality mới < quality cũ: Bỏ qua, giữ nguyên record cũ

**Notes:**
- Quá trình generate có thể mất nhiều thời gian do số lượng records lớn (~89,280 records/tag)
- Data được commit theo batches (mỗi 10,000 records) để tránh memory issues
- Progress được log mỗi 10 tags đã xử lý
- Nếu một tag lỗi, toàn bộ quá trình sẽ dừng và trả về lỗi

**Performance Considerations:**

Với ~200 tags và ~89,280 records/tag (2 tháng data), tổng số records có thể lên đến ~17.8 triệu records. Quá trình generate có thể mất vài phút đến vài chục phút tùy thuộc vào hardware.

---

### Query Timeseries Data

Query timeseries data từ database với filtering theo date range và tags.

**Endpoint:** `GET /api/timeseriesdata/{start}/{end}`

**Path Parameters:**

| Parameter | Type | Required | Description | Example |
|-----------|------|----------|-------------|---------|
| `start` | string | Yes | Start timestamp (ISO 8601 format) | `2024-01-01T00:00:00` |
| `end` | string | Yes | End timestamp (ISO 8601 format) | `2025-08-31T23:59:59` |

**Query Parameters:**

| Parameter | Type | Required | Description | Example |
|-----------|------|----------|-------------|---------|
| `tags` | string | No | Comma-separated list of tags | `RP447628.RPSYSFEDFR001A,RP447628.RPSYSFEDFR001B` |

**Request Examples:**

**Query với một tag:**
```bash
curl "http://localhost:8080/api/timeseriesdata/2024-01-01T00:00:00/2025-08-31T23:59:59/?tags=RP447628.RPSYSFEDFR001A"
```

**Query với nhiều tags:**
```bash
curl "http://localhost:8080/api/timeseriesdata/2024-01-01T00:00:00/2025-08-31T23:59:59/?tags=RP447628.RPSYSFEDFR001A,RP447628.RPSYSFEDFR001B"
```

**Query tất cả tags trong date range:**
```bash
curl "http://localhost:8080/api/timeseriesdata/2024-01-01T00:00:00/2025-08-31T23:59:59/"
```

**Response:**

**Success (200 OK):**
```json
{
  "result": {
    "RP447628.RPSYSFEDFR001A": [
      {
        "timestamp": "2025-01-01T07:00:48",
        "value": 2493.5625,
        "quality": 3
      },
      {
        "timestamp": "2025-01-01T07:01:49",
        "value": 2489.0625,
        "quality": 3
      }
    ]
  }
}
```

**Error (400 Bad Request):**
```json
{
  "error": "Invalid start time: unable to parse timestamp: ..."
}
```

**Response Structure:**

- `result` (object): Object chứa data grouped by tag
  - Key: Tag name (string)
  - Value: Array of data points
    - `timestamp` (string): ISO 8601 timestamp format
    - `value` (number): Giá trị số
    - `quality` (integer): Quality code

**Status Codes:**
- `200 OK` - Query thành công
- `400 Bad Request` - Invalid timestamp format hoặc start > end
- `404 Not Found` - Route không tồn tại
- `500 Internal Server Error` - Database error

**Notes:**
- Timestamps phải ở format ISO 8601: `YYYY-MM-DDTHH:MM:SS`
- Start time phải <= end time
- Nếu không có tags parameter, sẽ trả về tất cả tags trong date range
- Results được sắp xếp theo tag và timestamp
- Timestamps trong response được convert từ milliseconds về ISO format

---

## Data Format

### Input JSON Format

File JSON input có format:

```json
{
  "result": {
    "RP447628.RPSYSFEDFR001A": [
      {
        "quality": 3,
        "timestamp": "2025-01-01T23:59:12",
        "value": 2499.75
      }
    ]
  }
}
```

### Database Schema

Table: `insight_raws`

| Column | Type | Description |
|--------|------|-------------|
| `id` | INTEGER | Primary key, auto increment |
| `tag` | TEXT | Tag name (e.g., "RP447628.RPSYSFEDFR001A") |
| `timestamp` | INTEGER | Unix timestamp in milliseconds |
| `value` | REAL | Numeric value |
| `quality` | INTEGER | Quality code |
| UNIQUE(tag, timestamp) | - | Unique constraint |

---

## Error Handling

### Common Error Responses

**Invalid Timestamp Format:**
```json
{
  "error": "Invalid start time: unable to parse timestamp: 2024-01-01"
}
```

**Invalid Date Range:**
```json
{
  "error": "start time must be before or equal to end time"
}
```

**File Not Found:**
```json
{
  "success": false,
  "message": "failed to open file: open example.json: no such file or directory"
}
```

**Invalid JSON:**
```json
{
  "success": false,
  "message": "failed to decode JSON: invalid character '}' looking for beginning of value"
}
```

---

## Examples

### Complete Workflow

**1. Load data từ tất cả JSON files trong raw_data folder:**
```bash
curl -X POST "http://localhost:8080/api/load" \
  -H "Content-Type: application/json"
```

**Response:**
```json
{
  "success": true,
  "message": "Data loaded successfully",
  "count": 1438,
  "files_count": 1
}
```

**2. Generate dummy data cho tất cả tags:**
```bash
curl -X POST "http://localhost:8080/api/generate-dummy" \
  -H "Content-Type: application/json"
```

**Response:**
```json
{
  "success": true,
  "message": "Dummy data generated successfully",
  "count": 17856000,
  "tags_count": 200
}
```

**Note:** Quá trình này có thể mất vài phút đến vài chục phút tùy thuộc vào số lượng tags và hardware. Có thể chạy trong background hoặc sử dụng timeout lớn.

**3. Query data cho một ngày:**
```bash
curl "http://localhost:8080/api/timeseriesdata/2025-01-01T00:00:00/2025-01-01T23:59:59/?tags=RP447628.RPSYSFEDFR001A"
```

**Response:**
```json
{
  "result": {
    "RP447628.RPSYSFEDFR001A": [
      {
        "timestamp": "2025-01-01T07:00:48",
        "value": 2493.5625,
        "quality": 3
      },
      ...
    ]
  }
}
```

**4. Query data cho một giờ:**
```bash
curl "http://localhost:8080/api/timeseriesdata/2025-01-01T00:00:00/2025-01-01T01:00:00/?tags=RP447628.RPSYSFEDFR001A"
```

**5. Query tất cả tags trong date range:**
```bash
curl "http://localhost:8080/api/timeseriesdata/2024-01-01T00:00:00/2025-08-31T23:59:59/"
```

### Using jq for Better Output

**Count records:**
```bash
curl -s "http://localhost:8080/api/timeseriesdata/2025-01-01T00:00:00/2025-01-01T23:59:59/?tags=RP447628.RPSYSFEDFR001A" | \
  jq '.result."RP447628.RPSYSFEDFR001A" | length'
```

**Get first 3 records:**
```bash
curl -s "http://localhost:8080/api/timeseriesdata/2025-01-01T00:00:00/2025-01-01T23:59:59/?tags=RP447628.RPSYSFEDFR001A" | \
  jq '.result."RP447628.RPSYSFEDFR001A" | .[0:3]'
```

**List all tags:**
```bash
curl -s "http://localhost:8080/api/timeseriesdata/2024-01-01T00:00:00/2025-08-31T23:59:59/" | \
  jq '.result | keys'
```

---

## Running the Server

### Local Development

**Build:**
```bash
go build -o server ./cmd/server
```

**Run:**
```bash
./server -db insightsim.db -port 8080
```

**Flags:**
- `-db`: Path to SQLite database file (default: `insightsim.db`)
- `-port`: Server port (default: `8080`)

**Example:**
```bash
./server -db /path/to/database.db -port 3000
```

### Deployment to AWS EC2

Deploy backend lên AWS EC2 instance sử dụng deployment script.

**Prerequisites:**
- AWS EC2 instance đã được tạo và running
- SSH key để kết nối đến EC2
- Security Group đã mở port 8080 (hoặc port bạn sử dụng)

**Basic Usage:**
```bash
./deploy.sh --host ec2-1-2-3-4.compute-1.amazonaws.com
```

**With Custom Options:**
```bash
./deploy.sh -h 1.2.3.4 -u ec2-user -k ~/.ssh/my-key.pem -p 8080
```

**Using Environment Variables:**
```bash
export EC2_HOST="1.2.3.4"
export EC2_USER="ubuntu"
export EC2_KEY="~/.ssh/id_rsa"
export PORT="8080"
./deploy.sh
```

**Deployment Script Options:**
- `-h, --host HOST`: EC2 instance hostname or IP (required)
- `-u, --user USER`: SSH user (default: ubuntu)
- `-k, --key KEY`: SSH private key path (default: ~/.ssh/id_rsa)
- `-p, --port PORT`: Application port (default: 8080)
- `--skip-build`: Skip building the application locally
- `--skip-upload`: Skip uploading files to EC2
- `--skip-service`: Skip creating systemd service
- `--help`: Show help message

**What the Script Does:**
1. Builds application locally
2. Creates deployment package (binary + raw_data + configs)
3. Uploads to EC2 via SCP
4. Extracts and sets up on EC2
5. Creates systemd service
6. Starts service automatically

**After Deployment:**

Service được quản lý bởi systemd với tên `insightsim`:

```bash
# Check service status
ssh user@ec2-host 'sudo systemctl status insightsim'

# View logs
ssh user@ec2-host 'sudo journalctl -u insightsim -f'

# Restart service
ssh user@ec2-host 'sudo systemctl restart insightsim'

# Stop service
ssh user@ec2-host 'sudo systemctl stop insightsim'
```

**Application Location on EC2:**
- Binary: `/opt/insightsim/insightsim`
- Database: `/opt/insightsim/data/insightsim.db`
- Raw Data: `/opt/insightsim/raw_data/`

**Verify Deployment:**
```bash
# Test health endpoint
curl http://EC2_HOST:8080/health

# Should return: OK
```

Xem thêm chi tiết trong [DEPLOYMENT.md](DEPLOYMENT.md).

---

## Notes

- Server tự động tạo database và table nếu chưa tồn tại
- Data được upsert (cập nhật nếu đã tồn tại, thêm mới nếu chưa có)
- Timestamps được lưu trong database dưới dạng Unix milliseconds
- API response trả về timestamps ở format ISO 8601
- Database có index trên (tag, timestamp) để tối ưu query performance
