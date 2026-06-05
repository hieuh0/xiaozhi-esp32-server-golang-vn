# 🚀 xiaozhi-esp32-server-golang

> **Backend AI Xiaozhi cho Thiết Bị ESP32**

---

## Giới Thiệu Dự Án

xiaozhi-esp32-server-golang là dịch vụ backend AI hiệu suất cao, hoàn toàn streaming, được thiết kế dành riêng cho các ứng dụng IoT và tương tác giọng nói thông minh. Dự án được phát triển bằng ngôn ngữ Go, tích hợp các năng lực cốt lõi gồm ASR (Nhận Dạng Giọng Nói Tự Động), LLM (Mô Hình Ngôn Ngữ Lớn), TTS (Tổng Hợp Giọng Nói), hỗ trợ đồng thời quy mô lớn và kết nối đa giao thức, giúp thiết bị đầu cuối thông minh và thiết bị biên tương tác giọng nói AI hiệu quả.

---

## ✨ Tính Năng Chính

- ⚡ **Chuỗi AI Giọng Nói Streaming Đầu Cuối**: Xử lý streaming toàn bộ quy trình ASR → LLM → TTS, tương tác thời gian thực độ trễ thấp
- 🎙️ **Nhận Dạng Giọng Nói & Chuyển Đổi TTS Động**: Tự động chuyển đổi giọng TTS theo danh tính người nói, trải nghiệm giọng nói cá nhân hóa
- 🔌 **Trừu Tượng Hóa Tầng Giao Tiếp Transport**: Trừu tượng hóa thống nhất WebSocket / MQTT UDP, linh hoạt tích hợp logic chính, dễ mở rộng giao thức
- 📬 **Xử Lý Hàng Đợi Tin Nhắn**: LLM và TTS xử lý bất đồng bộ qua hàng đợi tin nhắn, hỗ trợ tích hợp logic nghiệp vụ linh hoạt
- 🌐 **Kết Nối Đa Giao Thức Đồng Thời Cao**: Hỗ trợ kết nối đồng thời quy mô lớn và đẩy tin nhắn cho nhiều thiết bị
- ♻️ **Pool Tài Nguyên Hiệu Quả & Tái Sử Dụng Kết Nối**: Cơ chế pool kết nối tài nguyên ngoài, giảm độ trễ phản hồi, tăng thông lượng hệ thống
- 🤖 **Tích Hợp Đa Công Cụ AI**: Dựa trên framework Eino, hỗ trợ FunASR, tương thích OpenAI, Ollama, Doubao, EdgeTTS, CosyVoice và nhiều engine khác
- 🧩 **Kiến Trúc Module Hóa Có Thể Mở Rộng**: Các module cốt lõi VAD/ASR/LLM/TTS/MCP/Vision độc lập, có thể cắm rút
- 🎵 **MCP Audio Server**: Phân trang và xử lý streaming tài nguyên âm thanh, điều khiển phát nhạc và âm lượng
- 🦞 **Tích Hợp Agent OpenClaw**: Tạo OpenClaw Endpoint riêng theo agent, hỗ trợ xem trạng thái kết nối, kiểm tra phiên, định tuyến theo từ khóa vào/thoát (mặc định "mở tôm hùm/vào tôm hùm" và "đóng tôm hùm/thoát tôm hùm")
- 🖥️ **Bảng Điều Khiển Web Quản Lý Đầy Đủ Tính Năng**: Hướng dẫn cấu hình trực quan, kiểm tra khả dụng toàn chuỗi VAD/ASR/LLM/TTS, quản lý thiết bị và chèn tin nhắn, giám sát độ trễ thời gian thực và xác thực OTA
- 🧠 **Tính Năng Nghiệp Vụ Nâng Cao**: Tổng hợp và nhập MCP Market, sao chép giọng nói, kho kiến thức (Dify/RAGFlow/WeKnora), gỡ lỗi lệnh gọi MCP từ xa theo chiều thiết bị/agent
- 📦 **Giải Pháp Triển Khai Một Chạm Dễ Dùng**: Gói aio tiền biên dịch dùng ngay (chương trình chính + bảng điều khiển + dịch vụ nhận dạng giọng nói), triển khai Docker một lệnh, hỗ trợ biên dịch nội bộ Linux/Windows/macOS
- 🔐 **Hệ Thống Bảo Mật & Phân Quyền** (đang lên kế hoạch): Giao diện xác thực người dùng và quản lý quyền đã được dự phòng

