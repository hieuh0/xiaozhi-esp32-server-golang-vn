# Mô tả luồng kết nối WebSocket

## Tổng quan

Tài liệu này mô tả luồng kết nối và giao tiếp WebSocket giữa `internal/domain/config/manager/websocket_client.go` và `websocket.go`.

## Thiết kế kiến trúc

### Định nghĩa vai trò

1. **`internal/domain/config/manager/websocket_client.go`** - WebSocket client của máy chủ chính
   - Đóng vai trò client kết nối đến Manager Backend
   - Có thể gửi yêu cầu và nhận phản hồi
   - Hỗ trợ giao tiếp hai chiều

2. **`websocket.go`** - WebSocket server của Manager Backend
   - Đóng vai trò server nhận kết nối WebSocket từ máy chủ chính
   - Xử lý các yêu cầu do máy chủ chính gửi đến
   - **Chỉ giữ lại kết nối hợp lệ cuối cùng** (kết nối mới sẽ ngắt kết nối cũ)
   - Hỗ trợ chủ động đẩy tin nhắn

### Luồng kết nối

```
Máy chủ chính (internal/domain/config/manager/websocket_client.go)  →  Manager Backend (websocket.go)
        Client                          Server (kết nối đơn)
```

## Luồng chi tiết

### 1. Thiết lập kết nối

#### Manager Backend khởi động WebSocket server
```go
// Trong websocket.go
controller := NewWebSocketController(db)
// Đăng ký trong router
router.GET("/ws", controller.HandleWebSocket)
```

#### Máy chủ chính kết nối đến Manager Backend
```go
// Trong internal/domain/config/manager/websocket_client.go
client := manager.NewWebSocketClient()
err := client.Connect(ctx)
```

Định dạng URL kết nối:
- Nếu cấu hình là `http://localhost:8080`
- Thực tế kết nối đến `ws://localhost:8080/ws`

**Quan trọng**: Nếu có yêu cầu kết nối mới, Manager Backend sẽ tự động ngắt kết nối hiện tại và chỉ giữ lại kết nối mới nhất.

### 2. Luồng yêu cầu danh sách công cụ

#### Máy chủ chính yêu cầu danh sách công cụ MCP
```go
// Trong internal/domain/config/manager/websocket_client.go
response, err := client.SendRequest(ctx, "GET", "/api/mcp/tools", map[string]interface{}{
    "agent_id": "some_agent_id",
})
```

#### Manager Backend xử lý yêu cầu
```go
// Trong websocket.go
func (client *WebSocketClient) handleMcpToolListRequest(request *WebSocketRequest) {
    agentID := request.Body["agent_id"].(string)
    
    // Logic lấy danh sách công cụ
    response := map[string]interface{}{
        "agent_id": agentID,
        "tools":    []string{"tool1", "tool2", "tool3"},
        "count":    3,
    }
    
    client.sendResponse(request.ID, 200, response, "")
}
```

### 3. Hỗ trợ giao tiếp hai chiều

### Client → Server (chức năng hiện có)
#### Máy chủ chính yêu cầu danh sách công cụ MCP
```go
// Trong internal/domain/config/manager/websocket_client.go
response, err := client.SendRequest(ctx, "GET", "/api/mcp/tools", map[string]interface{}{
    "agent_id": "some_agent_id",
})
```

#### Manager Backend xử lý yêu cầu
```go
// Trong websocket.go
func (client *WebSocketClient) handleMcpToolListRequest(request *WebSocketRequest) {
    agentID := request.Body["agent_id"].(string)
    
    // Logic lấy danh sách công cụ
    response := map[string]interface{}{
        "agent_id": agentID,
        "tools":    []string{"tool1", "tool2", "tool3"},
        "count":    3,
    }
    
    client.sendResponse(request.ID, 200, response, "")
}
```

### Server → Client (chức năng mới bổ sung)
#### Manager Backend chủ động yêu cầu client
```go
// Trong websocket.go
func (ctrl *WebSocketController) RequestMcpToolsFromClient(ctx context.Context, agentID string) (*WebSocketResponse, error) {
    body := map[string]interface{}{
        "agent_id": agentID,
    }
    return ctrl.SendRequestToClient(ctx, "GET", "/api/mcp/tools", body)
}

// Yêu cầu thông tin server từ client
func (ctrl *WebSocketController) RequestServerInfoFromClient(ctx context.Context) (*WebSocketResponse, error) {
    return ctrl.SendRequestToClient(ctx, "GET", "/api/server/info", nil)
}

// Yêu cầu ping từ client
func (ctrl *WebSocketController) RequestPingFromClient(ctx context.Context) (*WebSocketResponse, error) {
    return ctrl.SendRequestToClient(ctx, "GET", "/api/server/ping", nil)
}
```

#### Client xử lý yêu cầu từ server
```go
// Trong internal/domain/config/manager/websocket_client.go
client.SetRequestHandler(func(request *WebSocketRequest) {
    // Xử lý yêu cầu nhận được
    switch request.Path {
    case "/api/mcp/tools":
        // Xử lý yêu cầu danh sách công cụ MCP
        c.handleMcpToolListRequest(request)
    case "/api/server/info":
        // Xử lý yêu cầu thông tin server
        c.handleServerInfoRequest(request)
    case "/api/server/ping":
        // Xử lý yêu cầu ping
        c.handlePingRequest(request)
    }
})
```

