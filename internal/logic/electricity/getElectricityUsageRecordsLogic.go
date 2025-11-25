package electricity

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"yxy-go/internal/manager/auth"

	"yxy-go/internal/consts"
	"yxy-go/internal/svc"
	"yxy-go/internal/types"
	"yxy-go/internal/utils/yxyClient"
	"yxy-go/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetElectricityUsageRecordsLogic struct {
	logx.Logger
	ctx        context.Context
	svcCtx     *svc.ServiceContext
	authManger *auth.ElectricityAuthManager
}

func NewGetElectricityUsageRecordsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetElectricityUsageRecordsLogic {
	return &GetElectricityUsageRecordsLogic{
		Logger:     logx.WithContext(ctx),
		ctx:        ctx,
		svcCtx:     svcCtx,
		authManger: auth.NewElectricityAuthManager(ctx, svcCtx),
	}
}

type GetElectricityZhpfUsageRecords struct {
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message"`
	Rows       []struct {
		// Roomdm   string `json:"roomdm"`
		Datetime string `json:"datetime"`
		Used     string `json:"used"`
	} `json:"rows"`
	// Total   int  `json:"total"`
	Success bool `json:"success"`
}

type GetElectricityMgsUsageRecords struct {
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message"`
	Rows       []struct {
		DateTime string `json:"dateTime"`
		DayUsage string `json:"dayUsage"`
	} `json:"rows"`
	// Total   int  `json:"total"`
	Success bool `json:"success"`
}

func (l *GetElectricityUsageRecordsLogic) fetchElectricityUsageRecords(req *types.GetElectricityUsageRecordsReq, token string) (resp *types.GetElectricityUsageRecordsResp, err error) {
	parts := strings.Split(req.RoomStrConcat, "#")
	requiredParts := 4
	if req.Campus == "zhpf" {
		requiredParts = 5
	}
	if len(parts) < requiredParts {
		return nil, xerr.WithCode(xerr.ErrParam, fmt.Sprintf("Param room_str_concat error: %v", req.RoomStrConcat))
	}
	for _, part := range parts {
		if part == "" || !regexp.MustCompile(`^\d+$`).MatchString(part) {
			return nil, xerr.WithCode(xerr.ErrParam, fmt.Sprintf("Invalid part in room_str_concat: %v", req.RoomStrConcat))
		}
	}

	yxyReq := map[string]interface{}{
		"areaId":       parts[0],
		"buildingCode": parts[1],
		"floorCode":    parts[2],
		"roomCode":     parts[3],
		"platform":     "YUNMA_APP",
	}

	_, yxyHeaders := yxyClient.GetYxyBaseReqParam("")
	yxyHeaders["Cookie"] = "shiroJID=" + token
	var records []types.ElectricityUsageRecord
	switch req.Campus {
	case "zhpf":
		yxyReq["mdtype"] = parts[4]

		var yxyZhpfResp GetElectricityZhpfUsageRecords
		r, err := yxyClient.HttpSendPost(consts.GET_ELECTRICITY_ZHPF_USAGE_RECORDS_URL, yxyReq, yxyHeaders, &yxyZhpfResp)
		if err != nil {
			return nil, err
		}

		if yxyZhpfResp.StatusCode != 0 {
			errCode := xerr.ErrUnknown
			if yxyZhpfResp.Message == "请重新登录" {
				errCode = xerr.ErrElectricityTokenInvalid
			} else if yxyZhpfResp.Message == "系统维护中，请稍后再试！" || yxyZhpfResp.Message == "对不起，数据不存在！" || yxyZhpfResp.Message == "常工接口返回异常First Element must contain the local name, Envelope , but found script" {
				errCode = xerr.ErrRoomInfoWrongOrCampusMismatch
			}
			return nil, xerr.WithCode(errCode, fmt.Sprintf("yxy response: %v", r))
		}

		for _, row := range yxyZhpfResp.Rows {
			record := types.ElectricityUsageRecord{
				Usage:    row.Used + "度",
				Datetime: row.Datetime,
			}
			records = append(records, record)
		}

	case "mgs":
		yxyReq["pageNo"] = 1
		yxyReq["pageSize"] = 30

		var yxyMgsResp GetElectricityMgsUsageRecords
		r, err := yxyClient.HttpSendPost(consts.GET_ELECTRICITY_MGS_USAGE_RECORDS_URL, yxyReq, yxyHeaders, &yxyMgsResp)
		if err != nil {
			return nil, err
		}

		if yxyMgsResp.StatusCode != 0 {
			errCode := xerr.ErrUnknown
			if yxyMgsResp.Message == "请重新登录" {
				errCode = xerr.ErrElectricityTokenInvalid
			} else if yxyMgsResp.Message == "校区不存在" || yxyMgsResp.Message == "暂不支持" {
				errCode = xerr.ErrRoomInfoWrongOrCampusMismatch
			}
			return nil, xerr.WithCode(errCode, fmt.Sprintf("yxy response: %v", r))
		}

		for _, row := range yxyMgsResp.Rows {
			record := types.ElectricityUsageRecord{
				Usage:    row.DayUsage,
				Datetime: row.DateTime,
			}
			records = append(records, record)
		}
	}

	return &types.GetElectricityUsageRecordsResp{
		List: records,
	}, nil
}
func (l *GetElectricityUsageRecordsLogic) GetElectricityUsageRecords(req *types.GetElectricityUsageRecordsReq) (resp *types.GetElectricityUsageRecordsResp, err error) {
	result, err := l.authManger.WithAuthToken(req.Uid, func(token string) (any, error) {
		return l.fetchElectricityUsageRecords(req, token)
	})
	if err != nil {
		return nil, err
	}
	return result.(*types.GetElectricityUsageRecordsResp), nil
}
