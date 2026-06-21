# Deployment Guide

**xiaozhi-esp32-server-golang**  
Production and local development deployment instructions.

---

## Prerequisites

### System Requirements

**Minimum (Development)**
- OS: Linux, macOS, or Windows with WSL2
- RAM: 2 GB
- CPU: 2 cores
- Disk: 10 GB (for Docker images + data)
- Network: Stable TCP/UDP connectivity

**Recommended (Production)**
- OS: Linux (Ubuntu 20.04+, Debian 11+)
- RAM: 8+ GB
- CPU: 4+ cores
- Disk: 100 GB (SSD recommended)
- Network: Gigabit Ethernet, low latency
- Backup: External storage for database snapshots

### Software

**All Deployments**
- Docker & Docker Compose (v2.0+)
- Git (to clone repository)

**Local Development Only**
- Go 1.24.2+ (if building from source)
- Node.js 18+ (for frontend development)
- npm 9+ or yarn

**Optional**
- Make (for build automation)
- jq (for JSON manipulation in scripts)
- curl / wget (for health checks)

---

## Quick Start: Docker Compose (Production)

### 1. Clone Repository

```bash
git clone https://github.com/hackers365/xiaozhi-esp32-server-golang.git
cd xiaozhi-esp32-server-golang
```

### 2. Configure Environment

Create `config/config.personal.yaml` (local overrides):

```yaml
# Key configurations to customize for your deployment

server:
  pprof:
    enable: false  # Set to true for debugging only

auth:
  enable: true

chat:
  max_idle_duration: 30000
  chat_max_silence_duration: 400
  realtime_mode: 4

manager:
  backend_url: "http://backend:8080"          # Internal hostname
  auth_token: "CHANGE_THIS_TO_RANDOM_STRING"  # Change this!
  endpoint_auth_token: "CHANGE_THIS_TOO"

# Logging
log:
  level: "info"       # Use "debug" only for troubleshooting
  stdout: true

# Redis (for production)
redis:
  host: "redis"
  port: 6379
  password: "CHANGE_PASSWORD"  # Change this!
  db: 0

# WebSocket
websocket:
  host: "0.0.0.0"
  port: 8989

# MQTT client (if connecting to external broker; leave disabled for embedded)
mqtt:
  enable: false

# MQTT server (embedded broker)
mqtt_server:
  enable: true
  listen_host: "0.0.0.0"
  listen_port: 2883
  username: "admin"
  password: "CHANGE_PASSWORD"  # Change this!
  enable_auth: false             # Enable in production

# UDP
udp:
  external_host: "YOUR_SERVER_IP"  # E.g., "192.168.1.100" or domain name
  external_port: 8990
  listen_host: "0.0.0.0"
  listen_port: 8990

# System prompt
system_prompt: "你是一个友好的AI助手，帮助用户解决问题。"

# Provider configurations (update with your API keys)
vad:
  provider: "silero"

asr:
  provider: "funasr"
  funasr:
    ws_url: "ws://funasr-server:10095"
    language: "zh"

llm:
  provider: "openai"
  openai:
    api_key: "YOUR_OPENAI_API_KEY"
    model: "gpt-3.5-turbo"

tts:
  provider: "edge_tts"
  edge_tts:
    voice: "zh-CN-XiaomoNeural"
```

### 3. Configure Docker Compose

Edit `docker/docker-compose.yml` or use as-is for defaults. Key services:

```yaml
services:
  main-server:
    # Streams audio pipeline
    ports:
      - "8989:8989"    # WebSocket
      - "2883:2883"    # MQTT
      - "8990:8990"    # UDP
    volumes:
      - ./config:/app/config
      - ./logs:/app/logs

  backend:
    # REST API + dashboard backend
    ports:
      - "8080:8080"
    environment:
      - DB_HOST=mysql
      - REDIS_HOST=redis

  frontend:
    # React dashboard (served by backend)
    ports:
      - "3000:3000"

  mysql:
    # Database (config, devices, users)
    ports:
      - "3306:3306"
    volumes:
      - mysql_data:/var/lib/mysql

  redis:
    # Cache & session store
    ports:
      - "6379:6379"

  funasr-server:
    # Speech recognition (optional)
    ports:
      - "10095:10095"
```

### 4. Start Services

```bash
cd docker
docker-compose -f docker-compose.yml up -d
```

