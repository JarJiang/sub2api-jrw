# 服务器部署检查清单

## 部署前准备

### 1. 服务器环境检查
```bash
# 检查 Docker 版本
docker --version  # 需要 20.10+

# 检查 Docker Compose 版本
docker compose version  # 需要 v2.0+

# 检查 Git
git --version

# 检查磁盘空间 (建议至少 10GB 可用)
df -h
```

### 2. 拉取最新代码
```bash
# 如果是首次部署
git clone git@github.com:JarJiang/sub2api-jrw.git
cd sub2api-jrw

# 如果已有仓库
cd sub2api-jrw
git pull origin main

# 验证最新提交
git log --oneline -3
# 应该看到:
# 8b1467339 chore: update Ent ORM generated files for prompt_injection_enabled field
# 4187d8ca9 feat: add intelligent dynamic prompt injection system
```

### 3. 配置环境变量
```bash
# 复制环境变量模板
cp .env.pi.example .env.pi

# 编辑环境变量 (重要：必须修改以下密码和密钥)
nano .env.pi  # 或使用 vim

# 必须修改的配置项:
# - POSTGRES_PASSWORD_PI (PostgreSQL 密码)
# - REDIS_PASSWORD_PI (Redis 密码)
# - DATABASE_URL (包含 PostgreSQL 密码)
# - REDIS_URL (包含 Redis 密码)
# - SESSION_SECRET (随机生成，至少 32 字符)
# - JWT_SECRET (随机生成，至少 32 字符)
```

### 4. 生成安全密钥 (推荐)
```bash
# 生成 SESSION_SECRET
openssl rand -base64 32

# 生成 JWT_SECRET
openssl rand -base64 32

# 生成强密码
openssl rand -base64 16
```

## 部署执行

### 5. 运行部署脚本
```bash
# 赋予执行权限
chmod +x deploy-prompt-injection.sh

# 执行部署
./deploy-prompt-injection.sh

# 脚本会自动:
# - 检查环境变量文件
# - 停止旧容器
# - 构建新镜像
# - 启动测试环境容器
# - 运行健康检查
```

## 部署后验证

### 6. 验证生产环境 (端口 8080)
```bash
# 检查生产环境容器状态
docker ps | grep -E 'sub2api|postgres|redis' | grep -v '_pi'

# 访问生产环境健康检查
curl http://localhost:8080/health

# 预期响应: {"status":"ok"}
```

### 7. 验证测试环境 (端口 8081)
```bash
# 检查测试环境容器状态
docker ps | grep '_pi'

# 应该看到 3 个运行中的容器:
# - sub2api-backend-pi
# - sub2api-postgres-pi
# - sub2api-redis-pi

# 访问测试环境健康检查
curl http://localhost:8081/health

# 预期响应: {"status":"ok"}

# 检查前端是否可访问
curl -I http://localhost:8081/

# 预期: HTTP/1.1 200 OK
```

### 8. 查看容器日志
```bash
# 查看后端日志
docker logs sub2api-backend-pi --tail 50

# 查看 PostgreSQL 日志
docker logs sub2api-postgres-pi --tail 20

# 查看 Redis 日志
docker logs sub2api-redis-pi --tail 20

# 实时跟踪后端日志
docker logs -f sub2api-backend-pi
```

### 9. 验证数据库连接
```bash
# 进入后端容器
docker exec -it sub2api-backend-pi sh

# 在容器内检查环境变量
echo $DATABASE_URL

# 退出容器
exit
```

### 10. 验证网络隔离
```bash
# 检查 Docker 网络
docker network ls | grep sub2api

# 检查测试环境容器的网络
docker inspect sub2api-backend-pi | grep -A 10 Networks
```

## 功能测试

### 11. 测试 Prompt Injection 功能

#### A. 创建 API 密钥 (启用 prompt injection)
```bash
# 访问前端创建密钥页面
# http://your-server:8081/

# 或使用 API 创建密钥
curl -X POST http://localhost:8081/api/keys \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-key-with-injection",
    "prompt_injection_enabled": true
  }'
```

#### B. 测试聊天完成 API (带 prompt injection)
```bash
# 测试 Python + Django + debugging
curl -X POST http://localhost:8081/v1/chat/completions \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [{"role": "user", "content": "帮我调试这段 Django 代码"}],
    "language": "python",
    "framework": "django",
    "task_type": "debugging"
  }'

# 检查响应中是否包含注入的提示词
```

