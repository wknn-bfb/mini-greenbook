// service/user.go, 用户相关的核心业务逻辑
package service

import (
	"errors"
	"mini-greenbook/config"
	"mini-greenbook/model"
	"mini-greenbook/utils"

	"golang.org/x/crypto/bcrypt"
)

// RegisterUser 注册核心业务逻辑
func RegisterUser(username, password string) error {
	// 1. 检查用户名是否已被注册
	var count int64
	config.DB.Model(&model.User{}).Where("username = ?", username).Count(&count)
	if count > 0 {
		return errors.New("该用户名已被注册")
	}

	// 2. 使用 Bcrypt 对密码进行加盐加密 (DefaultCost 为默认强度)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("密码加密失败")
	}

	// 3. 构造用户对象并写入数据库
	user := model.User{
		Username: username,
		Password: string(hashedPassword),
	}

	if err := config.DB.Create(&user).Error; err != nil {
		return errors.New("数据库写入失败")
	}

	return nil
}

// LoginUser 登录核心业务逻辑
// 返回值: 生成的 JWT 字符串, 错误信息
func LoginUser(username, password string) (string, error) {
	var user model.User

	// 1. 去数据库里找这个用户
	// First() 如果找不到记录会返回错误
	if err := config.DB.Where("username = ?", username).First(&user).Error; err != nil {
		return "", errors.New("用户名不存在")
	}

	// 2. 校验密码
	// CompareHashAndPassword 专门用来对比明文密码和数据库里的 Bcrypt 密文
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", errors.New("密码错误")
	}

	// 3. 密码正确，签发 JWT Token
	token, err := utils.GenerateToken(user.ID)
	if err != nil {
		return "", errors.New("Token 签发失败")
	}

	return token, nil
}
