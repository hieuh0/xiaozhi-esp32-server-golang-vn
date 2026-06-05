# Cấu hình xác thực MQTT cho giao diện OTA

## Tổng quan

Giao diện OTA hiện hỗ trợ cơ chế xác thực mật khẩu MQTT dựa trên chữ ký HMAC-SHA256, cung cấp phương thức xác thực an toàn hơn. Máy chủ MQTT cũng hỗ trợ logic xác minh tương ứng.

## Cấu trúc cấu hình

### Tệp cấu hình (config/config.yaml)

```yaml
mqtt_server:
  signature_key: "your_ota_signature_key_here"
ota:
  signature_key: "your_ota_signature_key_here"
  test:
    websocket:
      url: "ws://192.168.208.214:8989/xiaozhi/v1/"
    mqtt:
      enable: false
      endpoint: "192.168.208.214"
  external:
    websocket:
      url: "wss://www.tb263.cn:55555/go_ws/xiaozhi/v1/"
    mqtt:
      enable: false
      endpoint: "www.youdomain.cn"
```

### Giải thích cấu hình

- `mqtt_server.signature_key`: Khóa chữ ký MQTT, dùng để tạo chữ ký mật khẩu MQTT
- `ota.signature_key`: Khóa được dùng khi OTA gửi xuống mật khẩu MQTT, cần khớp với `mqtt_server.signature_key`
- `ota.test`: Cấu hình môi trường kiểm thử (dùng cho IP mạng nội bộ)
- `ota.external`: Cấu hình môi trường bên ngoài (dùng cho IP mạng công cộng)

### Tích hợp với xiaozhi-mqtt-gateway

