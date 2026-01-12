package model

import (
	"fmt"
	"regexp"

	log "github.com/sirupsen/logrus"
)

const (
	ErrorCodeUserIPNotMatch   = 100001
	ErrorUserNameEmpty        = 100002
	ErrorUserPasswordEmpty    = 100003
	ErrorUserNameOrPassword   = 100004
	ErrorLoginAgain           = 100005
	ErrorUserNotExist         = 100006
	ErrorUserLogout           = 100007
	ErrorUserForbidden        = 100008
	ErrorUserPhoneNumberExist = 100009
	ErrorParams               = 100010
	ErrorEmptyId              = 100011
	ErrorNewRepo              = 100012
	ErrorMakeToken            = 100013
	ErrorUserPhoneNumberEmpty = 100014
	ErrorDB                   = 100015
)

// 自定义扩展的 http 状态码
const (
	HttpStatusNoPermission = 491 // 无权限
)

var ErrorMessages = map[int]string{
	ErrorCodeUserIPNotMatch:   "该 IP 下，此用户禁止登录",
	ErrorUserNameEmpty:        "用户名不能为空",
	ErrorUserPasswordEmpty:    "密码不能为空",
	ErrorUserNameOrPassword:   "用户名或密码错误",
	ErrorLoginAgain:           "该账号于%v点%v分在其他设备登录",
	ErrorUserNotExist:         "用户不存在",
	ErrorUserLogout:           "用户已登出",
	ErrorUserForbidden:        "账号被锁定",
	ErrorUserPhoneNumberExist: "手机号已被注册",
	ErrorParams:               "参数错误",
	ErrorEmptyId:              "id 为空",
	ErrorNewRepo:              "新建 repo 失败",
	ErrorMakeToken:            "生成 token 失败",
	ErrorUserPhoneNumberEmpty: "手机号不能为空",
	ErrorDB:                   "db error",
}

type Error struct {
	Code       int    `json:"code"`
	Message    string `json:"message"`
	InnerError error  `json:"inner_error"`
}

func (err Error) Error() string {
	return err.Message
}

func (err Error) String() string {
	return err.InnerError.Error()
}

func NewError(code int, innerError error) *Error {
	if innerError != nil {
		var re = regexp.MustCompile(`[\n\t]+`)
		innerErrorString := re.ReplaceAllString(fmt.Sprintf("%+v", innerError), " ")
		log.Errorf("NewError code:%d, message:%s, innerError:%+v\n", code, ErrorMessages[code], innerErrorString)
	}
	return &Error{
		Code:       code,
		Message:    ErrorMessages[code],
		InnerError: nil,
	}
}

func NewErrorVerificationFailed(code int) *Error {
	return &Error{
		Code:       code,
		Message:    ErrorMessages[code],
		InnerError: fmt.Errorf("verification failed "),
	}
}
func NewErrorWithMessage(code int, message string) *Error {
	return &Error{
		Code:       code,
		Message:    message,
		InnerError: nil,
	}
}
