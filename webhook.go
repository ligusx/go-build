package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
)

type Config struct {
	ListenAddr string   `json:"listen_addr"`
	Secret     string   `json:"secret"`
	Path       string   `json:"path"`
	Methods    []string `json:"methods"`
	LogFile    string   `json:"log_file"`
}

type WebhookServer struct {
	config *Config
	logger *log.Logger
	mu     sync.Mutex
}

func loadConfig(configPath string) (*Config, error) {
	file, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("无法读取配置文件: %v", err)
	}

	var config Config
	err = json.Unmarshal(file, &config)
	if err != nil {
		return nil, fmt.Errorf("无法解析配置文件: %v", err)
	}

	if config.ListenAddr == "" {
		config.ListenAddr = ":8080"
	}
	if config.Path == "" {
		config.Path = "/sms-webhook"
	}
	if len(config.Methods) == 0 {
		config.Methods = []string{"POST"}
	}

	return &config, nil
}

func NewWebhookServer(config *Config) (*WebhookServer, error) {
	server := &WebhookServer{
		config: config,
	}

	var logOutput = os.Stdout
	if config.LogFile != "" {
		file, err := os.OpenFile(config.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("无法打开日志文件: %v", err)
		}
		logOutput = file
	}

	server.logger = log.New(logOutput, "SMS-WEBHOOK: ", log.Ldate|log.Ltime|log.Lshortfile)
	return server, nil
}

func (s *WebhookServer) handleWebhook(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.Printf("收到请求: %s %s", r.Method, r.URL.Path)

	// 检查方法
	methodAllowed := false
	for _, m := range s.config.Methods {
		if r.Method == m {
			methodAllowed = true
			break
		}
	}
	if !methodAllowed {
		s.logger.Printf("不允许的方法: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 读取请求体
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		s.logger.Printf("读取请求体错误: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// 解析 SmsForwarder 数据
	var sms struct {
		Secret  string `json:"secret"`
		Time    string `json:"time"`
		From    string `json:"from"`
		Content string `json:"content"`
		Device  string `json:"device"`
	}

	if err := json.Unmarshal(body, &sms); err != nil {
		s.logger.Printf("解析 SMS 数据错误: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// 验证密钥
	if s.config.Secret != "" && sms.Secret != s.config.Secret {
		s.logger.Println("无效的密钥")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 处理短信
	s.logger.Printf("收到短信 - 来自: %s, 时间: %s, 内容: %s, 设备: %s",
		sms.From, sms.Time, sms.Content, sms.Device)

	// 在这里添加你的业务逻辑

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "SMS received")
}

func (s *WebhookServer) Start() error {
	http.HandleFunc(s.config.Path, s.handleWebhook)

	s.logger.Printf("启动 SMS Webhook 服务器，监听 %s", s.config.ListenAddr)
	s.logger.Printf("Webhook 路径: %s", s.config.Path)
	s.logger.Printf("允许的方法: %v", s.config.Methods)
	if s.config.Secret != "" {
		s.logger.Println("已启用密钥验证")
	}

	return http.ListenAndServe(s.config.ListenAddr, nil)
}

func main() {
	configPath := flag.String("c", "config.json", "配置文件路径")
	flag.Parse()

	config, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	server, err := NewWebhookServer(config)
	if err != nil {
		log.Fatalf("创建服务器失败: %v", err)
	}

	if err := server.Start(); err != nil {
		log.Fatalf("服务器错误: %v", err)
	}
}
