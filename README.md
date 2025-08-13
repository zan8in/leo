# Leo

<p align="center">
    <img width="200" src="image/leo.png"/>
<p>

[![Go Version](https://img.shields.io/badge/go-1.24.1-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/zan8in/leo)](https://github.com/zan8in/leo/releases)

Leo 是一个高性能的网络登录破解工具，支持多种协议的并发扫描功能。它采用模块化插件架构，易于扩展，能够高效地进行凭据测试。

## 特性

- **高性能**: 多线程并发扫描，支持自定义并发级别
- **插件架构**: 模块化设计，易于扩展的插件系统
- **多协议支持**: 支持10+种网络服务
- **智能超时**: 自动超时计算，支持手动覆盖选项
- **进度跟踪**: 实时扫描进度显示
- **重试机制**: 可配置的失败连接重试次数
- **灵活输入**: 支持单个目标、目标文件
- **速率限制**: 内置速率限制，避免目标过载

## 支持的协议

| 协议 | 默认端口 | 状态 |
|----------|--------------|--------|
| SSH | 22 | ✅ |
| MySQL | 3306 | ✅ |
| MSSQL | 1433 | ✅ |
| FTP | 21 | ✅ |
| PostgreSQL | 5432 | ✅ |
| Oracle | 1521 | ✅ |
| Redis | 6379 | ✅ |
| MongoDB | 27017 | ✅ |
| RDP | 3389 | ✅ |
| 达梦数据库 | 5236 | ✅ |
| Telnet | 23 | ✅ |
| VNC | 5900 | ✅ |

## 安装

### 从源码编译
```bash
git clone https://github.com/zan8in/leo.git
cd leo
go build -o leo cmd/leo/main.go
```

### 从发布版本下载
从 [Releases](https://github.com/zan8in/leo/releases) 下载最新的二进制文件

## 使用方法

### 基本用法

```bash
# 默认端口扫描
leo -t [主机] -s [服务]

# 指定端口扫描
leo -t [主机]:[端口] -s [服务]

# 指定用户名和密码
leo -t [主机]:[端口] -s [服务] -u admin,root -p 123456,111111

# 使用字典文件
leo -t [主机]:[端口] -s [服务] -ul users.txt -pl passes.txt

# 高并发扫描
leo -t [主机]:[端口] -s [服务] -c 100

# 自定义超时和重试
leo -t [主机]:[端口] -s [服务] -timeout 1.5s -retries 2

# 全扫描模式（不在第一次成功后停止）
leo -t [主机]:[端口] -s [服务] -fs

# 批量扫描
leo -T hosts.txt -s [服务]
```

### 命令行选项

| 选项 | 描述 | 默认值 |
|------|------|--------|
| `-t` | 目标主机 | - |
| `-T` | 目标文件（每行一个目标） | - |
| `-s` | 服务类型（mysql, dameng, mssql, ftp, redis, oracle, postgresql, mongodb, ssh, rdp） | mysql |
| `-u` | 用户名（逗号分隔） | - |
| `-ul` | 用户名字典文件（每行一个用户名） | - |
| `-p` | 密码（逗号分隔） | - |
| `-pl` | 密码字典文件（每行一个密码） | - |
| `-c` | 并发级别 | 25 |
| `-timeout` | 连接超时时间 | 1500ms |
| `-retries` | 重试次数 | 2 |
| `-verbose` | 启用详细输出 | false |
| `-fs` | 全扫描模式 | false |
| `-target-timeout` | 单个目标的最大扫描时间（0表示自动计算） | 0 |
| `-global-timeout` | 全局扫描超时时间（0表示自动计算） | 0 |
| `-progress` | 显示扫描进度 | true |

## 使用示例

### 数据库服务扫描
```bash
# MySQL扫描
leo -t 192.168.1.100 -s mysql -u root -p root,123456,password

# MSSQL扫描
leo -t 192.168.1.100:1433 -s mssql -u sa -p sa,admin,123456

# PostgreSQL扫描
leo -t 192.168.1.100 -s postgresql -u postgres -p postgres,123456

# Oracle扫描
leo -t 192.168.1.100:1521 -s oracle -u system -p oracle,123456

# 达梦数据库扫描
leo -t 192.168.1.100:5236 -s dameng -u SYSDBA -p SYSDBA,123456
```

### NoSQL数据库扫描
```bash
# MongoDB扫描
leo -t 192.168.1.100:27017 -s mongodb -u admin -p admin,123456

# Redis扫描
leo -t 192.168.1.100:6379 -s redis -p 123456
```

### 网络服务扫描
```bash
# SSH扫描
leo -t 192.168.1.100:22 -s ssh -u root,admin -p 123456,password

# FTP扫描
leo -t 192.168.1.100:21 -s ftp -u ftp,admin -p ftp,123456

# RDP扫描
leo -t 192.168.1.100:3389 -s rdp -u administrator -p admin,123456
```

### 批量扫描
```bash
# 批量目标扫描
leo -T targets.txt -s ssh -c 50 -verbose

# 使用字典文件进行批量扫描
leo -T targets.txt -s mysql -ul users.txt -pl passwords.txt -c 100
```

### 高级选项
```bash
# 全扫描模式（找到弱口令后继续扫描）
leo -t 192.168.1.100 -s ssh -fs -verbose

# 自定义超时设置
leo -t 192.168.1.100 -s mysql -timeout 3s -retries 5 -target-timeout 60s

# 关闭进度显示的静默扫描
leo -T targets.txt -s ssh -progress=false

# 详细输出模式
leo -t 192.168.1.100 -s mysql -verbose
```

## 🏗️ 架构

### 插件系统
Leo 使用模块化插件架构，每个协议都作为独立的插件实现：