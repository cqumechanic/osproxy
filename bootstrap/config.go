package bootstrap

import (
	"fmt"           // 格式化
	"path/filepath" // 路径
	"sync"          // 同步

	"github.com/fsnotify/fsnotify"       // 文件系统监控
	"github.com/qinguoyi/osproxy/config" // 配置文件
	"github.com/spf13/pflag"             // 命令行参数解析
	"github.com/spf13/viper"             // 读取配置文件
	"go.uber.org/zap"                    // 日志
)

var (
	configPath   string
	rootPath     = ""                // utils.RootPath()
	lgConfig     = new(LangGoConfig) //
	confFilePath = "conf/config.yaml"
)

// LangGoConfig 自定义Log
type LangGoConfig struct {
	Conf *config.Configuration // 配置文件  app/pkg/utils/config.go
	Once *sync.Once            // 同步 Once
}

// newLangGoConfig .
func newLangGoConfig() *LangGoConfig {
	return &LangGoConfig{
		Conf: &config.Configuration{},
		Once: &sync.Once{},
	}
}

// NewConfig 初始化配置对象
func NewConfig(confFile string) *config.Configuration {
	if lgConfig.Conf != nil { //很多地方用到这个函数，所以会让他们都返回同一个配置文件（全局变量）
		return lgConfig.Conf
	} else {
		lgConfig = newLangGoConfig()
		if confFile == "" {
			lgConfig.initLangGoConfig(confFilePath)
		} else {
			lgConfig.initLangGoConfig(confFile)
		}
		return lgConfig.Conf
	}
}

// InitLangGoConfig 初始化日志
func (lg *LangGoConfig) initLangGoConfig(confFile string) {
	lg.Once.Do( // Once.Do()函数用于保证函数只执行一次
		func() {
			initConfig(lg.Conf, confFile)
		},
	)
}

func initConfig(conf *config.Configuration, confFile string) {
	pflag.StringVarP(&configPath, "conf", "", filepath.Join(rootPath, confFile),
		"config path, eg: --conf config.yaml") //stringVarP 用于绑定命令行参数和变量 copaliot真厉害
	if !filepath.IsAbs(configPath) { // filepath.IsAbs()函数用于判断路径是否为绝对路径
		configPath = filepath.Join(rootPath, configPath) // filepath.Join()函数用于将目录和文件名合成一个路径
	}

	//lgLogger.Logger.Info("load config:" + configPath)
	fmt.Println("load config:" + configPath) // 打印日志

	v := viper.New()            // viper.New()函数用于创建一个新的viper对象 viper是一个配置管理工具
	v.SetConfigFile(configPath) // SetConfigFile()函数用于设置配置文件的路径
	v.SetConfigType("yaml")     // SetConfigType()函数用于设置配置文件的类型

	if err := v.ReadInConfig(); err != nil { // ReadInConfig()函数用于读取配置文件
		//lgLogger.Logger.Error("read config failed: ", zap.String("err", err.Error()))
		fmt.Println("read config failed: ", zap.String("err", err.Error()))
		panic(err)
	}

	if err := v.Unmarshal(&conf); err != nil { // Unmarshal()函数用于将配置文件中的数据映射到结构体中
		//lgLogger.Logger.Error("config parse failed: ", zap.String("err", err.Error()))
		fmt.Println("config parse failed: ", zap.String("err", err.Error()))
	}

	v.WatchConfig()                            // WatchConfig()函数用于监控配置文件的变化
	v.OnConfigChange(func(in fsnotify.Event) { // OnConfigChange()函数用于配置文件发生变化时的回调函数 fsnotify.Event用于获取文件系统事件
		//lgLogger.Logger.Info("", zap.String("config file changed:", in.Name))
		fmt.Println("", zap.String("config file changed:", in.Name)) //zap.String()函数用于构造一个zap.Field类型的键值对
		defer func() {                                               // defer用于延迟执行函数
			if err := recover(); err != nil { // recover()函数用于捕获异常
				//lgLogger.Logger.Error("config file changed err:", zap.Any("err", err))
				fmt.Println("config file changed err:", zap.Any("err", err))
			}
		}()
		if err := v.Unmarshal(&conf); err != nil {
			//lgLogger.Logger.Error("config parse failed: ", zap.String("err", err.Error()))
			fmt.Println("config parse failed: ", zap.String("err", err.Error()))
		}
	})
	lgConfig.Conf = conf
}