### 5. Verify Deployment

**Check container status**:
```bash
docker-compose ps
# Output: all services should show "Up"
```

**Check main server health**:
```bash
curl http://localhost:8989/health
# Should return 200 OK (if health endpoint enabled)
```

**Check backend API**:
```bash
curl http://localhost:8080/api/health
# Should return JSON response
```

**Access dashboard**:
```
http://localhost:3000  (via frontend reverse proxy)
or
http://localhost:8080  (direct backend)
```

### 6. Initial Setup

1. **Open dashboard**: http://localhost:3000 (or http://YOUR_SERVER_IP:3000)

2. **Create admin user**:
   - Click "Register" (if first-time setup)
   - Or use default credentials from config

3. **Configure providers**:
   - Go to Admin → ASR Config
   - Enter FunASR endpoint, API keys
   - Repeat for TTS, LLM, VAD, etc.
   - Click "Save & Test"

4. **Activate first device**:
   - Go to Devices → Add Device
   - Enter device ID, name, agent assignment
   - Copy device activation token
   - On ESP32, configure with token

---

## Local Development Deployment

### Windows (WSL2)

**Prerequisites**:
```bash
# In WSL2 terminal
wsl --list --verbose
# Ensure you're using WSL2, not WSL1

# Install Docker Desktop for Windows
# Download: https://www.docker.com/products/docker-desktop/
# Ensure WSL2 integration is enabled in Docker Desktop settings
```

**Setup**:
```bash
# In PowerShell (Windows host)
.\docker-local.ps1 up
# Or manually:
cd docker
docker-compose -f docker-compose.local.yml up -d
```

**Access services**:
```bash
# Get WSL IP
wsl hostname -I

# Access main server
wscat -c ws://WSL_IP:8989

# Access dashboard
http://localhost:3000
```

### macOS / Linux

**Setup**:
```bash
chmod +x docker-local.sh
./docker-local.sh up

# Or manually:
cd docker
docker-compose -f docker-compose.local.yml up -d
```

**View logs**:
```bash
docker-compose -f docker-compose.local.yml logs -f main-server
docker-compose -f docker-compose.local.yml logs -f backend
```

**Access services**:
```bash
# Main server
wscat -c ws://localhost:8989

# Dashboard
http://localhost:3000

# Database
mysql -h 127.0.0.1 -u root -p
# Password: default_password (from compose)
```

### Building from Source (Optional)

**Go server**:
```bash
cd cmd/server
go mod download
go build -o xiaozhi_server main.go
./xiaozhi_server -c ../../config/config.yaml -manager-enable
```

**Manager backend**:
```bash
cd manager/backend
go mod download
go build -o backend main.go
./backend
```

**Manager frontend**:
```bash
cd manager/frontend
npm install
npm run dev          # Dev server (hot reload)
npm run build        # Production build
```

---

## Environment Configuration

### Key Environment Variables

| Variable | Purpose | Default | Example |
|----------|---------|---------|---------|
| `CONFIG_FILE` | Path to YAML config | `config/config.yaml` | `/etc/xiaozhi/config.yaml` |
| `LOG_LEVEL` | Logging verbosity | `info` | `debug` |
| `REDIS_URL` | Redis connection | `redis://redis:6379/0` | `redis://prod-redis:6379` |
| `DB_HOST` | Database hostname | `mysql` | `db.prod.internal` |
| `DB_PORT` | Database port | `3306` | `3306` |
| `DB_USER` | Database username | `root` | `xiaozhi_user` |
| `DB_PASSWORD` | Database password | *(required)* | *(secure)* |
| `JWT_SECRET` | JWT signing key | *(required)* | *(random 32+ chars)* |
| `MANAGER_AUTH_TOKEN` | Internal auth token | *(from config.yaml)* | *(random 32+ chars)* |

### Secrets Management

**Option 1: Environment Variables** (simple, local dev)
```bash
export OPENAI_API_KEY="sk-..."
export REDIS_PASSWORD="secure_password"
```

**Option 2: .env File** (local dev, not for production)
```bash
# Create .env
OPENAI_API_KEY=sk-...
DOUBAO_API_KEY=...

# Load in script
set -a
source .env
set +a
docker-compose up
```

