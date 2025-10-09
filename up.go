package main

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
    "net/url" 
	"os"
	"path/filepath"
	"strings"
)

// 笔记结构体
type Note struct {
	Title string
	Body  string
}

// 全局变量
var notes = make(map[string]*Note)
var noteTitles []string

func main() {
    // 确保up目录存在
    os.Mkdir("up", 0755)
    
    // 加载笔记
    loadNotes()
    
    // 设置路由
    http.HandleFunc("/", indexHandler)
    http.HandleFunc("/upload", uploadHandler)
    http.HandleFunc("/download/", downloadHandler)
    http.HandleFunc("/preview/", previewHandler) 
    http.HandleFunc("/notes", notesHandler)
    http.HandleFunc("/note/", noteHandler)
    http.HandleFunc("/save-note", saveNoteHandler)
    http.HandleFunc("/delete-note/", deleteNoteHandler)
    http.HandleFunc("/files", filesHandler)
    http.HandleFunc("/delete-file/", deleteFileHandler)
    
    // 静态文件服务
    http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
    
    fmt.Println("服务器启动在 http://localhost:8080")
    http.ListenAndServe(":8080", nil)
}

// 主页处理器
func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	
	tmpl := `
	<!DOCTYPE html>
	<html lang="zh-CN">
	<head>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<title>文件与笔记管理器</title>
		<style>
			* { margin: 0; padding: 0; box-sizing: border-box; }
			body { font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; line-height: 1.6; color: #333; background-color: #f5f7fa; }
			.container { max-width: 1200px; margin: 0 auto; padding: 20px; }
			header { background: linear-gradient(135deg, #6a11cb 0%, #2575fc 100%); color: white; padding: 2rem 0; text-align: center; border-radius: 10px; margin-bottom: 2rem; box-shadow: 0 4px 6px rgba(0,0,0,0.1); }
			h1 { font-size: 2.5rem; margin-bottom: 0.5rem; }
			.subtitle { font-size: 1.2rem; opacity: 0.9; }
			.card-container { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 20px; margin-bottom: 2rem; }
			.card { background: white; border-radius: 10px; padding: 1.5rem; box-shadow: 0 2px 10px rgba(0,0,0,0.05); transition: transform 0.3s, box-shadow 0.3s; }
			.card:hover { transform: translateY(-5px); box-shadow: 0 5px 15px rgba(0,0,0,0.1); }
			.card h2 { color: #2575fc; margin-bottom: 1rem; border-bottom: 2px solid #f0f0f0; padding-bottom: 0.5rem; }
			.btn { display: inline-block; background: #2575fc; color: white; padding: 10px 20px; border-radius: 5px; text-decoration: none; font-weight: bold; transition: background 0.3s; margin-top: 10px; }
			.btn:hover { background: #1a5fd8; }
			.btn-secondary { background: #6c757d; }
			.btn-secondary:hover { background: #5a6268; }
			.btn-success { background: #28a745; }
			.btn-success:hover { background: #218838; }
			.btn-danger { background: #dc3545; }
			.btn-danger:hover { background: #c82333; }
			.file-list, .note-list { list-style: none; }
			.file-list li, .note-list li { padding: 10px; border-bottom: 1px solid #eee; display: flex; justify-content: space-between; align-items: center; }
			.file-list li:last-child, .note-list li:last-child { border-bottom: none; }
			.file-actions, .note-actions { display: flex; gap: 10px; }
			.form-group { margin-bottom: 1rem; }
			.form-group label { display: block; margin-bottom: 0.5rem; font-weight: bold; }
			.form-control { width: 100%; padding: 10px; border: 1px solid #ddd; border-radius: 5px; font-size: 1rem; }
			textarea.form-control { min-height: 200px; resize: vertical; }
			.alert { padding: 15px; border-radius: 5px; margin-bottom: 1rem; }
			.alert-success { background: #d4edda; color: #155724; border: 1px solid #c3e6cb; }
			.alert-error { background: #f8d7da; color: #721c24; border: 1px solid #f5c6cb; }
			footer { text-align: center; margin-top: 2rem; padding: 1rem; color: #6c757d; border-top: 1px solid #eee; }
		</style>
	</head>
	<body>
		<div class="container">
			<header>
				<h1>文件与笔记管理器</h1>
				<p class="subtitle">上传下载文件，管理您的在线笔记</p>
			</header>
			
			<div class="card-container">
				<div class="card">
					<h2>文件管理</h2>
					<p>上传文件到服务器或下载已上传的文件</p>
					<a href="/files" class="btn">管理文件</a>
				</div>
				
				<div class="card">
					<h2>在线笔记</h2>
					<p>创建、编辑和管理您的在线笔记</p>
					<a href="/notes" class="btn">管理笔记</a>
				</div>
			</div>
			
			<footer>
				<p>© 2023 文件与笔记管理器 - 使用Go语言构建</p>
			</footer>
		</div>
	</body>
	</html>
	`
	
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, tmpl)
}

