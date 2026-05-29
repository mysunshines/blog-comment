# Comment Service - 评论服务

## 一、服务概述

评论服务负责处理文章的评论功能，支持评论的创建、回复、点赞和管理。

**端口配置**:
- HTTP: 8083
- gRPC: 9003
- Metrics: 9093

## 二、技术栈

| 类别 | 技术 |
|------|------|
| 语言 | Go 1.21+ |
| Web框架 | Gin |
| RPC框架 | gRPC + Protobuf |
| 数据库 | MySQL 8.0 |
| 缓存 | Redis |
| 监控 | Prometheus |
| 注册中心 | Consul |
| 熔断器 | gobreaker |
| 限流 | golang.org/x/time/rate |

## 三、项目结构

```
comment-service/
├── cmd/server/
│   └── main.go                 # 服务启动 & 路由注册
├── internal/
│   ├── config/config.go        # 配置管理
│   ├── handler/
│   │   ├── comment_handler.go  # HTTP 处理器
│   │   └── grpc_handler.go     # gRPC 处理器
│   ├── service/comment_service.go # 业务逻辑层
│   ├── repository/
│   │   ├── comment_repo.go      # 评论数据访问
│   │   └── comment_like_repo.go # 点赞数据访问
│   ├── model/comment.go         # 数据模型
│   └── middleware/middleware.go # 中间件
├── pkg/
│   ├── errors/errors.go        # 错误处理
│   └── response/response.go    # 响应封装
├── proto/
│   └── comment.proto           # gRPC定义
├── config.yaml                 # 配置文件
└── Dockerfile
```

## 四、API 列表

### 4.1 HTTP API

| 方法 | 路径 | 描述 | 认证 |
|------|------|------|------|

#### 4.1.1 GET `/api/v1/comment` - 获取评论列表

**查询参数 (Query Parameters)**:
```
page: int,            // int, 选填, 页码, 默认1
size: int,            // int, 选填, 每页数量, 默认10
user_id: uint,        // uint, 选填, 用户ID筛选
```

#### 4.1.2 GET `/api/v1/comment/:id` - 获取评论详情

**路径参数**:
```
id: int,              // int, 必填, 评论ID
```

#### 4.1.3 GET `/api/v1/comment/article/:article_id` - 获取文章评论

**路径参数**:
```
article_id: int,       // int, 必填, 文章ID
```

**查询参数 (Query Parameters)**:
```
page: int,            // int, 选填, 页码, 默认1
size: int,            // int, 选填, 每页数量, 默认10
include_replies: bool // bool, 选填, 是否包含回复, 默认false
```

#### 4.1.4 GET `/api/v1/comment/:id/replies` - 获取评论回复

**路径参数**:
```
id: int,              // int, 必填, 评论ID
```

**查询参数 (Query Parameters)**:
```
page: int,            // int, 选填, 页码, 默认1
size: int,            // int, 选填, 每页数量, 默认10
```

#### 4.1.5 POST `/api/v1/comment` - 创建评论

**请求头**:
```
Authorization: Bearer <token>     // string, 必填, JWT Token
```

**请求体 (JSON)**:
```json
{
    "article_id": "uint",        // uint, 必填, 文章ID
    "content": "string",         // string, 必填, 评论内容, 长度1-2000字符
    "parent_id": "uint"          // uint, 选填, 父评论ID (0为顶级评论)
}
```

#### 4.1.6 PUT `/api/v1/comment/:id` - 更新评论

**请求头**:
```
Authorization: Bearer <token>     // string, 必填, JWT Token
```

**路径参数**:
```
id: int,              // int, 必填, 评论ID
```

**请求体 (JSON)**:
```json
{
    "user_id": "uint",           // uint, 必填, 用户ID (从Token解析)
    "content": "string"          // string, 必填, 评论内容, 长度1-2000字符
}
```

#### 4.1.7 DELETE `/api/v1/comment/:id` - 删除评论

**请求头**:
```
Authorization: Bearer <token>     // string, 必填, JWT Token
```

**路径参数**:
```
id: int,              // int, 必填, 评论ID
```

#### 4.1.8 POST `/api/v1/comment/:id/reply` - 回复评论

**请求头**:
```
Authorization: Bearer <token>     // string, 必填, JWT Token
```

**路径参数**:
```
id: int,              // int, 必填, 被回复的评论ID
```

**请求体 (JSON)**:
```json
{
    "user_id": "uint",           // uint, 必填, 用户ID (从Token解析)
    "content": "string"          // string, 必填, 回复内容, 长度1-2000字符
}
```

#### 4.1.9 POST `/api/v1/comment/:id/like` - 点赞评论

**请求头**:
```
Authorization: Bearer <token>     // string, 必填, JWT Token
```

**路径参数**:
```
id: int,              // int, 必填, 评论ID
```

#### 4.1.10 POST `/api/v1/comment/article/:article_id/enable` - 开启文章评论

**请求头**:
```
Authorization: Bearer <token>     // string, 必填, JWT Token
```

**路径参数**:
```
article_id: int,       // int, 必填, 文章ID
```

#### 4.1.11 POST `/api/v1/comment/article/:article_id/disable` - 关闭文章评论

**请求头**:
```
Authorization: Bearer <token>     // string, 必填, JWT Token
```

**路径参数**:
```
article_id: int,       // int, 必填, 文章ID
```

#### 4.1.12 GET `/health` - 健康检查

**说明**: 无参数

#### 4.1.13 GET `/ready` - 就绪探针

**说明**: 无参数

### 4.2 gRPC API

```protobuf
service CommentService {
    rpc CreateComment(CreateCommentRequest) returns (CreateCommentResponse);
    rpc GetComment(GetCommentRequest) returns (GetCommentResponse);
    rpc UpdateComment(UpdateCommentRequest) returns (UpdateCommentResponse);
    rpc DeleteComment(DeleteCommentRequest) returns (DeleteCommentResponse);
    rpc ListComments(ListCommentsRequest) returns (ListCommentsResponse);
    rpc GetArticleComments(GetArticleCommentsRequest) returns (GetArticleCommentsResponse);
    rpc ReplyComment(ReplyCommentRequest) returns (ReplyCommentResponse);
    rpc LikeComment(LikeCommentRequest) returns (LikeCommentResponse);
    rpc GetCommentReplies(GetCommentRepliesRequest) returns (GetCommentRepliesResponse);
    rpc EnableComment(EnableCommentRequest) returns (EnableCommentResponse);
    rpc DisableComment(DisableCommentRequest) returns (DisableCommentResponse);
}
```