**Option 3: Docker Secrets** (production, encrypted)
```bash
# Create secrets
echo "sk-..." | docker secret create openai_api_key -
echo "password" | docker secret create redis_password -

# Reference in docker-compose.yml
services:
  main-server:
    secrets:
      - openai_api_key
    environment:
      OPENAI_API_KEY_FILE: /run/secrets/openai_api_key
```

**Option 4: Kubernetes Secrets** (cloud production)
```bash
kubectl create secret generic xiaozhi-secrets \
  --from-literal=openai_api_key=sk-... \
  --from-literal=redis_password=...
```

**Best Practices**:
- Never commit secrets to Git
- Use `.gitignore` for `.env`, `config.personal.yaml`
- Rotate secrets quarterly
- Audit secret access logs
- Use least-privilege principle (limit which service needs which secret)

---

## Database Setup

### SQLite (Default, Development)

```yaml
# config.yaml
storage:
  type: "sqlite"
  path: "./data/xiaozhi.db"
```

Minimal setup; no additional configuration needed.

### MySQL (Production)

**Initialize database**:

```bash
# Via docker-compose (automatic on first run)
docker-compose up mysql

# Or manually:
mysql -h localhost -u root -p

CREATE DATABASE xiaozhi CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'xiaozhi'@'%' IDENTIFIED BY 'SECURE_PASSWORD';
GRANT ALL PRIVILEGES ON xiaozhi.* TO 'xiaozhi'@'%';
FLUSH PRIVILEGES;
```

**Configure in YAML**:

```yaml
# config.yaml
storage:
  type: "mysql"
  host: "mysql.prod.internal"
  port: 3306
  username: "xiaozhi"
  password: "SECURE_PASSWORD"
  database: "xiaozhi"
  max_connections: 20
  connection_timeout: "30s"
```

**Backups**:

```bash
# Daily backup script
#!/bin/bash
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="backups/xiaozhi_$TIMESTAMP.sql"

mysqldump -h $DB_HOST -u $DB_USER -p$DB_PASSWORD xiaozhi > $BACKUP_FILE
gzip $BACKUP_FILE

# Upload to S3
aws s3 cp "$BACKUP_FILE.gz" s3://backups/xiaozhi/

# Keep only last 7 days locally
find backups/ -name "xiaozhi_*.sql.gz" -mtime +7 -delete
```

**Migrations** (GORM auto-migrations):

```go
// manager/backend/database/db.go
// Automatically runs on startup; creates/updates tables
type Device struct { /* fields */ }
type User struct { /* fields */ }
// ... other models

// Run in init:
db.AutoMigrate(&Device{}, &User{}, &Config{}, ...)
```

---

## Health Checks & Monitoring

**Health endpoints**:
```bash
curl http://localhost:8080/api/health
curl http://localhost:8080/api/health/db
```

**Docker health checks**: Configured in docker-compose.yml. Each service includes healthcheck block for automatic restart on failure.

**Provider health** (dashboard): Admin → Health Checks → Run for ASR/TTS/LLM/VAD provider validation and latency measurement.

**Monitoring** (optional): Use Prometheus + Grafana for metrics collection and visualization. See **docs/deployment-advanced.md** for detailed setup.

---

## Scaling Considerations

**Single server**: Supports 1000-5000 concurrent devices. Increase ulimit and tune kernel parameters for optimal performance.

**Multi-server scaling**: Use load balancer (HAProxy/NGINX) with central MySQL + Redis for shared state. Each main-server remains stateless. See **docs/deployment-advanced.md** for detailed multi-region setup and performance tuning.

---

## Disaster Recovery

### Backup Strategy

**Daily backups**:
```bash
# Database
mysqldump -h $DB_HOST -u $DB_USER -p$DB_PASSWORD xiaozhi | \
  gzip > backups/db_$(date +%Y%m%d).sql.gz

# Config files
tar czf backups/config_$(date +%Y%m%d).tar.gz config/

# Upload to S3
aws s3 sync backups/ s3://xiaozhi-backups/
```

**Retention**:
- Daily backups: keep 7 days
- Weekly backups: keep 4 weeks
- Monthly backups: keep 12 months

### Restore Procedure

**Database**:
```bash
# Stop services
docker-compose down

# Restore from backup
gunzip < backups/db_20260620.sql.gz | \
  mysql -h $DB_HOST -u $DB_USER -p$DB_PASSWORD xiaozhi

# Start services
docker-compose up -d
```