// 文件上传处理器
func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		// 解析表单
		r.ParseMultipartForm(32 << 20) // 32MB
		
		file, handler, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "无法获取文件: "+err.Error(), http.StatusBadRequest)
			return
		}
		defer file.Close()
		
		// 创建目标文件
		dst, err := os.Create(filepath.Join("up", handler.Filename))
		if err != nil {
			http.Error(w, "无法创建文件: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer dst.Close()
		
		// 复制文件内容
		if _, err := io.Copy(dst, file); err != nil {
			http.Error(w, "无法保存文件: "+err.Error(), http.StatusInternalServerError)
			return
		}
		
		// 重定向到文件列表页，显示成功消息
		http.Redirect(w, r, "/files?msg=文件上传成功", http.StatusSeeOther)
		return
	}
	
	// ==================== 显示上传表单 - 带进度条美化版 ====================
	tmpl := `
	<!DOCTYPE html>
	<html lang="zh-CN">
	<head>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<title>上传文件 - 文件与笔记管理器</title>
		<style>
			* { margin: 0; padding: 0; box-sizing: border-box; }
			body { 
				font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; 
				line-height: 1.6; 
				color: #333; 
				background: linear-gradient(135deg, #f5f7fa 0%, #c3cfe2 100%);
				min-height: 100vh;
			}
			.container { 
				max-width: 800px; 
				margin: 0 auto; 
				padding: 20px; 
			}
			.header-content {
				display: flex;
				justify-content: space-between;
				align-items: center;
				background: linear-gradient(135deg, #6a11cb 0%, #2575fc 100%);
				color: white;
				padding: 1.5rem 2rem;
				border-radius: 10px;
				margin-bottom: 2rem;
				box-shadow: 0 4px 6px rgba(0,0,0,0.1);
			}
			.header-content h1 {
				font-size: 2rem;
				margin: 0;
			}
			.btn { 
				display: inline-block; 
				background: #2575fc; 
				color: white; 
				padding: 12px 24px; 
				border-radius: 8px; 
				text-decoration: none; 
				font-weight: 600; 
				transition: all 0.3s ease;
				border: none;
				cursor: pointer;
				font-size: 1rem;
				font-family: inherit;
			}
			.btn:hover { 
				background: #1a5fd8; 
				transform: translateY(-2px);
				box-shadow: 0 6px 12px rgba(0,0,0,0.15);
			}
			.btn:active {
				transform: translateY(0);
			}
			.btn:disabled {
				background: #6c757d;
				cursor: not-allowed;
				transform: none;
				box-shadow: none;
			}
			.btn-secondary { 
				background: #6c757d; 
			}
			.btn-secondary:hover { 
				background: #5a6268; 
			}
			.card {
				background: white;
				border-radius: 12px;
				padding: 2.5rem;
				box-shadow: 0 8px 25px rgba(0,0,0,0.1);
				margin-bottom: 2rem;
				transition: transform 0.3s ease;
			}
			.card:hover {
				transform: translateY(-2px);
			}
			.card h2 {
				color: #2575fc;
				margin-bottom: 1.5rem;
				padding-bottom: 0.75rem;
				border-bottom: 2px solid #f0f0f0;
				font-size: 1.5rem;
			}
			.form-group { 
				margin-bottom: 1.5rem; 
			}
			.form-group label { 
				display: block; 
				margin-bottom: 0.75rem; 
				font-weight: 600;
				color: #495057;
				font-size: 1rem;
			}
			.form-control { 
				width: 100%; 
				padding: 14px 16px; 
				border: 2px solid #e9ecef; 
				border-radius: 8px; 
				font-size: 1rem; 
				transition: all 0.3s ease;
				font-family: inherit;
				background: #fafbfc;
			}
			.form-control:focus {
				outline: none;
				border-color: #2575fc;
				box-shadow: 0 0 0 3px rgba(37, 117, 252, 0.1);
				background: white;
			}
			.file-input-wrapper {
				position: relative;
				display: inline-block;
				width: 100%;
			}
			.file-input-wrapper input[type="file"] {
				position: absolute;
				left: 0;
				top: 0;
				opacity: 0;
				width: 100%;
				height: 100%;
				cursor: pointer;
			}
			.file-input-custom {
				display: flex;
				flex-direction: column;
				align-items: center;
				justify-content: center;
				padding: 3rem 2rem;
				border: 2px dashed #dee2e6;
				border-radius: 12px;
				background: #f8f9fa;
				transition: all 0.3s ease;
				text-align: center;
				color: #6c757d;
				font-weight: 500;
				min-height: 200px;
			}
			.file-input-custom:hover {
				border-color: #2575fc;
				background: #e7f1ff;
				color: #2575fc;
				transform: scale(1.02);
			}
			.file-input-custom.has-file {
				border-color: #28a745;
				background: #f0fff4;
				color: #28a745;
				border-style: solid;
			}
			.file-input-custom.dragover {
				border-color: #2575fc;
				background: #e7f1ff;
				color: #2575fc;
				transform: scale(1.02);
			}
			.upload-icon {
				font-size: 3rem;
				margin-bottom: 1rem;
				transition: transform 0.3s ease;
			}
			.file-input-custom:hover .upload-icon {
				transform: scale(1.1);
			}
			.form-actions {
				display: flex;
				justify-content: flex-end;
				gap: 12px;
				margin-top: 2rem;
				padding-top: 1.5rem;
				border-top: 1px solid #e9ecef;
			}
			
			/* 进度条样式 */
			.progress-container {
				display: none;
				margin-top: 2rem;
				padding: 1.5rem;
				background: #f8f9fa;
				border-radius: 12px;
				border: 1px solid #e9ecef;
			}
			.progress-container.show {
				display: block;
				animation: fadeIn 0.5s ease;
			}
			.progress-header {
				display: flex;
				justify-content: space-between;
				align-items: center;
				margin-bottom: 1rem;
			}
			.progress-title {
				font-weight: 600;
				color: #495057;
				font-size: 1.1rem;
			}
			.progress-percentage {
				font-weight: 700;
				color: #2575fc;
				font-size: 1.1rem;
			}
			.progress-bar {
				width: 100%;
				height: 12px;
				background: #e9ecef;
				border-radius: 6px;
				overflow: hidden;
				position: relative;
			}
			.progress-fill {
				height: 100%;
				background: linear-gradient(90deg, #2575fc, #6a11cb);
				border-radius: 6px;
				width: 0%;
				transition: width 0.3s ease;
				position: relative;
				overflow: hidden;
			}
			.progress-fill::after {
				content: '';
				position: absolute;
				top: 0;
				left: -100%;
				width: 100%;
				height: 100%;
				background: linear-gradient(90deg, 
					transparent, 
					rgba(255,255,255,0.4), 
					transparent);
				animation: shimmer 1.5s infinite;
			}
			.progress-details {
				display: flex;
				justify-content: space-between;
				align-items: center;
				margin-top: 0.75rem;
				font-size: 0.9rem;
				color: #6c757d;
			}
			.progress-speed {
				font-weight: 500;
			}
			.progress-time {
				font-weight: 500;
			}
			
			/* 成功状态 */
			.progress-success .progress-fill {
				background: linear-gradient(90deg, #28a745, #20c997);
			}
			.progress-success .progress-percentage {
				color: #28a745;
			}
			
			/* 动画 */
			@keyframes fadeIn {
				from { opacity: 0; transform: translateY(10px); }
				to { opacity: 1; transform: translateY(0); }
			}
			@keyframes shimmer {
				0% { transform: translateX(-100%); }
				100% { transform: translateX(200%); }
			}
			@keyframes pulse {
				0% { transform: scale(1); }
				50% { transform: scale(1.05); }
				100% { transform: scale(1); }
			}
			@keyframes bounce {
				0%, 20%, 53%, 80%, 100% {
					transform: translate3d(0,0,0);
				}
				40%, 43% {
					transform: translate3d(0,-8px,0);
				}
				70% {
					transform: translate3d(0,-4px,0);
				}
				90% {
					transform: translate3d(0,-2px,0);
				}
			}
			
			/* 加载动画 */
			.uploading .upload-icon {
				animation: bounce 1s infinite;
			}
			
			/* 成功动画 */
			.success-animation {
				animation: pulse 0.6s ease;
			}
			
			/* 状态消息 */
			.status-message {
				text-align: center;
				padding: 1rem;
				margin-top: 1rem;
				border-radius: 8px;
				font-weight: 500;
				display: none;
			}
			.status-success {
				background: #d4edda;
				color: #155724;
				border: 1px solid #c3e6cb;
				display: block;
			}
			.status-error {
				background: #f8d7da;
				color: #721c24;
				border: 1px solid #f5c6cb;
				display: block;
			}
		</style>
	</head>
	<body>
		<div class="container">
			<header class="header-content">
				<h1>上传文件</h1>
				<a href="/files" class="btn btn-secondary">返回文件列表</a>
			</header>
			
			<div class="card">
				<h2>选择要上传的文件</h2>
				<form action="/upload" method="post" enctype="multipart/form-data" id="uploadForm">
					<div class="form-group">
						<label for="file">选择文件:</label>
						<div class="file-input-wrapper">
							<input type="file" name="file" id="file" class="form-control" required 
								   onchange="updateFileName(this)">
							<div class="file-input-custom" id="fileInputCustom">
								<div>
									<div class="upload-icon">📁</div>
									<div style="font-size: 1.1rem; margin-bottom: 0.5rem;">点击选择文件或拖拽文件到这里</div>
									<div style="font-size: 0.9rem; color: #868e96;">支持所有类型文件，最大32MB</div>
								</div>
							</div>
						</div>
					</div>
					
					<!-- 上传进度条 -->
					<div class="progress-container" id="progressContainer">
						<div class="progress-header">
							<div class="progress-title">上传进度</div>
							<div class="progress-percentage" id="progressPercentage">0%</div>
						</div>
						<div class="progress-bar">
							<div class="progress-fill" id="progressFill"></div>
						</div>
						<div class="progress-details">
							<div class="progress-speed" id="progressSpeed">0 KB/s</div>
							<div class="progress-time" id="progressTime">--</div>
						</div>
					</div>
					
					<!-- 状态消息 -->
					<div class="status-message" id="statusMessage"></div>
					
					<div class="form-actions">
						<button type="submit" class="btn" id="submitBtn">
							<span>开始上传</span>
						</button>
					</div>
				</form>
			</div>
		</div>

		<script>
			let uploadStartTime;
			let lastLoaded = 0;
			
			function updateFileName(input) {
				const customInput = document.getElementById('fileInputCustom');
				if (input.files.length > 0) {
					const fileName = input.files[0].name;
					const fileSize = formatFileSize(input.files[0].size);
					customInput.innerHTML = '<div><div class="upload-icon">✅</div><div style="font-size: 1.1rem; margin-bottom: 0.5rem;">已选择文件</div><div style="font-weight: 600; margin-bottom: 0.25rem;">' + fileName + '</div><div style="font-size: 0.9rem; color: #868e96;">大小: ' + fileSize + '</div></div>';
					customInput.classList.add('has-file');
				} else {
					customInput.innerHTML = '<div><div class="upload-icon">📁</div><div style="font-size: 1.1rem; margin-bottom: 0.5rem;">点击选择文件或拖拽文件到这里</div><div style="font-size: 0.9rem; color: #868e96;">支持所有类型文件，最大32MB</div></div>';
					customInput.classList.remove('has-file');
				}
			}

			// 拖拽功能
			const fileInput = document.getElementById('file');
			const customInput = document.getElementById('fileInputCustom');
			
			customInput.addEventListener('dragover', (e) => {
				e.preventDefault();
				customInput.classList.add('dragover');
			});
			
			customInput.addEventListener('dragleave', (e) => {
				e.preventDefault();
				customInput.classList.remove('dragover');
			});
			
			customInput.addEventListener('drop', (e) => {
				e.preventDefault();
				customInput.classList.remove('dragover');
				const files = e.dataTransfer.files;
				if (files.length > 0) {
					fileInput.files = files;
					updateFileName(fileInput);
				}
			});

			// 格式化文件大小
			function formatFileSize(bytes) {
				if (bytes === 0) return '0 B';
				const k = 1024;
				const sizes = ['B', 'KB', 'MB', 'GB'];
				const i = Math.floor(Math.log(bytes) / Math.log(k));
				return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
			}

			// 上传表单提交
			document.getElementById('uploadForm').addEventListener('submit', function(e) {
				e.preventDefault();
				
				const fileInput = document.getElementById('file');
				const submitBtn = document.getElementById('submitBtn');
				const progressContainer = document.getElementById('progressContainer');
				const progressFill = document.getElementById('progressFill');
				const progressPercentage = document.getElementById('progressPercentage');
				const progressSpeed = document.getElementById('progressSpeed');
				const progressTime = document.getElementById('progressTime');
				const statusMessage = document.getElementById('statusMessage');
				
				if (!fileInput.files.length) {
					showStatus('请选择要上传的文件', 'error');
					return;
				}

				// 重置状态
				statusMessage.style.display = 'none';
				progressContainer.classList.remove('progress-success');
				progressFill.style.width = '0%';
				progressPercentage.textContent = '0%';
				progressSpeed.textContent = '0 KB/s';
				progressTime.textContent = '--';
				
				// 显示进度条
				progressContainer.classList.add('show');
				submitBtn.disabled = true;
				submitBtn.innerHTML = '<span>上传中...</span>';
				
				// 添加上传中样式
				customInput.classList.add('uploading');
				
				const formData = new FormData(this);
				const xhr = new XMLHttpRequest();
				
				uploadStartTime = Date.now();
				lastLoaded = 0;
				
				// 进度事件
				xhr.upload.addEventListener('progress', function(e) {
					if (e.lengthComputable) {
						const percent = Math.round((e.loaded / e.total) * 100);
						progressFill.style.width = percent + '%';
						progressPercentage.textContent = percent + '%';
						
						// 计算上传速度
						const currentTime = Date.now();
						const timeDiff = (currentTime - uploadStartTime) / 1000; // 秒
						if (timeDiff > 0) {
							const speed = e.loaded / timeDiff; // bytes per second
							progressSpeed.textContent = formatSpeed(speed);
							
							// 计算剩余时间
							if (percent < 100) {
								const remainingBytes = e.total - e.loaded;
								const remainingTime = remainingBytes / speed;
								progressTime.textContent = formatTime(remainingTime);
							}
						}
						
						lastLoaded = e.loaded;
					}
				});
				
				// 完成事件
				xhr.addEventListener('load', function(e) {
					if (xhr.status === 200 || xhr.status === 303) {
						// 上传成功
						progressContainer.classList.add('progress-success');
						progressFill.style.width = '100%';
						progressPercentage.textContent = '100%';
						progressTime.textContent = '完成!';
						
						customInput.classList.remove('uploading');
						customInput.classList.add('success-animation');
						
						showStatus('文件上传成功！正在跳转...', 'success');
						
						// 2秒后跳转
						setTimeout(function() {
							window.location.href = '/files?msg=文件上传成功';
						}, 2000);
					} else {
						handleUploadError('上传失败: ' + xhr.statusText);
					}
				});
				
				// 错误事件
				xhr.addEventListener('error', function() {
					handleUploadError('上传过程中发生错误');
				});
				
				// 中止事件
				xhr.addEventListener('abort', function() {
					handleUploadError('上传已取消');
				});
				
				xhr.open('POST', '/upload');
				xhr.send(formData);
				
				function handleUploadError(message) {
					showStatus(message, 'error');
					submitBtn.disabled = false;
					submitBtn.innerHTML = '<span>重新上传</span>';
					customInput.classList.remove('uploading');
				}
			});
			
			function formatSpeed(bytesPerSecond) {
				if (bytesPerSecond === 0) return '0 KB/s';
				const k = 1024;
				const sizes = ['B/s', 'KB/s', 'MB/s', 'GB/s'];
				const i = Math.floor(Math.log(bytesPerSecond) / Math.log(k));
				return parseFloat((bytesPerSecond / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
			}
			
			function formatTime(seconds) {
				if (seconds === 0) return '--';
				if (seconds < 60) {
					return Math.ceil(seconds) + '秒';
				} else if (seconds < 3600) {
					return Math.ceil(seconds / 60) + '分钟';
				} else {
					return Math.ceil(seconds / 3600) + '小时';
				}
			}
			
			function showStatus(message, type) {
				const statusMessage = document.getElementById('statusMessage');
				statusMessage.textContent = message;
				statusMessage.className = 'status-message status-' + type;
				statusMessage.style.display = 'block';
			}
		</script>
	</body>
	</html>
	`
	
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, tmpl)
}

