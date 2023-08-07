package app

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/qinguoyi/osproxy/app/pkg/base"
	"github.com/qinguoyi/osproxy/app/pkg/event/dispatch"
	"github.com/qinguoyi/osproxy/config"
	"go.uber.org/zap"
)

// App 应用结构体
type App struct {
	conf    *config.Configuration
	logger  *zap.Logger
	httpSrv *http.Server
}

func NewHttpServer(
	conf *config.Configuration,
	router *gin.Engine,
) *http.Server { // Server是http的核心结构体，包含了路由、中间件等信息
	return &http.Server{
		Addr:    ":" + conf.App.Port, // Addr是http服务的地址，这里的地址是从配置文件中获取的
		Handler: router,              // Handler是http服务的路由，这里的路由是gin的实例
	}
}

func NewApp(
	conf *config.Configuration,
	logger *zap.Logger,
	httpSrv *http.Server,
) *App {
	return &App{
		conf:    conf,
		logger:  logger,
		httpSrv: httpSrv,
	}
}

// RunServer 启动服务
func (a *App) RunServer() {
	// 启动应用
	a.logger.Info("start app ...")
	if err := a.Run(); err != nil {
		panic(err)
	}

	// service register
	go base.NewServiceRegister().HeartBeat()

	// 启动 任务
	a.logger.Info("start task ...")
	p, consumers := dispatch.RunTask() // RunTask()函数用于启动任务，返回值是一个生产者和一个消费者的切片

	// 等待中断信号以优雅地关闭应用
	quit := make(chan os.Signal) // make()函数用于创建一个信号通道,channel是一种数据结构，它的特点是：1.先进先出；2.线程安全；3.可以用于多个goroutine之间的数据传递
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// 关闭任务
	log.Printf("stop task ...")
	dispatch.StopTask(p, consumers)

	// 设置 5 秒的超时时间
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 关闭应用
	log.Printf("shutdown app ...")
	if err := a.Stop(ctx); err != nil {
		panic(err)
	}
}

// Run 启动服务
func (a *App) Run() error {
	// 启动 http server
	go func() {
		if err := a.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed { // ListenAndServe()函数用于启动http服务
			panic(err)
		}

	}()
	return nil
}

// Stop 停止服务
func (a *App) Stop(ctx context.Context) error {
	// 关闭 http server
	if err := a.httpSrv.Shutdown(ctx); err != nil {
		return err
	}
	return nil
}
