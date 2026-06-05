# Tài liệu các loại nội dung trả về khi gọi công cụ MCP

## Tổng quan

Tài liệu này mô tả chi tiết các loại nội dung trả về khi gọi công cụ mà chương trình hỗ trợ. Chương trình sử dụng **hệ thống phản hồi có cấu trúc**, hỗ trợ xử lý và render nhiều loại nội dung khác nhau.

## 🔧 Luồng xử lý cốt lõi

### Xử lý phản hồi gọi công cụ

Bộ xử lý cốt lõi của phản hồi gọi công cụ chịu trách nhiệm:

1. **Thực thi gọi công cụ**: Duyệt qua tất cả các yêu cầu gọi công cụ
2. **Phân tích kết quả**: Phân tích kết quả trả về từ công cụ
3. **Nhận diện loại nội dung**: Xử lý khác nhau tùy theo loại nội dung
4. **Render tài nguyên**: Xử lý các loại nội dung khác nhau như âm thanh, văn bản, liên kết tài nguyên

## 📋 Các loại nội dung được hỗ trợ

### 1. Nội dung âm thanh (AudioContent)

**Kiểu**: `mcp_go.AudioContent`

**Đặc điểm**:
- Chứa dữ liệu âm thanh được mã hóa Base64
- Hỗ trợ nhiều định dạng âm thanh (MIME Type)
- Phát trực tiếp, kết thúc quá trình xử lý LLM tiếp theo

**Luồng xử lý**:
```go
if audioContent, ok := content.(mcp_go.AudioContent); ok {
    // Giải mã dữ liệu âm thanh Base64
    rawAudioData, err := base64.StdEncoding.DecodeString(audioContent.Data)
    // Sử dụng music_player để phát âm thanh
    audioChan, err := play_music.PlayMusicFromAudioData(ctx, rawAudioData, ...)
    // Gửi thông báo trạng thái phát
    l.serverTransport.SendSentenceStart(playText)
    // Phát âm thanh thông qua TTS manager
    l.ttsManager.SendTTSAudio(ctx, audioChan, true)
}
```

**Trường hợp sử dụng**:
- Công cụ phát nhạc
- Công cụ tổng hợp giọng nói
- Phát tệp âm thanh

### 2. Liên kết tài nguyên (ResourceLink)

**Kiểu**: `mcp_go.ResourceLink`

**Đặc điểm**:
- Chứa URI tài nguyên và metadata
- Hỗ trợ đọc phân trang cho tài nguyên lớn
- Xử lý luồng, phù hợp với tệp lớn
- Sử dụng cơ chế Pipe để phát luồng âm thanh theo thời gian thực

**Luồng xử lý**:
```go
if resourceLink, ok := content.(mcp_go.ResourceLink); ok {
    // Tạo Pipe để truyền dữ liệu luồng
    pipeReader, pipeWriter = io.Pipe()
    
    // Khởi động goroutine đọc phân trang
    go func() {
        // Đọc tài nguyên theo trang
        resourceResult, err := client.ReadResource(readCtx, mcp_go.ReadResourceRequest{
            Params: mcp_go.ReadResourceParams{
                URI: resourceLink.URI,
                Arguments: map[string]any{
                    "url": resourceLink.Description, 
                    "start": start, 
                    "end": start + page
                },
            },
        })
        
        // Xử lý BlobResourceContents
        for _, content := range resourceResult.Contents {
            if audioContent, ok := content.(mcp_go.BlobResourceContents); ok {
                // Giải mã và gửi vào kênh luồng âm thanh
                rawAudioData, err := base64.StdEncoding.DecodeString(audioContent.Blob)
                streamChan <- rawAudioData
            }
        }
    }()
    
    // Sử dụng music_player để phát luồng âm thanh
    audioChan, err := play_music.PlayMusicFromPipe(ctx, pipeReader, ...)
}
```

**Chi tiết tham số đọc phân trang**:

#### Định dạng tham số yêu cầu
```go
Arguments: map[string]any{
    "url": resourceLink.Description,  // URL tài nguyên thực tế
    "start": start,                   // Vị trí byte bắt đầu
    "end": start + page,              // Vị trí byte kết thúc
}
```

#### Giải thích tham số
- **url**: Địa chỉ URL của tài nguyên thực tế, lấy từ `resourceLink.Description`
- **start**: Vị trí byte bắt đầu, tính từ 0
- **end**: Vị trí byte kết thúc (không bao gồm), tức là phạm vi đọc [start, end)
- **Kích thước trang**: Được định nghĩa bởi hằng số `McpReadResourcePageSize`, mặc định 100KB

#### Luồng đọc phân trang
```go
start := 0
page := McpReadResourcePageSize  // 100 * 1024
totalRead := 0
pageCount := 0

for {
    // Tạo context với timeout
    readCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
    
    // Gửi yêu cầu đọc phân trang
    resourceResult, err := client.ReadResource(readCtx, mcp_go.ReadResourceRequest{
        Params: mcp_go.ReadResourceParams{
            URI: resourceLink.URI,
            Arguments: map[string]any{
                "url": resourceLink.Description, 
                "start": start, 
                "end": start + page
            },
        },
    })
    cancel()
    
    // Xử lý BlobResourceContents trả về
    for _, content := range resourceResult.Contents {
        if audioContent, ok := content.(mcp_go.BlobResourceContents); ok {
            // Giải mã dữ liệu Base64
            rawAudioData, err := base64.StdEncoding.DecodeString(audioContent.Blob)
            
            // Kiểm tra có phải cờ kết thúc không
            if string(rawAudioData) == McpReadResourceStreamDoneFlag {
                return nil // Đọc hoàn tất
            }
            
            // Gửi vào kênh luồng âm thanh
            streamChan <- rawAudioData
            totalRead += len(rawAudioData)
        }
    }
    
    // Kiểm tra điều kiện đọc hoàn tất
    if len(rawAudioData) < page || !hasData {
        return nil // Đọc hoàn tất
    }
    
    // Cập nhật vị trí bắt đầu
    start += page
    pageCount++
}
```

#### Cơ chế xử lý luồng

**Kiến trúc truyền dữ liệu qua Pipe**:
```go
// Tạo Pipe để truyền luồng âm thanh
pipeReader, pipeWriter = io.Pipe()

// Khởi động goroutine ghi dữ liệu
go func() {
    for {
        select {
        case audioData, ok := <-streamChan:
            if !ok {
                pipeWriter.Close()
                return
            }
            pipeWriter.Write(audioData)
        case <-ctx.Done():
            return
        }
    }
}()

// Sử dụng music_player phát âm thanh từ Pipe
audioChan, err := play_music.PlayMusicFromPipe(ctx, pipeReader, ...)
```

#### Cơ chế xử lý lỗi

**Thử lại khi timeout**:
```go
if err != nil {
    // Nếu là lỗi timeout, thử lại
    if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "deadline") {
        log.Warnf("Đọc tài nguyên bị timeout, đang thử lại...")
        time.Sleep(1 * time.Second)
        continue
    }
    return fmt.Errorf("đọc tài nguyên thất bại: %v", err)
}
```

**Hủy context**:
```go
select {
case <-ctx.Done():
    log.Debugf("Đọc tài nguyên bị hủy")
    return nil
case streamChan <- rawAudioData:
    // Gửi dữ liệu bình thường
}
```

#### Đặc điểm cơ chế phân trang
- **Tối ưu bộ nhớ**: Đọc phân trang tránh tải toàn bộ tệp lớn vào bộ nhớ cùng lúc
- **Xử lý luồng**: Đọc và phát đồng thời, hỗ trợ luồng âm thanh thời gian thực
- **Tự động kết thúc**: Phát hiện cờ `McpReadResourceStreamDoneFlag` để xác định hoàn tất
- **Khôi phục lỗi**: Hỗ trợ thử lại khi timeout và hủy context
- **Phát thời gian thực**: Sử dụng cơ chế Pipe để phát trong khi đọc
- **Kiểm soát timeout**: Mỗi lần đọc phân trang có giới hạn timeout 30 giây