// 文件下载处理器
func downloadHandler(w http.ResponseWriter, r *http.Request) {
	filename := strings.TrimPrefix(r.URL.Path, "/download/")
	if filename == "" {
		http.Error(w, "文件名不能为空", http.StatusBadRequest)
		return
	}
	
	filepath := filepath.Join("up", filename)
	
	// 检查文件是否存在
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	}
	
	// 设置响应头，触发下载
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	w.Header().Set("Content-Type", "application/octet-stream")
	
	// 提供文件下载
	http.ServeFile(w, r, filepath)
}

// 预览文件处理器
func previewHandler(w http.ResponseWriter, r *http.Request) {
    filename := strings.TrimPrefix(r.URL.Path, "/preview/")
    if filename == "" {
        http.Error(w, "文件名不能为空", http.StatusBadRequest)
        return
    }
    
    // 解码文件名
    decodedFilename, err := url.QueryUnescape(filename)
    if err != nil {
        decodedFilename = filename
    }
    
    filePath := filepath.Join("up", decodedFilename)  // 注意这里使用 filePath 而不是 filepath
    
    // 检查文件是否存在
    if _, err := os.Stat(filePath); os.IsNotExist(err) {
        http.NotFound(w, r)
        return
    }
    
    // 设置正确的Content-Type - 这是你的第434行
    ext := strings.ToLower(filepath.Ext(decodedFilename))  // 这里使用 filepath 包
    
    switch ext {
    case ".jpg", ".jpeg":
        w.Header().Set("Content-Type", "image/jpeg")
    case ".png":
        w.Header().Set("Content-Type", "image/png")
    case ".gif":
        w.Header().Set("Content-Type", "image/gif")
    case ".bmp":
        w.Header().Set("Content-Type", "image/bmp")
    case ".webp":
        w.Header().Set("Content-Type", "image/webp")
    case ".svg":
        w.Header().Set("Content-Type", "image/svg+xml")
    case ".mp4":
        w.Header().Set("Content-Type", "video/mp4")
    case ".avi":
        w.Header().Set("Content-Type", "video/x-msvideo")
    case ".mov":
        w.Header().Set("Content-Type", "video/quicktime")
    case ".mkv":
        w.Header().Set("Content-Type", "video/x-matroska")
    case ".webm":
        w.Header().Set("Content-Type", "video/webm")
    case ".flv":
        w.Header().Set("Content-Type", "video/x-flv")
    case ".mp3":
        w.Header().Set("Content-Type", "audio/mpeg")
    case ".wav":
        w.Header().Set("Content-Type", "audio/wav")
    case ".flac":
        w.Header().Set("Content-Type", "audio/flac")
    case ".ogg":
        w.Header().Set("Content-Type", "audio/ogg")
    case ".m4a":
        w.Header().Set("Content-Type", "audio/mp4")
    case ".aac":
        w.Header().Set("Content-Type", "audio/aac")
    default:
        w.Header().Set("Content-Type", "application/octet-stream")
    }
    
    // 提供文件预览
    http.ServeFile(w, r, filePath)
}

