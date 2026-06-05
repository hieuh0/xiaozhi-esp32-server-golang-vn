# Dịch vụ Mock ASR/LLM/TTS độc lập (không thay đổi chương trình chính)

Phương án này cung cấp một tiến trình dịch vụ mock **chạy độc lập**, dùng để thay thế các dịch vụ đám mây ASR/LLM/TTS thực tế trong quá trình kiểm thử tải (stress test).

## 1. Khởi động

```bash
go run ./cmd/mock_ai_server \
  -addr :18080 \
  -asr-text "你好，这是压测mock识别结果" \
  -llm-reply "这是mock llm回复" \
  -tts-mode silence
```

Kiểm tra sức khỏe dịch vụ:

```bash
curl http://127.0.0.1:18080/healthz
```

## 2. Các endpoint được cung cấp

- `ws://127.0.0.1:18080/asr/`
  - Tương thích kiểu kết nối WebSocket theo phong cách FunASR (nhận các khung nhị phân âm thanh)
  - Sau khi nhận `{"is_speaking": false}`, trả về kết quả nhận dạng cuối cùng

- `POST http://127.0.0.1:18080/v1/chat/completions`
  - Endpoint tương thích OpenAI Chat Completions
  - Hỗ trợ `stream=false/true`

- `POST http://127.0.0.1:18080/v1/audio/speech`
  - Endpoint tương thích OpenAI TTS
  - Trả về `audio/wav` (im lặng hoặc beep)

## 3. Khuyến nghị cấu hình chương trình chính (chỉ thay đổi cấu hình, không thay đổi mã nguồn)

### ASR (FunASR)

- `host=127.0.0.1`
- `port=18080`
- Đường dẫn giao thức theo triển khai hiện tại sử dụng `ws://host:port/`; nếu tầng cấu hình của bạn yêu cầu đường dẫn cụ thể, hãy dùng `/asr/`.

> Nếu adapter ASR hiện tại của bạn phụ thuộc chặt vào đường dẫn gốc `ws://host:port/`, bạn cũng có thể chuyển tiếp `/` sang `/asr/` ở tầng gateway.

### LLM (tương thích OpenAI)

- provider chọn `eino` (`type=openai`)
- `base_url=http://127.0.0.1:18080/v1`
- `api_key` là bất kỳ giá trị nào khác rỗng
- `model_name` là bất kỳ giá trị nào (ví dụ: `mock-gpt`)

### TTS (tương thích OpenAI)

- provider chọn `openai`
- `api_url=http://127.0.0.1:18080/v1/audio/speech`
- `response_format=wav`
- `api_key` là bất kỳ giá trị nào khác rỗng

## 4. Các tham số có thể điều chỉnh

```bash
-asr-delay-ms         # Độ trễ trả về kết quả cuối của ASR
-llm-first-delay-ms   # Độ trễ token đầu tiên của LLM
-llm-chunk-delay-ms   # Độ trễ giữa các chunk trong luồng streaming của LLM
-tts-first-delay-ms   # Độ trễ gói đầu tiên của TTS
-tts-mode             # silence|beep
-tts-duration-ms      # Thời lượng âm thanh trả về
```

## 5. Khuyến nghị kiểm thử tải

1. Trước tiên xác minh kết nối đơn lẻ ở máy cục bộ (đảm bảo thiết bị có thể đi qua toàn bộ luồng xử lý và nhận được âm thanh).
2. Sau đó dùng `ws_multi` để kiểm thử đồng thời theo bậc thang (ví dụ: 50/100/200/500).
3. Sử dụng các tổ hợp delay khác nhau để mô phỏng biến động của các phụ thuộc bên ngoài trong thực tế, quan sát P95/P99 và tỷ lệ lỗi.


## 6. Đánh giá: ws_multi có cần tối ưu không

Kết luận: **Khuyến nghị thực hiện tối ưu nhỏ, không nhất thiết phải tái cấu trúc lớn**. Hiện tại có thể dùng trực tiếp để kiểm thử tải, nhưng để đo lường chính xác hơn "hiệu năng của dịch vụ chính" thay vì "điểm nghẽn của client kiểm thử", khuyến nghị bổ sung các khả năng sau:

1. **Thêm chế độ phát lại âm thanh thuần túy (ưu tiên làm trước)**
   - Hiện nay cách làm phổ biến là TTS ở client trước rồi mới đẩy âm thanh, điều này đưa thời gian TTS phía client vào kết quả đo.
   - Khuyến nghị thêm `-audio_file`/`-audio_dir` để gửi trực tiếp các khung opus đã mã hóa sẵn hoặc wav đã chuyển đổi sang opus.

2. **Xuất thống kê độ trễ dạng có cấu trúc**
   - Thêm thống kê RT của khung đầu tiên, RT hoàn thành toàn bộ luồng, phân loại mã lỗi.
   - Khuyến nghị xuất JSONL để dễ tổng hợp P95/P99 trong xử lý hậu kỳ.

3. **Kiểm soát throttle kết nối và gửi dữ liệu**
   - Thêm cơ chế khởi tạo kết nối theo lô (ví dụ: khởi động N client mỗi giây), tránh kết nối đồng thời tức thời làm khuếch đại biến động phía client.
   - Thêm tham số jitter gói tin, mô phỏng mạng thiết bị thực tế.

4. **Cấu hình được chiến lược retry và timeout khi gặp lỗi**
   - Ví dụ: `-dial_timeout`, `-read_timeout`, `-retry`, nâng cao độ ổn định trong các bài kiểm thử tải dài.

5. **Thu thập chỉ số tài nguyên (tùy chọn)**
   - Ghi lại CPU/bộ nhớ của client, giúp phân biệt "điểm nghẽn phía server" và "điểm nghẽn của máy kiểm thử".

Trong phương án "dịch vụ mock độc lập" này, `ws_multi` **không sửa vẫn chạy được**, nhưng khuyến nghị ít nhất thực hiện mục 1 và 2 — kết quả kiểm thử tải sẽ đáng tin cậy hơn đáng kể.