## 五、API 流程图

### 5.1 创建评论流程

```
┌─────────┐     ┌──────────────┐     ┌─────────────┐     ┌──────────────┐
│  Client │────▶│ CreateComment│────▶│   Validate   │────▶│   Check       │
│         │     │   Handler    │     │   Input     │     │  User Service│
└─────────┘     └──────────────┘     └─────────────┘     └──────┬───────┘
                                                                │
                        ┌───────────────────────────────────────┘
                        ▼
                   ┌─────────────┐     ┌──────────────┐     ┌───────────────┐
                   │   Check     │────▶│   Validate   │────▶│   Check        │
                   │   Login     │     │   Article    │     │   Parent       │
                   └─────────────┘     └──────────────┘     └───────┬───────┘
                                                                      │
                        ┌─────────────────────────────────────────────┘
                        ▼
                   ┌─────────────┐     ┌──────────────┐     ┌───────────────┐
                   │   Create    │────▶│   Update     │────▶│   Invalidate  │
                   │   Comment   │     │   ReplyCount  │     │   Cache       │
                   └─────────────┘     └──────────────┘     └───────────────┘
                                                                      │
                        ┌─────────────────────────────────────────────┘
                        ▼
                   ┌─────────────┐
                   │   Return    │
                   │   Success   │
                   └─────────────┘
```

### 5.2 获取文章评论流程

```
┌─────────┐     ┌─────────────────┐     ┌──────────────┐     ┌──────────────┐
│  Client │────▶│GetArticleComments│────▶│   Check      │────▶│   Query      │
│         │     │    Handler      │     │   Cache      │     │   Database   │
└─────────┘     └─────────────────┘     └──────────────┘     └──────┬───────┘
                                                                     │
                        ┌────────────────────────────────────────────┘
                        ▼
                   ┌─────────────┐     ┌──────────────┐     ┌──────────────┐
                   │   Build     │────▶│   Set        │────▶│   Return     │
                   │   Tree      │     │   Cache      │     │   Response   │
                   └─────────────┘     └──────────────┘     └──────────────┘
```

## 六、Prometheus Metrics

| 指标名称 | 类型 | 标签 | 描述 |
|----------|------|------|------|
| `http_requests_total` | Counter | method, endpoint, status | HTTP请求总数 |
| `http_request_duration_seconds` | Histogram | method, endpoint | HTTP请求延迟 |
| `rpc_requests_total` | Counter | service, method, status | RPC请求总数 |
| `rpc_request_duration_seconds` | Histogram | service, method | RPC请求延迟 |
| `requests_in_flight` | Gauge | - | 当前处理中的请求数 |
| `request_duration_seconds` | Histogram | method, endpoint | 请求总延迟 |
| `errors_total` | Counter | type | 错误总数 |
| `cpu_usage_percent` | Gauge | - | CPU使用率 |
| `memory_usage_bytes` | Gauge | - | 内存使用量 |
| `goroutine_count` | Gauge | - | Goroutine数量 |
| `panic_counter_total` | Counter | service | Panic次数 |
| `mysql_slow_queries_total` | Counter | - | MySQL慢查询数 |
| `redis_hit_rate` | Gauge | - | Redis命中率 |
| `redis_hot_keys_total` | Counter | key | 热键访问次数 |
| `cache_operations_total` | Counter | operation, status | 缓存操作数 |
| `db_operations_total` | Counter | operation, status | 数据库操作数 |
| `service_health` | Gauge | service | 服务健康状态 |

## 七、高并发特性

### 7.1 熔断器 (Circuit Breaker)

#### 原理详解

熔断器是防止级联故障的关键组件，当下游服务不可用时快速失败，避免请求堆积和资源耗尽。

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           熔断器状态机                                       │
│                                                                             │
│  ┌───────────────────────────────────────────────────────────────────────┐ │
│  │                                                                       │ │
│  │                        ┌──────────────────┐                           │ │
│  │                        │      CLOSED      │                           │ │
│  │                        │      关闭        │                           │ │
│  │                        │  ┌───────────┐  │                           │ │
│  │                        │  │ 执行请求   │  │                           │ │
│  │                        │  │ 记录成功/失│  │                           │ │
│  │                        │  │ 败计数     │  │                           │ │
│  │                        │  └─────┬─────┘  │                           │ │
│  │                        └────────┼────────┘                           │ │
│  │                                 │                                    │ │
│  │                    失败率>阈值 │ & 请求>=5                           │ │
│  │                    ════════════╪════════════                        │ │
│  │                                 ▼                                    │ │
│  │                        ┌──────────────────┐                           │ │
│  │                        │       OPEN       │                           │ │
│  │                        │       熔断       │                           │ │
│  │                        │  ┌───────────┐  │                           │ │
│  │                        │  │ 快速失败  │  │                           │ │
│  │                        │  │ 直接返回  │  │                           │ │
│  │                        │  │ 错误      │  │                           │ │
│  │                        │  └───────────┘  │                           │ │
│  │                        └────────┬─────────┘                           │ │
│  │                                 │                                     │ │
│  │                    Timeout(30s) │ 后                                  │ │
│  │                    ════════════╪══════                               │ │
│  │                                 ▼                                     │ │
│  │                        ┌──────────────────┐                           │ │
│  │                        │    HALF_OPEN     │                           │ │
│  │                        │      半开        │                           │ │
│  │                        │  ┌───────────┐  │                           │ │
│  │                        │  │ 放行请求  │  │                           │ │
│  │                        │  │ 探测恢复  │  │                           │ │
│  │                        │  └─────┬─────┘  │                           │ │
│  │                        └────────┼────────┘                           │ │
│  │                                 │                                    │ │
│  │              ┌──────────────────┴──────────────────┐                │ │
│  │              │                                     │                  │ │
│  │       探测成功                               探测失败                 │ │
│  │       ═══════╪══════                           ═══════╪══════        │ │
│  │              │                                     │                  │ │
│  │              ▼                                     ▼                  │ │
│  │     ┌───────────────┐                     ┌───────────────┐           │ │
│  │     │    CLOSED     │                     │     OPEN      │           │ │
│  │     │    恢复正常    │                     │   继续熔断    │           │ │
│  │     └───────────────┘                     └───────────────┘           │ │
│  │                                                                       │ │
│  └───────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

**熔断器解决的问题**：

