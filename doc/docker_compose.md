# Hướng Dẫn Triển Khai Docker Compose

## Tổng Quan

Dự án này sử dụng Docker Compose để triển khai container hóa, bao gồm các dịch vụ cốt lõi sau:

- **Dịch vụ cơ sở dữ liệu MySQL**: Lưu trữ dữ liệu
- **Dịch vụ chương trình chính**: Logic nghiệp vụ cốt lõi
- **Dịch vụ quản lý backend**: Dịch vụ giao diện API
- **Dịch vụ quản lý frontend**: Giao diện quản lý Web

## Hướng Dẫn Nhanh (Bổ Sung)

Phần này là tài liệu bổ sung cho `doc/docker.md`, giúp nhanh chóng lựa chọn và triển khai phương thức deployment.

### 1. Chọn Phương Thức Triển Khai

- Khuyến nghị: Docker Compose (bao gồm bảng điều khiển quản lý và dịch vụ đầy đủ)
- Đơn giản hóa: Docker container đơn (không có bảng điều khiển hoặc chế độ gọn nhẹ)

### 2. Lộ Trình Nhanh Docker Compose

1) Kéo mã nguồn hoặc chuẩn bị `docker-compose.yml`
2) Tham khảo phần "Chuẩn Bị File Cấu Hình" và "Khởi Động Dịch Vụ" bên dưới để hoàn tất cấu hình
3) Khởi động:

```bash
docker compose up -d
```

4) Địa chỉ bảng quản lý mặc định: `http://<IP-hoặc-tên-miền-máy-chủ>:8080/`

### 3. Docker Container Đơn (Bổ Sung)

Xây dựng hoặc kéo image theo `doc/docker.md` rồi chạy. Các khuyến nghị phổ biến:

- Mount thư mục `config/`, `logs/`, `storage/` làm data volume
- Expose cổng WebSocket / MQTT / UDP ra ngoài
- Khi cần bảng quản lý, bật tham số tương ứng hoặc dùng Compose

### 4. Hướng Dẫn Cấu Hình và Kiểm Tra

Sau khi khởi động, có thể dùng trình hướng dẫn cấu hình trong bảng quản lý để hoàn tất cấu hình engine, và sử dụng công cụ kiểm tra để chạy kiểm tra tính khả dụng và độ trễ của VAD/ASR/LLM/TTS, cũng như xác minh toàn bộ quy trình OTA.

### 5. Các Vấn Đề Thường Gặp

- Xung đột cổng: Kiểm tra tình trạng sử dụng cổng 8080/8989/2883/8990
- Cấu hình chưa có hiệu lực: Xác nhận đường dẫn mount data volume chính xác, khởi động lại container để áp dụng
- Vấn đề quyền truy cập: Trên Linux, chú ý quyền thư mục được mount và giới hạn SELinux

## Kiến Trúc Dịch Vụ

### 1. Dịch Vụ Cơ Sở Dữ Liệu MySQL (xiaozhi-mysql)

**Thông Tin Cấu Hình:**
- Image: `docker.jsdelivr.fyi/mysql:8.0`
- Ánh xạ cổng: `23306:3306`
- Tên cơ sở dữ liệu: `xiaozhi_admin`
- Tên người dùng: `root`
- Mật khẩu: `password`

**Tính Năng:**
- Sử dụng MySQL 8.0
- Cấu hình kiểm tra sức khỏe
- Lưu trữ dữ liệu bền vững

### 2. Dịch Vụ Chương Trình Chính (xiaozhi-main-server)

**Thông Tin Cấu Hình:**
- Image: `docker.jsdelivr.fyi/hackers365/xiaozhi_server:0.5`
- Ánh xạ cổng:
  - `8989:8989` - Dịch vụ WebSocket
  - `2882:2883` - Dịch vụ MQTT
  - `8888:8888/udp` - Dịch vụ UDP

**Quan Hệ Phụ Thuộc:**
- Phụ thuộc vào trạng thái sức khỏe của dịch vụ MySQL
- Phụ thuộc vào việc khởi động xong dịch vụ backend

