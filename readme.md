# 🌿 小绿书 · Mini Greenbook

> 基于 Go 语言独立开发的社区内容平台后端，参考小红书核心业务场景设计，并针对高并发场景完成了缓存优化、分页重构、限流与分布式锁设计。

## 技术栈

| 分类     | 技术                             |
| -------- | -------------------------------- |
| 语言     | Go 1.21                          |
| Web 框架 | Gin                              |
| ORM      | GORM                             |
| 数据库   | MySQL 8.0                        |
| 缓存     | Redis 7                          |
| 鉴权     | JWT（golang-jwt/jwt）            |
| 配置管理 | Viper                            |
| 日志     | Zap                              |
| 测试     | Testify                          |
| 容器化   | Docker                           |
| AI 接口  | DeepSeek API（兼容 OpenAI 格式） |

## 功能模块

### 用户模块

- 注册：bcrypt 加盐哈希存储密码
- 登录：密码校验 + JWT Token 签发
- 鉴权：JWT 中间件统一拦截受保护接口，通过 Context 传递用户身份

### 笔记模块

- 发布笔记：支持标题、正文、图片链接（OSS 直传）、AI 标签
- 发现流：游标分页，基于主键 B+ 树索引，O(log N) 复杂度
- 标签搜索：LIKE 模糊匹配 + Offset 分页，支持跳页
- 热榜：Redis ZSet 维护，综合点赞数与时间衰减因子动态计算热度值

### AI 模块

- 独立的标签生成接口（置于受保护路由下，防止额度滥用）
- 调用 DeepSeek API 提取 3-5 个核心标签，用户确认后随笔记入库
- 基于 Redis ZSet + Pipeline 的滑动窗口限流，防止恶意调用

### 互动模块

- 点赞 / 取消点赞：GORM many2many 中间表，Toggle 设计
- 分布式锁：Redis SetNX + UUID + Lua 脚本，防止并发重复写入

## 架构设计

```
HTTP 请求
    ↓
Router（路由注册 + 中间件挂载）
    ↓
Middleware（JWT 鉴权 / 滑动窗口限流）
    ↓
Controller（参数绑定与校验）
    ↓
Service（核心业务逻辑）
    ↓
Model / Config（GORM 数据库操作 / Redis 缓存）
```

## 性能优化

### 缓存设计（防击穿 + 防雪崩 + 保实时性）

| 问题         | 方案                                                     |
| ------------ | -------------------------------------------------------- |
| 缓存击穿     | Singleflight 合并并发请求，热点 Key 过期时只查一次 MySQL |
| 缓存雪崩     | TTL 随机抖动，避免多个 Key 同时失效                      |
| 第一页实时性 | cursor=0 跳过缓存直查数据库，新帖即时可见                |
| 缓存一致性   | Cache Aside 模式，写操作成功后主动删除对应缓存 Key       |

### 游标分页替代 Offset 分页

```
Offset 分页：扫描并丢弃前 N 条，O(N)，深翻页时极慢
游标分页：WHERE id < cursor，走主键 B+ 树索引，O(log N)
```

发现流采用游标分页（无限滚动，无需跳页），搜索接口保留 Offset 分页（用户有跳页需求），根据业务场景混合使用两种策略。

### 分布式锁优化

```
问题一：并发点赞导致重复写入中间表
解法：Redis SetNX，针对"同一用户+同一笔记"加锁

问题二：锁过期后 A 误删 B 的锁
解法：锁的值存 UUID，删除时用 Lua 脚本原子判断"是否是自己的锁"
```

### 压测结果（本地环境，并发 100，总请求 10000）

| 场景                        | QPS   | TP99  |
| --------------------------- | ----- | ----- |
| 发现流（缓存命中）          | 8780  | 52ms  |
| 发现流（Singleflight 生效） | 10986 | 35ms  |
| 翻页（缓存未命中）          | 4780  | 56ms  |
| 翻页（缓存命中）            | 5853  | 51ms  |
| 发帖（写操作）              | 878   | 182ms |

## 项目结构

```
mini-greenbook/
├── cmd/
│   └── main.go                 # 程序入口
├── config/
│   └── db.go                   # MySQL / Redis / Viper / Zap 初始化
├── controller/
│   ├── user.go                 # 用户相关 HTTP 处理
│   ├── post.go                 # 笔记相关 HTTP 处理
│   └── llm_tag.go              # AI 标签 HTTP 处理
├── service/
│   ├── user.go                 # 用户业务逻辑
│   ├── post.go                 # 笔记业务逻辑（缓存 + 分页 + 热榜 + 分布式锁）
│   ├── llm_tag.go              # DeepSeek API 调用
│   └── post_test.go            # 核心业务单元测试
├── model/
│   ├── user.go                 # 用户数据模型
│   └── post.go                 # 笔记数据模型（含 Tags 和多对多）
├── middleware/
│   ├── auth.go                 # JWT 鉴权中间件
│   └── rate_limit.go           # 滑动窗口限流中间件
├── router/
│   └── router.go               # 路由注册
├── utils/
│   ├── jwt.go                  # Token 签发与解析
│   └── response.go             # 统一响应封装
└── config.yaml                 # 配置文件（不提交到 git）
```

