# Hướng dẫn sử dụng MCP Audio Server từ kho lưu trữ độc lập

## Tổng quan

MCP Audio Server đã được tách ra thành kho lưu trữ độc lập. Khuyến nghị sử dụng trực tiếp dự án độc lập để chạy, gỡ lỗi và phát triển thêm.

Tên dự án độc lập:

- `mcp_audio_server`
- `github.com/hackers365/mcp_audio_server`

Mục tiêu cốt lõi là minh họa:

- Cách trả về `ResourceLink` thông qua công cụ `musicPlayer`
- Cách đọc dữ liệu âm thanh theo phân trang qua `resource/read`
- Cách trả về đoạn âm thanh mã hóa base64 bằng `BlobResourceContents`

Kho lưu trữ độc lập này vừa có thể chạy trực tiếp, vừa phù hợp để làm mẫu tích hợp.

## Cách sử dụng khuyến nghị

Khuyến nghị sử dụng MCP Audio Server từ kho lưu trữ độc lập.

Khuyến nghị lấy kho lưu trữ độc lập trước, sau đó vào thư mục dự án:

```bash
git clone https://github.com/hackers365/mcp_audio_server.git
cd mcp_audio_server
```

## Khả năng mà dịch vụ cung cấp

Hiện tại dịch vụ chỉ cung cấp hai loại khả năng:

1. Công cụ `musicPlayer`
2. Tài nguyên `resource://read_from_http`

### `musicPlayer`

- Chức năng: Tìm kiếm bài hát theo tên người dùng nhập và trả về tài nguyên có thể phát
- Tham số đầu vào: `query`
- Giá trị trả về: `ResourceLink`

Ý nghĩa các trường quan trọng trong `ResourceLink` trả về:

- `URI`: `resource://read_from_http`
- `Name`: Tên bài hát thực tế
- `Description`: URL âm thanh thực tế
- `MIMEType`: `audio/mpeg`

### `resource://read_from_http`

- Chức năng: Đọc dữ liệu âm thanh từ xa theo phân trang
- Cách gọi: Thông qua `resource/read`
- Tham số truyền qua `Arguments`

Định dạng tham số yêu cầu:

```json
{
  "url": "URL âm thanh thực tế",
  "start": 0,
  "end": 102400
}
```

Mô tả tham số:

- `url`: Địa chỉ âm thanh thực, lấy từ `ResourceLink.Description`
- `start`: Vị trí byte bắt đầu
- `end`: Vị trí byte kết thúc, không bao gồm vị trí này

Nội dung trả về là `BlobResourceContents`:

- `MIMEType`: `audio/mpeg`
- `Blob`: Dữ liệu nhị phân âm thanh được mã hóa base64

Khi đọc hết dữ liệu, server sẽ trả về `[DONE]` được mã hóa base64 làm dấu hiệu kết thúc.

## Luồng gọi

Luồng đầy đủ như sau:

1. Client gọi `musicPlayer`
2. Công cụ tìm kiếm bài hát và trả về `ResourceLink`
3. Client gửi `resource/read` tới `resource://read_from_http`
4. Mỗi lần truyền `url`, `start`, `end` qua `Arguments`
5. Server trả về `BlobResourceContents` được mã hóa base64
6. Client giải mã và phát liên tục theo luồng âm thanh cho đến khi nhận được `[DONE]`

## Cách chạy

Kho lưu trữ độc lập hỗ trợ hai phương thức truyền tải:

- Mặc định: `stdio`
- Tùy chọn: HTTP Streamable MCP

### Chế độ stdio

Khởi động trực tiếp:

```bash
git clone https://github.com/hackers365/mcp_audio_server.git
cd mcp_audio_server
go run .
```

### Chế độ HTTP

Chỉ định rõ ràng phương thức truyền tải HTTP:

```bash
cd mcp_audio_server
go run . -t http
```

Hoặc:

```bash
cd mcp_audio_server
go run . --transport http
```

Thông tin lắng nghe ở chế độ HTTP:

- Cổng: `3001`
- Đường dẫn: `/mcp`
- Địa chỉ đầy đủ: `http://localhost:3001/mcp`

## Lưu ý khi sử dụng hiện tại

Kho lưu trữ độc lập có thể xây dựng và chạy trực tiếp. Trước khi sử dụng, nên lưu ý các điểm sau:

- Việc tìm kiếm bài hát và lấy URL thực tế phụ thuộc vào `github.com/scroot/music-sd/pkg/netease` và `github.com/scroot/music-sd/pkg/qq`
- Độ ổn định của kết quả tìm kiếm nhạc và các liên kết có thể phát phụ thuộc vào khả năng của các trang web bên ngoài
- Nếu chuyển dự án độc lập này sang dự án khác, thường cần bổ sung đồng bộ các phụ thuộc và logic tìm kiếm nêu trên

Nếu mục tiêu của bạn là tích hợp nhanh công cụ âm thanh của riêng mình, khuyến nghị ưu tiên tái sử dụng giao thức và luồng dữ liệu thay vì trực tiếp tái sử dụng phần triển khai tìm kiếm bài hát.

## Các phần cần giữ nguyên khi sử dụng làm mẫu tích hợp

Nếu muốn chuyển đổi dự án độc lập này thành MCP Audio Server của riêng bạn, khuyến nghị giữ nguyên các quy ước giao thức sau:

- Công cụ trả về `ResourceLink`
- `resource/read` sử dụng `Arguments` để đọc theo phân trang
- Dữ liệu âm thanh được trả về qua `BlobResourceContents.Blob`
- Nội dung `Blob` giữ nguyên dạng mã hóa base64
- Loại MIME âm thanh phải khớp với dữ liệu thực tế; kho lưu trữ độc lập hiện tại là `audio/mpeg`
- Trả về `[DONE]` khi kết thúc luồng

Như vậy có thể duy trì tính tương thích với logic tiêu thụ âm thanh trong dịch vụ chính hiện tại.

## Tính tương thích với dịch vụ chính hiện tại

Logic tiêu thụ công cụ MCP loại âm thanh trong dịch vụ chính hiện tại đã được xử lý theo cách sau:

- Nhận dạng `ResourceLink`
- Gọi `resource/read` theo phân trang bằng phương thức `Arguments`
- Giải mã `BlobResourceContents.Blob`
- Phân tích định dạng âm thanh theo loại MIME
- Phát liên tục cho đến khi đọc xong

Do đó, hình thái giao thức của dự án độc lập này có thể tiếp tục được sử dụng làm mẫu tham chiếu cho các công cụ MCP loại âm thanh.
