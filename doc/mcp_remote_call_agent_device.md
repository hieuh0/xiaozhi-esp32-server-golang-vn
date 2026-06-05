# Hướng Dẫn Gọi Từ Xa MCP Theo Chiều Thiết Bị / Tác Nhân

Tài liệu này mô tả **khả năng gỡ lỗi gọi từ xa MCP** trong bảng điều khiển quản lý, bao gồm:

- Tạo điểm kết nối (Endpoint) MCP theo chiều tác nhân
- Lấy danh sách công cụ và gọi từ xa theo chiều tác nhân
- Lấy danh sách công cụ và gọi từ xa theo chiều thiết bị
- Sự khác biệt về quyền hạn giữa quản trị viên và người dùng thông thường

Tài liệu liên quan:

- [Mô tả kiến trúc MCP](./mcp.md)
- [Mô tả chức năng thị trường MCP](./mcp_market.md)
- [Hướng dẫn sử dụng bảng quản trị](./manager_console_guide.md)

---

## 1. Mục Đích Tính Năng

Tính năng này chủ yếu dùng cho "gỡ lỗi và xác minh":

- Nhanh chóng xem tác nhân/thiết bị hiện tại đang phơi ra những công cụ MCP nào
- Trực tiếp xây dựng tham số và gọi công cụ trong bảng điều khiển
- Lấy MCP endpoint theo chiều tác nhân để các MCP client bên ngoài kết nối kiểm thử

Phù hợp với các trường hợp:

- Xác minh xem một dịch vụ MCP từ xa đã có hiệu lực chưa
- Kiểm tra schema công cụ / ví dụ tham số
- Phân tích sự khác biệt hành vi MCP giữa tác nhân và thiết bị

---

## 2. Sự Khác Biệt Giữa Hai Chiều Gọi

## 2.1 Chiều Tác Nhân (Agent)

Đặc điểm:

- Hướng đến góc nhìn "cấu hình tác nhân"
- Hỗ trợ lấy MCP endpoint của tác nhân (kèm token)
- Hỗ trợ lấy danh sách công cụ, trực tiếp gọi công cụ
- Bị ảnh hưởng bởi cấu hình tác nhân (ví dụ `mcp_service_names`)

Dùng phổ biến để:

- Xác minh tập hợp công cụ MCP khả dụng sau khi tác nhân lọc
- Sao chép endpoint cho client gỡ lỗi bên ngoài sử dụng

## 2.2 Chiều Thiết Bị (Device)

Đặc điểm:

- Hướng đến góc nhìn "kết nối thiết bị cụ thể"
- Trực tiếp yêu cầu chi tiết công cụ / gọi công cụ qua ngữ cảnh kết nối hiện tại của thiết bị
- Thường phụ thuộc vào việc thiết bị đang trực tuyến và bộ điều khiển WebSocket khả dụng

Dùng phổ biến để:

- Phân tích "cùng một tác nhân nhưng biểu hiện công cụ khác nhau trên các thiết bị khác nhau"
- Xác minh khả năng MCP phía phiên làm việc trực tuyến hiện tại của thiết bị

---

## 3. Lối Vào Trang (Quản Trị Viên / Người Dùng Thông Thường)

### 3.1 Quản Trị Viên

- `Quản trị viên -> Quản lý tác nhân` (endpoint / tools / call theo chiều tác nhân)
- `Quản trị viên -> Quản lý thiết bị` (tools / call theo chiều thiết bị)

### 3.2 Người Dùng Thông Thường

- `Tác nhân của tôi` (tools / call theo chiều tác nhân)
- `Thiết bị của tôi` / `Thiết bị tác nhân` (tools / call theo chiều thiết bị)
- `Chỉnh sửa tác nhân` (cấu hình `mcp_service_names`, ảnh hưởng đến phạm vi dịch vụ hiển thị theo chiều tác nhân)

---

## 4. Chiều Tác Nhân: Quy Trình Gỡ Lỗi Hoàn Chỉnh

