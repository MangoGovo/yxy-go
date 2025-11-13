# yxy-go

YXY(yiSchool) platform HTTP API bindings, written in go, refactored from [yxy](https://github.com/zjutjh/yxy).
A monolithic service written using the go-zero framework.

---

## Features

-   [x] APP login stage simulation.
-   [x] Query school card balance and consumption records.
-   [x] Query electricity surplus, recharge and usage records.
-   [ ] More...

## Development

1. Set up the `go-zero` development environment. [reference](https://go-zero.dev/docs/tasks)
2. Clone the repo.
    ```sh
    git clone https://github.com/zjutjh/yxy-go.git
    ```
3. Modify `api/handler.tpl` template. [reference](https://go-zero.dev/docs/tutorials/customization/template)

    ```go
    package {{.PkgName}}

    import (
        "net/http"

        "github.com/zeromicro/go-zero/rest/httpx"
        "yxy-go/pkg/response"
        {{.ImportPackages}}
    )

    {{if .HasDoc}}{{.Doc}}{{end}}
    func {{.HandlerName}}(svcCtx *svc.ServiceContext) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
            {{if .HasRequest}}var req types.{{.RequestType}}
            if err := httpx.Parse(r, &req); err != nil {
                response.ParamErrorResponse(r, w, err)
                return
            }

            {{end}}l := {{.LogicName}}.New{{.LogicType}}(r.Context(), svcCtx)
            {{if .HasResp}}resp, {{end}}err := l.{{.Call}}({{if .HasRequest}}&req{{end}})
            response.HttpResponse(r, w, resp, err)
        }
    }
    ```

4. Create or edit `.api` files in the `api` directory. [reference](https://go-zero.dev/docs/tutorials)
5. Use `goctl` to automatically generate code. [reference](https://go-zero.dev/docs/tutorials/cli/api)
    ```sh
    goctl api go -api api/yxy.api -dir . --style goZero
    ```
6. Implement the business logic in the `internal/logic` directory.

## Disclaimer

Completely FREE software for learning only. **Any inappropriate use is at your own risk.**

## TODO

添加授权逻辑。逻辑如下:

常规登录模式：

向`https://open.xiaofubao.com/routeauth/auth/route/ua/authorize/getCodeV2`发送请求后，获取响应头中的`location`进行重定向，重定向之后即可从响应中的`location`字段获取`openid`及`corpcode`，此后带上这两个参数向`https://api.pinbayun.com/api/v1/staff/auths/wx_auth/`发送请求即可

新登录模式：

向 `https://open.xiaofubao.com/routeauth/auth/route/ua/authorize/getCodeV2`发送请求后，若响应头`location`为空，则检查响应体（需验证响应体正常登录的时候是否为空或者为其他的东西），若响应体是一个页面（弹窗），则向从页面中获取 stateCode,并向`https://auth.xiaofubao.com/auth/route/authorize/agreementAuth?stateCode=&appid=2011112043190345310`发送请求。获取响应头中的`location`字段，继续按照正常登录流程即可。
