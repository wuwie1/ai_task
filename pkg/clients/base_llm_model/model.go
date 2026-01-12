package base_llm_model

// Config 基础LLM模型配置
type Config struct {
	BaseURL     string  `json:"base_url"`     // API基础地址
	APIKey      string  `json:"api_key"`      // API密钥
	ModelName   string  `json:"model_name"`   // 模型名称
	Temperature float32 `json:"temperature"`  // 温度参数，控制输出随机性
	MaxTokens   int     `json:"max_tokens"`   // 最大输出token数
}

// ClientParams 客户端必填参数结构体
type ClientParams struct {
	BaseURL   string `json:"base_url"`   // API基础地址（必填）
	APIKey    string `json:"api_key"`    // API密钥（必填）
	ModelName string `json:"model_name"` // 模型名称（必填）
}

// Option 配置选项函数类型
type Option func(*Config)

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Temperature: 0.7,
		MaxTokens:   4096,
	}
}

// WithBaseURL 设置API基础地址
func WithBaseURL(baseURL string) Option {
	return func(c *Config) {
		c.BaseURL = baseURL
	}
}

// WithAPIKey 设置API密钥
func WithAPIKey(apiKey string) Option {
	return func(c *Config) {
		c.APIKey = apiKey
	}
}

// WithModelName 设置模型名称
func WithModelName(modelName string) Option {
	return func(c *Config) {
		c.ModelName = modelName
	}
}

// WithTemperature 设置温度参数
func WithTemperature(temperature float32) Option {
	return func(c *Config) {
		c.Temperature = temperature
	}
}

// WithMaxTokens 设置最大输出token数
func WithMaxTokens(maxTokens int) Option {
	return func(c *Config) {
		c.MaxTokens = maxTokens
	}
}