// ==================== 文件列表处理器 - 美化版（带搜索功能） ====================
func filesHandler(w http.ResponseWriter, r *http.Request) {
    // 获取消息参数和搜索参数
    msg := r.URL.Query().Get("msg")
    searchQuery := r.URL.Query().Get("search")
    
    // 读取up目录下的文件
    files, err := os.ReadDir("up")
    if err != nil {
        http.Error(w, "无法读取文件目录", http.StatusInternalServerError)
        return
    }
    
    // 生成文件列表HTML
    fileListHTML := ""
    fileCount := 0
    filteredCount := 0
    
    for _, file := range files {
        if !file.IsDir() {
            fileCount++
            
            // 搜索过滤
            if searchQuery != "" && !strings.Contains(strings.ToLower(file.Name()), strings.ToLower(searchQuery)) {
                continue
            }
            
            filteredCount++
            fileInfo, _ := file.Info()
            fileSize := formatFileSize(fileInfo.Size())
            fileIcon := getFileIcon(file.Name())
            previewButton := getPreviewButton(file.Name())  // 获取预览按钮
            
            fileListHTML += fmt.Sprintf(`
            <li>
                <div class="file-info">
                    <div class="file-icon">%s</div>
                    <div class="file-details">
                        <div class="file-name">%s</div>
                        <div class="file-size">%s</div>
                    </div>
                </div>
                <div class="file-actions">
                    %s
                    <a href="/download/%s" class="btn btn-download" title="下载文件">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path>
                            <polyline points="7 10 12 15 17 10"></polyline>
                            <line x1="12" y1="15" x2="12" y2="3"></line>
                        </svg>
                        下载
                    </a>
                    <a href="/delete-file/%s" class="btn btn-danger" onclick="return confirm('确定删除文件 %s 吗？')" title="删除文件">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <polyline points="3 6 5 6 21 6"></polyline>
                            <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
                        </svg>
                        删除
                    </a>
                </div>
            </li>
            `, fileIcon, template.HTMLEscapeString(file.Name()), fileSize, 
               previewButton,  // 添加预览按钮
               url.QueryEscape(file.Name()), url.QueryEscape(file.Name()), template.HTMLEscapeString(file.Name()))
        }
    }
    
    if fileListHTML == "" {
        if searchQuery != "" {
            fileListHTML = `
            <li class="empty-state">
                <div class="empty-icon">🔍</div>
                <div class="empty-text">未找到匹配的文件</div>
                <div class="empty-subtext">没有找到包含"` + template.HTMLEscapeString(searchQuery) + `"的文件</div>
                <a href="/files" class="btn">查看所有文件</a>
            </li>
            `
        } else {
            fileListHTML = `
            <li class="empty-state">
                <div class="empty-icon">📁</div>
                <div class="empty-text">暂无文件</div>
                <div class="empty-subtext">上传您的第一个文件开始使用</div>
                <a href="/upload" class="btn">上传文件</a>
            </li>
            `
        }
    }
    
    // 显示消息
    alertHTML := ""
    if msg != "" {
        alertHTML = fmt.Sprintf(`<div class="alert alert-success">%s</div>`, template.HTMLEscapeString(msg))
    }
    
    // 搜索框HTML
    searchBoxHTML := `
    <div class="search-box">
        <form method="get" action="/files" class="search-form">
            <div class="search-input-group">
                <input type="text" name="search" value="` + template.HTMLEscapeString(searchQuery) + `" 
                       placeholder="搜索文件..." class="search-input">
                <button type="submit" class="search-btn">
                    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <circle cx="11" cy="11" r="8"></circle>
                        <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
                    </svg>
                </button>
            </div>
            ` + func() string {
                if searchQuery != "" {
                    return `<a href="/files" class="search-clear">清除搜索</a>`
                }
                return ""
            }() + `
        </form>
    </div>`
    
    // 构建完整的HTML
    html := `<!DOCTYPE html>
    <html lang="zh-CN">
    <head>
        <meta charset="UTF-8">
        <meta name="viewport" content="width=device-width, initial-scale=1.0">
        <title>文件管理 - 文件与笔记管理器</title>
        <style>
            * { margin: 0; padding: 0; box-sizing: border-box; }
            body { 
                font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; 
                line-height: 1.6; 
                color: #333; 
                background: linear-gradient(135deg, #f5f7fa 0%, #c3cfe2 100%);
                min-height: 100vh;
            }
            .container { 
                max-width: 1000px; 
                margin: 0 auto; 
                padding: 20px; 
            }
            .header-content {
                display: flex;
                justify-content: space-between;
                align-items: center;
                background: linear-gradient(135deg, #6a11cb 0%, #2575fc 100%);
                color: white;
                padding: 1.5rem 2rem;
                border-radius: 10px;
                margin-bottom: 2rem;
                box-shadow: 0 4px 6px rgba(0,0,0,0.1);
            }
            .header-content h1 {
                font-size: 2rem;
                margin: 0;
            }
            .header-actions {
                display: flex;
                gap: 10px;
            }
            .btn { 
                display: inline-flex;
                align-items: center;
                gap: 8px;
                background: #2575fc; 
                color: white; 
                padding: 10px 20px; 
                border-radius: 5px; 
                text-decoration: none; 
                font-weight: bold; 
                transition: all 0.3s ease;
                border: none;
                cursor: pointer;
                font-size: 0.9rem;
            }
            .btn:hover { 
                background: #1a5fd8; 
                transform: translateY(-2px);
                box-shadow: 0 4px 8px rgba(0,0,0,0.2);
            }
            .btn-secondary { 
                background: #6c757d; 
            }
            .btn-secondary:hover { 
                background: #5a6268; 
            }
            .btn-success { 
                background: #28a745; 
            }
            .btn-success:hover { 
                background: #218838; 
            }
            .btn-preview {
                background: #17a2b8;
            }
            .btn-preview:hover {
                background: #138496;
            }
            .btn-download {
                background: #6f42c1;
            }
            .btn-download:hover {
                background: #5e34b1;
            }
            .btn-danger { 
                background: #dc3545; 
            }
            .btn-danger:hover { 
                background: #c82333; 
            }
            .card {
                background: white;
                border-radius: 10px;
                padding: 2rem;
                box-shadow: 0 5px 15px rgba(0,0,0,0.08);
                margin-bottom: 2rem;
            }
            .card h2 {
                color: #2575fc;
                margin-bottom: 1.5rem;
                padding-bottom: 0.5rem;
                border-bottom: 2px solid #f0f0f0;
            }
            .file-list { 
                list-style: none; 
            }
            .file-list li { 
                padding: 1rem; 
                border-bottom: 1px solid #eee; 
                display: flex; 
                justify-content: space-between; 
                align-items: center;
                transition: background-color 0.2s;
            }
            .file-list li:hover {
                background-color: #f8f9fa;
            }
            .file-list li:last-child { 
                border-bottom: none; 
            }
            .file-info {
                display: flex;
                align-items: center;
                gap: 12px;
                flex: 1;
            }
            .file-icon {
                font-size: 1.5rem;
                width: 40px;
                text-align: center;
            }
            .file-details {
                flex: 1;
            }
            .file-name {
                font-weight: 600;
                color: #212529;
                margin-bottom: 2px;
            }
            .file-size {
                font-size: 0.85rem;
                color: #6c757d;
            }
            .file-actions { 
                display: flex; 
                gap: 8px; 
            }
            .alert { 
                padding: 15px; 
                border-radius: 5px; 
                margin-bottom: 1rem; 
            }
            .alert-success { 
                background: #d4edda; 
                color: #155724; 
                border: 1px solid #c3e6cb; 
            }
            .empty-state {
                text-align: center;
                padding: 3rem 1rem !important;
                flex-direction: column;
                gap: 1rem;
            }
            .empty-icon {
                font-size: 3rem;
                opacity: 0.5;
            }
            .empty-text {
                font-size: 1.2rem;
                font-weight: 600;
                color: #6c757d;
            }
            .empty-subtext {
                color: #6c757d;
                margin-bottom: 1rem;
            }
            .stats {
                display: flex;
                gap: 1rem;
                margin-bottom: 1.5rem;
            }
            .stat-card {
                background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
                color: white;
                padding: 1rem;
                border-radius: 8px;
                flex: 1;
                text-align: center;
            }
            .stat-number {
                font-size: 1.5rem;
                font-weight: bold;
                margin-bottom: 0.5rem;
            }
            .stat-label {
                font-size: 0.9rem;
                opacity: 0.9;
            }
            /* 搜索框样式 */
            .search-box {
                margin-bottom: 1.5rem;
            }
            .search-form {
                display: flex;
                align-items: center;
                gap: 10px;
            }
            .search-input-group {
                display: flex;
                flex: 1;
                max-width: 400px;
                position: relative;
            }
            .search-input {
                flex: 1;
                padding: 12px 50px 12px 15px;
                border: 2px solid #e9ecef;
                border-radius: 25px;
                font-size: 1rem;
                transition: all 0.3s ease;
                background: white;
            }
            .search-input:focus {
                outline: none;
                border-color: #2575fc;
                box-shadow: 0 0 0 3px rgba(37, 117, 252, 0.1);
            }
            .search-btn {
                position: absolute;
                right: 5px;
                top: 50%;
                transform: translateY(-50%);
                background: #2575fc;
                border: none;
                border-radius: 50%;
                width: 36px;
                height: 36px;
                display: flex;
                align-items: center;
                justify-content: center;
                color: white;
                cursor: pointer;
                transition: all 0.3s ease;
            }
            .search-btn:hover {
                background: #1a5fd8;
                transform: translateY(-50%) scale(1.05);
            }
            .search-clear {
                color: #6c757d;
                text-decoration: none;
                font-size: 0.9rem;
                padding: 8px 16px;
                border-radius: 5px;
                transition: all 0.3s ease;
            }
            .search-clear:hover {
                color: #495057;
                background: #f8f9fa;
            }
            .search-info {
                color: #6c757d;
                font-size: 0.9rem;
                margin-bottom: 1rem;
            }
            /* 预览模态框样式 */
            .modal {
                display: none;
                position: fixed;
                z-index: 1000;
                left: 0;
                top: 0;
                width: 100%;
                height: 100%;
                background-color: rgba(0,0,0,0.8);
                backdrop-filter: blur(5px);
            }
            .modal-content {
                position: relative;
                margin: 5% auto;
                width: 90%;
                max-width: 800px;
                max-height: 90vh;
                background: white;
                border-radius: 10px;
                overflow: hidden;
                box-shadow: 0 10px 30px rgba(0,0,0,0.3);
            }
            .modal-header {
                display: flex;
                justify-content: space-between;
                align-items: center;
                padding: 1rem 1.5rem;
                background: #f8f9fa;
                border-bottom: 1px solid #dee2e6;
            }
            .modal-title {
                font-weight: 600;
                color: #212529;
            }
            .close {
                background: none;
                border: none;
                font-size: 1.5rem;
                cursor: pointer;
                color: #6c757d;
                padding: 0;
                width: 30px;
                height: 30px;
                display: flex;
                align-items: center;
                justify-content: center;
                border-radius: 50%;
                transition: all 0.3s ease;
            }
            .close:hover {
                background: #e9ecef;
                color: #495057;
            }
            .modal-body {
                padding: 0;
                text-align: center;
                max-height: calc(90vh - 60px);
                overflow: auto;
            }
            .preview-image, .preview-video, .preview-audio {
                max-width: 100%;
                max-height: 70vh;
                display: block;
                margin: 0 auto;
            }
            .preview-audio {
                width: 100%;
                padding: 2rem;
            }
            .unsupported-preview {
                padding: 3rem 2rem;
                text-align: center;
                color: #6c757d;
            }
            .unsupported-icon {
                font-size: 3rem;
                margin-bottom: 1rem;
                opacity: 0.5;
            }
            .audio-preview-container {
                padding: 3rem 2rem;
                background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
                border-radius: 12px;
                margin: 1rem;
                text-align: center;
                color: white;
            }
            .audio-icon {
                font-size: 3rem;
                margin-bottom: 1rem;
            }
            .audio-title {
                font-size: 1.2rem;
                font-weight: 600;
                margin-bottom: 0.5rem;
            }
            .audio-filename {
                font-size: 0.9rem;
                opacity: 0.9;
                margin-bottom: 2rem;
            }
            .audio-player {
                width: 100%;
                max-width: 500px;
                margin: 0 auto;
                background: white;
                border-radius: 25px;
                padding: 10px;
            }
        </style>
    </head>
    <body>
        <div class="container">
            <header class="header-content">
                <h1>文件管理</h1>
                <div class="header-actions">
                    <a href="/" class="btn btn-secondary">返回主页</a>
                    <a href="/upload" class="btn btn-success">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path>
                            <polyline points="17 8 12 3 7 8"></polyline>
                            <line x1="12" y1="3" x2="12" y2="15"></line>
                        </svg>
                        上传文件
                    </a>
                </div>
            </header>
            
            ` + alertHTML + `
            
            <div class="stats">
                <div class="stat-card">
                    <div class="stat-number">` + fmt.Sprintf("%d", fileCount) + `</div>
                    <div class="stat-label">总文件数</div>
                </div>
                <div class="stat-card">
                    <div class="stat-number">` + fmt.Sprintf("%d", filteredCount) + `</div>
                    <div class="stat-label">` + func() string {
                        if searchQuery != "" {
                            return "匹配文件数"
                        }
                        return "显示文件数"
                    }() + `</div>
                </div>
            </div>
            
            ` + searchBoxHTML + `
            
            ` + func() string {
                if searchQuery != "" {
                    return `<div class="search-info">搜索关键词: "<strong>` + template.HTMLEscapeString(searchQuery) + `</strong>" - 找到 ` + fmt.Sprintf("%d", filteredCount) + ` 个文件</div>`
                }
                return ""
            }() + `
            
            <div class="card">
                <h2>文件列表</h2>
                <ul class="file-list">
                    ` + fileListHTML + `
                </ul>
            </div>
        </div>

        <!-- 预览模态框 -->
        <div id="previewModal" class="modal">
            <div class="modal-content">
                <div class="modal-header">
                    <div class="modal-title" id="previewTitle">文件预览</div>
                    <button class="close" onclick="closePreview()">&times;</button>
                </div>
                <div class="modal-body" id="previewBody">
                    <!-- 预览内容将在这里动态加载 -->
                </div>
            </div>
        </div>

        <script>
            // 预览文件函数
            function previewFile(filename, fileType) {
                const modal = document.getElementById('previewModal');
                const modalTitle = document.getElementById('previewTitle');
                const modalBody = document.getElementById('previewBody');
                
                // 解码文件名（处理中文等特殊字符）
                const decodedFilename = decodeURIComponent(filename);
                modalTitle.textContent = '预览: ' + decodedFilename;
                
                // 根据文件类型设置预览内容
                let previewContent = '';
                
                switch(fileType) {
                    case 'image':
                        previewContent = '<img src="/preview/' + filename + '" class="preview-image" alt="' + decodedFilename + '" style="max-width: 100%; max-height: 70vh; display: block; margin: 0 auto;">';
                        break;
                    case 'video':
                        previewContent = '<video controls class="preview-video" style="width: 100%; max-height: 70vh; display: block; margin: 0 auto;">' +
                                        '<source src="/preview/' + filename + '" type="' + getVideoMimeType(filename) + '">' +
                                        '您的浏览器不支持视频预览。' +
                                        '</video>';
                        break;
                    case 'audio':
                        previewContent = '<div class="audio-preview-container">' +
                                        '<div class="audio-icon">🎵</div>' +
                                        '<div class="audio-title">正在播放音频</div>' +
                                        '<div class="audio-filename">' + decodedFilename + '</div>' +
                                        '<audio controls class="audio-player">' +
                                        '<source src="/preview/' + filename + '" type="' + getAudioMimeType(filename) + '">' +
                                        '您的浏览器不支持音频预览。' +
                                        '</audio>' +
                                        '</div>';
                        break;
                    default:
                        previewContent = '<div class="unsupported-preview">' +
                                        '<div class="unsupported-icon">📄</div>' +
                                        '<h3>不支持预览</h3>' +
                                        '<p>此文件类型不支持在线预览。</p>' +
                                        '<p>请下载文件后查看。</p>' +
                                        '</div>';
                }
                
                modalBody.innerHTML = previewContent;
                modal.style.display = 'block';
                
                // 点击模态框背景关闭
                modal.addEventListener('click', function(e) {
                    if (e.target === modal) {
                        closePreview();
                    }
                });
                
                // ESC键关闭
                document.addEventListener('keydown', function(e) {
                    if (e.key === 'Escape') {
                        closePreview();
                    }
                });
            }
            
            // 获取音频MIME类型
            function getAudioMimeType(filename) {
                const ext = filename.toLowerCase().split('.').pop();
                const mimeTypes = {
                    'mp3': 'audio/mpeg',
                    'wav': 'audio/wav',
                    'flac': 'audio/flac',
                    'ogg': 'audio/ogg',
                    'm4a': 'audio/mp4',
                    'aac': 'audio/aac'
                };
                return mimeTypes[ext] || 'audio/mpeg';
            }
            
            // 获取视频MIME类型
            function getVideoMimeType(filename) {
                const ext = filename.toLowerCase().split('.').pop();
                const mimeTypes = {
                    'mp4': 'video/mp4',
                    'avi': 'video/x-msvideo',
                    'mov': 'video/quicktime',
                    'mkv': 'video/x-matroska',
                    'webm': 'video/webm',
                    'flv': 'video/x-flv'
                };
                return mimeTypes[ext] || 'video/mp4';
            }
            
            // 关闭预览
            function closePreview() {
                const modal = document.getElementById('previewModal');
                const modalBody = document.getElementById('previewBody');
                
                // 停止所有媒体播放
                const videos = modalBody.getElementsByTagName('video');
                const audios = modalBody.getElementsByTagName('audio');
                
                for (let video of videos) {
                    video.pause();
                    video.currentTime = 0;
                }
                
                for (let audio of audios) {
                    audio.pause();
                    audio.currentTime = 0;
                }
                
                modal.style.display = 'none';
            }
        </script>
    </body>
    </html>`
    
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    fmt.Fprint(w, html)
}

