# Hướng dẫn tính năng nhân bản giọng nói

Tài liệu này mô tả tính năng **Nhân bản giọng nói (Voice Clone)** trong dự án, bao gồm quy trình tạo/nghe thử/thử lại của người dùng thông thường và quản lý hạn ngạch nhân bản của quản trị viên.

Các trang và tài liệu liên quan:

- Quản trị viên `Quản lý cấu hình TTS` (cung cấp cấu hình TTS khả dụng cho người dùng)
- Quản trị viên `Quản lý người dùng -> Hạn ngạch nhân bản`
- Người dùng thông thường `Nhân bản giọng nói`
- [Hướng dẫn sử dụng trang quản trị](./manager_console_guide.md)

---

## 1. Tổng quan tính năng

Tính năng nhân bản giọng nói cho phép người dùng tải lên âm thanh (hoặc ghi âm qua trình duyệt), tạo "giọng nhân bản" trên nhà cung cấp TTS được hỗ trợ, sau đó chọn giọng đó trong agent/nhân vật để phát âm.

Các nhà cung cấp nhân bản hiện được hỗ trợ ở cả frontend lẫn backend:

- `minimax`
- `cosyvoice`
- `aliyun_qwen` (Qianwen)

Các nhà cung cấp TTS không có trong danh sách trên, dù có thể dùng cho tổng hợp TTS thông thường, vẫn không thể dùng cho nhân bản giọng nói.

---

## 2. Vai trò và quyền hạn

### 2.1 Người dùng thông thường

Có thể:

- Tạo giọng nhân bản
- Xem trạng thái tác vụ nhân bản
- Nghe thử âm thanh gốc và âm thanh nhân bản
- Chỉnh sửa tên nhân bản
- Thử lại các tác vụ thất bại

### 2.2 Quản trị viên

Có thể:

- Cấu hình và bật nhà cung cấp TTS hỗ trợ nhân bản
- Thiết lập hạn ngạch nhân bản cho từng người dùng theo `Cấu hình TTS` (tùy chọn)

---

## 3. Điều kiện tiên quyết

Trước khi sử dụng, hãy xác nhận:

1. Quản trị viên đã tạo và bật ít nhất một cấu hình TTS (provider là `minimax` / `cosyvoice` / `aliyun_qwen`)
2. Người dùng thông thường có thể thấy cấu hình TTS đó trong trang "Nhân bản giọng nói"
3. (Tùy chọn) Quản trị viên đã phân bổ hạn ngạch nhân bản cho người dùng đó

Ghi chú:

- Nếu chưa cấu hình hạn ngạch, mặc định tương thích với hành vi cũ, thường được hiểu là "không giới hạn"

---

## 4. Quy trình sử dụng cho người dùng thông thường

Điểm truy cập:

- `Người dùng thông thường -> Nhân bản giọng nói`

## 4.1 Tạo giọng nhân bản

Nhấn `Tạo giọng nhân bản`, điền vào:

- `Tên nhân bản` (tùy chọn, nếu để trống sẽ dùng tên file)
- `Cấu hình TTS` (phải chọn cấu hình hỗ trợ nhân bản)
- `Nguồn âm thanh` (tải lên âm thanh / ghi âm qua trình duyệt)
- `Văn bản tương ứng âm thanh` (bắt buộc hay không tùy thuộc vào khả năng của provider)
- `Ngôn ngữ văn bản` (ví dụ `zh-CN` / `en-US`)

Sau khi gửi có thể xảy ra hai kết quả:

- Thành công ngay lập tức (hiếm gặp)
- Trả về "Đã gửi tác vụ nhân bản, đang xử lý nền" (phổ biến, bất đồng bộ)

## 4.2 Xem trạng thái tác vụ

Danh sách sẽ hiển thị:

- Nhà cung cấp
- Cấu hình TTS liên kết
- ID giọng nhân bản
- Trạng thái tác vụ
- Lý do thất bại (nếu có)
- Thời gian tạo

Các trạng thái thường gặp:

- Đang xếp hàng / Đang xử lý
- Đã hoàn thành (có thể nghe thử)
- Thất bại (có thể xem lý do và thử lại)

## 4.3 Nghe thử và quản lý

Mỗi bản ghi nhân bản hỗ trợ các thao tác sau:

- `Âm gốc`: Phát mẫu âm thanh người dùng đã gửi
- `Nghe thử nhân bản`: Phát giọng nhân bản do provider trả về (chỉ hiển thị khi trạng thái thành công)
- `Chỉnh sửa`: Sửa tên nhân bản
- `Thử lại nhân bản`: Gửi lại tác vụ thất bại (chỉ hiển thị khi trạng thái thất bại)

---

## 5. Sự khác biệt giữa các Provider và lưu ý

## 5.1 Minimax

