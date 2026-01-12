package tools

func GetParamsFields(params map[string]interface{}, listObject, object, id string) map[string][]string {
	res := make(map[string][]string)
	list, ok := params[listObject].(map[string]interface{})[object].([]interface{})
	if !ok {
		return res
	}
	for i := 0; i < len(list); i++ {
		temp, ok := list[i].(map[string]interface{})
		if !ok {
			continue
		}
		appId, ok := temp[id].(string)
		if !ok {
			continue
		}
		for k := range temp {
			if k == id {
				continue
			}
			res[appId] = append(res[appId], k)
		}
	}
	return res
}