// 删除文件处理器
func deleteFileHandler(w http.ResponseWriter, r *http.Request) {
    // 获取完整的URL路径
    path := r.URL.Path
    
    // 确保路径以/delete-file/开头
    if !strings.HasPrefix(path, "/delete-file/") {
        http.Error(w, "无效的路径", http.StatusBadRequest)
        return
    }
    
    // 提取文件名（保留原始编码）
    filename := strings.TrimPrefix(path, "/delete-file/")
    if filename == "" {
        http.Error(w, "文件名不能为空", http.StatusBadRequest)
        return
    }
    
    // 解码URL编码的文件名
    decodedFilename, err := url.QueryUnescape(filename)
    if err != nil {
        // 如果解码失败，使用原始文件名
        decodedFilename = filename
    }
    
    filepath := filepath.Join("up", decodedFilename)
    
    // 检查文件是否存在
    if _, err := os.Stat(filepath); os.IsNotExist(err) {
        http.NotFound(w, r)
        return
    }
    
    // 删除文件
    err = os.Remove(filepath)
    if err != nil {
        http.Error(w, "删除文件失败: "+err.Error(), http.StatusInternalServerError)
        return
    }
    
    // 重定向到文件列表页，显示成功消息
    http.Redirect(w, r, "/files?msg=文件 "+decodedFilename+" 删除成功", http.StatusSeeOther)
}

