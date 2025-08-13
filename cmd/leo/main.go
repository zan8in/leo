package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/zan8in/leo/internal/core"
	// 导入插件包以触发init函数
	_ "github.com/zan8in/leo/plugins"
)

func main() {
	var (
		target        = flag.String("t", "", "Target host")
		targetFile    = flag.String("T", "", "Target file (one target per line)")
		service       = flag.String("s", "mysql", "Service type (mysql, dameng, mssql, ftp, redis, oracle, postgresql, mongodb)")
		users         = flag.String("u", "", "Usernames (comma separated)")
		userList      = flag.String("ul", "", "Username dictionary file (one username per line)")
		passes        = flag.String("p", "", "Passwords (comma separated)")
		passList      = flag.String("pl", "", "Password dictionary file (one password per line)")
		concurrency   = flag.Int("c", 25, "Concurrency level")
		timeout       = flag.Duration("timeout", 1500*time.Millisecond, "Connection timeout")
		retries       = flag.Int("retries", 2, "Number of retry attempts")
		verbose       = flag.Bool("verbose", false, "Enable verbose output")
		fullScan      = flag.Bool("fs", false, "Full scan mode")
		targetTimeout = flag.Duration("target-timeout", 0, "单个目标的最大扫描时间（0表示自动计算）")
		globalTimeout = flag.Duration("global-timeout", 0, "全局扫描超时时间（0表示自动计算）")
		showProgress  = flag.Bool("progress", true, "显示扫描进度")
	)
	flag.Parse()

	// 如果不是 verbose 模式，禁用所有日志输出
	if !*verbose {
		log.SetOutput(io.Discard)
	}

	// 验证参数
	if *target == "" && *targetFile == "" {
		fmt.Println("Error: Must specify either -t or -T")
		os.Exit(1)
	}

	// 检查插件是否存在
	pluginFunc, exists := core.GlobalRegistry.Get(*service)
	if !exists {
		fmt.Printf("Error: Service '%s' not supported\n", *service)
		fmt.Printf("Available services: %s\n", strings.Join(core.GlobalRegistry.List(), ", "))
		os.Exit(1)
	}

	// 获取目标列表
	targets := getTargets(*target, *targetFile)
	if len(targets) == 0 {
		fmt.Println("Error: No valid targets found")
		os.Exit(1)
	}

	// 获取用户名和密码列表
	usernames := getUsernames(*users, *userList, *service)
	passwords := getPasswords(*passes, *passList, *service)

	// 优先级排序
	usernames, passwords = prioritizeCredentials(usernames, passwords, *service)

	if *verbose {
		fmt.Printf("[*] Starting %s scan\n", *service)
		fmt.Printf("[*] Targets: %d\n", len(targets))
		fmt.Printf("[*] Usernames: %d\n", len(usernames))
		fmt.Printf("[*] Passwords: %d\n", len(passwords))
		fmt.Printf("[*] Concurrency: %d\n", *concurrency)
	}

	// 计算超时时间
	calculatedTargetTimeout := *targetTimeout
	if calculatedTargetTimeout == 0 {
		calculatedTargetTimeout = calculateTargetTimeout(usernames, passwords, *service)
	}

	calculatedGlobalTimeout := *globalTimeout
	if calculatedGlobalTimeout == 0 {
		calculatedGlobalTimeout = calculateGlobalTimeout(len(targets), len(usernames), len(passwords), *concurrency)
	}

	if *verbose {
		fmt.Printf("[*] Target timeout: %v\n", calculatedTargetTimeout)
		fmt.Printf("[*] Global timeout: %v\n", calculatedGlobalTimeout)
	}

	// 执行扫描
	runScan(targets, usernames, passwords, *service, pluginFunc, *concurrency, *timeout, *retries, *fullScan, *verbose, calculatedTargetTimeout, calculatedGlobalTimeout, *showProgress)

	if *verbose {
		fmt.Println("[*] Scan completed")
	}
}

