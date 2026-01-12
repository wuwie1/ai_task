项目目录结构：

```text
├── config                      // 配置, 配置文件解析
├── constant                    // 全局常量
├── controller                  // 控制层，api 入口
├── bin                         // 二进制包
├── docs                        // 文档相关
├── entity                      // 领域实体 定义基础组件，不依赖其他的组件，基层的交互，数据表定义的地方
├── env                         // 环境变量相关
├── middleware                  // 中间件，api 请求中间件封装等
├── model                       // 请求、响应 struct 等
├── pkgs                        // 各种包封装			
│    ├── clients                // 各种第三方 client 请求封装
│    │    ├── embedding         // 调用 embedding 模型进行向量化
│    │    ├── httptool          // 封装的 http 调用的工具类方法
│    │    ├── llm_model         // 调用大语言模型的封装
│    │    └── redis             // redis 缓存 client
│    ├── daemon                 // 持久化进程
│    ├── file                   // 文件操作相关方法封装
│    ├── str                    // 字符串相关方法封装
│    ├── time                   // 时间相关方法封装
│    └── tools                  // 工具类方法封装
├── repository                  // DAO 层，持久层，与存储交互
├── routers                     // api 接入，路由定义等
├── services                    // 业务层，各种业务逻辑，一般逻辑是 routers->controller->services->repository
```
