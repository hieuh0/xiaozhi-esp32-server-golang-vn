# MQTT Broker Setup Guide

## Tổng quan

Hệ thống cung cấp 2 thành phần MQTT độc lập:

| Thành phần | Trang cấu hình | Mục đích |
|---|---|---|
| **Built-in MQTT Broker** | `mqtt-server-config` | Broker nội bộ — thiết bị (ESP32) kết nối trực tiếp vào đây |
| **External MQTT Client** | `mqtt-config` | Client bridge ra ngoài — kết nối tới cloud broker bên ngoài |

> **Hầu hết trường hợp** chỉ cần cấu hình Built-in Broker. Trang `mqtt-config` chỉ dùng khi cần relay dữ liệu lên cloud MQTT (AWS IoT, HiveMQ, Mosquitto remote).

---

## 1. Built-in MQTT Broker

### Cấu hình cơ bản

Vào **Admin → MQTT Server Config**:

| Trường | Giá trị khuyến nghị | Ghi chú |
|---|---|---|
| `listen_host` | `0.0.0.0` | Lắng nghe trên tất cả interfaces |
| `listen_port` | `1883` | Port MQTT standard |
| `enable_auth` | tuỳ | Bật để yêu cầu username/password |

### Khái niệm quan trọng

**`listen_host: 0.0.0.0`** không phải là IP để device kết nối — đây là bind address, nghĩa là broker mở cửa trên **tất cả** network interface của server:

```
Server (IP: 192.168.1.100)
  listen_host: 0.0.0.0:1883
  ↕
  Device kết nối tới: 192.168.1.100:1883  ← IP thực của server
```

---

## 2. Device kết nối — Cấu hình phía ESP32

### Cùng mạng LAN

```
broker = 192.168.1.100   ← IP của server trong mạng nội bộ
port   = 1883
type   = tcp
```

### Qua Internet (dùng IP public)

```
broker = 203.0.113.50    ← IP public của server
port   = 1883
type   = tcp
```

### Dùng subdomain (khuyến nghị cho production)

```
broker = mqtt.yourdomain.com
port   = 1883
type   = tcp
```

---

## 3. Cấu hình Domain / Subdomain

Subdomain giúp device không phụ thuộc vào IP cố định — khi đổi server chỉ cần cập nhật DNS.

### Bước 1: Tạo DNS A record

Vào DNS provider (Cloudflare, GoDaddy, etc.) và thêm:

```
Type: A
Name: mqtt          ← tạo ra mqtt.yourdomain.com
Value: 203.0.113.50 ← IP public của server
TTL: Auto
```

### Bước 2: Kiểm tra DNS propagation

```bash
nslookup mqtt.yourdomain.com
# hoặc
dig mqtt.yourdomain.com
```

### Bước 3: Cập nhật device firmware

```
broker = mqtt.yourdomain.com
port   = 1883
```

> **Lưu ý**: Broker config không cần thay đổi — vẫn để `listen_host: 0.0.0.0`. Domain chỉ là alias để device tìm IP server.

---

## 4. Cấu hình TLS (MQTTS)

TLS mã hóa kết nối giữa device và broker. Port mặc định: **8883**.

### Yêu cầu

- Có domain thật trỏ về server (subdomain như trên)
- File certificate: `cert.pem` + `key.pem`

### Cách A: Let's Encrypt (Khuyến nghị — miễn phí, tự động renew)

```bash
# Cài certbot
sudo apt install certbot

# Lấy cert (port 80 phải mở)
sudo certbot certonly --standalone -d mqtt.yourdomain.com

# Cert được lưu tại:
# /etc/letsencrypt/live/mqtt.yourdomain.com/fullchain.pem
# /etc/letsencrypt/live/mqtt.yourdomain.com/privkey.pem
```

> Device tự động trust Let's Encrypt cert — không cần cài cert thủ công.

### Cách B: Self-signed (Test/nội bộ)

```bash
openssl req -x509 -newkey rsa:4096 \
  -keyout key.pem -out cert.pem \
  -days 365 -nodes \
  -subj "/CN=mqtt.yourdomain.com"
```

> Device phải được cài cert thủ công để trust.

### Bật TLS trong Admin UI

Vào **Admin → MQTT Server Config → TLS Config**:

| Trường | Giá trị |
|---|---|
| Enable TLS | ✅ bật |
| TLS Port | `8883` |
| Cert File | `/etc/letsencrypt/live/mqtt.yourdomain.com/fullchain.pem` |
| Key File | `/etc/letsencrypt/live/mqtt.yourdomain.com/privkey.pem` |

### Device kết nối qua TLS

```
broker = mqtt.yourdomain.com
port   = 8883                ← đổi từ 1883
type   = SSL/TLS             ← đổi từ tcp
```

---

## 5. So sánh các kịch bản

| Kịch bản | Broker config | Device config | Bảo mật |
|---|---|---|---|
| Dev/test local | `0.0.0.0:1883` | `192.168.1.x:1883` | Không mã hóa |
| Production LAN | `0.0.0.0:1883` | `192.168.1.x:1883` + auth | Auth only |
| Production Internet | `0.0.0.0:1883` + `0.0.0.0:8883 TLS` | `mqtt.domain.com:8883` | Mã hóa TLS |

---

## 6. Mở port Firewall (nếu cần)

```bash
# Cho phép MQTT plain
sudo ufw allow 1883/tcp

# Cho phép MQTTS
sudo ufw allow 8883/tcp

# Kiểm tra
sudo ufw status
```

---

## 7. External MQTT Client Bridge (Tuỳ chọn)

Chỉ cần thiết khi muốn relay dữ liệu lên cloud MQTT broker.

Vào **Admin → MQTT Config**:

```
broker    = broker.hivemq.com   ← địa chỉ cloud broker
type      = tcp (hoặc ssl)
port      = 1883 (hoặc 8883)
client_id = xiaozhi-bridge-001
username  = (nếu cloud broker yêu cầu)
password  = (nếu cloud broker yêu cầu)
```

> Trang này **không liên quan** đến device kết nối vào hệ thống. Device vẫn connect vào built-in broker (port 1883/8883).
