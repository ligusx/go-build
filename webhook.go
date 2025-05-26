package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Config struct {
	ListenAddr string   `json:"listen_addr"`
	Secret     string   `json:"secret"`
	Path       string   `json:"path"`
	Methods    []string `json:"methods"`
	LogFile    string   `json:"log_file"`
}

type SMS struct {
	Secret  string `json:"secret"`
	Time    string `json:"time"`
	From    string `json:"from"`
	Content string `json:"content"`
	Device  string `json:"device"`
}

type WebhookServer struct {
	config   *Config
	logger   *log.Logger
	mu       sync.Mutex
	messages []SMS
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
	var sms SMS
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

	// 存储短信
	s.messages = append(s.messages, sms)

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "SMS received")
}

// 获取短信数据，支持分页和过滤
func (s *WebhookServer) getMessages(limit, offset int, fromFilter, timeFilter string) []SMS {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 复制消息以避免修改原始数据
	messages := make([]SMS, len(s.messages))
	copy(messages, s.messages)

	// 按时间降序排序
	sort.Slice(messages, func(i, j int) bool {
		return messages[i].Time > messages[j].Time
	})

	var filtered []SMS

	// 应用过滤器
	for _, msg := range messages {
		match := true

		if fromFilter != "" && !strings.Contains(strings.ToLower(msg.From), strings.ToLower(fromFilter)) {
			match = false
		}

		if timeFilter != "" {
			msgTime, err := time.Parse("2006-01-02 15:04:05", msg.Time)
			if err == nil {
				filterTime, err := time.Parse("2006-01-02", timeFilter)
				if err == nil {
					if msgTime.Year() != filterTime.Year() || 
					   msgTime.Month() != filterTime.Month() || 
					   msgTime.Day() != filterTime.Day() {
						match = false
					}
				}
			}
		}

		if match {
			filtered = append(filtered, msg)
		}
	}

	// 应用分页
	start := offset
	if start > len(filtered) {
		start = len(filtered)
	}

	end := start + limit
	if end > len(filtered) {
		end = len(filtered)
	}

	return filtered[start:end]
}

