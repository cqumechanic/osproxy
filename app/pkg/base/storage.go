package base

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/qinguoyi/osproxy/app/models"
	"github.com/qinguoyi/osproxy/app/pkg/utils"
	"github.com/qinguoyi/osproxy/bootstrap"
	"github.com/qinguoyi/osproxy/bootstrap/plugins"
)

var lgLogger *bootstrap.LangGoLogger

func GetExtension(filename string) string {
	ext := filepath.Ext(filename)
	if ext == "" {
		return ""
	}
	return strings.ToLower(ext[1:])
}

// selectBucketBySuffix .
func selectBucketBySuffix(filename string) string {
	suffix := GetExtension(filename)
	if suffix == "" {
		return ""
	}
	switch suffix {
	case "jpg", "jpeg", "png", "gif", "bmp":
		return "image"
	case "mp4", "avi", "wmv", "mpeg":
		return "video"
	case "mp3", "wav", "flac":
		return "audio"
	case "pdf", "doc", "docx", "ppt", "pptx", "xls", "xlsx":
		return "doc"
	case "zip", "rar", "tar", "gz", "7z":
		return "archive"
	default:
		return "unknown"
	}
}

func CheckValid(uidStr, date, expireStr string) (int64, error, string) {
	// check
	uid, err := strconv.ParseInt(uidStr, 10, 64)
	if err != nil {
		return 0, err, fmt.Sprintf("uid参数有误，详情:%s", err)
	}

	loc, _ := time.LoadLocation("Local")
	t, err := time.ParseInLocation("2006-01-02T15:04:05Z", date, loc)
	if err != nil {
		return uid, err, fmt.Sprintf("时间参数转换失败，详情:%s", err)
	}

	expire, err := strconv.ParseInt(expireStr, 10, 64)
	if err != nil {
		return uid, err, fmt.Sprintf("expire参数有误，详情:%s", err)
	}
	now := time.Now().In(loc)
	duration := now.Sub(t)
	if int64(duration.Seconds()) > expire {
		return uid, errors.New("链接时间已过期"), "链接时间已过期"
	}
	return uid, nil, ""
}