#### Tham số cấu hình
- **McpReadResourcePageSize**: Kích thước trang đọc, mặc định 100KB (100 * 1024)
- **McpReadResourceStreamDoneFlag**: Cờ kết thúc luồng, giá trị là `"[DONE]"`
- **Timeout đọc**: Thời gian timeout mỗi lần đọc phân trang, mặc định 30 giây
- **Cơ chế thử lại**: Tự động thử lại khi có lỗi timeout, khoảng cách 1 giây

**Trường hợp sử dụng**:
- Phát tệp âm thanh lớn
- Xử lý tài nguyên media trực tuyến
- Truy cập tài nguyên mạng
- Phát luồng âm thanh thời gian thực

### 3. Nội dung văn bản (TextContent)

**Kiểu**: `mcp_go.TextContent`

**Đặc điểm**:
- Nội dung văn bản thuần túy
- Tích lũy vào thông điệp phản hồi
- Không kết thúc quá trình xử lý tiếp theo

**Luồng xử lý**:
```go
if textContent, ok := content.(mcp_go.TextContent); ok {
    mcpContent += textContent.Text
}
```

**Trường hợp sử dụng**:
- Trả về kết quả truy vấn
- Hiển thị thông tin trạng thái
- Hiển thị thông báo lỗi

### 4. Nội dung tài nguyên Blob (BlobResourceContents)

**Kiểu**: `mcp_go.BlobResourceContents`

**Đặc điểm**:
- Nội dung dữ liệu nhị phân
- Mã hóa Base64
- Hỗ trợ xử lý luồng

**Luồng xử lý**:
```go
if audioContent, ok := content.(mcp_go.BlobResourceContents); ok {
    rawAudioData, err := base64.StdEncoding.DecodeString(audioContent.Blob)
    // Kiểm tra có phải cờ kết thúc không
    if string(rawAudioData) == McpReadResourceStreamDoneFlag {
        return nil
    }
    // Gửi vào kênh luồng âm thanh
    streamChan <- rawAudioData
}
```

## 🏗️ Hệ thống phản hồi có cấu trúc

### Phân loại kiểu phản hồi

Chương trình hỗ trợ bốn loại phản hồi chính:

#### 1. Phản hồi hành động (MCPActionResponse)
- **Mục đích**: Thực thi các hành động cụ thể, như phát nhạc, thoát hội thoại
- **Tính kết thúc**: Có thể cấu hình, thường kết thúc quá trình xử lý LLM tiếp theo
- **Cờ điều khiển**: `FinalAction`, `NoFurtherResponse`, `SilenceLLM`

#### 2. Phản hồi âm thanh (MCPAudioResponse)
- **Mục đích**: Phát tài nguyên âm thanh
- **Tính kết thúc**: Thường kết thúc quá trình xử lý tiếp theo
- **Đặc điểm**: Chứa dữ liệu âm thanh và thông tin phát

#### 3. Phản hồi nội dung (MCPContentResponse)
- **Mục đích**: Trả về dữ liệu truy vấn, thông tin trạng thái
- **Tính kết thúc**: Không kết thúc quá trình xử lý tiếp theo
- **Đặc điểm**: Chứa dữ liệu và gợi ý hiển thị

#### 4. Phản hồi lỗi (MCPErrorResponse)
- **Mục đích**: Xử lý lỗi thống nhất
- **Tính kết thúc**: Không kết thúc quá trình xử lý tiếp theo
- **Đặc điểm**: Chứa mã lỗi và gợi ý

### Giao diện xử lý phản hồi