| 问题 | 描述 | 熔断器如何解决 |
|------|------|---------------|
| **雪崩效应** | 单点故障扩散到整个系统 | 快速失败阻止请求堆积 |
| **资源耗尽** | 线程池/连接池耗尽 | 限制对故障服务的调用 |
| **故障传播** | A→B→C 的调用链断裂 | 隔离故障，优先保护核心链路 |

**gobreaker 源码核心逻辑**：

```go
// gobreaker 工作流程
type CircuitBreaker struct {
    name        string
    maxRequests uint32        // 半开状态最大请求数
    interval    time.Duration // 统计窗口
    timeout     time.Duration // 熔断持续时间
    
    state     State
    counts    counts          // 成功/失败计数
    stateTime time.Time       // 状态变更时间
}

// 允许请求检查
func (cb *CircuitBreaker) AllowRequest() bool {
    switch cb.getState() {
    case StateClosed:
        return true  // 关闭状态: 始终允许
        
    case StateOpen:
        // 检查是否到达重试时间
        if time.Since(cb.to.openTime) > cb.settings.Timeout {
            cb.toState(StateHalfOpen)  // 进入半开状态
            return true
        }
        return false  // 继续熔断
        
    case StateHalfOpen:
        // 半开状态: 限制并发请求数
        return cb.to.requestCount() < cb.settings.MaxRequests
    }
    return false
}

// 记录执行结果
func (cb *CircuitBreaker) recordResult(err error) {
    if err == nil {
        cb.onSuccess()   // 成功计数
    } else {
        cb.onFailure()   // 失败计数
    }
    
    // 检查是否需要熔断
    cb.toCircuitOpen()
}

// 熔断条件判断
func (cb *CircuitBreaker) toCircuitOpen() bool {
    if cb.getState() != StateClosed {
        return false
    }
    
    // 失败率判断
    if cb.requestCount() >= cb.settings.MinRequests &&
       float64(cb.counts.Failures)/float64(cb.requestCount()) >= cb.settings.FailureRate {
        cb.toState(StateOpen)
        return true
    }
    return false
}
```

#### 配置示例

```go
cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
    Name:        "comment-service",
    MaxRequests: 3,              // 半开状态最多3个请求探测
    Interval:    10 * time.Second, // 10秒统计窗口
    Timeout:     30 * time.Second,  // 熔断30秒后尝试恢复
})
```

### 7.2 限流 (Rate Limiting)

#### 原理详解

使用令牌桶算法实现精确的流量控制。

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         令牌桶算法原理                                        │
│                                                                             │
│  核心概念:                                                                   │
│  ────────                                                                   │
│                                                                             │
│         ┌────────────────────────────────────────────────────────────┐     │
│         │                                                            │     │
│         │                    ┌───────────────┐                       │     │
│         │                    │   令牌桶      │                       │     │
│         │                    │ ┌─┬─┬─┬─┬─┬─┐ │  ← 当前令牌数        │     │
│         │                    │ │█│█│█│█│ │ │ │                       │     │
│         │                    │ └─┴─┴─┴─┴─┴─┘ │                       │     │
│         │                    │   8/10 个令牌 │                       │     │
│         │                    └───────┬───────┘                       │     │
│         │                            │                               │     │
│         │          定时补充            │        每次请求消耗            │     │
│         │    ◀──────────────────────┼───────────────────────────▶     │     │
│         │                            │                               │     │
│         │                     ┌──────┴───────┐                       │     │
│         │                     │              │                       │     │
│         │            ┌───────┴────┐    ┌────┴────────┐              │     │
│         │            │  令牌足够   │    │  令牌不足   │              │     │
│         │            │  执行请求   │    │  拒绝请求   │              │     │
│         │            └────────────┘    └─────────────┘              │     │
│         │                                                            │     │
│         └────────────────────────────────────────────────────────────┘     │
│                                                                             │
│  令牌补充过程:                                                               │
│  ───────────                                                               │
│                                                                             │
│  每秒补充 QPS 个令牌                                                         │
│                                                                             │
│  t=0s:     ┌███████████████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░│  初始10个 │
│                                                                             │
│  t=0.1s:   ┌████████████████████████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░│  处理5个   │
│                                                                             │
│  t=1s:     ┌████████████████████████████████████████████░░░░░░░░░░░│  补充5个   │
│                                                                             │
│  t=1.5s:   ┌██████████████████████████████████████████████████░░░░│  处理3个   │
│                                                                             │
│  容量控制:                                                                   │
│  ──────                                                                     │
│                                                                             │
│  tokens = min(burst, tokens + elapsed * rate)                              │
│                                                                             │
│  • 令牌数永远不会超过 burst (桶容量)                                         │
│  • 即使补充速率很高,也不会溢出                                               │
│  • 确保突发流量可预测                                                        │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

**IP 级别限流实现**：

```go
// 为每个 IP 创建独立的限流器
type IPRateLimiter struct {
    limiters sync.Map  // map[string]*rate.Limiter
    rate     rate.Limit
    burst    int
}

func (m *IPRateLimiter) getLimiter(ip string) *rate.Limiter {
    limiter, ok := m.limiters.Load(ip)
    if !ok {
        // 首次为该 IP 创建限流器
        limiter = rate.NewLimiter(m.rate, m.burst)
        m.limiters.Store(ip, limiter)
    }
    return limiter.(*rate.Limiter)
}

// 中间件中使用
func RateLimitMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        ip := c.ClientIP()
        limiter := ipLimiter.getLimiter(ip)
        
        if !limiter.Allow() {
            c.AbortWithStatusJSON(429, gin.H{
                "code":    429,
                "message": "请求过于频繁，请稍后重试",
            })
            return
        }
        c.Next()
    }
}
```

### 7.3 HTTP Server 超时配置