Hệ thống này được sử dụng kết hợp với dự án chính thức [xiaozhi-mqtt-gateway](https://github.com/78/xiaozhi-mqtt-gateway) để thực hiện quy trình xác thực MQTT đầy đủ:

1. **Yêu cầu nhất quán cấu hình**: `ota.signature_key` phải hoàn toàn giống với khóa chữ ký trong dự án xiaozhi-mqtt-gateway
2. **Quy trình xác thực**:
   - xiaozhi-mqtt-gateway chịu trách nhiệm tạo thông tin xác thực kết nối MQTT
   - Hệ thống này chịu trách nhiệm xác minh thông tin xác thực kết nối MQTT
   - Cả hai bên sử dụng cùng thuật toán chữ ký và khóa để đảm bảo xác thực thành công
3. **Khuyến nghị triển khai**: Nên triển khai cả hai dự án trong cùng một môi trường mạng, đảm bảo cấu hình được cập nhật đồng bộ

## Các hàm tiện ích

### 1. Tạo chữ ký mật khẩu

```go
// Tạo chữ ký mật khẩu HMAC-SHA256
password := util.GeneratePasswordSignature(data, key)
```

### 2. Tạo thông tin xác thực MQTT

```go
// Tạo thông tin xác thực kết nối MQTT đầy đủ
credentials, err := util.GenerateMqttCredentials(deviceId, clientId, ip, signatureKey)
if err != nil {
    // Xử lý lỗi
}
// credentials bao gồm: ClientId, Username, Password
```

### 3. Xác minh thông tin xác thực MQTT

```go
// Xác minh thông tin xác thực kết nối MQTT
credentialInfo, err := util.ValidateMqttCredentials(clientId, username, password, signatureKey)
if err != nil {
    // Xác minh thất bại
}
// credentialInfo bao gồm: GroupId, MacAddress, UUID, UserData
```

## Logic xác thực MQTT

### 1. Định dạng Client ID

```
GID_test@@@{deviceId}@@@{clientId}
```

Ví dụ:
```
GID_test@@@02_4A_7D_E3_89_BF@@@e3b0c442-98fc-4e1a-8c3d-6a5b6a5b6a5b
```

### 2. Định dạng Username

JSON được mã hóa Base64, chứa thông tin IP của client:

```yaml
ip: "1.202.193.194"
```

Sau khi mã hóa Base64:
```
eyJpcCI6IjEuMjAyLjE5My4xOTQifQ==
```

### 3. Tạo Password

Sử dụng thuật toán HMAC-SHA256 để tạo chữ ký mật khẩu:

```go
signatureData := clientId + "|" + username
password := HMAC-SHA256(signatureData, signature_key)
```

### 4. Logic xác minh

Khi xác minh phía client cần:

1. Phân tích clientId, trích xuất groupId, macAddress, uuid
2. Giải mã username, lấy thông tin IP
3. Xác minh mật khẩu bằng cùng khóa và thuật toán chữ ký

## Xác thực máy chủ MQTT

### Quy trình xác thực

1. **Xác minh quản trị viên cấp cao**
   - Tên đăng nhập: `admin` (có thể cấu hình)
   - Mật khẩu: `shijingbo!@#` (có thể cấu hình)

2. **Xác minh người dùng thông thường**
   - Ưu tiên sử dụng xác minh chữ ký HMAC-SHA256
   - Nếu chưa cấu hình khóa chữ ký, fallback về phương thức xác minh AES

### Triển khai hook xác thực

```go
func (h *AuthHook) OnConnectAuthenticate(cl *mqttServer.Client, pk packets.Packet) bool {
    username := string(pk.Connect.Username)
    password := string(pk.Connect.Password)
    clientId := string(pk.Connect.ClientIdentifier)

    // Kiểm tra quản trị viên cấp cao
    if username == adminUsername && password == adminPassword {
        return true
    }

    // Kiểm tra người dùng thông thường - sử dụng logic xác minh chữ ký mới
    signatureKey := viper.GetString("mqtt_server.signature_key")
    if signatureKey != "" {
        credentialInfo, err := util.ValidateMqttCredentials(clientId, username, password, signatureKey)
        if err != nil {
            return false
        }
        return true
    }

    // Fallback về logic xác minh AES
    return h.validateWithAes(username, password)
}
```

## Khả năng tương thích

- Nếu chưa cấu hình `mqtt_server.signature_key`, hệ thống sẽ fallback về phương thức tạo mật khẩu SHA256/AES ban đầu
- Duy trì khả năng tương thích ngược, không ảnh hưởng đến các chức năng hiện có
- Máy chủ MQTT hỗ trợ nhiều phương thức xác thực cùng tồn tại

## Khuyến nghị bảo mật

1. Sử dụng chuỗi ngẫu nhiên mạnh làm khóa chữ ký
2. Định kỳ xoay vòng khóa chữ ký
3. Sử dụng kết nối HTTPS/WSS trong môi trường sản xuất
4. Giám sát các lần đăng nhập bất thường
5. Bật ghi nhật ký, theo dõi các lần xác thực thành công/thất bại
6. **Đảm bảo khóa chữ ký của xiaozhi-mqtt-gateway và hệ thống này được cập nhật đồng bộ**

## Cấu trúc dữ liệu

### MqttCredentials
```go
type MqttCredentials struct {
    ClientId string `json:"client_id"`
    Username string `json:"username"`
    Password string `json:"password"`
}
```

### MqttCredentialInfo
```go
type MqttCredentialInfo struct {
    GroupId    string                 `json:"groupId"`
    MacAddress string                 `json:"macAddress"`
    UUID       string                 `json:"uuid"`
    UserData   map[string]interface{} `json:"userData"`
}
``` 

# Hướng dẫn sử dụng xiaozhi-mqtt-gateway chính thức

Hệ thống này có thể được sử dụng kết hợp với dự án chính thức [xiaozhi-mqtt-gateway](https://github.com/78/xiaozhi-mqtt-gateway).

Chỉ cần tên đăng nhập và mật khẩu MQTT trong giao diện OTA vượt qua xác thực của xiaozhi-mqtt-gateway. Để đảm bảo xác thực MQTT hoạt động bình thường, **cấu hình `ota.signature_key` phải giữ nhất quán với khóa chữ ký trong xiaozhi-mqtt-gateway**.

Cấu hình như sau:
1. Không bật mqtt server (sử dụng xiaozhi-mqtt-gateway)
2. Cấu hình `ota.signature_key` phải giữ nhất quán với khóa chữ ký trong xiaozhi-mqtt-gateway
3. Cấu hình backend websocket của xiaozhi-mqtt-gateway trỏ đến địa chỉ dự án này

```yaml
mqtt_server:
  enable: false
ota:
  signature_key: "your_ota_signature_key_here"
  test:  # Kết quả trả về cho môi trường kiểm thử nội bộ
    websocket:
      url: "ws://192.168.208.214:8989/xiaozhi/v1/"
    mqtt:
      enable: true
      endpoint: "192.168.208.214:1883"  # Địa chỉ mqtt server trong xiaozhi-mqtt-gateway
  external:  # Kết quả trả về cho mạng công cộng
    websocket:
      url: "wss://www.tb263.cn:55555/go_ws/xiaozhi/v1/"
    mqtt:
      enable: true
      endpoint: "mqtt.youdomain.com:1883"  # Địa chỉ mqtt server trong xiaozhi-mqtt-gateway
```
