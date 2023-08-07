package v0

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/qinguoyi/osproxy/app/pkg/web"
	"github.com/qinguoyi/osproxy/bootstrap"
	"github.com/qinguoyi/osproxy/bootstrap/plugins"
)

var lgLogger *bootstrap.LangGoLogger //LangGoLogger是一个结构体，包含两个成员变量，一个是Logger，一个是Once

// 不能提前创建，变量的初始化在main之前，导致lgDB为nil
//var lgDB = new(plugins.LangGoDB).NewDB()

// PingHandler 测试
//
//  @Summary      测试接口
//  @Description  测试接口
//  @Tags         测试
//  @Accept       application/json
//  @Produce      application/json
//  @Success      200  {object}  web.Response
//  @Router       /api/storage/v0/ping [get]
func PingHandler(c *gin.Context) { // PingHandler()函数用于处理GET请求，路由为/api/storage/v0/ping
	// PingHandler函数具体的逻辑包含两个测试，一个是数据库测试，一个是redis测试
	var lgDB = new(plugins.LangGoDB).Use("default").NewDB() // 这里的lgDB是局部变量，所以调用方需要传入参数
	// LangGoDB是一个结构体，包含两个成员变量，一个是DB，一个是Once
	// DB是一个sql.DB类型的指针，DB是gorm的底层，gorm是一个orm框架，orm是对象关系映射，它的作用是将对象和数据库中的表进行映射，这样就可以通过对象来操作数据库了

	var lgRedis = new(plugins.LangGoRedis).NewRedis()

	// DB Test
	lgDB.Exec("select now();") // Exec()函数用于执行sql语句，这里的sql语句是select now();，now()函数用于获取当前时间
	lgLogger.WithContext(c).Info("test router")

	// Redis Test
	err := lgRedis.Set(c, "key", "value", 0).Err() // Set()函数用于设置key-value，这里的key是key，value是value，0表示不设置过期时间
	if err != nil {
		panic(err)
	}
	val, err := lgRedis.Get(c, "key").Result() // Get()函数用于获取key对应的value，这里的key是key
	if err != nil {
		panic(err)
	}
	lgLogger.WithContext(c).Info(fmt.Sprintf("%v", val))
	web.Success(c, "Test Router...") // Success()函数用于返回成功的响应
	return
}

// HealthCheckHandler 健康检查
//
//  @Summary      健康检查
//  @Description  健康检查
//  @Tags         检查
//  @Accept       application/json
//  @Produce      application/json
//  @Success      200  {object}  web.Response
//  @Router       /api/storage/v0/health [get]
func HealthCheckHandler(c *gin.Context) {
	web.Success(c, "Health...")
	return
}
