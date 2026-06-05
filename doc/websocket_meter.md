### Kiểm tra tải (Stress Test)

```
root@hackers365-System-Product-Name:~# docker run -itd --name websocket_meter docker.jsdelivr.fyi/hackers365/xiaozhi_websocket_client                      
87311584e5fef592f32e0b7d7062d9053e956d5e0d50edb220370ff37d2293ac
root@hackers365-System-Product-Name:~# 
root@hackers365-System-Product-Name:~# docker exec -it websocket_meter /bin/bash                                                      
root@87311584e5fe:/workspace# 
root@87311584e5fe:/workspace# ./ws_multi  -h
Usage of ./ws_multi:
  -count int
        Số lượng client (default 10)
  -device string
        ID thiết bị
  -server string
        Địa chỉ máy chủ (default "ws://localhost:8989/xiaozhi/v1/")
  -text string
        Nội dung trò chuyện, nhiều câu phân cách bằng dấu phẩy sẽ được gửi lần lượt (default "你好")
root@87311584e5fe:/workspace# ./ws_multi -count 1 -server wss://joeyzhou.chat/ws/xiaozhi/v1/ -text "你好,在干什么,一起出去玩吧" 
Đang chạy client Xiaozhi
Máy chủ: wss://joeyzhou.chat/ws/xiaozhi/v1/
Số lượng client: 1
Nội dung gửi: 你好,在干什么,一起出去玩吧
2025-05-27 09:54:51.095 [info] [audio_utils.go:199] tts thời gian nhận khung đầu tiên từ cloud: 532 ms
2025-05-27 09:54:51.098 [info] [audio_utils.go:269] tts cloud->hoàn thành giải mã khung đầu tiên: 535 ms
2025-05-27 09:54:51.401 [info] [cosyvoice.go:306] tts thời gian xử lý: từ đầu vào đến khi nhận xong dữ liệu MP3: 838 ms
2025-05-27 09:54:51.748 [info] [audio_utils.go:199] tts thời gian nhận khung đầu tiên từ cloud: 344 ms
2025-05-27 09:54:51.752 [info] [audio_utils.go:269] tts cloud->hoàn thành giải mã khung đầu tiên: 347 ms
2025-05-27 09:54:51.901 [info] [cosyvoice.go:306] tts thời gian xử lý: từ đầu vào đến khi nhận xong dữ liệu MP3: 497 ms
2025-05-27 09:54:52.292 [info] [audio_utils.go:199] tts thời gian nhận khung đầu tiên từ cloud: 387 ms
2025-05-27 09:54:52.296 [info] [audio_utils.go:269] tts cloud->hoàn thành giải mã khung đầu tiên: 391 ms
2025-05-27 09:54:52.628 [info] [cosyvoice.go:306] tts thời gian xử lý: từ đầu vào đến khi nhận xong dữ liệu MP3: 723 ms
0 Client bắt đầu chạy
0 Client đã kết nối đến máy chủ: wss://joeyzhou.chat/ws/xiaozhi/v1/
Nhận được tin nhắn: {Type:hello Text: State: SessionID:cafd2800-1979-06d5-19cf-b8bf53bb55dc Transport:websocket AudioFormat:<nil>}
Đang gửi khung Opus: 20
Đang gửi khung Opus: 50
Đang gửi khung Opus: 59
```

#### Mô tả tổng quan
    1. Chương trình sẽ dựa vào văn bản người dùng nhập vào, gọi giao diện tts để tạo dữ liệu âm thanh, sau đó gửi lần lượt đến máy chủ
    2. Thống kê thời gian phản hồi bắt đầu từ khi gửi type: listen, state: stop và dừng lại khi nhận được khung âm thanh đầu tiên từ máy chủ

#### Mô tả tham số:
    -count: Số lượng kết nối đồng thời
    -device: Mặc định sẽ tạo deviceId ngẫu nhiên, nếu dùng tham số này để chỉ định thiết bị, -count phải là 1
    -server: Địa chỉ máy chủ websocket
    -text: Nội dung cần gửi, phân cách bằng dấu ",", gửi vòng lặp

#### Mô tả đầu ra
    Có thể chuyển hướng đầu ra vào file log, sau đó dùng tail -f xx.log | grep 'Thời gian phản hồi trung bình'
