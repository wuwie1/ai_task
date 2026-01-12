package time

import (
	"time"
)

const (
	TimeFormatTableStyleYear  = "2006"                     //sql表风格到年级别，时间格式化模版
	TimeFormatTableStyleMonth = "200601"                   //sql表风格到月级别，时间格式化模版
	TimeFormatTableStyleDay   = "20060102"                 //sql表风格到天级别，时间格式化模版
	TimeFormatTableStyleMin   = "200601021504"             //sql表风格到分钟级别，时间格式化模版
	TimeFormatTableStyleSec   = "20060102150405"           //sql表风格到秒级别，时间格式化模版
	TimeFormatTableStyleMilli = "20060102150405.000"       //sql表风格到毫秒级别，时间格式化模版
	TimeFormatTableStyleMicro = "20060102150405.000000"    //sql表风格到微秒级别，时间格式化模版
	TimeFormatTableStyleNano  = "20060102150405.000000000" //sql表风格到纳秒级别，时间格式化模版

	TimeFormatCommonStyleDay = "2006-01-02"
	TimeFormatCommonStyleMin = "2006-01-02 15:04"
	TimeFormatCommonStyleSec = "2006-01-02 15:04:05"

	TimeFormatExcelStyleDay = "2006/1/2"
)

func GetNowTimestamp() int64 {
	return time.Now().UnixNano() / 1000000
}

func GetNowTimeByFormat(format string) string {
	return time.Now().Format(format)
}

func GetNowTimeStr() string {
	return time.Now().Format(TimeFormatTableStyleSec)
}

func ConvertInt64ToStr(timestamp int64) string {
	return time.Unix(timestamp/1000, 0).Format(TimeFormatTableStyleSec)
}

func ConvertInt64ToStrByFormat(timestamp int64, format string) string {
	return time.Unix(timestamp/1000, 0).Format(format)
}

func ConvertStrToInt64(timeStr string) (int64, error) {
	loc, err := time.LoadLocation("Local")
	if err != nil {
		return 0, err
	}
	t, err := time.ParseInLocation(TimeFormatTableStyleSec, timeStr, loc)
	if err != nil {
		return 0, err
	}
	return t.UnixNano() / 1000000, nil
}

func ConvertStrToInt64ToFormat(format, timeStr string) (int64, error) {
	loc, err := time.LoadLocation("Local")
	if err != nil {
		return 0, err
	}
	t, err := time.ParseInLocation(format, timeStr, loc)
	if err != nil {
		return 0, err
	}
	return t.UnixNano() / 1000000, nil
}

// @Title TimeFormatTableStyleSecToUnixTimestampDefaultZero
// @Description format time change to millisecond
// @example 20220523012821  =>  1653240501000
func TimeFormatTableStyleSecToUnixTimestampDefaultZero(str string) int64 {
	temp, err := time.ParseInLocation(TimeFormatTableStyleSec, parseToTimeFormatTableStyleSec(str), time.Local)
	if err != nil {
		return 0
	}
	return temp.Unix() * 1000
}

// @Title parseToTimeFormatTableStyleSec
// @Description parseToTimeFormatTableStyleSec 补全或者截取至TimeFormatTableStyleSec模版长度
func parseToTimeFormatTableStyleSec(t string) string {
	if len(t) == 0 {
		return "19700101080000"
	}

	if len(t) > len(TimeFormatTableStyleSec) {
		return t[:len(TimeFormatTableStyleSec)]
	}

	if len(t) < len(TimeFormatTableStyleSec) {
		l := len(t)
		slice := []byte(t)
		for l < len(TimeFormatTableStyleSec) {
			slice = append(slice, '0')
			l++
		}
		return string(slice)
	}
	return t
}
