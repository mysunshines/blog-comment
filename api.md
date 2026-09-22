# comment-service 对外 API 文档

> 自动生成自 `comment.proto`（模式：proto）。
> 网关按 `/api/v1/comment/<snake_method>` 反射代理到 gRPC 方法 `comment.v1.CommentService/<Method>`。
> 生成时间：2026-09-21 19:37:19
> Base URL（网关入口）：http://localhost:8081

## 接口列表

| Method | Path | 鉴权 | 说明 |
| --- | --- | --- | --- |
| `POST` | `/api/v1/comment/create_comment` | 公开 |  |
| `GET` | `/api/v1/comment/get_comment` | 公开 |  |
| `PUT` | `/api/v1/comment/update_comment` | 公开 |  |
| `DELETE` | `/api/v1/comment/delete_comment` | 公开 |  |
| `GET` | `/api/v1/comment/list_comments` | 公开 |  |
| `GET` | `/api/v1/comment/get_article_comments` | 公开 |  |
| `GET` | `/api/v1/comment/get_article_annotations` | 公开 | 轻量接口：仅返回某文章的行内批注锚点（供 article-service 渲染时注入高亮标记，避免拉取整棵评论树）。 |
| `POST` | `/api/v1/comment/reply_comment` | 公开 |  |
| `POST` | `/api/v1/comment/like_comment` | 公开 |  |
| `GET` | `/api/v1/comment/get_comment_replies` | 公开 |  |
| `POST` | `/api/v1/comment/enable_comment` | 公开 |  |
| `POST` | `/api/v1/comment/disable_comment` | 公开 |  |
| `POST` | `/api/v1/comment/admin_list_comments` | 管理员 |  |
| `POST` | `/api/v1/comment/admin_delete_comment` | 公开 |  |

## CreateComment

- **URL**: `http://localhost:8081/api/v1/comment/create_comment`
- **Method**: `POST`
- **鉴权**: 公开（无需鉴权）

### Headers
```http
Content-Type: application/json
```

### Request
**参数位置**：Request Body（JSON）

| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `article_id` | `uint32` |  | `0` |
| `user_id` | `uint32` |  | `0` |
| `content` | `string` |  | `""` |
| `parent_id` | `uint32` |  | `0` |
| `paragraph_index` | `int32` |  | `0` |
| `anchor_text` | `string` |  | `""` |
| `anchor_offset` | `int32` |  | `0` |
| `anchor_prefix` | `string` |  | `""` |
| `anchor_suffix` | `string` |  | `""` |

**Body 示例**：
```json
{"article_id": 0, "user_id": 0, "content": "", "parent_id": 0, "paragraph_index": 0, "anchor_text": "", "anchor_offset": 0, "anchor_prefix": "", "anchor_suffix": ""}
```