// calculateTargetTimeout 动态计算单个目标的超时时间
func calculateTargetTimeout(usernames, passwords []string, service string) time.Duration {
	// 基础计算：每次尝试平均耗时
	avgTimePerAttempt := 2 * time.Second // 平均每次尝试2秒
	totalAttempts := len(usernames) * len(passwords)

	// 考虑并发因子（假设可以并发3个连接）
	concurrencyFactor := 3
	if totalAttempts < concurrencyFactor {
		concurrencyFactor = totalAttempts
	}

	estimatedTime := time.Duration(totalAttempts/concurrencyFactor) * avgTimePerAttempt

	// 设置合理的边界
	minTimeout := 1 * time.Minute
	maxTimeout := 10 * time.Minute

	if estimatedTime < minTimeout {
		return minTimeout
	}
	if estimatedTime > maxTimeout {
		return maxTimeout
	}

	return estimatedTime
}

// calculateGlobalTimeout 动态计算全局超时时间
func calculateGlobalTimeout(targetCount, usernameCount, passwordCount, concurrency int) time.Duration {
	// 基础计算
	totalCombinations := targetCount * usernameCount * passwordCount
	avgTimePerCombination := 2 * time.Second

	// 考虑并发
	estimatedTime := time.Duration(totalCombinations/concurrency) * avgTimePerCombination

	// 添加缓冲时间（20%）
	estimatedTime = time.Duration(float64(estimatedTime) * 1.2)

	// 设置边界
	minTimeout := 5 * time.Minute
	maxTimeout := 2 * time.Hour

	if estimatedTime < minTimeout {
		return minTimeout
	}
	if estimatedTime > maxTimeout {
		return maxTimeout
	}

	return estimatedTime
}

// prioritizeCredentials 对凭据进行优先级排序（使用您原有的密码逻辑）
func prioritizeCredentials(usernames, passwords []string, service string) ([]string, []string) {
	// 获取服务特定的优先级顺序（从默认列表中获取）
	defaultUsernames := getDefaultUsernames(service)
	defaultPasswords := getDefaultPasswords(service)

	// 用户名优先级排序
	prioritizedUsernames := []string{}
	for _, username := range defaultUsernames {
		if contains(usernames, username) {
			prioritizedUsernames = append(prioritizedUsernames, username)
		}
	}
	// 添加其他用户名
	for _, username := range usernames {
		if !contains(prioritizedUsernames, username) {
			prioritizedUsernames = append(prioritizedUsernames, username)
		}
	}

	// 密码优先级排序
	prioritizedPasswords := []string{}
	for _, password := range defaultPasswords {
		if contains(passwords, password) {
			prioritizedPasswords = append(prioritizedPasswords, password)
		}
	}
	// 添加其他密码
	for _, password := range passwords {
		if !contains(prioritizedPasswords, password) {
			prioritizedPasswords = append(prioritizedPasswords, password)
		}
	}

	return prioritizedUsernames, prioritizedPasswords
}

