package bus

import (
	"net/http"

	"yxy-go/internal/logic/bus"
	"yxy-go/internal/svc"
	"yxy-go/internal/types"
	"yxy-go/pkg/response"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func GetBusRecordHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetBusRecordReq
		if err := httpx.Parse(r, &req); err != nil {
			response.ParamErrorResponse(r, w, err)
			return
		}

		l := bus.NewGetBusRecordLogic(r.Context(), svcCtx)
		resp, err := l.GetBusRecord(&req)
		response.HttpResponse(r, w, resp, err)
	}
}
