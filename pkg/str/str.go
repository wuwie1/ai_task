package str

import (
	"strconv"
	"strings"
	"unsafe"
)

// 字符串转int
func StringToInt(str string) (int, error) {
	if str == "" {
		return 0, nil
	}

	i, err := strconv.Atoi(str)
	if err != nil {
		return 0, err
	}

	return i, err
}

// RemoveBothSidesBrackets 去除字符串两边括号
func RemoveBothSidesBrackets(str string) string {
	if strings.Contains(str, "(") || strings.Contains(str, ")") {
		runeArr := []rune(str)
		startPoint := 0
		endPoint := len(runeArr) - 1
		for {
			if runeArr[startPoint] == '(' && runeArr[endPoint] == ')' {
				startPoint++
				endPoint--
			} else {
				break
			}
		}
		str = string(runeArr[startPoint : endPoint+1])
	}
	return str
}

// @Title StrToBytes
// @Description 字符串转 byte
// @params s string ""
// @return  []byte ""
func StrToBytes(s string) []byte {
	x := (*[2]uintptr)(unsafe.Pointer(&s))
	h := [3]uintptr{x[0], x[1], x[1]}
	return *(*[]byte)(unsafe.Pointer(&h))
}

// @Title FloorFloatLimitToString
// @Description 截取 float 并转为 string
// @params f float64 ""
// @params limit int "截取的小数点位数"
// @return  - string ""
func FloorFloatLimitToString(f float64, limit int) string {
	if f == 0 || limit == 0 {
		return ""
	}
	temp := strconv.FormatFloat(f, 'g', limit+1, 64)
	if limit+2 > len(temp) {
		limit = len(temp)
	} else {
		limit = limit + 2
	}
	return temp[:limit]
}
