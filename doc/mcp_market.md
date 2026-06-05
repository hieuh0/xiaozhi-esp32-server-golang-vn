# Mô tả chức năng MCP Market

Tài liệu này mô tả chức năng **MCP Market** trong trang quản trị: cách kết nối với các chợ MCP bên thứ ba, khám phá dịch vụ tổng hợp, nhập cấu hình dịch vụ và đưa vào danh sách MCP service toàn cục của hệ thống.

Tài liệu liên quan:

- [Mô tả kiến trúc MCP](./mcp.md)
- [Hướng dẫn sử dụng trang quản trị](./manager_console_guide.md)

---

## 1. Định vị chức năng

MCP Market dùng để giải quyết vấn đề "hiệu quả kết nối dịch vụ MCP từ xa thấp", hỗ trợ:

- Cấu hình nhiều kết nối MCP Market (ví dụ: ModelScope, v.v.)
- Tổng hợp danh mục dịch vụ từ nhiều chợ
- Xem chi tiết dịch vụ (endpoint, giao thức truyền tải, v.v.)
- Nhập cấu hình dịch vụ vào hệ thống với một cú nhấp
- Bật/tắt/chỉnh sửa/xóa các dịch vụ đã nhập

Các dịch vụ sau khi nhập sẽ tham gia vào quá trình hợp nhất cấu hình MCP service toàn cục của hệ thống (hoạt động cùng với các MCP service được cấu hình thủ công).

---

## 2. Phân quyền vai trò và đường dẫn truy cập

Phân quyền vai trò:

- Chỉ quản trị viên mới có quyền thao tác

Đường dẫn vào trang quản trị:

- `Quản trị viên -> MCP Market`

Trang bao gồm hai tab:

- `Khám phá chợ`
- `Dịch vụ đã nhập`

---

## 3. Các khái niệm cốt lõi

### 3.1 MCP Market (Market)

Đại diện cho một "nguồn danh mục MCP Market có thể truy cập được", bao gồm:

- Tên chợ
- Định danh nhà cung cấp (provider)
- URL danh mục (catalog_url)
- Template URL chi tiết (detail_url_template, tùy chọn)
- Token xác thực (tùy chọn)
- Trạng thái bật/tắt

### 3.2 Danh sách dịch vụ tổng hợp

Hệ thống sẽ lấy danh mục dịch vụ từ các chợ đang được bật và hiển thị tổng hợp, hỗ trợ:

- Tìm kiếm theo tên dịch vụ / mô tả / Service ID
- Xem chi tiết
- Nhập cấu hình

Khi một số chợ lấy dữ liệu thất bại, trang sẽ hiển thị danh sách cảnh báo "một số chợ lấy dữ liệu thất bại", không ảnh hưởng đến việc hiển thị kết quả từ các chợ khác.

### 3.3 Dịch vụ đã nhập

Các dịch vụ sau khi nhập sẽ tạo thành các mục cấu hình độc lập trong hệ thống, có thể trực tiếp tham gia kết nối dịch vụ MCP lúc chạy. Hỗ trợ cấu hình:

- Tên
- Loại truyền tải (`sse` / `streamablehttp`)
- URL
- Headers (JSON)
- Định danh chợ nguồn và provider (thông tin metadata tùy chọn)
- Trạng thái bật/tắt

---

## 4. Quy trình thao tác thường dùng (Quản trị viên)

## 4.1 Thêm kết nối MCP Market mới

Trong tab `Khám phá chợ`, nhấp `Thêm kết nối` và điền vào:

- `Nhà cung cấp`: Ưu tiên chọn preset provider tích hợp sẵn (sẽ tự động điền template URL danh mục)
- `Tên`
- `URL danh mục`
- `Template URL chi tiết` (tùy chọn)
- `Bật`
- `Token` (nếu chợ yêu cầu)

Nên thực hiện kiểm tra kết nối (xem bên dưới) trước khi lưu để sử dụng.

## 4.2 Kiểm tra kết nối chợ

Trong menu thao tác danh sách chợ, nhấp `Kiểm tra`:

- Thành công sẽ trả về "số lượng dịch vụ có thể khám phá"
- Thất bại sẽ thông báo lỗi kết nối danh mục / xác thực

Phù hợp để khắc phục sự cố:

- Token không hợp lệ
- URL danh mục sai
- Chợ tạm thời không khả dụng

## 4.3 Duyệt và tìm kiếm dịch vụ tổng hợp

Trong khu vực `Danh sách dịch vụ tổng hợp` có thể:

- Nhập từ khóa để tìm kiếm dịch vụ
- Xem kết quả tổng hợp theo trang
- Nhấp `Chi tiết` để xem thông tin endpoint dịch vụ

Trang chi tiết dịch vụ thường bao gồm:

- Tên dịch vụ
- Chợ nguồn
- Service ID
- Mô tả
- Danh sách endpoint (giao thức truyền tải + URL)

