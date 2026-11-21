# Prompt Injection 功能部署指南

## 概述

本文档说明如何在服务器上部署带有智能动态提示词注入功能的 Sub2API 测试环境，与生产环境并行运行，互不干扰。

## 架构说明

- **生产环境**: 端口 8080，使用 `docker-compose.yml`
- **测试环境**: 端口 8081，使用 `docker-compose.prompt-injection.yml`
- **隔离方式**: 独立容器名、数据目录、端口，共享网络

## 前置条件

服务器需要安装:
- Docker (20.10+)
- Docker Compose (v2.0+)
- Git

## 部署步骤

### 1. 准备源代码

在本地仓库中，已完成以下文件:

```
D:\sub2api\sub2api-jrw\
├── Dockerfile                              # 多阶段构建配置
├── docker-compose.prompt-injection.yml     # 测试环境配置
├── .env.pi.example                         # 环境变量模板
└── deploy-prompt-injection.sh              # 部署脚本
```

### 2. 传输文件到服务器

#### 方案 A: 使用 Git (推荐)

```bash
# 在本地提交所有更改
git add .
git commit -m "feat: add prompt injection feature with parallel deployment"
git push origin main

# 在服务器上拉取
ssh user@your-server
cd /path/to/sub2api
git pull origin main
```

#### 方案 B: 使用 SCP

```bash
# 从本地传输整个项目目录
scp -r D:\sub2api\sub2api-jrw user@your-server:/path/to/sub2api
```

#### 方案 C: 使用 rsync

```bash
# 增量同步 (更快)
rsync -avz --exclude 'node_modules' --exclude 'data*' --exclude 'postgres_data*' \
  D:\sub2api\sub2api-jrw/ user@your-server:/path/to/sub2api/
```

### 3. 在服务器上配置环境变量

```bash
# SSH 登录服务器
ssh user@your-server
cd /path/to/sub2api

# 复制环境变量模板
cp .env.pi.example .env.pi

# 编辑配置 (重要: 必须配置正确的数据库密码和 JWT 密钥)
nano .env.pi  # 或使用 vim/vi
```

**关键配置项:**

```bash
# PostgreSQL 配置
POSTGRES_USER=sub2api_pi
POSTGRES_PASSWORD=<生成安全密码>  # openssl rand -hex 32
POSTGRES_DB=sub2api_pi

# Redis 配置
REDIS_PASSWORD=<生成安全密码>  # openssl rand -hex 32

# 应用配置
SERVER_PORT=8081
JWT_SECRET=<生成 JWT 密钥>  # openssl rand -hex 32
TOTP_SECRET_KEY=<生成 TOTP 密钥>  # openssl rand -hex 32

# 管理员账号
ADMIN_USERNAME=admin
ADMIN_PASSWORD=<设置管理员密码>
```

生成安全密钥的命令:

```bash
openssl rand -hex 32  # 生成 64 字符的十六进制密钥
```

### 4. 执行部署脚本

```bash
# 赋予执行权限
chmod +x deploy-prompt-injection.sh

# 执行部署
./deploy-prompt-injection.sh
```

脚本将自动完成:
1. 创建数据目录 (`data_pi`, `postgres_data_pi`, `redis_data_pi`)
2. 检查/创建 Docker 网络 (`sub2api-network`)
3. 构建 Docker 镜像 (`sub2api:prompt-injection`)
4. 启动服务 (PostgreSQL、Redis、Sub2API)

### 5. 验证部署

#### 检查服务状态

```bash
# 查看容器状态
docker-compose -f docker-compose.prompt-injection.yml ps

# 预期输出:
# NAME              STATUS    PORTS
# sub2api-pi        running   0.0.0.0:8081->8080/tcp
# postgres-pi       running   5432/tcp
# redis-pi          running   6379/tcp
```

#### 检查服务健康

```bash
# 测试环境健康检查
curl http://localhost:8081/health

# 生产环境健康检查 (应该仍在运行)
curl http://localhost:8080/health
```

#### 查看日志