## 4.1 Cấu Hình Dịch Vụ MCP Khả Dụng Cho Tác Nhân (Tùy Chọn Nhưng Khuyến Nghị)

Trong trang chỉnh sửa tác nhân có thể đặt `mcp_service_names` (danh sách tên dịch vụ, phân cách bằng dấu phẩy):

- Để trống: sử dụng toàn bộ dịch vụ MCP toàn cục đã được bật
- Điền vào: chỉ sử dụng tên dịch vụ được chỉ định (phải là dịch vụ đã tồn tại và đang được bật trong hệ thống)

Hệ thống sẽ thực hiện các xử lý sau với trường này:

- Loại bỏ trùng lặp
- Xóa khoảng trắng
- Kiểm tra tính hợp lệ (tên dịch vụ phải tồn tại trong tập hợp dịch vụ toàn cục đang được bật)

## 4.2 Lấy MCP Endpoint Của Tác Nhân

Bảng điều khiển có thể lấy URL điểm kết nối MCP dành riêng cho tác nhân, định dạng tương tự:

```text
ws(s)://<host>/mcp?token=<jwt>
```

Giải thích:

- endpoint suy ra tên miền và giao thức từ `external.websocket.url` trong cấu hình OTA mặc định
- token chứa ngữ cảnh người dùng và tác nhân hiện tại (dùng để kiểm tra quyền / ràng buộc)
- Phù hợp để các MCP client bên ngoài gỡ lỗi tạm thời, không khuyến nghị chia sẻ công khai

## 4.3 Lấy Danh Sách Công Cụ

Bảng điều khiển sẽ yêu cầu chi tiết công cụ MCP theo chiều tác nhân, nội dung trả về thường bao gồm:

- `name`
- Mô tả công cụ
- Schema tham số
- Ví dụ tham số (nếu phía thiết bị / máy chủ cung cấp)

Nếu không thể lấy được (ví dụ bộ điều khiển chưa khởi tạo hoặc client tạm thời không tiếp cận được), backend sẽ trả về danh sách rỗng thay vì báo lỗi, để trang tiếp tục hoạt động.

## 4.4 Gọi Công Cụ Trực Tiếp

Trong bảng điều khiển điền vào:

- `tool_name`
- `arguments` (JSON)

Sau khi gọi có thể xem toàn bộ phần thân trả về (định dạng JSON) trong ô kết quả.

---

## 5. Chiều Thiết Bị: Quy Trình Gỡ Lỗi Hoàn Chỉnh

## 5.1 Lấy Danh Sách Công Cụ Của Thiết Bị

Sau khi chọn thiết bị, bảng điều khiển sẽ dùng định danh thiết bị (nội bộ sẽ ánh xạ sang tên thiết bị) để yêu cầu bộ điều khiển WebSocket cung cấp chi tiết công cụ MCP.

Các trường hợp thất bại phổ biến:

- Thiết bị không trực tuyến
- Thiết bị không thuộc người dùng hiện tại (góc nhìn người dùng)
- Bộ điều khiển WebSocket tạm thời không khả dụng

Trong những trường hợp này, giao diện thường trả về danh sách công cụ rỗng hoặc lỗi quyền hạn.

## 5.2 Gọi Công Cụ MCP Của Thiết Bị

Tương tự chiều tác nhân, điền vào:

- `tool_name`
- `arguments` (JSON)

Điểm khác biệt là phần thân gọi dùng ngữ cảnh `device_id` (backend thực tế truyền tên thiết bị), do đó gần hơn với môi trường thực thi thực sự của "phiên làm việc thiết bị hiện tại".

---

## 6. Sự Khác Biệt Quyền Hạn và Giao Diện (Quản Trị Viên vs Người Dùng Thông Thường)

### 6.1 Giao Diện Người Dùng Thông Thường

Chiều tác nhân:

- `GET /user/agents/:id/mcp-endpoint`
- `GET /user/agents/:id/mcp-tools`
- `POST /user/agents/:id/mcp-call`

