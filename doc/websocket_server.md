# Hướng dẫn cấu hình WebSocket Server và OTA

Tài liệu này dành cho người dùng mới bắt đầu, hướng dẫn chi tiết cách cấu hình WebSocket Server và các thông số liên quan đến OTA (nâng cấp firmware).

---

## 1. Vị trí tệp cấu hình

Tất cả cấu hình chính đều nằm trong:

- `config/config.yaml`

Nếu không tìm thấy tệp này, bạn cũng có thể tham khảo `config/config.json.git`.

---

## 2. Cấu hình WebSocket Server

### 2.1 Chức năng
WebSocket Server dùng để giao tiếp thời gian thực giữa thiết bị và máy chủ.

### 2.2 Các thông số cấu hình quan trọng
Tìm nội dung sau trong tệp `config/config.yaml`:

```yaml
websocket:
  host: "0.0.0.0"
  port: 8989
```
- `host`: Địa chỉ lắng nghe, thông thường giữ nguyên `0.0.0.0`.
- `port`: Cổng lắng nghe, mặc định là `8989`, có thể thay đổi theo nhu cầu.

### 2.3 Cách thay đổi
Nếu muốn đổi cổng sang 9000:
```yaml
websocket:
  host: "0.0.0.0"
  port: 9000
```

---

## 3. Cấu hình OTA (Nâng cấp firmware)

### 3.1 Chức năng
OTA dùng để thiết bị tự động lấy các thông số kết nối WebSocket/MQTT và thông tin nâng cấp firmware từ máy chủ.

### 3.2 Các thông số cấu hình quan trọng
Tìm phần `ota` trong tệp `config/config.yaml`:

```yaml
ota:
  test:
    websocket:
      url: "ws://192.168.208.214:8989/xiaozhi/v1/"
    mqtt:
      enable: false
      endpoint: "192.168.208.214"
  external:
    websocket:
      url: "wss://www.tb263.cn:55555/go_ws/xiaozhi/v1/"
    mqtt:
      enable: false
      endpoint: "www.youdomain.cn"
```
- `test`: Thông số thiết bị lấy trong môi trường mạng nội bộ; điều kiện xác định trong chương trình là địa chỉ IP bắt đầu bằng 192.168 hoặc 127.0.
- `external`: Thông số thiết bị lấy trong môi trường mạng bên ngoài.
- `websocket.url`: Địa chỉ WebSocket Server mà thiết bị sẽ kết nối đến.
- `mqtt.enable`: Nếu bật, giao diện OTA sẽ trả về địa chỉ MQTT đã cấu hình; thiết bị sẽ ưu tiên sử dụng phương thức mqtt+udp.
- `mqtt.endpoint`: Địa chỉ MQTT Server; phía thiết bị mặc định sử dụng cổng 8883 (kết nối TLS), nếu chỉ định cổng khác 8883 thì sẽ dùng kết nối TCP không mã hóa.

### 3.3 Ví dụ thay đổi thường gặp
- Thay đổi địa chỉ WebSocket mạng nội bộ:
  ```yaml
  ota:
    test:
      websocket:
        url: "ws://192.168.1.100:8989/xiaozhi/v1/"
  ```
- Thay đổi địa chỉ WebSocket mạng bên ngoài:
  ```yaml
  ota:
    external:
      websocket:
        url: "wss://yourdomain.com:55555/go_ws/xiaozhi/v1/"
  ```

---

## 4. Mô tả giao diện OTA (thiết bị lấy cấu hình như thế nào)

1. Thiết bị gửi yêu cầu HTTP POST đến `http://địa-chỉ-máy-chủ:cổng/xiaozhi/ota/`.
2. Header của yêu cầu cần bao gồm:
   - `Device-Id`: ID duy nhất của thiết bị (ví dụ: địa chỉ MAC)
   - `Client-Id`: ID duy nhất của client
3. Máy chủ sẽ tự động chọn cấu hình `test` hoặc `external` dựa trên IP của thiết bị, rồi trả về các thông số WebSocket/MQTT.
4. Thiết bị phân tích nội dung trả về, kết nối đến WebSocket Server theo `websocket.url`.

---

## 5. Các vấn đề thường gặp

- **Cổng bị chiếm dụng?**
  - Thay đổi `websocket.port`, khởi động lại dịch vụ.
- **Thiết bị không kết nối được với máy chủ?**
  - Kiểm tra `websocket.url` trong cấu hình `ota` có đúng không, cổng máy chủ có được mở không.
- **Cần MQTT?**
  - Đặt `mqtt.enable` thành `true` và cấu hình `endpoint`.

---

Nếu có thắc mắc, hãy kiểm tra trước các thông số trong `config/config.yaml`, sau đó tham khảo tài liệu này.