#### 原理详解

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         HTTP 超时机制详解                                    │
│                                                                             │
│  请求生命周期:                                                               │
│  ────────────                                                               │
│                                                                             │
│  Client ──────────────────────────────────────────────────────────── Server │
│    │                                                                   │     │
│    │                                                                   │     │
│    │                                                                   ▼     │
│    │                                                          ┌─────────────┐ │
│    │◀──────────── ReadHeaderTimeout (10s) ───────────────────│ 读取请求头  │ │
│    │                                                          └──────┬──────┘ │
│    │                                                                   │       │
│    │                                                                   ▼       │
│    │                                                          ┌─────────────┐ │
│    │◀───────────── ReadTimeout (30s) ────────────────────────│ 读取请求体  │ │
│    │                                                          └──────┬──────┘ │
│    │                                                                   │       │
│    │                                                                   ▼       │
│    │                                                          ┌─────────────┐ │
│    │◀───────────── Handler 处理时间 ─────────────────────────│ 业务逻辑    │ │
│    │                                                          └──────┬──────┘ │
│    │                                                                   │       │
│    │                                                                   ▼       │
│    │                                                          ┌─────────────┐ │
│    │◀───────────── WriteTimeout (30s) ───────────────────────│ 写入响应    │ │
│    │                                                          └─────────────┘ │
│    │                                                                   │       │
│    │   IdleTimeout (120s 无活动)                                       │       │
│    │◀────────────────────────────────────────────────────────────────│       │
│    │                                                                   │       │
│    │                         连接关闭                                    │       │
│    │                                                                   │       │
│    └────────────────────────────────────────────────────────────────────────▶│
│                                                                             │
│  各超时作用:                                                                 │
│  ──────────                                                                 │
│                                                                             │
│  ┌────────────────────┬──────────────────────────────────────────────────┐  │
│  │ ReadHeaderTimeout  │ 防止 Slowloris 攻击 (缓慢发送请求头)              │  │
│  │                   │ 建议: 设置为 ReadTimeout 的 1/3                  │  │
│  ├────────────────────┼──────────────────────────────────────────────────┤  │
│  │ ReadTimeout       │ 防止客户端发送过慢                                │  │
│  │                   │ 考虑大文件上传场景                                │  │
│  ├────────────────────┼──────────────────────────────────────────────────┤  │
│  │ WriteTimeout      │ 防止慢客户端读取响应                              │  │
│  │                   │ 考虑大响应/慢网络场景                             │  │
│  ├────────────────────┼──────────────────────────────────────────────────┤  │
│  │ IdleTimeout       │ 释放空闲连接资源                                 │  │
│  │                   │ Keep-Alive 连接的生命周期                         │  │
│  └────────────────────┴──────────────────────────────────────────────────┘  │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 7.4 gRPC Keepalive 原理

#### 原理详解

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         gRPC Keepalive 机制                                  │
│                                                                             │
│   Client                              Server                                │
│      │                                   │                                   │
│      │                                   │                                   │
│      │◀═══════════════════════════════════▶                                 │
│      │         正常数据传输                                                │
│      │◀═══════════════════════════════════▶                                 │
│      │                                   │                                   │
│      │                                   │                                   │
│      │                                   │◀── MaxConnectionIdle              │
│      │                                   │    空闲超时关闭                   │
│      │                                   │                                   │
│      │                                   │◀── MaxConnectionAge              │
│      │                                   │    强制重置                       │
│      │                                   │                                   │
│      │   Time: 2h ───────────┐           │                                   │
│      │                      │           │                                   │
│      │◀─────────────────────┼──────────▶│                                   │
│      │                      │  PING     │                                   │
│      │                      │           │                                   │
│      │                      │  PING ACK │                                   │
│      │─────────────────────┼──────────▶│                                   │
│      │                      │           │                                   │
│      │   Timeout: 20s      │           │                                   │
│      │◀────────────────────┘           │                                   │
│      │   ACK 超时 = 判定连接断开          │                                   │
│      │                                   │                                   │
│  ┌───────────────────────────────────────────────────────────────────────┐ │
│  │                        参数对比                                        │ │
│  │                                                                        │ │
│  │  ┌──────────────────────┬─────────────────────────────────────────┐   │ │
│  │  │       参数           │              作用                       │   │ │
│  │  ├──────────────────────┼─────────────────────────────────────────┤   │ │
│  │  │ MaxConnectionIdle   │ 空闲多久关闭连接 (5min)                 │   │ │
│  │  ├──────────────────────┼─────────────────────────────────────────┤   │ │
│  │  │ MaxConnectionAge    │ 连接最大存活 (10min)                   │   │ │
│  │  ├──────────────────────┼─────────────────────────────────────────┤   │ │
│  │  │ Time (Client→Server) │ 探测帧发送间隔 (2h)                     │   │ │
│  │  ├──────────────────────┼─────────────────────────────────────────┤   │ │
│  │  │ Timeout              │ PING 响应超时 (20s)                     │   │ │
│  │  ├──────────────────────┼─────────────────────────────────────────┤   │ │
│  │  │ MinTime              │ 允许的最小 PING 间隔 (5min)            │   │ │
│  │  ├──────────────────────┼─────────────────────────────────────────┤   │ │
│  │  │ PermitWithoutStream  │ 无流时是否响应 PING                     │   │ │
│  │  └──────────────────────┴─────────────────────────────────────────┘   │ │
│  │                                                                        │ │
│  │  Keepalive 解决场景:                                                    │ │
│  │  1. NAT 超时: 网络地址转换清理长时间空闲映射                             │ │
│  │  2. 防火墙超时: 中间设备关闭空闲连接                                     │ │
│  │  3. 负载均衡: LB 清理空闲连接                                           │ │
│  │  4. 对端故障: 检测已崩溃的对端                                          │ │
│  │                                                                        │ │
│  └───────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## 八、中间件链

```
请求 → RecoveryMiddleware → LoggingMiddleware → CORSMiddleware 
    → MetricsMiddleware → TraceMiddleware → RateLimitMiddleware 
    → JWTValidMiddleware → Handler → Response
```

## 九、数据库模型

### 9.1 comments 表

| 字段 | 类型 | 约束 | 描述 |
|------|------|------|------|
| id | INT UNSIGNED | PRIMARY KEY, AUTO_INCREMENT | 评论ID |
| article_id | INT UNSIGNED | NOT NULL, INDEX | 文章ID |
| user_id | INT UNSIGNED | NOT NULL, INDEX | 用户ID |
| parent_id | INT UNSIGNED | DEFAULT 0, INDEX | 父评论ID (0为顶级评论) |
| content | TEXT | NOT NULL | 评论内容 |
| like_count | INT UNSIGNED | DEFAULT 0 | 点赞数 |
| reply_count | INT UNSIGNED | DEFAULT 0 | 回复数 |
| status | TINYINT UNSIGNED | DEFAULT 1 | 状态(1=正常, 0=删除) |
| created_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | ON UPDATE CURRENT_TIMESTAMP | 更新时间 |

