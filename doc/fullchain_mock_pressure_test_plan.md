# Phương án Mock Kiểm thử Áp lực Toàn Chuỗi VAD/ASR/LLM/TTS (Chờ Xác nhận)

> Mục tiêu: Không gọi các dịch vụ tính phí ASR/LLM/TTS thực, giữ nguyên hành vi toàn chuỗi WebSocket hiện có, hỗ trợ kiểm thử áp lực đồng thời cao, tiêm độ trễ có kiểm soát và thống kê khả năng quan sát.

## 1. Mục tiêu Thiết kế

1. **Chuỗi đầy đủ**: Giữ nguyên luồng chính "đầu vào âm thanh từ thiết bị -> VAD -> ASR -> LLM -> TTS -> phát âm thanh".
2. **Chi phí ngoài bằng không**: ASR/LLM/TTS đều trả về dữ liệu mock cục bộ, không truy cập dịch vụ đám mây bên thứ ba.
3. **Ít xâm phạm**: Dựa trên cơ chế factory provider hiện có để mở rộng `mock` provider, hạn chế tối đa thay đổi luồng nghiệp vụ chính.
4. **Có thể kiểm thử áp lực và tái hiện**: Hỗ trợ trả về cố định, trả về theo mẫu, tiêm lỗi theo xác suất, tiêm độ trễ theo cấu hình.
5. **Có thể đối chiếu với dịch vụ thực**: Thông qua chuyển đổi cấu hình, có thể khôi phục provider thực bất cứ lúc nào để so sánh hiệu năng.

## 2. Phương án Tổng thể

Sử dụng phương án **Mock cấp Provider + Tái sử dụng Client Kiểm thử Áp lực**:

- Thêm mới ba provider:
  - `asr/mock`
  - `llm/mock`
  - `tts/mock`
- Thêm mục cấu hình tương ứng trong cấu hình backend (`type=asr|llm|tts`, `provider=mock`).
- Liên kết cấu hình mock thông qua role/agent để thực hiện mock toàn chuỗi trong phiên.
- Phía kiểm thử áp lực tiếp tục sử dụng công cụ kiểm thử áp lực websocket hiện có (`ws_multi`) để đẩy âm thanh đồng thời.

Điều này đảm bảo:
- Giao thức WebSocket, state machine phiên, logic điều phối message đều đi qua đường mã thực.
- Chỉ thay thế các lời gọi đến dịch vụ đám mây ngoài, chi phí thấp nhất, rủi ro nhỏ nhất.

## 3. Thiết kế Hành vi Mock

### 3.1 ASR Mock

Đầu vào: luồng khung âm thanh (giữ nguyên interface hiện có).
Đầu ra: văn bản nhận dạng (cố định / vòng quay / theo quy tắc).

Cấu hình đề xuất:

- `mode`: `fixed` | `sequence` | `echo_hint`
- `fixed_text`: trả về cố định, ví dụ "Xin chào, đây là văn bản kiểm thử áp lực"
- `sequence_texts`: mảng văn bản, xoay vòng theo yêu cầu
- `first_token_delay_ms`: mô phỏng độ trễ gói đầu tiên
- `final_delay_ms`: mô phỏng độ trễ gói kết thúc
- `error_rate`: xác suất 0~1 tiêm lỗi nhận dạng

### 3.2 LLM Mock

Đầu vào: văn bản ASR + các message ngữ cảnh.
Đầu ra: văn bản phản hồi (có thể mang thông tin độ dài ngữ cảnh).

Cấu hình đề xuất:

- `mode`: `fixed` | `template` | `echo`
- `fixed_answer`: phản hồi cố định
- `template`: mẫu, ví dụ `"Đã nhận: {{input}}"`
- `first_token_delay_ms`: độ trễ token đầu tiên
- `stream_chunk_chars`: số ký tự mỗi đoạn khi stream
- `total_delay_ms`: mô phỏng tổng thời gian hoàn thành
- `error_rate`: lỗi theo xác suất

### 3.3 TTS Mock

Đầu vào: văn bản LLM.
Đầu ra: khung Opus/PCM có thể phát (ưu tiên Opus, tương thích chuỗi hiện tại).

Cấu hình đề xuất:

- `audio_source`: `builtin_silence` | `builtin_beep` | `file`
- `file_path`: đường dẫn âm thanh được cài sẵn (wav/opus cục bộ)
- `frame_duration_ms`: độ dài mỗi khung (ví dụ 20ms)
- `first_frame_delay_ms`: độ trễ khung đầu tiên
- `inter_frame_delay_ms`: độ trễ giữa các khung
- `error_rate`: lỗi theo xác suất

> Để giảm độ phức tạp, phiên bản đầu tiên đề xuất: trả về "khung im lặng + độ trễ cố định" trước, sau đó bổ sung "beep/phát lại file".

## 4. Ma trận Kịch bản Kiểm thử Áp lực

### Kịch bản A: Chuỗi thành công thuần túy (cơ sở)
- ASR văn bản cố định
- LLM phản hồi ngắn cố định
- TTS khung im lặng
- Mục tiêu: đo đồng thời ổn định tối đa, RT trung bình, P95/P99