---

[Phân tích kiến trúc deepwiki](https://deepwiki.com/hackers365/xiaozhi-esp32-server-golang)

## 🚀 Bắt Đầu Nhanh

### Cách 1: Gói Khởi Động Một Chạm (Khuyến nghị)

Tải về gói nén cho nền tảng tương ứng, giải nén và chạy:

- **Trang Release**: <https://github.com/hackers365/xiaozhi-esp32-server-golang/releases>
- **Hướng dẫn sử dụng**: [doc/quickstart_bundle_tutorial.md](doc/quickstart_bundle_tutorial.md)

Sau khi khởi động, truy cập **http://<IP máy chủ hoặc tên miền>:8080** để vào bảng điều khiển Web và cấu hình.

### Cách 2: Triển Khai Docker

- [Docker Compose (có bảng điều khiển)](doc/docker_compose.md)
- [Docker (không có bảng điều khiển)](doc/docker.md)

### Cách 3: Biên Dịch Nội Bộ

Phù hợp cho môi trường phát triển hoặc các trường hợp cần biên dịch tùy chỉnh.

**Cài đặt phụ thuộc** (ví dụ với Ubuntu)

```bash
# Go 1.20+
# Codec Opus
sudo apt-get install -y pkg-config libopus0 libopusfile-dev

# ONNX Runtime (1.21.0)
wget https://github.com/microsoft/onnxruntime/releases/download/v1.21.0/onnxruntime-linux-x64-1.21.0.tgz
tar -xzf onnxruntime-linux-x64-1.21.0.tgz
sudo cp -r onnxruntime-linux-x64-1.21.0/include/* /usr/local/include/onnxruntime/
sudo cp -r onnxruntime-linux-x64-1.21.0/lib/* /usr/local/lib/
sudo ldconfig

# Phụ thuộc runtime ten_vad
sudo apt install -y libc++1 libc++abi1
```

> 📖 Tài liệu phụ thuộc đầy đủ và cấu hình Windows/macOS xem tại [config.md](doc/config.md)

Quy trình biên dịch tách biệt cho chương trình chính, frontend/backend bảng điều khiển, dịch vụ nhận dạng giọng nói và đóng gói AIO xem tại [doc/compile_deploy.md](doc/compile_deploy.md)

Tham khảo [tài liệu chính thức FunASR](https://github.com/modelscope/FunASR/blob/main/runtime/docs/SDK_advanced_guide_online_zh.md) để triển khai.

**Biên dịch và khởi động**

```bash
# Biên dịch
go build -o xiaozhi_server ./cmd/server/

# Khởi động (xem chi tiết file cấu hình tại config/config.yaml)
./xiaozhi_server -c config/config.yaml
```

---

## 📚 Điều Hướng Tài Liệu

### Liên Quan Đến Triển Khai
- [Hướng dẫn gói khởi động một chạm](doc/quickstart_bundle_tutorial.md)
- [Triển khai Docker Compose](doc/docker_compose.md)
- [Triển khai Docker](doc/docker.md)
- [Hướng dẫn biên dịch và triển khai](doc/compile_deploy.md)
- [Giải thích cấu hình chi tiết](doc/config.md)

### Hướng Dẫn Sử Dụng
- [Hướng dẫn sử dụng bảng quản trị](doc/manager_console_guide.md)
- [Cấu hình dịch vụ WebSocket & OTA](doc/websocket_server.md)
- [Cấu hình MQTT + UDP](doc/mqtt_udp.md)
- [Giao thức MQTT UDP](doc/mqtt_udp_protocol.md)

### Các Module Tính Năng
- [Khả năng thị giác](doc/vision.md)
- [Nhận dạng giọng nói](doc/speaker_identification.md)
- [Kiến trúc MCP](doc/mcp.md)
- [Tài nguyên âm thanh MCP](doc/mcp_resource.md)
- [MCP Market (Khám phá/Nhập/Cập nhật nóng)](doc/mcp_market.md)
- [Tích hợp Agent OpenClaw (Endpoint/Định tuyến từ khóa/Kiểm tra phiên)](doc/openclaw_integration.md)
- [Sao chép giọng nói (Thao tác người dùng & hạn mức quản trị viên)](doc/voice_clone.md)
- [Kho kiến thức (Cấu hình Provider/Đồng bộ/Kiểm tra thu hồi/RAG)](doc/knowledge_base.md)
- [Lệnh gọi MCP từ xa theo chiều thiết bị/agent (Endpoint/Tools/Call)](doc/mcp_remote_call_agent_device.md)

### Kết Nối Thiết Bị
- [Hướng dẫn kết nối ESP32](doc/esp32_xiaozhi_backend_guide.md)
- [Hướng dẫn ủy quyền OTA MQTT](doc/ota_mqtt_auth.md)

---

## 🧩 Kiến Trúc Module

| Module | Mô tả chức năng | Công nghệ |
|--------|----------------|-----------|
| VAD | Phát hiện hoạt động giọng nói | Silero VAD / WebRTC VAD / ten_vad |
| ASR | Nhận dạng giọng nói | FunASR / Doubao ASR |
| LLM | Suy luận mô hình lớn | Tương thích framework Eino, OpenAI, Ollama, v.v. |
| TTS | Tổng hợp giọng nói | Doubao / EdgeTTS / CosyVoice |
| MCP | Kết nối đa giao thức, khám phá và nhập MCP Market, gỡ lỗi lệnh gọi từ xa theo chiều thiết bị/agent | MCP Server / Điểm kết nối / MCP Market / SSE / StreamableHTTP / WebSocket Controller / MCP Tool Call |
| OpenClaw | Điểm kết nối theo chiều agent, chuyển đổi chế độ từ khóa vào/thoát, chuyển tiếp và kiểm tra tin nhắn phiên | OpenClaw WebSocket / Agent Endpoint / Chat Router |
| Thị giác | Xử lý thị giác | Doubao / Alibaba Cloud Vision |
| Nhận dạng giọng nói | Nhận dạng người nói | sherpa-onnx + cơ sở dữ liệu vector |
| Sao chép giọng nói | Tạo và thử nghiệm giọng nói sao chép phía người dùng | Minimax / CosyVoice / Qwen |
| Kho kiến thức (RAG) | Đồng bộ tài liệu, kiểm tra thu hồi và truy vấn hội thoại | Dify / RAGFlow / WeKnora |

---

## 📈 Hiệu Suất & Kiểm Thử

- [Báo cáo kiểm thử độ trễ](doc/delay_test.md)
- Bảng quản trị cung cấp điểm vào kiểm tra khả dụng và độ trễ VAD/ASR/LLM/TTS

---

## 🛠️ Lộ Trình Phát Triển

- AI chủ động

---

## 🤝 Đóng Góp

Chào mừng gửi Issue, PR hoặc đề xuất!

---

## 📄 Giấy Phép

MIT License

---

## 📬 Liên Hệ

**WeChat cá nhân**: hackers365 (thêm WeChat để tham gia nhóm trao đổi)

![WeChat cá nhân](https://github.com/user-attachments/assets/6b8d3d11-7bf5-4fa4-a73e-5109019dab85)


**Mã nguồn mở không dễ, sự tài trợ của bạn giúp dự án tiếp tục được cập nhật**

<img width="250" height="250" alt="eab0f4d3d8b6f977863a7bef36e3d64b" src="https://github.com/user-attachments/assets/9a949cb3-d788-446b-a0b9-8542edbb0842" />

---

> © 2024 xiaozhi-esp32-server-golang
