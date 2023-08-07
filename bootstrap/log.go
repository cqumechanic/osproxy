package bootstrap

import (
	"os" // os是一个操作系统库
	"path/filepath"
	"strconv" // strconv是一个字符串转换库
	"sync"
	"time"

	"github.com/gin-gonic/gin"           // gin是一个web框架
	"github.com/qinguoyi/osproxy/config" // config是一个配置文件库
	"github.com/qinguoyi/osproxy/utils"  // utils是一个工具库
	"go.uber.org/zap"                    // zap是一个日志库
	"go.uber.org/zap/zapcore"            // zapcore是zap的核心库 有什么主要函数:
	"gopkg.in/natefinch/lumberjack.v2"   // lumberjack是一个日志切割库
)

const loggerKey = iota

var (
	level    zapcore.Level // zap 日志等级
	options  []zap.Option  // zap 配置项
	lgLogger = new(LangGoLogger)
)

// LangGoLogger 自定义Logger结构
type LangGoLogger struct {
	Logger *zap.Logger
	Once   *sync.Once
}

// newLangGoLogger .
func newLangGoLogger() *LangGoLogger {
	return &LangGoLogger{
		Logger: &zap.Logger{},
		Once:   &sync.Once{},
	}
}

// NewLogger 生成新Logger
func NewLogger() *LangGoLogger {
	if lgLogger.Logger != nil {
		return lgLogger
	} else {
		lgLogger = newLangGoLogger()             // newLangGoLogger()函数用于创建一个新的LangGoLogger对象
		lgLogger.initLangGoLogger(lgConfig.Conf) // lgConfig是全局变量，所以调用方直接声明变量就能访问到
		return lgLogger
	}
}

// initLangGoLogger 初始化全局log
func (lg *LangGoLogger) initLangGoLogger(conf *config.Configuration) {
	lg.Once.Do(
		func() {
			lg.Logger = initializeLog(conf)
		},
	)
}

// NewContext 给指定的context添加字段 这里的loggerKey是全局变量，所以调用方直接声明变量就能访问到
func (lg *LangGoLogger) NewContext(ctx *gin.Context, fields ...zapcore.Field) {
	ctx.Set(strconv.Itoa(loggerKey), lg.WithContext(ctx).With(fields...))
}

// WithContext 从指定的context返回一个zap实例
func (lg *LangGoLogger) WithContext(ctx *gin.Context) *zap.Logger {
	if ctx == nil {
		return lg.Logger
	}
	l, _ := ctx.Get(strconv.Itoa(loggerKey)) // Get()函数用于获取指定键的值 Itoa()函数用于将数字转换为字符串
	ctxLogger, ok := l.(*zap.Logger)
	if ok {
		return ctxLogger
	}
	return lg.Logger
}

func initializeLog(conf *config.Configuration) *zap.Logger {
	// 创建根目录
	createRootDir(conf)

	// 设置日志等级
	setLogLevel(conf) // setLogLevel()函数用于设置日志等级

	if conf.Log.ShowLine {
		options = append(options, zap.AddCaller()) // AddCaller()函数用于添加调用者信息  append()函数用于向切片中追加元素
	}

	// 初始化zap
	return zap.New(getZapCore(conf), options...) // New()函数用于创建一个新的zap对象,需要两个参数，分别是zapcore.Core对象和zap.Option对象
}

func createRootDir(conf *config.Configuration) {
	logFileDir := conf.Log.RootDir
	if !filepath.IsAbs(logFileDir) { // filepath.IsAbs()函数用于判断路径是否为绝对路径
		logFileDir = filepath.Join(rootPath, logFileDir)
	}

	if ok, _ := utils.Exists(logFileDir); !ok { // Exists()函数用于判断文件或目录是否存在
		_ = os.Mkdir(conf.Log.RootDir, os.ModePerm) //Mkdir()函数用于创建目录,参数1是目录名，参数2是权限，返回值是error ModePerm是0777
	}
}

func setLogLevel(conf *config.Configuration) { // logLevel有7种等级，具体参考zapcore.Level
	switch conf.Log.Level {
	case "debug":
		level = zap.DebugLevel
		options = append(options, zap.AddStacktrace(level))
	case "info":
		level = zap.InfoLevel
	case "warn":
		level = zap.WarnLevel
	case "error":
		level = zap.ErrorLevel
		options = append(options, zap.AddStacktrace(level))
	case "dpanic":
		level = zap.DPanicLevel
	case "panic":
		level = zap.PanicLevel
	case "fatal":
		level = zap.FatalLevel
	default:
		level = zap.InfoLevel
	}
}

func getZapCore(conf *config.Configuration) zapcore.Core { // zapcore.Core对象用于配置zap对象,zap.Core需要三个参数，分别是编码器，日志输出位置，日志等级
	// 编码器代表日志的输出格式，具体有两种格式，一种是json格式，一种是console格式，console格式是指输出到控制台,json格式是指输出到文件
	var encoder zapcore.Encoder

	// 调整编码器默认配置 输出内容
	encoderConfig := zap.NewProductionEncoderConfig()                                        // NewProductionEncoderConfig()函数用于创建一个新的zapcore.EncoderConfig对象.EncoderConfig对象用于配置zapcore.Encoder对象.Encoder对象用于配置zapcore.Core对象
	encoderConfig.EncodeTime = func(time time.Time, encoder zapcore.PrimitiveArrayEncoder) { // EncodeTime()函数用于设置时间格式.PrimitiveArrayEncoder对象用于配置zapcore.Encoder对象,time.Time对象用于表示时间,格式为2006-01-02 15:04:05.000
		encoder.AppendString(time.Format("[" + "2006-01-02 15:04:05.000" + "]"))
	}
	encoderConfig.EncodeLevel = func(l zapcore.Level, encoder zapcore.PrimitiveArrayEncoder) {
		encoder.AppendString(conf.App.Env + "." + l.String()) //Env是一个string类型的变量，用于表示环境
	}

	// 设置编码器，日志的输出格式
	if conf.Log.Format == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	// 同时输出到控制台和文件
	var multiWS zapcore.WriteSyncer // WriteSyncer对象用于配置zapcore.Core对象,用于将日志输出到指定的地方
	if conf.Log.EnableFile {        // EnableFile是一个bool类型的变量，用于判断是否启用日志文件，作者默认值是false
		multiWS = zapcore.NewMultiWriteSyncer(getLogWriter(conf), zapcore.AddSync(os.Stdout)) // NewMultiWriteSyncer()函数用于创建一个新的zapcore.WriteSyncer对象
		// getLogWriter()函数用于获取日志写入器,AddSync()函数用于将日志写入器添加到zapcore.WriteSyncer对象中
	} else {
		multiWS = zapcore.AddSync(os.Stdout) //os.Stdout是一个标准输出对象
	}

	return zapcore.NewCore(encoder, multiWS, level)
}

// 使用 lumberjack 作为日志写入器 lumberjack是一个日志切割库
func getLogWriter(conf *config.Configuration) zapcore.WriteSyncer {
	file := &lumberjack.Logger{ //
		Filename:   conf.Log.RootDir + "/" + conf.Log.Filename,
		MaxSize:    conf.Log.MaxSize,
		MaxBackups: conf.Log.MaxBackups, // MaxBackups是一个int类型的变量，用于表示最大备份数量
		MaxAge:     conf.Log.MaxAge,
		Compress:   conf.Log.Compress,
	}
	return zapcore.AddSync(file) // AddSync()函数用于将日志写入器添加到zapcore.WriteSyncer对象中
}
