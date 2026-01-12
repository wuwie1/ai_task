package llm_model

type Config struct {
	Addr        string  `json:"addr"`
	V1Addr      string  `json:"v1Addr"`
	Model       string  `json:"llm_model"`
	Token       string  `json:"token"`
	Temperature float32 `json:"temperature"`
	MaxTokens   int     `json:"maxTokens"`
}
