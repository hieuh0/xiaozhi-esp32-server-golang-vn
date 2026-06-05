# Tài liệu tính năng nhận dạng giọng nói (Speaker Identification)

> Nhận dạng giọng nói (Speaker Identification) là một tính năng cốt lõi trong dự án xiaozhi-esp32-server-golang, dùng để nhận dạng danh tính người dùng phía thiết bị và tự động chuyển đổi giọng TTS dựa trên kết quả nhận dạng.

---

## I. Tổng quan tính năng

Nhận dạng giọng nói hoạt động bằng cách trích xuất đặc trưng giọng nói (embedding) của người dùng, so sánh với dữ liệu giọng đã đăng ký sẵn để xác định danh tính.

### Khả năng cốt lõi

| Khả năng | Mô tả |
|------|------|
| 🎤 **Đăng ký giọng nói** | Tải lên mẫu âm thanh của người dùng, trích xuất đặc trưng giọng và lưu trữ |
| 🔍 **Nhận dạng giọng nói** | Nhận dạng danh tính người nói theo thời gian thực |
| ✅ **Xác minh giọng nói** | Xác minh âm thanh có thuộc về người dùng chỉ định hay không |
| 📡 **Nhận dạng theo luồng** | Nhận dạng giọng nói theo luồng thời gian thực qua WebSocket |
| 🔊 **Chuyển đổi TTS động** | Tự động chuyển đổi giọng TTS tương ứng dựa trên kết quả nhận dạng |

---

## II. Kiến trúc hệ thống

### 2.1 Kiến trúc tổng thể

```
┌──────────────────┐     ┌──────────────────────┐     ┌──────────────────┐
│   Thiết bị ESP32 │────▶│ xiaozhi-esp32-server │────▶│   voice-server   │
│  (thu âm thanh)  │     │   (dịch vụ chính)    │     │(dịch vụ nhận dạng│
└──────────────────┘     └──────────────────────┘     │   giọng nói)     │
                                                       └──────────────────┘
                                                              │
                                                              ▼
                                                      ┌──────────────────┐
                                                      │  Qdrant vector DB│
                                                      │(lưu đặc trưng    │
                                                      │   giọng nói)     │
                                                      └──────────────────┘
```

### 2.2 Mô tả các thành phần

| Thành phần | Trách nhiệm |
|------|------|
| **xiaozhi-esp32-server** | Dịch vụ chính, quản lý kết nối thiết bị, quản lý phiên, xử lý kết quả nhận dạng giọng nói |
| **voice-server (asr_server)** | Dịch vụ nhận dạng giọng nói, trích xuất đặc trưng, đăng ký, nhận dạng, xác minh |
| **Manager (quản trị web)** | Giao diện quản trị web, cung cấp API và UI quản lý nhóm giọng nói và mẫu âm thanh |
| **Qdrant** | Cơ sở dữ liệu vector, lưu trữ các vector đặc trưng giọng nói |

---

## III. Mô tả luồng xử lý đầy đủ

### 3.1 Luồng đăng ký giọng nói

```
Người dùng tải âm thanh → Manager API → voice-server đăng ký → trích xuất embedding → lưu vào Qdrant
                  │
                  ▼
            Lưu file cục bộ + ghi nhận vào cơ sở dữ liệu
```

**Các bước chi tiết:**

1. Người dùng tải lên file âm thanh (định dạng WAV) qua giao diện Manager Web
2. Backend Manager tạo UUID duy nhất, lưu file âm thanh vào bộ nhớ cục bộ
3. Gọi endpoint `/api/v1/speaker/register` của voice-server
4. voice-server dùng mô hình sherpa-onnx trích xuất đặc trưng giọng (vector 192 chiều)
5. Đặc trưng giọng được lưu vào cơ sở dữ liệu vector Qdrant
6. Manager tạo bản ghi `SpeakerSample` trong cơ sở dữ liệu

### 3.2 Luồng nhận dạng giọng nói theo thời gian thực

```
ESP32 thu âm → VAD phát hiện giọng → gửi đồng thời đến ASR và nhận dạng giọng nói
                                        │
                                        ▼
                              Nhận dạng theo luồng qua WebSocket
                                        │
                                        ▼
                              Lấy kết quả khi kết thúc giọng nói
                                        │
                                        ▼
                              Chuyển đổi giọng TTS theo kết quả nhận dạng
```

