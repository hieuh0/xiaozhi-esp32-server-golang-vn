# Phương án tích hợp asr_server (theo cùng hình thức với manager/backend)

## Mục tiêu

- **asr_server giữ nguyên dạng kho lưu trữ độc lập**: có `go.mod`, `main.go` riêng, có thể clone, build, chạy độc lập.
- **Chương trình chính có thể khởi tạo**: giống như `manager/backend`, tiến trình chính dùng `replace` để tham chiếu thư mục con, và khi cần thiết sẽ khởi động dịch vụ HTTP của asr_server (cổng riêng) ngay trong tiến trình, không cần khởi chạy tiến trình riêng biệt.

## Cách tích hợp: Khuyến nghị dùng Git Submodule

Kho chính có thể lấy thư mục `asr_server/` theo hai cách:

| Cách | Mô tả |
|------|------|
| **Git Submodule (khuyến nghị)** | asr_server giữ nguyên kho Git độc lập; kho chính dùng `git submodule add` để tham chiếu, thu được thư mục "trỏ tới một commit nhất định của asr_server", kho chính chỉ lưu đường dẫn submodule và số commit. |
| Sao chép/di chuyển code | Đặt trực tiếp code của asr_server vào thư mục kho chính, asr_server và kho chính dùng chung lịch sử Git (hoặc trở thành một phần của kho chính). |

Các bước dưới đây trình bày theo cách **Submodule**; logic `replace` phía kho chính và khởi động nhúng giống hệt với "cách sao chép".

## Tham khảo: Hình thức tích hợp của manager/backend

| Mục | Cách làm của manager/backend |
|--------|----------------------|
| Thư mục | `manager/backend/` trong kho chính |
| Tên module | `xiaozhi/manager/backend` (go.mod trong backend) |
| Tham chiếu từ kho chính | `replace xiaozhi/manager/backend => ./manager/backend` |
| Chạy độc lập | `manager/backend/main.go`: LoadWithPath → database.Init → router.Setup → r.Run() |
| Nhúng vào chương trình chính | `cmd/server/manager_http.go`: cùng bộ config/database/router, tự khởi động `http.Server` trên cổng khác |

## Thiết kế tích hợp asr_server (căn chỉnh theo hình thức trên)

### 1. Thư mục và module (theo cách Submodule)

- **asr_server phải có kho Git độc lập trước** (nếu hiện đang nằm trong monorepo, có thể tách thành repo độc lập trước, hoặc dùng URL của kho asr_server hiện có).
- **Thêm submodule vào kho chính** (thực thi tại thư mục gốc của kho chính, và thư mục `asr_server` chưa tồn tại):
  ```bash
  cd xiaozhi-esp32-server-golang
  git submodule add <URL kho asr_server> asr_server
  ```
  Sau khi hoàn tất, kho chính sẽ có thêm:
  - Thư mục `asr_server/` (nội dung là commit hiện tại được checkout của kho asr_server)
  - File `.gitmodules`, và bản ghi submodule có thể xem bằng `git submodule status`
- **Đường dẫn thư mục**: trong kho chính là `xiaozhi-esp32-server-golang/asr_server/`, giống hệt "cách sao chép", `replace` trong go code và go.mod của kho chính đều trỏ vào `./asr_server`.
- **Tên module**: giữ nguyên tên module hiện có của asr_server là **`voice_server`** (để khi dùng như kho độc lập có thể `go build` trực tiếp mà không cần sửa import).
- **go.mod của kho chính**: thêm một dòng:
  - `replace voice_server => ./asr_server`
- **go.mod của asr_server**: giữ `module voice_server`, không tham chiếu kho chính; khi là kho độc lập thì không có replace, sau khi tích hợp vào kho chính chỉ cần replace phía kho chính là đủ.

**Khi clone kho chính, lấy submodule** (chọn một trong hai):

```bash
# Clone đồng thời lấy submodule một lần
git clone --recurse-submodules <URL kho chính>

# Hoặc clone trước rồi khởi tạo submodule sau
git clone <URL kho chính>
cd xiaozhi-esp32-server-golang
git submodule update --init --recursive
```

**CI / Build tự động**: Nếu kho chính cần build code phụ thuộc asr_server, cần thực thi `git submodule update --init --recursive` trước khi build (hoặc dùng `--recurse-submodules` khi clone).

### 2. Chạy độc lập (asr_server vẫn là "kho độc lập")

- Khi clone/mở thư mục `asr_server` riêng lẻ:
  - `go build -o asr_server .`
  - `./asr_server` dùng `config.json` (hoặc chỉ định đường dẫn bằng `-config`), hành vi giống như hiện tại.
- Không phụ thuộc kho chính; `replace` trong kho chính chỉ ảnh hưởng đến việc build kho chính.

### 3. Khởi tạo từ chương trình chính (nhúng asr_server)

- **Điểm vào**: thêm `cmd/server/asr_server_http.go` vào kho chính (cùng cấp với `manager_http.go`).
- **Logic** (giống với manager_http):
  1. Tiến trình chính quyết định lúc khởi động có gọi hay không dựa trên cấu hình (ví dụ: `-asr-enable` + `-asr-config`).
  2. Sử dụng các package của asr_server:
     - `voice_server/config`: `InitConfig(configPath)`, rồi `GetConfig()` để lấy `*Config`.
     - `voice_server/internal/bootstrap`: `InitApp(cfg)` để lấy `*AppDependencies`.
     - `voice_server/internal/router`: `NewRouter(deps)` để lấy `*gin.Engine`.
  3. Dùng `deps.RateLimiter.Middleware(r)` làm Handler, khởi động `http.Server` trên **cổng riêng** (ví dụ: 8080), chạy `ListenAndServe` trong goroutine.
  4. Khi thoát cung cấp `StopAsrServerHTTP()`, thực hiện `Shutdown` cho `http.Server` và giải phóng tài nguyên cần thiết (ví dụ: các component cần Close trong bootstrap).
