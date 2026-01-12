PROJECT_PATH := $(shell pwd)
PROJECT_NAME ?= $(shell basename $(PROJECT_PATH))
PROJECT_BRANCH ?= $(shell git symbolic-ref --short -q HEAD)
PROJECT_VERSION ?= $(shell git describe --abbrev=0 --tags)
PROJECT_TAG ?= $(shell git describe --abbrev=0 --tags)
PROJECT_ARCH ?= "amd64"
PROJECT_REPOSITORY ?= "dockerhub.deepglint.com/deepface/$(PROJECT_NAME)"
RELEASE_FROM ?= "docker"
CONFIG_PATH ?= "$(PROJECT_PATH)/config.yaml"

.PHONY: info lint test coverage clear build release release_docker release_binary


swag: ## build swagger
	#swag init --propertyStrategy pascalcase --parseDependency --parseInternal
	@swag init --propertyStrategy pascalcase

info: ## 打印信息
	@echo "项目路径: $(PROJECT_PATH)"
	@echo "项目名称: $(PROJECT_NAME)"
	@echo "项目分支: $(PROJECT_BRANCH)"
	@echo "项目版本: $(PROJECT_VERSION)"
	@echo "项目Tag: $(PROJECT_TAG)"
	@echo "编译平台: $(PROJECT_ARCH)"
	@echo "镜像名称: $(PROJECT_REPOSITORY)"
	@echo "发布方式: $(RELEASE_FROM)"

lint: ## 格式检查
	@echo "格式检查开始"
	@golangci-lint run --skip-dirs=vendor/,model/dg_model --no-config --issues-exit-code=1
	@echo "格式检查完成"

test: ## 测试
	@echo "配置文件: $(CONFIG_PATH)"
	@echo "测试开始"
	@CONFIG_PATH=$(PWD) go test -v -failfast -short -race -coverprofile="bin/cover.out" ./...|tee bin/ut.tmp
	@tar -zcvf bin/unitTestReport.tar.gz bin/ut.tmp
	@#grep -E '===|---' bin/ut.tmp > bin/utStatistics.tmp
	@echo "测试完成"

coverage: ## 覆盖率
	@echo "生成覆盖率文件开始"
	@go tool cover -func=bin/cover.out|tee bin/ut_coverage.tmp
	@go tool cover -html="bin/cover.out" -o "bin/cover.html"
	@tar -zcvf bin/cover.tar.gz bin/cover.html
	@echo "生成覆盖率文件完成"

clear: ## 清理
	@echo "清理开始"
	@rm -rf bin/*
	@rm -rf dist/*
	@echo "清理完成"

build: ## 编译
	@make info
	@make clear
	@echo "编译开始"
	@CGO_ENABLED=0 GOOS=linux GOARCH=$(PROJECT_ARCH) go build -mod=vendor -v -o bin/$(PROJECT_NAME)
	@echo "编译完成"

release: ## 发布
ifeq ($(RELEASE_FROM),"docker")
	@make release_docker
else
	@make release_binary
endif

release_docker: ## 容器发布_发版
	@make build
	@echo "容器发布开始"
	@cp config.yaml bin/config.yaml
	@mkdir bin/tmp
	@docker build --build-arg path=bin --build-arg name=$(PROJECT_NAME) --tag $(PROJECT_REPOSITORY):$(PROJECT_TAG) .
	@docker push $(PROJECT_REPOSITORY):$(PROJECT_TAG)
	@echo "容器发布完成"

release_docker_debug: ## 容器发布_调试
	@make build
	@echo "容器发布开始"
	@cp config.yaml bin/config.yaml
	@cp -rf resources bin
	@mkdir bin/tmp
	@docker build --build-arg path=bin --build-arg name=$(PROJECT_NAME) --tag $(PROJECT_REPOSITORY):debug .
	@docker push $(PROJECT_REPOSITORY):debug
	@echo "容器发布完成"

release_binary: ## 二进制发布
	@make build
	@echo "二进制发布开始"
	@mkdir -p n/$(PROJECT_NAME)-$(PROJECT_VERSION)
	@mv bin/$(PROJECT_NAME) bin/$(PROJECT_NAME)-$(PROJECT_VERSION)
	@mkdir bin/$(PROJECT_NAME)-$(PROJECT_VERSION)/tmp
	@mkdir -p bin/$(PROJECT_NAME)
	@mv bin/$(PROJECT_NAME)-$(PROJECT_VERSION) bin/$(PROJECT_NAME)
	@ln -s $(PROJECT_NAME)-$(PROJECT_VERSION) bin/$(PROJECT_NAME)/latest
	@cd bin && tar -zcvf $(PROJECT_NAME)-$(PROJECT_VERSION).tar.gz $(PROJECT_NAME)/
	@echo "二进制发布完成"

help: ## 帮助信息
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
