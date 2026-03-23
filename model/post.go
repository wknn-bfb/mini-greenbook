// model/post.go, 笔记模型定义
package model

import (
	"gorm.io/gorm"
)

// Post 笔记表结构体
type Post struct {
	gorm.Model
	UserID   uint   `gorm:"not null" json:"user_id"`
	Title    string `gorm:"type:varchar(100);not null" json:"title"`
	Content  string `gorm:"type:text;not null" json:"content"`
	ImageURL string `gorm:"type:varchar(255)" json:"image_url"`

	// 新增标签字段，存储逗号分隔的字符串
	Tags string `gorm:"type:varchar(255)" json:"tags"`

	// 定义多对多关系，GORM 会自动生成 user_like_posts 中间表
	LikedByUsers []User `gorm:"many2many:user_like_posts;" json:"-"`
}
