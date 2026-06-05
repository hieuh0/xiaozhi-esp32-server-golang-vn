# Hướng dẫn sử dụng trang quản trị

## Truy cập trang quản trị

- Địa chỉ: http://<IP máy chủ hoặc tên miền>:8080

---

## I. Trình hướng dẫn cấu hình

Sau khi đăng nhập lần đầu, hệ thống tự động vào trình hướng dẫn cấu hình gồm 5 bước.

### Step 1: Cấu hình OTA

Cấu hình thông tin máy chủ OTA, dùng để phân phối địa chỉ websocket và mqtt xuống phần cứng Xiaozhi.

<!-- 截图位置：OTA配置界面 -->
> Hình: Giao diện hướng dẫn cấu hình OTA

| Mục cấu hình | Mô tả |
|-------|------|
| MQTT Broker | Địa chỉ máy chủ MQTT |
| MQTT Port | Cổng MQTT (mặc định 1883) |
| UDP Port | Cổng UDP |
| ... | ... |

**Kiểm tra kết nối**: Nhấn "Kiểm tra cấu hình hiện tại" để xác minh kết nối MQTT/UDP.

---

### Step 2: Cấu hình VAD

Chọn engine phát hiện hoạt động giọng nói:

<!-- 截图位置：VAD配置界面 -->
> Hình: Giao diện hướng dẫn cấu hình VAD

| Engine | Mô tả | Trường hợp khuyến nghị |
|-----|------|---------|
| Silero VAD | Độ chính xác cao | Môi trường production |
| WebRTC VAD | Nhẹ | Tài nguyên hạn chế |
| ten_vad | Phiên bản C++ cục bộ | Yêu cầu hiệu năng cao |

---

### Step 3: Cấu hình ASR

Chọn engine nhận dạng giọng nói:

<!-- 截图位置：ASR配置界面 -->
> Hình: Giao diện hướng dẫn cấu hình ASR

| Engine | Mô tả |
|-----|------|
| FunASR | Nhận dạng cục bộ, cần tải model |
| Doubao ASR | API đám mây |

---

### Step 4: Cấu hình LLM

Chọn mô hình ngôn ngữ lớn:

<!-- 截图位置：LLM配置界面 -->
> Hình: Giao diện hướng dẫn cấu hình LLM

| Engine | Mô tả |
|-----|------|
| Tương thích OpenAI | Hỗ trợ nhiều loại API |
| Ollama | Triển khai cục bộ |
| Doubao | Doubao của ByteDance |

---

### Step 5: Cấu hình TTS

Chọn engine tổng hợp giọng nói:

<!-- 截图位置：TTS配置界面 -->
> Hình: Giao diện hướng dẫn cấu hình TTS

| Engine | Mô tả |
|-----|------|
| Doubao TTS | API đám mây |
| EdgeTTS | TTS miễn phí của Microsoft |
| CosyVoice | Chất lượng cao cục bộ |

---

## II. Kiểm tra cấu hình

### Kiểm tra từng cấu hình

Tại các trang cấu hình, nhấn nút "Kiểm tra" ở bên phải mục cấu hình:

<!-- 截图位置：单个配置测试按钮 -->
> Hình: Nút kiểm tra cấu hình

Giải thích kết quả kiểm tra:

| Trường | Mô tả |
|-----|------|
| Trạng thái | Thành công / Thất bại |
| Độ trễ gói đầu | Thời gian phản hồi tính bằng mili giây |
| Thông báo | Chi tiết lỗi (nếu thất bại) |

<!-- 截图位置：测试结果弹窗 -->
> Hình: Hộp thoại kết quả kiểm tra cấu hình

### Kiểm tra hàng loạt

Tại trang quản lý cấu hình, nhấn "Kiểm tra tất cả" để kiểm tra hàng loạt tất cả cấu hình:

<!-- 截图位置：批量测试界面 -->
> Hình: Giao diện kiểm tra hàng loạt

### Các loại kiểm tra được hỗ trợ

| Loại kiểm tra | Mô tả |
|---------|------|
| VAD | Kết nối và thời gian phản hồi phát hiện hoạt động giọng nói |
| ASR | Kết nối và độ trễ gói đầu nhận dạng giọng nói |
| LLM | Kết nối và độ trễ gói đầu suy luận mô hình lớn |
| TTS | Kết nối và độ trễ gói đầu tổng hợp giọng nói |
| OTA | Kiểm tra kết nối MQTT/UDP |

---

## III. Giám sát độ trễ

Xem thống kê độ trễ gói đầu của các module trong hệ thống:

<!-- 截图位置：延迟监控界面 -->
> Hình: Giao diện giám sát độ trễ

### Gợi ý tối ưu hóa độ trễ

| Module | Hướng tối ưu hóa |
|-----|---------|
| ASR | Sử dụng model cục bộ hoặc node API gần nhất |
| LLM | Chọn model nhỏ hơn hoặc dùng đầu ra streaming |
| TTS | Sử dụng edge TTS hoặc model cục bộ |

---

## IV. Quản lý cấu hình

### Chỉnh sửa cấu hình

Vào "Quản lý cấu hình" → Module tương ứng → Chỉnh sửa mục cấu hình

<!-- 截图位置：配置管理界面 -->
> Hình: Giao diện quản lý cấu hình

### Bật/Tắt cấu hình

Dùng công tắc để kiểm soát cấu hình có hiệu lực hay không.

### Đặt cấu hình mặc định

Mỗi module có thể đặt một cấu hình mặc định, được dùng khi thiết bị chưa chỉ định cấu hình cụ thể.

---

## Câu hỏi thường gặp

### Q1: Kiểm tra cấu hình thất bại?

1. Kiểm tra kết nối mạng
2. Xác minh API key có chính xác không
3. Xem log console của chương trình chính

### Q2: Làm thế nào để khôi phục cấu hình mặc định?

Xóa các file cấu hình trong thư mục `config/`, sau đó khởi động lại dịch vụ.

### Q3: Cấu hình sau khi chỉnh sửa có cần khởi động lại không?

Hầu hết các thay đổi cấu hình có hiệu lực ngay lập tức, tuy nhiên một số module có thể cần khởi động lại kết nối thiết bị.