**Hỗ Trợ File Cấu Hình:**
- Nhập file cấu hình tùy chỉnh qua volume mount
- Đường dẫn file cấu hình: `../../config:/workspace/config`

**Hỗ Trợ ten_vad:**
- Docker image đã bao gồm thư viện ten_vad (`/workspace/lib/ten-vad/`)
- Đường dẫn thư viện runtime đã được cấu hình tự động qua `LD_LIBRARY_PATH`

### 3. Dịch Vụ Quản Lý Backend (xiaozhi-backend)

**Thông Tin Cấu Hình:**
- Image: `docker.jsdelivr.fyi/hackers365/xiaozhi_manager_backend:0.5`
- Ánh xạ cổng: `8081:8080`

**Chức Năng:**
- Cung cấp RESTful API
- Quản lý thiết bị và người dùng

**Hỗ Trợ File Cấu Hình:**
- Nhập file cấu hình tùy chỉnh qua volume mount
- Đường dẫn file cấu hình: `../../manager/backend/config:/root/config`

### 4. Dịch Vụ Quản Lý Frontend (xiaozhi-frontend)

**Thông Tin Cấu Hình:**
- Image: `docker.jsdelivr.fyi/hackers365/xiaozhi_manager_frontend:0.5`
- Ánh xạ cổng: `8080:80`

**Chức Năng:**
- Giao diện quản lý Web (cổng vào nội bộ)
- Quản lý trạng thái thiết bị và cấu hình hệ thống

## Quy Trình Triển Khai

### 1. Chuẩn Bị Môi Trường

Đảm bảo hệ thống đã cài đặt Docker và Docker Compose:

```bash
docker --version
docker compose version
```

### 2. Chuẩn Bị File Cấu Hình

Đảm bảo các thư mục và file sau tồn tại:

```
xiaozhi-esp32-server-golang/
├─ docker/docker-composer/
│  └─ docker-compose.yml
├─ config/
│  ├─ config.yaml
│  ├─ config.json
│  └─ (các file cấu hình khác)
├─ logs/
│  └─ (thư mục log)
└─ manager/backend/config/
   ├─ config.yaml
   └─ (các file cấu hình khác)
```

**Hướng Dẫn Nhập File Cấu Hình:**
- File cấu hình chương trình chính được nhập qua volume mount `../../config:/workspace/config`
- File cấu hình backend được nhập qua volume mount `../../manager/backend/config:/root/config`

### 3. Khởi Động Dịch Vụ

**Phải vào thư mục `docker/docker-composer/` trước khi thực thi lệnh:**

```bash
cd docker/docker-composer/
docker compose up -d

docker compose ps
docker compose logs -f
```

### 4. Truy Cập Dịch Vụ

- Giao diện quản lý frontend: `http://<IP-hoặc-tên-miền-máy-chủ>:8080`
- Backend API: `http://localhost:8081`
- WebSocket: `ws://localhost:8989`
- MQTT: `localhost:2882`
- UDP: `localhost:8888`
- MySQL: `localhost:23306`

## Các Thao Tác Thường Dùng

```bash
cd docker/docker-composer/

docker compose ps

docker compose logs

docker compose logs -f main-server

docker compose restart

docker compose down

docker compose down -v

docker compose pull

docker compose up -d
```

## Cấu Hình Mạng

Dự án sử dụng mạng tùy chỉnh `xiaozhi-network`:

- MySQL: `mysql:3306`
- Backend: `backend:8080`
- Frontend: `frontend:80`
- Chương trình chính: `main-server:8989` (WebSocket) / `main-server:2883` (MQTT) / `main-server:8888` (UDP)

**Tóm Tắt Ánh Xạ Cổng:**
- 8080 → Giao diện quản lý frontend
- 8081 → Backend API
- 8989 → WebSocket
- 2882 → MQTT
- 8888 → UDP
- 23306 → MySQL

## Lưu Trữ Dữ Liệu Bền Vững

### Dữ Liệu MySQL

Được lưu trữ bền vững qua Docker volume `mysql_data`, dữ liệu không bị mất khi khởi động lại container.

### File Cấu Hình

