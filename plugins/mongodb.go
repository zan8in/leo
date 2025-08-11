package plugins

import (
	"context"
	"fmt"
	"time"

	"github.com/zan8in/leo/internal/core"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongodbScan MongoDB扫描函数（参考fscan的MongodbScan和MongodbUnauth）
func MongodbScan(info *core.HostInfo) error {
	if info.Port == 0 {
		info.Port = 27017
	}

	// 获取context，如果没有则创建默认的
	ctx := info.Context
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
	}

	// 检查context是否已取消
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// 优先检测未授权访问（类似fscan的MongodbUnauth）
	if info.Username == "" && info.Password == "" {
		if err := mongodbUnauth(info, ctx); err == nil {
			fmt.Printf("[+] %s:%d mongodb unauthorized access\n", info.Host, info.Port)
			return nil // 发现未授权访问，停止进一步检测
		}
	}

	// 进行认证检测
	return mongodbAuth(info, ctx)
}

// mongodbUnauth 检测未授权访问
func mongodbUnauth(info *core.HostInfo, parentCtx context.Context) error {
	timeout := info.Timeout
	if timeout == 0 {
		timeout = 3 * time.Second
	}

	// 创建带超时的context用于单个请求
	requestCtx, requestCancel := context.WithTimeout(parentCtx, timeout)
	defer requestCancel()

	// 检查context是否已取消
	select {
	case <-parentCtx.Done():
		return parentCtx.Err()
	default:
	}

	// 无认证连接URI
	uri := fmt.Sprintf("mongodb://%s:%d/?connectTimeoutMS=%d&serverSelectionTimeoutMS=%d",
		info.Host, info.Port,
		int(timeout.Milliseconds()), int(timeout.Milliseconds()))

	clientOptions := options.Client().ApplyURI(uri)
	clientOptions.SetConnectTimeout(timeout)
	clientOptions.SetServerSelectionTimeout(timeout)

	client, err := mongo.Connect(requestCtx, clientOptions)
	if err != nil {
		return err
	}
	defer client.Disconnect(requestCtx)

	// 检查context是否已取消
	select {
	case <-parentCtx.Done():
		return parentCtx.Err()
	default:
	}

	// 快速连接测试
	if err = client.Ping(requestCtx, nil); err != nil {
		return err
	}

	// 检查context是否已取消
	select {
	case <-parentCtx.Done():
		return parentCtx.Err()
	default:
	}

	// 尝试列出数据库（未授权访问的关键验证）
	_, err = client.ListDatabaseNames(requestCtx, bson.D{})
	return err
}

// mongodbAuth 认证检测
func mongodbAuth(info *core.HostInfo, parentCtx context.Context) error {
	timeout := info.Timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	// 创建带超时的context用于单个请求
	requestCtx, requestCancel := context.WithTimeout(parentCtx, timeout)
	defer requestCancel()

	// 检查context是否已取消
	select {
	case <-parentCtx.Done():
		return parentCtx.Err()
	default:
	}

	// 带认证的连接URI
	uri := fmt.Sprintf("mongodb://%s:%s@%s:%d/?connectTimeoutMS=%d&serverSelectionTimeoutMS=%d",
		info.Username, info.Password, info.Host, info.Port,
		int(timeout.Milliseconds()), int(timeout.Milliseconds()))

	clientOptions := options.Client().ApplyURI(uri)
	clientOptions.SetConnectTimeout(timeout)
	clientOptions.SetServerSelectionTimeout(timeout)

	client, err := mongo.Connect(requestCtx, clientOptions)
	if err != nil {
		return err
	}
	defer client.Disconnect(requestCtx)

	// 检查context是否已取消
	select {
	case <-parentCtx.Done():
		return parentCtx.Err()
	default:
	}

	err = client.Ping(requestCtx, nil)
	if err == nil {
		fmt.Printf("[+] %s:%d mongodb %s:%s\n", info.Host, info.Port, info.Username, info.Password)
	}

	return err
}

// 注册插件
func init() {
	core.GlobalRegistry.Register("mongodb", MongodbScan)
}
