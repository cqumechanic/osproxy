package plugins

import (
	"fmt"

	"github.com/qinguoyi/osproxy/bootstrap" // bootstrap包用于初始化配置文件和日志
)

// Plugin 插件接口
type Plugin interface { // interface关键字用于定义接口,接口是一种类型，一种抽象的类型
	// 它不会暴露所含数据的格式，也不会暴露相关的基本操作，它只会展示出它自己的方法
	// interface是一组method的组合，我们通过interface来定义对象的一组行为
	// interface的好处是，它可以使代码更加灵活，更加有效，更加具有扩展性
	// Flag 是否启动
	Flag() bool
	// Name 插件名称
	Name() string
	// New 初始化插件资源
	New() interface{}
	// Health 插件健康检查
	Health()
	// Close 释放插件资源
	Close()
}

// Plugins 插件注册集合
var Plugins = make(map[string]Plugin)

// RegisteredPlugin 插件注册
func RegisteredPlugin(plugin Plugin) {
	Plugins[plugin.Name()] = plugin
}

func NewPlugins() {
	for _, p := range Plugins {
		if !p.Flag() {
			continue
		}
		bootstrap.NewLogger().Logger.Info(fmt.Sprintf("%s Init ... ", p.Name()))
		p.New()
		bootstrap.NewLogger().Logger.Info(fmt.Sprintf("%s HealthCheck ... ", p.Name()))
		p.Health()
		bootstrap.NewLogger().Logger.Info(fmt.Sprintf("%s Success Init. ", p.Name()))
	}
}

func ClosePlugins() {
	for _, p := range Plugins {
		if !p.Flag() {
			continue
		}
		p.Close()
		bootstrap.NewLogger().Logger.Info(fmt.Sprintf("%s Success Close ... ", p.Name()))
	}
}