// contains 检查切片是否包含指定元素
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// runScan 执行扫描（改进版本）
func runScan(targets, usernames, passwords []string, service string, pluginFunc core.PluginFunc, concurrency int, timeout time.Duration, retries int, fullScan, verbose bool, targetTimeout, globalTimeout time.Duration, showProgress bool) {
	// 创建全局上下文
	globalCtx, globalCancel := context.WithTimeout(context.Background(), globalTimeout)
	defer globalCancel()

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	foundTargets := make(map[string]bool)
	var mu sync.Mutex

	// 进度统计
	var (
		completedTargets int64
		totalTargets     = int64(len(targets))
		progressMu       sync.Mutex
	)

	// 启动进度显示协程
	if showProgress {
		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-globalCtx.Done():
					return
				case <-ticker.C:
					progressMu.Lock()
					completed := completedTargets
					progressMu.Unlock()

					progress := float64(completed) / float64(totalTargets) * 100
					fmt.Printf("[*] Progress: %.1f%% (%d/%d targets completed)\n", progress, completed, totalTargets)
				}
			}
		}()
	}

	for _, target := range targets {
		host, port := parseTarget(target, service)

		// 检查全局上下文是否已取消
		select {
		case <-globalCtx.Done():
			if verbose {
				fmt.Printf("[!] Global timeout reached, stopping scan\n")
			}
			return
		default:
		}

		// 检查是否已找到该目标的弱口令（非全扫描模式）
		if !fullScan {
			mu.Lock()
			if foundTargets[fmt.Sprintf("%s:%d", host, port)] {
				mu.Unlock()
				continue
			}
			mu.Unlock()
		}

		wg.Add(1)
		go func(h string, p int) {
			defer wg.Done()
			defer func() {
				progressMu.Lock()
				completedTargets++
				progressMu.Unlock()
			}()

			sem <- struct{}{}
			defer func() { <-sem }()

			// 为每个目标创建独立的超时上下文
			targetCtx, targetCancel := context.WithTimeout(globalCtx, targetTimeout)
			defer targetCancel()

			// 优先检测未授权访问
			info := &core.HostInfo{
				Host:     h,
				Port:     p,
				Timeout:  timeout,
				Retries:  retries,
				Service:  service,
				Username: "",
				Password: "",
				Context:  targetCtx, // 传递目标级上下文
			}

			if err := pluginFunc(info); err == nil {
				// 发现未授权访问，标记该目标已找到
				if !fullScan {
					mu.Lock()
					foundTargets[fmt.Sprintf("%s:%d", h, p)] = true
					mu.Unlock()
				}
				return
			}

			// 未授权访问失败，进行弱口令检测
			for _, username := range usernames {
				// 检查目标上下文是否已取消
				select {
				case <-targetCtx.Done():
					if verbose {
						fmt.Printf("[!] Target %s:%d timeout reached\n", h, p)
					}
					return
				default:
				}

				// 检查是否已找到该目标的弱口令
				if !fullScan {
					mu.Lock()
					if foundTargets[fmt.Sprintf("%s:%d", h, p)] {
						mu.Unlock()
						break
					}
					mu.Unlock()
				}

				for _, password := range passwords {
					// 再次检查上下文
					select {
					case <-targetCtx.Done():
						return
					default:
					}

					info.Username = username
					info.Password = password

					if err := pluginFunc(info); err == nil {
						// 找到弱口令，标记该目标
						if !fullScan {
							mu.Lock()
							foundTargets[fmt.Sprintf("%s:%d", h, p)] = true
							mu.Unlock()
							break
						}
					}
				}
			}
		}(host, port)
	}

	// 等待所有goroutine完成
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		if verbose {
			fmt.Printf("[*] Scan completed successfully\n")
		}
	case <-globalCtx.Done():
		if verbose {
			fmt.Printf("[!] Scan terminated due to global timeout\n")
		}
	case <-time.After(globalTimeout + 30*time.Second): // 额外30秒缓冲
		if verbose {
			fmt.Printf("[!] Force terminating scan - some goroutines may be stuck\n")
		}
	}
}

// parseTarget 解析目标地址，返回主机和端口
func parseTarget(target, service string) (string, int) {
	parts := strings.Split(target, ":")
	if len(parts) == 2 {
		if port, err := strconv.Atoi(parts[1]); err == nil {
			return parts[0], port
		}
	}
	return target, getDefaultPort(service)
}

func getDefaultPort(service string) int {
	ports := map[string]int{
		"mysql":      3306,
		"dameng":     5236,
		"mssql":      1433,
		"ftp":        21,
		"redis":      6379,
		"oracle":     1521,
		"postgresql": 5432,
		"mongodb":    27017,
		"ssh":        22,
		"rdp":        3389,
	}
	if port, exists := ports[service]; exists {
		return port
	}
	return 80
}

func getTargets(target, targetFile string) []string {
	var targets []string

	if target != "" {
		targets = append(targets, target)
	}

	if targetFile != "" {
		file, err := os.Open(targetFile)
		if err != nil {
			fmt.Printf("Error opening target file: %v\n", err)
			return targets
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" && !strings.HasPrefix(line, "#") {
				targets = append(targets, line)
			}
		}
	}

	return targets
}