Chiều thiết bị:

- `GET /user/devices/:id/mcp-tools`
- `POST /user/devices/:id/mcp-call`

Hỗ trợ lọc dịch vụ tác nhân:

- `GET /user/agents/:id/mcp-services/options`

Người dùng thông thường chỉ có thể thao tác với tác nhân / thiết bị thuộc về mình.

### 6.2 Giao Diện Quản Trị Viên

Chiều tác nhân:

- `GET /admin/agents/:id/mcp-endpoint`
- `GET /admin/agents/:id/mcp-tools`
- `POST /admin/agents/:id/mcp-call`

Chiều thiết bị:

- `GET /admin/devices/:id/mcp-tools`
- `POST /admin/devices/:id/mcp-call`

Quản trị viên có thể gỡ lỗi bất kỳ tác nhân / thiết bị nào xuyên người dùng (với điều kiện bản ghi tồn tại và liên kết kết nối bình thường).

---

## 7. Logic Tạo Endpoint (Chiều Tác Nhân)

Việc tạo endpoint tác nhân phụ thuộc vào:

1. Cấu hình OTA mặc định (`type=ota` và `is_default=true`)
2. `external.websocket.url` trong cấu hình OTA
3. Token ổn định được tạo dựa trên ID người dùng hiện tại + ID tác nhân

Kết quả tạo ra sẽ dùng:

- Cùng giao thức (`ws` / `wss`)
- Cùng host (tên miền/IP + cổng)
- Đường dẫn cố định `/mcp`

Do đó nếu không thể tạo endpoint, hãy ưu tiên kiểm tra cấu hình WebSocket mạng ngoài của OTA.

---

## 8. Câu Hỏi Thường Gặp và Cách Phân Tích

### 8.1 Danh Sách Công Cụ Trống

Nguyên nhân có thể:

- Thiết bị không trực tuyến (chiều thiết bị)
- Bộ điều khiển WebSocket chưa khởi tạo
- Client không trả về chi tiết công cụ
- Chiều tác nhân bị lọc bởi `mcp_service_names` dẫn đến không còn dịch vụ khả dụng

Trình tự phân tích đề nghị:

1. Xác nhận trạng thái trực tuyến của thiết bị
2. Kiểm tra xem dịch vụ MCP toàn cục có được bật không
3. Kiểm tra cấu hình `mcp_service_names` của tác nhân
4. Thử lại lấy công cụ trên bảng điều khiển

### 8.2 Báo Lỗi JSON Tham Số Khi Gọi

Khu vực tham số trong bảng điều khiển yêu cầu đối tượng JSON hợp lệ, ví dụ:

```json
{
  "query": "hello"
}
```

Lỗi thường gặp:

- Dấu nháy đơn
- Dấu phẩy cuối dòng
- Cấp cao nhất không phải là đối tượng

### 8.3 Lấy Endpoint Tác Nhân Thất Bại

Thường do thiếu cấu hình OTA mặc định hoặc `external.websocket.url` chưa được cấu hình.

### 8.4 Đã Nhập Dịch Vụ MCP Nhưng Không Thấy Khi Gọi Tác Nhân

Kiểm tra:

1. Dịch vụ nhập vào có được bật không
2. Tổng công tắc cấu hình MCP toàn cục và trạng thái bật của dịch vụ
3. Tác nhân có loại trừ dịch vụ đó qua `mcp_service_names` không

---

## 9. Thực Hành Tốt Nhất

- Trước tiên xác minh khả năng dùng công cụ theo "chiều thiết bị" ở phía quản trị viên, sau đó xác minh kết quả lọc công cụ theo "chiều tác nhân"
- Đối với tác nhân trong môi trường sản xuất, nên cấu hình tường minh `mcp_service_names` để tránh công cụ không liên quan bị phơi ra cho mô hình
- Coi endpoint như một lối vào gỡ lỗi nhạy cảm, tránh truyền URL có kèm token qua các kênh công khai
