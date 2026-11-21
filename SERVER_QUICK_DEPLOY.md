# 服务器快速部署指南

## 前置条件

确保服务器已安装：
- Docker 20.10+
- Docker Compose 2.0+
- Git

## 部署步骤

### 1. 拉取代码

```bash
cd /opt  # 或您选择的部署目录
git clone git@github.com:JarJiang/sub2api-jrw.git
cd sub2api-jrw
```

如果已克隆,则更新：

```bash
cd /opt/sub2api-jrw
git pull origin main
```

### 2. 配置环境变量

```bash
cp .env.pi.example .env.pi
vi .env.pi  # 或使用 nano
```

**必须修改的配置：**

```bash
# 数据库密码（必改）
POSTGRES_PASSWORD_PI=your_secure_password_here

# Redis 密码（必改）
REDIS_PASSWORD_PI=your_secure_redis_password

# Session 密钥（必改，生成方法见下）
SESSION_SECRET=your_session_secret_32_chars_min

# JWT 密钥（必改，生成方法见下）
JWT_SECRET=your_jwt_secret_32_chars_min
```

**生成安全密钥：**

```bash
# 生成 SESSION_SECRET
openssl rand -base64 32

# 生成 JWT_SECRET
openssl rand -base64 32
```

**其他配置：**

根据需要调整端口、超时等参数。默认配置适用于大多数场景。

### 3. 执行部署

```bash
chmod +x deploy-prompt-injection.sh
./deploy-prompt-injection.sh
```

部署脚本会自动：
- 创建数据目录
- 检查 Docker 网络
- 构建镜像
- 启动服务（PostgreSQL, Redis, sub2api）

### 4. 验证部署

#### 检查容器状态

```bash
docker-compose -f docker-compose.prompt-injection.yml ps
```

应看到所有容器状态为 `Up`：
```
sub2api-pi          Up      8081/tcp
postgres-pi         Up      5432/tcp
redis-pi            Up      6379/tcp
```

#### 检查健康状态

```bash
curl http://localhost:8081/health
```

期望输出：
```json
{"status":"ok"}
```

#### 查看日志

```bash
# 所有服务日志
docker-compose -f docker-compose.prompt-injection.yml logs

# 仅 sub2api 日志
docker-compose -f docker-compose.prompt-injection.yml logs sub2api

# 实时跟踪日志
docker-compose -f docker-compose.prompt-injection.yml logs -f sub2api
```

### 5. 功能测试

#### 创建测试 API Key

访问前端：`http://your-server:8081`

1. 登录/注册账户
2. 进入 API Keys 管理页面
3. 创建新的 API Key
4. **启用"提示词注入"切换开关**

#### 测试请求

```bash
# 替换 YOUR_API_KEY 为实际的 API key
curl -X POST http://localhost:8081/v1/chat/completions \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [
      {
        "role": "user",
        "content": "用 Python 和 Django 实现一个用户登录功能"
      }
    ]
  }'
```

**验证要点：**
- 响应中应包含 Django 最佳实践的提示内容
- 响应中应包含 Python 编码规范
- 响应中应包含安全相关的建议（如密码哈希）

#### 对比测试

创建一个**未启用**提示词注入的 API Key，用相同请求测试，对比响应差异。

### 6. 监控和维护

#### 磁盘空间监控

```bash
du -sh data_pi/
```

#### 资源使用监控

```bash
docker stats sub2api-pi postgres-pi redis-pi
```

#### 日志轮转

编辑 `/etc/docker/daemon.json`：

```json
{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "10m",
    "max-file": "3"
  }
}
```

重启 Docker：

```bash
systemctl restart docker
```

### 7. 更新部署

```bash
cd /opt/sub2api-jrw
git pull origin main
./deploy-prompt-injection.sh
```

## 故障排除

### 容器无法启动

```bash
# 查看详细错误
docker-compose -f docker-compose.prompt-injection.yml logs sub2api

# 检查网络
docker network inspect sub2api-network

# 重新创建网络
docker network rm sub2api-network
docker network create sub2api-network
```

### 数据库连接失败

```bash
# 检查 PostgreSQL 日志
docker-compose -f docker-compose.prompt-injection.yml logs postgres

# 验证连接
docker exec -it postgres-pi psql -U postgres -d sub2api_pi -c "SELECT 1;"
```

### 端口冲突

编辑 `.env.pi`，修改 `SERVER_PORT=8081` 为其他可用端口。

### 性能问题

检查资源使用：

```bash
docker stats --no-stream
```

如需调整资源限制，编辑 `docker-compose.prompt-injection.yml` 的 `deploy.resources` 部分。

## 回滚

```bash
# 停止测试环境
docker-compose -f docker-compose.prompt-injection.yml down

# 回退代码
git log --oneline -5  # 查看提交历史
git checkout <previous-commit-hash>

# 重新部署
./deploy-prompt-injection.sh
```

## 安全建议

1. **修改所有默认密码**：`.env.pi` 中的所有密码必须更改
2. **防火墙配置**：仅开放必要端口（8081 用于外部访问）
3. **定期备份**：备份 `data_pi/` 目录
4. **SSL/TLS**：生产环境使用 Nginx/Caddy 反向代理并启用 HTTPS
5. **日志审计**：定期检查访问日志和错误日志

## 生产环境建议

- 使用外部 PostgreSQL 和 Redis（更高可用性）
- 配置 Nginx 反向代理
- 启用 HTTPS（Let's Encrypt）
- 配置日志收集（ELK/Grafana Loki）
- 设置告警（Prometheus + Alertmanager）

## 联系支持

如遇到问题：
1. 查看 [SERVER_DEPLOYMENT_CHECKLIST.md](./SERVER_DEPLOYMENT_CHECKLIST.md) 完整检查清单
2. 查看 [DEPLOYMENT.md](./DEPLOYMENT.md) 详细部署文档
3. 检查 GitHub Issues

---

部署完成后，记得验证：
✅ 健康检查通过
✅ 前端可访问
✅ API Key 创建成功
✅ 提示词注入功能正常工作
