package protocol

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

// RegisterDevice registers a virtual device and returns credentials.
func RegisterDevice(ctx context.Context, client *http.Client) (*Credentials, error) {
	cdid := uuid.New().String()
	openudid := generateOpenUDID()
	clientudid := uuid.New().String()

	body := buildRegisterBody(cdid, openudid, clientudid)
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal register body: %w", err)
	}

	params := buildRegisterParams(cdid)
	reqURL := RegisterURL + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(string(bodyJSON)))
	if err != nil {
		return nil, fmt.Errorf("create register request: %w", err)
	}
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("device register request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read register response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("device register HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		DeviceID    int64  `json:"device_id"`
		DeviceIDStr string `json:"device_id_str"`
		InstallID   int64  `json:"install_id"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse register response: %w", err)
	}

	deviceID := fmt.Sprintf("%d", result.DeviceID)
	if result.DeviceID == 0 && result.DeviceIDStr != "" && result.DeviceIDStr != "0" {
		deviceID = result.DeviceIDStr
	}
	if deviceID == "0" || deviceID == "" {
		return nil, fmt.Errorf("device register: no device_id returned")
	}

	return &Credentials{
		DeviceID:   deviceID,
		InstallID:  fmt.Sprintf("%d", result.InstallID),
		CDID:       cdid,
		OpenUDID:   openudid,
		ClientUDID: clientudid,
	}, nil
}

// FetchASRToken fetches the ASR token using the device credentials.
func FetchASRToken(ctx context.Context, client *http.Client, deviceID, cdid string) (string, error) {
	if cdid == "" {
		cdid = uuid.New().String()
	}

	bodyStr := "body=null"
	hash := md5.Sum([]byte(bodyStr))
	xSSStub := fmt.Sprintf("%X", hash)

	params := buildSettingsParams(deviceID, cdid)
	reqURL := SettingsURL + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(bodyStr))
	if err != nil {
		return "", fmt.Errorf("create settings request: %w", err)
	}
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("x-ss-stub", xSSStub)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("settings request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read settings response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("settings HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Data struct {
			Settings struct {
				ASRConfig struct {
					AppKey string `json:"app_key"`
				} `json:"asr_config"`
			} `json:"settings"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parse settings response: %w", err)
	}

	token := result.Data.Settings.ASRConfig.AppKey
	if token == "" {
		return "", fmt.Errorf("settings: no asr_config.app_key in response")
	}

	return token, nil
}

func generateOpenUDID() string {
	u := uuid.New()
	b := u[:]
	return fmt.Sprintf("%x", b[:8])
}

func nowMillis() int64 {
	return time.Now().UnixMilli()
}

func buildRegisterBody(cdid, openudid, clientudid string) map[string]any {
	return map[string]any{
		"magic_tag": "ss_app_log",
		"header": map[string]any{
			"device_id":             0,
			"install_id":            0,
			"aid":                   AID,
			"app_name":              "oime",
			"version_code":          100102018,
			"version_name":          "1.1.2",
			"manifest_version_code": 100102018,
			"update_version_code":   100102018,
			"channel":               "official",
			"package":               "com.bytedance.android.doubaoime",
			"device_platform":       "android",
			"os":                    "android",
			"os_api":                "34",
			"os_version":            "16",
			"device_type":           "Pixel 7 Pro",
			"device_brand":          "google",
			"device_model":          "Pixel 7 Pro",
			"resolution":            "1080*2400",
			"dpi":                   "420",
			"language":              "zh",
			"timezone":              8,
			"access":                "wifi",
			"rom":                   "UP1A.231005.007",
			"rom_version":           "UP1A.231005.007",
			"region":                "CN",
			"tz_name":               "Asia/Shanghai",
			"tz_offset":             28800,
			"sim_region":            "cn",
			"carrier_region":        "cn",
			"cpu_abi":               "arm64-v8a",
			"build_serial":          "unknown",
			"not_request_sender":    0,
			"sig_hash":              "",
			"google_aid":            "",
			"mc":                    "",
			"serial_number":         "",
			"openudid":              openudid,
			"clientudid":            clientudid,
			"cdid":                  cdid,
		},
		"_gen_time": nowMillis(),
	}
}

func buildRegisterParams(cdid string) url.Values {
	p := url.Values{}
	p.Set("device_platform", "android")
	p.Set("os", "android")
	p.Set("ssmix", "a")
	p.Set("_rticket", fmt.Sprintf("%d", nowMillis()))
	p.Set("cdid", cdid)
	p.Set("channel", "official")
	p.Set("aid", fmt.Sprintf("%d", AID))
	p.Set("app_name", "oime")
	p.Set("version_code", "100102018")
	p.Set("version_name", "1.1.2")
	p.Set("manifest_version_code", "100102018")
	p.Set("update_version_code", "100102018")
	p.Set("resolution", "1080*2400")
	p.Set("dpi", "420")
	p.Set("device_type", "Pixel 7 Pro")
	p.Set("device_brand", "google")
	p.Set("language", "zh")
	p.Set("os_api", "34")
	p.Set("os_version", "16")
	p.Set("ac", "wifi")
	return p
}

func buildSettingsParams(deviceID, cdid string) url.Values {
	p := url.Values{}
	p.Set("device_platform", "android")
	p.Set("os", "android")
	p.Set("ssmix", "a")
	p.Set("_rticket", fmt.Sprintf("%d", nowMillis()))
	p.Set("cdid", cdid)
	p.Set("channel", "official")
	p.Set("aid", fmt.Sprintf("%d", AID))
	p.Set("app_name", "oime")
	p.Set("version_code", "100102018")
	p.Set("version_name", "1.1.2")
	p.Set("device_id", deviceID)
	return p
}
