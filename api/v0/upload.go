package v0

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/qinguoyi/osproxy/app/models"
	"github.com/qinguoyi/osproxy/app/pkg/base"
	"github.com/qinguoyi/osproxy/app/pkg/repo"
	"github.com/qinguoyi/osproxy/app/pkg/storage"
	"github.com/qinguoyi/osproxy/app/pkg/thirdparty"
	"github.com/qinguoyi/osproxy/app/pkg/utils"
	"github.com/qinguoyi/osproxy/app/pkg/web"
	"github.com/qinguoyi/osproxy/bootstrap"
	"github.com/qinguoyi/osproxy/bootstrap/plugins"
	"go.uber.org/zap"
)

/*
对象上传
*/

// UploadSingleHandler    上传单个文件
//
//	@Summary      上传单个文件
//	@Description  上传单个文件
//	@Tags         上传
//	@Accept       multipart/form-data
//	@Param        file       formData  file    true  "上传的文件"
//	@Param        uid        query     string  true  "文件uid"
//	@Param        md5        query     string  true  "md5"
//	@Param        date       query     string  true  "链接生成时间"
//	@Param        expire     query     string  true  "过期时间"
//	@Param        signature  query     string  true  "签名"
//	@Produce      application/json
//	@Success      200  {object}  web.Response
//	@Router       /api/storage/v0/upload [put]
func UploadSingleHandler(c *gin.Context) {
	uidStr := c.Query("uid")
	md5 := c.Query("md5")
	date := c.Query("date")
	expireStr := c.Query("expire")
	signature := c.Query("signature")

	uid, err, errorInfo := base.CheckValid(uidStr, date, expireStr) // CheckValid()函数用于校验参数
	if err != nil {
		web.ParamsError(c, errorInfo)
		return
	}

	if !base.CheckUploadSignature(date, expireStr, signature) {
		web.ParamsError(c, "签名校验失败")
		return
	}

	file, err := c.FormFile("file") // FormFile()函数用于获取上传的文件
	// 具体点说，FormFile()函数用于获取表单数据项，表单数据项是指表单中的一个数据项，比如<input type="file" name="file" />，这里的name就是表单数据项
	if err != nil {
		web.ParamsError(c, fmt.Sprintf("解析文件参数失败，详情：%s", err))
		return
	}

	// 判断记录是否存在
	// 为什么上传文件的时候
	lgDB := new(plugins.LangGoDB).Use("default").NewDB()
	metaData, err := repo.NewMetaDataInfoRepo().GetByUid(lgDB, uid) // GetByUid()函数用于根据uid获取元数据信息
	if err != nil {
		web.NotFoundResource(c, "当前上传链接无效，uid不存在")
		return
	}

	dirName := path.Join(utils.LocalStore, uidStr)
	// 判断是否上传过，md5
	resumeInfo, err := repo.NewMetaDataInfoRepo().GetResumeByMd5(lgDB, []string{md5})
	// GetResumeByMd5()函数用于根据md5获取秒传数据，返回值是一个切片，切片的元素是MetaDataInfo类型
	if err != nil {
		lgLogger.WithContext(c).Error("查询文件是否已上传失败")
		web.InternalError(c, "")
		return
	}
	if len(resumeInfo) != 0 {
		now := time.Now()
		if err := repo.NewMetaDataInfoRepo().Updates(lgDB, uid, map[string]interface{}{
			// Updates()函数用于更新元数据信息，元数据信息是指文件的元数据信息，比如文件的md5、文件的大小、文件的类型等
			"bucket":       resumeInfo[0].Bucket,
			"storage_name": resumeInfo[0].StorageName,
			"address":      resumeInfo[0].Address,
			"md5":          md5,
			"storage_size": resumeInfo[0].StorageSize,
			"multi_part":   false,
			"status":       1,
			"updated_at":   &now,
			"content_type": resumeInfo[0].ContentType,
		}); err != nil {
			lgLogger.WithContext(c).Error("上传完更新数据失败")
			web.InternalError(c, "上传完更新数据失败")
			return
		}
		// 这里为什要删除目录呢？因为这里是上传单个文件，而不是上传多个文件，所以这里的目录是空的，所以可以删除
		// 上传文件的过程是：1.创建一个目录；2.将文件存储到目录中；3.将目录中的文件上传到minio中；4.删除目录
		// 创建目录是指在本地创建一个目录，将文件存储到目录中，这里的目录是uidStr
		// minio 是一个对象存储服务器，它的作用是存储对象，对象是指文件，比如图片、视频、音频等
		if err := os.RemoveAll(dirName); err != nil {
			lgLogger.WithContext(c).Error(fmt.Sprintf("删除目录失败，详情%s", err.Error()))
			web.InternalError(c, fmt.Sprintf("删除目录失败，详情%s", err.Error()))
			return
		}
		// 写入redis的目的是什么呢？写入redis的目的是为了加快访问速度，因为redis是内存数据库，访问速度比较快
		//
		// 首次写入redis 元数据
		lgRedis := new(plugins.LangGoRedis).NewRedis()
		metaCache, err := repo.NewMetaDataInfoRepo().GetByUid(lgDB, uid)
		if err != nil {
			lgLogger.WithContext(c).Error("上传数据，查询数据元信息失败")
			web.InternalError(c, "内部异常")
			return
		}
		b, err := json.Marshal(metaCache)
		if err != nil {
			lgLogger.WithContext(c).Warn("上传数据，写入redis失败")
		}
		lgRedis.SetNX(context.Background(), fmt.Sprintf("%s-meta", uidStr), b, 5*60*time.Second)
		// SetNX()函数用于向redis中写入数据 SetNX()函数的第一个参数是上下文，第二个参数是key，第三个参数是value，第四个参数是过期时间
		// context.Background()函数用于创建一个上下文，上下文是gin的上下文，它包含了请求和响应的信息，比如请求头、请求体、响应头、响应体等

		web.Success(c, "")
		return
	}
	// 上面的代码是判断文件是否已上传，如果已上传，那么就直接返回，不再上传，如果没有上传，那么就继续上传

	//

	// 判断是否在本地，什么叫本地呢？本地是指本地服务器，本地服务器是指部署了osproxy的服务器
	// 分布式系统是指多台服务器组成的系统，分布式系统的特点是：1.多台服务器之间是对等的，没有主从之分；2.多台服务器之间是通过网络进行通信的
	// 在这个项目中，分布式系统是指多台部署了osproxy的服务器组成的系统，这些服务器之间是对等的，没有主从之分，这些服务器之间是通过网络进行通信的
	if _, err := os.Stat(dirName); os.IsNotExist(err) { // Stat()函数用于获取文件信息，IsNotExist()函数用于判断文件是否存在
		// 不在本地，询问集群内其他服务并转发
		serviceList, err := base.NewServiceRegister().Discovery() // Discovery()函数用于发现其他服务
		if err != nil || serviceList == nil {
			lgLogger.WithContext(c).Error("发现其他服务失败")
			web.InternalError(c, "发现其他服务失败")
			return
		}
		var wg sync.WaitGroup
		var ipList []string //
		ipChan := make(chan string, len(serviceList))
		for _, service := range serviceList {
			wg.Add(1)
			go func(ip string, port string, ipChan chan string, wg *sync.WaitGroup) {
				// 这里的匿名函数的任务是：
				// 1.调用thirdparty.NewStorageService().Locate()函数，这个函数的作用是定位其他服务；2.将定位的结果写入ipChan中；
				// 3.将wg的值减1
				defer wg.Done()
				res, err := thirdparty.NewStorageService().Locate(utils.Scheme, ip, port, uidStr) // Locate()函数用于定位其他服务
				if err != nil {
					fmt.Print(err.Error())
					return
				}
				ipChan <- res
			}(service.IP, service.Port, ipChan, &wg)
		}
		wg.Wait()
		close(ipChan)
		for re := range ipChan {
			ipList = append(ipList, re)
		}
		if len(ipList) == 0 {
			lgLogger.WithContext(c).Error("发现其他服务失败")
			web.InternalError(c, "发现其他服务失败")
			return
		}
		proxyIP := ipList[0] // 为什么是ipList[0]作为代理ip呢？
		// 找到其他服务器上的后,转发
		_, _, _, err = thirdparty.NewStorageService().UploadForward(c, utils.Scheme, proxyIP,
			bootstrap.NewConfig("").App.Port, uidStr, true) // UploadForward()函数用于上传文件，这里的上传是指将文件上传到云端
		if err != nil {
			lgLogger.WithContext(c).Error("上传单文件，转发失败")
			web.InternalError(c, err.Error())
			return
		}
		web.Success(c, "")
		return
	}
	// 在本地
	fileName := path.Join(utils.LocalStore, uidStr, metaData.StorageName)
	out, err := os.Create(fileName)
	if err != nil {
		lgLogger.WithContext(c).Error("本地创建文件失败")
		web.InternalError(c, "本地创建文件失败")
		return
	}
	src, err := file.Open()
	if err != nil {
		lgLogger.WithContext(c).Error("打开本地文件失败")
		web.InternalError(c, "打开本地文件失败")
		return
	}
	if _, err = io.Copy(out, src); err != nil {
		lgLogger.WithContext(c).Error("请求数据存储到文件失败")
		web.InternalError(c, "请求数据存储到文件失败")
		return
	}
	// 校验md5
	md5Str, err := base.CalculateFileMd5(fileName)
	if err != nil {
		lgLogger.WithContext(c).Error(fmt.Sprintf("生成md5失败，详情%s", err.Error()))
		web.InternalError(c, err.Error())
		return
	}
	if md5Str != md5 {
		web.ParamsError(c, fmt.Sprintf("校验md5失败，计算结果:%s, 参数:%s", md5Str, md5))
		return
	}
	// 上传到minio
	contentType, err := base.DetectContentType(fileName) // DetectContentType()函数用于判断文件的类型
	if err != nil {
		lgLogger.WithContext(c).Error("判断文件content-type失败")
		web.InternalError(c, "判断文件content-type失败")
		return
	}
	if err := storage.NewStorage().Storage.PutObject(metaData.Bucket, metaData.StorageName, fileName, contentType); err != nil {
		lgLogger.WithContext(c).Error("上传到minio失败")
		web.InternalError(c, "上传到minio失败")
		return
	}
	// 更新元数据，元数据存储在数据库中
	now := time.Now()
	fileInfo, _ := os.Stat(fileName)
	if err := repo.NewMetaDataInfoRepo().Updates(lgDB, metaData.UID, map[string]interface{}{
		"md5":          md5Str,
		"storage_size": fileInfo.Size(),
		"multi_part":   false,
		"status":       1,
		"updated_at":   &now,
		"content_type": contentType,
	}); err != nil {
		lgLogger.WithContext(c).Error("上传完更新数据失败")
		web.InternalError(c, "上传完更新数据失败")
		return
	}
	_, _ = out.Close(), src.Close()

	if err := os.RemoveAll(dirName); err != nil {
		lgLogger.WithContext(c).Error(fmt.Sprintf("删除目录失败，详情%s", err.Error()))
		web.InternalError(c, fmt.Sprintf("删除目录失败，详情%s", err.Error()))
		return
	}

	// 首次写入redis 元数据
	lgRedis := new(plugins.LangGoRedis).NewRedis()
	metaCache, err := repo.NewMetaDataInfoRepo().GetByUid(lgDB, uid)
	if err != nil {
		lgLogger.WithContext(c).Error("上传数据，查询数据元信息失败")
		web.InternalError(c, "内部异常")
		return
	}
	b, err := json.Marshal(metaCache)
	if err != nil {
		lgLogger.WithContext(c).Warn("上传数据，写入redis失败")
	}
	lgRedis.SetNX(context.Background(), fmt.Sprintf("%s-meta", uidStr), b, 5*60*time.Second)
	// setNX是否自带锁？
	// SetNX是原子操作的，这里的SetNX是指向redis中写入数据，如果写入成功，那么就返回true，否则返回false

	web.Success(c, "")
	return
}

