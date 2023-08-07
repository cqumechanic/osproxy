package v0

import (
	"fmt"
	"os"
	"path"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/qinguoyi/osproxy/app/pkg/base"
	"github.com/qinguoyi/osproxy/app/pkg/utils"
	"github.com/qinguoyi/osproxy/app/pkg/web"
)

// IsOnCurrentServerHandler   .
//
//	@Summary      询问文件是否在当前服务
//	@Description  询问文件是否在当前服务
//	@Tags         proxy
//	@Accept       application/json
//	@Param        uid  query  string  true  "uid"
//	@Produce      application/json
//	@Success      200  {object}  web.Response
//	@Router       /api/storage/v0/proxy [get]
func IsOnCurrentServerHandler(c *gin.Context) {
	uidStr := c.Query("uid")
	_, err := strconv.ParseInt(uidStr, 10, 64) // ParseInt()函数用于将字符串转换成int64类型的数字
	if err != nil {
		web.ParamsError(c, fmt.Sprintf("uid参数有误，详情:%s", err))
		return
	}
	dirName := path.Join(utils.LocalStore, uidStr) // Join()函数用于将多个字符串拼接成一个字符串
	if _, err := os.Stat(dirName); os.IsNotExist(err) {
		web.NotFoundResource(c, "")
		return
	} else {
		ip, err := base.GetOutBoundIP()
		if err != nil {
			panic(err)
		}
		web.Success(c, ip)
		return
	}
}
