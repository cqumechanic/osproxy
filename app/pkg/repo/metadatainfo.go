package repo

import (
	"github.com/qinguoyi/osproxy/app/models"
	"gorm.io/gorm"
)

type metaDataInfoRepo struct{}

func NewMetaDataInfoRepo() *metaDataInfoRepo { return &metaDataInfoRepo{} }

// 这个go文件中的函数都是对数据库的操作
// repo是repository的缩写，意思是仓库，这里的仓库是指对数据库的操作
// 与multipartinfo.go中的函数相比，两者的区别是：多文件与单文件的区别
// GetByUid .
// GetByUid()函数用于根据uid获取元数据信息
func (r *metaDataInfoRepo) GetByUid(db *gorm.DB, uid int64) (*models.MetaDataInfo, error) {
	ret := &models.MetaDataInfo{}
	if err := db.Where("uid = ?", uid).First(ret).Error; err != nil { // First()函数用于查询第一条数据
		return ret, err
	}
	return ret, nil
}

// GetResumeByMd5 .
// GetResumeByMd5()函数用于根据md5获取秒传数据
func (r *metaDataInfoRepo) GetResumeByMd5(db *gorm.DB, md5 []string) ([]models.MetaDataInfo, error) {
	var ret []models.MetaDataInfo
	if err := db.Where("md5 in ? and status = 1 and multi_part = ?", md5, false). // Where()函数用于指定查询条件，这里的查询条件是md5 in ? and status = 1 and multi_part = ?
											Find(&ret).Error; err != nil { // Find()函数用于查询数据，这里的查询条件是md5 in ? and status = 1 and multi_part = ?
		return ret, err
	}
	return ret, nil
}

// GetPartByMd5 .
// GetPartByMd5()函数用于根据md5获取分片数据
func (r *metaDataInfoRepo) GetPartByMd5(db *gorm.DB, md5 []string) ([]models.MetaDataInfo, error) {
	var ret []models.MetaDataInfo
	if err := db.Where("md5 in ? and status = -1 and multi_part = ? ", md5, true).
		Find(&ret).Error; err != nil {
		return ret, err
	}
	return ret, nil
}

// GetPartByUid .
// GetPartByUid()函数用于根据uid获取分片数据
func (r *metaDataInfoRepo) GetPartByUid(db *gorm.DB, uid int64) ([]models.MetaDataInfo, error) {
	var ret []models.MetaDataInfo
	if err := db.Where("uid = ? and status = -1 and multi_part = ? ", uid, true).
		Find(&ret).Error; err != nil {
		return ret, err
	}
	return ret, nil
}

// GetByUidList .
// GetByUidList()函数用于根据uid列表获取元数据信息
func (r *metaDataInfoRepo) GetByUidList(db *gorm.DB, uid []int64) ([]models.MetaDataInfo, error) {
	var ret []models.MetaDataInfo
	if err := db.Where("uid in ?", uid).Find(&ret).Error; err != nil {
		return ret, err
	}
	return ret, nil
}

// BatchCreate .
// BatchCreate()函数用于批量创建元数据信息
func (r *metaDataInfoRepo) BatchCreate(db *gorm.DB, m *[]models.MetaDataInfo) error {
	err := db.Create(m).Error // Create()函数用于创建数据,，参数是一个切片，
	//切片的元素是MetaDataInfo类型，建表的时候，会自动将MetaDataInfo类型的数据映射到数据库中
	return err
}

// Updates .
// Updates()函数用于更新元数据信息,更新到数据库中
func (r *metaDataInfoRepo) Updates(db *gorm.DB, uid int64, columns map[string]interface{}) error {
	err := db.Model(&models.MetaDataInfo{}).Where("uid = ?", uid).Updates(columns).Error
	return err
}
