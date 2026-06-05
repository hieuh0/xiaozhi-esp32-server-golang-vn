Sử dụng redis để lưu trữ cấu trúc dữ liệu cấu hình người dùng

#### I. Cấu hình
##### 1. Cấu trúc hget cấu hình toàn cục
xiaozhi:global:config

##### 2. Cấu hình người dùng có thể ghi đè cấu hình trong file cấu hình, cấu trúc hget
```
xiaozhi:userconfig:{deviceid}
    "llm": {
        "provider": "deepseek",         //tương ứng với key trong phần llm của file cấu hình
    },
    "tts": {
        "provider": "cosyvoice",        //tương ứng với key trong phần tts của file cấu hình
    }
```

#### II. Prompt
##### 1. Lấy/đặt prompt hệ thống
>xiaozhi:llm:system:{deviceid}

##### 2. Lưu trữ prompt phiên chat, cấu trúc sorted set
>xiaozhi:llm:{deviceid}