- Cấu hình chương trình chính: `../../config:/workspace/config`
- Cấu hình backend: `../../manager/backend/config:/root/config`

Sau khi chỉnh sửa cấu hình, khởi động lại dịch vụ tương ứng để áp dụng:

```bash
cd docker/docker-composer/
docker compose restart main-server

docker compose restart backend
```

### File Log

- Log chương trình chính: `../../logs:/workspace/logs`

## Phương Pháp Nhập File Cấu Hình

### 1. Cấu Hình Chương Trình Chính

**Vị Trí:**
```
xiaozhi-esp32-server-golang/config/
├─ config.yaml
├─ config.json
├─ mqtt_config.json
└─ (các file cấu hình khác)
```

**Nhập:**
1) Đặt file cấu hình vào `config/`
2) Sau khi khởi động sẽ tự động mount vào container tại `/workspace/config/`
3) Sau khi chỉnh sửa, khởi động lại dịch vụ chương trình chính:

```bash
cd docker/docker-composer/
docker compose restart main-server
```

### 2. Cấu Hình Quản Lý Backend

**Vị Trí:**
```
xiaozhi-esp32-server-golang/manager/backend/config/
├─ config.yaml
└─ (các file cấu hình khác)
```

**Nhập:**
1) Đặt file cấu hình vào `manager/backend/config/`
2) Sau khi khởi động sẽ tự động mount vào container tại `/root/config/`
3) Sau khi chỉnh sửa, khởi động lại dịch vụ backend:

```bash
cd docker/docker-composer/
docker compose restart backend
```

### 3. File Thư Viện ten_vad

**Ghi Chú:**
- Docker image đã bao gồm thư viện ten_vad (`/workspace/lib/ten-vad/`)
- Đường dẫn thư viện runtime đã được cấu hình tự động qua `LD_LIBRARY_PATH`
- Không cần mount thêm khi sử dụng ten_vad

## Kiểm Tra Sức Khỏe

Dịch vụ MySQL được cấu hình kiểm tra sức khỏe:

```yaml
healthcheck:
  test: ["CMD", "mysqladmin", "ping", "-h", "localhost", "-u", "root", "-ppassword"]
  timeout: 20s
  retries: 10
  interval: 10s
  start_period: 30s
```

## Xử Lý Sự Cố

### 1. Dịch Vụ Không Khởi Động Được

```bash
cd docker/docker-composer/

docker compose logs [tên-dịch-vụ]

# Kiểm tra cổng đang bị chiếm dụng (Linux)
netstat -tulpn | grep [cổng]
```

### 2. Kết Nối Cơ Sở Dữ Liệu Thất Bại

```bash
cd docker/docker-composer/

docker compose ps mysql

docker compose logs mysql

docker compose exec mysql mysql -u root -ppassword
```

### 3. Vấn Đề Kết Nối Mạng

```bash
cd docker/docker-composer/

docker network ls
docker network inspect xiaozhi-network

docker compose exec main-server ping mysql
```

## Khuyến Nghị Tối Ưu Hiệu Suất

1) Đặt giới hạn tài nguyên cho từng dịch vụ trong môi trường production
2) Cấu hình xoay vòng log để tránh log quá lớn
3) Sao lưu dữ liệu MySQL định kỳ
4) Tích hợp hệ thống giám sát

## Lưu Ý Bảo Mật

1) Thay đổi mật khẩu cơ sở dữ liệu mặc định trong môi trường production
2) Chỉ expose các cổng cần thiết
3) Cấu hình tường lửa và kiểm soát truy cập
4) Sử dụng nguồn image đáng tin cậy

---

## Bước Tiếp Theo

### Truy Cập Bảng Quản Lý

Sau khi khởi động dịch vụ, truy cập http://<IP-hoặc-tên-miền-máy-chủ>:8080 để vào bảng quản lý.

**[Hướng Dẫn Sử Dụng Bảng Quản Lý →](manager_console_guide.md)**

### Cấu Hình Thiết Bị ESP32

Tham khảo [Hướng Dẫn Kết Nối Phía ESP32](esp32_xiaozhi_backend_guide.md) để hoàn tất kết nối thiết bị.
