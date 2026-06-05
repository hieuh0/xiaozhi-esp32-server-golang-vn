
# Môi trường chạy

#### I. Triển khai funasr

Tham khảo [tài liệu triển khai funasr bằng docker](https://github.com/modelscope/FunASR/blob/main/runtime/docs/SDK_advanced_guide_online_zh.md)

#### II. Clone mã nguồn
>git clone 'https://github.com/hackers365/xiaozhi-esp32-server-golang'

#### III. Cấu hình config/config.yaml, xem chi tiết tại [hướng dẫn cấu hình config](config.md)

Các mục cần chỉnh sửa chính như sau:
```yaml
# 1. asr nhận dạng giọng nói
asr:
  provider: "funasr"
  funasr:
    host: "127.0.0.1"      # ip của dịch vụ funasr websocket đã triển khai
    port: "10096"          # port của funasr websocket đã triển khai
    mode: "offline"        # chế độ, dùng offline là được
    # ...

# 2. tts
tts:
  provider: "xiaozhi"      # loại tts sử dụng, khuyến nghị doubao_ws, hoặc có thể chọn edge miễn phí
  doubao_ws:
    appid: "6886011847"                         # appid của bạn
    access_token: "access_token"                # access token của bạn
    cluster: "volcano_tts"
    voice: "zh_female_wanwanxiaohe_moon_bigtts" # giọng đọc, mặc định là Wanwan Xiaohe
    ws_host: "openspeech.bytedance.com"
    use_stream: true
  edge:
    voice: "zh-CN-XiaoxiaoNeural"
    rate: "+0%"
    volume: "+0%"
    pitch: "+0Hz"
    connect_timeout: 10
    receive_timeout: 60
  # ....

# 3. llm mô hình ngôn ngữ lớn
llm:
  provider: "deepseek"                        # nhà cung cấp, tương ứng với key bên dưới
  deepseek:
    type: "openai"                            # loại giao diện tương thích phía server
    model_name: "Pro/deepseek-ai/DeepSeek-V3" # tên mô hình
    api_key: "api_key"                        # api key
    base_url: "https://api.siliconflow.cn/v1" # endpoint dịch vụ, mặc định là SiliconFlow
    max_tokens: 500
  # ...

```

#### IV. Khởi động docker
Tại thư mục gốc của dự án, khởi động docker và mount thư mục config cùng cổng (http/websocket:8989, các cổng khác ánh xạ theo nhu cầu)

```
docker run -itd --name xiaozhi_server -v $(pwd)/config:/workspace/config -p 8989:8989 hackers365/xiaozhi_server:latest

Nếu không kết nối được từ trong nước, sử dụng nguồn sau

docker run -itd --name xiaozhi_server -v $(pwd)/config:/workspace/config -p 8989:8989 docker.jsdelivr.fyi/hackers365/xiaozhi_server:latest
```

**Ghi chú hỗ trợ ten_vad:**
- Docker image đã tự động bao gồm file thư viện ten_vad, không cần mount thêm
- Nếu sử dụng ten_vad làm nhà cung cấp VAD, chỉ cần đặt `vad.provider: "ten_vad"` trong file cấu hình

Bây giờ bạn có thể kết nối tới 
>ws://ip-máy-chủ:8989/xiaozhi/v1/ 

để bắt đầu trò chuyện


# Môi trường phát triển
```
docker run -itd --name xiaozhi_server_golang -v $(pwd):/workspace/ -p 8989:8989 hackers365/xiaozhi_golang:0.1
Nếu không kết nối được từ trong nước, sử dụng nguồn sau
docker run -itd --name xiaozhi_server_golang -v $(pwd):/workspace/ -p 8989:8989 docker.jsdelivr.fyi/hackers365/xiaozhi_golang:0.1

go build -o xiaozhi_server cmd/server/*.go
```

**Ghi chú ten_vad trong môi trường phát triển:**
- Image môi trường phát triển đã bao gồm các dependency biên dịch và runtime của ten_vad
- Nếu cần sử dụng ten_vad trong môi trường phát triển, hãy đảm bảo thư mục `lib/ten-vad` tồn tại ở thư mục gốc của dự án
- Khi biên dịch, file header và file thư viện của ten_vad sẽ được sử dụng tự động
