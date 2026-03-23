// model/user.go, 用户模型定义
package model

import (
	"gorm.io/gorm"
)

// User 用户表结构体
type User struct {
	gorm.Model
	Username string `gorm:"type:varchar(50);uniqueIndex;not null" json:"username"`
	Password string `gorm:"type:varchar(255);not null" json:"-"`
}
