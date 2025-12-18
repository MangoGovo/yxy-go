package bus

import (
	"net/http"
	"yxy-go/pkg/response"

	"yxy-go/internal/logic/bus"
	"yxy-go/internal/svc"
	"yxy-go/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func GetBusAnnouncementHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetBusAnnouncementReq
		if err := httpx.Parse(r, &req); err != nil {
			response.ParamErrorResponse(r, w, err)
			return
		}

		l := bus.NewGetBusAnnouncementLogic(r.Context(), svcCtx)
		resp, err := l.GetBusAnnouncement(&req)
		response.HttpResponse(r, w, resp, err)
	}
}
