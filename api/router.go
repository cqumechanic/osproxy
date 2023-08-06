package api

import (
	"github.com/gin-gonic/gin"                   // gin是一个web框架
	v0 "github.com/qinguoyi/osproxy/api/v0"      // api
	"github.com/qinguoyi/osproxy/app/middleware" // 中间件
	"github.com/qinguoyi/osproxy/bootstrap"
	"github.com/qinguoyi/osproxy/config"
	"github.com/qinguoyi/osproxy/docs"
	gs "github.com/swaggo/gin-swagger" // swagger gs是gin-swagger的简写,golang包名简写规则：如果包名是一个单词，那么就用这个单词的小写；如果包名是多个单词，那么就用首字母缩写，比如gin-swagger就是gs
	"github.com/swaggo/gin-swagger/swaggerFiles"
)

func NewRouter(
	conf *config.Configuration,
	lgLogger *bootstrap.LangGoLogger,
) *gin.Engine { // Engine是gin的核心结构体，包含了路由、中间件等信息
	if conf.App.Env == "prod" { // 如果是生产环境，那么就设置gin的运行模式为生产模式,prod模式下，不输出调试信息，只输出错误信息
		gin.SetMode(gin.ReleaseMode) // 设置gin的运行模式,gin.releaseMode是生产模式，gin.debugMode是调试模式
	}
	router := gin.New() // New()函数用于创建一个gin的实例，是一个Engine类型的指针

	// middleware
	corsM := middleware.NewCors()                        // NewCors()函数用于创建一个跨域中间件
	traceL := middleware.NewTrace(lgLogger)              // NewTrace()函数用于创建一个trace中间件
	requestL := middleware.NewRequestLog(lgLogger)       // NewRequestLog()函数用于创建一个request-log中间件
	panicRecover := middleware.NewPanicRecover(lgLogger) // NewPanicRecover()函数用于创建一个panic-recover中间件

	// 跨域 trace-id 日志
	router.Use(corsM.Handler(), traceL.Handler(), requestL.Handler(), panicRecover.Handler()) // Use()函数用于注册中间件
	// 中间件在请求处理函数之前执行，所以中间件可以在请求处理函数之前做一些前置处理，也可以在请求处理函数之后做一些后置处理，比如日志记录、权限验证、异常处理等
	// 在gin中，中间件是一个HandlerFunc，它的定义如下： type HandlerFunc func(*Context)。中间件的参数是一个Context指针，返回值是一个空接口
	// 在java中，中间件是一个Filter（拦截器），它的定义如下： public void doFilter(ServletRequest request, ServletResponse response, FilterChain chain) throws IOException, ServletException

	// 静态资源
	router.StaticFile("/assets", "../../static/image/back.png") // StaticFile()函数用于注册静态资源
	// StaticFile()函数的第一个参数是路由，第二个参数是静态资源的路径，这里的静态资源是一个图片，路径是相对于项目根目录的，所以是../../static/image/back.png
	// 静态资源包括图片、css、js等，这些资源不需要经过处理，直接返回给客户端即可，所以不需要注册路由，只需要注册静态资源即可

	// swag docs
	docs.SwaggerInfo.BasePath = "/"
	router.GET("/swagger/*any", gs.WrapHandler(swaggerFiles.Handler)) // GET请求，路由为/swagger/*any，处理函数为swaggerFiles.Handler
	// gs.WrapHandler()函数用于将swaggerFiles.Handler包装成一个gin的处理函数，这样就可以注册到路由中了

	// 动态资源 注册 api 分组路由
	setApiGroupRoutes(router)

	return router
}

func setApiGroupRoutes(
	router *gin.Engine,
) *gin.RouterGroup {
	// api group
	group := router.Group("/api/storage/v0") // Group()函数用于创建一个路由组，路由组的路径是/api/storage/v0
	{
		// 这里是restful风格的api，比如GET请求，路由为/api/storage/v0/ping，处理函数为PingHandler
		// restful风格的api，是一种软件架构风格，它的特点是：1.每一个URI代表一种资源；2.客户端和服务器之间，传递这种资源的某种表现层；3.客户端通过四个HTTP动词，对服务器端资源进行操作，实现"表现层状态转化"。
		//health
		group.GET("/ping", v0.PingHandler)
		group.GET("/health", v0.HealthCheckHandler)

		// resume
		// 秒传是指：如果文件已经上传过了，那么就不需要再次上传了，直接返回文件的url即可
		// 断点续传是指：如果文件已经上传了一部分，那么就不需要再次上传这部分了，直接上传剩下的部分即可
		group.POST("/resume", v0.ResumeHandler)        // 秒传及断点续传
		group.GET("/checkpoint", v0.CheckPointHandler) // 断点续传

		// link
		group.POST("/link/upload", v0.UploadLinkHandler)     // upload link 上传链接
		group.POST("/link/download", v0.DownloadLinkHandler) // DownloadLinkHandler 下载链接

		// proxy
		group.GET("/proxy", v0.IsOnCurrentServerHandler) // IsOnCurrentServerHandler()函数用于判断是否在当前服务器

		// upload
		group.PUT("/upload", v0.UploadSingleHandler)          // PUT请求，路由为/upload，处理函数为UploadSingleHandler
		group.PUT("/upload/multi", v0.UploadMultiPartHandler) // PUT请求，路由为/upload/multi，处理函数为UploadMultiPartHandler
		group.PUT("/upload/merge", v0.UploadMergeHandler)     // PUT请求，路由为/upload/merge，处理函数为UploadMergeHandler

		//download
		group.GET("/download", v0.DownloadHandler)

	}
	return group
}
