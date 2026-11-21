# 动态提示词注入系统 - 监控和维护指南

## 概览

本指南提供完整的监控、维护、备份和故障排查程序，确保提示词注入系统的稳定性和可靠性。

---

## 目录

1. [监控](#监控)
2. [维护](#维护)
3. [备份和恢复](#备份和恢复)
4. [性能调优](#性能调优)
5. [日志管理](#日志管理)
6. [故障排查](#故障排查)
7. [紧急应对](#紧急应对)

---

## 监控

### 1. 服务健康检查

#### 检查所有服务状态

```bash
# 查看 Docker 容器状态
docker-compose -f docker-compose.prompt-injection.yml ps

# 预期输出
NAME                    STATUS              PORTS
sub2api-pi              Up (healthy)        0.0.0.0:8081->8080/tcp
postgres-pi             Up (healthy)        5432/tcp
redis-pi                Up (healthy)        6379/tcp
```

#### API 健康检查端点

```bash
# 主服务健康检查
curl -s http://localhost:8081/health | jq '.'

# 响应示例
{
  "status": "ok",
  "version": "0.2.25",
  "uptime_seconds": 3600,
  "database": "connected",
  "redis": "connected",
  "timestamp": "2026-11-21T10:30:45Z"
}
```

#### 数据库连接检查

```bash
# 连接到 PostgreSQL 容器
docker exec -it sub2api-postgres-pi psql -U sub2api_pi -d sub2api_pi -c "SELECT version();"

# 预期输出：PostgreSQL version information
```

#### Redis 连接检查

```bash
# 连接到 Redis 容器
docker exec -it sub2api-redis-pi redis-cli -n 1 PING

# 预期输出：PONG
```

### 2. 性能指标监控

#### 部署监控脚本

**文件：** `scripts/monitor-prompt-injection.sh`

```bash
#!/bin/bash

# 监控间隔（秒）
INTERVAL=60

# 日志文件
LOG_FILE="logs/monitor-$(date +%Y-%m-%d).log"

while true; do
    TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')
    
    # 获取容器统计信息
    STATS=$(docker stats sub2api-pi --no-stream --format "table {{.CPUPerc}}\t{{.MemUsage}}")
    
    # 获取 API 响应时间
    RESPONSE_TIME=$(curl -s -w '%{time_total}' -o /dev/null http://localhost:8081/health)
    
    # 获取数据库连接数
    DB_CONNECTIONS=$(docker exec sub2api-postgres-pi psql -U sub2api_pi -d sub2api_pi -t -c "SELECT count(*) FROM pg_stat_activity;")
    
    # 记录日志
    echo "$TIMESTAMP | CPU: $(echo $STATS | awk '{print $1}') | MEM: $(echo $STATS | awk '{print $2}') | Response Time: ${RESPONSE_TIME}s | DB Connections: $DB_CONNECTIONS" >> "$LOG_FILE"
    
    # 输出到控制台
    echo "$TIMESTAMP | API Response: ${RESPONSE_TIME}s | DB Connections: $DB_CONNECTIONS"
    
    sleep $INTERVAL
done
```

使用方法：
```bash
chmod +x scripts/monitor-prompt-injection.sh
./scripts/monitor-prompt-injection.sh &  # 后台运行

# 查看实时日志
tail -f logs/monitor-$(date +%Y-%m-%d).log
```

### 3. 日志监控

#### 实时日志查看

```bash
# 查看最后 100 行日志
docker logs -f --tail 100 sub2api-pi

# 查看特定时间范围的日志
docker logs --since 2026-11-21T10:00:00 sub2api-pi

# 查看特定模块的日志
docker logs sub2api-pi | grep "prompt_injection"
```

#### 日志级别配置

在 `.env.pi` 中设置：
```bash
LOG_LEVEL=info    # debug, info, warn, error
LOG_FORMAT=json   # json, text
LOG_OUTPUT=file   # file, stdout, both
```

#### 关键日志模式

**监视这些日志行以识别问题：**

```
# 提示词注入相关
"action":"chat_completion","prompt_injection_enabled":true

# 数据库连接问题
"error":"database connection failed"

# Redis 缓存问题
"error":"redis connection error"

# 速率限制触发
"error":"rate limit exceeded"

# 配额用尽
"error":"quota exhausted"
```

### 4. 告警配置

#### 使用 Prometheus + AlertManager

**文件：** `config/prometheus.yml`

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

alerting:
  alertmanagers:
    - static_configs:
        - targets:
            - localhost:9093

rule_files:
  - 'alert-rules.yml'

scrape_configs:
  - job_name: 'sub2api-pi'
    static_configs:
      - targets: ['localhost:8081']
    metrics_path: '/metrics'
```

**告警规则：** `config/alert-rules.yml`

```yaml
groups:
  - name: sub2api-pi
    interval: 30s
    rules:
      # API 响应时间过长
      - alert: HighAPIResponseTime
        expr: http_request_duration_seconds > 5
        for: 5m
        annotations:
          summary: "高 API 响应时间"
          description: "API 响应时间超过 5 秒，持续 5 分钟"

      # 容器 CPU 使用率过高
      - alert: HighCPUUsage
        expr: container_cpu_usage_seconds_total > 0.8
        for: 10m
        annotations:
          summary: "CPU 使用率过高"
          description: "CPU 使用率超过 80%，持续 10 分钟"

      # 容器内存使用过高
      - alert: HighMemoryUsage
        expr: container_memory_usage_bytes / container_spec_memory_limit_bytes > 0.85
        for: 10m
        annotations:
          summary: "内存使用率过高"
          description: "内存使用率超过 85%，持续 10 分钟"

      # 数据库连接数过多
      - alert: HighDatabaseConnections
        expr: pg_stat_activity_count > 50
        for: 5m
        annotations:
          summary: "数据库连接过多"
          description: "活动数据库连接超过 50"

      # 频繁错误
      - alert: HighErrorRate
        expr: rate(http_requests_total{status=~"5.."}[5m]) > 0.05
        for: 5m
        annotations:
          summary: "错误率过高"
          description: "错误率超过 5%"
```

---

## 维护

### 1. 定期备份

#### 自动备份脚本

**文件：** `scripts/backup-prompt-injection.sh`

```bash
#!/bin/bash

BACKUP_DIR="backups"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
POSTGRES_BACKUP="$BACKUP_DIR/postgres_backup_$TIMESTAMP.sql.gz"
REDIS_BACKUP="$BACKUP_DIR/redis_backup_$TIMESTAMP.rdb"
CONFIG_BACKUP="$BACKUP_DIR/config_backup_$TIMESTAMP.tar.gz"

# 创建备份目录
mkdir -p $BACKUP_DIR

# 备份 PostgreSQL
echo "备份 PostgreSQL 数据库..."
docker exec sub2api-postgres-pi pg_dump -U sub2api_pi sub2api_pi | gzip > "$POSTGRES_BACKUP"
echo "✓ PostgreSQL 备份完成: $POSTGRES_BACKUP"

# 备份 Redis
echo "备份 Redis 数据..."
docker exec sub2api-redis-pi redis-cli -n 1 BGSAVE
sleep 2
docker cp sub2api-redis-pi:/data/dump.rdb "$REDIS_BACKUP"
echo "✓ Redis 备份完成: $REDIS_BACKUP"

# 备份配置文件
echo "备份配置文件..."
tar -czf "$CONFIG_BACKUP" .env.pi docker-compose.prompt-injection.yml
echo "✓ 配置备份完成: $CONFIG_BACKUP"

# 清理旧备份（保留最近 30 天）
find $BACKUP_DIR -mtime +30 -delete
echo "✓ 清理旧备份完成"

# 显示备份总大小
TOTAL_SIZE=$(du -sh $BACKUP_DIR | cut -f1)
echo "📦 备份目录总大小: $TOTAL_SIZE"
```

#### 备份计划

```bash
# 在 crontab 中添加定时备份
crontab -e

# 每天凌晨 2 点执行备份
0 2 * * * /path/to/scripts/backup-prompt-injection.sh >> /var/log/sub2api-backup.log 2>&1

# 每周日凌晨 3 点执行完整备份
0 3 * * 0 /path/to/scripts/backup-prompt-injection.sh >> /var/log/sub2api-backup.log 2>&1
```

### 2. 数据库维护

#### PostgreSQL 定期维护

```bash
# 清理死行（VACUUM）
docker exec sub2api-postgres-pi psql -U sub2api_pi -d sub2api_pi -c "VACUUM ANALYZE;"

# 检查表大小
docker exec sub2api-postgres-pi psql -U sub2api_pi -d sub2api_pi -c "
  SELECT schemaname, tablename, pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
  FROM pg_tables
  ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;"

# 检查索引使用情况
docker exec sub2api-postgres-pi psql -U sub2api_pi -d sub2api_pi -c "
  SELECT schemaname, tablename, indexname
  FROM pg_indexes
  WHERE schemaname NOT IN ('pg_catalog', 'information_schema');"

# 重建索引（定期维护）
docker exec sub2api-postgres-pi psql -U sub2api_pi -d sub2api_pi -c "REINDEX DATABASE sub2api_pi;"
```

#### Redis 内存管理

```bash
# 检查 Redis 内存使用
docker exec sub2api-redis-pi redis-cli -n 1 INFO memory

# 清理过期键
docker exec sub2api-redis-pi redis-cli -n 1 EVICT_FREQUENCY

# 设置内存管理策略（在 Redis 配置中）
maxmemory 256mb           # 最大内存
maxmemory-policy allkeys-lru  # 驱逐策略
```

### 3. 日志轮转

#### 配置 logrotate

**文件：** `/etc/logrotate.d/sub2api-pi`

```
/var/log/sub2api-pi/*.log {
    daily              # 每天轮转
    rotate 30          # 保留 30 个文件
    compress           # 压缩旧日志
    delaycompress      # 延迟压缩前一个日志
    missingok          # 如果日志不存在，不报错
    notifempty         # 如果日志为空，不轮转
    create 0644 root root
    sharedscripts
    postrotate
        docker exec sub2api-pi kill -HUP 1
    endscript
}
```

### 4. 更新和升级

#### 检查可用更新

```bash
# 列出本地镜像
docker images sub2api:*

# 从 GitHub 检查新版本
git fetch origin
git log --oneline origin/main -5

# 比较本地和远程版本
git log --oneline -1
git log --oneline -1 origin/main
```

#### 升级流程

```bash
# 1. 备份当前环境
./scripts/backup-prompt-injection.sh

# 2. 停止当前服务
docker-compose -f docker-compose.prompt-injection.yml down

# 3. 拉取新代码
git pull origin main

# 4. 重建镜像
docker build -t sub2api:prompt-injection \
  -f Dockerfile \
  --target production .

# 5. 重新启动服务
docker-compose -f docker-compose.prompt-injection.yml up -d

# 6. 验证升级
curl http://localhost:8081/health
docker-compose -f docker-compose.prompt-injection.yml logs -f sub2api-pi
```

---

## 备份和恢复

### 1. 恢复 PostgreSQL

```bash
# 恢复数据库
zcat backups/postgres_backup_20261121_020000.sql.gz | \
  docker exec -i sub2api-postgres-pi psql -U sub2api_pi -d sub2api_pi

# 验证恢复
docker exec sub2api-postgres-pi psql -U sub2api_pi -d sub2api_pi -c "
  SELECT COUNT(*) as table_count FROM information_schema.tables WHERE table_schema='public';"
```

### 2. 恢复 Redis

```bash
# 复制备份文件到容器
docker cp backups/redis_backup_20261121_020000.rdb sub2api-redis-pi:/data/dump.rdb

# 重启 Redis 加载数据
docker restart sub2api-redis-pi

# 验证恢复
docker exec sub2api-redis-pi redis-cli -n 1 DBSIZE
```

### 3. 完整系统恢复

```bash
# 1. 停止所有服务
docker-compose -f docker-compose.prompt-injection.yml down

# 2. 删除旧数据
rm -rf data_pi postgres_data_pi redis_data_pi

# 3. 恢复备份文件
tar -xzf backups/config_backup_20261121_020000.tar.gz

# 4. 恢复数据库
mkdir -p postgres_data_pi redis_data_pi data_pi
zcat backups/postgres_backup_20261121_020000.sql.gz | \
  docker exec -i sub2api-postgres-pi psql -U sub2api_pi -d sub2api_pi

# 5. 重启服务
docker-compose -f docker-compose.prompt-injection.yml up -d

# 6. 验证
curl http://localhost:8081/health
```

---

## 性能调优

### 1. 容器资源限制优化

**修改 `docker-compose.prompt-injection.yml`：**

```yaml
services:
  sub2api-pi:
    deploy:
      resources:
        limits:
          cpus: '2'           # 最大 CPU
          memory: 2G          # 最大内存
        reservations:
          cpus: '1'           # 预留 CPU
          memory: 1G          # 预留内存
```

### 2. 数据库连接池优化

在环境配置中设置：
```bash
# PostgreSQL 连接池
DB_CONN_MAX_IDLE_TIME=5m
DB_CONN_MAX_LIFETIME=10m
DB_CONN_POOL_SIZE=25

# Redis 连接池
REDIS_CONN_POOL_SIZE=10
REDIS_CONN_MAX_IDLE_TIME=5m
```

### 3. 缓存优化

```bash
# 启用 Redis 缓存
CACHE_ENABLED=true
CACHE_TTL=3600           # 1 小时缓存

# 缓存预热（启动时）
CACHE_PRELOAD=true

# 监视缓存命中率
docker exec sub2api-redis-pi redis-cli -n 1 INFO stats | grep keyspace
```

### 4. 查询优化

```bash
# 启用慢查询日志
docker exec sub2api-postgres-pi psql -U sub2api_pi -d sub2api_pi -c "
  ALTER SYSTEM SET log_min_duration_statement = 1000;
  SELECT pg_reload_conf();"

# 查看慢查询
docker exec sub2api-postgres-pi psql -U sub2api_pi -d sub2api_pi -c "
  SELECT query, mean_time, calls FROM pg_stat_statements 
  ORDER BY mean_time DESC LIMIT 10;"
```

---

## 日志管理

### 1. 日志聚合

使用 ELK Stack (Elasticsearch, Logstash, Kibana) 集中管理日志

**Logstash 配置：** `config/logstash.conf`

```
input {
  file {
    path => "/var/log/sub2api-pi/*.log"
    start_position => "beginning"
  }
}

filter {
  if [message] =~ /prompt_injection/ {
    grok {
      match => { "message" => "%{TIMESTAMP_ISO8601:timestamp} %{LOGLEVEL:level} %{DATA:service} %{GREEDYDATA:msg}" }
    }
    mutate {
      add_field => { "system" => "sub2api-pi" }
    }
  }
}

output {
  elasticsearch {
    hosts => ["localhost:9200"]
    index => "sub2api-pi-%{+YYYY.MM.dd}"
  }
}
```

### 2. 日志分析查询

```bash
# 搜索特定错误
docker logs sub2api-pi | grep -i "error" | tail -20

# 统计错误频率
docker logs sub2api-pi | grep -i "error" | wc -l

# 按小时统计请求
docker logs sub2api-pi | grep "POST /v1/chat/completions" | \
  awk '{print substr($1,1,13)}' | sort | uniq -c

# 查找性能问题
docker logs sub2api-pi | grep "response_time" | sort -t: -k2 -rn | head -10
```

---

## 故障排查

### 1. 常见问题

#### 问题：API 无法访问

```bash
# 检查容器是否运行
docker ps | grep sub2api-pi

# 检查端口是否开放
netstat -tuln | grep 8081

# 检查防火墙
sudo ufw status
sudo ufw allow 8081

# 检查容器日志
docker logs sub2api-pi | tail -50
```

#### 问题：数据库连接失败

```bash
# 检查 PostgreSQL 状态
docker exec sub2api-postgres-pi pg_isready

# 验证环境变量
docker exec sub2api-pi env | grep POSTGRES

# 检查网络连接
docker network inspect sub2api-network
```

#### 问题：内存使用过高

```bash
# 检查容器内存
docker stats sub2api-pi --no-stream

# 检查 Go 进程内存
docker exec sub2api-pi top -b -n 1 | head -10

# 分析内存泄漏
curl http://localhost:8081/debug/pprof/heap > heap.prof
go tool pprof heap.prof
```

### 2. 调试命令

```bash
# 进入容器调试
docker exec -it sub2api-pi /bin/bash

# 使用 curl 测试 API
curl -v http://localhost:8081/health

# 检查网络
docker exec sub2api-pi ping postgres-pi
docker exec sub2api-pi ping redis-pi

# 监视进程
docker exec sub2api-pi ps aux | grep sub2api
```

---

## 紧急应对

### 1. 紧急停止

```bash
# 停止所有服务
docker-compose -f docker-compose.prompt-injection.yml down

# 强制停止（如果 graceful shutdown 失败）
docker-compose -f docker-compose.prompt-injection.yml down -t 10
```

### 2. 快速重启

```bash
# 重启单个服务
docker-compose -f docker-compose.prompt-injection.yml restart sub2api-pi

# 重启所有服务
docker-compose -f docker-compose.prompt-injection.yml restart
```

### 3. 回滚到上个版本

```bash
# 1. 查看镜像历史
docker images sub2api:prompt-injection

# 2. 停止当前服务
docker-compose -f docker-compose.prompt-injection.yml down

# 3. 切换回上个版本
# 编辑 docker-compose.prompt-injection.yml，修改镜像标签或回滚代码

# 4. 重新启动
docker-compose -f docker-compose.prompt-injection.yml up -d
```

### 4. 服务恢复检查清单

- [ ] 所有容器运行正常 (`docker ps`)
- [ ] API 可访问 (`curl http://localhost:8081/health`)
- [ ] 数据库连接正常
- [ ] Redis 连接正常
- [ ] 日志无错误信息
- [ ] 性能指标正常
- [ ] 生产环境（8080）未受影响

---

## 监控指标参考

| 指标 | 警告阈值 | 严重阈值 | 单位 |
|------|---------|---------|------|
| CPU 使用率 | > 60% | > 80% | % |
| 内存使用率 | > 70% | > 90% | % |
| API 响应时间 | > 2s | > 5s | 秒 |
| 错误率 | > 1% | > 5% | % |
| 数据库连接数 | > 40 | > 60 | 个 |
| 磁盘使用率 | > 70% | > 90% | % |
| Redis 内存使用 | > 200MB | > 250MB | MB |

---

## 联系和支持

**问题报告：** https://github.com/JarJiang/sub2api-jrw/issues
**文档：** https://github.com/JarJiang/sub2api-jrw/wiki

---

*最后更新：2026-11-21*
*维护版本：1.0*
