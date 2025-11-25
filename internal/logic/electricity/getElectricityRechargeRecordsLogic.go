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

type GetElectricityRechargeRecordsLogic struct {
	logx.Logger
	ctx        context.Context
	svcCtx     *svc.ServiceContext
	authManger *auth.ElectricityAuthManager
}

func NewGetElectricityRechargeRecordsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetElectricityRechargeRecordsLogic {
	return &GetElectricityRechargeRecordsLogic{
		Logger:     logx.WithContext(ctx),
		ctx:        ctx,
		svcCtx:     svcCtx,
		authManger: auth.NewElectricityAuthManager(ctx, svcCtx),
	}
}

type GetElectricityZhpfRechargeRecords struct {
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message"`
	Rows       []struct {
		// Roomdm      string `json:"roomdm"`
		Datetime string `json:"datetime"`
		// Buytype     string `json:"buytype"`
		// Buyusingtpe string `json:"buyusingtpe"`
		Money string `json:"money"`
		// Issend      string `json:"issend"`
	} `json:"rows"`
	// Total   int  `json:"total"`
	Success bool `json:"success"`
}

type GetElectricityMgsRechargeRecords struct {
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message"`
	Rows       []struct {
		DateTime string `json:"dateTime"`
		// TypeName string `json:"typeName"`
		Amount string `json:"amount"`
	} `json:"rows"`
	// Total   int  `json:"total"`
	Success bool `json:"success"`
}

func (l *GetElectricityRechargeRecordsLogic) fetchElectricityRechargeRecords(req *types.GetElectricityRechargeRecordsReq, token string) (resp *types.GetElectricityRechargeRecordsResp, err error) {
	parts := strings.Split(req.RoomStrConcat, "#")
	if len(parts) < 4 {
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

	var records []types.ElectricityRechargeRecord
	switch req.Campus {
	case "zhpf":
		yxyReq["subType"] = "100304"
		yxyReq["currentPage"] = req.Page

		var yxyZhpfResp GetElectricityZhpfRechargeRecords
		r, err := yxyClient.HttpSendPost(consts.GET_ELECTRICITY_ZHPF_RECHARGE_RECORDS_URL, yxyReq, yxyHeaders, &yxyZhpfResp)
		if err != nil {
			return nil, err
		}

		if yxyZhpfResp.StatusCode != 0 {
			errCode := xerr.ErrUnknown
			if yxyZhpfResp.Message == "请重新登录" {
				errCode = xerr.ErrElectricityTokenInvalid
			} else if yxyZhpfResp.Message == "系统维护中，请稍后再试！" || yxyZhpfResp.Message == "常工接口返回异常First Element must contain the local name, Envelope , but found script" {
				errCode = xerr.ErrRoomInfoWrongOrCampusMismatch
			}
			return nil, xerr.WithCode(errCode, fmt.Sprintf("yxy response: %v", r))
		}

		for _, row := range yxyZhpfResp.Rows {
			record := types.ElectricityRechargeRecord{
				Money:    row.Money + "元",
				Datetime: row.Datetime,
			}
			records = append(records, record)
		}

	case "mgs":
		yxyReq["pageNo"] = req.Page
		yxyReq["pageSize"] = 30

		var yxyMgsResp GetElectricityMgsRechargeRecords
		r, err := yxyClient.HttpSendPost(consts.GET_ELECTRICITY_MGS_RECHARGE_RECORDS_URL, yxyReq, yxyHeaders, &yxyMgsResp)
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
			record := types.ElectricityRechargeRecord{
				Money:    row.Amount,
				Datetime: row.DateTime,
			}
			records = append(records, record)
		}
	}

	return &types.GetElectricityRechargeRecordsResp{
		List: records,
	}, nil
}

func (l *GetElectricityRechargeRecordsLogic) GetElectricityRechargeRecords(req *types.GetElectricityRechargeRecordsReq) (resp *types.GetElectricityRechargeRecordsResp, err error) {
	result, err := l.authManger.WithAuthToken(req.Uid, func(token string) (any, error) {
		return l.fetchElectricityRechargeRecords(req, token)
	})
	if err != nil {
		return nil, err
	}
	return result.(*types.GetElectricityRechargeRecordsResp), nil
}