**索引信息**:
| 索引名 | 类型 | 列 | 唯一 | 说明 |
|--------|------|-----|------|------|
| PRIMARY | 主键 | id | - | 主键索引 |
| idx_article_id | 普通 | article_id | 否 | 文章评论列表查询 |
| idx_user_id | 普通 | user_id | 否 | 用户评论列表查询 |
| idx_parent_id | 普通 | parent_id | 否 | 获取评论回复列表 |
| idx_status | 普通 | status | 否 | 状态筛选 |
| idx_created_at | 普通 | created_at | 否 | 创建时间排序 |

### 9.2 comment_likes 表

| 字段 | 类型 | 约束 | 描述 |
|------|------|------|------|
| id | INT UNSIGNED | PRIMARY KEY, AUTO_INCREMENT | 点赞ID |
| comment_id | INT UNSIGNED | NOT NULL, INDEX | 评论ID |
| user_id | INT UNSIGNED | NOT NULL, INDEX | 用户ID |
| created_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | 创建时间 |

**索引信息**:
| 索引名 | 类型 | 列 | 唯一 | 说明 |
|--------|------|-----|------|------|
| PRIMARY | 主键 | id | - | 主键索引 |
| uk_comment_user | 唯一 | (comment_id, user_id) | 是 | 防止重复点赞 |
| idx_comment_id | 普通 | comment_id | 否 | 评论点赞列表查询 |
| idx_user_id | 普通 | user_id | 否 | 用户点赞记录查询 |

### 9.3 articles 表 (评论开关)

| 字段 | 类型 | 约束 | 描述 |
|------|------|------|------|
| id | INT UNSIGNED | PRIMARY KEY | 文章ID |
| user_id | INT UNSIGNED | INDEX | 作者ID |
| title | VARCHAR(256) | - | 文章标题 |
| allow_comment | BOOLEAN | DEFAULT TRUE | 是否允许评论 |
| comment_count | INT | DEFAULT 0 | 评论数量 |
| created_at | TIMESTAMP | - | 创建时间 |
| updated_at | TIMESTAMP | - | 更新时间 |

**索引信息**:
| 索引名 | 类型 | 列 | 唯一 | 说明 |
|--------|------|-----|------|------|
| PRIMARY | 主键 | id | - | 主键索引 |
| idx_user_id | 普通 | user_id | 否 | 作者ID查询 |

## 十、GORM 原理

### 10.1 GORM 架构概览

GORM 是 Go 语言中最流行的 ORM 库之一，采用分层架构设计，将用户操作转换为 SQL 执行。

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              GORM 架构分层                                   │
│                                                                             │
│  ┌───────────────────────────────────────────────────────────────────────┐ │
│  │                        应用层 (Application Layer)                       │ │
│  │                                                                        │ │
│  │   comment := &Comment{}                                               │ │
│  │   db.First(comment, 1)                                                │ │
│  │   db.Create(&Comment{Content: "test"})                                 │ │
│  │                                                                        │ │
│  └───────────────────────────────────────────────────────────────────────┘ │
│                                    │                                        │
│                                    ▼                                        │
│  ┌───────────────────────────────────────────────────────────────────────┐ │
│  │                    链式 API (Chainable API)                            │ │
│  │                                                                        │ │
│  │   db.Where("article_id = ?", articleID).                              │ │
│  │      Order("created_at desc").                                         │ │
│  │      Preload("User").                                                  │ │
│  │      Find(&comments)                                                   │ │
│  │                                                                        │ │
│  └───────────────────────────────────────────────────────────────────────┘ │
│                                    │                                        │
│                                    ▼                                        │
│  ┌───────────────────────────────────────────────────────────────────────┐ │
│  │                     核心层 (Core Layer)                                 │ │
│  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐         │ │
│  │  │    Dialector   │  │    Clause      │  │    Callback     │         │ │
│  │  │   数据库适配器   │  │    SQL构建器    │  │    钩子函数      │         │ │
│  │  └─────────────────┘  └─────────────────┘  └─────────────────┘         │ │
│  └───────────────────────────────────────────────────────────────────────┘ │
│                                    │                                        │
│                                    ▼                                        │
│  ┌───────────────────────────────────────────────────────────────────────┐ │
│  │                      数据库驱动层 (Driver Layer)                        │ │
│  │                                                                        │ │
│  │   MySQL: go-sql-driver/mysql                                           │ │
│  │   PostgreSQL: lib/pq                                                   │ │
│  │   SQLite: modernc.org/sqlite                                           │ │
│  │                                                                        │ │
│  └───────────────────────────────────────────────────────────────────────┘ │
│                                    │                                        │
│                                    ▼                                        │
│  ┌───────────────────────────────────────────────────────────────────────┐ │
│  │                          数据库 (Database)                              │ │
│  │                                                                        │ │
│  │                        MySQL / PostgreSQL / SQLite                      │ │
│  │                                                                        │ │
│  └───────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 10.2 SQL 生成原理

#### 10.2.1 链式调用原理

GORM 的链式 API 基于 Go 的方法链模式，每个方法返回 `*gorm.DB`，可以继续调用其他方法。

```go
// GORM DB 结构
type DB struct {
    Error        error
    RowsAffected int64
    
    // 内部状态
    Statement    *Statement
    Config       *Config
    Clone        *bool
    // ... 其他字段
}

// 链式调用示例
func (db *DB) Where(query interface{}, args ...interface{}) *DB {
    return db.clone().Session(&gorm.Session{}).Where(query, args...)
}

func (db *DB) Order(value interface{}) *DB {
    return db.clone().Session(&gorm.Session{}).Order(value)
}

func (db *DB) Find(dest interface{}, conds ...interface{}) *DB {
    return db.session(func(tx *DB) *DB {
        return tx.CallFunction(func(tx *DB) *DB {
            return tx.NewRecord(dest)
        }, dest, conds...)
    })
}
```

