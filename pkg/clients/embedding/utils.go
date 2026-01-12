package embedding

import (
	"fmt"
	"strings"
)

// VectorToString 将 float64 切片转换为 PostgreSQL vector 格式字符串
// 格式: [1.0,2.0,3.0]
func VectorToString(vec []float64) string {
	if len(vec) == 0 {
		return "[]"
	}

	var builder strings.Builder
	builder.WriteString("[")
	for i, v := range vec {
		if i > 0 {
			builder.WriteString(",")
		}
		builder.WriteString(fmt.Sprintf("%.6f", v))
	}
	builder.WriteString("]")
	return builder.String()
}

