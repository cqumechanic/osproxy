package storage

import (
	"fmt"
	"io"
	"log"
	"os"
	"path"

	"github.com/qinguoyi/osproxy/app/pkg/utils"
)

// LocalStorage 本地存储
type LocalStorage struct {
	RootPath string
}

func NewLocalStorage() *LocalStorage {
	return &LocalStorage{
		RootPath: utils.LocalStore,
	}
}

// MakeBucket .
func (s *LocalStorage) MakeBucket(bucketName string) error { // 本质上是创建一个目录
	dirName := path.Join(s.RootPath, bucketName)
	if _, err := os.Stat(dirName); os.IsNotExist(err) { // Stat()函数用于获取文件的信息，参数是一个文件名，返回值是一个FileInfo对象，IsNotExist()函数用于判断文件是否存在，参数是一个error对象，返回值是一个bool类型的值
		if err := os.MkdirAll(dirName, 0755); err != nil { // MkdirAll()函数用于创建一个目录，参数是一个目录名，一个权限，返回值是一个error对象
			//lgLogger.WithContext(c).Error("创建本地目录失败，详情：", zap.Any("err", err.Error()))
			return err
		}
	}
	return nil
}

// GetObject .
func (s *LocalStorage) GetObject(bucketName, objectName string, offset, length int64) ([]byte, error) { // 本质上是读取一个文件
	objectPath := path.Join(s.RootPath, bucketName, objectName)
	file, err := os.Open(objectPath)
	if err != nil {
		fmt.Println("Failed to open file:", err)
		return nil, err
	}
	defer file.Close()
	_, err = file.Seek(offset, io.SeekStart) // Seek()函数用于设置下一次读/写的位置，参数是一个int64类型的值，一个int类型的值，返回值是一个int64类型的值 SeekStart是一个常量，值为0
	if err != nil {
		fmt.Println("Error:", err)
		return nil, err
	}
	buffer := make([]byte, length) // make()函数用于创建一个切片，参数是一个类型，一个长度，返回值是一个切片 在这里是字节数组
	_, err = file.Read(buffer)     // Read()函数用于读取文件内容，参数是一个字节数组，返回值是一个int类型的值，一个error对象
	if err != nil && err != io.EOF {
		fmt.Println("Error:", err)
		return nil, err
	}
	return buffer, nil
}

// PutObject .
func (s *LocalStorage) PutObject(bucketName, objectName, filePath, contentType string) error { // 本质上是复制一个文件 几个参数分别是存储桶名，对象名，文件路径，文件类型
	// copy 数据到 具体的目录
	// 打开源文件
	// filePath代表源文件的路径 contentType代表文件的类型
	sourceFile, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer sourceFile.Close()

	objectPath := path.Join(s.RootPath, bucketName, objectName)
	file, err := os.Create(objectPath) // Create()函数用于创建一个文件，参数是一个文件名，返回值是一个文件对象
	if err != nil {
		fmt.Println("Failed to create file:", err)
		return err
	}
	defer file.Close() // defer函数具体什么时候执行？

	// 复制文件内容
	_, err = io.Copy(file, sourceFile) // Copy()函数用于复制文件内容，参数是一个文件对象，一个文件对象，返回值是一个int64类型的值，一个error对象
	if err != nil {
		fmt.Println("Failed to copy file:", err)
		return err
	}
	return nil
}

func (s *LocalStorage) DeleteObject(bucketName, objectName string) error { // 本质上是删除一个文件
	objectPath := path.Join(s.RootPath, bucketName, objectName)
	err := os.RemoveAll(objectPath)
	return err
}