**Verification**:
```bash
# Check table counts
mysql -h $DB_HOST -u $DB_USER -p$DB_PASSWORD xiaozhi -e \
  "SELECT table_name, table_rows FROM information_schema.tables WHERE table_schema='xiaozhi';"
```

---

## Security Hardening

### Network

1. **Firewall**:
```bash
# Only allow necessary ports
sudo ufw allow 22/tcp      # SSH
sudo ufw allow 80/tcp      # HTTP
sudo ufw allow 443/tcp     # HTTPS
sudo ufw allow 8989/tcp    # WebSocket
sudo ufw allow 2883/tcp    # MQTT
sudo ufw deny 3306/tcp     # MySQL (internal only)
sudo ufw enable
```

2. **TLS/SSL**:

```yaml
# MQTT TLS
mqtt_server:
  tls:
    enable: true
    port: 8883
    pem: "/etc/xiaozhi/certs/server.pem"
    key: "/etc/xiaozhi/certs/server.key"
```

```bash
# Generate self-signed cert
openssl req -x509 -newkey rsa:2048 -keyout server.key -out server.pem -days 365 -nodes
```

3. **Reverse Proxy** (NGINX):

```nginx
server {
    listen 443 ssl;
    server_name xiaozhi.example.com;

    ssl_certificate /etc/letsencrypt/live/xiaozhi.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/xiaozhi.example.com/privkey.pem;

    location / {
        proxy_pass http://localhost:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }

    location /api/ {
        proxy_pass http://localhost:8080;
    }
}
```

### Authentication & Authorization

```yaml
# config.yaml
auth:
  enable: true
  jwt_secret: "CHANGE_TO_RANDOM_32_CHAR_STRING"
  jwt_expiry: 24h
  bcrypt_cost: 12  # Higher = slower (more secure)

# Require strong passwords
# password_policy:
#   min_length: 12
#   require_uppercase: true
#   require_numbers: true
#   require_symbols: true
```

### API Security

```go
// Rate limiting
import "github.com/gin-gonic/gin"

// Per-IP rate limit
limiter := NewRateLimiter(100) // 100 requests/minute

router.Use(limiter.Middleware())

// CORS
router.Use(cors.New(cors.Config{
    AllowOrigins:     []string{"https://xiaozhi.example.com"},
    AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
    AllowCredentials: true,
}))

// NoSQL injection prevention
// Always use parameterized queries (GORM does this)
db.Where("email = ?", email).First(&user)
```

---

## Troubleshooting

| Issue | Diagnosis | Solution |
|-------|-----------|----------|
| DB connection fails | `docker-compose logs mysql` | Check credentials in config |
| Provider API fails | Check API key in config; test API manually | Verify key in `config.personal.yaml` |
| High latency | `docker stats` to check CPU | Profile with pprof; check network bandwidth |
| Device drops | `docker-compose logs main-server` | Check firewall (ufw status) and network stability |

---

## Maintenance

**Daily**: Monitor disk space; check error logs; verify backups.

**Weekly**: Review latency metrics; check security updates (Docker); run DB consistency check.

**Monthly**: Update Go/npm dependencies; rebuild images; rotate secrets.

**Upgrade**:
```bash
docker-compose exec mysql mysqldump xiaozhi > backup_$(date +%Y%m%d).sql
git pull origin main
docker-compose build && docker-compose up -d
curl http://localhost:8080/api/health
docker-compose logs -f main-server
```

---

## Cost Optimization

- Use spot instances for non-critical services
- Cache provider responses (Redis)
- Monitor API usage: `grep "provider.*call" logs/server.log | awk '{print $3}' | sort | uniq -c`
- Use cheaper LLM models for non-critical paths
- Rate limiting to prevent runaway costs

---

## Support & Resources

- **Documentation**: `docs/` folder in repository
- **Issues**: GitHub Issues (bugs, feature requests)
- **Community**: WeChat group (invite-only)
- **Commercial Support**: Contact hackers365

---

## Next Steps

1. **Deploy**: Follow Quick Start above
2. **Configure**: Update `config.personal.yaml` with your providers
3. **Test**: Activate a device and run test conversation
4. **Monitor**: Set up health checks and alerting
5. **Backup**: Enable automated database backups
6. **Scale**: Plan for growth (load testing, capacity planning)
