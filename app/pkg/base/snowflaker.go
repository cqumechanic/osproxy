package base

/*
雪花算法
*/

import (
	"context" // context包用于传递请求的上下文，它允许在处理链中传递请求作用域、取消信号和截止时间
	"errors"  // errors包实现了创建错误值的函数
	"strconv" // strconv包实现了基本数据类型和其字符串表示的相互转换
	"sync"    // sync包提供了基本的同步基元，如互斥锁
	"time"    // time包提供了时间的显示和测量用的函数，日历的计算采用的是公历

	"github.com/qinguoyi/osproxy/app/pkg/utils"
	"github.com/qinguoyi/osproxy/bootstrap/plugins"
)

var (
	snowFlake *Snowflake
	once      sync.Once
)

const (
	twepoch            = int64(1417937700000) // Unix纪元时间戳,1970-01-01 00:00:00?
	workerIdBits       = uint(5)              // 机器ID所占位数
	datacenterBits     = uint(5)              // 数据中心ID所占位数
	maxWorkerId        = int64(-1) ^ (int64(-1) << workerIdBits)
	maxDatacenterId    = int64(-1) ^ (int64(-1) << datacenterBits)
	sequenceBits       = uint(12)                                     // 序列号所占位数
	workerIdShift      = sequenceBits                                 // 机器ID左移位数
	datacenterIdShift  = sequenceBits + workerIdBits                  // 数据中心ID左移位数
	timestampLeftShift = sequenceBits + workerIdBits + datacenterBits // 时间戳左移位数
	sequenceMask       = int64(-1) ^ (int64(-1) << sequenceBits)      // 生成序列号的掩码 int64(-1) ^ (int64(-1) << sequenceBits) = 4095 int64(-1) = -1 <<符号是位运算符，左移运算符，左移n位就是乘以2的n次方
)

type Snowflake struct {
	mu            sync.Mutex
	lastTimestamp int64 // 上一次生成ID的时间戳
	workerId      int64 // 机器ID
	datacenterId  int64 // 数据中心ID
	sequence      int64 // 序列号
}

// InitSnowFlake .
func InitSnowFlake() {
	// get local ip
	ip, err := GetClientIp() // GetClientIp()函数用于获取客户端ip
	if err != nil {
		panic(err)
	}

	// get workId from redis
	var workId int64
	ctx := context.Background()                    // Background()函数用于创建一个空的context对象
	lgRedis := new(plugins.LangGoRedis).NewRedis() // new(plugins.LangGoRedis)用于创建一个LangGoRedis对象，NewRedis()函数用于创建一个新的redis对象，返回值是一个redis对象，这个对象是一个指针

	ipExist := lgRedis.Exists(ctx, ip).Val() // Exists()函数用于判断key是否存在，参数是一个context对象，返回值是一个int64类型的值
	if ipExist == 1 {
		curWorkId, err := lgRedis.Get(ctx, ip).Result() // Get()函数用于获取key的值，参数是一个context对象，返回值是一个string类型的值 Result()函数返回值是一个string类型的值
		if err != nil {
			panic(err)
		}
		workId, err = strconv.ParseInt(curWorkId, 10, 64) // ParseInt()函数用于将字符串转换为int64类型的值，参数是一个字符串，一个int类型的进制，一个int类型的位数
		if err != nil {
			panic(err)
		}
	} else {
		newWorkId, err := lgRedis.Incr(ctx, utils.WorkID).Result() // Incr()函数用于将key的值加1，参数是一个context对象，返回值是一个int64类型的值 有关reids的incr命令，参考https://www.runoob.com/redis/strings-incr.html
		if err != nil {
			panic(err)
		}
		lgRedis.Set(ctx, ip, newWorkId, -1) // Set()函数用于设置key的值，参数是一个context对象，一个key，一个value，一个过期时间
		workId = newWorkId
	}
	once.Do(func() {
		res, err := newSnowFlake(workId, 0)
		if err != nil {
			panic(err)
		}
		snowFlake = res
	})
}

func newSnowFlake(workerId, datacenterId int64) (*Snowflake, error) {
	if workerId < 0 || workerId > maxWorkerId {
		return nil, errors.New("worker id out of range")
	}
	if datacenterId < 0 || datacenterId > maxDatacenterId {
		return nil, errors.New("datacenter id out of range")
	}
	return &Snowflake{
		lastTimestamp: 0,
		workerId:      workerId,
		datacenterId:  datacenterId,
		sequence:      0,
	}, nil
}

// NewSnowFlake .
func NewSnowFlake() *Snowflake {
	if snowFlake == nil {
		once.Do(func() {
			res, err := newSnowFlake(10, 10)
			if err != nil {
				panic(err)
			}
			snowFlake = res
		})
	}
	return snowFlake
}

// NextId . 有时钟回拨问题
func (sf *Snowflake) NextId() (int64, error) {
	sf.mu.Lock()
	defer sf.mu.Unlock()

	timestamp := time.Now().UnixNano() / 1000000 // UnixNano()函数用于获取当前时间的纳秒数，返回值是一个int64类型的值 time.Now()函数用于获取当前时间，返回值是一个time.Time类型的值

	if timestamp < sf.lastTimestamp { // 如果当前时间小于上一次生成ID的时间戳，说明时钟回拨
		return 0, errors.New("clock moved backwards")
	}

	if timestamp == sf.lastTimestamp { // 如果当前时间等于上一次生成ID的时间戳，说明在同一毫秒内
		sf.sequence = (sf.sequence + 1) & sequenceMask // &
		if sf.sequence == 0 {                          // 如果当前序列号等于0，说明当前毫秒内的序列号已经达到最大值,即12位的序列号已经达到最大值，4095，需要等待下一毫秒
			// 时钟回拨
			for timestamp <= sf.lastTimestamp {
				timestamp = time.Now().UnixNano() / 1000000
			}
		}
	} else { // 如果当前时间大于上一次生成ID的时间戳，说明不在同一毫秒内
		sf.sequence = 0
	}

	sf.lastTimestamp = timestamp
	// 相当于
	id := ((timestamp - twepoch) << timestampLeftShift) | (sf.datacenterId << datacenterIdShift) | (sf.workerId << workerIdShift) | sf.sequence

	return id, nil
}
