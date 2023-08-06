package utils

import "time"

const (
	Scheme                = "http"
	WorkID                = "workId"
	LinkLimit             = 50 // 限制秒传数量
	EncryKey              = "*&^@#$storage"
	LocalStore            = "/home/yt/go/src/Go项目/osproxy/storage/localstore"
	ServiceRedisPrefix    = "service:proxy" // redis前缀
	ServiceRedisTTl       = time.Second * 3 * 60
	S3StoragePutThreadNum = 10
	MultiPartDownload     = 10
)

// 任务类型
const (
	TaskPartMerge  = "partMerge"
	TaskPartDelete = "partDelete"
)

// 任务状态
const (
	TaskStatusUndo    = 0
	TaskStatusRunning = 1
	TaskStatusFinish  = 2
	TaskStatusError   = 99
)

// worker和队列
const (
	MaxWorker = 100
	MaxQueue  = 200
)

const CompensationTotal = 5 // 补偿次数总量
