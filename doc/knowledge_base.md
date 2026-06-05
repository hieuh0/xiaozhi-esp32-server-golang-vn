# Hướng dẫn chức năng Cơ sở tri thức

Tài liệu này mô tả chức năng **Cơ sở tri thức (Knowledge Base / RAG)** trong dự án, bao gồm cấu hình provider phía quản trị viên, quản lý cơ sở tri thức và tài liệu phía người dùng thường, kiểm thử truy hồi, cũng như tích hợp truy hồi cơ sở tri thức trong luồng chat của chương trình chính.

Tài liệu liên quan:

- [Hướng dẫn sử dụng trang quản trị](./manager_console_guide.md)
- [Mô tả kiến trúc MCP](./mcp.md)（Công cụ truy hồi cơ sở tri thức `search_knowledge` sẽ được kích hoạt qua chuỗi công cụ nội bộ）

---

## 1. Tổng quan chức năng

Chức năng cơ sở tri thức dùng để cung cấp khả năng "trả lời dựa trên tài liệu" cho các agent, bao gồm ba tầng:

1. Quản trị viên cấu hình provider truy hồi cơ sở tri thức（Dify / RAGFlow / WeKnora）
2. Người dùng thường tạo cơ sở tri thức và tài liệu, đồng bộ không đồng bộ lên provider
3. Agent liên kết cơ sở tri thức, kích hoạt công cụ `search_knowledge` nội bộ để thực hiện truy hồi khi đối thoại

Các provider hiện được hỗ trợ trên trang quản lý frontend:

- `dify`
- `ragflow`
- `weknora`

---

## 2. Phân công vai trò

## 2.1 Quản trị viên

Chịu trách nhiệm:

- Cấu hình provider truy hồi cơ sở tri thức（toàn cục）
- Duy trì tham số kết nối provider và ngưỡng mặc định
- （Tuỳ chọn）Quản lý cơ sở tri thức thay người dùng

Đường dẫn truy cập:

- `Quản trị viên -> Cấu hình truy hồi cơ sở tri thức`

## 2.2 Người dùng thường

Chịu trách nhiệm:

- Tạo/chỉnh sửa/xóa cơ sở tri thức của mình
- Quản lý tài liệu cơ sở tri thức（nhập văn bản / tải file lên）
- Khởi động đồng bộ thủ công và thử lại
- Dùng "Kiểm thử truy hồi" để xác minh hiệu quả truy hồi từ khóa
- Chọn liên kết cơ sở tri thức trong agent

Đường dẫn truy cập:

- `Người dùng -> Cơ sở tri thức của tôi`
- `Người dùng -> Chỉnh sửa agent（liên kết cơ sở tri thức）`

---

## 3. Quản trị viên: Cấu hình truy hồi cơ sở tri thức（Cấu hình Provider）

Trang quản trị hỗ trợ duy trì nhiều cấu hình provider và chỉ định provider mặc định.

Các mục cấu hình thường gặp（có thể khác nhau tuỳ provider）:

- `Base URL`
- `API Key / Token`
- Ngưỡng truy hồi mặc định
- Tham số đặc thù của provider（ví dụ: ngưỡng độ tương đồng RAGFlow, tham số phân đoạn WeKnora, v.v.）

### 3.1 Dify

Các mục cấu hình điển hình:

- `base_url`
- `api_key`
- `score_threshold`
- Các tham số provider khác

### 3.2 RAGFlow

Các mục cấu hình điển hình:

- `base_url`
- `api_key`
- `similarity_threshold`

### 3.3 WeKnora

Các mục cấu hình điển hình:

- `base_url`
- `api_key`
- `score_threshold`
- Tham số phân đoạn（`chunk_size` / `chunk_overlap` / `separators`）
- Tham số polling phân tích（`parse_poll_interval_ms` / `parse_timeout_ms`）

Trang quản trị còn hỗ trợ lấy danh sách mô hình WeKnora（embedding / llm / rerank）để hỗ trợ điền cấu hình.

---

## 4. Người dùng thường: Cơ sở tri thức của tôi（Quản lý KB）

Đường dẫn truy cập:

- `Người dùng -> Cơ sở tri thức của tôi`

Các thao tác được hỗ trợ:

- Thêm/chỉnh sửa cơ sở tri thức
- Đặt trạng thái（`active` / `inactive`）
- Đặt ngưỡng truy hồi（có thể kế thừa toàn cục）
- Quản lý tài liệu
- Thử lại đồng bộ thủ công
- Kiểm thử truy hồi
- Xóa cơ sở tri thức

