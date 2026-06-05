# Hỗ trợ biên dịch cục bộ với Docker

Đã bổ sung tệp `docker-compose.local.yml` hỗ trợ biên dịch cục bộ và triển khai đa kiến trúc.

## Tệp mới thêm

- `docker/docker-composer/docker-compose.local.yml` - Tệp cấu hình biên dịch cục bộ

## Cách biên dịch

### Biên dịch mặc định (AMD64)

```bash
cd docker/docker-composer
docker-compose -f docker-compose.local.yml up --build
```

### Biên dịch ARM64 (Apple Silicon)

```bash
cd docker/docker-composer
TARGETARCH=arm64 docker-compose -f docker-compose.local.yml up --build
```

## Cách chạy

Sau khi biên dịch xong, các dịch vụ sẽ tự động khởi động, bao gồm:
- Máy chủ chính (cổng 8989)
- Quản trị backend (cổng 8081)
- Giao diện frontend (cổng 8080)
- Cơ sở dữ liệu MySQL (cổng 23306)

Truy cập http://<IP hoặc tên miền máy chủ>:8080 để xem giao diện frontend.

## 🏗️ Hỗ trợ đa kiến trúc

### Tự động phát hiện kiến trúc (khuyến nghị)

`docker-compose.local.yml` hỗ trợ tự động phát hiện kiến trúc hệ thống hiện tại:

```bash
# Tự động phát hiện kiến trúc và build (hành vi mặc định)
docker-compose -f docker-compose.local.yml up --build
```

### Chỉ định kiến trúc thủ công

Nếu cần build cho một kiến trúc cụ thể:

```bash
# Build cho kiến trúc ARM64
TARGETARCH=arm64 docker-compose -f docker-compose.local.yml up --build

# Build cho kiến trúc AMD64
TARGETARCH=amd64 docker-compose -f docker-compose.local.yml up --build
```

### Các kiến trúc được hỗ trợ

- **AMD64/x86_64**: Bộ xử lý Intel/AMD (mặc định)
- **ARM64**: Apple Silicon (M1/M2), máy chủ ARM

## 📁 Mô tả tệp cấu hình

### docker-compose.yml

Sử dụng image chính thức đã được build sẵn, phù hợp cho môi trường production:

```yaml
services:
  mysql:
    image: docker.jsdelivr.fyi/mysql:8.0
  main-server:
    image: docker.jsdelivr.fyi/hackers365/xiaozhi_golang:0.1
  backend:
    image: docker.jsdelivr.fyi/hackers365/xiaozhi_backend:0.1
  frontend:
    image: docker.jsdelivr.fyi/hackers365/xiaozhi_frontend:0.1
```

### docker-compose.local.yml

Phiên bản build cục bộ, hỗ trợ chỉnh sửa mã nguồn và đa kiến trúc:

```yaml
services:
  main-server:
    build:
      context: ../..
      dockerfile: docker/Dockerfile.main
      args:
        TARGETARCH: ${TARGETARCH:-amd64}
```

## 🔧 Cấu hình biến môi trường

### Liên quan đến kiến trúc

| Tên biến | Giá trị mặc định | Mô tả |
|-------|-------|------|
| `TARGETARCH` | `amd64` | Kiến trúc đích (amd64/arm64) |


## 🛠️ Các thao tác thường dùng

### Xem trạng thái dịch vụ

```bash
# Xem trạng thái tất cả dịch vụ
docker-compose ps

# Xem log dịch vụ
docker-compose logs -f main-server
docker-compose logs -f backend
docker-compose logs -f frontend
```

### Dừng và khởi động lại dịch vụ

```bash
# Dừng tất cả dịch vụ
docker-compose down

# Khởi động lại dịch vụ cụ thể
docker-compose restart main-server

# Build lại và khởi động
docker-compose up --build
```
