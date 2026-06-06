package websocket

/*
{
    "version": 2,
    "language": "zh-CN",
    "flash_size": 16777216,
    "minimum_free_heap_size": 8318916,
    "mac_address": "28:0A:C6:1D:3B:E8",
    "uuid": "550e8600-e29b-61d6-a716-666655660000",
    "chip_model_name": "ESP32-S3",
    "chip_info": {
        "model": 9,
        "cores": 2,
        "revision": 2,
        "features": 18
    },
    "application": {
        "name": "xiaozhi",
        "version": "0.9.9",
        "compile_time": "Jan 22 2025T20:40:23Z",
        "idf_version": "v5.3.2-dirty",
        "elf_sha256": "22986216df095587c42f8aeb06b239781c68ad8df80321e260556da7fcf5f522"
    },
    "partition_table": [{
        "label": "nvs",
        "type": 1,
        "subtype": 2,
        "address": 36864,
        "size": 16384
    },  {
        "label": "otadata",
        "type": 1,
        "subtype": 0,
        "address": 53248,
        "size": 8192
    },  {
        "label": "phy_init",
        "type": 1,
        "subtype": 1,
        "address": 61440,
        "size": 4096
    },  {
        "label": "model",
        "type": 1,
        "subtype": 130,
        "address": 65536,
        "size": 983040
    },  {
        "label": "storage",
        "type": 1,
        "subtype": 130,
        "address": 1048576,
        "size": 1048576
    },  {
        "label": "factory",
        "type": 0,
        "subtype": 0,
        "address": 2097152,
        "size": 4194304
    },  {
        "label": "ota_0",
        "type": 0,
        "subtype": 16,
        "address": 6291456,
        "size": 4194304
    },  {
        "label": "ota_1",
        "type": 0,
        "subtype": 17,
        "address": 10485760,
        "size": 4194304
    }],
    "ota": {
        "label": "factory"
    },
    "board": {
        "type": "esp-box-3",
        "ssid": "MyWiFiNetwork",
        "rssi": -65,
        "channel": 6,
        "ip": "192.168.1.100",
        "mac": "28:0A:C6:1D:3B:E8"
    }
}
*/
// Header contains: Device-Id: 02:4A:7D:E3:89:BF, Client-Id: e3b0c442-98fc-4e1a-8c3d-6a5b6a5b6a5b
type OtaRequest struct {
	Version             int         `json:"version"`
	Language            string      `json:"language"`
	FlashSize           int         `json:"flash_size"`
	MinimumFreeHeapSize int         `json:"minimum_free_heap_size"`
	MacAddress          string      `json:"mac_address"`
	UUID                string      `json:"uuid"`
	ChipModelName       string      `json:"chip_model_name"`
	ChipInfo            ChipInfo    `json:"chip_info"`
	Application         Application `json:"application"`
	PartitionTable      []Partition `json:"partition_table"`
	Ota                 Ota         `json:"ota"`
	Board               Board       `json:"board"`
}

type ChipInfo struct {
	Model    int `json:"model"`
	Cores    int `json:"cores"`
	Revision int `json:"revision"`
	Features int `json:"features"`
}

type Application struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	CompileTime string `json:"compile_time"`
	IdfVersion  string `json:"idf_version"`
	ElfSha256   string `json:"elf_sha256"`
}

type Partition struct {
	Label   string `json:"label"`
	Type    int    `json:"type"`
	Subtype int    `json:"subtype"`
	Address int    `json:"address"`
	Size    int    `json:"size"`
}

type Ota struct {
	Label string `json:"label"`
}

type Board struct {
	Type    string `json:"type"`
	Ssid    string `json:"ssid"`
	Rssi    int    `json:"rssi"`
	Channel int    `json:"channel"`
	Ip      string `json:"ip"`
	Mac     string `json:"mac"`
}

/*
	{
	    "mqtt": {
	        "endpoint": "mqtt.xiaozhi.me",
	        "client_id": "GID_test@@@02_4A_7D_E3_89_BF@@@e3b0c442-98fc-4e1a-8c3d-6a5b6a5b6a5b",
	        "username": "eyJpcCI6IjEuMjAyLjE5My4xOTQifQ==",
	        "password": "Ru9zRLdD/4wrBYorxIyABtHe8EiA1hdZ4v34juJ2BUU=",
	        "publish_topic": "device-server",
	        "subscribe_topic": "null"
	    },
	    "server_time": {
	        "timestamp": 1745995478882,
	        "timezone_offset": 480
	    },
	    "firmware": {
	        "version": "0.9.9",
	        "url": ""
	    },
	    "activation": {
	        "code": "738133",
	        "message": "xiaozhi.me\n738133",
	        "challenge": "ee2af2f0-0ca0-45f2-8b8c-6f34edd62156"
	    }
	}
*/
// If already registered, activation will not be returned
type OtaResponse struct {
	Mqtt       *MqttInfo       `json:"mqtt,omitempty"`
	ServerTime ServerTimeInfo  `json:"server_time"`
	Firmware   FirmwareInfo    `json:"firmware"`
	Activation *ActivationInfo `json:"activation,omitempty"`
	Websocket  WebsocketInfo   `json:"websocket"`
}

type WebsocketInfo struct {
	Url   string `json:"url"`
	Token string `json:"token"`
}

type MqttInfo struct {
	Endpoint       string `json:"endpoint"`
	ClientId       string `json:"client_id"`
	Username       string `json:"username"`
	Password       string `json:"password"`
	PublishTopic   string `json:"publish_topic"`
	SubscribeTopic string `json:"subscribe_topic"`
}

type ServerTimeInfo struct {
	Timestamp      int64 `json:"timestamp"`
	TimezoneOffset int   `json:"timezone_offset"`
}

type FirmwareInfo struct {
	Version string `json:"version"`
	Url     string `json:"url"`
}

type ActivationInfo struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Challenge string `json:"challenge"`
	TimeoutMs int    `json:"timeout_ms"`
}