// UploadMultiPartHandler    上传分片文件
//
//	@Summary      上传分片文件
//	@Description  上传分片文件
//	@Tags         上传
//	@Accept       multipart/form-data
//	@Param        file       formData  file    true  "上传的文件"
//	@Param        uid        query     string  true  "文件uid"
//	@Param        md5        query     string  true  "md5"
//	@Param        chunkNum   query     string  true  "当前分片id"
//	@Param        date       query     string  true  "链接生成时间"
//	@Param        expire     query     string  true  "过期时间"
//	@Param        signature  query     string  true  "签名"
//	@Produce      application/json
//	@Success      200  {object}  web.Response
//	@Router       /api/storage/v0/upload/multi [put]
func UploadMultiPartHandler(c *gin.Context) {
	// 整个这个处理函数只在分片上传的时候调用，调用的时候，用了goroutine。分片上传的时候，涉及锁的操作是redis，在这里不再需要调用goroutine
	// 相当于一个请求，一个goroutine，一个goroutine，一个请求
	uidStr := c.Query("uid")
	md5 := c.Query("md5")
	chunkNumStr := c.Query("chunkNum")
	date := c.Query("date")
	expireStr := c.Query("expire")
	signature := c.Query("signature")

	uid, err, errorInfo := base.CheckValid(uidStr, date, expireStr)
	if err != nil {
		web.ParamsError(c, errorInfo)
		return
	}

	chunkNum, err := strconv.ParseInt(chunkNumStr, 10, 64) // chunkNum指的是当前分片id
	if err != nil {
		web.ParamsError(c, errorInfo)
		return
	}

	if !base.CheckUploadSignature(date, expireStr, signature) {
		web.ParamsError(c, "签名校验失败")
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		web.ParamsError(c, fmt.Sprintf("解析文件参数失败，详情：%s", err))
		return
	}

	// 判断记录是否存在
	lgDB := new(plugins.LangGoDB).Use("default").NewDB()
	metaData, err := repo.NewMetaDataInfoRepo().GetByUid(lgDB, uid)
	if err != nil {
		web.NotFoundResource(c, "当前上传链接无效，uid不存在")
		return
	}
	// 判断当前分片是否已上传
	var lgRedis = new(plugins.LangGoRedis).NewRedis()
	ctx := context.Background()
	createLock := base.NewRedisLock(&ctx, lgRedis, fmt.Sprintf("multi-part-%d-%d-%s", uid, chunkNum, md5))
	// NewRedisLock()函数用于创建一个redis锁，redis锁是一种分布式锁，分布式锁是指多个goroutine之间的锁，这些goroutine之间是通过网络进行通信的
	if flag, err := createLock.Acquire(); err != nil || !flag { // Acquire()函数用于获取锁
		lgLogger.WithContext(c).Error("上传多文件抢锁失败")
		web.InternalError(c, "上传多文件抢锁失败")
		return
	}
	partInfo, err := repo.NewMultiPartInfoRepo().GetPartInfo(lgDB, uid, chunkNum, md5)
	// GetPartInfo()函数用于获取分片信息
	// 在这里，查询是否成功，如果成功，那么就说明当前分片已上传，那么就不再上传，如果不成功，那么就说明当前分片没有上传，那么就继续上传
	if err != nil {
		lgLogger.WithContext(c).Error("多文件上传，查询分片信息失败")
		web.InternalError(c, "内部异常")
		return
	}
	if len(partInfo) != 0 {
		web.Success(c, "")
		return
	}
	_, _ = createLock.Release() // Release()函数用于释放锁

	// 判断是否在本地
	dirName := path.Join(utils.LocalStore, uidStr)
	if _, err := os.Stat(dirName); os.IsNotExist(err) {
		// 不在本地，询问集群内其他服务并转发
		serviceList, err := base.NewServiceRegister().Discovery()
		if err != nil || serviceList == nil {
			lgLogger.WithContext(c).Error("发现其他服务失败")
			web.InternalError(c, "发现其他服务失败")
			return
		}
		var wg sync.WaitGroup
		var ipList []string
		ipChan := make(chan string, len(serviceList))
		for _, service := range serviceList {
			wg.Add(1)
			go func(ip string, port string, ipChan chan string, wg *sync.WaitGroup) {
				defer wg.Done()
				res, err := thirdparty.NewStorageService().Locate(utils.Scheme, ip, port, uidStr)
				if err != nil {
					fmt.Print(err.Error())
					return
				}
				ipChan <- res
			}(service.IP, service.Port, ipChan, &wg)
		}
		wg.Wait()
		close(ipChan)
		for re := range ipChan {
			ipList = append(ipList, re)
		}
		if len(ipList) == 0 {
			lgLogger.WithContext(c).Error("发现其他服务失败")
			web.InternalError(c, "发现其他服务失败")
			return
		}
		proxyIP := ipList[0]
		_, _, _, err = thirdparty.NewStorageService().UploadForward(c, utils.Scheme, proxyIP,
			bootstrap.NewConfig("").App.Port, uidStr, false)
		if err != nil {
			lgLogger.WithContext(c).Error("多文件上传，转发失败")
			web.InternalError(c, err.Error())
			return
		}
		web.Success(c, "")
		return
	}

	// 在本地
	fileName := path.Join(utils.LocalStore, uidStr, fmt.Sprintf("%d_%d", uid, chunkNum))
	out, err := os.Create(fileName)
	if err != nil {
		lgLogger.WithContext(c).Error("本地创建文件失败")
		web.InternalError(c, "本地创建文件失败")
		return
	}
	defer func(out *os.File) {
		_ = out.Close()
	}(out)
	src, err := file.Open()
	if err != nil {
		lgLogger.WithContext(c).Error("打开本地文件失败")
		web.InternalError(c, "打开本地文件失败")
		return
	}
	if _, err = io.Copy(out, src); err != nil {
		lgLogger.WithContext(c).Error("请求数据存储到文件失败")
		web.InternalError(c, "请求数据存储到文件失败")
		return
	}
	// 校验md5
	md5Str, err := base.CalculateFileMd5(fileName)
	if err != nil {
		lgLogger.WithContext(c).Error(fmt.Sprintf("生成md5失败，详情%s", err.Error()))
		web.InternalError(c, err.Error())
		return
	}
	if md5Str != md5 {
		lgLogger.WithContext(c).Error(fmt.Sprintf("校验md5失败，计算结果:%s, 参数:%s", md5Str, md5))
		web.ParamsError(c, fmt.Sprintf("校验md5失败，计算结果:%s, 参数:%s", md5Str, md5))
		return
	}
	// 上传到minio
	contentType := "application/octet-stream"
	if err := storage.NewStorage().Storage.PutObject(metaData.Bucket, fmt.Sprintf("%d_%d", uid, chunkNum),
		fileName, contentType); err != nil {
		lgLogger.WithContext(c).Error("上传到minio失败")
		web.InternalError(c, "上传到minio失败")
		return
	}

	// 创建元数据
	now := time.Now()
	fileInfo, _ := os.Stat(fileName)
	if err := repo.NewMultiPartInfoRepo().Create(lgDB, &models.MultiPartInfo{ // Create()函数用于创建分片信息
		StorageUid:   uid,
		ChunkNum:     int(chunkNum),
		Bucket:       metaData.Bucket,
		StorageName:  fmt.Sprintf("%d_%d", uid, chunkNum),
		StorageSize:  fileInfo.Size(),
		PartFileName: fmt.Sprintf("%d_%d", uid, chunkNum),
		PartMd5:      md5Str,
		Status:       1,
		CreatedAt:    &now,
		UpdatedAt:    &now,
	}); err != nil {
		lgLogger.WithContext(c).Error("上传完更新数据失败")
		web.InternalError(c, "上传完更新数据失败")
		return
	}
	web.Success(c, "")
	return
}

