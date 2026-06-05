# Hướng dẫn tích hợp giao diện IndexTTS vLLM

Tài liệu này mô tả các yêu cầu đối với giao diện phía máy chủ khi tích hợp `indextts_vllm` vào dự án, áp dụng cho:

- Suy luận TTS chương trình chính (`/audio/speech`)
- Giao diện quản trị lấy danh sách giọng đọc (`/audio/voices`)
- Nhân bản giọng nói người dùng (`/audio/clone`, dùng cho quy trình nhân bản trong dự án này)

## 1. Danh sách kiểm tra tương thích nhanh

Dịch vụ IndexTTS của bạn cần đáp ứng tối thiểu ba điểm sau:

- Cung cấp `POST /audio/speech`, tham số đầu vào tương thích với phong cách OpenAI TTS: `input`, `voice`, `model`
- Cung cấp `GET /audio/voices`, trả về danh sách giọng đọc có thể liệt kê (đối tượng JSON)
- Nếu sử dụng tính năng "nhân bản giọng nói" của dự án này, cung cấp `POST /audio/clone` (`multipart/form-data`)

Định dạng âm thanh trả về được khuyến nghị: `audio/wav` (16-bit PCM).

## 2. Ánh xạ cấu hình (Quản trị viên -> Cấu hình TTS -> IndexTTS(vLLM))

| Trường phía quản trị | Mục đích | Vị trí gửi |
| --- | --- | --- |
| `api_url` | Địa chỉ dịch vụ IndexTTS | Dùng làm URL cơ sở, ghép với endpoint |
| `api_key` | Xác thực tùy chọn | `Authorization: Bearer <api_key>` |
| `model` | Tên mô hình | Trường `model` trong body yêu cầu `/audio/speech` |
| `voice` | Giọng đọc mặc định | Trường `voice` trong body yêu cầu `/audio/speech` |
| `frame_duration` | Thời lượng khung (ms) | Tham số cắt khung âm thanh cục bộ |

Ghi chú:

- Khi nhấp vào danh sách thả xuống "Giọng đọc" trên giao diện quản trị, hệ thống sẽ dùng giá trị `api_url` mới nhất trong ô nhập để lấy `/audio/voices`.
- `api_url` hỗ trợ nhập địa chỉ cơ sở (ví dụ: `http://127.0.0.1:7860`) hoặc đường dẫn cụ thể (ví dụ: `/audio/speech`).

## 3. Yêu cầu giao diện

### 3.1 `GET /audio/voices`

Mục đích: Danh sách thả xuống "Giọng đọc" trên trang cấu hình quản trị, tùy chọn giọng đọc phía người dùng.

Header yêu cầu:

- `Accept: application/json`
- `Authorization: Bearer <api_key>` (tùy chọn)

Ví dụ phản hồi (khuyến nghị):

```json
{
  "demo_speaker": ["assets/speaker/demo.wav"],
  "narrator_cn_female": ["assets/speaker/narrator_cn_female.wav"]
}
```

Yêu cầu:

- Loại trả về được khuyến nghị là đối tượng JSON (tên khóa sẽ được dùng làm ID giọng đọc).
- Dự án này sẽ lọc bỏ các giọng hệ thống có tiền tố `indextts_vllm`, sau đó thêm vào các giọng nhân bản của người dùng.

### 3.2 `POST /audio/speech`

Mục đích: Tổng hợp TTS của chương trình chính, nghe thử sau khi nhân bản.

Header yêu cầu:

- `Content-Type: application/json`
- `Accept: audio/wav,application/octet-stream,*/*`
- `Authorization: Bearer <api_key>` (tùy chọn)

Ví dụ body yêu cầu:

```json
{
  "model": "indextts-vllm",
  "input": "你好，欢迎使用 IndexTTS。",
  "voice": "demo_speaker"
}
```

Phản hồi:

- Thành công: Luồng âm thanh nhị phân (khuyến nghị `audio/wav`)
- Thất bại: HTTP 4xx/5xx, kèm thông báo lỗi có thể đọc được

### 3.3 `POST /audio/clone` (cần thiết cho tính năng nhân bản của dự án)

Mục đích: Được gọi khi `/user/voice-clones` gửi tác vụ nhân bản.

Loại yêu cầu: `multipart/form-data`

Các trường form:

- `voice`: ID giọng đọc mong muốn tạo ra
- `audio`: File âm thanh tham chiếu (wav/mp3/m4a, v.v.)

Ví dụ phản hồi:

```json
{
  "voice": "demo_speaker_clone_001",
  "ok": true
}
```

Yêu cầu:

- Khuyến nghị phản hồi chứa trường `voice`; nếu thiếu, dự án này sẽ dùng giá trị trường `voice` trong yêu cầu thay thế.

## 4. Tham khảo tương thích (api_server.py)

Có thể tham khảo phong cách triển khai sau:

- `POST /audio/speech`: Đọc `input`, `voice`, `model`
- `GET /audio/voices`: Trả về từ điển giọng đọc khả dụng

Liên kết tham khảo:

- https://github.com/hackers365/index-tts-vllm/blob/master/api_server.py

## 5. Xử lý sự cố thường gặp

### 5.1 Lỗi khi nhấp vào danh sách thả xuống giọng đọc trên trang quản trị

Kiểm tra trước tiên:

- `api_url` có thể truy cập được không (giá trị nhập mới nhất)
- `/audio/voices` có trả về đối tượng JSON không
- Có cần `api_key` không

### 5.2 Tổng hợp thành công nhưng phát âm thanh bất thường

Kiểm tra trước tiên:

- Máy chủ có trả về WAV chuẩn không (PCM16, tốc độ lấy mẫu đúng)
- Có bị chuyển mã hoặc cắt bớt ở các liên kết trung gian không
- Header phản hồi `Content-Type` có đúng không

### 5.3 Tác vụ nhân bản thất bại

Kiểm tra trước tiên:

- `/audio/clone` có chấp nhận yêu cầu multipart với `voice + audio` không
- JSON phản hồi có thể phân tích được không, có chứa `voice` khả dụng không
