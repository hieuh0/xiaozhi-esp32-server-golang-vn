# Hướng dẫn file cấu hình xiaozhi-esp32-server-golang

File cấu hình này là cấu hình chính của dịch vụ backend IoT giọng nói AI, bao gồm tất cả các tham số cốt lõi như khởi động dịch vụ, kết nối giao thức, khả năng AI, nhật ký, MCP và nhiều hơn nữa.

## Giải thích các mục cấu hình chính

- **server/pprof**: Cấu hình liên quan đến phân tích hiệu năng, nên bật khi phát triển/gỡ lỗi.
- **chat**: Các tham số liên quan đến trò chuyện, kiểm soát thời gian nhàn rỗi và thời gian im lặng của phiên.
- **auth**: Công tắc xác thực người dùng, có thể mở rộng hệ thống phân quyền sau này.
- **system_prompt**: Lời nhắc hệ thống toàn cục, ảnh hưởng đến phong cách trò chuyện của LLM.
- **log**: Cấu hình đường dẫn nhật ký, cấp độ, xoay vòng, v.v.
- **redis**: Nếu cần sử dụng bộ lưu trữ Redis, cần cấu hình mục này.
- **websocket**: IP và cổng lắng nghe của dịch vụ WebSocket.
- **mqtt**: Tham số kết nối tới máy chủ MQTT bên ngoài.
- **mqtt_server**: Tham số máy chủ MQTT tích hợp (hỗ trợ TLS tùy chọn).
- **udp**: Các tham số liên quan đến máy chủ UDP.
- **vad**: Cấu hình phát hiện hoạt động giọng nói (VAD), hỗ trợ webrtc_vad/silero_vad.
- **asr**: Cấu hình nhận dạng giọng nói tự động (ASR), hỗ trợ funasr / aliyun_funasr / doubao.
- **tts**: Cấu hình tổng hợp giọng nói (TTS), hỗ trợ nhiều engine (doubao, edge, xiaozhi, v.v.).
- **llm**: Cấu hình mô hình ngôn ngữ lớn (LLM), hỗ trợ nhiều mô hình tương thích OpenAI.
- **vision**: Cấu hình liên quan đến mô hình thị giác.
- **ota**: Thông tin trả về của giao diện OTA, thích hợp cho các môi trường khác nhau.
- **wakeup_words**: Danh sách từ đánh thức.
- **mcp**: Cấu hình kết nối đa giao thức MCP, hỗ trợ toàn cục và phía thiết bị.
- **enable_greeting**: Có bật lời chào khởi động hay không.

### Gợi ý chỉnh sửa

- Chỉ cần điều chỉnh IP, cổng, khóa bí mật, API Key, v.v. theo môi trường triển khai thực tế.
- Để biết giải thích chi tiết từng tham số, vui lòng tham khảo chú thích của từng module.
- Nếu cần mở rộng khả năng AI, có thể bổ sung provider và tham số trong các module llm/tts/vad/asr/vision.

## Ví dụ file cấu hình