### 4.1 Các trường cơ sở tri thức（hiển thị với người dùng）

Các cột thường hiển thị:

- ID
- Tên
- Mô tả
- Nhà cung cấp
- Trạng thái
- Trạng thái đồng bộ
- Thời gian đồng bộ gần nhất
- Thao tác

Lưu ý:

- Khi đồng bộ thất bại, thông tin lỗi sẽ hiển thị dưới dạng "tooltip" trong cột "Trạng thái đồng bộ", tránh bảng bị quá rộng theo chiều ngang

### 4.2 Trạng thái đồng bộ（thường gặp）

Cả cơ sở tri thức lẫn tài liệu đều có thể xuất hiện các trạng thái tương tự:

- Chờ đồng bộ
- Đang tải lên / Đã tải lên / Đang phân tích
- Đã đồng bộ
- Thất bại（bao gồm lỗi tải lên, lỗi phân tích, v.v.）

Nếu thất bại có thể nhấn `Thử lại đồng bộ` để đưa lại vào hàng đợi tác vụ không đồng bộ.

---

## 5. Quản lý tài liệu（trong cơ sở tri thức）

Mỗi cơ sở tri thức có thể chứa nhiều tài liệu, hỗ trợ:

- Tài liệu dạng văn bản（chỉnh sửa trực tuyến）
- Tạo tài liệu bằng cách tải file lên（định dạng giới hạn theo provider）

Chức năng trên trang:

- Thêm tài liệu
- Chỉnh sửa tài liệu（tài liệu dạng file thường không hỗ trợ chỉnh sửa trực tuyến）
- Xóa tài liệu
- Thử lại đồng bộ
- Tải file lên

### 5.1 Định dạng file tải lên

Frontend sẽ hiển thị gợi ý `accept` và hướng dẫn tải lên khác nhau tuỳ theo provider của cơ sở tri thức hiện tại:

- Dify：Hỗ trợ các định dạng văn bản/tài liệu thông dụng（ví dụ: txt/md/pdf/html/xlsx/docx/csv, v.v.）
- RAGFlow：Hỗ trợ nhiều loại file hơn（bao gồm hình ảnh, log, file cấu hình, v.v.）
- WeKnora：Hỗ trợ nhiều loại file（bao gồm Office, hình ảnh, email, v.v.）

Các định dạng có thể tải lên cụ thể xin tham khảo gợi ý trên trang.

---

## 6. Kiểm thử truy hồi（phía người dùng）

Trong danh sách cơ sở tri thức, có thể thực hiện `Kiểm thử truy hồi` cho từng cơ sở tri thức, dùng để xác minh trực tiếp hiệu quả truy hồi của provider.

Các mục kiểm thử:

- `query`：Từ khóa hoặc câu hỏi kiểm thử
- `top_k`
- `threshold`（Chỉ có hiệu lực cho lần kiểm thử này, có thể để trống）

Nội dung trả về:

- Số lượng kết quả trúng
- Nguồn trúng（title）
- score
- Đoạn văn bản trúng
- Thời gian phản hồi

### 6.1 Thứ tự ưu tiên ngưỡng（giải thích logic）

Thông thường ngưỡng được lấy theo thứ tự ưu tiên sau:

1. Ngưỡng trong yêu cầu kiểm thử lần này（nếu đã điền）
2. Ngưỡng của cơ sở tri thức
3. Ngưỡng mặc định toàn cục của provider

### 6.2 Giải thích tham số WeKnora（quan trọng）

Kiểm thử truy hồi WeKnora hiện đã sử dụng theo chiều cơ sở tri thức:

- `knowledge_base_ids`（danh sách ID cơ sở tri thức）

Dùng để giới hạn phạm vi truy hồi chính xác vào cơ sở tri thức hiện tại.

---

## 7. Agent liên kết cơ sở tri thức

Trong trang chỉnh sửa agent có thể chọn nhiều cơ sở tri thức cho agent（chọn nhiều）.

Giải thích hành vi:

- Hỗ trợ liên kết nhiều cơ sở tri thức
- Khi đối thoại, mô hình sẽ tự phán đoán có cần kích hoạt truy hồi cơ sở tri thức hay không
- Nếu có thể xác định được cơ sở tri thức cụ thể, lệnh gọi công cụ sẽ truyền `knowledge_base_ids`
- Khi truy hồi thất bại sẽ hạ cấp về đối thoại LLM thông thường（frontend có thông báo gợi ý）

---

## 8. Truy hồi cơ sở tri thức trong luồng đối thoại của chương trình chính

