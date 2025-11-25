package auth

import (
	"os"
	"testing"
)

var uid = os.Getenv("YxyUid")

func TestFetchAuthToken(t *testing.T) {
	if uid == "" {
		t.Skip("YxyUid 未设置，跳过此测试")
	}
	bm := BusAuthManager{}
	token, err := bm.FetchAuthToken(uid)
	if err != nil {
		t.Error(err)
		return
	}
	if token == "" {
		t.Error("token 获取失败")
		return
	}
	t.Log(token)
}