Frontend và backend sẽ kiểm tra ràng buộc âm thanh, các quy tắc thường gặp:

- Định dạng âm thanh thường yêu cầu `WAV`
- Thời lượng âm thanh nên/phải không dưới `10 giây`

Trang sẽ hiển thị thông báo ở vùng tải lên/ghi âm và ngăn gửi khi thời lượng không đủ.

## 5.2 CosyVoice

Đặc điểm:

- Hỗ trợ nhân bản
- Trong các tình huống phổ biến yêu cầu điền "văn bản tương ứng âm thanh" (do giao diện khả năng provider trả về)

Có bắt buộc hay không, hãy tham khảo thông báo khả năng provider hiện tại trên trang.

## 5.3 Qianwen (`aliyun_qwen`)

Đặc điểm:

- Hỗ trợ nhân bản
- Hỗ trợ nhiều định dạng âm thanh hơn (ví dụ `WAV/MP3/M4A`, tham khảo thông báo trên trang)
- Sau khi chọn giọng nhân bản loại này, khi chạy sẽ tự động chuyển sang mô hình nhân bản tương ứng (frontend sẽ hiển thị thông báo)

---

## 6. Quản lý hạn ngạch nhân bản (Quản trị viên)

Điểm truy cập:

- `Quản trị viên -> Quản lý người dùng -> Hạn ngạch nhân bản`

Quản trị viên có thể cấu hình hạn ngạch nhân bản cho một người dùng thông thường theo `ID Cấu hình TTS`:

- `-1`: Không giới hạn số lần
- `0`: Cấm tạo
- `Số nguyên dương`: Số lần nhân bản tối đa

Thống kê hạn ngạch thường tính theo "số lần gửi tác vụ nhân bản" (việc thử lại khi thất bại cũng nên được tính vào chiến lược đếm, hãy sử dụng kết hợp với quy tắc kinh doanh hiện tại).

---

## 7. Mô tả API (phía người dùng)

### 7.1 Kiểm tra khả năng

- `GET /user/voice-clone/capabilities?provider=<provider>`

Mục đích:

- Lấy thông tin provider có được bật không
- Có yêu cầu điền transcript không
- Phạm vi độ dài văn bản
- Danh sách ngôn ngữ hỗ trợ

### 7.2 Bản ghi nhân bản và thao tác tác vụ

- `POST /user/voice-clones` (tạo nhân bản, `multipart/form-data`)
- `GET /user/voice-clones` (danh sách)
- `PUT /user/voice-clones/:id` (sửa tên)
- `POST /user/voice-clones/:id/retry` (thử lại khi thất bại)
- `GET /user/voice-clones/:id/preview` (nghe thử giọng nhân bản)

### 7.3 Quản lý âm thanh gốc

- `GET /user/voice-clones/:id/audios`
- `GET /user/voice-clones/audios/:audio_id/file`

---

## 8. Mô tả API (hạn ngạch quản trị viên)

- `GET /admin/users/:id/voice-clone-quotas`
- `PUT /admin/users/:id/voice-clone-quotas`

---

## 9. Các vấn đề thường gặp và cách xử lý

### 9.1 Trang không thấy cấu hình TTS để chọn

Kiểm tra:

1. Quản trị viên đã bật cấu hình TTS chưa
2. TTS provider có thuộc danh sách hỗ trợ nhân bản không (`minimax/cosyvoice/aliyun_qwen`)
3. Người dùng hiện tại có quyền truy cập cấu hình đó không

### 9.2 Khi gửi báo lỗi "Nhà cung cấp này yêu cầu điền văn bản tương ứng âm thanh"

Provider này yêu cầu transcript bắt buộc, hãy bổ sung văn bản tương ứng âm thanh rồi gửi lại.

### 9.3 Khi gửi báo lỗi hết hạn ngạch

Quản trị viên cần tăng hạn ngạch hoặc đặt thành `-1` cho `ID Cấu hình TTS` tương ứng của người dùng đó trong `Quản lý người dùng -> Hạn ngạch nhân bản`.

### 9.4 Nhân bản thành công nhưng không thể nghe thử

Kiểm tra:

1. Trạng thái tác vụ đã hoàn thành chưa
2. Giao diện xem trước của provider có hoạt động bình thường không
3. Trình duyệt có chặn tự động phát âm thanh không (thử nhấn thủ công nút phát)

---

## 10. Khuyến nghị sử dụng

- Chuẩn bị cấu hình TTS riêng biệt cho từng tình huống (thuận tiện cho kiểm soát hạn ngạch và quy kết chi phí)
- Khi gửi âm thanh, hãy dùng giọng người rõ ràng, môi trường ít tiếng ồn
- Transcript nên khớp với nội dung âm thanh, giúp cải thiện chất lượng và độ ổn định của nhân bản
