package env

import "os"

func GetModelBaseUrl() string {
	return os.Getenv("MODEL_BASE_URL")
}

func GetLLMModel() string {
	return os.Getenv("LLM_MODEL")
}

func GetModelApiKey() string {
	return os.Getenv("LLM_API_KEY")
}

func GetGMLModelBaseUrl() string {
	return os.Getenv("MODEL_BASE_GLM_URL")
}

func GetGMLLLMModel() string {
	return os.Getenv("LLM_GML_API_KEY")
}

func GetGMLModelApiKey() string {
	return os.Getenv("LLM_GML_MODEL")
}