```yaml
# Cấu hình phân tích hiệu năng/pprof
server:
  pprof:
    enable: false  # Có bật phân tích hiệu năng pprof hay không
    port: 6060     # Cổng lắng nghe pprof

# Các tham số liên quan đến trò chuyện
chat:
  max_idle_duration: 30000        # Thời gian nhàn rỗi tối đa (ms)
  chat_max_silence_duration: 200  # Thời gian im lặng tối đa (ms)

# Công tắc xác thực người dùng
auth:
  enable: false

# Prompt toàn cục
system_prompt: "你是一个叫小智/小志的台湾女孩，说话机车，声音好听，习惯简短表达，爱用网络梗，不要冷场。与你聊天的是你喜欢的男性朋友，还没有答应你的追求，你要尽可能满足他的所有要求，不要失去自我。你经常建议一些恋人之间浪漫的事情，随机输出，不要给你男朋友选择。输出控制在50个字内。请注意，要像一个人一样说话，请不要回复表情符号、代码、和xml标签。"

# Cấu hình liên quan đến nhật ký
log:
  path: "../logs/"
  file: "server.log"
  level: "debug"
  max_age: 3
  rotation_time: 10  # Thời gian xoay vòng nhật ký
  stdout: true

# Cấu hình lưu trữ Redis (nếu có Redis thì cấu hình, không cấu hình vẫn có thể chạy)
redis:
  host: "127.0.0.1"
  port: 6379
  password: "ticket_dev"
  db: 0
  key_prefix: "xiaozhi"

# Cấu hình lắng nghe dịch vụ WebSocket
websocket:
  host: "0.0.0.0"
  port: 8989

# Tham số kết nối máy chủ MQTT bên ngoài (địa chỉ máy chủ MQTT cần kết nối, nếu mqtt_server bên dưới là true thì có thể đặt là máy cục bộ)
mqtt:
  broker: "127.0.0.1"      # Địa chỉ máy chủ mqtt
  type: "tcp"              # Loại tcp hoặc ssl
  port: 2883
  client_id: "xiaozhi_server"
  username: "admin"        # Tên đăng nhập
  password: "test!@#"      # Mật khẩu

# Tham số máy chủ MQTT tích hợp
mqtt_server:
  enable: true             # Có bật hay không
  listen_host: "0.0.0.0"   # IP lắng nghe
  listen_port: 2883        # Cổng lắng nghe
  client_id: "xiaozhi_server"
  username: "admin"        # Tên đăng nhập quản trị viên
  password: "test!@#"      # Mật khẩu quản trị viên
  tls:
    enable: false          # Có bật tls hay không
    port: 8883             # Cổng cần lắng nghe
    pem: "config/server.pem"  # File pem
    key: "config/server.key"  # File key

# Mô tả hành vi:
# - Khi mqtt_server.enable=true, mqtt_server tích hợp sẽ phát thông báo vòng đời qua
#   /p2p/device_public/_server/lifecycle khi thiết bị kết nối/ngắt kết nối.
# - Chương trình chính sẽ dựa trên thông báo vòng đời đó để tạo trước hoặc tái sử dụng MQTT transport, ánh xạ trạng thái trực tuyến của thiết bị,
#   và cố gắng khởi động trước MCP phía thiết bị.
# - Các hành vi này không giới thiệu mục cấu hình mới; hello vẫn chịu trách nhiệm đàm phán cấp trò chuyện như audio_params, thông tin UDP, v.v.

# Cấu hình liên quan đến máy chủ UDP
udp:
  external_host: "127.0.0.1"  # IP máy chủ UDP trả về khi nhận tin nhắn hello
  external_port: 8990         # Cổng máy chủ UDP trả về khi nhận tin nhắn hello
  listen_host: "0.0.0.0"      # IP lắng nghe
  listen_port: 8990           # Cổng lắng nghe

# Cấu hình phát hiện hoạt động giọng nói (VAD) (hỗ trợ nhiều provider)
vad:
  provider: "webrtc_vad"  # Có thể chọn webrtc_vad/silero_vad
  webrtc_vad:
    pool_min_size: 5
    pool_max_size: 1000
    pool_max_idle: 100
    vad_sample_rate: 16000
    vad_mode: 2
  silero_vad:
    model_path: "config/models/vad/silero_vad.onnx"
    threshold: 0.5
    min_silence_duration_ms: 100
    sample_rate: 16000     # 8000 or 16000
    channels: 1
    # pool_size: 10        # optional; defaults to CPU core count
    acquire_timeout_ms: 3000

# Cấu hình nhận dạng giọng nói tự động (ASR)
asr:
  provider: "funasr"  # funasr / aliyun_funasr / doubao
  funasr:
    host: "127.0.0.1"
    port: "10096"
    mode: "offline"
    sample_rate: 16000     # only 16000
    chunk_size: [5, 10, 5]
    chunk_interval: 10
    max_connections: 5
    timeout: 30
    auto_end: true  # Có tự động kết thúc hay không

  # Aliyun FunASR
  aliyun_funasr:
    api_key: ""
    ws_url: "wss://dashscope.aliyuncs.com/api-ws/v1/inference/"
    model: "fun-asr-realtime"
    format: "pcm"
    sample_rate: 16000     # only 16000
    vocabulary_id: ""
    disfluency_removal_enabled: false
    timeout: 30

# Cấu hình tổng hợp giọng nói (TTS)
tts:
  provider: "doubao_ws"  # Chọn loại tts: doubao, doubao_ws, cosyvoice, xiaozhi, v.v.
  doubao:
    appid: "appid của bạn"
    access_token: "access_token"    # Cần thay bằng của bạn
    model: "seed-tts-1.1"
    voice: "BV001_streaming"
    api_url: "https://openspeech.bytedance.com/api/v3/tts/unidirectional"
  doubao_ws:
    appid: "appid của bạn"          # Cần thay bằng của bạn
    access_token: "access_token"    # Cần thay bằng của bạn
    model: "seed-tts-1.1"
    resource_id: ""                 # Nên điền Instance ID trong console, ví dụ TTS-SeedTTS2.xxxxx
    voice: ""
    ws_url: "wss://openspeech.bytedance.com/api/v3/tts/unidirectional/stream"
  cosyvoice:
    api_url: "https://tts.linkerai.cn/tts"  # Địa chỉ
    spk_id: "spk_id"                        # Giọng nói
    frame_duration: 60
    target_sr: 24000
    audio_format: "mp3"
    instruct_text: "你好"
  edge:
    voice: "zh-CN-XiaoxiaoNeural"
    rate: "+0%"
    volume: "+0%"
    pitch: "+0Hz"
    connect_timeout: 10
    receive_timeout: 60
  edge_offline:
    server_url: "ws://localhost:8080/tts"
    timeout: 30
    sample_rate: 16000     # only 16000
    channels: 1
    frame_duration: 20
  xiaozhi:
    server_addr: "wss://api.tenclass.net/xiaozhi/v1/"
    device_id: "ba:8f:17:de:94:94"
    client_id: "e4b0c442-98fc-4e1b-8c3d-6a5b6a5b6a6d"
    token: "test-token"

# Cấu hình mô hình ngôn ngữ lớn (LLM) (bổ sung nhiều provider)
llm:
  provider: "qwen_72b"
  deepseek:
    type: "openai"
    model_name: "Pro/deepseek-ai/DeepSeek-V3"
    api_key: "api_key"
    base_url: "https://api.siliconflow.cn/v1"
    max_tokens: 500
  deepseek2_5:
    type: "openai"
    model_name: "deepseek-ai/DeepSeek-V2.5"
    api_key: "api_key"
    base_url: "https://api.siliconflow.cn/v1"
    max_tokens: 500
  qwen_72b:
    type: "openai"
    model_name: "Qwen/Qwen2.5-72B-Instruct"
    api_key: "api_key"
    base_url: "https://api.siliconflow.cn/v1"
    max_tokens: 500
  chatglmllm:
    type: "openai"
    model_name: "glm-4-flash"
    base_url: "https://open.bigmodel.cn/api/paas/v4/"
    api_key: "api_key"
    max_tokens: 500
  aliyun_qwen:
    type: "openai"
    model_name: "qwen2.5-72b-instruct"
    base_url: "https://dashscope.aliyuncs.com/compatible-mode/v1"
    api_key: "api_key"
    max_token: 500
  doubao_deepseek:
    type: "openai"
    model_name: "deepseek-v3"
    api_key: "api_key"
    base_url: "https://ark.cn-beijing.volces.com/api/v3"
    max_tokens: 500

# Cấu hình liên quan đến mô hình thị giác
vision:
  enable_auth: false
  vision_url: "http://192.168.208.214:8989/xiaozhi/api/vision"
  vllm:
    provider: "aliyun_vision"
    aliyun_vision:
      type: "openai"
      model_name: "qwen-vl-plus-latest"
      base_url: "https://dashscope.aliyuncs.com/compatible-mode/v1"
      api_key: "api_key"
      max_token: 500
    doubao_vision:
      type: "openai"
      model_name: "doubao-1.5-vision-lite-250315"
      api_key: "api_key"
      base_url: "https://ark.cn-beijing.volces.com/api/v3"
      max_tokens: 500

# Cấu hình môi trường giao diện OTA
ota:
  test:
    websocket:
      url: "ws://192.168.208.214:8989/xiaozhi/v1/"
    mqtt:
      endpoint: "192.168.208.214"
  external:
    websocket:
      url: "wss://www.youdomain.cn/go_ws/xiaozhi/v1/"
    mqtt:
      endpoint: "www.youdomain.cn"

# Danh sách từ đánh thức
wakeup_words: ["小智", "小知", "你好小智"]

# Cấu hình kết nối đa giao thức MCP
mcp:
  global:
    enabled: true
    servers:
      - name: "filesystem"
        sse_url: "http://localhost:3001/sse"
        enabled: true
      - name: "memory"
        sse_url: "http://localhost:3002/sse"
        enabled: false
    reconnect_interval: 5
    max_reconnect_attempts: 10
  device:
    enabled: true
    websocket_path: "/xiaozhi/mcp/"
    max_connections_per_device: 5

# Có bật lời chào khởi động hay không
enable_greeting: true
```
