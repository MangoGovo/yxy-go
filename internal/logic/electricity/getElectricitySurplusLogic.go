package electricity

import (
	"context"
	"fmt"
	"strings"
	"yxy-go/internal/consts"
	"yxy-go/internal/manager/auth"
	"yxy-go/internal/svc"
	"yxy-go/internal/types"
	"yxy-go/internal/utils/yxyClient"
	"yxy-go/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetElectricitySurplusLogic struct {
	logx.Logger
	ctx        context.Context
	svcCtx     *svc.ServiceContext
	authManger *auth.ElectricityAuthManager
}

func NewGetElectricitySurplusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetElectricitySurplusLogic {
	return &GetElectricitySurplusLogic{
		Logger:     logx.WithContext(ctx),
		ctx:        ctx,
		svcCtx:     svcCtx,
		authManger: auth.NewElectricityAuthManager(ctx, svcCtx),
	}
}

type QueryElectricityBindYxyResp struct {
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message"`
	Rows       []struct {
		// ID            string `json:"id"`
		// SchoolCode    string `json:"schoolCode"`
		// SchoolName    string `json:"schoolName"`
		// JobNo         string `json:"jobNo"`
		// UserID        string `json:"userId"` // 仅mgs有
		// UserName      string `json:"userName"`
		// BindTypeStr   string `json:"bindTypeStr"`
		// SourceStr     string `json:"sourceStr"` // 仅mgs有
		// Source        string `json:"source"`    // 仅mgs有
		AreaId string `json:"areaId"`
		// AreaCode      string `json:"areaCode"` // 仅mgs有
		// AreaName      string `json:"areaName"`
		BuildingCode string `json:"buildingCode"`
		// BuildingName  string `json:"buildingName"`
		FloorCode string `json:"floorCode"`
		// FloorName     string `json:"floorName"`
		RoomCode string `json:"roomCode"`
		// RoomName      string `json:"roomName"`
		// CreateTime    string `json:"createTime"`
		// IsAllowChange uint8  `json:"isAllowChange"` // 仅zhpf有
	} `json:"rows"`
	Total   int  `json:"total"`
	Success bool `json:"success"`
}

type GetElectricityZhpfSurplusYxyResp struct {
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message"`
	Data       struct {
		// SchoolCode      string `json:"schoolCode"`
		// AreaId          string `json:"areaId"`
		// BuildingCode    string `json:"buildingCode"`
		// FloorCode       string `json:"floorCode"`
		// RoomCode        string `json:"roomCode"`
		DisplayRoomName string `json:"displayRoomName"`
		// Remind          string `json:"remind"`
		SurplusList []struct {
			// Surplus       float64 `json:"surplus"`
			// Amount        float64 `json:"amount"`
			// Subsidy       float64 `json:"subsidy"`
			// SubsidyAmount float64 `json:"subsidyAmount"`
			// TotalSurplus  float64 `json:"totalSurplus"`
			Mdtype string `json:"mdtype"`
			// Mdname        string  `json:"mdname"`
			// RoomStatus    string  `json:"roomStatus"`
		} `json:"surplusList"`
		// TopUpTypeList []struct {
		// 	Mdname string `json:"mdname"`
		// 	Cztype string `json:"cztype"`
		// } `json:"topUpTypeList"`
		Soc float64 `json:"soc"`
		// TotalSocAmount  float64 `json:"totalSocAmount"`
		// IsAllowChange   uint8   `json:"isAllowChange"`
		// ShowType        uint8   `json:"showType"`
		// RecordShow      uint8   `json:"recordShow"`
		// Style           uint8   `json:"style"`
		// IsShowRemainder uint8   `json:"isShowRemainder"`
		// SurplusDetail   uint8   `json:"surplusDetail"`
	} `json:"data"`
	Success bool `json:"success"`
}

type GetElectricityMgsSurplusYxyResp struct {
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message"`
	Data       struct {
		Surplus float64 `json:"surplus"`
		// Amount          float64 `json:"amount"`
		// IsShowSurplus   uint8   `json:"isShowSurplus"`
		// IsShowMoney     uint8   `json:"isShowMoney"`
		// Remind          string  `json:"remind"`
		// System          uint8   `json:"system"`
		DisplayRoomName string `json:"displayRoomName"`
		// RecordTime      string  `json:"recordTime"`
		// FooterLink      uint8   `json:"footerLink"`
		// CanBuy          uint8   `json:"canBuy"`
	} `json:"data"`
	Success bool `json:"success"`
}

