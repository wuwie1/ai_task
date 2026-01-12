package tools

import (
	"fmt"
	"reflect"
	"strings"
)

func ParseStructFieldXormTag(data interface{}) (map[string]string, error) {
	res := make(map[string]string)

	typeOfData := reflect.TypeOf(data)
	if typeOfData.Kind() != reflect.Ptr {
		return res, fmt.Errorf("params must be pt")
	}
	if typeOfData.Elem().Kind() != reflect.Struct {
		return res, fmt.Errorf("params must be struct")
	}

	prefixString := `xorm:"`
	elem := typeOfData.Elem()
	for i := 0; i < elem.NumField(); i++ {
		tempTag := string(elem.Field(i).Tag)
		if tempTag == "" {
			continue
		}
		if !strings.HasPrefix(tempTag, prefixString) {
			continue
		}
		res[elem.Field(i).Name] = strings.Split(tempTag[len(prefixString):], " ")[0]
	}
	return res, nil
}
