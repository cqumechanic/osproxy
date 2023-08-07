package repo

import (
	"github.com/qinguoyi/osproxy/app/models"
	"gorm.io/gorm"
)

type multiPartInfoRepo struct{}

func NewMultiPartInfoRepo() *multiPartInfoRepo { return &multiPartInfoRepo{} }

// multiPartInfoRepo .
// 这个go文件中的函数与metadatainfo.go中的函数的区别是什么？这个go文件中的函数是对多文件上传的操作，metadatainfo.go中的函数是对单文件上传的操作

// GetPartMaxNumByUid .
func (r *multiPartInfoRepo) GetPartMaxNumByUid(db *gorm.DB, uidList []int64) ([]models.PartInfo, error) {
	var ret []models.PartInfo

	if err := db.Model(&models.MultiPartInfo{}).
		Select("storage_uid, max(chunk_num) as max_chunk").
		Where("storage_uid in ?", uidList).
		Group("storage_uid").Find(&ret).Error; err != nil {
		return nil, err
	}

	return ret, nil
}

// GetPartNumByUid .
func (r *multiPartInfoRepo) GetPartNumByUid(db *gorm.DB, uid int64) ([]models.MultiPartInfo, error) {
	var ret []models.MultiPartInfo
	if err := db.Model(&models.MultiPartInfo{}).Where("storage_uid = ?", uid).Find(&ret).Error; err != nil {
		return nil, err
	}

	return ret, nil
}

// GetPartInfo .
// GetPartInfo()函数用于根据uid、num、md5获取多文件上传信息
func (r *multiPartInfoRepo) GetPartInfo(db *gorm.DB, uid, num int64, md5 string) ([]models.MultiPartInfo, error) {
	var ret []models.MultiPartInfo
	if err := db.Model(&models.MultiPartInfo{}).Where( // where()函数用于添加where条件
		"storage_uid = ? and chunk_num  = ? and part_md5 = ? and status = 1", uid, num, md5,
	).Find(&ret).Error; err != nil { // Find()函数用于查询，查询的结果保存在ret中
		return nil, err
	}
	return ret, nil
}

// Updates .
func (r *multiPartInfoRepo) Updates(db *gorm.DB, uid int64, columns map[string]interface{}) error {
	err := db.Model(&models.MultiPartInfo{}).Where("storage_uid = ?", uid).Updates(columns).Error
	return err
}

// Create .
func (r *multiPartInfoRepo) Create(db *gorm.DB, m *models.MultiPartInfo) error {
	err := db.Create(m).Error
	return err
}

// BatchCreate .
func (r *multiPartInfoRepo) BatchCreate(db *gorm.DB, m *[]models.MultiPartInfo) error {
	err := db.Create(m).Error
	return err
}