// 网页界面处理函数
func (s *WebhookServer) handleWebUI(w http.ResponseWriter, r *http.Request) {
	// 解析查询参数
	limit := 10
	offset := 0
	fromFilter := ""
	timeFilter := ""

	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 {
			limit = val
		}
	}

	if o := r.URL.Query().Get("offset"); o != "" {
		if val, err := strconv.Atoi(o); err == nil && val >= 0 {
			offset = val
		}
	}

	if f := r.URL.Query().Get("from"); f != "" {
		fromFilter = f
	}

	if t := r.URL.Query().Get("time"); t != "" {
		timeFilter = t
	}

	// 获取过滤后的消息
	messages := s.getMessages(limit, offset, fromFilter, timeFilter)

	// 准备模板数据
	data := struct {
		Messages   []SMS
		Limit      int
		Offset     int
		FromFilter string
		TimeFilter string
		Total      int
	}{
		Messages:   messages,
		Limit:      limit,
		Offset:     offset,
		FromFilter: fromFilter,
		TimeFilter: timeFilter,
		Total:      len(s.messages),
	}

	// 解析并执行模板
	tmpl := template.Must(template.New("sms").Parse(webUITemplate))
	err := tmpl.Execute(w, data)
	if err != nil {
		s.logger.Printf("模板执行错误: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// API端点处理函数
func (s *WebhookServer) handleAPI(w http.ResponseWriter, r *http.Request) {
	// 解析查询参数
	limit := 10
	offset := 0
	fromFilter := ""
	timeFilter := ""

	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 {
			limit = val
		}
	}

	if o := r.URL.Query().Get("offset"); o != "" {
		if val, err := strconv.Atoi(o); err == nil && val >= 0 {
			offset = val
		}
	}

	if f := r.URL.Query().Get("from"); f != "" {
		fromFilter = f
	}

	if t := r.URL.Query().Get("time"); t != "" {
		timeFilter = t
	}

	// 获取过滤后的消息
	messages := s.getMessages(limit, offset, fromFilter, timeFilter)

	// 设置响应头并返回JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(messages); err != nil {
		s.logger.Printf("JSON编码错误: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (s *WebhookServer) Start() error {
	http.HandleFunc(s.config.Path, s.handleWebhook)
	http.HandleFunc("/sms", s.handleWebUI)
	http.HandleFunc("/api", s.handleAPI)

	s.logger.Printf("启动 SMS Webhook 服务器，监听 %s", s.config.ListenAddr)
	s.logger.Printf("Webhook 路径: %s", s.config.Path)
	s.logger.Printf("允许的方法: %v", s.config.Methods)
	s.logger.Printf("Web界面: /sms")
	s.logger.Printf("API端点: /api")
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

// Web界面HTML模板
const webUITemplate = `
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>SMS 消息中心</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            line-height: 1.6;
            margin: 0;
            padding: 20px;
            color: #333;
            max-width: 1200px;
            margin: 0 auto;
        }
        h1 {
            color: #444;
            text-align: center;
            margin-bottom: 30px;
        }
        .filter-form {
            background: #f4f4f4;
            padding: 20px;
            border-radius: 5px;
            margin-bottom: 20px;
        }
        .filter-form label {
            margin-right: 10px;
        }
        .filter-form input, .filter-form button {
            padding: 8px;
            margin-right: 10px;
            border: 1px solid #ddd;
            border-radius: 4px;
        }
        .filter-form button {
            background: #5cb85c;
            color: white;
            border: none;
            cursor: pointer;
        }
        .filter-form button:hover {
            background: #4cae4c;
        }
        .message {
            background: #fff;
            border: 1px solid #ddd;
            border-radius: 5px;
            padding: 15px;
            margin-bottom: 15px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .message-header {
            display: flex;
            justify-content: space-between;
            margin-bottom: 10px;
            font-weight: bold;
            border-bottom: 1px solid #eee;
            padding-bottom: 5px;
        }
        .message-content {
            white-space: pre-wrap;
        }
        .pagination {
            display: flex;
            justify-content: center;
            margin-top: 20px;
        }
        .pagination button {
            padding: 8px 15px;
            margin: 0 5px;
            background: #337ab7;
            color: white;
            border: none;
            border-radius: 4px;
            cursor: pointer;
        }
        .pagination button:disabled {
            background: #ccc;
            cursor: not-allowed;
        }
        .stats {
            text-align: right;
            margin-bottom: 20px;
            color: #666;
        }
    </style>
</head>
<body>
    <h1>SMS 消息中心</h1>
    
    <div class="stats">
        共 {{.Total}} 条消息 | 显示 {{len .Messages}} 条
    </div>
    
    <div class="filter-form">
        <form method="get">
            <label for="from">发送者:</label>
            <input type="text" id="from" name="from" value="{{.FromFilter}}" placeholder="过滤发送者">
            
            <label for="time">日期:</label>
            <input type="date" id="time" name="time" value="{{.TimeFilter}}">
            
            <label for="limit">每页:</label>
            <input type="number" id="limit" name="limit" min="1" value="{{.Limit}}">
            
            <button type="submit">过滤</button>
            <button type="button" onclick="window.location.href='/sms'">重置</button>
        </form>
    </div>
    
    {{range .Messages}}
    <div class="message">
        <div class="message-header">
            <span>来自: {{.From}}</span>
            <span>时间: {{.Time}}</span>
            <span>设备: {{.Device}}</span>
        </div>
        <div class="message-content">{{.Content}}</div>
    </div>
    {{else}}
    <div class="message">
        <p>没有找到匹配的消息</p>
    </div>
    {{end}}
    
    <div class="pagination">
        {{if gt .Offset 0}}
        <button onclick="window.location.href='/sms?limit={{.Limit}}&offset={{sub .Offset .Limit}}&from={{.FromFilter}}&time={{.TimeFilter}}'">上一页</button>
        {{else}}
        <button disabled>上一页</button>
        {{end}}
        
        {{if eq (len .Messages) .Limit}}
        <button onclick="window.location.href='/sms?limit={{.Limit}}&offset={{add .Offset .Limit}}&from={{.FromFilter}}&time={{.TimeFilter}}'">下一页</button>
        {{else}}
        <button disabled>下一页</button>
        {{end}}
    </div>
    
    <script>
        // 添加简单的加减函数供模板使用
        function add(a, b) {
            return a + b;
        }
        
        function sub(a, b) {
            return a - b;
        }
    </script>
</body>
</html>
`