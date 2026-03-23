// utils/jwt.go, JWT 签发与解析工具类
package utils

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/viper"
)

// Claims 定义 JWT 中要携带的有效载荷（即你希望 Token 里保存什么数据）
type Claims struct {
	UserID uint `json:"user_id"` // 我们把用户的 ID 存在 Token 里
	jwt.RegisteredClaims
}

// GenerateToken 签发 Token
func GenerateToken(userID uint) (string, error) {
	// 设置 Token 的过期时间，这里设置为 24 小时后
	expirationTime := time.Now().Add(24 * time.Hour)

	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "mini-greenbook",
		},
	}

	// 从配置文件动态读取 Secret
	jwtSecret := []byte(viper.GetString("jwt.secret"))

	// 使用 HS256 算法生成 Token 对象
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// 使用私钥签名并获得完整的字符串
	return token.SignedString(jwtSecret)
}

// ParseToken 解析并校验 Token
func ParseToken(tokenString string) (*Claims, error) {
	// 从配置文件动态读取 Secret
	jwtSecret := []byte(viper.GetString("jwt.secret"))

	token, err := jwt.ParseWithClaims(
		tokenString,
		&Claims{},
		func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		},
	)

	if err != nil {
		return nil, err
	}

	// 校验 Token 是否有效，并提取 Claims
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("无效的 Token")
}