#### 10.2.2 SQL 构建流程

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         GORM SQL 构建流程                                    │
│                                                                             │
│  1. 用户调用                                                                 │
│  ──────────                                                                 │
│                                                                             │
│  db.Select("id, content, like_count").                                      │
│     Where("article_id = ?", 1).                                             │
│     Where("parent_id = ?", 0).                                              │
│     Order("created_at desc").                                               │
│     Limit(20).                                                              │
│     Find(&comments)                                                         │
│                                                                             │
│                                    │                                        │
│                                    ▼                                        │
│  2. 构建 Statement                                                           │
│  ────────────────                                                            │
│                                                                             │
│  Statement {                                                                │
│      Schema:     Comment{}         // 模型结构体                                │
│      Selects:    ["id", "content", "like_count"]  // SELECT 字段            │
│      Clause:     {where: [article_id=?, parent_id=?], order: [...]}         │
│      TableOpts:  {}              // 表选项                                  │
│  }                                                                           │
│                                                                             │
│                                    │                                        │
│                                    ▼                                        │
│  3. 注册 Callback                                                            │
│  ────────────────                                                            │
│                                                                             │
│  callbacks.Create() → callbacks.Query() → callbacks.Update() → ...          │
│                                                                             │
│                                    │                                        │
│                                    ▼                                        │
│  4. 执行 SQL                                                                 │
│  ──────────                                                                 │
│                                                                             │
│  SELECT id, content, like_count FROM comments                                │
│  WHERE article_id = 1 AND parent_id = 0 ORDER BY created_at DESC LIMIT 20   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 10.3 模型定义与映射

#### 10.3.1 Tag 解析机制

GORM 通过反射解析结构体的字段和 Tag，生成数据库表的映射关系。

```go
// 评论模型定义
type Comment struct {
    ID         uint      `gorm:"primaryKey;autoIncrement"`
    ArticleID  uint      `gorm:"index;not null"`
    UserID     uint      `gorm:"index;not null"`
    ParentID   uint      `gorm:"default:0;index"`
    Content    string    `gorm:"type:text;not null"`
    LikeCount  uint      `gorm:"default:0"`
    ReplyCount uint      `gorm:"default:0"`
    Status     uint8     `gorm:"default:1;index"`
    CreatedAt  time.Time `gorm:"autoCreateTime"`
    UpdatedAt  time.Time `gorm:"autoUpdateTime"`
    
    // 关联
    User    User    `gorm:"foreignKey:UserID"`
    Replies []Comment `gorm:"foreignKey:ParentID"`
}
```

#### 10.3.2 Tag 解析流程

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         GORM Tag 解析流程                                    │
│                                                                             │
│  ┌───────────────────────────────────────────────────────────────────────┐ │
│  │                        反射获取结构体                                   │ │
│  │                                                                        │ │
│  │   t := reflect.TypeOf(Comment{})                                       │ │
│  │   for i := 0; i < t.NumField(); i++ {                                  │ │
│  │       field := t.Field(i)                                               │ │
│  │       tag := field.Tag.Get("gorm")                                      │ │
│  │   }                                                                     │ │
│  │                                                                        │ │
│  └───────────────────────────────────────────────────────────────────────┘ │
│                                    │                                        │
│                                    ▼                                        │
│  ┌───────────────────────────────────────────────────────────────────────┐ │
│  │                        Tag 解析器                                      │ │
│  │                                                                        │ │
│  │   // Tag 格式: "key1:value1;key2:value2"                              │ │
│  │   // 例如: "type:text;not null"                                        │ │
│  │                                                                        │ │
│  │   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                    │ │
│  │   │  type       │  │    index    │  │  not null   │                    │ │
│  │   │  text       │  │   创建普通索引 │  │  非空约束    │                    │ │
│  │   └─────────────┘  └─────────────┘  └─────────────┘                    │ │
│  │                                                                        │ │
│  └───────────────────────────────────────────────────────────────────────┘ │
│                                    │                                        │
│                                    ▼                                        │
│  ┌───────────────────────────────────────────────────────────────────────┐ │
│  │                        生成 Schema                                     │ │
│  │                                                                        │ │
│  │   Schema {                                                              │ │
│  │       Table: "comments",                                                │ │
│  │       Fields: [                                                         │ │
│  │           {Name: "ID", DBName: "id", Type: "uint", PK: true},          │ │
│  │           {Name: "Content", DBName: "content", Type: "text"},           │ │
│  │           ...                                                           │ │
│  │       ],                                                                │ │
│  │       Relations: {User: BelongsTo, Replies: HasMany}                   │ │
│  │   }                                                                     │ │
│  │                                                                        │ │
│  └───────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 10.4 数据库操作原理

#### 10.4.1 查询操作 (Find)

```go
// 用户代码
var comments []Comment
db.Where("article_id = ? AND parent_id = ?", articleID, 0).Find(&comments)

// 内部执行流程
func (db *DB) Find(dest interface{}, conds ...interface{}) *DB {
    return db.Session(&gorm.Session{}).callback.Find → Generate SQL
}
```

#### 10.4.2 创建操作 (Create)

```go
// 用户代码
comment := Comment{ArticleID: 1, UserID: 1, Content: "test"}
db.Create(&comment)

// 内部执行流程
func (db *DB) Create(value interface{}, opts ...Option) *DB {
    return db.Session(&gorm.Session{}).callback.Create → Generate INSERT SQL
}
```

#### 10.4.3 更新操作 (Update)

```go
// 用户代码
db.Model(&comment).Update("like_count", gorm.Expr("like_count + 1"))

// 内部执行流程
func (db *DB) Update(column string, value interface{}) *DB {
    return db.Session(&gorm.Session{}).callback.Update → Generate UPDATE SQL
}
```

### 10.5 钩子函数 (Hooks) 原理

#### 10.5.1 钩子类型

GORM 支持在 CRUD 操作前后执行钩子函数：

```go
type Comment struct {
    Content string
}

// 创建前钩子
func (c *Comment) BeforeCreate(tx *gorm.DB) error {
    // 敏感词过滤
    c.Content = filterSensitiveWords(c.Content)
    return nil
}

// 创建后钩子 - 更新评论数
func (c *Comment) AfterCreate(tx *gorm.DB) error {
    // 更新文章评论数
    tx.Model(&Article{}).Where("id = ?", c.ArticleID).
        UpdateColumn("comment_count", gorm.Expr("comment_count + 1"))
    
    // 如果是回复，更新父评论回复数
    if c.ParentID > 0 {
        tx.Model(&Comment{}).Where("id = ?", c.ParentID).
            UpdateColumn("reply_count", gorm.Expr("reply_count + 1"))
    }
    return nil
}
```

#### 10.5.2 钩子执行时机

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         钩子函数执行时机                                     │
│                                                                             │
│  CREATE 操作:                                                                │
│  ──────────                                                                 │
│  BeforeSave → BeforeCreate → [INSERT] → AfterCreate → AfterSave             │
│                                                                             │
│  UPDATE 操作:                                                                │
│  ──────────                                                                 │
│  BeforeSave → BeforeUpdate → [UPDATE] → AfterUpdate → AfterSave             │
│                                                                             │
│  DELETE 操作:                                                                │
│  ──────────                                                                 │
│  BeforeDelete → [DELETE] → AfterDelete                                       │
│                                                                             │
│  QUERY 操作:                                                                │
│  ──────────                                                                 │
│  [SELECT] → AfterFind                                                        │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 10.6 关联处理原理