### Ví dụ giao tiếp hai chiều hoàn chỉnh
```go
// 1. Client kết nối đến server
client := manager.NewWebSocketClient()
err := client.Connect(ctx)

// 2. Client thiết lập trình xử lý yêu cầu
client.SetRequestHandler(func(request *WebSocketRequest) {
    // Xử lý yêu cầu từ server
    // và gửi phản hồi
})

// 3. Client chủ động yêu cầu server
response, err := client.SendRequest(ctx, "GET", "/api/mcp/tools", map[string]interface{}{
    "agent_id": "agent_123",
})

// 4. Server chủ động yêu cầu client
serverResponse, err := websocketController.RequestMcpToolsFromClient(ctx, "agent_456")

// 5. Giao tiếp hai chiều hoàn tất
```

## Định dạng tin nhắn

### Tin nhắn yêu cầu (WebSocketRequest)
```json
{
    "id": "uuid-string",
    "method": "GET",
    "path": "/api/mcp/tools",
    "body": {
        "agent_id": "agent_123"
    }
}
```

### Tin nhắn phản hồi (WebSocketResponse)
```json
{
    "id": "uuid-string",
    "status": 200,
    "body": {
        "agent_id": "agent_123",
        "tools": ["tool1", "tool2", "tool3"],
        "count": 3
    },
    "error": ""
}
```

### Tin nhắn Ping/Pong
```json
// Ping
{"ping": 1640995200}

// Pong
{"pong": 1640995200}
```

## Quản lý kết nối

### Chiến lược kết nối đơn
- **Chỉ giữ lại kết nối hợp lệ cuối cùng**
- Kết nối mới sẽ tự động ngắt kết nối hiện tại
- Đơn giản hóa logic quản lý kết nối
- Phù hợp với tình huống giao tiếp một-một

### Giám sát trạng thái kết nối
```go
// Kiểm tra xem có client đang kết nối không
func (ctrl *WebSocketController) HasConnectedClient() bool

// Lấy client đang kết nối hiện tại
func (ctrl *WebSocketController) GetCurrentClient() *WebSocketClient
```

### Logic chuyển đổi kết nối
```go
// Trong HandleWebSocket
if ctrl.currentClient != nil && ctrl.currentClient.isConnected {
    log.Printf("Ngắt kết nối hiện có: %s", ctrl.currentClient.ID)
    ctrl.currentClient.conn.Close()
    ctrl.currentClient.isConnected = false
}

// Đặt kết nối mới làm client hiện tại
ctrl.currentClient = client
```

## Xử lý lỗi

### Lỗi kết nối
- Tự động phát hiện bằng heartbeat
- Tự động ngắt kết nối khi hết thời gian chờ
- Tự động dọn dẹp khi kết nối bị lỗi
- Kết nối mới tự động thay thế kết nối cũ

### Lỗi tin nhắn
- Xác thực định dạng tin nhắn
- Trả về phản hồi lỗi
- Ghi log

## Yêu cầu cấu hình

### Cấu hình máy chủ chính
```yaml
manager:
  backend_url: "http://localhost:8080"
```

### Cấu hình Manager Backend
```go
// Đăng ký endpoint WebSocket trong router
router.GET("/ws", websocketController.HandleWebSocket)
```

## Khuyến nghị kiểm thử

1. **Kiểm thử kết nối**
   - Xác nhận việc thiết lập kết nối WebSocket
   - Kiểm thử kết nối mới ngắt kết nối cũ
   - Kiểm thử kết nối lại sau khi ngắt

2. **Kiểm thử chức năng**
   - Kiểm thử yêu cầu danh sách công cụ MCP
   - Xác nhận giao tiếp hai chiều
   - Kiểm thử đẩy tin nhắn

3. **Kiểm thử lỗi**
   - Kết nối lại sau khi mạng bị ngắt
   - Xử lý tin nhắn không hợp lệ
   - Xử lý timeout
   - Timeout heartbeat
   - Chuyển đổi kết nối

## Lưu ý

1. **Giới hạn kết nối đơn**
   - Tại một thời điểm chỉ có thể có một kết nối hoạt động
   - Kết nối mới sẽ buộc ngắt kết nối cũ
   - Phù hợp với kiến trúc master-slave, không phù hợp với tình huống nhiều client

2. **An toàn đồng thời**
   - Sử dụng read-write lock để bảo vệ tham chiếu client hiện tại
   - Chuyển đổi client an toàn
   - Gửi tin nhắn an toàn theo luồng

3. **Quản lý tài nguyên**
   - Dọn dẹp kịp thời các kết nối đã ngắt
   - Đóng kết nối WebSocket đúng cách
   - Tránh rò rỉ bộ nhớ

