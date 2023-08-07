package base

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/qinguoyi/osproxy/app/pkg/utils"
	"github.com/qinguoyi/osproxy/bootstrap"
	"github.com/qinguoyi/osproxy/bootstrap/plugins"
)

// 这个文件的作用是：1.服务注册；2.心跳检测；3.服务发现，主打一个分布式的不同服务器的打招呼

type serviceRegister struct {
	client *redis.Client
}

func NewServiceRegister() *serviceRegister {
	lgRedis := new(plugins.LangGoRedis).NewRedis()
	return &serviceRegister{
		client: lgRedis,
	}
}

type Service struct {
	IP        string
	Port      string
	CreatedAt int64 // 创建时间
}

// Register 服务注册
func (s *serviceRegister) Register() {
	ip, err := GetOutBoundIP()
	if err != nil {
		panic(err)
	}
	jsonByte, err := json.Marshal(Service{
		IP:        ip,
		Port:      bootstrap.NewConfig("").App.Port,
		CreatedAt: time.Now().Unix(),
	})
	if err != nil {
		panic(err)
	}
	err = s.client.HSet(context.Background(), utils.ServiceRedisPrefix, ip, jsonByte).Err()
	if err != nil {
		panic(err)
	}
}

// HeartBeat 心跳检测
func (s *serviceRegister) HeartBeat() {
	timer := time.NewTimer(1 * time.Nanosecond)
	defer timer.Stop()

	ip, err := GetClientIp()
	bootstrap.NewLogger().Logger.Info(fmt.Sprintf("当前上报ip:%s", ip))
	if err != nil {
		panic(err)
	}
	for {
		select {
		case <-timer.C:
			urlStr := "/api/storage/v0/health"
			req := Request{
				Url: fmt.Sprintf("%s://%s:%s%s", utils.Scheme, "127.0.0.1", bootstrap.NewConfig("").App.Port,
					urlStr),
				Body:   io.NopCloser(strings.NewReader("")),
				Method: "GET",
				Params: map[string]string{},
			}
			_, _, _, err := Ask(req)
			if err == nil {
				jsonByte, _ := json.Marshal(Service{
					IP:        ip,
					Port:      bootstrap.NewConfig("").App.Port,
					CreatedAt: time.Now().Unix(),
				})
				s.client.HSet(context.Background(), utils.ServiceRedisPrefix, ip, jsonByte)
			} else {
				bootstrap.NewLogger().Logger.Error(fmt.Sprintf("注册失败:%s", err.Error()))
				for {
					_, _, _, err := Ask(req)
					if err != nil {
						bootstrap.NewLogger().Logger.Error(fmt.Sprintf("注册失败:%s", err.Error()))
					} else {
						jsonByte, _ := json.Marshal(Service{
							IP:        ip,
							Port:      bootstrap.NewConfig("").App.Port,
							CreatedAt: time.Now().Unix(),
						})
						s.client.HSet(context.Background(), utils.ServiceRedisPrefix, ip, jsonByte)
						break
					}
					time.Sleep(3 * time.Second)
				}
			}

		}
		timer.Reset(utils.ServiceRedisTTl)
	}
}

// Discovery 服务发现
func (s *serviceRegister) Discovery() ([]*Service, error) {
	result := s.client.HGetAll(context.Background(), utils.ServiceRedisPrefix) // HGetAll()函数用于获取redis中的所有数据
	if result.Err() == redis.Nil {
		return nil, nil
	}
	if result.Err() != nil {
		return nil, result.Err()
	}
	arr, err := result.Result()
	if err != nil {
		return nil, err
	}
	resp := make([]*Service, 0)
	boundary := time.Now().Add(-5 * time.Minute).Unix() // 5分钟之前的数据不要
	for _, value := range arr {
		var ser *Service
		err := json.Unmarshal([]byte(value), &ser)
		if err != nil {
			return nil, err
		}
		if ser.CreatedAt < boundary {
			_ = s.client.HDel(context.Background(), utils.ServiceRedisPrefix, ser.IP).Err()
			continue
		}
		resp = append(resp, ser)
	}
	return resp, nil
}