// UploadMergeHandler     合并分片文件
//
//	@Summary      合并分片文件
//	@Description  合并分片文件
//	@Tags         上传
//	@Accept       multipart/form-data
//	@Param        uid        query  string  true  "文件uid"
//	@Param        md5        query  string  true  "md5"
//	@Param        num        query  string  true  "总分片数量"
//	@Param        size       query  string  true  "文件总大小"
//	@Param        date       query  string  true  "链接生成时间"
//	@Param        expire     query  string  true  "过期时间"
//	@Param        signature  query  string  true  "签名"
//	@Produce      application/json
//	@Success      200  {object}  web.Response
//	@Router       /api/storage/v0/upload/merge [put]
func UploadMergeHandler(c *gin.Context) {
	//uid从哪里来的呢？uid是从前端传过来的，前端传过来的时候，是在前端调用UploadMultiPartHandler()函数的时候传过来的
	uidStr := c.Query("uid")
	md5 := c.Query("md5")
	numStr := c.Query("num")
	size := c.Query("size")
	date := c.Query("date")
	expireStr := c.Query("expire")
	signature := c.Query("signature")

	uid, err, errorInfo := base.CheckValid(uidStr, date, expireStr)
	if err != nil {
		web.ParamsError(c, errorInfo)
		return
	}

	num, err := strconv.ParseInt(numStr, 10, 64)
	if err != nil {
		web.ParamsError(c, errorInfo)
		return
	}

	if !base.CheckUploadSignature(date, expireStr, signature) {
		web.ParamsError(c, "签名校验失败")
		return
	}

	// 判断记录是否存在
	//
	lgDB := new(plugins.LangGoDB).Use("default").NewDB()
	metaData, err := repo.NewMetaDataInfoRepo().GetByUid(lgDB, uid)
	if err != nil {
		web.NotFoundResource(c, "当前合并链接无效，uid不存在")
		return
	}

	// 判断分片数量是否一致
	var multiPartInfoList []models.MultiPartInfo
	if err := lgDB.Model(&models.MultiPartInfo{}).Where(
		"storage_uid = ? and status = ?", uid, 1).Order("chunk_num ASC").Find(&multiPartInfoList).Error; err != nil {
		lgLogger.WithContext(c).Error("查询分片数据失败")
		web.InternalError(c, "查询分片数据失败")
		return
	}

	if num != int64(len(multiPartInfoList)) {
		// 创建脏数据删除任务
		msg := models.MergeInfo{
			StorageUid: uid,
			ChunkSum:   num,
		}
		b, err := json.Marshal(msg)
		if err != nil {
			lgLogger.WithContext(c).Error("消息struct转成json字符串失败", zap.Any("err", err.Error()))
			web.InternalError(c, "分片数量和整体数量不一致，创建删除任务失败")
			return
		}
		newModelTask := models.TaskInfo{
			Status:    utils.TaskStatusUndo,
			TaskType:  utils.TaskPartDelete,
			ExtraData: string(b),
		}
		if err := repo.NewTaskRepo().Create(lgDB, &newModelTask); err != nil {
			lgLogger.WithContext(c).Error("分片数量和整体数量不一致，创建删除任务失败", zap.Any("err", err.Error()))
			web.InternalError(c, "分片数量和整体数量不一致，创建删除任务失败")
			return
		}
		web.ParamsError(c, "分片数量和整体数量不一致")
		return
	}

	// 判断是否在本地
	dirName := path.Join(utils.LocalStore, uidStr)
	if _, err := os.Stat(dirName); os.IsNotExist(err) {
		// 不在本地，询问集群内其他服务并转发
		serviceList, err := base.NewServiceRegister().Discovery()
		if err != nil || serviceList == nil {
			lgLogger.WithContext(c).Error("发现其他服务失败")
			web.InternalError(c, "发现其他服务失败")
			return
		}
		var wg sync.WaitGroup
		var ipList []string
		ipChan := make(chan string, len(serviceList))
		for _, service := range serviceList {
			wg.Add(1)
			go func(ip string, port string, ipChan chan string, wg *sync.WaitGroup) {
				defer wg.Done()
				res, err := thirdparty.NewStorageService().Locate(utils.Scheme, ip, port, uidStr)
				if err != nil {
					return
				}
				ipChan <- res
			}(service.IP, service.Port, ipChan, &wg)
		}
		wg.Wait()
		close(ipChan)
		for re := range ipChan {
			ipList = append(ipList, re)
		}
		if len(ipList) == 0 {
			lgLogger.WithContext(c).Error("发现其他服务失败")
			web.InternalError(c, "发现其他服务失败")
			return
		}
		proxyIP := ipList[0]
		_, _, _, err = thirdparty.NewStorageService().MergeForward(c, utils.Scheme, proxyIP,
			bootstrap.NewConfig("").App.Port, uidStr)
		if err != nil {
			lgLogger.WithContext(c).Error("合并文件，转发失败")
			web.InternalError(c, err.Error())
			return
		}
		web.Success(c, "")
		return
	}
	// 获取文件的content-type
	firstPart := multiPartInfoList[0]
	partName := path.Join(utils.LocalStore, fmt.Sprintf("%d", uid), firstPart.PartFileName)
	contentType, err := base.DetectContentType(partName)
	if err != nil {
		lgLogger.WithContext(c).Error("判断文件content-type失败")
		web.InternalError(c, "判断文件content-type失败")
		return
	}

	// 更新metadata的数据
	now := time.Now()
	if err := repo.NewMetaDataInfoRepo().Updates(lgDB, metaData.UID, map[string]interface{}{
		"part_num":     int(num),
		"md5":          md5,
		"storage_size": size,
		"multi_part":   true,
		"status":       1,
		"updated_at":   &now,
		"content_type": contentType,
	}); err != nil {
		lgLogger.WithContext(c).Error("上传完更新数据失败")
		web.InternalError(c, "上传完更新数据失败")
		return
	}
	// 创建合并任务
	msg := models.MergeInfo{
		StorageUid: uid,
		ChunkSum:   num,
	}
	b, err := json.Marshal(msg)
	if err != nil {
		lgLogger.WithContext(c).Error("消息struct转成json字符串失败", zap.Any("err", err.Error()))
		web.InternalError(c, "创建合并任务失败")
		return
	}
	newModelTask := models.TaskInfo{
		Status:    utils.TaskStatusUndo,
		TaskType:  utils.TaskPartMerge,
		ExtraData: string(b),
	}
	if err := repo.NewTaskRepo().Create(lgDB, &newModelTask); err != nil {
		lgLogger.WithContext(c).Error("创建合并任务失败", zap.Any("err", err.Error()))
		web.InternalError(c, "创建合并任务失败")
		return
	}

	// 首次写入redis 元数据和分片信息
	lgRedis := new(plugins.LangGoRedis).NewRedis()
	metaCache, err := repo.NewMetaDataInfoRepo().GetByUid(lgDB, uid)
	if err != nil {
		lgLogger.WithContext(c).Error("上传数据，查询数据元信息失败")
		web.InternalError(c, "内部异常")
		return
	}
	b, err = json.Marshal(metaCache)
	if err != nil {
		lgLogger.WithContext(c).Warn("上传数据，写入redis失败")
	}
	lgRedis.SetNX(context.Background(), fmt.Sprintf("%s-meta", uidStr), b, 5*60*time.Second)

	var multiPartInfoListCache []models.MultiPartInfo
	if err := lgDB.Model(&models.MultiPartInfo{}).Where(
		"storage_uid = ? and status = ?", uid, 1).Order("chunk_num ASC").Find(&multiPartInfoListCache).Error; err != nil {
		lgLogger.WithContext(c).Error("上传数据，查询分片数据失败")
		web.InternalError(c, "查询分片数据失败")
		return
	}
	// 写入redis
	b, err = json.Marshal(multiPartInfoListCache)
	if err != nil {
		lgLogger.WithContext(c).Warn("上传数据，写入redis失败")
	}
	lgRedis.SetNX(context.Background(), fmt.Sprintf("%s-multiPart", uidStr), b, 5*60*time.Second)

	web.Success(c, "")
	return
}