## 4.4 Nhập cấu hình dịch vụ với một cú nhấp (được khuyến nghị)

Trong hộp thoại chi tiết dịch vụ, nhấp `Nhập cấu hình dịch vụ và cập nhật nóng`:

- Hệ thống sẽ tạo một hoặc nhiều mục cấu hình dịch vụ nhập dựa trên chi tiết dịch vụ
- Sau khi nhập thành công sẽ làm mới danh sách "Dịch vụ đã nhập"
- Trang sẽ chuyển sang tab `Dịch vụ đã nhập`

"Cập nhật nóng" có nghĩa là sau khi nhập cấu hình hoàn tất, dịch vụ có thể tham gia ngay vào tập hợp dịch vụ lúc chạy (không cần khởi động lại backend).

## 4.5 Thêm/chỉnh sửa dịch vụ nhập thủ công

Trong tab `Dịch vụ đã nhập`, có thể nhấp `Thêm dịch vụ` để nhập thủ công, cũng có thể chỉnh sửa các mục đã nhập.

Giải thích các trường chính:

- `Truyền tải`: Hiện hỗ trợ `SSE`, `StreamableHTTP`
- `URL`: Điểm vào dịch vụ MCP từ xa
- `Headers(JSON)`: Dùng để mang thông tin xác thực, ví dụ `Authorization`
- `Bật`: Sau khi tắt sẽ không tham gia vào tập hợp dịch vụ khả dụng lúc chạy

`Headers(JSON)` phải là đối tượng JSON, ví dụ:

```json
{
  "Authorization": "Bearer <token>"
}
```

---

## 5. Mối quan hệ với cấu hình MCP toàn cục

MCP Market không phải là sự thay thế cho trang `Cấu hình MCP`, mà là nguồn bổ sung.

Tập hợp MCP service toàn cục khả dụng lúc chạy được hợp nhất từ hai phần:

- Các dịch vụ toàn cục được quản trị viên duy trì thủ công trên trang `Cấu hình MCP`
- Các dịch vụ đã nhập từ MCP Market và đang được bật

Do đó cách thực hành được khuyến nghị là:

1. Sử dụng MCP Market để khám phá và nhập nhanh
2. Bật và chọn dịch vụ theo nhu cầu trong `Cấu hình MCP` / agent

---

## 6. API (Giao diện backend)

Dưới đây là các giao diện liên quan đến phía quản trị (yêu cầu quyền quản trị viên):

### 6.1 Quản lý kết nối chợ

- `GET /admin/mcp-markets`
- `POST /admin/mcp-markets`
- `PUT /admin/mcp-markets/:id`
- `DELETE /admin/mcp-markets/:id`
- `POST /admin/mcp-markets/:id/test`

### 6.2 Khám phá chợ và chi tiết

- `GET /admin/mcp-market/providers`
- `GET /admin/mcp-market/services`
- `GET /admin/mcp-market/services/:market_id/*service_id`
- `POST /admin/mcp-market/import`

### 6.3 Quản lý dịch vụ đã nhập

- `GET /admin/mcp-market/imported-services`
- `POST /admin/mcp-market/imported-services`
- `PUT /admin/mcp-market/imported-services/:id`
- `DELETE /admin/mcp-market/imported-services/:id`

---

## 7. Câu hỏi thường gặp và khắc phục sự cố

### 7.1 Danh sách tổng hợp trống

Thứ tự khắc phục:

1. Kiểm tra kết nối chợ có được bật hay không
2. Thực hiện "Kiểm tra" với chợ đó
3. Kiểm tra Token có hợp lệ không
4. Kiểm tra URL danh mục / template URL chi tiết có đúng không

### 7.2 Nhập thành công nhưng không thấy dịch vụ lúc chạy

Nguyên nhân thường gặp:

- Dịch vụ nhập bị tắt
- Công tắc tổng thể MCP toàn cục bị tắt
- Agent đã cấu hình `mcp_service_names` nhưng không bao gồm tên dịch vụ đó

### 7.3 Để trống Token khi chỉnh sửa chợ thì sao?

Để trống Token trong hộp thoại chỉnh sửa thường có nghĩa là "không thay đổi Token hiện tại" (giao diện sẽ hiển thị gợi ý trạng thái ẩn danh hiện tại).

---

## 8. Khuyến nghị sử dụng

- Ưu tiên sử dụng preset provider tích hợp sẵn để giảm thiểu vấn đề do sự khác biệt về trường giao diện danh mục
- Đặt tên thống nhất cho các dịch vụ cần sử dụng ổn định lâu dài sau khi nhập, để agent có thể chọn theo tên
- Đối với các dịch vụ từ xa trong môi trường production, nên sử dụng `Headers(JSON)` để cấu hình xác thực và thực hiện luân chuyển token định kỳ
