package base

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
)

// 这个文件的作用是：有关文件的操作
// 共4有个函数：
// 1.InTurnPrint()函数用于交替打印1-100；2.CalculateByteMd5()函数用于计算byte类型数据的md5值；
// 3.CalculateFileMd5()函数用于计算文件的md5值; 4.DetectContentType()函数用于检测文件的类型

// InTurnPrint .
// InTurnPrint()函数用于交替打印1-100
func InTurnPrint(filename string) string {
	// 分块计算，流式计算(避免打爆内存)，顺序合并，类似N个协程交替打印1-100
	goNum := 10
	var chanSlice []chan int
	for i := 0; i < goNum; i++ {
		chanSlice = append(chanSlice, make(chan int, 1))
	}

	var wg *sync.WaitGroup
	chanSlice[0] <- 1
	for i := 0; i < goNum; i++ {
		wg.Add(1)
		go func(i int, wg *sync.WaitGroup) {
			defer wg.Done()
			// part 计算
			// 当前块计算完后，需要等待前一个块合并到主哈希
			<-chanSlice[i]
			// 合并到主哈希
			chanSlice[i+1] <- 1
		}(i, wg)
	}
	return ""
}

// CalculateByteMd5 .
func CalculateByteMd5(b []byte) (string, error) {
	hash := md5.New()
	_, err := io.Copy(hash, bytes.NewReader(b))
	if err != nil {
		fmt.Println("io copy error")
	}
	md5Str := hex.EncodeToString(hash.Sum(nil)) // Sum()函数用于计算md5值,hex.EncodeToString()函数用于将md5值转换成字符串
	return md5Str, nil
}

// CalculateFileMd5 .
func CalculateFileMd5(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)
	hash := md5.New()
	_, err = io.Copy(hash, file)
	if err != nil {
		fmt.Println("io copy error")
	}
	md5Str := hex.EncodeToString(hash.Sum(nil))
	return md5Str, nil
}

func DetectContentType(fileName string) (string, error) {
	// 打开文件
	file, err := os.Open(fileName)
	if err != nil {
		return "", err
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	// 读取文件头部信息
	buf := make([]byte, 512)
	_, err = file.Read(buf)
	if err != nil {
		return "", err
	}
	contentType := http.DetectContentType(buf)
	return contentType, nil
}