// ==================== 笔记列表处理器 - 美化版（带搜索功能） ====================
func notesHandler(w http.ResponseWriter, r *http.Request) {
    // 获取搜索参数
    searchQuery := r.URL.Query().Get("search")
    
    // 生成笔记列表HTML
    noteListHTML := ""
    filteredCount := 0
    
    for _, title := range noteTitles {
        note := notes[title]
        preview := getNotePreview(note.Body)
        
        // 搜索过滤（搜索标题和内容）
        if searchQuery != "" {
            titleMatch := strings.Contains(strings.ToLower(title), strings.ToLower(searchQuery))
            bodyMatch := strings.Contains(strings.ToLower(note.Body), strings.ToLower(searchQuery))
            if !titleMatch && !bodyMatch {
                continue
            }
        }
        
        filteredCount++
        noteListHTML += fmt.Sprintf(`
        <li>
            <div class="note-info">
                <div class="note-title">%s</div>
                <div class="note-preview">%s</div>
                <div class="note-meta">创建时间: 刚刚</div>
            </div>
            <div class="note-actions">
                <a href="/note/%s" class="btn btn-edit" title="编辑笔记">
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"></path>
                        <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"></path>
                    </svg>
                    编辑
                </a>
                <a href="/delete-note/%s" class="btn btn-danger" onclick="return confirm('确定删除笔记 %s 吗？')" title="删除笔记">
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <polyline points="3 6 5 6 21 6"></polyline>
                        <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
                    </svg>
                    删除
                </a>
            </div>
        </li>
        `, template.HTMLEscapeString(title), template.HTMLEscapeString(preview), 
           url.QueryEscape(title), url.QueryEscape(title), template.HTMLEscapeString(title))
    }
    
    if noteListHTML == "" {
        if searchQuery != "" {
            noteListHTML = `
            <li class="empty-state">
                <div class="empty-icon">🔍</div>
                <div class="empty-text">未找到匹配的笔记</div>
                <div class="empty-subtext">没有找到包含"` + template.HTMLEscapeString(searchQuery) + `"的笔记</div>
                <a href="/notes" class="btn">查看所有笔记</a>
            </li>
            `
        } else {
            noteListHTML = `
            <li class="empty-state">
                <div class="empty-icon">📝</div>
                <div class="empty-text">暂无笔记</div>
                <div class="empty-subtext">创建您的第一个笔记开始记录</div>
                <a href="/note/new" class="btn">新建笔记</a>
            </li>
            `
        }
    }
    
    // 搜索框HTML
    searchBoxHTML := `
    <div class="search-box">
        <form method="get" action="/notes" class="search-form">
            <div class="search-input-group">
                <input type="text" name="search" value="` + template.HTMLEscapeString(searchQuery) + `" 
                       placeholder="搜索笔记标题或内容..." class="search-input">
                <button type="submit" class="search-btn">
                    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <circle cx="11" cy="11" r="8"></circle>
                        <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
                    </svg>
                </button>
            </div>
            ` + func() string {
                if searchQuery != "" {
                    return `<a href="/notes" class="search-clear">清除搜索</a>`
                }
                return ""
            }() + `
        </form>
    </div>`
    
    // 构建完整的HTML
    html := `<!DOCTYPE html>
    <html lang="zh-CN">
    <head>
        <meta charset="UTF-8">
        <meta name="viewport" content="width=device-width, initial-scale=1.0">
        <title>笔记管理 - 文件与笔记管理器</title>
        <style>
            * { margin: 0; padding: 0; box-sizing: border-box; }
            body { 
                font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; 
                line-height: 1.6; 
                color: #333; 
                background: linear-gradient(135deg, #f5f7fa 0%, #c3cfe2 100%);
                min-height: 100vh;
            }
            .container { 
                max-width: 1000px; 
                margin: 0 auto; 
                padding: 20px; 
            }
            .header-content {
                display: flex;
                justify-content: space-between;
                align-items: center;
                background: linear-gradient(135deg, #6a11cb 0%, #2575fc 100%);
                color: white;
                padding: 1.5rem 2rem;
                border-radius: 10px;
                margin-bottom: 2rem;
                box-shadow: 0 4px 6px rgba(0,0,0,0.1);
            }
            .header-content h1 {
                font-size: 2rem;
                margin: 0;
            }
            .header-actions {
                display: flex;
                gap: 10px;
            }
            .btn { 
                display: inline-flex;
                align-items: center;
                gap: 8px;
                background: #2575fc; 
                color: white; 
                padding: 10px 20px; 
                border-radius: 5px; 
                text-decoration: none; 
                font-weight: bold; 
                transition: all 0.3s ease;
                border: none;
                cursor: pointer;
                font-size: 0.9rem;
            }
            .btn:hover { 
                background: #1a5fd8; 
                transform: translateY(-2px);
                box-shadow: 0 4px 8px rgba(0,0,0,0.2);
            }
            .btn-secondary { 
                background: #6c757d; 
            }
            .btn-secondary:hover { 
                background: #5a6268; 
            }
            .btn-success { 
                background: #28a745; 
            }
            .btn-success:hover { 
                background: #218838; 
            }
            .btn-edit {
                background: #ffc107;
                color: #212529;
            }
            .btn-edit:hover {
                background: #e0a800;
            }
            .btn-danger { 
                background: #dc3545; 
            }
            .btn-danger:hover { 
                background: #c82333; 
            }
            .card {
                background: white;
                border-radius: 10px;
                padding: 2rem;
                box-shadow: 0 5px 15px rgba(0,0,0,0.08);
                margin-bottom: 2rem;
            }
            .card h2 {
                color: #2575fc;
                margin-bottom: 1.5rem;
                padding-bottom: 0.5rem;
                border-bottom: 2px solid #f0f0f0;
            }
            .note-list { 
                list-style: none; 
            }
            .note-list li { 
                padding: 1.5rem; 
                border-bottom: 1px solid #eee; 
                display: flex; 
                justify-content: space-between; 
                align-items: flex-start;
                transition: background-color 0.2s;
                border-radius: 8px;
                margin-bottom: 0.5rem;
            }
            .note-list li:hover {
                background-color: #f8f9fa;
                box-shadow: 0 2px 8px rgba(0,0,0,0.05);
            }
            .note-list li:last-child { 
                border-bottom: none; 
                margin-bottom: 0;
            }
            .note-info {
                flex: 1;
                margin-right: 1rem;
            }
            .note-title {
                font-size: 1.2rem;
                font-weight: 600;
                color: #212529;
                margin-bottom: 0.5rem;
            }
            .note-preview {
                color: #6c757d;
                font-size: 0.95rem;
                line-height: 1.4;
                margin-bottom: 0.5rem;
                display: -webkit-box;
                -webkit-line-clamp: 2;
                -webkit-box-orient: vertical;
                overflow: hidden;
            }
            .note-meta {
                font-size: 0.8rem;
                color: #adb5bd;
            }
            .note-actions { 
                display: flex; 
                gap: 8px; 
            }
            .empty-state {
                text-align: center;
                padding: 3rem 1rem !important;
                flex-direction: column;
                gap: 1rem;
            }
            .empty-icon {
                font-size: 3rem;
                opacity: 0.5;
            }
            .empty-text {
                font-size: 1.2rem;
                font-weight: 600;
                color: #6c757d;
            }
            .empty-subtext {
                color: #6c757d;
                margin-bottom: 1rem;
            }
            .stats {
                display: flex;
                gap: 1rem;
                margin-bottom: 1.5rem;
            }
            .stat-card {
                background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
                color: white;
                padding: 1rem;
                border-radius: 8px;
                flex: 1;
                text-align: center;
            }
            .stat-number {
                font-size: 1.5rem;
                font-weight: bold;
                margin-bottom: 0.5rem;
            }
            .stat-label {
                font-size: 0.9rem;
                opacity: 0.9;
            }
            /* 搜索框样式 */
            .search-box {
                margin-bottom: 1.5rem;
            }
            .search-form {
                display: flex;
                align-items: center;
                gap: 10px;
            }
            .search-input-group {
                display: flex;
                flex: 1;
                max-width: 400px;
                position: relative;
            }
            .search-input {
                flex: 1;
                padding: 12px 50px 12px 15px;
                border: 2px solid #e9ecef;
                border-radius: 25px;
                font-size: 1rem;
                transition: all 0.3s ease;
                background: white;
            }
            .search-input:focus {
                outline: none;
                border-color: #2575fc;
                box-shadow: 0 0 0 3px rgba(37, 117, 252, 0.1);
            }
            .search-btn {
                position: absolute;
                right: 5px;
                top: 50%;
                transform: translateY(-50%);
                background: #2575fc;
                border: none;
                border-radius: 50%;
                width: 36px;
                height: 36px;
                display: flex;
                align-items: center;
                justify-content: center;
                color: white;
                cursor: pointer;
                transition: all 0.3s ease;
            }
            .search-btn:hover {
                background: #1a5fd8;
                transform: translateY(-50%) scale(1.05);
            }
            .search-clear {
                color: #6c757d;
                text-decoration: none;
                font-size: 0.9rem;
                padding: 8px 16px;
                border-radius: 5px;
                transition: all 0.3s ease;
            }
            .search-clear:hover {
                color: #495057;
                background: #f8f9fa;
            }
            .search-info {
                color: #6c757d;
                font-size: 0.9rem;
                margin-bottom: 1rem;
            }
        </style>
    </head>
    <body>
        <div class="container">
            <header class="header-content">
                <h1>笔记管理</h1>
                <div class="header-actions">
                    <a href="/" class="btn btn-secondary">返回主页</a>
                    <a href="/note/new" class="btn btn-success">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <line x1="12" y1="5" x2="12" y2="19"></line>
                            <line x1="5" y1="12" x2="19" y2="12"></line>
                        </svg>
                        新建笔记
                    </a>
                </div>
            </header>
            
            <div class="stats">
                <div class="stat-card">
                    <div class="stat-number">` + fmt.Sprintf("%d", len(noteTitles)) + `</div>
                    <div class="stat-label">总笔记数</div>
                </div>
                <div class="stat-card">
                    <div class="stat-number">` + fmt.Sprintf("%d", filteredCount) + `</div>
                    <div class="stat-label">` + func() string {
                        if searchQuery != "" {
                            return "匹配笔记数"
                        }
                        return "显示笔记数"
                    }() + `</div>
                </div>
            </div>
            
            ` + searchBoxHTML + `
            
            ` + func() string {
                if searchQuery != "" {
                    return `<div class="search-info">搜索关键词: "<strong>` + template.HTMLEscapeString(searchQuery) + `</strong>" - 找到 ` + fmt.Sprintf("%d", filteredCount) + ` 个笔记</div>`
                }
                return ""
            }() + `
            
            <div class="card">
                <h2>笔记列表</h2>
                <ul class="note-list">
                    ` + noteListHTML + `
                </ul>
            </div>
        </div>
    </body>
    </html>`
    
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    fmt.Fprint(w, html)
}