#### C. 测试聊天完成 API (不带 prompt injection)
```bash
# 创建不启用 prompt injection 的密钥
curl -X POST http://localhost:8081/api/keys \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-key-without-injection",
    "prompt_injection_enabled": false
  }'

# 使用该密钥测试
curl -X POST http://localhost:8081/v1/chat/completions \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [{"role": "user", "content": "Hello"}]
  }'

# 响应应该不包含注入的提示词
```

### 12. 测试前端切换功能
```bash
# 在浏览器中访问
http://your-server:8081/

# 测试步骤:
# 1. 登录/注册账户
# 2. 进入 API 密钥管理页面
# 3. 创建新密钥时，观察 "Prompt Injection" 切换按钮
# 4. 切换按钮，观察状态变化
# 5. 保存密钥
# 6. 编辑已有密钥，测试切换功能
```

## 监控和维护

### 13. 设置日志轮转
```bash
# 检查 Docker 日志大小
docker ps -q | xargs docker inspect --format='{{.Name}} {{.HostConfig.LogConfig.Type}}' 

# 配置日志限制 (在 docker-compose.prompt-injection.yml 中已配置)
# logging:
#   driver: "json-file"
#   options:
#     max-size: "10m"
#     max-file: "3"
```

### 14. 定期备份
```bash
# 备份 PostgreSQL 数据
docker exec sub2api-postgres-pi pg_dump -U sub2api sub2api_pi > backup_$(date +%Y%m%d).sql

# 备份 Redis 数据
docker exec sub2api-redis-pi redis-cli --pass YOUR_REDIS_PASSWORD SAVE
docker cp sub2api-redis-pi:/data/dump.rdb redis_backup_$(date +%Y%m%d).rdb
```

### 15. 资源监控
```bash
# 查看容器资源使用情况
docker stats sub2api-backend-pi sub2api-postgres-pi sub2api-redis-pi

# 查看磁盘使用情况
du -sh data_pi/*
```

## 故障排查

### 常见问题

#### 问题 1: 容器启动失败
```bash
# 检查日志
docker logs sub2api-backend-pi

# 常见原因:
# - 环境变量配置错误
# - 端口被占用
# - 数据库连接失败

# 解决方法:
# - 检查 .env.pi 文件
# - 检查端口 8081 是否被占用: netstat -tuln | grep 8081
# - 检查数据库密码是否正确
```

#### 问题 2: 无法访问服务
```bash
# 检查防火墙
sudo ufw status
sudo ufw allow 8081/tcp

# 检查容器网络
docker network inspect sub2api-network

# 检查容器端口映射
docker port sub2api-backend-pi
```

#### 问题 3: 数据库连接错误
```bash
# 进入数据库容器
docker exec -it sub2api-postgres-pi psql -U sub2api -d sub2api_pi

# 检查数据库表
\dt

# 应该看到包含 api_keys 表及 prompt_injection_enabled 字段
\d api_keys

# 退出
\q
```

#### 问题 4: Prompt Injection 不生效
```bash
# 检查后端日志中的 prompt injection 相关输出
docker logs sub2api-backend-pi | grep -i "prompt"

# 检查 API 密钥的 prompt_injection_enabled 字段
docker exec -it sub2api-postgres-pi psql -U sub2api -d sub2api_pi \
  -c "SELECT id, name, prompt_injection_enabled FROM api_keys;"

# 检查请求头中的 language, framework, task_type 参数
```

## 回滚操作

### 如果需要回滚
```bash
# 停止测试环境
docker compose -f docker-compose.prompt-injection.yml down

# 删除数据目录 (谨慎操作)
rm -rf data_pi/

# 回到之前的版本
git checkout <previous-commit-hash>

# 重新部署
./deploy-prompt-injection.sh
```

## 完成清单

部署完成后，确认以下各项:

- [ ] 生产环境 (8080) 正常运行
- [ ] 测试环境 (8081) 正常运行
- [ ] 测试环境健康检查通过
- [ ] 数据库表结构包含 prompt_injection_enabled 字段
- [ ] 创建 API 密钥功能正常
- [ ] Prompt Injection 切换按钮可见且可操作
- [ ] 启用 prompt injection 的密钥请求包含注入内容
- [ ] 未启用 prompt injection 的密钥请求不包含注入内容
- [ ] 前端页面正常显示
- [ ] 容器日志无错误
- [ ] 资源使用正常
- [ ] 备份策略已配置

## 联系支持

如遇到问题，请提供:
1. 容器日志: `docker logs sub2api-backend-pi > logs.txt`
2. 容器状态: `docker ps -a > containers.txt`
3. 环境变量 (脱敏): `.env.pi` (移除敏感信息)
4. 错误截图或消息

---
最后更新: 2024-11-21
版本: 1.0.0