### Kịch bản B: Chuỗi độ trễ cao
- ASR/LLM/TTS mỗi bước tiêm độ trễ 100~500ms
- Mục tiêu: đo ngưỡng timeout, tình trạng xếp hàng tích lũy

### Kịch bản C: Chuỗi tiêm lỗi
- error_rate đặt 1%/5%/10%
- Mục tiêu: đo khả năng phục hồi lỗi, độ ổn định kết nối, chiến lược retry

### Kịch bản D: Chuỗi văn bản dài
- LLM xuất văn bản rất dài (ví dụ 500~1500 ký tự)
- Mục tiêu: đo phân khung TTS, back pressure gửi và độ ổn định bộ nhớ

## 5. Chỉ số và Tiêu chuẩn Nghiệm thu (Đề xuất)

Chỉ số cốt lõi:
- Tỷ lệ thành công phiên (trả về giọng nói thành công)
- Độ trễ khung đầu tiên end-to-end (listen stop -> gói âm thanh đầu tiên)
- Độ trễ hoàn thành end-to-end (listen stop -> tts finish)
- Số phiên hoạt động mỗi giây / đỉnh đồng thời
- Tỷ lệ lỗi (phân theo giai đoạn ASR/LLM/TTS)
- Tài nguyên dịch vụ: CPU, bộ nhớ, Goroutine, số lần GC

Nghiệm thu đề xuất (có thể điều chỉnh sau):
- Tỷ lệ thành công >= 99%
- P95 độ trễ khung đầu tiên < 1.5s ở mức đồng thời mục tiêu
- Không rò rỉ bộ nhớ rõ ràng trong 30 phút liên tục (biến động RSS có thể kiểm soát)

## 6. Các Bước Triển khai (Chia hai giai đoạn)

### Phase 1 (Khả dụng tối thiểu, 1~2 ngày)
1. Thêm đăng ký ba mock provider ASR/LLM/TTS.
2. Mỗi provider hỗ trợ trả về cố định + độ trễ cố định + tỷ lệ lỗi.
3. Thêm ba cấu hình mock mới ở backend và có thể đặt làm mặc định.
4. Chạy thông `ws_multi` và xuất kết quả kiểm thử áp lực cơ sở.

### Phase 2 (Tăng cường, 1~2 ngày)
1. Thêm phản hồi theo mẫu, phản hồi tuần tự, phát lại âm thanh từ file.
2. Thêm log chỉ số chi tiết hơn (thời gian tiêu thụ theo từng giai đoạn).
3. Thêm script kiểm thử áp lực (thực thi hàng loạt kịch bản + báo cáo tổng hợp).

## 7. Rủi ro và Biện pháp Phòng tránh

1. **Định dạng âm thanh không khớp**: định dạng đầu ra mock tts cần nhất quán với bộ giải mã downstream hiện tại.
   - Phòng tránh: phiên bản đầu tiên dùng lại đường mã encoding phổ biến hiện có và thêm log kiểm tra định dạng.
2. **Log quá lớn khi đồng thời cao**: log chi tiết đồng thời cao sẽ ảnh hưởng hiệu năng.
   - Phòng tránh: chế độ kiểm thử áp lực giảm cấp log level, các chỉ số quan trọng xuất ra dạng tổng hợp.
3. **Cấu hình nhầm sang dịch vụ thực**: dẫn đến vẫn gọi interface ngoài.
   - Phòng tránh: môi trường kiểm thử áp lực chặn mạng hoặc thêm kiểm tra whitelist provider (không phải mock thì từ chối khởi động).

## 8. Nội dung Triển khai Tôi Sẽ Thực hiện Sau Khi Bạn Xác nhận

Sau khi xác nhận, tôi sẽ sửa code trực tiếp theo danh sách sau:

1. Thêm mới `internal/domain/asr/mock`, `internal/domain/llm/mock`, `internal/domain/tts/mock`.
2. Gắn `mock` provider vào điểm đăng ký provider factory / pool.
3. Bổ sung mẫu cấu hình mặc định (có thể chọn mock trực tiếp trong trang quản trị).
4. Thêm unit test tối thiểu (ít nhất test hành vi provider).
5. Cung cấp một danh sách lệnh thực thi kiểm thử áp lực (bậc thang đồng thời + thu thập chỉ số).

---

## Các Lựa chọn Cần Bạn Xác nhận

Vui lòng xác nhận 4 điểm sau, tôi sẽ bắt đầu cải tạo chính thức:

1. **Độ chi tiết Mock**: Bạn có đồng ý mock theo cấp provider không (khuyến nghị)?
2. **Đầu ra TTS**: Phiên bản đầu tiên có chấp nhận "khung im lặng" làm âm thanh mock không (nhanh nhất)?
3. **Mức đồng thời mục tiêu kiểm thử áp lực**: Mục tiêu ban đầu là bao nhiêu đồng thời (ví dụ 100/300/500)?
4. **Ngưỡng nghiệm thu**: Có thực hiện theo tiêu chuẩn nghiệm thu mặc định của tài liệu này không?