Chương trình chính thực hiện truy hồi cơ sở tri thức thông qua công cụ nội bộ `search_knowledge`.

Các trường cốt lõi trong tham số gọi công cụ:

- `query`
- `top_k`
- `knowledge_base_ids`（Tuỳ chọn, danh sách ID cơ sở tri thức）

Giải thích hành vi:

- Không truyền `knowledge_base_ids`：Truy hồi trong tất cả các cơ sở tri thức khả dụng được liên kết với agent hiện tại
- Truyền `knowledge_base_ids`：Chỉ truy hồi trong các cơ sở tri thức được chỉ định

Điều này cho phép mô hình thu hẹp phạm vi truy hồi khi đã biết câu hỏi thuộc về đâu, nâng cao độ liên quan và giảm truy hồi không liên quan.

### 8.1 Tham số truy hồi chương trình chính WeKnora

Yêu cầu truy hồi chương trình chính WeKnora hiện đã sử dụng:

- `knowledge_base_ids`

Nhất quán với kiểm thử truy hồi trên console.

---

## 9. Danh sách API（phía người dùng）

### 9.1 CRUD cơ sở tri thức

- `GET /user/knowledge-bases`
- `POST /user/knowledge-bases`
- `GET /user/knowledge-bases/:id`
- `PUT /user/knowledge-bases/:id`
- `DELETE /user/knowledge-bases/:id`
- `POST /user/knowledge-bases/:id/sync`

### 9.2 Kiểm thử truy hồi

- `POST /user/knowledge-bases/:id/test-search`

### 9.3 Quản lý tài liệu

- `GET /user/knowledge-bases/:id/documents`
- `POST /user/knowledge-bases/:id/documents`
- `POST /user/knowledge-bases/:id/documents/upload`
- `PUT /user/knowledge-bases/:id/documents/:doc_id`
- `DELETE /user/knowledge-bases/:id/documents/:doc_id`
- `POST /user/knowledge-bases/:id/documents/:doc_id/sync`

### 9.4 Agent liên kết cơ sở tri thức

- `GET /user/agents/:id/knowledge-bases`
- `PUT /user/agents/:id/knowledge-bases`

---

## 10. Danh sách API（phía quản trị viên）

### 10.1 Quản lý cấu hình provider

- `GET /admin/knowledge-search-configs`
- `POST /admin/knowledge-search-configs`
- `PUT /admin/knowledge-search-configs/:id`
- `DELETE /admin/knowledge-search-configs/:id`

### 10.2 Lấy danh sách mô hình WeKnora（hỗ trợ cấu hình）

- `POST /admin/knowledge-search-configs/weknora/models`

### 10.3 Quản trị viên quản lý cơ sở tri thức thay người dùng（theo chiều người dùng）

- `GET /admin/users/:id/knowledge-bases`
- `POST /admin/users/:id/knowledge-bases`
- `PUT /admin/users/:id/knowledge-bases/:kb_id`
- `DELETE /admin/users/:id/knowledge-bases/:kb_id`

---

## 11. Câu hỏi thường gặp và cách xử lý

### 11.1 Cơ sở tri thức tạo xong nhưng mãi không truy hồi được

Ưu tiên kiểm tra:

1. Cơ sở tri thức / tài liệu đã đồng bộ thành công chưa
2. Provider bên ngoài đã hoàn thành xây dựng chỉ mục chưa
3. Ngưỡng truy hồi có quá cao không
4. `query` có quá chung chung hoặc lệch xa nội dung tài liệu không

### 11.2 Tài liệu không thể chỉnh sửa sau khi tải file lên

Tài liệu được tạo bằng cách tải file lên thường được xử lý như "tài liệu dạng file", frontend sẽ hạn chế chỉnh sửa trực tuyến, nên xóa và tải lên lại.

### 11.3 Phạm vi truy hồi WeKnora không đúng

Xác nhận:

- Kiểm thử truy hồi trên console có đang sử dụng cơ sở tri thức hiện tại để kiểm thử không
- Lệnh gọi công cụ của agent có truyền đúng `knowledge_base_ids` không

---

## 12. Khuyến nghị sử dụng

- Chia thành nhiều cơ sở tri thức theo từng lĩnh vực nghiệp vụ（ví dụ: hậu mãi, sản phẩm, hợp đồng）
- Dùng "Kiểm thử truy hồi" để điều chỉnh ngưỡng trước, rồi mới tích hợp vào agent
- Trong mô tả agent, nêu rõ khi nào cần trả lời từ cơ sở tri thức để nâng cao chất lượng kích hoạt
