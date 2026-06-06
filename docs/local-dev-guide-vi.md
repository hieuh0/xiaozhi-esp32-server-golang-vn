# Hướng Dẫn Chạy Local (Dev) — Windows

> Dành cho developer muốn chạy toàn bộ hệ thống trên máy Windows mà **không cần Docker**.

---

## Mục Lục

1. [Yêu cầu](#1-yêu-cầu)
2. [Cấu trúc dự án](#2-cấu-trúc-dự-án)
3. [Cấu hình backend](#3-cấu-hình-backend)
4. [Chạy nhanh bằng script](#4-chạy-nhanh-bằng-script)
5. [Chạy thủ công từng phần](#5-chạy-thủ-công-từng-phần)
6. [Truy cập ứng dụng](#6-truy-cập-ứng-dụng)
7. [Các lệnh hay dùng](#7-các-lệnh-hay-dùng)
8. [Xử lý sự cố](#8-xử-lý-sự-cố)

---

## 1. Yêu Cầu

Cài đặt các công cụ sau trước khi bắt đầu:

| Công cụ | Phiên bản tối thiểu | Tải về |
|---|---|---|
| **Go** | 1.24+ | https://go.dev/dl/ |
| **Node.js** | 18+ | https://nodejs.org/ |
| **Git** | bất kỳ | https://git-scm.com/ |
| **MySQL** | 8.0+ *(tuỳ chọn)* | https://dev.mysql.com/downloads/ |

> **Mẹo:** Nếu không muốn cài MySQL, dùng chế độ **SQLite** — database được lưu trực tiếp vào file, không cần cài thêm gì.

Kiểm tra nhanh sau khi cài:

```powershell
go version      # go version go1.24.x ...
node --version  # v18.x.x hoặc cao hơn
npm --version   # 9.x.x hoặc cao hơn
```

---

## 2. Cấu Trúc Dự Án

```
xiaozhi-esp32-server-golang-vn/
├── dev.ps1                        ← Script chạy dev tự động (Windows)
├── manager/
│   ├── backend/                   ← Backend Go (API, port 8080)
│   │   ├── main.go
│   │   ├── config/
│   │   │   ├── config.json        ← Config mặc định (MySQL)
│   │   │   └── config.sqlite.json ← Tự động tạo khi dùng --sqlite
│   │   └── data/                  ← Tự tạo: SQLite DB + lịch sử chat
│   └── frontend/                  ← Frontend Vue 3 (UI, port 3000)
│       ├── src/
│       └── vite.config.js         ← Proxy /api → localhost:8080
```

---

## 3. Cấu Hình Backend

File config nằm tại `manager/backend/config/config.json`.

### Tùy chọn A — SQLite (khuyến nghị cho dev)

Không cần cài MySQL. Database lưu vào file `manager/backend/data/xiaozhi.db`.

Khi chạy script với flag `--sqlite`, file `config.sqlite.json` sẽ được tạo tự động từ `config.json` với phần database thay đổi thành:

```json
{
  "database": {
    "type": "sqlite",
    "sqlite": {
      "file_path": "./data/xiaozhi.db"
    }
  }
}
```

### Tùy chọn B — MySQL

Chỉnh sửa `manager/backend/config/config.json`:

```json
{
  "server": {
    "port": "8080",
    "mode": "debug"
  },
  "database": {
    "type": "mysql",
    "mysql": {
      "host": "127.0.0.1",
      "port": 3306,
      "username": "root",
      "password": "your_password",
      "database": "xiaozhi_admin"
    }
  },
  "jwt": {
    "secret": "xiaozhi_admin_secret_key",
    "expire_hour": 24
  },
  "internal_auth_token": "xiaozhi_admin_secret_key",
  "endpoint_auth_token": "xiaozhi_mcp_openclaw_secret_key",
  "speaker_service": {
    "url": "http://127.0.0.1:9000"
  },
  "storage": {
    "speaker_audio_path": "storage/speakers",
    "max_file_size": 10485760
  }
}
```

Tạo database MySQL trước:

```sql
CREATE DATABASE xiaozhi_admin CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

---

## 4. Chạy Nhanh Bằng Script

Script `dev.ps1` ở thư mục gốc tự động mở **2 cửa sổ PowerShell riêng** cho backend và frontend.

### Lần đầu tiên (khuyến nghị dùng SQLite)

```powershell
# Mở PowerShell tại thư mục gốc dự án, rồi chạy:
.\dev.ps1 --sqlite
```

Script sẽ:
1. Kiểm tra Go, Node.js đã cài chưa
2. Tự tạo `config.sqlite.json` nếu chưa có
3. Chạy `npm install` nếu chưa cài dependencies
4. Mở cửa sổ riêng chạy backend (`go run`)
5. Mở cửa sổ riêng chạy frontend (`npm run dev`)

### Các lệnh script

```powershell
# Dùng SQLite (không cần MySQL) — khuyến nghị lần đầu
.\dev.ps1 -sqlite

# Dùng MySQL (config.json mặc định)
.\dev.ps1

# Dùng SQLite + reset toàn bộ dữ liệu
.\dev.ps1 -sqlite -resetDb

# Chỉ chạy backend (frontend đang chạy ở nơi khác)
.\dev.ps1 -sqlite -backendOnly

# Chỉ chạy frontend (backend đang chạy ở nơi khác)
.\dev.ps1 -frontendOnly

# Xem help
.\dev.ps1 -help
```

> **Lưu ý PowerShell Execution Policy:** Nếu gặp lỗi `cannot be loaded because running scripts is disabled`, chạy lệnh sau một lần rồi thử lại:
> ```powershell
> Set-ExecutionPolicy -Scope CurrentUser -ExecutionPolicy RemoteSigned
> ```

---

## 5. Chạy Thủ Công Từng Phần

Nếu muốn kiểm soát từng bước hoặc script gặp vấn đề.

### Chạy Backend

Mở PowerShell, `cd` vào thư mục backend:

```powershell
cd manager\backend

# Tải Go dependencies
go mod tidy

# Chạy với SQLite
go run main.go -config=config\config.sqlite.json

# Hoặc chạy với MySQL (config mặc định)
go run main.go -config=config\config.json

# Reset database (thêm flag -reset-db)
go run main.go -config=config\config.sqlite.json -reset-db
```

Backend khởi động thành công khi thấy:

```
server listening on port: 8080
```

### Chạy Frontend

Mở PowerShell mới (để backend vẫn chạy), `cd` vào thư mục frontend:

```powershell
cd manager\frontend

# Cài dependencies (chỉ cần làm 1 lần)
npm install

# Chạy dev server (proxy /api → http://127.0.0.1:8080)
npm run dev
```

Frontend khởi động thành công khi thấy:

```
  VITE v4.x.x  ready in xxx ms

  ➜  Local:   http://localhost:3000/
```

---

## 6. Truy Cập Ứng Dụng

| Địa chỉ | Mô tả |
|---|---|
| http://localhost:3000 | **Giao diện quản trị** (Vue 3) |
| http://localhost:8080/api | Backend API (Go/Gin) |
| http://localhost:3000/openapi-docs | Tài liệu API |

### Đăng nhập lần đầu

Tài khoản admin mặc định được tạo tự động khi database khởi tạo:

| Trường | Giá trị |
|---|---|
| Email | `admin@xiaozhi.me` |
| Mật khẩu | `admin123` |

> Đổi mật khẩu ngay sau lần đăng nhập đầu tiên.

### Luồng proxy

```
Trình duyệt  →  http://localhost:3000/api/...
                        ↓ (Vite proxy)
              http://localhost:8080/api/...  (Go backend)
```

Frontend dev server tự proxy mọi request `/api/*` sang backend. Không cần cấu hình CORS khi dev.

---

## 7. Các Lệnh Hay Dùng

### Build frontend (tạo file tĩnh)

```powershell
cd manager\frontend
npm run build
# Output: manager/frontend/dist/
```

### Xem log backend chi tiết

Backend chạy ở chế độ `debug` theo mặc định — mọi request đều được in ra console.

Thay `"mode": "debug"` thành `"mode": "release"` trong config để tắt log verbose.

### Reset database

```powershell
cd manager\backend
go run main.go -config=config\config.sqlite.json -reset-db
```

### Kiểm tra Go dependencies

```powershell
cd manager\backend
go mod tidy   # Dọn dẹp và tải dependencies
go mod verify # Xác minh checksum
```

---

## 8. Xử Lý Sự Cố

### Backend không khởi động — lỗi database

**Triệu chứng:**
```
database initialization failed, exiting
```

**Giải pháp:**
- Nếu dùng MySQL: kiểm tra MySQL đang chạy, đúng host/port/password trong config
- Nếu dùng SQLite: đảm bảo thư mục `manager/backend/data/` tồn tại (tự tạo nếu thiếu):
  ```powershell
  New-Item -ItemType Directory -Path manager\backend\data -Force
  ```

---

### Frontend báo lỗi API (Network Error / 404)

**Triệu chứng:** Login không được, console báo lỗi kết nối API.

**Giải pháp:** Đảm bảo backend đang chạy trên cổng 8080. Kiểm tra:

```powershell
# Xem port 8080 có đang được lắng nghe không
netstat -an | Select-String ":8080"
```

---

### Lỗi `go: command not found` hoặc `npm: command not found`

Go hoặc Node.js chưa được thêm vào PATH. Sau khi cài, **khởi động lại PowerShell** hoặc reboot máy.

---

### Cổng 3000 hoặc 8080 đã bị chiếm

```powershell
# Tìm process đang dùng cổng 8080
netstat -ano | Select-String ":8080" | Select-String "LISTENING"
# Cột cuối là PID — kill nó:
Stop-Process -Id <PID> -Force
```

---

### Script dev.ps1 bị chặn bởi Execution Policy

```powershell
Set-ExecutionPolicy -Scope CurrentUser -ExecutionPolicy RemoteSigned
# Sau đó chạy lại script
.\dev.ps1 --sqlite
```

---

### `go run` chậm lần đầu

Bình thường — Go cần biên dịch toàn bộ dependencies. Các lần sau sẽ nhanh hơn do cache. Nếu muốn build trước:

```powershell
cd manager\backend
go build -o backend.exe .
.\backend.exe -config=config\config.sqlite.json
```