```bash
# 实时查看所有服务日志
docker-compose -f docker-compose.prompt-injection.yml logs -f

# 仅查看 API 服务日志
docker-compose -f docker-compose.prompt-injection.yml logs -f sub2api-pi

# 查看最近 100 行日志
docker-compose -f docker-compose.prompt-injection.yml logs --tail=100
```

### 6. 功能测试

#### 登录并创建 API Key

```bash
# 访问测试环境前端
http://your-server:8081

# 使用 .env.pi 中配置的管理员账号登录
# 在 API Keys 页面创建一个新的 API Key
# 启用 "Prompt Injection" 选项
```

#### 测试提示词注入

```bash
# 使用生成的 API Key 发送请求
curl -X POST http://your-server:8081/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-xxxxx" \
  -d '{
    "model": "gpt-4",
    "messages": [
      {
        "role": "user",
        "content": "Write a Python function to calculate fibonacci numbers"
      }
    ]
  }'
```

**预期行为:**
- 如果 API Key 启用了 Prompt Injection，系统会自动注入优化提示词
- 注入内容包括: 语言特性(Python)、框架建议、任务类型(code_implementation)、上下文模块
- 响应中应包含符合最佳实践的代码实现

#### 验证注入矩阵覆盖

测试不同组合以验证完整功能:

**语言 (12种):**
- Python, JavaScript, Java, C++, C#, Go, Rust, TypeScript, Ruby, PHP, Swift, Kotlin

**框架 (11种):**
- React, Vue, Angular, Django, Flask, Spring, Express, FastAPI, Next.js, Rails, Laravel

**任务类型 (8种):**
- code_implementation, debugging, optimization, refactoring, testing, documentation, architecture, code_review

**上下文模块 (2种):**
- best_practices, common_pitfalls

### 7. 管理命令

#### 重启服务

```bash
# 重启所有服务
docker-compose -f docker-compose.prompt-injection.yml restart

# 重启特定服务
docker-compose -f docker-compose.prompt-injection.yml restart sub2api-pi
```

#### 停止服务

```bash
# 停止服务 (保留数据)
docker-compose -f docker-compose.prompt-injection.yml down

# 停止服务并删除数据卷
docker-compose -f docker-compose.prompt-injection.yml down -v
```

#### 更新代码

```bash
# 1. 拉取最新代码
git pull origin main

# 2. 重新构建镜像
docker build -t sub2api:prompt-injection .

# 3. 重启服务
docker-compose -f docker-compose.prompt-injection.yml up -d
```

#### 查看资源使用

```bash
# 查看容器资源使用情况
docker stats sub2api-pi postgres-pi redis-pi
```

## 数据目录结构

```
/path/to/sub2api/
├── data_pi/              # Sub2API 应用数据 (测试)
├── postgres_data_pi/     # PostgreSQL 数据 (测试)
├── redis_data_pi/        # Redis 数据 (测试)
├── data/                 # Sub2API 应用数据 (生产)
├── postgres_data/        # PostgreSQL 数据 (生产)
└── redis_data/           # Redis 数据 (生产)
```

## 网络配置

两个环境共享 `sub2api-network` 网络，容器间可以通信:

- 生产环境:
  - `sub2api:8080` → 外部
  - `postgres:5432` → 内部
  - `redis:6379` → 内部

- 测试环境:
  - `sub2api-pi:8080` (映射到主机 8081) → 外部
  - `postgres-pi:5432` → 内部
  - `redis-pi:6379` → 内部

## 防火墙配置

如果服务器启用了防火墙，需要开放端口:

```bash
# UFW (Ubuntu)
sudo ufw allow 8081/tcp

# firewalld (CentOS/RHEL)
sudo firewall-cmd --permanent --add-port=8081/tcp
sudo firewall-cmd --reload

# iptables
sudo iptables -A INPUT -p tcp --dport 8081 -j ACCEPT
sudo iptables-save > /etc/iptables/rules.v4
```

## 故障排查

### 容器无法启动