### Response
| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `code` | `uint32` |  | `0` |
| `message` | `string` |  | `""` |
| `comment` | `Comment` |  | 见 [Comment](#comment) |

**Response 示例**：
```json
{"code": 0, "message": "success", "comment": {}}
```

### curl 示例
```bash
curl -X POST 'http://localhost:8081/api/v1/comment/create_comment' \
  -H 'Content-Type: application/json' \
  -d '{"article_id": 0, "user_id": 0, "content": "", "parent_id": 0, "paragraph_index": 0, "anchor_text": "", "anchor_offset": 0, "anchor_prefix": "", "anchor_suffix": ""}'
```

## GetComment

- **URL**: `http://localhost:8081/api/v1/comment/get_comment?comment_id=0`
- **Method**: `GET`
- **鉴权**: 公开（无需鉴权）

### Headers
```http
Content-Type: application/json
```

### Request
**参数位置**：Query String

| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `comment_id` | `uint32` |  | `0` |

**Query 示例**：
```json
comment_id=0
```

### Response
| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `code` | `uint32` |  | `0` |
| `message` | `string` |  | `""` |
| `comment` | `Comment` |  | 见 [Comment](#comment) |

**Response 示例**：
```json
{"code": 0, "message": "success", "comment": {}}
```

### curl 示例
```bash
curl -X GET 'http://localhost:8081/api/v1/comment/get_comment?comment_id=0'
```

## UpdateComment

- **URL**: `http://localhost:8081/api/v1/comment/update_comment`
- **Method**: `PUT`
- **鉴权**: 公开（无需鉴权）

### Headers
```http
Content-Type: application/json
```

### Request
**参数位置**：Request Body（JSON）

| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `comment_id` | `uint32` |  | `0` |
| `user_id` | `uint32` |  | `0` |
| `content` | `string` |  | `""` |

**Body 示例**：
```json
{"comment_id": 0, "user_id": 0, "content": ""}
```

### Response
| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `code` | `uint32` |  | `0` |
| `message` | `string` |  | `""` |
| `comment` | `Comment` |  | 见 [Comment](#comment) |

**Response 示例**：
```json
{"code": 0, "message": "success", "comment": {}}
```

### curl 示例
```bash
curl -X PUT 'http://localhost:8081/api/v1/comment/update_comment' \
  -H 'Content-Type: application/json' \
  -d '{"comment_id": 0, "user_id": 0, "content": ""}'
```

## DeleteComment

- **URL**: `http://localhost:8081/api/v1/comment/delete_comment`
- **Method**: `DELETE`
- **鉴权**: 公开（无需鉴权）

### Headers
```http
Content-Type: application/json
```

### Request
**参数位置**：Request Body（JSON）

| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `comment_id` | `uint32` |  | `0` |
| `user_id` | `uint32` |  | `0` |
| `is_admin` | `uint32` |  | `0` |

**Body 示例**：
```json
{"comment_id": 0, "user_id": 0, "is_admin": 0}
```

### Response
| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `code` | `uint32` |  | `0` |
| `message` | `string` |  | `""` |

**Response 示例**：
```json
{"code": 0, "message": "success"}
```

### curl 示例
```bash
curl -X DELETE 'http://localhost:8081/api/v1/comment/delete_comment' \
  -H 'Content-Type: application/json' \
  -d '{"comment_id": 0, "user_id": 0, "is_admin": 0}'
```

## ListComments

- **URL**: `http://localhost:8081/api/v1/comment/list_comments?page=0&page_size=0&user_id=0`
- **Method**: `GET`
- **鉴权**: 公开（无需鉴权）

### Headers
```http
Content-Type: application/json
```

### Request
**参数位置**：Query String

| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `page` | `uint32` |  | `0` |
| `page_size` | `uint32` |  | `0` |
| `user_id` | `uint32` |  | `0` |

**Query 示例**：
```json
page=0&page_size=0&user_id=0
```

### Response
| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `code` | `uint32` |  | `0` |
| `message` | `string` |  | `""` |
| `comments` | `Comment[]` |  | [] |
| `total` | `uint32` |  | `0` |

**Response 示例**：
```json
{"code": 0, "message": "success", "comments": [], "total": 0}
```

### curl 示例
```bash
curl -X GET 'http://localhost:8081/api/v1/comment/list_comments?page=0&page_size=0&user_id=0'
```

## GetArticleComments

- **URL**: `http://localhost:8081/api/v1/comment/get_article_comments?article_id=0&page=0&page_size=0&include_replies=false&sort=<sort>`
- **Method**: `GET`
- **鉴权**: 公开（无需鉴权）

### Headers
```http
Content-Type: application/json
```

### Request
**参数位置**：Query String

| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `article_id` | `uint32` |  | `0` |
| `page` | `uint32` |  | `0` |
| `page_size` | `uint32` |  | `0` |
| `include_replies` | `bool` |  | `false` |
| `sort` | `string` |  | `""` |

**Query 示例**：
```json
article_id=0&page=0&page_size=0&include_replies=false&sort=<sort>
```

### Response
| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `code` | `uint32` |  | `0` |
| `message` | `string` |  | `""` |
| `comments` | `Comment[]` |  | [] |
| `total` | `uint32` |  | `0` |
| `comment_enabled` | `bool` |  | `false` |

**Response 示例**：
```json
{"code": 0, "message": "success", "comments": [], "total": 0, "comment_enabled": false}
```

### curl 示例
```bash
curl -X GET 'http://localhost:8081/api/v1/comment/get_article_comments?article_id=0&page=0&page_size=0&include_replies=false&sort=<sort>'
```

## GetArticleAnnotations

- **URL**: `http://localhost:8081/api/v1/comment/get_article_annotations?article_id=0`
- **Method**: `GET`
- **鉴权**: 公开（无需鉴权）

### Headers
```http
Content-Type: application/json
```

### Request
**参数位置**：Query String

| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `article_id` | `uint32` |  | `0` |

**Query 示例**：
```json
article_id=0
```

### Response
| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `code` | `uint32` |  | `0` |
| `message` | `string` |  | `""` |
| `annotations` | `Annotation[]` |  | [] |

**Response 示例**：
```json
{"code": 0, "message": "success", "annotations": []}
```

### curl 示例
```bash
curl -X GET 'http://localhost:8081/api/v1/comment/get_article_annotations?article_id=0'
```

## ReplyComment

- **URL**: `http://localhost:8081/api/v1/comment/reply_comment`
- **Method**: `POST`
- **鉴权**: 公开（无需鉴权）

### Headers
```http
Content-Type: application/json
```

### Request
**参数位置**：Request Body（JSON）

| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `comment_id` | `uint32` |  | `0` |
| `user_id` | `uint32` |  | `0` |
| `content` | `string` |  | `""` |

**Body 示例**：
```json
{"comment_id": 0, "user_id": 0, "content": ""}
```

### Response
| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `code` | `uint32` |  | `0` |
| `message` | `string` |  | `""` |
| `reply` | `Comment` |  | 见 [Comment](#comment) |

**Response 示例**：
```json
{"code": 0, "message": "success", "reply": {}}
```

### curl 示例
```bash
curl -X POST 'http://localhost:8081/api/v1/comment/reply_comment' \
  -H 'Content-Type: application/json' \
  -d '{"comment_id": 0, "user_id": 0, "content": ""}'
```

## LikeComment

- **URL**: `http://localhost:8081/api/v1/comment/like_comment`
- **Method**: `POST`
- **鉴权**: 公开（无需鉴权）

### Headers
```http
Content-Type: application/json
```

### Request
**参数位置**：Request Body（JSON）

| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `comment_id` | `uint32` |  | `0` |
| `user_id` | `uint32` |  | `0` |

**Body 示例**：
```json
{"comment_id": 0, "user_id": 0}
```

### Response
| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `code` | `uint32` |  | `0` |
| `message` | `string` |  | `""` |
| `like_count` | `uint32` |  | `0` |
| `liked` | `bool` | 操作后的点赞状态：true=已点赞，false=已取消点赞 | `false` |

**Response 示例**：
```json
{"code": 0, "message": "success", "like_count": 0, "liked": false}
```

### curl 示例
```bash
curl -X POST 'http://localhost:8081/api/v1/comment/like_comment' \
  -H 'Content-Type: application/json' \
  -d '{"comment_id": 0, "user_id": 0}'
```

## GetCommentReplies

- **URL**: `http://localhost:8081/api/v1/comment/get_comment_replies?comment_id=0&page=0&page_size=0`
- **Method**: `GET`
- **鉴权**: 公开（无需鉴权）

### Headers
```http
Content-Type: application/json
```

### Request
**参数位置**：Query String

| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `comment_id` | `uint32` |  | `0` |
| `page` | `uint32` |  | `0` |
| `page_size` | `uint32` |  | `0` |

**Query 示例**：
```json
comment_id=0&page=0&page_size=0
```

### Response
| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `code` | `uint32` |  | `0` |
| `message` | `string` |  | `""` |
| `replies` | `Comment[]` |  | [] |
| `total` | `uint32` |  | `0` |

**Response 示例**：
```json
{"code": 0, "message": "success", "replies": [], "total": 0}
```

### curl 示例
```bash
curl -X GET 'http://localhost:8081/api/v1/comment/get_comment_replies?comment_id=0&page=0&page_size=0'
```

## EnableComment

- **URL**: `http://localhost:8081/api/v1/comment/enable_comment`
- **Method**: `POST`
- **鉴权**: 公开（无需鉴权）

### Headers
```http
Content-Type: application/json
```

### Request
**参数位置**：Request Body（JSON）

| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `article_id` | `uint32` |  | `0` |
| `user_id` | `uint32` |  | `0` |

**Body 示例**：
```json
{"article_id": 0, "user_id": 0}
```

### Response
| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `code` | `uint32` |  | `0` |
| `message` | `string` |  | `""` |

**Response 示例**：
```json
{"code": 0, "message": "success"}
```

### curl 示例
```bash
curl -X POST 'http://localhost:8081/api/v1/comment/enable_comment' \
  -H 'Content-Type: application/json' \
  -d '{"article_id": 0, "user_id": 0}'
```

## DisableComment

- **URL**: `http://localhost:8081/api/v1/comment/disable_comment`
- **Method**: `POST`
- **鉴权**: 公开（无需鉴权）

### Headers
```http
Content-Type: application/json
```

### Request
**参数位置**：Request Body（JSON）

| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `article_id` | `uint32` |  | `0` |
| `user_id` | `uint32` |  | `0` |

**Body 示例**：
```json
{"article_id": 0, "user_id": 0}
```

### Response
| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `code` | `uint32` |  | `0` |
| `message` | `string` |  | `""` |

**Response 示例**：
```json
{"code": 0, "message": "success"}
```

### curl 示例
```bash
curl -X POST 'http://localhost:8081/api/v1/comment/disable_comment' \
  -H 'Content-Type: application/json' \
  -d '{"article_id": 0, "user_id": 0}'
```

## AdminListComments

- **URL**: `http://localhost:8081/api/v1/comment/admin_list_comments`
- **Method**: `POST`
- **鉴权**: 管理员（需 JWT + 管理员角色）

### Headers
```http
Authorization: Bearer <token>
Content-Type: application/json
```

### Request
**参数位置**：Request Body（JSON）

| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `page` | `uint32` |  | `0` |
| `page_size` | `uint32` |  | `0` |
| `article_id` | `uint32` |  | `0` |
| `user_id` | `uint32` |  | `0` |
| `keyword` | `string` |  | `""` |

**Body 示例**：
```json
{"page": 0, "page_size": 0, "article_id": 0, "user_id": 0, "keyword": ""}
```

### Response
| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `code` | `uint32` |  | `0` |
| `message` | `string` |  | `""` |
| `comments` | `Comment[]` |  | [] |
| `total` | `uint32` |  | `0` |

**Response 示例**：
```json
{"code": 0, "message": "success", "comments": [], "total": 0}
```

### curl 示例
```bash
curl -X POST 'http://localhost:8081/api/v1/comment/admin_list_comments' \
  -H 'Authorization: Bearer <token>' \
  -H 'Content-Type: application/json' \
  -d '{"page": 0, "page_size": 0, "article_id": 0, "user_id": 0, "keyword": ""}'
```

## AdminDeleteComment

- **URL**: `http://localhost:8081/api/v1/comment/admin_delete_comment`
- **Method**: `POST`
- **鉴权**: 公开（无需鉴权）

### Headers
```http
Content-Type: application/json
```

### Request
**参数位置**：Request Body（JSON）

| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `comment_id` | `uint32` |  | `0` |

**Body 示例**：
```json
{"comment_id": 0}
```

### Response
| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `code` | `uint32` |  | `0` |
| `message` | `string` |  | `""` |

**Response 示例**：
```json
{"code": 0, "message": "success"}
```

### curl 示例
```bash
curl -X POST 'http://localhost:8081/api/v1/comment/admin_delete_comment' \
  -H 'Content-Type: application/json' \
  -d '{"comment_id": 0}'
```

---

## 数据结构

> 下列 message / enum 被上述接口的请求或响应引用；结构体字段中的 message 类型可点击跳转到对应定义。

### Comment

| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `id` | `uint32` |  | `0` |
| `article_id` | `uint32` |  | `0` |
| `user_id` | `uint32` |  | `0` |
| `username` | `string` |  | `""` |
| `nickname` | `string` |  | `""` |
| `avatar` | `string` |  | `""` |
| `parent_id` | `uint32` |  | `0` |
| `content` | `string` |  | `""` |
| `like_count` | `uint32` |  | `0` |
| `reply_count` | `uint32` |  | `0` |
| `status` | `uint32` |  | `0` |
| `created_at` | `string` |  | `""` |
| `updated_at` | `string` |  | `""` |
| `replies` | `Comment[]` |  | [] |
| `paragraph_index` | `int32` |  | `0` |
| `anchor_text` | `string` |  | `""` |
| `anchor_offset` | `int32` |  | `0` |
| `anchor_prefix` | `string` |  | `""` |
| `anchor_suffix` | `string` |  | `""` |

### Annotation

| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `comment_id` | `uint32` |  | `0` |
| `paragraph_index` | `int32` |  | `0` |
| `anchor_text` | `string` |  | `""` |
| `anchor_offset` | `int32` |  | `0` |
| `anchor_prefix` | `string` |  | `""` |
| `anchor_suffix` | `string` |  | `""` |