func (l *GetElectricitySurplusLogic) fetchElectricitySurplus(req *types.GetElectricitySurplusReq, token string) (resp *types.GetElectricitySurplusResp, err error) {
	var bindType string
	switch req.Campus {
	case "zhpf":
		bindType = "3"
	case "mgs":
		bindType = "1"
	}

	yxyReq := map[string]interface{}{
		"bindType": bindType,
		"platform": "YUNMA_APP",
	}

	_, yxyHeaders := yxyClient.GetYxyBaseReqParam("")
	yxyHeaders["Cookie"] = "shiroJID=" + token

	var yxyResp QueryElectricityBindYxyResp
	r, err := yxyClient.HttpSendPost(consts.QUERY_ELECTRICITY_BIND_URL, yxyReq, yxyHeaders, &yxyResp)
	if err != nil {
		return nil, err
	}

	if yxyResp.StatusCode != 0 {
		errCode := xerr.ErrUnknown
		if yxyResp.Message == "请重新登录" {
			errCode = xerr.ErrElectricityTokenInvalid
		} else if yxyResp.Message == "学校编码不能为空" {
			errCode = xerr.ErrNotBindCard
		}
		return nil, xerr.WithCode(errCode, fmt.Sprintf("yxy response: %v", r))
	}

	if yxyResp.Total == 0 {
		return nil, xerr.WithCode(xerr.ErrElectricityBindNotFound, fmt.Sprintf("No electricity binding information found for %v", req.Campus))
	}

	areaId := yxyResp.Rows[0].AreaId
	buildingCode := yxyResp.Rows[0].BuildingCode
	floorCode := yxyResp.Rows[0].FloorCode
	roomCode := yxyResp.Rows[0].RoomCode
	yxyReq = map[string]interface{}{
		"areaId":       areaId,
		"buildingCode": buildingCode,
		"floorCode":    floorCode,
		"roomCode":     roomCode,
		"platform":     "YUNMA_APP",
	}

	roomStrConcat := strings.Join([]string{areaId, buildingCode, floorCode, roomCode}, "#")
	switch req.Campus {
	case "zhpf":
		var yxyZhpfResp GetElectricityZhpfSurplusYxyResp
		r, err := yxyClient.HttpSendPost(consts.GET_ELECTRICITY_ZHPF_SURPLUS_URL, yxyReq, yxyHeaders, &yxyZhpfResp)
		if err != nil {
			return nil, err
		}

		if yxyZhpfResp.StatusCode != 0 {
			return nil, xerr.WithCode(xerr.ErrUnknown, fmt.Sprintf("yxy response: %v", r))
		}

		resp = &types.GetElectricitySurplusResp{
			DisplayRoomName: yxyZhpfResp.Data.DisplayRoomName,
			RoomStrConcat:   strings.Join([]string{roomStrConcat, yxyZhpfResp.Data.SurplusList[0].Mdtype}, "#"),
			Surplus:         yxyZhpfResp.Data.Soc,
		}

	case "mgs":
		var yxyMgsResp GetElectricityMgsSurplusYxyResp
		r, err := yxyClient.HttpSendPost(consts.GET_ELECTRICITY_MGS_SURPLUS_URL, yxyReq, yxyHeaders, &yxyMgsResp)
		if err != nil {
			return nil, err
		}

		if yxyMgsResp.StatusCode != 0 {
			return nil, xerr.WithCode(xerr.ErrUnknown, fmt.Sprintf("yxy response: %v", r))
		}

		resp = &types.GetElectricitySurplusResp{
			DisplayRoomName: yxyMgsResp.Data.DisplayRoomName,
			RoomStrConcat:   roomStrConcat,
			Surplus:         yxyMgsResp.Data.Surplus,
		}
	}

	return resp, nil
}

func (l *GetElectricitySurplusLogic) GetElectricitySurplus(req *types.GetElectricitySurplusReq) (resp *types.GetElectricitySurplusResp, err error) {
	result, err := l.authManger.WithAuthToken(req.Uid, func(token string) (any, error) {
		return l.fetchElectricitySurplus(req, token)
	})
	if err != nil {
		return nil, err
	}
	return result.(*types.GetElectricitySurplusResp), nil
}