func getUsernames(users, userList, service string) []string {
	var usernames []string

	if users != "" {
		usernames = strings.Split(users, ",")
	}

	if userList != "" {
		file, err := os.Open(userList)
		if err != nil {
			fmt.Printf("Error opening username file: %v\n", err)
		} else {
			defer file.Close()
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line != "" && !strings.HasPrefix(line, "#") {
					usernames = append(usernames, line)
				}
			}
		}
	}

	if len(usernames) == 0 {
		usernames = getDefaultUsernames(service)
	}

	return usernames
}

func getPasswords(passes, passList, service string) []string {
	var passwords []string

	if passes != "" {
		passwords = strings.Split(passes, ",")
	}

	if passList != "" {
		file, err := os.Open(passList)
		if err != nil {
			fmt.Printf("Error opening password file: %v\n", err)
		} else {
			defer file.Close()
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line != "" && !strings.HasPrefix(line, "#") {
					passwords = append(passwords, line)
				}
			}
		}
	}

	if len(passwords) == 0 {
		passwords = getDefaultPasswords(service)
	}

	return passwords
}

func getDefaultUsernames(service string) []string {
	usernames := map[string][]string{
		"ftp":        {"anonymous", "ftp", "admin", "root", "user"},
		"mysql":      {"root", "admin", "mysql", "user", "test"},
		"ssh":        {"root", "admin", "ubuntu", "centos", "user"},
		"postgresql": {"postgres", "admin", "root", "user"},
		"mongodb":    {"admin", "root", "mongodb", "user"},
		"redis":      {"admin", "root", "redis", "user"},
		"oracle":     {"sys", "system", "oracle", "admin", "root"},
		"mssql":      {"sa", "admin", "administrator", "root"},
		"dameng":     {"SYSDBA", "SYSAUDITOR", "SYSSSO", "SYS", "SYSDBO"},
		"rdp":        {"administrator", "admin", "guest"},
		"telnet":     {"admin", "root", "user", "administrator", "guest", "cisco", "manager", "operator", "support", "test"},
	}
	if users, exists := usernames[service]; exists {
		return users
	}
	return []string{"admin", "root", "user"}
}

// 保持您的原始 getDefaultPasswords 函数逻辑
func getDefaultPasswords(service string) []string {
	switch service {
	case "mysql":
		return []string{"", "root", "123456", "password", "admin", "mysql"}
	case "dameng":
		return []string{"", "SYSDBA", "SYSDBA001", "123456", "SYSAUDITOR", "SYSSSO", "SYS", "SYSDBO"}
	case "mssql":
		return []string{"", "sa", "123456", "password", "admin"}
	case "oracle":
		return []string{"", "oracle", "123456", "password", "admin", "manager"}
	case "postgresql":
		return []string{"", "postgres", "123456", "password", "admin"}
	case "redis":
		return []string{"", "123456", "password", "redis"}
	case "mongodb":
		return []string{"", "123456", "password", "admin", "mongo"}
	case "ftp":
		return []string{"", "ftp", "123456", "password", "admin"}
	case "ssh":
		return []string{"", "123456", "password", "admin", "root", "123123", "111111", "000000", "888888", "666666", "ubuntu", "centos", "raspberry", "toor", "pass", "qwerty", "abc123"}
	case "rdp":
		return []string{"", "123456", "password", "admin", "administrator", "123123", "111111", "000000", "888888", "666666", "P@ssw0rd", "Password123", "admin123", "root123", "guest"}
	case "telnet":
		return []string{"", "123456", "password", "admin", "root", "123123", "111111", "000000", "888888", "666666", "cisco", "manager", "public", "private", "enable", "secret", "guest", "test", "support", "operator"}
	case "vnc":
		return []string{"", "123456", "password", "admin", "vnc", "123123", "111111", "000000", "888888", "666666", "secret", "pass", "qwerty", "abc123", "root123", "admin123"}
	default:
		return []string{"", "123456", "password", "admin", "root"}
	}
}