```bash
# 查看详细日志
docker-compose -f docker-compose.prompt-injection.yml logs

# 检查端口占用
netstat -tuln | grep 8081

# 检查磁盘空间
df -h

# 检查 Docker 资源
docker system df
```

### 数据库连接失败

```bash
# 检查 PostgreSQL 容器状态
docker-compose -f docker-compose.prompt-injection.yml ps postgres-pi

# 进入 PostgreSQL 容器
docker exec -it postgres-pi psql -U sub2api_pi -d sub2api_pi

# 验证数据库密码
docker-compose -f docker-compose.prompt-injection.yml exec sub2api-pi \
  env | grep POSTGRES
```

### Redis 连接失败

```bash
# 检查 Redis 容器状态
docker-compose -f docker-compose.prompt-injection.yml ps redis-pi

# 测试 Redis 连接
docker exec -it redis-pi redis-cli -a <REDIS_PASSWORD> ping

# 应返回: PONG
```

### API 响应 502/504

```bash
# 检查 API 服务日志
docker-compose -f docker-compose.prompt-injection.yml logs sub2api-pi

# 检查健康检查
curl -v http://localhost:8081/health

# 检查容器资源
docker stats sub2api-pi
```

## 性能优化建议

### 1. 数据库优化

编辑 `docker-compose.prompt-injection.yml` 中的 PostgreSQL 配置:

```yaml
postgres-pi:
  command: >
    postgres
    -c shared_buffers=256MB
    -c effective_cache_size=1GB
    -c maintenance_work_mem=64MB
    -c checkpoint_completion_target=0.9
    -c wal_buffers=16MB
    -c default_statistics_target=100
    -c random_page_cost=1.1
    -c effective_io_concurrency=200
```

### 2. Redis 优化

```yaml
redis-pi:
  command: >
    redis-server
    --requirepass ${REDIS_PASSWORD}
    --maxmemory 256mb
    --maxmemory-policy allkeys-lru
    --save ""
```

### 3. 应用资源限制

```yaml
sub2api-pi:
  deploy:
    resources:
      limits:
        cpus: '2'
        memory: 2G
      reservations:
        cpus: '1'
        memory: 1G
```

## 安全建议

1. **密码管理**: 使用强密码，定期轮换
2. **网络隔离**: 仅暴露必要端口 (8081)
3. **日志审计**: 定期检查访问日志
4. **备份策略**: 定期备份数据目录
5. **SSL/TLS**: 生产环境使用 HTTPS (Nginx/Caddy 反向代理)
6. **访问控制**: 配置 IP 白名单或 VPN

## 备份与恢复

### 备份数据

```bash
# 备份 PostgreSQL
docker exec postgres-pi pg_dump -U sub2api_pi sub2api_pi > backup_pi_$(date +%Y%m%d).sql

# 备份数据目录
tar -czf backup_pi_data_$(date +%Y%m%d).tar.gz data_pi/ postgres_data_pi/ redis_data_pi/
```

### 恢复数据

```bash
# 恢复 PostgreSQL
docker exec -i postgres-pi psql -U sub2api_pi sub2api_pi < backup_pi_20261121.sql

# 恢复数据目录
tar -xzf backup_pi_data_20261121.tar.gz
```

## 迁移到生产

测试通过后，可以将功能迁移到生产环境:

```bash
# 1. 备份生产环境
./backup-production.sh

# 2. 停止生产服务
docker-compose down

# 3. 更新代码
git pull origin main

# 4. 重新构建生产镜像
docker build -t sub2api:latest .

# 5. 更新生产配置 (如需要)
nano docker-compose.yml

# 6. 启动生产服务
docker-compose up -d

# 7. 验证
curl http://localhost:8080/health
```

## 技术支持

如遇问题，请提供以下信息:

1. 完整日志输出
2. 容器状态 (`docker ps -a`)
3. 系统资源使用情况 (`docker stats`)
4. 错误复现步骤
5. 配置文件内容 (去除敏感信息)

---

**部署日期**: 2026-11-21  
**版本**: Prompt Injection v1.0  
**维护者**: Sub2API Team