4. **Cơ chế heartbeat**
   - Gửi ping mỗi 30 giây
   - Tự động ngắt kết nối sau 60 giây không có phản hồi
   - Hỗ trợ tin nhắn ping/pong

5. **Ghi log**
   - Ghi lại các thay đổi trạng thái kết nối
   - Ghi lại việc chuyển đổi kết nối
   - Ghi lại thông tin yêu cầu và phản hồi
   - Ghi lại các lỗi và tình huống bất thường

## Ví dụ sử dụng hoàn chỉnh

### Mã kiểm thử giao tiếp hai chiều
```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"
    
    "xiaozhi-esp32-server-golang/internal/domain/config/manager"
)

func main() {
    ctx := context.Background()
    
    // 1. Tạo client và kết nối
    client := manager.NewWebSocketClient()
    if err := client.Connect(ctx); err != nil {
        log.Fatalf("Kết nối thất bại: %v", err)
    }
    defer client.Disconnect()
    
    // 2. Thiết lập trình xử lý yêu cầu (xử lý yêu cầu từ server)
    client.SetRequestHandler(func(request *manager.WebSocketRequest) {
        log.Printf("Nhận được yêu cầu từ server: %s %s", request.Method, request.Path)
        
        switch request.Path {
        case "/api/mcp/tools":
            // Xử lý yêu cầu danh sách công cụ MCP
            agentID := ""
            if request.Body != nil {
                if id, ok := request.Body["agent_id"].(string); ok {
                    agentID = id
                }
            }
            
            response := map[string]interface{}{
                "agent_id": agentID,
                "tools":    []string{"client_tool_1", "client_tool_2"},
                "count":    2,
            }
            
            client.SendResponse(request.ID, 200, response, "")
            
        case "/api/server/info":
            response := map[string]interface{}{
                "server_name": "xiaozhi-client",
                "version":     "1.0.0",
                "uptime":      time.Now().Format(time.RFC3339),
            }
            client.SendResponse(request.ID, 200, response, "")
            
        case "/api/server/ping":
            response := map[string]interface{}{
                "message": "pong from client",
                "time":    time.Now().Format(time.RFC3339),
            }
            client.SendResponse(request.ID, 200, response, "")
        }
    })
    
    // 3. Client chủ động yêu cầu server
    fmt.Println("=== Client yêu cầu Server ===")
    response, err := client.SendRequest(ctx, "GET", "/api/mcp/tools", map[string]interface{}{
        "agent_id": "client_agent_123",
    })
    if err != nil {
        log.Printf("Yêu cầu từ client thất bại: %v", err)
    } else {
        fmt.Printf("Phản hồi từ server: %+v\n", response)
    }
    
    // 4. Chờ một khoảng thời gian để server có cơ hội gửi yêu cầu
    fmt.Println("Đang chờ yêu cầu từ server...")
    time.Sleep(5 * time.Second)
    
    fmt.Println("Kiểm thử giao tiếp hai chiều hoàn tất!")
}
```

### Mã kiểm thử phía server
```go
// Trong Manager Backend
func testBidirectionalCommunication() {
    ctx := context.Background()
    
    // 1. Kiểm tra trạng thái kết nối client
    status := websocketController.GetClientConnectionStatus()
    fmt.Printf("Trạng thái client: %+v\n", status)
    
    // 2. Server chủ động yêu cầu client
    fmt.Println("=== Server yêu cầu Client ===")
    
    // Yêu cầu danh sách công cụ MCP
    response, err := websocketController.RequestMcpToolsFromClient(ctx, "server_agent_456")
    if err != nil {
        log.Printf("Yêu cầu danh sách công cụ MCP thất bại: %v", err)
    } else {
        fmt.Printf("Phản hồi công cụ MCP từ client: %+v\n", response)
    }
    
    // Yêu cầu thông tin server
    infoResponse, err := websocketController.RequestServerInfoFromClient(ctx)
    if err != nil {
        log.Printf("Yêu cầu thông tin server thất bại: %v", err)
    } else {
        fmt.Printf("Thông tin server từ client: %+v\n", infoResponse)
    }
    
    // Yêu cầu ping
    pingResponse, err := websocketController.RequestPingFromClient(ctx)
    if err != nil {
        log.Printf("Yêu cầu ping thất bại: %v", err)
    } else {
        fmt.Printf("Phản hồi ping từ client: %+v\n", pingResponse)
    }
}
```

## Lưu ý

1. **Yêu cầu giao tiếp hai chiều**
   - Client phải thiết lập trình xử lý yêu cầu
   - Cả server và client đều phải triển khai các phương thức xử lý yêu cầu tương ứng
   - ID yêu cầu phải khớp để đảm bảo phản hồi được định tuyến chính xác

2. **Xử lý lỗi**
   - Giao tiếp hai chiều sẽ thất bại khi mạng bị ngắt
   - Xử lý timeout rất quan trọng
   - Kiểm tra trạng thái kết nối là không thể thiếu

3. **Cân nhắc hiệu năng**
   - Tránh các yêu cầu hai chiều quá thường xuyên
   - Đặt thời gian timeout hợp lý
   - Giám sát trạng thái kết nối
