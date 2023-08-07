package main

import (
	"github.com/qinguoyi/osproxy/api"               // api包用于初始化路由
	"github.com/qinguoyi/osproxy/app"               // app包用于初始化http服务
	"github.com/qinguoyi/osproxy/app/pkg/base"      // base包用于初始化Snowflake
	"github.com/qinguoyi/osproxy/app/pkg/storage"   // storage包用于初始化storage
	"github.com/qinguoyi/osproxy/bootstrap"         // bootstrap包用于初始化配置文件和日志
	"github.com/qinguoyi/osproxy/bootstrap/plugins" // plugins包用于初始化插件资源
)

// @title    ObjectStorageProxy
// @version  1.0
// @description
// @contact.name  qinguoyi
// @host          127.0.0.1:8888
// @BasePath      /
func main() {
	// config log
	lgConfig := bootstrap.NewConfig("conf/config.yaml") // 这里的lgConfig是全局变量，所以调用方直接声明变量就能访问到,将配置文件读入lgConfig，lgConfig是一个结构体,包含两个成员变量，一个是Conf，一个是Once
	lgLogger := bootstrap.NewLogger()

	// plugins DB Redis Minio
	plugins.NewPlugins()         // NewPlugins()函数用于初始化插件资源
	defer plugins.ClosePlugins() //	ClosePlugins()函数用于释放插件资源 defer关键字用于延迟函数的执行

	// init Snowflake
	base.InitSnowFlake() // InitSnowFlake()函数用于初始化Snowflake,雪花算法

	// init storage
	storage.InitStorage(lgConfig) // InitStorage()函数用于初始化storage

	// router
	engine := api.NewRouter(lgConfig, lgLogger)   // NewRouter()函数用于初始化路由,enigne是gin的核心结构体，包含了路由、中间件等信息
	server := app.NewHttpServer(lgConfig, engine) // NewHttpServer()函数用于初始化http服务

	// app run-server
	application := app.NewApp(lgConfig, lgLogger.Logger, server) // NewApp()函数用于初始化app
	application.RunServer()                                      // RunServer()函数用于运行app
}
