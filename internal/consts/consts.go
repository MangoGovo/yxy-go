package consts

const (
	APP_ALL_VERSION = "7.3.6"
	APP_VERSION     = "730"
	CLIENT_ID       = "65l3attk4r095ib"
	SCHOOL_CODE     = "10337" // ZJUT
	COMPUS_URL      = "https://compus.xiaofubao.com"
	APPLICATION_URL = "https://application.xiaofubao.com"
	AUTH_URL        = "https://auth.xiaofubao.com"
	BUS_URL         = "https://api.pinbayun.com"
	BUS_AUTH_URL    = "https://open.xiaofubao.com"
)

const (
	GET_SECURITY_TOKEN_URL = COMPUS_URL + "/common/security/token"
	GET_CAPTCHA_IMAGE_URL  = COMPUS_URL + "/common/security/imageCaptcha"
	SEND_CODE_URL          = COMPUS_URL + "/compus/user/sendLoginVerificationCode"
	LOGIN_BY_CODE_URL      = COMPUS_URL + "/login/doLoginByVerificationCode"
	LOGIN_BY_Silent_URL    = COMPUS_URL + "/login/doLoginBySilent"
	GET_AUTH_TOKEN         = COMPUS_URL + "/compus/user/getAuthToken"
)

const (
	GET_CARD_BALANCE_URL             = COMPUS_URL + "/compus/user/getCardMoney"
	GET_CARD_CONSUMPTION_RECORDS_URL = COMPUS_URL + "/routeauth/auth/route/user/cardQuerynoPage"
)

const (
	GET_AUTH_CODE_URL  = AUTH_URL + "/authoriz/getCodeV2"
	GET_AUTH_TOKEN_URL = APPLICATION_URL + "/app/login/getUser4Authorize"
)

const (
	QUERY_ELECTRICITY_BIND_URL                = APPLICATION_URL + "/app/electric/queryBind"
	GET_ELECTRICITY_ZHPF_SURPLUS_URL          = APPLICATION_URL + "/app/electric/queryISIMSRoomSurplus"
	GET_ELECTRICITY_MGS_SURPLUS_URL           = APPLICATION_URL + "/app/electric/queryRoomSurplus"
	GET_ELECTRICITY_ZHPF_RECHARGE_RECORDS_URL = APPLICATION_URL + "/app/electric/queryISIMSRoomBuyRecord"
	GET_ELECTRICITY_MGS_RECHARGE_RECORDS_URL  = APPLICATION_URL + "/app/electric/roomBuyRecord"
	GET_ELECTRICITY_ZHPF_USAGE_RECORDS_URL    = APPLICATION_URL + "/app/electric/getISIMSRecords"
	GET_ELECTRICITY_MGS_USAGE_RECORDS_URL     = APPLICATION_URL + "/app/electric/queryUsageRecord"
)

const (
	GET_BUS_AUTH_CODE_URL    = BUS_AUTH_URL + "/routeauth/auth/route/ua/authorize/getCodeV2"
	GET_BUS_AUTH_TOKEN_URL   = BUS_URL + "/api/v1/staff/auths/wx_auth/"
	GET_BUS_ACCESS_URL       = AUTH_URL + "/auth/route/authorize/agreementAuth"
	GET_BUS_INFO_URL         = BUS_URL + "/api/v2/staff/shuttlebus/"
	GET_BUS_TIME_URL         = BUS_URL + "/api/v2/staff/shuttlebus/{id}/bustimes/"
	GET_BUS_DATE_URL         = BUS_URL + "/api/v2/staff/shuttlebus/{id}/dates/"
	GET_BUS_RECORD_URL       = BUS_URL + "/api/v1/staff/busorders/"
	GET_BUS_ANNOUNCEMENT_URL = BUS_URL + "/api/v1/staff/messages/"
	// GET_BUS_MESSAGE_UNREAD_COUNT_URL = BUS_URL + "/api/v1/staff/messages/unread_count/"
)

const (
	ELECTRICTY_APPID = "1810181825222034"
	BUS_APPID        = "2011112043190345310"
)
