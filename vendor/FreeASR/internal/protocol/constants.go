package protocol

const (
	// RegisterURL is the device registration endpoint.
	RegisterURL = "https://log.snssdk.com/service/2/device_register/"

	// SettingsURL is the endpoint for fetching the ASR token.
	SettingsURL = "https://is.snssdk.com/service/settings/v3/"

	// WebSocketURL is the ASR WebSocket endpoint.
	WebSocketURL = "wss://frontier-audio-ime-ws.doubao.com/ocean/api/v1/ws"

	// AID is the DoubaoIME application ID.
	AID = 401734

	// UserAgent is the fake Android client user agent.
	UserAgent = "com.bytedance.android.doubaoime/100102018 (Linux; U; Android 16; en_US; Pixel 7 Pro; Build/BP2A.250605.031.A2; Cronet/TTNetVersion:94cf429a 2025-11-17 QuicVersion:1f89f732 2025-05-08)"

	// TokenRefreshInterval is 12 hours in milliseconds.
	TokenRefreshInterval = 12 * 60 * 60 * 1000
)

// AppConfig contains the DoubaoIME application configuration.
var AppConfig = map[string]any{
	"aid":                   AID,
	"app_name":              "oime",
	"version_code":          100102018,
	"version_name":          "1.1.2",
	"manifest_version_code": 100102018,
	"update_version_code":   100102018,
	"channel":               "official",
	"package":               "com.bytedance.android.doubaoime",
}

// DeviceConfig contains the fake Android device configuration.
var DeviceConfig = map[string]any{
	"device_platform": "android",
	"os":              "android",
	"os_api":          "34",
	"os_version":      "16",
	"device_type":     "Pixel 7 Pro",
	"device_brand":    "google",
	"device_model":    "Pixel 7 Pro",
	"resolution":      "1080*2400",
	"dpi":             "420",
	"language":        "zh",
	"timezone":        8,
	"access":          "wifi",
	"rom":             "UP1A.231005.007",
	"rom_version":     "UP1A.231005.007",
}
