//nolint:typecheck
package config

import (
	"ai_task/constant"
	"ai_task/pkg/file"
	"fmt"
	"os"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	Path = "config"

	OSConfigPath      = "CONFIG_PATH"
	DefaultConfigName = "config.yaml"
	TypeYaml          = "yaml"
	ProjectName       = "ai_web"

	ApplicationLogRequest = "app.log.request"
	AppLogLevel           = "app.log.level"
	AppLogReportcaller    = "app.log.reportcaller"
	AppHost               = "app.host"

	BaseDbXormType     = "base.db.xorm.type"
	BaseDbXormUsername = "base.db.xorm.username"
	BaseDbXormPassword = "base.db.xorm.password"
	BaseDbXormHost     = "base.db.xorm.host"
	BaseDbXormPort     = "base.db.xorm.port"
	BaseDbXormName     = "base.db.xorm.name"
	BaseDbXormShowsql  = "base.db.xorm.showsql"

	ClientsCommonRequestLog = "clients.http.requestLog" // pkg/clients http client 是否打印请

	// 大模型调用配置
	ClientChatModelAddr        = "clients.llmModel.addr"
	ClientChatModelModel       = "clients.llmModel.model"
	ClientChatModelTemperature = "clients.llmModel.temperature"
	ClientChatModelMaxTokens   = "clients.llmModel.maxTokens"

	// Embedding 客户端配置键
	EmbeddingConfigKeyModelName = "clients.embedding.model_name"
	EmbeddingConfigKeyBaseURL   = "clients.embedding.base_url"

	// redis 配置
	RedisClientDb       = "clients.redisClient.db"
	RedisClientHost     = "clients.redisClient.host"
	RedisClientPassword = "clients.redisClient.password"

	// 记忆系统配置
	MemoryEnableSessionMemory = "memory.enable_session_memory"
	MemoryEnableChunking      = "memory.enable_chunking"
	MemorySessionMemoryLimit  = "memory.session_memory_limit"
	MemorySemanticMemoryLimit = "memory.semantic_memory_limit"
	MemorySemanticThreshold   = "memory.semantic_threshold"
	MemoryCompressThreshold   = "memory.compress_threshold"
	MemoryEnableSummary       = "memory.enable_summary"
	MemoryEnableAutoExtract   = "memory.enable_auto_extract"
	MemoryChunkMaxSize        = "memory.chunk_max_size"
	MemoryChunkOverlap        = "memory.chunk_overlap"
	MemoryChunkMinSize        = "memory.chunk_min_size"
	MemoryChunkStrategy       = "memory.chunk_strategy"
)

var instance *config
var once sync.Once

type config struct {
	*viper.Viper
}

func GetInstance() *config {
	once.Do(func() {
		var configPath string

		envConfigPath := os.Getenv(OSConfigPath)
		if strings.EqualFold(envConfigPath, constant.EmptyString) {
			configPath = fmt.Sprintf("./%v", DefaultConfigName)
			if !file.CheckFileIsExist(configPath) {
				path, err := os.Getwd()
				if err != nil {
					panic("get config path error:" + err.Error())
				}
				configPath = fmt.Sprintf("%v/%v", path[:strings.Index(path, ProjectName)+len(ProjectName)], DefaultConfigName)
			}
			log.Infof("use default path %s", configPath)
		} else {
			log.Infof("find success in constant CONFIG_PATH, use %s", envConfigPath)
			configPath = fmt.Sprintf("%v/%v", envConfigPath, DefaultConfigName)
		}

		configInstance := &config{Viper: viper.New()}
		configInstance.SetConfigType(TypeYaml)
		configInstance.SetConfigFile(configPath)
		if err := configInstance.ReadInConfig(); err != nil {
			panic(err)
		}

		configInstance.AutomaticEnv()
		replacer := strings.NewReplacer(".", "_")
		configInstance.SetEnvKeyReplacer(replacer)

		keys := configInstance.AllKeys()
		for _, key := range keys {
			fmt.Println(key, ":", configInstance.Get(key))
		}

		instance = configInstance
	})
	return instance
}

func (c *config) GetString(key string) string {
	return c.Viper.GetString(key)
}

func (c *config) GetStringOrDefault(key string, defaultValue string) string {
	if c.IsSet(key) {
		return c.GetString(key)
	}

	return defaultValue
}

func (c *config) GetInt(key string) int {
	return c.Viper.GetInt(key)
}

func (c *config) GetIntOrDefault(key string, defaultValue int) int {
	if c.IsSet(key) {
		return c.GetInt(key)
	}

	return defaultValue
}

func (c *config) GetBool(key string) bool {
	return c.Viper.GetBool(key)
}

func (c *config) GetBoolOrDefault(key string, defaultValue bool) bool {
	if c.IsSet(key) {
		return c.GetBool(key)
	}

	return defaultValue
}

func (c *config) GetFloat64(key string) float64 {
	return c.Viper.GetFloat64(key)
}

func (c *config) GetFloat64OrDefault(key string, defaultValue float64) float64 {
	if c.IsSet(key) {
		return c.GetFloat64(key)
	}

	return defaultValue
}
