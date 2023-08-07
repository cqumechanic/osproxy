package v0

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/qinguoyi/osproxy/app/models" // models包用于定义数据模型
	"github.com/qinguoyi/osproxy/app/pkg/base"
	"github.com/qinguoyi/osproxy/app/pkg/repo" // repo包用于定义仓库,仓库是对数据模型的操作,包含增删改查等操作
	"github.com/qinguoyi/osproxy/app/pkg/utils"
	"github.com/qinguoyi/osproxy/app/pkg/web"
	"github.com/qinguoyi/osproxy/bootstrap/plugins"
	"go.uber.org/zap"
)

/*
秒传及断点续传(上传)
*/

// ResumeHandler    秒传&断点续传
//
//	@Summary      秒传&断点续传
//	@Description  秒传&断点续传
//	@Tags         秒传
//	@Accept       application/json
//	@Param        RequestBody  body  models.ResumeReq  true  "秒传请求体"
//	@Produce      application/json
//	@Success      200  {object}  web.Response{data=[]models.ResumeResp}
//	@Router       /api/storage/v0/resume [post]
//
// ResumeHandler函数总共做了三件事：分别是1.参数校验；2.查询秒传数据；3.返回响应
func ResumeHandler(c *gin.Context) { //context是gin的上下文，它包含了请求和响应的信息，比如请求头、请求体、响应头、响应体等
	// 1.参数校验
	resumeReq := models.ResumeReq{} // ResumeReq是一个结构体，包含两个成员变量，一个是Data，一个是CheckPoint
	// 这里的models是一个包，包含了数据库scheme和requestModel
	if err := c.ShouldBindJSON(&resumeReq); err != nil { // ShouldBindJSON()函数用于将请求体绑定到结构体上,
		web.ParamsError(c, "参数有误")
		return
	}
	if len(resumeReq.Data) > utils.LinkLimit {
		web.ParamsError(c, fmt.Sprintf("判断文件秒传，数量不能超过%d个", utils.LinkLimit))
		return
	}

	var md5List []string               // md5List是一个切片，切片的元素是string类型
	md5MapName := map[string]string{}  // md5MapName是一个map，key是string类型，value是string类型
	for _, i := range resumeReq.Data { // 遍历resumeReq.Data，resumeReq.Data是一个切片，切片的元素是MD5Name类型,Md5Name是一个结构体，包含两个成员变量，一个是Md5，一个是Path
		md5MapName[i.Md5] = i.Path
		md5List = append(md5List, i.Md5) // append()函数用于向切片中添加元素，这里的切片是md5List，元素是i.Md5
	}

	// 这个循环的作用是：1.将resumeReq.Data中的数据添加到md5MapName中；2.将resumeReq.Data中的数据添加到md5List中

	md5List = utils.RemoveDuplicates(md5List) // RemoveDuplicates()函数用于去重

	md5MapResp := map[string]*models.ResumeResp{} // md5MapResp是一个map，key是string类型，value是ResumeResp类型的指针
	// md5MapResp是一个map，key是string类型，value是ResumeResp类型的指针
	for _, md5 := range md5List { // 遍历md5List，md5List是一个切片，切片的元素是string类型
		tmp := models.ResumeResp{
			Uid: "",
			Md5: md5,
		}
		md5MapResp[md5] = &tmp
	} // 这个循环的作用是：1.将md5List中的数据添加到md5MapResp中；

	// 秒传只看已上传且完整文件的数据
	lgDB := new(plugins.LangGoDB).Use("default").NewDB()                        //use()函数用于指定数据库，这里的数据库是default，NewDB()函数用于创建一个数据库实例
	resumeInfo, err := repo.NewMetaDataInfoRepo().GetResumeByMd5(lgDB, md5List) // resumeInfo是一个切片，切片的元素是MetaDataInfo类型
	// GetResumeByMd5()函数用于根据md5获取秒传数据 NewMetaDataInfoRepo()函数用于创建一个元数据信息仓库
	// 这里涉及了gorm的使用，gorm是一个orm框架，orm是对象关系映射，它的作用是将对象和数据库中的表进行映射，这样就可以通过对象来操作数据库了
	// gorm的使用方式是：1.创建一个数据库实例；2.创建一个仓库实例；3.调用仓库实例的方法，这些方法会调用数据库实例的方法，从而实现对数据库的操作
	if err != nil {
		lgLogger.WithContext(c).Error("查询秒传数据失败") // WithContext()函数用于创建一个日志实例，这里的日志实例是lgLogger
		web.InternalError(c, "")
		return
	}
	// 去重 这里去重与上面的去重不一样，上面的去重是对md5List进行去重，这里的去重是对resumeInfo进行去重
	// resumeInfo是一个切片，切片的元素是MetaDataInfo类型,秒传信息做MD5验证后，如果有重复的MD5，只保留一个
	md5MapMetaInfo := map[string]models.MetaDataInfo{}
	for _, resume := range resumeInfo {
		if _, ok := md5MapMetaInfo[resume.Md5]; !ok { // 如果md5MapMetaInfo中没有resume.Md5这个key，那么就将resume.Md5作为key，resume作为value，添加到md5MapMetaInfo中
			// ok是一个bool类型的值，如果md5MapMetaInfo中有resume.Md5这个key，那么ok为true，否则为false,ok是如何判断的呢？ok的值是通过map的key来判断的，如果map中有这个key，那么ok为true，否则为false
			// map是一种数据结构，它的特点是：1.无序；2.线程不安全；3.可以用于多个goroutine之间的数据传递，他可以返回多个值，第一个值是key对应的value，第二个值是key是否存在
			md5MapMetaInfo[resume.Md5] = resume
		}
	} // 这个循环的作用是：1.将resumeInfo中的数据添加到md5MapMetaInfo中，如果有重复的MD5，只保留一个

	var newMetaDataList []models.MetaDataInfo
	for _, resume := range resumeReq.Data { // 遍历resumeReq.Data，resumeReq.Data是一个切片，切片的元素是MD5Name类型
		if _, ok := md5MapMetaInfo[resume.Md5]; !ok { // 如果md5MapMetaInfo中没有resume.Md5这个key，那么就跳过这个循环
			continue
		}
		// 相同数据上传需要复制一份数据
		uid, _ := base.NewSnowFlake().NextId() // NewSnowFlake()函数用于创建一个雪花算法实例，NextId()函数用于获取下一个id
		now := time.Now()
		newMetaDataList = append(newMetaDataList,
			models.MetaDataInfo{
				UID:         uid,
				Bucket:      md5MapMetaInfo[resume.Md5].Bucket,
				Name:        filepath.Base(resume.Path), // filepath.Base()函数用于获取路径的最后一个元素
				StorageName: md5MapMetaInfo[resume.Md5].StorageName,
				Address:     md5MapMetaInfo[resume.Md5].Address,
				Md5:         resume.Md5,
				MultiPart:   false,
				StorageSize: md5MapMetaInfo[resume.Md5].StorageSize,
				Status:      1,
				ContentType: md5MapMetaInfo[resume.Md5].ContentType,
				CreatedAt:   &now,
				UpdatedAt:   &now,
			})
		md5MapResp[resume.Md5].Uid = fmt.Sprintf("%d", uid)
	} // 这个循环的作用是：newMetaDataList中添加数据，md5MapResp中添加数据

	if len(newMetaDataList) != 0 { // 如果newMetaDataList的长度不为0，那么就将newMetaDataList中的数据落到数据库中
		if err := repo.NewMetaDataInfoRepo().BatchCreate(lgDB, &newMetaDataList); err != nil {
			// BatchCreate()函数用于批量创建元数据信息，这里的批量创建是指一次性创建多条数据
			// NewMetaDataInfoRepo()函数用于创建一个元数据信息仓库
			lgLogger.WithContext(c).Error("秒传批量落数据库失败，详情：", zap.Any("err", err.Error()))
			web.InternalError(c, "内部异常")
			return
		}
	}
	lgRedis := new(plugins.LangGoRedis).NewRedis()
	for _, metaDataCache := range newMetaDataList { // 遍历newMetaDataList，newMetaDataList是一个切片，切片的元素是MetaDataInfo类型
		// newMetaDataList是最后要落到数据库中的数据，这里的遍历是为了将newMetaDataList中的数据写入redis中
		b, err := json.Marshal(metaDataCache) // Marshal()函数用于将结构体转换成json字符串
		if err != nil {
			lgLogger.WithContext(c).Warn("秒传数据，写入redis失败")
		}
		lgRedis.SetNX(context.Background(), fmt.Sprintf("%d-meta", metaDataCache.UID), b, 5*60*time.Second)
	}

	var respList []models.ResumeResp
	for _, resp := range md5MapResp {
		respList = append(respList, *resp)
	}
	web.Success(c, respList)
	return
}