#### 10.6.1 预加载 (Preload)

```go
// 预加载评论的用户信息
var comments []Comment
db.Preload("User").Where("article_id = ?", articleID).Find(&comments)

// 生成的 SQL:
// SELECT * FROM comments WHERE article_id = 1;
// SELECT * FROM users WHERE id IN (1, 2, 3, ...);
```

#### 10.6.2 嵌套预加载 (Nested Preload)

```go
// 预加载评论及回复
var comments []Comment
db.Preload("Replies").Preload("Replies.User").Where("article_id = ? AND parent_id = ?", articleID, 0).Find(&comments)

// 生成的 SQL:
// SELECT * FROM comments WHERE article_id = 1 AND parent_id = 0;
// SELECT * FROM comments WHERE parent_id IN (1, 2, 3, ...);
// SELECT * FROM users WHERE id IN (1, 2, 3, ...);
```

#### 10.6.3 关联模式

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         GORM 关联处理                                        │
│                                                                             │
│  ┌───────────────────────────────────────────────────────────────────────┐ │
│  │                        一对多 (HasMany) - 自关联                        │ │
│  │                                                                        │ │
│  │   Comment ──────< Comment (回复)                                         │ │
│  │   parent_id                                                          │ │
│  │   db.Preload("Replies").Find(&comments)                                │ │
│  │                                                                        │ │
│  └───────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
│  ┌───────────────────────────────────────────────────────────────────────┐ │
│  │                        多对一 (BelongsTo)                              │ │
│  │                                                                        │ │
│  │   Comment >───── User                                                  │ │
│  │   user_id                                                             │ │
│  │   db.Preload("User").Find(&comments)                                   │ │
│  │                                                                        │ │
│  └───────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 10.7 事务处理

#### 10.7.1 事务 API

```go
// 方式1: Transaction 方法
db.Transaction(func(tx *gorm.DB) error {
    // 创建评论
    if err := tx.Create(&comment).Error; err != nil {
        return err
    }
    
    // 更新文章评论数
    if err := tx.Model(&Article{}).Where("id = ?", comment.ArticleID).
        UpdateColumn("comment_count", gorm.Expr("comment_count + 1")).Error; err != nil {
        return err
    }
    
    return nil
})

// 方式2: 手动控制
tx := db.Begin()
err := tx.Create(&comment).Error
if err != nil {
    tx.Rollback()
    return err
}
tx.Commit()
```

#### 10.7.2 事务隔离级别

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         事务隔离级别                                         │
│                                                                             │
│  MySQL 支持的隔离级别:                                                       │
│  ──────────────────                                                        │
│                                                                             │
│  ┌────────────────────┬─────────────────────────────────────────────────┐   │
│  │  隔离级别          │  说明                                             │   │
│  ├────────────────────┼─────────────────────────────────────────────────┤   │
│  │ READ UNCOMMITTED   │  脏读: 可读取未提交的数据 (一般不推荐)              │   │
│  ├────────────────────┼─────────────────────────────────────────────────┤   │
│  │ READ COMMITTED     │  不可脏读: 只能读取已提交的数据 (Oracle默认)        │   │
│  ├────────────────────┼─────────────────────────────────────────────────┤   │
│  │ REPEATABLE READ    │  可重复读: 事务内多次读取同一数据结果一致 (MySQL默认)│   │
│  ├────────────────────┼─────────────────────────────────────────────────┤   │
│  │ SERIALIZABLE       │  串行化: 完全隔离，最高性能损耗                   │   │
│  └────────────────────┴─────────────────────────────────────────────────┘   │
│                                                                             │
│  GORM 设置隔离级别:                                                         │
│  ──────────────────                                                        │
│                                                                             │
│  tx := db.Session(&gorm.Session{                                           │
│      PrepareStmt: true,                                                    │
│  })                                                                        │
│  tx.Exec("SET TRANSACTION ISOLATION LEVEL READ COMMITTED")                  │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 10.8 软删除与硬删除

#### 10.8.1 软删除实现

```go
// 定义带软删除的模型
type Comment struct {
    ID        uint      `gorm:"primaryKey"`
    Content   string    `gorm:"type:text"`
    DeletedAt gorm.DeletedAt `gorm:"index"`
}

// GORM 会自动过滤已删除的记录
var comments []Comment
db.Find(&comments)  // SELECT * FROM comments WHERE deleted_at IS NULL

// 硬删除
db.Unscoped().Delete(&comment)  // DELETE FROM comments WHERE id = ?

// 查询已删除记录
db.Unscoped().Where("id = ?", id).Find(&comment)
```

#### 10.8.2 软删除原理

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         软删除原理                                           │
│                                                                             │
│  ┌───────────────────────────────────────────────────────────────────────┐ │
│  │                                                                        │ │
│  │   表结构:                                                               │ │
│  │   CREATE TABLE comments (                                             │ │
│  │       id BIGINT PRIMARY KEY,                                          │ │
│  │       content TEXT,                                                   │ │
│  │       deleted_at DATETIME NULL                                        │ │
│  │   );                                                                  │ │
│  │                                                                        │ │
│  └───────────────────────────────────────────────────────────────────────┘ │
│                                    │                                        │
│                                    ▼                                        │
│  ┌───────────────────────────────────────────────────────────────────────┐ │
│  │                                                                        │ │
│  │   删除操作 (软删除):                                                    │ │
│  │   UPDATE comments SET deleted_at = NOW() WHERE id = ?                  │ │
│  │                                                                        │ │
│  │   普通查询:                                                             │ │
│  │   SELECT * FROM comments WHERE deleted_at IS NULL                     │ │
│  │                    AND article_id = ?                                 │ │
│  │                                                                        │ │
│  └───────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 10.9 性能优化建议