// ==================== 笔记编辑器处理器 - 美化版 ====================
func noteHandler(w http.ResponseWriter, r *http.Request) {
	title := strings.TrimPrefix(r.URL.Path, "/note/")
	
	var note *Note
	var isNew bool
	
	if title == "new" {
		// 新建笔记
		note = &Note{Title: "", Body: ""}
		isNew = true
	} else {
		// 编辑现有笔记
		var exists bool
		note, exists = notes[title]
		if !exists {
			http.NotFound(w, r)
			return
		}
		isNew = false
	}
	
	pageTitle := "新建笔记"
	if !isNew {
		pageTitle = "编辑笔记: " + note.Title
	}
	
	// 显示笔记编辑器
	tmpl := `
	<!DOCTYPE html>
	<html lang="zh-CN">
	<head>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<title>{{.PageTitle}} - 文件与笔记管理器</title>
		<style>
			* { margin: 0; padding: 0; box-sizing: border-box; }
			body { 
				font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; 
				line-height: 1.6; 
				color: #333; 
				background: linear-gradient(135deg, #f5f7fa 0%, #c3cfe2 100%);
				min-height: 100vh;
			}
			.container { 
				max-width: 900px; 
				margin: 0 auto; 
				padding: 20px; 
			}
			.header-content {
				display: flex;
				justify-content: space-between;
				align-items: center;
				background: linear-gradient(135deg, #6a11cb 0%, #2575fc 100%);
				color: white;
				padding: 1.5rem 2rem;
				border-radius: 10px;
				margin-bottom: 2rem;
				box-shadow: 0 4px 6px rgba(0,0,0,0.1);
			}
			.header-content h1 {
				font-size: 1.8rem;
				margin: 0;
			}
			.btn { 
				display: inline-flex;
				align-items: center;
				gap: 8px;
				background: #2575fc; 
				color: white; 
				padding: 10px 20px; 
				border-radius: 5px; 
				text-decoration: none; 
				font-weight: bold; 
				transition: all 0.3s ease;
				border: none;
				cursor: pointer;
				font-size: 0.9rem;
			}
			.btn:hover { 
				background: #1a5fd8; 
				transform: translateY(-2px);
				box-shadow: 0 4px 8px rgba(0,0,0,0.2);
			}
			.btn-secondary { 
				background: #6c757d; 
			}
			.btn-secondary:hover { 
				background: #5a6268; 
			}
			.btn-success { 
				background: #28a745; 
			}
			.btn-success:hover { 
				background: #218838; 
			}
			.card {
				background: white;
				border-radius: 10px;
				padding: 2rem;
				box-shadow: 0 5px 15px rgba(0,0,0,0.08);
				margin-bottom: 2rem;
			}
			.card h2 {
				color: #2575fc;
				margin-bottom: 1.5rem;
				padding-bottom: 0.5rem;
				border-bottom: 2px solid #f0f0f0;
			}
			.form-group { 
				margin-bottom: 1.5rem; 
			}
			.form-group label { 
				display: block; 
				margin-bottom: 0.5rem; 
				font-weight: 600;
				color: #495057;
			}
			.form-control { 
				width: 100%; 
				padding: 12px 15px; 
				border: 2px solid #e9ecef; 
				border-radius: 8px; 
				font-size: 1rem; 
				transition: border-color 0.3s, box-shadow 0.3s;
				font-family: inherit;
			}
			.form-control:focus {
				outline: none;
				border-color: #2575fc;
				box-shadow: 0 0 0 3px rgba(37, 117, 252, 0.1);
			}
			.form-control[readonly] {
				background-color: #f8f9fa;
				border-color: #dee2e6;
				color: #6c757d;
				cursor: not-allowed;
			}
			textarea.form-control { 
				min-height: 400px; 
				resize: vertical; 
				line-height: 1.5;
				font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
			}
			.form-actions {
				display: flex;
				justify-content: space-between;
				align-items: center;
				margin-top: 2rem;
				padding-top: 1rem;
				border-top: 1px solid #e9ecef;
			}
			.form-help {
				color: #6c757d;
				font-size: 0.9rem;
			}
			.char-count {
				text-align: right;
				color: #6c757d;
				font-size: 0.85rem;
				margin-top: 0.5rem;
			}
		</style>
	</head>
	<body>
		<div class="container">
			<header class="header-content">
				<h1>{{.PageTitle}}</h1>
				<a href="/notes" class="btn btn-secondary">
					<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<line x1="19" y1="12" x2="5" y2="12"></line>
						<polyline points="12 19 5 12 12 5"></polyline>
					</svg>
					返回笔记列表
				</a>
			</header>
			
			<div class="card">
				<form action="/save-note" method="post" id="noteForm">
					<div class="form-group">
						<label for="title">笔记标题:</label>
						<input type="text" name="title" id="title" class="form-control" 
							   value="{{.Title}}" {{if not .IsNew}}readonly{{end}} 
							   required maxlength="100" placeholder="请输入笔记标题">
						<div class="char-count">
							<span id="titleCount">0</span>/100
						</div>
					</div>
					<div class="form-group">
						<label for="body">笔记内容:</label>
						<textarea name="body" id="body" class="form-control" 
								  required placeholder="请输入笔记内容...">{{.Body}}</textarea>
						<div class="char-count">
							<span id="bodyCount">0</span> 字符
						</div>
					</div>
					<input type="hidden" name="isNew" value="{{.IsNew}}">
					<input type="hidden" name="oldTitle" value="{{.Title}}">
					
					<div class="form-actions">
						<div class="form-help">
							{{if .IsNew}}
								创建新笔记 - 内容将自动保存
							{{else}}
								编辑现有笔记 - 修改将自动保存
							{{end}}
						</div>
						<button type="submit" class="btn btn-success">
							<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
								<path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z"></path>
								<polyline points="17 21 17 13 7 13 7 21"></polyline>
								<polyline points="7 3 7 8 15 8"></polyline>
							</svg>
							保存笔记
						</button>
					</div>
				</form>
			</div>
		</div>

		<script>
			// 字符计数
			const titleInput = document.getElementById('title');
			const bodyInput = document.getElementById('body');
			const titleCount = document.getElementById('titleCount');
			const bodyCount = document.getElementById('bodyCount');
			
			function updateCounts() {
				titleCount.textContent = titleInput.value.length;
				bodyCount.textContent = bodyInput.value.length;
			}
			
			titleInput.addEventListener('input', updateCounts);
			bodyInput.addEventListener('input', updateCounts);
			
			// 初始化计数
			updateCounts();
			
			// 自动保存草稿（可选功能）
			let saveTimeout;
			bodyInput.addEventListener('input', () => {
				clearTimeout(saveTimeout);
				saveTimeout = setTimeout(() => {
					// 这里可以添加自动保存逻辑
					console.log('内容已更改，可以添加自动保存功能');
				}, 2000);
			});
			
			// 表单提交确认
			document.getElementById('noteForm').addEventListener('submit', function(e) {
				const title = titleInput.value.trim();
				const body = bodyInput.value.trim();
				
				if (!title) {
					e.preventDefault();
					alert('请输入笔记标题');
					titleInput.focus();
					return;
				}
				
				if (!body) {
					e.preventDefault();
					alert('请输入笔记内容');
					bodyInput.focus();
					return;
				}
			});
		</script>
	</body>
	</html>
	`
	
	data := struct {
		Title     string
		Body      string
		IsNew     bool
		PageTitle string
	}{
		Title:     note.Title,
		Body:      note.Body,
		IsNew:     isNew,
		PageTitle: pageTitle,
	}
	
	t, err := template.New("note").Parse(tmpl)
	if err != nil {
		http.Error(w, "模板解析错误", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	t.Execute(w, data)
}

// 保存笔记处理器
func saveNoteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}
	
	r.ParseForm()
	title := r.FormValue("title")
	body := r.FormValue("body")
	isNew := r.FormValue("isNew") == "true"
	oldTitle := r.FormValue("oldTitle")
	
	if title == "" {
		http.Error(w, "标题不能为空", http.StatusBadRequest)
		return
	}
	
	// 如果是编辑现有笔记且标题改变，需要删除旧笔记文件
	if !isNew && oldTitle != title {
		deleteNoteFile(oldTitle)
		delete(notes, oldTitle)
		// 从noteTitles中移除旧标题
		for i, t := range noteTitles {
			if t == oldTitle {
				noteTitles = append(noteTitles[:i], noteTitles[i+1:]...)
				break
			}
		}
	}
	
	// 保存笔记到文件
	err := saveNoteToFile(title, body)
	if err != nil {
		http.Error(w, "保存笔记失败: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	// 更新内存中的笔记数据
	notes[title] = &Note{Title: title, Body: body}
	
	// 如果标题不在noteTitles中，添加它
	found := false
	for _, t := range noteTitles {
		if t == title {
			found = true
			break
		}
	}
	if !found {
		noteTitles = append(noteTitles, title)
	}
	
	// 重定向到笔记列表
	http.Redirect(w, r, "/notes", http.StatusSeeOther)
}

// 删除笔记处理器
func deleteNoteHandler(w http.ResponseWriter, r *http.Request) {
	title := strings.TrimPrefix(r.URL.Path, "/delete-note/")
	
	if title == "" {
		http.Error(w, "笔记标题不能为空", http.StatusBadRequest)
		return
	}
	
	// 删除笔记文件
	err := deleteNoteFile(title)
	if err != nil && !os.IsNotExist(err) {
		http.Error(w, "删除笔记文件失败: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	// 从内存中删除笔记
	delete(notes, title)
	
	// 从noteTitles中移除
	for i, t := range noteTitles {
		if t == title {
			noteTitles = append(noteTitles[:i], noteTitles[i+1:]...)
			break
		}
	}
	
	// 重定向到笔记列表
	http.Redirect(w, r, "/notes", http.StatusSeeOther)
}

// 辅助函数：格式化文件大小
func formatFileSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	} else if size < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(size)/1024)
	} else if size < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
	} else {
		return fmt.Sprintf("%.1f GB", float64(size)/(1024*1024*1024))
	}
}