```go
type MCPResponse interface {
    GetType() MCPResponseType
    GetSuccess() bool
    IsTerminal() bool // Quan trọng: xác định có kết thúc xử lý LLM tiếp theo không
    ToJSON() (string, error)
    GetContent() []mcp_go.Content
}
```

## 🔄 Chi tiết luồng xử lý

### 1. Thực thi gọi công cụ
```go
fcResult, err := tool.InvokableRun(toolCtx, toolCall.Function.Arguments)
```

### 2. Phân tích kết quả
```go
// Thử phân tích kết quả công cụ nội bộ
if mcpResp, ok := l.handleLocalToolResult(fcResult); ok {
    contentList = mcpResp.GetContent()
} else if toolCallResult, ok := l.handleToolResult(fcResult); ok {
    contentList = toolCallResult.Content
}
```

> `handleToolResult` **không còn yêu cầu giá trị trả về của công cụ phải là JSON**.  
> - Nếu trả về là JSON `CallToolResult` MCP chuẩn, sẽ được phân tích theo nội dung có cấu trúc.  
> - Nếu trả về là chuỗi thông thường, sẽ tự động được bọc thành `TextContent` để tiếp tục xử lý.  
> Như vậy cả công cụ văn bản thông thường và công cụ MCP có cấu trúc đều có thể được xử lý thống nhất.

### 3. Xử lý loại nội dung
```go
for _, content := range contentList {
    switch content.(type) {
    case mcp_go.AudioContent:
        // Xử lý nội dung âm thanh
    case mcp_go.ResourceLink:
        // Xử lý liên kết tài nguyên
    case mcp_go.TextContent:
        // Xử lý nội dung văn bản
    }
}
```

### 4. Kiểm soát xử lý tiếp theo
```go
if invokeToolSuccess && !shouldStopLLMProcessing {
    l.DoLLmRequest(ctx, nil, l.einoTools, true)
}
```

## 📊 Bảng so sánh các loại nội dung

| Loại nội dung | Tính kết thúc | Cách xử lý | Trường hợp sử dụng | Công cụ ví dụ |
|----------|--------|----------|----------|----------|
| **AudioContent** | Kết thúc | Phát trực tiếp | Tệp âm thanh nhỏ | play_music |
| **ResourceLink** | Kết thúc | Đọc phân trang + phát luồng | Tệp lớn/media trực tuyến | music_player |
| **TextContent** | Không kết thúc | Tích lũy văn bản | Truy vấn thông tin | get_datetime |
| **BlobResourceContents** | Kết thúc | Xử lý luồng | Dữ liệu luồng âm thanh | audio_stream |

## 🎯 Thực hành tốt nhất

### 1. Khuyến nghị triển khai công cụ
- **Công cụ âm thanh**: Trả về `AudioContent` hoặc `ResourceLink`
- **Công cụ truy vấn**: Trả về `TextContent`
- **Công cụ hành động**: Sử dụng hệ thống phản hồi có cấu trúc

### 2. Tối ưu hiệu suất
- Tệp lớn sử dụng `ResourceLink` để đọc phân trang, hỗ trợ phát luồng
- Tệp âm thanh nhỏ dùng trực tiếp `AudioContent`, giảm chi phí mạng
- Tránh nội dung văn bản quá dài, ảnh hưởng tốc độ phản hồi
- Sử dụng cơ chế Pipe để phát trong khi đọc, nâng cao trải nghiệm người dùng

### 3. Xử lý lỗi
- Sử dụng `MCPErrorResponse` để thống nhất định dạng lỗi
- Cung cấp mã lỗi và gợi ý có ý nghĩa
- Duy trì tính tương thích ngược

## 🔧 Tham số cấu hình

### Cấu hình phân trang
- `McpReadResourcePageSize`: Kích thước trang đọc tài nguyên, mặc định 100KB (100 * 1024)
- `McpReadResourceStreamDoneFlag`: Cờ kết thúc luồng, giá trị là `"[DONE]"`
- **Timeout đọc**: Thời gian timeout mỗi lần đọc phân trang, mặc định 30 giây
- **Cơ chế thử lại**: Tự động thử lại khi có lỗi timeout, khoảng cách 1 giây

