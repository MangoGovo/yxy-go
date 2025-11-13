package bus

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"yxy-go/internal/logic/bus"
	"yxy-go/internal/svc"
	"yxy-go/internal/types"
	"yxy-go/pkg/response"
)

func GetBusQrcodeHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetBusQrcodeReq
		if err := httpx.Parse(r, &req); err != nil {
			response.ParamErrorResponse(r, w, err)
			return
		}

		l := bus.NewGetBusQrcodeLogic(r.Context(), svcCtx)
		resp, err := l.GetBusQrcode(&req)
		response.HttpResponse(r, w, resp, err)
	}
}