## 快速启动

### 前置依赖

- Go 1.21+
- Docker

### 1. 启动数据库

```bash
docker run -d --name greenbook_mysql \
  -e MYSQL_ROOT_PASSWORD=rootpassword \
  -e MYSQL_DATABASE=greenbook_db \
  -p 3306:3306 mysql:8.0

docker run -d --name greenbook_redis \
  -p 6379:6379 redis:latest
```

### 2. 配置文件

在项目根目录创建 `config.yaml`：

```yaml
mysql:
  dsn: "root:rootpassword@tcp(127.0.0.1:3306)/greenbook_db?charset=utf8mb4&parseTime=True&loc=Local"

redis:
  addr: "127.0.0.1:6379"
  password: ""
  db: 0

jwt:
  secret: "your_jwt_secret_key"

llm:
  api_url: "https://api.deepseek.com/v1/chat/completions"
  api_key:  "your_deepseek_api_key"
  model:    "deepseek-chat"
```

> ⚠️ `config.yaml` 已加入 `.gitignore`，请勿提交到 git

### 3. 启动服务

```bash
go mod tidy
go run cmd/main.go
```

服务启动后监听 `8080` 端口。

### 4. 运行单元测试

```bash
go test ./service/ -v -run "TestCreatePost|TestToggleLike"
```

## API 文档

### 公开接口

| 方法 | 路径                       | 描述                     |
| ---- | -------------------------- | ------------------------ |
| POST | /v1/auth/register          | 用户注册                 |
| POST | /v1/auth/login             | 用户登录，返回 JWT Token |
| GET  | /v1/posts?cursor=0&size=10 | 发现流（游标分页）       |
| GET  | /v1/posts/hot              | 热榜                     |
| GET  | /v1/posts/search?tag=美食  | 按标签搜索笔记           |

### 受保护接口（需携带 Token）

请求头：`Authorization: Bearer <token>`

| 方法 | 路径               | 描述                  |
| ---- | ------------------ | --------------------- |
| POST | /v1/posts          | 发布笔记              |
| POST | /v1/posts/:id/like | 点赞 / 取消点赞       |
| POST | /v1/ai/tags        | AI 生成标签（有限流） |

### 响应格式

所有接口统一返回以下格式：

```json
{
  "code": 200,
  "msg": "success",
  "data": {}
}
```

### 示例

**登录**

```json
// POST /v1/auth/login
// Request
{"username": "alice", "password": "123456"}

// Response
{
  "code": 200,
  "msg": "登录成功",
  "data": {"token": "eyJhbGciOiJIUzI1NiIs..."}
}
```

**发现流**

```json
// GET /v1/posts?cursor=0&size=10

// Response
{
  "code": 200,
  "msg": "获取成功",
  "data": {
    "list": [...],
    "next_cursor": 91,
    "size": 10
  }
}
```

**AI 生成标签**

```json
// POST /v1/ai/tags
// Request
{"content": "今天去成都吃了超好吃的火锅，麻辣鲜香..."}

// Response
{
  "code": 200,
  "msg": "AI 标签生成成功",
  "data": {"tags": "美食,川菜,火锅,成都,探店"}
}
```

## 设计决策

**为什么发现流用游标分页，搜索用 Offset 分页？**

发现流是无限滚动场景，用户不需要跳页，游标分页走主键索引 O(log N) 性能好；搜索结果用户可能直接跳到第 N 页，Offset 分页支持跳页，两种场景选用不同策略。

**为什么第一页不走缓存？**

第一页内容最新，用户刷新后应该看到刚发布的帖子。翻页是历史数据，不会有新内容插入，走缓存收益大且风险小。

**Singleflight 和加锁有什么区别？**

加锁是串行，同时只有一个请求执行，其余排队等待。Singleflight 是合并，第一个请求执行，其余等待同一份结果，减少的是重复查询次数而不是限制并发，总耗时只有一个请求的时间。

**为什么用 UUID + Lua 脚本优化分布式锁？**

`SetNX` 存固定值时，锁过期后 A 线程可能误删 B 线程的锁。用 UUID 作为锁的值，删除时先比较是否是自己的 UUID，再决定是否删除。"判断+删除"两步用 Lua 脚本保证原子性，防止判断通过后锁被别人抢走再删除的竞态。

## .gitignore

```
config.yaml
*.log
/tmp
```