| 优化项 | 说明 | 示例 |
|--------|------|------|
| 使用 `Select` 限制字段 | 减少数据传输 | `db.Select("id, content, like_count").Find(&comments)` |
| 使用 `Limit` 限制数量 | 避免全表扫描 | `db.Limit(20).Find(&comments)` |
| 使用索引列查询 | 确保查询走索引 | `Where("article_id = ?", articleID)` |
| 使用 `Preload` 代替 `Joins` | 避免 N+1 问题 | `Preload("User").Find(&comments)` |
| 批量操作使用 `CreateInBatches` | 减少 SQL 执行次数 | `db.CreateInBatches(comments, 100)` |
| 使用 `Count` 统计总数 | 分页场景 | `db.Model(&Comment{}).Where("article_id = ?", aid).Count(&total)` |
| 合理使用软删除 | 保留数据便于审计 | `db.Delete(&comment)` 自动设置 deleted_at |
| 避免深度嵌套预加载 | 性能开销大 | `Preload("Replies.User")` 慎用 |

## 十一、API SQL 与索引分析

### 10.1 获取评论详情 (GetComment)

**执行的SQL**:
```sql
SELECT * FROM comments WHERE id = ? LIMIT 1
```

| SQL条件 | 命中索引 | 说明 |
|---------|----------|------|
| `id = ?` | PRIMARY (id) | 主键索引，O(1) 时间复杂度 |

---

### 10.2 获取文章评论列表 (GetArticleComments)

**执行的SQL**:
```sql
-- 查询顶级评论 (parent_id = 0)
SELECT * FROM comments 
WHERE article_id = ? AND parent_id = 0
ORDER BY created_at DESC
LIMIT ? OFFSET ?

-- 如果 include_replies=true，查询所有回复
SELECT * FROM comments 
WHERE article_id = ? AND parent_id IN (?, ?, ...)
ORDER BY created_at ASC
```

| SQL条件 | 命中索引 | 说明 |
|---------|----------|------|
| `article_id = ?` | idx_article_id | 文章索引 |
| `parent_id = 0` | idx_parent_id | 顶级评论筛选 |
| `ORDER BY created_at` | idx_created_at | 创建时间排序 |
| `parent_id IN (...)` | idx_parent_id | 批量查询回复 |

**优化说明**: 顶级评论 + 排序，可创建复合索引 `INDEX idx_article_parent_created (article_id, parent_id, created_at)` 进一步优化。

---

### 10.3 获取评论回复 (GetCommentReplies)

**执行的SQL**:
```sql
SELECT * FROM comments 
WHERE parent_id = ? AND status = 1
ORDER BY created_at ASC
LIMIT ? OFFSET ?
```

| SQL条件 | 命中索引 | 说明 |
|---------|----------|------|
| `parent_id = ?` | idx_parent_id | 父评论索引 |
| `status = 1` | idx_status | 状态筛选 |
| `ORDER BY created_at` | idx_created_at | 创建时间排序 |

**优化建议**: 可创建复合索引 `INDEX idx_parent_status_created (parent_id, status, created_at)` 优化此查询。

---

### 10.4 创建评论 (CreateComment)

**执行的SQL**:
```sql
-- 1. 创建评论
INSERT INTO comments (article_id, user_id, parent_id, content, ...) VALUES (?, ?, ?, ?, ...)

-- 2. 更新父评论回复数 (如果有父评论)
UPDATE comments SET reply_count = reply_count + 1 WHERE id = ?

-- 3. 更新文章评论数
UPDATE articles SET comment_count = comment_count + 1 WHERE id = ?
```

| SQL操作 | 命中索引 | 说明 |
|---------|----------|------|
| INSERT comments | PRIMARY (id) | 主键自增 |
| `comments.id = ?` | PRIMARY (comments) | 更新回复数 |
| `articles.id = ?` | PRIMARY (articles) | 更新评论数 |

---

### 10.5 更新评论 (UpdateComment)

**执行的SQL**:
```sql
UPDATE comments SET content = ? WHERE id = ? AND user_id = ?
```

| SQL条件 | 命中索引 | 说明 |
|---------|----------|------|
| `id = ? AND user_id = ?` | PRIMARY + idx_user_id | 需要验证用户权限 |

---

### 10.6 删除评论 (DeleteComment)

**执行的SQL**:
```sql
-- 1. 获取评论信息
SELECT * FROM comments WHERE id = ? LIMIT 1

-- 2. 删除评论 (软删除，更新status)
UPDATE comments SET status = 0 WHERE id = ?

-- 3. 更新父评论回复数
UPDATE comments SET reply_count = reply_count - 1 WHERE id = ?

-- 4. 更新文章评论数
UPDATE articles SET comment_count = comment_count - 1 WHERE id = ?
```

| SQL操作 | 命中索引 | 说明 |
|---------|----------|------|
| `id = ?` | PRIMARY (comments) | 查询评论 |
| `comments.id = ?` | PRIMARY (comments) | 更新状态 |
| `comments.id = ?` | PRIMARY (comments) | 更新回复数 |
| `articles.id = ?` | PRIMARY (articles) | 更新评论数 |

---

### 10.7 点赞评论 (LikeComment)

**执行的SQL**:
```sql
-- 1. 检查是否已点赞
SELECT * FROM comment_likes WHERE comment_id = ? AND user_id = ? LIMIT 1

-- 2. 如果未点赞，添加点赞记录
INSERT INTO comment_likes (comment_id, user_id) VALUES (?, ?)

-- 3. 更新评论点赞数
UPDATE comments SET like_count = like_count + 1 WHERE id = ?
```

| SQL条件 | 命中索引 | 说明 |
|---------|----------|------|
| `comment_id = ? AND user_id = ?` | uk_comment_user | 唯一索引，检查重复点赞 |
| INSERT comment_likes | PRIMARY + uk_comment_user | 主键自增 + 唯一性约束 |
| `comments.id = ?` | PRIMARY (comments) | 更新点赞数 |

---

### 10.8 获取用户评论列表 (ListComments)

**执行的SQL**:
```sql
SELECT * FROM comments 
WHERE user_id = ?
ORDER BY created_at DESC
LIMIT ? OFFSET ?
```

| SQL条件 | 命中索引 | 说明 |
|---------|----------|------|
| `user_id = ?` | idx_user_id | 用户索引 |
| `ORDER BY created_at` | idx_created_at | 创建时间排序 |

**优化建议**: 可创建复合索引 `INDEX idx_user_created (user_id, created_at)` 优化此查询。

---

### 10.9 开启/关闭评论 (EnableComment/DisableComment)

**执行的SQL**:
```sql
UPDATE articles SET allow_comment = ? WHERE id = ?
```

| SQL条件 | 命中索引 | 说明 |
|---------|----------|------|
| `id = ?` | PRIMARY (articles) | 主键索引，快速定位 |
