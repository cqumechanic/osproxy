package models

import "time"

/*
MetaDataInfo 表结构定义及增删改查接口
*/

// MetaDataInfo 元数据表
type MetaDataInfo struct {
	ID          int        `gorm:"column:id;primaryKey;not null;autoIncrement;comment:自增ID"`
	UID         int64      `gorm:"column:uid;primaryKey;not null;comment:唯一ID"`
	Bucket      string     `gorm:"column:bucket;not null;comment:桶"`
	Name        string     `gorm:"column:name;not null;comment:原始名称"`
	StorageName string     `gorm:"column:storage_name;not null;comment:存储名称"`
	Address     string     `gorm:"column:address;not null;comment:存储地址"`
	Md5         string     `gorm:"column:md5;comment:md5"`
	Height      int        `gorm:"column:height;comment:高度"`
	Width       int        `gorm:"column:width;comment:宽度"`
	StorageSize int64      `gorm:"column:storage_size;comment:文件大小"`
	MultiPart   bool       `gorm:"column:multi_part;not null;comment:是否分片"`
	PartNum     int        `gorm:"column:part_num;comment:分片总量"`
	Status      int        `gorm:"column:status;comment:是否上传"`
	ContentType string     `gorm:"column:content_type;comment:文件类型"`
	CompressUid int64      `gorm:"column:compress_uid;comment:压缩文件ID"`
	CreatedAt   *time.Time `gorm:"column:created_at;not null;comment:创建时间"`
	UpdatedAt   *time.Time `gorm:"column:updated_at;not null;comment:更新时间"`
}

// GenUpload 上传链接请求体
type GenUpload struct {
	FilePath []string `json:"filePath" binding:"required"` // 文件路径
	Expire   int      `json:"expire"`                      // 过期时间
}

// MultiUrlResult .
// 在这里实现了单文件上传、合并文件、多文件上传
type MultiUrlResult struct {
	Upload string `json:"upload"`
	Merge  string `json:"merge"`
}

type UrlResult struct {
	Single string          `json:"single"`
	Multi  *MultiUrlResult `json:"multi"`
}

type GenUploadResp struct {
	Uid  string     `json:"uid"`
	Url  *UrlResult `json:"url"`
	Path string     `json:"path"`
}

// GenDownload 下载链接请求体
type GenDownload struct {
	Uid    []string `json:"uid" binding:"required"`    // 文件路径
	Expire int      `json:"expire" binding:"required"` // 过期时间
}

type MetaInfo struct {
	SrcName string `json:"srcName"`
	DstName string `json:"dstName"`
	Height  int    `json:"height"`
	Width   int    `json:"width"`
	Md5     string `json:"md5"`
	Size    string `json:"size"`
}

type GenDownloadResp struct {
	Uid  string   `json:"uid"`
	Url  string   `json:"url"`
	Meta MetaInfo `json:"meta"`
}

type MD5Name struct {
	Md5  string `json:"md5"`  // Md5是一个字符串，Md5是文件的md5值，md5是一种哈希算法，它的作用是将任意长度的数据转换成固定长度的数据，这样就可以用固定长度的数据来表示任意长度的数据了
	Path string `json:"path"` // Path是一个字符串，Path是文件的路径
}

type ResumeReq struct {
	Data []MD5Name `json:"data"` // Data是一个切片，切片的元素是MD5Name类型 这里的`json:"data"`是结构体标签，用于指定结构体成员变量在json中的名称
}

type ResumeResp struct {
	Md5 string `json:"md5"`
	Uid string `json:"uid"`
}
