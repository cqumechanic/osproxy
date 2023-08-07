package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"mime/multipart" // multipart是一个包，包含了multipart的实现
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/qinguoyi/osproxy/app/models"
	"github.com/qinguoyi/osproxy/app/pkg/base"
)

func minH(a, b int64) int64 {
	if a <= b {
		return a
	} else {
		return b
	}
}

func main() {
	// 基础信息
	baseUrl := "http://127.0.0.1:8888"
	uploadFilePath := "../test/xxx.jpg"
	uploadFile := filepath.Base(uploadFilePath)

	// ##################### 获取上传连接 ###################
	fmt.Println("获取上传连接")
	urlStr := "/api/storage/v0/link/upload"
	body := map[string]interface{}{ // body是一个map，key是string类型，value是interface{}类型
		"filePath": []string{fmt.Sprintf("%s", uploadFile)}, // filePath是一个切片，切片的元素是string类型
		"expire":   86400,
	}
	jsonBytes, err := json.Marshal(body) // Marshal()函数用于将结构体转换成json字符串
	if err != nil {
		panic(err)
	}
	req := base.Request{
		Url:    fmt.Sprintf("%s%s", baseUrl, urlStr),
		Body:   io.NopCloser(strings.NewReader(string(jsonBytes))), // NopCloser()函数用于将io.Reader类型的数据转换成io.ReadCloser类型的数据
		Method: "POST",
		Params: map[string]string{},
	}
	_, data, _, err := base.Ask(req) // Ask()函数用于发送请求
	if err != nil {
		panic(err)
	}
	// data.Data是一个json字符串，格式是[{"uid":"xxx","url":{"single":"xxx","multi":{"upload":"xxx","merge":"xxx"}}}]
	var uploadLink []*models.GenUploadResp // uploadLink是一个切片，切片的元素是GenUploadResp类型
	if err := json.Unmarshal(data.Data, &uploadLink); err != nil {
		// Unmarshal()函数用于将json字符串转换成结构体,这里是将data.Data转换成uploadLink
		panic(err)
	}

	// ##################### 上传文件 ######################
	// +++++++ 单文件 ++++++

	filePath := uploadFilePath
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Failed to open file:", err)
		return
	}
	defer file.Close()
	md5Str, _ := base.CalculateFileMd5(filePath) // CalculateFileMd5()函数用于计算文件的md5值

	fileInfo, _ := os.Stat(filePath) // Stat()函数用于获取文件信息
	fileSize := fileInfo.Size()
	fmt.Println(fileSize)
	if fileSize <= 1024*1024*1 { // 如果文件大小小于1M，那么就是单文件上传
		fmt.Println("单文件上传")
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		// 打开文件
		defer func(srcFile multipart.File) {
			err := srcFile.Close()
			if err != nil {

			}
		}(file)

		// 创建表单数据项
		dst, err := writer.CreateFormFile("file", uploadFile)
		if err != nil {
			panic(err)
		}

		// 将文件内容写入表单数据项
		if _, err = io.Copy(dst, file); err != nil {
			panic(err)
		}
		err = writer.Close()
		if err != nil {
			panic(err)
		}
		u, err := url.Parse(uploadLink[0].Url.Single)
		if err != nil {
			panic(err)
		}
		query := u.Query()
		uidStr := base.Query(query, "uid")
		date := base.Query(query, "date")
		expireStr := base.Query(query, "expire")
		signature := base.Query(query, "signature")
		req := base.Request{
			Url:  fmt.Sprintf("%s%s", baseUrl, uploadLink[0].Url.Single),
			Body: io.NopCloser(body),
			HeaderSet: map[string]string{
				"Content-Type": writer.FormDataContentType(),
			},
			Method: "PUT",
			Params: map[string]string{"md5": md5Str, "uid": uidStr,
				"date": date, "expire": expireStr, "signature": signature},
		}
		_, _, _, err = base.Ask(req)
		if err != nil {
			panic(err)
		}
	} else { // 如果文件大小大于1M，那么就是多文件上传
		// +++++++ 多文件 ++++++
		// 分片上传
		fmt.Println("多文件上传")
		chunkSize := 1024.0 * 1024
		currentChunk := int64(1)
		totalChunk := int64(math.Ceil(float64(fileSize) / chunkSize)) // Ceil()函数用于向上取整
		var wg sync.WaitGroup
		ch := make(chan struct{}, 5)
		for currentChunk <= totalChunk { //这个循环的作用是：1.将文件分成多个分片；2.将分片上传到服务器

			start := (currentChunk - 1) * int64(chunkSize)
			end := minH(fileSize, start+int64(chunkSize))
			buffer := make([]byte, end-start)
			// 循环读取，会自动偏移
			n, err := file.Read(buffer) // Read()函数用于读取文件
			if err != nil && err != io.EOF {
				fmt.Println("读取文件长度失败", err)
				break
			}
			fmt.Println("当前read长度", n)
			md5Part, _ := base.CalculateByteMd5(buffer) // CalculateByteMd5()函数用于计算字节的md5值

			// 多协程上传
			ch <- struct{}{} // struct{}{}是一个空结构体，struct{}{}的作用是：1.用于多个goroutine之间的数据传递；2.用于goroutine和主线程之间的数据传递
			// ch是一个通道，通道的元素是struct{}{}类型，这一行代码的作用是将struct{}{}类型的数据添加到ch中
			wg.Add(1)
			go func(data []byte, md5V string, chunkNum int64, wg *sync.WaitGroup) { //这里的括号中的参数是goroutine的参数，这里的goroutine是匿名函数
				defer wg.Done()
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body) // NewWriter()函数用于创建一个multipart.Writer类型的数据
				// 创建表单数据项,表单数据项是指表单中的一个数据项，比如<input type="file" name="file" />，这里的name就是表单数据项
				// 表单数据项的作用是：1.用于上传文件；2.用于上传普通数据，在这里的作用是上传文件
				// 表单数据项的形式是：Content-Disposition: form-data; name="file"; filename="xxx.jpg"
				dst, err := writer.CreateFormFile("file",
					fmt.Sprintf("%s%d", uploadLink[0].Uid, chunkNum))
				if err != nil {
					panic(err)
				}

				// 将文件内容写入表单数据项
				if _, err = io.Copy(dst, bytes.NewReader(data)); err != nil { // NewReader()函数用于创建一个io.Reader类型的数据,这里的数据是data
					panic(err)
				}
				err = writer.Close()
				if err != nil {
					panic(err)
				}

				u, err := url.Parse(uploadLink[0].Url.Multi.Upload) // Parse()函数用于解析url
				if err != nil {
					panic(err)
				}
				query := u.Query() // Query()函数用于获取url中的query,query是一个字符串，格式是uidStr-date-expire-signature

				// 将query中的数据添加到params中
				uidStr := base.Query(query, "uid")
				date := base.Query(query, "date")
				expireStr := base.Query(query, "expire")
				signature := base.Query(query, "signature")
				req := base.Request{
					Url:  fmt.Sprintf("%s%s", baseUrl, uploadLink[0].Url.Multi.Upload),
					Body: io.NopCloser(body),
					HeaderSet: map[string]string{
						"Content-Type": writer.FormDataContentType(), // FormDataContentType()函数用于获取表单数据项的类型
					},
					Method: "PUT",
					Params: map[string]string{"uid": uidStr, "date": date, "expire": expireStr, "signature": signature,
						"md5": md5V, "chunkNum": fmt.Sprintf("%d", chunkNum)},
				}
				code, _, _, err := base.Ask(req)
				if err != nil {
					fmt.Println(code)
					fmt.Println(err)
				}

				<-ch // 从ch中取出一个元素
			}(buffer, md5Part, currentChunk, &wg) // 这里括号中的参数是goroutine的参数，与前面的参数对应
			currentChunk += 1
		}
		wg.Wait()

		// 合并
		u, err := url.Parse(uploadLink[0].Url.Multi.Merge) // Parse()函数用于解析url
		if err != nil {
			panic(err)
		}
		query := u.Query()
		uidStr := base.Query(query, "uid")
		date := base.Query(query, "date")
		expireStr := base.Query(query, "expire")
		signature := base.Query(query, "signature")
		req := base.Request{
			Url:       fmt.Sprintf("%s%s", baseUrl, uploadLink[0].Url.Multi.Merge),
			Body:      io.NopCloser(strings.NewReader("")),
			HeaderSet: map[string]string{},
			Method:    "PUT",
			Params: map[string]string{"uid": uidStr, "date": date, "expire": expireStr, "signature": signature,
				"md5": md5Str, "num": fmt.Sprintf("%d", totalChunk), "size": fmt.Sprintf("%d", fileSize)},
		}
		_, _, _, err = base.Ask(req)
		if err != nil {
			panic(err)
		}
	}

	// ##################### 获取下载链接 ###################
	fmt.Println("获取下载链接")

	urlStr = "/api/storage/v0/link/download"
	body = map[string]interface{}{
		"uid":    []string{uploadLink[0].Uid},
		"expire": 86400,
	}
	jsonBytes, err = json.Marshal(body)
	if err != nil {
		panic(err)
	}
	req = base.Request{
		Url:    fmt.Sprintf("%s%s", baseUrl, urlStr),
		Body:   io.NopCloser(strings.NewReader(string(jsonBytes))),
		Method: "POST",
		Params: map[string]string{},
	}
	_, data, _, err = base.Ask(req)
	if err != nil {
		panic(err)
	}
	var downloadLink []*models.GenDownloadResp
	if err := json.Unmarshal(data.Data, &downloadLink); err != nil {
		panic(err)
	}

	// ##################### 下载文件 ######################
	fmt.Println("下载文件")
	downloadUrl := downloadLink[0].Url
	u, err := url.Parse(downloadUrl)
	if err != nil {
		panic(err)
	}
	query := u.Query()
	uidStr := base.Query(query, "uid")
	name := base.Query(query, "name")
	date := base.Query(query, "date")
	expireStr := base.Query(query, "expire")
	signature := base.Query(query, "signature")
	bucketName := base.Query(query, "bucket")
	objectName := base.Query(query, "object")

	fileName := fmt.Sprintf("%d", time.Now().Unix())
	dourl := fmt.Sprintf("%s%s", baseUrl, downloadUrl)
	fmt.Printf("下载链接为：%s", dourl)
	// Create the file
	out, err := os.Create(fileName)
	if err != nil {
		panic(err)
	}

	req = base.Request{
		Url:       dourl,
		Body:      io.NopCloser(strings.NewReader("")),
		HeaderSet: map[string]string{},
		Method:    "GET",
		Params: map[string]string{"uid": uidStr, "name": name, "date": date, "expire": expireStr, "signature": signature,
			"md5": md5Str, "bucket": bucketName, "object": objectName},
	}
	_, bodyData, _, err := base.AskFile(req)
	if err != nil {
		panic(err)
	}
	defer bodyData.Close()
	_, err = io.Copy(out, bodyData)
	if err != nil {
		panic(err)
	}

	// 计算md5
	md5New, _ := base.CalculateFileMd5(fileName)
	if md5New == md5Str {
		fmt.Println("测试成功.")
	} else {
		fmt.Println("测试失败", md5New, md5Str)
	}
	_ = out.Close()
	_ = os.Remove(fileName)
}