// GenUploadSingle .
// GenUploadSingle()函数用于生成上传链接
func GenUploadSingle(filename string, expire int, respChan chan models.GenUploadResp,
	metaDataInfoChan chan models.MetaDataInfo, wg *sync.WaitGroup) {
	defer wg.Done()
	bucket := selectBucketBySuffix(filename) // selectBucketBySuffix()函数用于根据文件后缀选择bucket,将文件分类后
	uid, err := NewSnowFlake().NextId()      // NewSnowFlake()函数用于创建一个雪花算法实例，NextId()函数用于生成一个id
	if err != nil {
		//lgLogger.WithContext(c).Error("雪花算法生成ID失败，详情：", zap.Any("err", err.Error()))
		return
	}
	uidStr := strconv.FormatInt(uid, 10) // FormatInt()函数用于将int64类型的数字转换成字符串
	name := filepath.Base(filename)      // Base()函数用于获取文件名
	// sprintf()函数用于格式化字符串，这里的格式化字符串是%s.%s，第一个%s是uidStr，第二个%s是GetExtension(filename)
	storageName := fmt.Sprintf("%s.%s", uidStr, GetExtension(filename)) // GetExtension()函数用于获取文件后缀
	objectName := fmt.Sprintf("%s/%s", bucket, storageName)             // objectName是一个字符串，格式是bucket/storageName

	// 在本地创建uid的目录
	if err := os.MkdirAll(path.Join(utils.LocalStore, uidStr), 0755); err != nil {
		//lgLogger.WithContext(c).Error("创建本地目录失败，详情：", zap.Any("err", err.Error()))
		return
	}

	// 生成加密query query是一个字符串，格式是uidStr-date-expire-signature
	date := time.Now().Format("2006-01-02T15:04:05Z")
	signature := decode(fmt.Sprintf("%s-%d", date, expire))            // decode()函数用于解密，解密具体是怎么实现的呢？这里的解密是将数字转换成字符串
	queryString := GenUploadSignature(uidStr, date, expire, signature) // GenUploadSignature()函数用于生成加密query
	// single、merge、multi区别是什么？single是单文件上传，merge是合并文件，multi是多文件上传
	single := fmt.Sprintf("/api/storage/v0/upload?%s", queryString)
	multi := fmt.Sprintf("/api/storage/v0/upload/multi?%s", queryString)
	merge := fmt.Sprintf("/api/storage/v0/upload/merge?%s", queryString)
	respChan <- models.GenUploadResp{ // respChan是一个通道，通道的元素是GenUploadResp类型，这一行代码的作用是将GenUploadResp类型的数据添加到respChan中
		// 通道的作用是：1.用于多个goroutine之间的数据传递；2.用于goroutine和主线程之间的数据传递
		// 为什么要用通道呢？因为通道是线程安全的，而且通道可以返回多个值，第一个值是key对应的value，第二个值是key是否存在
		// 在这里使用通道的原因是：1.在这里使用通道可以将数据传递给主线程，主线程可以将数据写入数据库
		Uid: uidStr,
		Url: &models.UrlResult{
			Single: single,
			Multi: &models.MultiUrlResult{
				Merge:  merge,
				Upload: multi,
			},
		},
		Path: filename,
	}
	// 生成DB信息
	now := time.Now()
	metaDataInfoChan <- models.MetaDataInfo{
		UID:         uid,
		Bucket:      bucket,
		Name:        name,
		StorageName: storageName,
		Address:     objectName,
		MultiPart:   false,
		Status:      -1,
		ContentType: "application/octet-stream", //先按照文件后缀占位，后面文件上传会覆盖
		CreatedAt:   &now,
		UpdatedAt:   &now,
	}
	return
}

func GenDownloadSingle(meta models.MetaDataInfo, expire string, respChan chan models.GenDownloadResp,
	wg *sync.WaitGroup) {
	defer wg.Done()
	uid := meta.UID
	bucketName := meta.Bucket
	srcName := meta.Name
	objectName := meta.StorageName

	// 生成加密query
	date := time.Now().Format("2006-01-02T15:04:05Z")
	signature := decode(fmt.Sprintf("%s-%s-%s-%s", date, expire,
		bucketName, objectName))
	queryString := GenDownloadSignature(uid, srcName, bucketName, objectName, expire, date, signature)
	url := fmt.Sprintf("/api/storage/v0/download?%s", queryString)
	info := models.GenDownloadResp{
		Uid: fmt.Sprintf("%d", uid),
		Url: url,
		Meta: models.MetaInfo{
			SrcName: srcName,
			DstName: objectName,
			Height:  meta.Height,
			Width:   meta.Width,
			Md5:     meta.Md5,
			Size:    fmt.Sprintf("%d", meta.StorageSize),
		},
	}
	respChan <- info
	// 写入redis
	key := fmt.Sprintf("%d-%s", uid, expire)
	lgRedis := new(plugins.LangGoRedis).NewRedis()
	b, err := json.Marshal(info)
	if err != nil {
	}
	lgRedis.SetNX(context.Background(), key, b, 5*60*time.Second)
}

func GetRange(rangeHeader string, size int64) (int64, int64) {
	var start, end int64
	if rangeHeader == "" {
		end = size - 1
	} else {
		split := strings.Split(rangeHeader, "=")
		ranges := strings.Split(split[1], "-")
		start, _ = strconv.ParseInt(ranges[0], 10, 64)
		if ranges[1] != "" {
			end, _ = strconv.ParseInt(ranges[1], 10, 64)
		}
		if end >= size || end == 0 {
			end = size - 1
		}
	}
	return start, end
}