**Các bước chi tiết:**

1. **Phát hiện VAD**: Âm thanh từ ESP32 được xử lý qua VAD (Voice Activity Detection)
2. **Gửi hai kênh**: Khi phát hiện giọng nói, dữ liệu âm thanh được gửi đồng thời đến:
   - Dịch vụ ASR (chuyển giọng thành văn bản)
   - Dịch vụ nhận dạng giọng nói (nhận dạng theo luồng WebSocket)
3. **Xử lý theo luồng**: Dịch vụ nhận dạng giọng nói liên tục nhận các khối âm thanh
4. **Lấy kết quả**: Khi phát hiện kết thúc giọng nói (im lặng), gọi `FinishAndIdentify` để lấy kết quả nhận dạng
5. **Chuyển đổi TTS**: Dựa trên kết quả nhận dạng, tự động chuyển đổi giọng TTS theo cấu hình của người dùng tương ứng

### 3.3 Điều kiện kích hoạt

Nhận dạng giọng nói chỉ khởi động khi thỏa mãn đồng thời các điều kiện sau:

- `voice_identify.enable = true`: Bật nhận dạng giọng nói trong cấu hình toàn cục
- Cấu hình thiết bị có chứa cấu hình nhóm giọng nói
- `speakerManager` đã khởi tạo thành công

---

## IV. Mô tả cấu hình

### 4.1 Cấu hình chương trình chính (config.yaml)

Thêm cấu hình sau vào `config.yaml`:

```yaml
# Cấu hình nhận dạng giọng nói
voice_identify:
  enable: true                              # Có bật nhận dạng giọng nói hay không
  base_url: "http://voice-server:8080"      # Địa chỉ dịch vụ voice-server
  threshold: 0.6                            # Ngưỡng nhận dạng giọng nói, phạm vi 0.0-1.0
```

| Tham số cấu hình | Kiểu | Giá trị mặc định | Mô tả |
|--------|------|--------|------|
| `enable` | bool | false | Có bật tính năng nhận dạng giọng nói hay không |
| `base_url` | string | - | Địa chỉ HTTP của dịch vụ voice-server |
| `threshold` | float | 0.6 | Ngưỡng nhận dạng, giá trị càng cao yêu cầu khớp càng chặt |

### 4.2 Cấu hình Docker Compose

#### Biến môi trường dịch vụ Backend

```yaml
backend:
  environment:
    - SPEAKER_SERVICE_URL=http://voice-server:8080
```

#### Biến môi trường dịch vụ voice-server

```yaml
voice-server:
  environment:
    - VAD_ASR_SPEAKER_ENABLED=true
    - VAD_ASR_SPEAKER_VECTOR_DB_HOST=qdrant
    - VAD_ASR_SPEAKER_VECTOR_DB_PORT=6334
    - VAD_ASR_SPEAKER_VECTOR_DB_COLLECTION_NAME=speaker_embeddings
    - VAD_ASR_SPEAKER_THRESHOLD=0.6
    - VAD_ASR_LOGGING_LEVEL=info
```

| Biến môi trường | Mô tả |
|----------|------|
| `VAD_ASR_SPEAKER_ENABLED` | Có bật tính năng nhận dạng giọng nói hay không |
| `VAD_ASR_SPEAKER_VECTOR_DB_HOST` | Địa chỉ dịch vụ Qdrant |
| `VAD_ASR_SPEAKER_VECTOR_DB_PORT` | Cổng gRPC của Qdrant |
| `VAD_ASR_SPEAKER_VECTOR_DB_COLLECTION_NAME` | Tên Collection trong Qdrant |
| `VAD_ASR_SPEAKER_THRESHOLD` | Ngưỡng nhận dạng giọng nói |
| `VAD_ASR_LOGGING_LEVEL` | Mức độ log |

---

## V. Mô tả API

### 5.1 API quản trị Manager

#### Quản lý nhóm giọng nói

| Phương thức | Đường dẫn | Mô tả |
|------|------|------|
| POST | `/api/speaker-groups` | Tạo nhóm giọng nói |
| GET | `/api/speaker-groups` | Lấy danh sách nhóm giọng nói |
| GET | `/api/speaker-groups/:id` | Lấy chi tiết nhóm giọng nói |
| PUT | `/api/speaker-groups/:id` | Cập nhật nhóm giọng nói |
| DELETE | `/api/speaker-groups/:id` | Xóa nhóm giọng nói |
| POST | `/api/speaker-groups/:id/verify` | Xác minh giọng nói |

