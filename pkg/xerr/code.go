package xerr

//go:generate stringer -type Code -linecomment

type Code int

// success
const ErrSuccess Code = 0 // Success

// common err
const (
	ErrUnknown    Code = iota + 100001 // 服务异常
	ErrParam                           // 参数错误
	ErrHttpClient                      // HTTP客户端请求错误
)

// yxy common err
const (
	ErrUserNotFound     Code = iota + 100101 // 用户不存在
	ErrAccountLoggedOut                      // 账号被登出
	ErrNotBindCard                           // 用户还未绑卡
)

// login err
const (
	ErrTokenInvalid         Code = iota + 110001 // Token无效
	ErrCaptchaInvalid                            // 图片验证码已失效
	ErrCaptchaWrong                              // 图片验证码错误
	ErrDeviceIDInconsistent                      // deviceId不一致
	ErrPhoneNumWrong                             // 手机号格式错误
	ErrSendLimit                                 // 短信发送超限
	ErrCodeWrong                                 // 手机验证码错误, 错误3次将锁定15分钟
	ErrCodeWrongThreeTimes                       // 手机验证码错误3次, 账号锁定15分钟
)

// electricity err
const (
	ErrElectricityTokenInvalid       Code = iota + 110101 // 电费Token无效
	ErrElectricityBindNotFound                            // 未找到电费绑定信息
	ErrRoomInfoWrongOrCampusMismatch                      // 房间信息有误或校区不匹配
)

// bus err
const (
	ErrBusTokenInvalid Code = iota + 110201 // 校车Token无效
)
