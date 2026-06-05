# 🚦 Luồng Dữ Liệu

1. **Gọi giao diện OTA**
   - Lấy địa chỉ **MQTT** và **WebSocket**

2. **Kết nối MQTT**
   - `mqtt_server` tích hợp sẽ phát một sự kiện vòng đời tới `/p2p/device_public/_server/lifecycle`
   - Chương trình chính tạo mới hoặc tái sử dụng MQTT transport dựa trên `device_id`, đồng thời cố gắng khởi động trước MCP phía thiết bị

3. **Gửi tin nhắn `hello`**
   - Nhận được:
     - 🎵 `audio_params`
     - 🌐 Địa chỉ máy chủ UDP
     - 🔑 `aes_key`
     - 🧩 `nonce`

4. **Kết nối máy chủ UDP**
   - Thực hiện gửi và nhận dữ liệu giọng nói

5. **Gửi các tín hiệu tiếp theo như `listen`, `abort`, v.v.**
   - Ngữ nghĩa tín hiệu không thay đổi, vẫn dựa trên khởi tạo cấp phiên trò chuyện sau khi hoàn thành `hello`

---

# 🧭 Topic Vòng Đời

- **Topic**: `/p2p/device_public/_server/lifecycle`
- **Mục đích**: Chỉ dùng nội bộ phía máy chủ, dùng để truyền sự kiện thiết bị kết nối/ngắt kết nối MQTT
- **Ví dụ nội dung tin nhắn**:
  ```json
  {
    "type": "mqtt_lifecycle",
    "device_id": "11:22:33:44:55:66",
    "state": "online",
    "client_id": "GID_test@@@11_22_33_44_55_66@@@uuid",
    "ts": 1710000000000
  }
  ```

- **Định nghĩa trạng thái**
  - `online`: Thiết bị vừa kết nối tới `mqtt_server`, chương trình chính có thể chuẩn bị trước transport và MCP
  - `offline`: Thiết bị ngắt kết nối khỏi `mqtt_server`, chương trình chính lập tức ánh xạ trạng thái ngoại tuyến, nhưng transport sẽ được giữ lại trong một khoảng thời gian để tái sử dụng khi kết nối lại nhanh

- **Lưu ý phạm vi**
  - Sự kiện vòng đời không thay thế `hello`
  - Sự kiện vòng đời chỉ duy trì tài nguyên cấp kết nối, không mang thông tin cấp phiên trò chuyện như `audio_params`, đàm phán UDP, v.v.

---

# 🛠️ Luồng Xử Lý Phía Máy Chủ

| Bước | Mô tả |
| :--- | :--- |
| 1. Lắng nghe vòng đời MQTT | Khi nhận sự kiện `online`, tạo mới hoặc tái sử dụng transport, đồng thời cố gắng khởi động trước MCP phía thiết bị |
| 2. Xử lý `hello` | Trả về `audio_params`, địa chỉ UDP, khóa và `nonce`, đồng thời chuẩn bị trạng thái phiên cấp trò chuyện |
| 3. Lắng nghe tin nhắn MQTT | Khi nhận `type: listen, state: start`, khởi tạo cấu trúc `clientState` với trạng thái `start` |
| 4. Dịch vụ UDP | Sau khi nhận gói tin, phân tích `nonce`, tìm `clientState` tương ứng, điền địa chỉ từ xa, trạng thái chuyển sang `recv` |
| 5. Dừng nhận | Khi nhận `type: listen, state: stop` hoặc tự động phát hiện không có âm thanh, dừng nhận |
| 6. Vòng đời MQTT ngoại tuyến | Khi nhận sự kiện `offline`, lập tức ánh xạ trạng thái ngoại tuyến và thu hồi transport sau thời gian giữ lại |

---

# 🔗 Quan Hệ Liên Kết

- OTA xác thực **địa chỉ MAC** và **clientId**, liên kết với **uid**
- **Địa chỉ MQTT** và **mqtt_clientId** được OTA cấp phát liên kết với **địa chỉ MAC** và **clientId**
- Qua **tin nhắn vòng đời kết nối MQTT** có thể liên kết trước **địa chỉ MAC**, `device_id`, `client_id`
- Qua **tin nhắn MQTT `hello`** có thể liên kết tới `audio_params`, `aes_key`, `nonce`
- Qua **tin nhắn âm thanh UDP** có thể liên kết tới `nonce`

---

> **Ghi chú:**
> - Cấu trúc `clientState` dùng để duy trì trạng thái phiên cấp trò chuyện và tài nguyên của từng client.
> - transport và MCP có thể được chuẩn bị trước trong giai đoạn kết nối MQTT, nhưng đàm phán cấp trò chuyện thực sự vẫn lấy `hello` làm chuẩn.
> - `nonce` là định danh duy nhất giữa client và máy chủ, dùng cho liên kết bảo mật và định tuyến dữ liệu.