- **Cấu hình**: asr_server vẫn dùng `config.json` của chính mình; khi nhúng, đường dẫn file cấu hình do tham số của tiến trình chính hoặc cấu hình kho chính chỉ định (ví dụ: `asr_server/config.json` hoặc `config/asr_server.json`).

### 4. Danh sách thay đổi phía kho chính (theo cách Submodule)

| Vị trí | Thay đổi |
|------|------|
| Gốc kho chính | Thực thi `git submodule add <URL kho asr_server> asr_server`, thu được thư mục `asr_server/` và `.gitmodules` (asr_server cần có kho Git độc lập trước) |
| `xiaozhi-esp32-server-golang/go.mod` | Thêm `replace voice_server => ./asr_server`; nếu code kho chính cần import voice_server, thêm `voice_server` vào `require` (hoặc để `go mod tidy` tự bổ sung) |
| `xiaozhi-esp32-server-golang/cmd/server/main.go` | Phân tích `-asr-enable`, `-asr-config`; nếu enable, gọi `StartAsrServerHTTP(configPath)` trước `Run()`; gọi `StopAsrServerHTTP()` sau `<-quit` |
| Thêm mới `xiaozhi-esp32-server-golang/cmd/server/asr_server_http.go` | Cài đặt `StartAsrServerHTTP(configPath string)`, `StopAsrServerHTTP()`, bên trong dùng `voice_server/config`, `voice_server/internal/bootstrap`, `voice_server/internal/router`, nhất quán với mô hình manager_http |

### 5. Những điểm cần phơi bày phía asr_server

- **config**: đã có `InitConfig(path)`, `GetConfig()`, tiến trình chính có thể dùng trực tiếp.
- **bootstrap**: đã có `InitApp(cfg *config.Config)`, trả về `*AppDependencies`, tiến trình chính có thể dùng trực tiếp.
- **router**: đã có `NewRouter(deps) *gin.Engine`; tiến trình chính dùng `deps.RateLimiter.Middleware(r)` làm Handler là được.
- **Thoát graceful**: nếu trong bootstrap có tài nguyên cần `Close()` (như VAD pool, global recognizer, v.v.), cần cung cấp hàm `Shutdown(deps *AppDependencies)` thống nhất hoặc tương tự trong asr_server, để `StopAsrServerHTTP()` gọi; nếu hiện chưa có, có thể chỉ làm `Server.Shutdown` trước, bổ sung sau.

### 6. Phụ thuộc và build

- Các phụ thuộc của asr_server (sherpa-onnx, qdrant, ten-vad, v.v.) giữ trong **asr_server/go.mod**; kho chính **không** nâng các phụ thuộc của asr_server lên require trong go.mod chính, chỉ tham chiếu submodule thông qua `require voice_server` (hoặc tương đương), để `go mod tidy` tự đồng bộ phụ thuộc trong kho chính.
- Nếu khi build kho chính có báo thiếu phụ thuộc, thêm trực tiếp các phụ thuộc trực tiếp mà asr_server dùng vào `require` trong go.mod của kho chính.
- CGO, các lib native (như so/dll của ten_vad, sherpa-onnx) vẫn đặt theo cách hiện có của asr_server trong thư mục asr_server hoặc `lib/` thống nhất của kho chính, ghi rõ trong script build/tài liệu là được.

### 7. Điểm khác biệt so với manager/backend

- Tên module manager/backend là `xiaozhi/manager/backend`, asr_server giữ `voice_server`, như vậy khi asr_server là kho độc lập không cần sửa import.
- Kho chính dùng `replace voice_server => ./asr_server` là đủ, không cần sửa đường dẫn package bên trong asr_server.
- Cách "khởi tạo" từ chương trình chính là như nhau: không gọi `main()` của asr_server, chỉ tái sử dụng config + bootstrap + router, khởi động dịch vụ HTTP với cổng riêng ngay trong tiến trình chính.

### 8. Tóm tắt (theo cách Submodule)

- **Kho độc lập**: asr_server là kho Git độc lập, có `go.mod` riêng (`module voice_server`) và `main.go`, có thể clone, build, chạy độc lập.
- **Tích hợp vào kho chính**: kho chính dùng **Git submodule** để tham chiếu asr_server, thu được thư mục `asr_server/`; kho chính dùng `replace voice_server => ./asr_server`; sau khi clone kho chính cần thực thi `git submodule update --init` (hoặc `git clone --recurse-submodules`).
- **Khởi tạo từ chương trình chính**: kho chính thêm `asr_server_http.go`, theo cấu hình khởi động dịch vụ HTTP của asr_server (cổng riêng) ngay trong tiến trình, logic căn chỉnh với `manager_http.go`.

**Hướng dẫn build**: asr_server phụ thuộc sherpa-onnx (CGO), kho chính dùng **build tag** để việc nhúng trở thành tùy chọn:
- **Build mặc định** (không bật nhúng asr_server): `go build -o xiaozhi_server ./cmd/server`, lúc này `-asr-enable` sẽ in thông báo "chưa được biên dịch vào binary này".
- **Bật nhúng asr_server**: `go build -tags asr_server -o xiaozhi_server ./cmd/server`, yêu cầu máy có CGO và môi trường cần thiết cho sherpa-onnx.

Nếu xác nhận triển khai theo phương án này, có thể làm rõ thêm: danh sách trách nhiệm của `Shutdown(deps)` trong asr_server, cổng mặc định và đường dẫn cấu hình, cũng như tên tham số và giá trị mặc định trong `main.go` của kho chính.