#### Quản lý mẫu giọng nói

| Phương thức | Đường dẫn | Mô tả |
|------|------|------|
| POST | `/api/speaker-groups/:id/samples` | Thêm mẫu giọng nói |
| GET | `/api/speaker-groups/:id/samples` | Lấy danh sách mẫu |
| GET | `/api/speaker-samples/:id/audio` | Lấy file âm thanh mẫu |
| DELETE | `/api/speaker-samples/:id` | Xóa mẫu |

### 5.2 API voice-server

#### Giao diện HTTP

| Phương thức | Đường dẫn | Mô tả |
|------|------|------|
| POST | `/api/v1/speaker/register` | Đăng ký giọng nói |
| POST | `/api/v1/speaker/identify` | Nhận dạng giọng nói |
| POST | `/api/v1/speaker/verify` | Xác minh giọng nói |
| GET | `/api/v1/speaker/list` | Lấy danh sách tất cả người nói |
| DELETE | `/api/v1/speaker/:id` | Xóa người nói |
| GET | `/api/v1/speaker/stats` | Lấy thông tin thống kê |

#### Nhận dạng theo luồng WebSocket

**Địa chỉ kết nối:** `ws://voice-server:8080/api/v1/speaker/stream`

**Luồng tin nhắn:**

1. Client gửi khối âm thanh (PCM float32, little-endian)
2. Client gửi lệnh hoàn thành: `{"action": "finish"}`
3. Server trả về kết quả nhận dạng

---

## VI. Cơ sở dữ liệu vector (Qdrant)

### 6.1 Cấu trúc lưu trữ dữ liệu

```json
{
    "uid": "ID người dùng",
    "agent_id": "ID agent",
    "speaker_id": "ID người nói (khóa chính nhóm giọng nói)",
    "speaker_name": "Tên người nói (tên nhóm giọng nói)",
    "uuid": "Định danh duy nhất của mẫu",
    "sample_index": 0,
    "created_at": 1704672000,
    "updated_at": 1704672000
}
```

### 6.2 Cấu hình vector

| Cấu hình | Giá trị |
|------|-----|
| Số chiều vector | 192 |
| Độ đo khoảng cách | Cosine (độ tương đồng cosine) |
| Tên Collection | `speaker_embeddings` (có thể cấu hình) |

### 6.3 Cách ly dữ liệu

Hỗ trợ cách ly dữ liệu đa chiều:

- **UID**: Cách ly theo cấp người dùng
- **Agent ID**: Cách ly theo cấp agent
- Các agent khác nhau của cùng một người dùng có thể có dữ liệu giọng nói độc lập

---

## VII. Cấu trúc bảng cơ sở dữ liệu

### 7.1 SpeakerGroup (Bảng nhóm giọng nói)

```sql
CREATE TABLE `speaker_groups` (
  `id` INT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` INT UNSIGNED NOT NULL COMMENT 'ID người dùng sở hữu',
  `agent_id` INT UNSIGNED NOT NULL COMMENT 'ID agent liên kết',
  `name` VARCHAR(100) NOT NULL COMMENT 'Tên giọng nói',
  `prompt` TEXT COMMENT 'Prompt nhân vật',
  `description` TEXT COMMENT 'Thông tin mô tả',
  `tts_config_id` VARCHAR(100) COMMENT 'ID cấu hình TTS',
  `voice` VARCHAR(200) COMMENT 'Giá trị giọng nói',
  `status` VARCHAR(20) NOT NULL DEFAULT 'active',
  `sample_count` INT NOT NULL DEFAULT 0 COMMENT 'Số lượng mẫu',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
);
```

### 7.2 SpeakerSample (Bảng mẫu giọng nói)

```sql
CREATE TABLE `speaker_samples` (
  `id` INT UNSIGNED NOT NULL AUTO_INCREMENT,
  `speaker_group_id` INT UNSIGNED NOT NULL COMMENT 'ID nhóm giọng nói liên kết',
  `user_id` INT UNSIGNED NOT NULL COMMENT 'ID người dùng sở hữu',
  `uuid` VARCHAR(36) NOT NULL COMMENT 'Định danh UUID duy nhất',
  `file_path` VARCHAR(500) NOT NULL COMMENT 'Đường dẫn lưu trữ file âm thanh cục bộ',
  `file_name` VARCHAR(255) COMMENT 'Tên file gốc',
  `file_size` BIGINT COMMENT 'Kích thước file (byte)',
  `duration` FLOAT COMMENT 'Thời lượng âm thanh (giây)',
  `status` VARCHAR(20) NOT NULL DEFAULT 'active',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `idx_uuid` (`uuid`)
);
```

---

## VIII. Hướng dẫn sử dụng

### 8.1 Triển khai voice-server

Tham khảo cấu hình triển khai đầy đủ trong [docker_compose.md](docker_compose.md), đảm bảo các dịch vụ sau đã được khởi động:

- **Qdrant**: Cơ sở dữ liệu vector
- **voice-server**: Dịch vụ nhận dạng giọng nói

### 8.2 Cấu hình chương trình chính

Thêm cấu hình nhận dạng giọng nói vào `config.yaml` của chương trình chính:

```yaml
voice_identify:
  enable: true
  base_url: "http://voice-server:8080"
  threshold: 0.6
