package main

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
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
	
	// ==================== 显示上传表单 - 美化版 ====================
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
				padding: 10px 20px; 
				border-radius: 5px; 
				text-decoration: none; 
				font-weight: bold; 
				transition: all 0.3s ease;
				border: none;
				cursor: pointer;
				font-size: 1rem;
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
				align-items: center;
				justify-content: center;
				padding: 2rem;
				border: 2px dashed #dee2e6;
				border-radius: 8px;
				background: #f8f9fa;
				transition: all 0.3s ease;
				text-align: center;
				color: #6c757d;
				font-weight: 500;
			}
			.file-input-custom:hover {
				border-color: #2575fc;
				background: #e7f1ff;
				color: #2575fc;
			}
			.file-input-custom.has-file {
				border-color: #28a745;
				background: #f0fff4;
				color: #28a745;
			}
			.upload-icon {
				font-size: 2rem;
				margin-bottom: 0.5rem;
			}
			.form-actions {
				display: flex;
				justify-content: flex-end;
				gap: 10px;
				margin-top: 1.5rem;
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
									<div>点击选择文件或拖拽文件到这里</div>
									<div style="font-size: 0.9rem; margin-top: 0.5rem;">支持所有类型文件，最大32MB</div>
								</div>
							</div>
						</div>
					</div>
					<div class="form-actions">
						<button type="submit" class="btn">上传文件</button>
					</div>
				</form>
			</div>
		</div>

		<script>
			function updateFileName(input) {
				const customInput = document.getElementById('fileInputCustom');
				if (input.files.length > 0) {
					const fileName = input.files[0].name;
					customInput.innerHTML = '<div><div class="upload-icon">✅</div><div>已选择文件: <strong>' + fileName + '</strong></div><div style="font-size: 0.9rem; margin-top: 0.5rem;">点击重新选择</div></div>';
					customInput.classList.add('has-file');
				} else {
					customInput.innerHTML = '<div><div class="upload-icon">📁</div><div>点击选择文件或拖拽文件到这里</div><div style="font-size: 0.9rem; margin-top: 0.5rem;">支持所有类型文件，最大32MB</div></div>';
					customInput.classList.remove('has-file');
				}
			}

			// 拖拽功能
			const fileInput = document.getElementById('file');
			const customInput = document.getElementById('fileInputCustom');
			
			customInput.addEventListener('dragover', (e) => {
				e.preventDefault();
				customInput.style.borderColor = '#2575fc';
				customInput.style.background = '#e7f1ff';
			});
			
			customInput.addEventListener('dragleave', (e) => {
				e.preventDefault();
				if (!customInput.classList.contains('has-file')) {
					customInput.style.borderColor = '#dee2e6';
					customInput.style.background = '#f8f9fa';
				}
			});
			
			customInput.addEventListener('drop', (e) => {
				e.preventDefault();
				const files = e.dataTransfer.files;
				if (files.length > 0) {
					fileInput.files = files;
					updateFileName(fileInput);
				}
			});
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

// ==================== 文件列表处理器 - 美化版 ====================
func filesHandler(w http.ResponseWriter, r *http.Request) {
	// 获取消息参数
	msg := r.URL.Query().Get("msg")
	
	// 读取up目录下的文件
	files, err := os.ReadDir("up")
	if err != nil {
		http.Error(w, "无法读取文件目录", http.StatusInternalServerError)
		return
	}
	
	// 生成文件列表HTML
	fileListHTML := ""
	fileCount := 0
	for _, file := range files {
		if !file.IsDir() {
			fileCount++
			fileInfo, _ := file.Info()
			fileSize := formatFileSize(fileInfo.Size())
			fileIcon := getFileIcon(file.Name())
			
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
			`, fileIcon, file.Name(), fileSize, file.Name(), file.Name(), template.HTMLEscapeString(file.Name()))
		}
	}
	
	if fileListHTML == "" {
		fileListHTML = `
		<li class="empty-state">
			<div class="empty-icon">📁</div>
			<div class="empty-text">暂无文件</div>
			<div class="empty-subtext">上传您的第一个文件开始使用</div>
			<a href="/upload" class="btn">上传文件</a>
		</li>
		`
	}
	
	// 显示消息
	alertHTML := ""
	if msg != "" {
		alertHTML = fmt.Sprintf(`<div class="alert alert-success">%s</div>`, template.HTMLEscapeString(msg))
	}
	
	tmpl := fmt.Sprintf(`
	<!DOCTYPE html>
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
			.btn-download {
				background: #17a2b8;
			}
			.btn-download:hover {
				background: #138496;
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
			
			%s
			
			<div class="stats">
				<div class="stat-card">
					<div class="stat-number">%d</div>
					<div class="stat-label">文件数量</div>
				</div>
			</div>
			
			<div class="card">
				<h2>文件列表</h2>
				<ul class="file-list">
					%s
				</ul>
			</div>
		</div>
	</body>
	</html>
	`, alertHTML, fileCount, fileListHTML)
	
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, tmpl)
}

// 删除文件处理器
func deleteFileHandler(w http.ResponseWriter, r *http.Request) {
	filename := strings.TrimPrefix(r.URL.Path, "/delete-file/")
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
	
	// 删除文件
	err := os.Remove(filepath)
	if err != nil {
		http.Error(w, "删除文件失败: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	// 重定向到文件列表页，显示成功消息
	http.Redirect(w, r, "/files?msg=文件删除成功", http.StatusSeeOther)
}

// ==================== 笔记列表处理器 - 美化版 ====================
func notesHandler(w http.ResponseWriter, r *http.Request) {
	// 生成笔记列表HTML
	noteListHTML := ""
	for _, title := range noteTitles {
		note := notes[title]
		preview := getNotePreview(note.Body)
		
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
		`, template.HTMLEscapeString(title), template.HTMLEscapeString(preview), template.HTMLEscapeString(title), template.HTMLEscapeString(title), template.HTMLEscapeString(title))
	}
	
	if noteListHTML == "" {
		noteListHTML = `
		<li class="empty-state">
			<div class="empty-icon">📝</div>
			<div class="empty-text">暂无笔记</div>
			<div class="empty-subtext">创建您的第一个笔记开始记录</div>
			<a href="/note/new" class="btn">新建笔记</a>
		</li>
		`
	}
	
	tmpl := fmt.Sprintf(`
	<!DOCTYPE html>
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
					<div class="stat-number">%d</div>
					<div class="stat-label">笔记数量</div>
				</div>
			</div>
			
			<div class="card">
				<h2>笔记列表</h2>
				<ul class="note-list">
					%s
				</ul>
			</div>
		</div>
	</body>
	</html>
	`, len(noteTitles), noteListHTML)
	
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, tmpl)
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