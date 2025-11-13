package yxyClient

import (
	"fmt"
	"strings"
	"time"
	"yxy-go/internal/consts"

	"github.com/google/uuid"
)

func GenYxyDeviceID(deviceID string) string {
	const prefix = "ym-"

	if deviceID == "" {
		deviceID = uuid.New().String()
	}

	return prefix + strings.ReplaceAll(deviceID, "-", "")
}

func GetYxyBaseReqParam(deviceID string) (baseReq map[string]interface{}, baseHeaders map[string]string) {
	deviceID = GenYxyDeviceID(deviceID)
	baseReq = map[string]interface{}{
		"appVersion": consts.APP_VERSION,
		"deviceId":   deviceID,
		"platform":   "YUNMA_APP",
		"schoolCode": consts.SCHOOL_CODE,
		"nt":         time.Now().UnixMilli(),
	}
	baseHeaders = map[string]string{
		"User-Agent": fmt.Sprintf("Mozilla/5.0 (Linux; Android 12; Android for arm64; wv) "+
			"AppleWebKit/537.36 (KHTML, like Gecko) "+
			"Version/4.0 Chrome/126.0.6478.186 Mobile Safari/537.36 "+
			"ZJYXYwebviewbroswer ZJYXYAndroid "+
			"tourCustomer/yunmaapp.NET/%v/%v", consts.APP_ALL_VERSION, deviceID),
	}
	return baseReq, baseHeaders
}

type YxyBusErrorResp struct {
	Detail struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
	} `json:"detail"`
}