// 辅助函数：获取文件图标
func getFileIcon(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".txt", ".md":
		return "📄"
	case ".pdf":
		return "📕"
	case ".doc", ".docx":
		return "📘"
	case ".xls", ".xlsx":
		return "📗"
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp":
		return "🖼️"
	case ".mp3", ".wav", ".flac":
		return "🎵"
	case ".mp4", ".avi", ".mov":
		return "🎬"
	case ".zip", ".rar", ".7z":
		return "📦"
	default:
		return "📎"
	}
}

// 辅助函数：获取笔记预览
func getNotePreview(body string) string {
	// 移除换行和多余空格
	preview := strings.TrimSpace(body)
	preview = strings.ReplaceAll(preview, "\n", " ")
	
	// 限制长度
	if len(preview) > 100 {
		preview = preview[:100] + "..."
	}
	
	if preview == "" {
		preview = "无内容"
	}
	
	return preview
}

// 辅助函数：获取预览按钮
func getPreviewButton(filename string) string {
    ext := strings.ToLower(filepath.Ext(filename))
    
    // 支持的图片格式
    imageExts := []string{".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp", ".svg"}
    // 支持的视频格式
    videoExts := []string{".mp4", ".avi", ".mov", ".mkv", ".webm", ".flv"}
    // 支持的音频格式
    audioExts := []string{".mp3", ".wav", ".flac", ".ogg", ".m4a", ".aac"}
    
    for _, imgExt := range imageExts {
        if ext == imgExt {
            return fmt.Sprintf(`<button type="button" class="btn btn-preview" onclick="previewFile('%s', 'image')" title="预览图片">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <rect x="3" y="3" width="18" height="18" rx="2" ry="2"></rect>
                    <circle cx="8.5" cy="8.5" r="1.5"></circle>
                    <polyline points="21 15 16 10 5 21"></polyline>
                </svg>
                预览
            </button>`, url.QueryEscape(filename))
        }
    }
    
    for _, vidExt := range videoExts {
        if ext == vidExt {
            return fmt.Sprintf(`<button type="button" class="btn btn-preview" onclick="previewFile('%s', 'video')" title="预览视频">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <polygon points="23 7 16 12 23 17 23 7"></polygon>
                    <rect x="1" y="5" width="15" height="14" rx="2" ry="2"></rect>
                </svg>
                预览
            </button>`, url.QueryEscape(filename))
        }
    }
    
    for _, audExt := range audioExts {
        if ext == audExt {
            return fmt.Sprintf(`<button type="button" class="btn btn-preview" onclick="previewFile('%s', 'audio')" title="预览音频">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M9 18V5l12-2v13"></path>
                    <circle cx="6" cy="18" r="3"></circle>
                    <circle cx="18" cy="16" r="3"></circle>
                </svg>
                预览
            </button>`, url.QueryEscape(filename))
        }
    }
    
    // 不支持预览的文件类型不显示预览按钮
    return ""
}

// 预览文件处理器 - 修复音频预览
func previewHandler(w http.ResponseWriter, r *http.Request) {
    filename := strings.TrimPrefix(r.URL.Path, "/preview/")
    if filename == "" {
        http.Error(w, "文件名不能为空", http.StatusBadRequest)
        return
    }
    
    // 解码文件名
    decodedFilename, err := url.QueryUnescape(filename)
    if err != nil {
        decodedFilename = filename
    }
    
    filePath := filepath.Join("up", decodedFilename)
    
    // 检查文件是否存在
    if _, err := os.Stat(filePath); os.IsNotExist(err) {
        http.NotFound(w, r)
        return
    }
    
    // 设置正确的Content-Type
    ext := strings.ToLower(filepath.Ext(decodedFilename))
    
    // 详细的MIME类型映射
    mimeTypes := map[string]string{
        // 图片格式
        ".jpg":  "image/jpeg",
        ".jpeg": "image/jpeg",
        ".png":  "image/png",
        ".gif":  "image/gif",
        ".bmp":  "image/bmp",
        ".webp": "image/webp",
        ".svg":  "image/svg+xml",
        
        // 视频格式
        ".mp4":  "video/mp4",
        ".avi":  "video/x-msvideo",
        ".mov":  "video/quicktime",
        ".mkv":  "video/x-matroska",
        ".webm": "video/webm",
        ".flv":  "video/x-flv",
        
        // 音频格式
        ".mp3":  "audio/mpeg",
        ".wav":  "audio/wav",
        ".flac": "audio/flac",
        ".ogg":  "audio/ogg",
        ".m4a":  "audio/mp4",
        ".aac":  "audio/aac",
    }
    
    if mimeType, exists := mimeTypes[ext]; exists {
        w.Header().Set("Content-Type", mimeType)
    } else {
        w.Header().Set("Content-Type", "application/octet-stream")
    }
    
    // 设置缓存控制头，避免重复请求
    w.Header().Set("Cache-Control", "public, max-age=3600") // 缓存1小时
    
    // 提供文件预览
    http.ServeFile(w, r, filePath)
}

// 加载笔记
func loadNotes() {
	// 确保note目录存在
	os.Mkdir("note", 0755)
	
	// 读取note目录下的所有文件
	files, err := os.ReadDir("note")
	if err != nil {
		fmt.Printf("读取笔记目录失败: %v\n", err)
		// 创建示例笔记
		createSampleNotes()
		return
	}
	
	// 清空当前笔记数据
	notes = make(map[string]*Note)
	noteTitles = []string{}
	
	// 遍历所有文件并加载笔记内容
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		
		// 只处理.txt文件
		if strings.HasSuffix(file.Name(), ".txt") {
			title := strings.TrimSuffix(file.Name(), ".txt")
			content, err := os.ReadFile(filepath.Join("note", file.Name()))
			if err != nil {
				fmt.Printf("读取笔记文件失败 %s: %v\n", file.Name(), err)
				continue
			}
			
			// 添加到笔记映射
			notes[title] = &Note{
				Title: title,
				Body:  string(content),
			}
			noteTitles = append(noteTitles, title)
		}
	}
	
	// 如果没有笔记，创建示例笔记
	if len(notes) == 0 {
		createSampleNotes()
	}
	
	fmt.Printf("成功加载 %d 个笔记\n", len(notes))
}

// 创建示例笔记
func createSampleNotes() {
	sampleNotes := map[string]string{
		"欢迎使用": "这是一个在线笔记应用的示例。您可以创建、编辑和删除笔记。\n\n笔记会自动保存到程序同文件夹下的note文件夹中。",
		"使用说明": "1. 点击'新建笔记'创建新笔记\n2. 点击笔记标题编辑现有笔记\n3. 使用删除按钮删除笔记\n4. 所有笔记会自动保存到note文件夹中",
	}
	
	for title, body := range sampleNotes {
		// 保存到文件
		err := os.WriteFile(filepath.Join("note", title+".txt"), []byte(body), 0644)
		if err != nil {
			fmt.Printf("创建示例笔记失败 %s: %v\n", title, err)
			continue
		}
		
		// 添加到内存
		notes[title] = &Note{
			Title: title,
			Body:  body,
		}
		noteTitles = append(noteTitles, title)
	}
	
	fmt.Println("已创建示例笔记")
}

// 保存笔记到文件
func saveNotes() {
	// 确保note目录存在
	os.Mkdir("note", 0755)
	
	// 保存所有笔记到文件
	for title, note := range notes {
		filename := filepath.Join("note", title+".txt")
		err := os.WriteFile(filename, []byte(note.Body), 0644)
		if err != nil {
			fmt.Printf("保存笔记失败 %s: %v\n", title, err)
		}
	}
	
	fmt.Printf("已保存 %d 个笔记到note文件夹\n", len(notes))
}

// 保存单个笔记到文件
func saveNoteToFile(title, body string) error {
	// 确保note目录存在
	os.Mkdir("note", 0755)
	
	filename := filepath.Join("note", title+".txt")
	return os.WriteFile(filename, []byte(body), 0644)
}

// 删除笔记文件
func deleteNoteFile(title string) error {
	filename := filepath.Join("note", title+".txt")
	return os.Remove(filename)
}
