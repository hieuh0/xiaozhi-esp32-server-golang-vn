
#### Kết quả kiểm tra độ trễ

Có thể phản hồi trong vòng 1-1.3 giây, nếu dùng mô hình nhỏ hơn có thể nhanh hơn

asr: funasr
llm: Alibaba Cloud API qwen2.5-72b-instruct
tts: cosyvoice 

```
time="2025-05-22 19:33:09.940" level=debug msg="từ khi nhận xong âm thanh asr->llm->tts khung đầu tiên tổng thời gian: 1394 ms" caller="client.go:428"
time="2025-05-22 19:33:33.458" level=debug msg="từ khi nhận xong âm thanh asr->llm->tts khung đầu tiên tổng thời gian: 1237 ms" caller="client.go:428"
time="2025-05-22 19:33:52.596" level=debug msg="từ khi nhận xong âm thanh asr->llm->tts khung đầu tiên tổng thời gian: 1190 ms" caller="client.go:428"
time="2025-05-22 19:34:12.272" level=debug msg="từ khi nhận xong âm thanh asr->llm->tts khung đầu tiên tổng thời gian: 1361 ms" caller="client.go:428"
time="2025-05-22 19:34:31.598" level=debug msg="từ khi nhận xong âm thanh asr->llm->tts khung đầu tiên tổng thời gian: 1347 ms" caller="client.go:428"
time="2025-05-22 19:35:00.281" level=debug msg="từ khi nhận xong âm thanh asr->llm->tts khung đầu tiên tổng thời gian: 1194 ms" caller="client.go:428"
time="2025-05-22 19:35:24.418" level=debug msg="từ khi nhận xong âm thanh asr->llm->tts khung đầu tiên tổng thời gian: 975 ms" caller="client.go:428"
time="2025-05-22 19:35:49.868" level=debug msg="từ khi nhận xong âm thanh asr->llm->tts khung đầu tiên tổng thời gian: 1150 ms" caller="client.go:428"
```

---

## Kiểm tra bảng quản trị

Gói khởi động một chạm và triển khai Docker đều tích hợp sẵn giao diện quản trị Web, cung cấp giao diện kiểm tra trực quan.

Hỗ trợ các loại kiểm tra sau:

| Loại kiểm tra | Mô tả |
|---------|------|
| VAD | Kiểm tra kết nối và thời gian phản hồi của nhận diện hoạt động giọng nói |
| ASR | Kiểm tra kết nối và độ trễ gói đầu tiên của nhận dạng giọng nói |
| LLM | Kiểm tra kết nối và độ trễ gói đầu tiên của suy luận mô hình ngôn ngữ lớn |
| TTS | Kiểm tra kết nối và độ trễ gói đầu tiên của tổng hợp giọng nói |
| OTA | Kiểm tra kết nối MQTT/UDP |

Cách sử dụng chi tiết vui lòng tham khảo: **[Hướng dẫn sử dụng bảng quản trị →](manager_console_guide.md)**