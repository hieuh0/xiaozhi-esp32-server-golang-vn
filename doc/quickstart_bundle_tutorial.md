# Hướng dẫn triển khai gói khởi động nhanh

## Tải xuống

Truy cập [trang Release](https://github.com/hackers365/xiaozhi-esp32-server-golang/releases) để tải xuống phiên bản phù hợp với nền tảng của bạn:

| Nền tảng | Tên tệp |
|-----|-------|
| Windows | `xiaozhi-server-windows-xxx.zip` |
| Linux | `xiaozhi-server-linux-xxx.tar.gz` |
| macOS | `xiaozhi-server-macos-xxx.tar.gz` |

---

## Giải nén và cấu trúc thư mục

Sau khi giải nén, cấu trúc thư mục như sau:

```
xiaozhi-aio/
├── xiaozhi_server          # Chương trình chính
├── config/                 # Thư mục chứa tệp cấu hình
├── models/                 # Thư mục chứa tệp mô hình (nếu sử dụng ASR/TTS cục bộ)
└── data/                   # Thư mục dữ liệu
```

---

## Khởi động dịch vụ

### Windows
Nhấp đúp vào `start.bat`

### Linux
```bash
# Phụ thuộc runtime của ten_vad
sudo apt install -y libc++1 libc++abi1

chmod +x xiaozhi_server
LD_LIBRARY_PATH="$PWD/ten-vad/lib/Linux/x64:${LD_LIBRARY_PATH:-}" ./xiaozhi_server
```

### macOS
```bash
chmod +x xiaozhi_server
./build/macos/fix_rpath.sh ./xiaozhi_server
./xiaozhi_server
```

Nếu cấu trúc thư mục được giữ nguyên như sau:

```text
./xiaozhi_server
./ten-vad/lib/macOS/ten_vad.framework
```

thì gói macOS sau khi đã chạy `fix_rpath.sh` sẽ không cần thiết lập thủ công `DYLD_FRAMEWORK_PATH`.

Nếu bạn đang gỡ lỗi từ thư mục tạm thời của IDE, hoặc đã di chuyển tệp nhị phân thủ công làm phá vỡ cấu trúc thư mục tương đối, bạn có thể dùng cách dự phòng:

```bash
DYLD_FRAMEWORK_PATH="$PWD/ten-vad/lib/macOS" ./xiaozhi_server
```

Nếu bạn tự đóng gói phân phối macOS từ kho mã nguồn, cần thực thi thêm một lần trước khi phát hành:

```bash
./build/macos/fix_rpath.sh ./xiaozhi_server
```

Bước này sẽ sửa `rpath` trong tệp nhị phân từ đường dẫn mã nguồn trên máy phát triển thành `@executable_path/ten-vad/lib/macOS`, để gói phát hành có thể chạy trực tiếp khi cấu trúc thư mục đúng.

---

## Các bước tiếp theo

### 1. Truy cập bảng điều khiển Web

Mở trình duyệt và truy cập: **http://<IP hoặc tên miền máy chủ>:8080**

<!-- Vị trí ảnh chụp màn hình: giao diện đăng nhập -->
> Hình: Giao diện đăng nhập bảng điều khiển Web

### 2. Cấu hình dịch vụ

Lần đầu sử dụng, vui lòng hoàn tất thiết lập theo hướng dẫn cấu hình, xem chi tiết tại:

**[Hướng dẫn sử dụng bảng quản trị →](manager_console_guide.md)**

---

## Dịch vụ nhận dạng giọng nói (Tùy chọn)

Dịch vụ nhận dạng giọng nói đã được tích hợp sẵn trong chương trình

---

## Câu hỏi thường gặp

### Q1: Sau khi khởi động không thể truy cập bảng điều khiển Web?

Kiểm tra cài đặt tường lửa, đảm bảo cổng 8080 có thể truy cập được.

### Q2: Làm thế nào để khởi động lại dịch vụ?

Đóng chương trình rồi chạy lại. Tệp cấu hình được lưu trong thư mục `config/`.

### Q3: Làm thế nào để xem nhật ký?

Nhật ký thời gian thực được xuất ra console, nếu cần lưu lại có thể chuyển hướng:

```bash
./xiaozhi_server > server.log 2>&1
```
