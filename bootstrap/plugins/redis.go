package plugins

import (
	"context"
	"sync"

	"github.com/go-redis/redis/extra/redisotel"
	"github.com/go-redis/redis/v8"
	"github.com/qinguoyi/osproxy/bootstrap"
	"github.com/qinguoyi/osproxy/config"
	"go.uber.org/zap"
)

var lgRedis = new(LangGoRedis)

type LangGoRedis struct {
	Once        *sync.Once
	RedisClient *redis.Client // Client是一个redis客户端，它实现了redis的命令,包含两个成员变量，一个是Options，一个是Ring
}

func (lg *LangGoRedis) NewRedis() *redis.Client { // NewRedis()函数用于创建一个新的redis对象，返回值是一个redis对象，这个对象是一个指针
	if lgRedis.RedisClient != nil {
		return lgRedis.RedisClient
	} else {
		return lg.New().(*redis.Client)
	}
}

func newLangGoRedis() *LangGoRedis {
	return &LangGoRedis{
		RedisClient: &redis.Client{},
		Once:        &sync.Once{},
	}
}

func (lg *LangGoRedis) Name() string {
	return "Redis"
}

func (lg *LangGoRedis) New() interface{} { // New()函数用于初始化插件资源,返回值是一个interface{}类型的对象
	lgRedis = newLangGoRedis()
	lgRedis.initRedis(bootstrap.NewConfig(""))
	return lgRedis.RedisClient
}

func (lg *LangGoRedis) Health() {
	if err := lgRedis.RedisClient.Ping(context.Background()).Err(); err != nil { // Ping()函数用于检查redis是否可用，参数是一个context对象，返回值是一个error，Background()函数用于创建一个空的context对象
		bootstrap.NewLogger().Logger.Error("redis connect failed, err:", zap.Any("err", err))
		panic(err)
	}
}

func (lg *LangGoRedis) Close() {
	if lg.RedisClient == nil {
		return
	} else {
		if err := lg.RedisClient.Close(); err != nil {
			bootstrap.NewLogger().Logger.Error("redis close failed, err:", zap.Any("err", err))
		}
	}
}

// Flag .
func (lg *LangGoRedis) Flag() bool { return true }

func init() {
	p := &LangGoRedis{}
	RegisteredPlugin(p)
}

func (lg *LangGoRedis) initRedis(conf *config.Configuration) {
	lg.Once.Do(func() {
		client := redis.NewClient(&redis.Options{
			Addr:     conf.Redis.Host + ":" + conf.Redis.Port,
			Password: conf.Redis.Password, // no password set
			DB:       conf.Redis.DB,       // use default DB
		})

		// redis链路追踪相关
		client.AddHook(redisotel.TracingHook{}) // AddHook()函数用于添加钩子函数 钩子函数是一种回调函数，它将在特定事件发生时被调用
		// AddHook() 参数是一个钩子函数，钩子函数是一个接口，它有一个方法，参数是一个redis命令，返回值是一个redis命令
		// redisotel.TracingHook()函数用于创建一个新的redis命令钩子函数
		lgRedis.RedisClient = client // lgRedis是全局变量，所以调用方直接声明变量就能访问到
	})

}