### Cấu hình âm thanh
- `OutputAudioFormat.SampleRate`: Tốc độ lấy mẫu âm thanh đầu ra
- `OutputAudioFormat.FrameDuration`: Thời lượng khung âm thanh đầu ra
- **Định dạng âm thanh**: Tự động nhận diện theo `resourceLink.MIMEType`

## 📝 Hướng dẫn mở rộng

### Thêm loại nội dung mới
1. Định nghĩa loại nội dung mới trong gói `mcp_go`
2. Thêm logic xử lý loại vào `handleToolCallResponse`
3. Triển khai hàm xử lý tương ứng
4. Cập nhật tài liệu và kiểm thử

### Tùy chỉnh kiểu phản hồi
1. Kế thừa `MCPResponseBase`
2. Triển khai giao diện `MCPResponse`
3. Thêm logic phân tích vào `ParseMCPResponse`
4. Cung cấp hàm khởi tạo tiện lợi

## 🎵 Kho lưu trữ độc lập MCP Audio Server

### Tổng quan

MCP Audio Server đã được tách thành kho lưu trữ độc lập, khuyến nghị chạy và gỡ lỗi Audio MCP Server thông qua dự án độc lập. Phần này trong tài liệu hiện tại chủ yếu giải thích cách thức tương thích giao thức với dịch vụ chính.

### Chức năng cốt lõi

#### 1. Công cụ phát nhạc
- **Tên công cụ**: `musicPlayer`
- **Chức năng**: Tìm kiếm và phát nhạc
- **Trả về**: Liên kết tài nguyên âm thanh kiểu `ResourceLink`

#### 2. Template tài nguyên âm thanh
- **Định dạng URI**: `resource://read_from_http`
- **Chức năng**: Hỗ trợ đọc phân trang dữ liệu âm thanh, truyền tham số qua Arguments
- **Tham số**: url (URL nhạc thực tế), start (vị trí bắt đầu), end (vị trí kết thúc)
- **Trả về**: Dữ liệu âm thanh kiểu `BlobResourceContents`

### Tính năng quan trọng

- **Đọc phân trang**: Hỗ trợ xử lý luồng cho tệp lớn
- **HTTP Range request**: Thực hiện lấy dữ liệu âm thanh theo từng đoạn
- **Xử lý lỗi**: Xử lý các trường hợp bất thường như mã trạng thái 416
- **Thử lại khi timeout**: Tự động thử lại lỗi timeout, khoảng cách 1 giây
- **Hủy context**: Hỗ trợ hủy đọc tài nguyên một cách graceful
- **Mã hóa Base64**: Truyền tham số URL nhạc an toàn
- **Hỗ trợ nhiều phương thức truyền tải**: Hai phương thức stdio và HTTP
- **Phát thời gian thực**: Sử dụng cơ chế Pipe để phát trong khi đọc

### Cách sử dụng

```bash
# Lấy và vào kho lưu trữ độc lập
git clone https://github.com/hackers365/mcp_audio_server.git
cd mcp_audio_server

# Khởi động server
go run .

# Gọi công cụ
{
  "name": "musicPlayer",
  "arguments": {"query": "周杰伦"}
}
```

Dự án độc lập này trình bày cách xây dựng công cụ MCP hỗ trợ xử lý tài nguyên âm thanh, có thể dùng làm template tham khảo khi phát triển các công cụ liên quan đến âm thanh khác. Hướng dẫn sử dụng đầy đủ hơn có thể tham khảo tại `doc/mcp_audio_example.md`.

---

*Tài liệu này phản ánh tất cả các loại nội dung trả về khi gọi công cụ mà chương trình hiện đang hỗ trợ.* 