```

### 8.3 Tạo nhóm giọng nói

1. Đăng nhập vào Manager Web Console
2. Vào "Agent" → chọn agent mục tiêu → "Quản lý giọng nói"
3. Nhấn "Tạo nhóm giọng nói mới", điền tên, mô tả và các thông tin khác
4. Cấu hình giọng TTS tương ứng (tùy chọn)

### 8.4 Tải lên mẫu giọng nói

1. Nhấn "Thêm mẫu" trên trang chi tiết nhóm giọng nói
2. Tải lên file âm thanh định dạng WAV (khuyến nghị giọng nói rõ ràng 3-10 giây)
3. Hệ thống tự động trích xuất đặc trưng giọng nói và lưu trữ

### 8.5 Kiểm tra nhận dạng giọng nói

1. Nhấn "Xác minh" trên trang chi tiết nhóm giọng nói
2. Tải lên âm thanh thử nghiệm
3. Xem kết quả nhận dạng và độ tin cậy

---

## IX. Các điểm kỹ thuật quan trọng

### 9.1 Trích xuất đặc trưng giọng nói

- Sử dụng mô hình **sherpa-onnx** để trích xuất đặc trưng giọng nói
- Đầu ra là vector embedding 192 chiều
- Hỗ trợ đầu vào với bất kỳ tần số lấy mẫu nào, tự động resample

### 9.2 Tính toán độ tương đồng

- Sử dụng **độ tương đồng cosine** (Cosine Similarity) để tính mức độ khớp giọng nói
- Phạm vi độ tương đồng: [-1, 1]
- Ngưỡng mặc định 0.6, có thể điều chỉnh theo tình huống thực tế

### 9.3 Tiền xử lý VAD

- Sử dụng TEN-VAD để lọc khoảng lặng
- Khi đăng ký, giữ lại biên 100ms im lặng trước và sau
- Khi nhận dạng thời gian thực, chỉ gửi các đoạn âm thanh được phát hiện có giọng nói

---

## X. Câu hỏi thường gặp

### Q1: Nhận dạng giọng nói không hoạt động?

Kiểm tra các cấu hình sau:
1. `voice_identify.enable` có đang là `true` không
2. `voice_identify.base_url` có đúng không
3. Thiết bị đã được cấu hình nhóm giọng nói chưa
4. Dịch vụ voice-server có đang chạy bình thường không

### Q2: Độ chính xác nhận dạng thấp?

- Nâng cao chất lượng mẫu giọng nói (rõ ràng, không tiếng ồn, 3-10 giây)
- Tăng số lượng mẫu giọng nói (khuyến nghị 3-5 mẫu)
- Điều chỉnh ngưỡng nhận dạng

### Q3: Giọng TTS không chuyển đổi?

Kiểm tra xem trường `tts_config_id` hoặc `voice` trong cấu hình nhóm giọng nói đã được cấu hình đúng chưa.

---

## XI. Tài liệu liên quan

- [Triển khai Docker Compose](docker_compose.md)
- [Tài liệu cấu hình](config.md)
- [Nhận dạng hình ảnh](vision.md)
