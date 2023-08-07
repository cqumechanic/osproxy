package base

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/qinguoyi/osproxy/app/pkg/utils"
)

// encryption.go文件中的函数用于加密和解密

// decode . 用于解密，解密具体是怎么实现的呢？这里的解密是将数字转换成字符串

func decode(message string) string {
	h := hmac.New(sha256.New, []byte(utils.EncryKey))
	h.Write([]byte(message))
	sha := hex.EncodeToString(h.Sum(nil))
	return sha
}

// GenUploadSignature . GenUploadSignature()函数用于生成加密query
func GenUploadSignature(uid, date string, expire int, signature string) string { // GenUploadSignature()函数用于生成加密query
	standardizedQueryString := fmt.Sprintf(
		"uid=%s&date=%s&expire=%d&signature=%s",
		uid,
		date,
		expire,
		signature,
	)
	return standardizedQueryString
}

// CheckUploadSignature . CheckUploadSignature()函数用于检查上传签名
func CheckUploadSignature(date, expire, signature string) bool {
	decodeRes := decode(fmt.Sprintf("%s-%s", date, expire))
	return decodeRes == signature
}

// GenDownloadSignature . GenDownloadSignature()函数用于生成下载签名
func GenDownloadSignature(uid int64, srcName, bucket, objectName, expire, date, signature string) string {
	standardizedQueryString := fmt.Sprintf(
		"uid=%d&name=%s&date=%s&expire=%s&bucket=%s&object=%s&signature=%s",
		uid,
		srcName,
		date,
		expire,
		bucket,
		objectName,
		signature,
	)
	return standardizedQueryString
}

func CheckDownloadSignature(date, expire, bucket, objectName, signature string) bool {
	decodeRes := decode(fmt.Sprintf("%s-%s-%s-%s", date, expire, bucket, objectName))
	return decodeRes == signature
}
