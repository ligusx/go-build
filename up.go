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
			textarea.form-control { min-height: 500px; resize: vertical; font-family: monospace; line-height: 1.4; }
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
	
	// 显示上传表单
	tmpl := `
	<!DOCTYPE html>
	<html lang="zh-CN">
	<head>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<title>上传文件</title>
		<style>
			* { margin: 0; padding: 0; box-sizing: border-box; }
			body { font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; line-height: 1.6; color: #333; background-color: #f5f7fa; }
			.container { max-width: 1200px; margin: 0 auto; padding: 20px; }
			header { background: linear-gradient(135deg, #6a11cb 0%, #2575fc 100%); color: white; padding: 2rem 0; text-align: center; border-radius: 10px; margin-bottom: 2rem; box-shadow: 0 4px 6px rgba(0,0,0,0.1); }
			h1 { font-size: 2.5rem; margin-bottom: 0.5rem; }
			.card { background: white; border-radius: 10px; padding: 1.5rem; box-shadow: 0 2px 10px rgba(0,0,0,0.05); margin-bottom: 2rem; }
			.btn { display: inline-block; background: #2575fc; color: white; padding: 10px 20px; border-radius: 5px; text-decoration: none; font-weight: bold; transition: background 0.3s; }
			.btn:hover { background: #1a5fd8; }
			.btn-secondary { background: #6c757d; }
			.btn-secondary:hover { background: #5a6268; }
			.form-group { margin-bottom: 1rem; }
			.form-group label { display: block; margin-bottom: 0.5rem; font-weight: bold; }
			.form-control { width: 100%; padding: 10px; border: 1px solid #ddd; border-radius: 5px; font-size: 1rem; }
		</style>
	</head>
	<body>
		<div class="container">
			<header>
				<h1>上传文件</h1>
				<a href="/files" class="btn btn-secondary">返回文件列表</a>
			</header>
			
			<div class="card">
				<form action="/upload" method="post" enctype="multipart/form-data">
					<div class="form-group">
						<label for="file">选择文件:</label>
						<input type="file" name="file" id="file" class="form-control" required>
					</div>
					<button type="submit" class="btn">上传文件</button>
				</form>
			</div>
		</div>
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

// 文件列表处理器 - 美化版本
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
	for _, file := range files {
		if !file.IsDir() {
			fileInfo, _ := file.Info()
			size := formatFileSize(fileInfo.Size())
			fileListHTML += fmt.Sprintf(`
			<div class="file-item">
				<div class="file-info">
					<div class="file-icon">📄</div>
					<div class="file-details">
						<div class="file-name">%s</div>
						<div class="file-size">%s</div>
					</div>
				</div>
				<div class="file-actions">
					<a href="/download/%s" class="btn btn-download">下载</a>
					<a href="/delete-file/%s" class="btn btn-delete" onclick="return confirm('确定删除文件 %s 吗？')">删除</a>
				</div>
			</div>
			`, file.Name(), size, file.Name(), file.Name(), file.Name())
		}
	}
	
	if fileListHTML == "" {
		fileListHTML = `<div class="empty-state">暂无文件，请上传文件</div>`
	}
	
	// 显示消息
	alertHTML := ""
	if msg != "" {
		alertHTML = fmt.Sprintf(`<div class="alert alert-success">%s</div>`, msg)
	}
	
	tmpl := fmt.Sprintf(`
	<!DOCTYPE html>
	<html lang="zh-CN">
	<head>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<title>文件管理</title>
		<style>
			* { margin: 0; padding: 0; box-sizing: border-box; }
			body { font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; line-height: 1.6; color: #333; background-color: #f5f7fa; }
			.container { max-width: 1200px; margin: 0 auto; padding: 20px; }
			header { background: linear-gradient(135deg, #6a11cb 0%, #2575fc 100%); color: white; padding: 2rem 0; text-align: center; border-radius: 10px; margin-bottom: 2rem; box-shadow: 0 4px 6px rgba(0,0,0,0.1); }
			h1 { font-size: 2.5rem; margin-bottom: 0.5rem; }
			.btn { display: inline-block; background: #2575fc; color: white; padding: 10px 20px; border-radius: 5px; text-decoration: none; font-weight: bold; transition: background 0.3s; }
			.btn:hover { background: #1a5fd8; }
			.btn-secondary { background: #6c757d; }
			.btn-secondary:hover { background: #5a6268; }
			.alert { padding: 15px; border-radius: 5px; margin-bottom: 1rem; }
			.alert-success { background: #d4edda; color: #155724; border: 1px solid #c3e6cb; }
			
			.file-manager {
				max-width: 1000px;
				margin: 0 auto;
			}
			.upload-area {
				background: white;
				border: 2px dashed #2575fc;
				border-radius: 12px;
				padding: 2rem;
				text-align: center;
				margin-bottom: 2rem;
				transition: all 0.3s ease;
			}
			.upload-area:hover {
				background: #f8faff;
				border-color: #1a5fd8;
			}
			.upload-area.dragover {
				background: #e3f2fd;
				border-color: #1565c0;
			}
			.upload-icon {
				font-size: 3rem;
				margin-bottom: 1rem;
				color: #2575fc;
			}
			.upload-text {
				margin-bottom: 1.5rem;
				color: #666;
			}
			.upload-btn {
				display: inline-block;
				background: #2575fc;
				color: white;
				padding: 12px 24px;
				border-radius: 8px;
				text-decoration: none;
				font-weight: bold;
				cursor: pointer;
				transition: background 0.3s;
			}
			.upload-btn:hover {
				background: #1a5fd8;
			}
			#fileInput {
				display: none;
			}
			.progress-container {
				display: none;
				background: #f1f3f4;
				border-radius: 8px;
				margin: 1rem 0;
				overflow: hidden;
			}
			.progress-bar {
				height: 8px;
				background: linear-gradient(90deg, #2575fc, #6a11cb);
				width: 0%%;
				transition: width 0.3s ease;
			}
			.progress-text {
				text-align: center;
				font-size: 0.9rem;
				color: #666;
				margin-top: 0.5rem;
			}
			.file-list {
				background: white;
				border-radius: 12px;
				overflow: hidden;
				box-shadow: 0 2px 10px rgba(0,0,0,0.05);
			}
			.file-item {
				display: flex;
				justify-content: space-between;
				align-items: center;
				padding: 1rem 1.5rem;
				border-bottom: 1px solid #f0f0f0;
				transition: background 0.2s;
			}
			.file-item:hover {
				background: #f8f9fa;
			}
			.file-item:last-child {
				border-bottom: none;
			}
			.file-info {
				display: flex;
				align-items: center;
				gap: 1rem;
			}
			.file-icon {
				font-size: 2rem;
			}
			.file-details {
				display: flex;
				flex-direction: column;
			}
			.file-name {
				font-weight: 600;
				color: #333;
			}
			.file-size {
				font-size: 0.9rem;
				color: #666;
			}
			.file-actions {
				display: flex;
				gap: 0.5rem;
			}
			.btn-download {
				background: #28a745;
				color: white;
				padding: 8px 16px;
				border-radius: 6px;
				text-decoration: none;
				font-weight: 500;
				font-size: 0.9rem;
				transition: all 0.2s;
			}
			.btn-download:hover {
				background: #218838;
			}
			.btn-delete {
				background: #dc3545;
				color: white;
				padding: 8px 16px;
				border-radius: 6px;
				text-decoration: none;
				font-weight: 500;
				font-size: 0.9rem;
				transition: all 0.2s;
			}
			.btn-delete:hover {
				background: #c82333;
			}
			.empty-state {
				text-align: center;
				padding: 3rem;
				color: #666;
				font-size: 1.1rem;
			}
		</style>
	</head>
	<body>
		<div class="container">
			<header>
				<h1>文件管理</h1>
				<div>
					<a href="/" class="btn btn-secondary">返回主页</a>
				</div>
			</header>
			
			%s
			
			<div class="file-manager">
				<div class="upload-area" id="uploadArea">
					<div class="upload-icon">📤</div>
					<div class="upload-text">
						<h3>拖放文件到此处或点击上传</h3>
						<p>支持单个文件上传，最大32MB</p>
					</div>
					<div class="upload-btn" onclick="document.getElementById('fileInput').click()">
						选择文件
					</div>
					<input type="file" id="fileInput" onchange="handleFileSelect(this.files)">
					
					<div class="progress-container" id="progressContainer">
						<div class="progress-bar" id="progressBar"></div>
						<div class="progress-text" id="progressText">准备上传...</div>
					</div>
				</div>
				
				<div class="file-list">
					%s
				</div>
			</div>
		</div>

		<script>
			const uploadArea = document.getElementById('uploadArea');
			const fileInput = document.getElementById('fileInput');
			const progressContainer = document.getElementById('progressContainer');
			const progressBar = document.getElementById('progressBar');
			const progressText = document.getElementById('progressText');

			// 拖放功能
			uploadArea.addEventListener('dragover', (e) => {
				e.preventDefault();
				uploadArea.classList.add('dragover');
			});

			uploadArea.addEventListener('dragleave', () => {
				uploadArea.classList.remove('dragover');
			});

			uploadArea.addEventListener('drop', (e) => {
				e.preventDefault();
				uploadArea.classList.remove('dragover');
				if (e.dataTransfer.files.length) {
					handleFileSelect(e.dataTransfer.files);
				}
			});

			function handleFileSelect(files) {
				if (files.length === 0) return;
				
				const file = files[0];
				if (file.size > 32 * 1024 * 1024) {
					alert('文件大小不能超过32MB');
					return;
				}

				uploadFile(file);
			}

			function uploadFile(file) {
				const formData = new FormData();
				formData.append('file', file);

				const xhr = new XMLHttpRequest();
				
				xhr.upload.addEventListener('progress', (e) => {
					if (e.lengthComputable) {
						const percent = (e.loaded / e.total) * 100;
						progressBar.style.width = percent + '%%';
						progressText.textContent = '上传中: ' + Math.round(percent) + '%%';
					}
				});

				xhr.addEventListener('load', () => {
					if (xhr.status === 200) {
						progressBar.style.width = '100%%';
						progressText.textContent = '上传完成！';
						setTimeout(() => {
							window.location.href = '/files?msg=文件上传成功';
						}, 1000);
					} else {
						progressText.textContent = '上传失败: ' + xhr.responseText;
					}
				});

				xhr.addEventListener('error', () => {
					progressText.textContent = '上传出错，请重试';
				});

				progressContainer.style.display = 'block';
				progressText.textContent = '准备上传...';
				xhr.open('POST', '/upload');
				xhr.send(formData);
			}
		</script>
	</body>
	</html>
	`, alertHTML, fileListHTML)
	
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

// 笔记列表处理器 - 美化版本
func notesHandler(w http.ResponseWriter, r *http.Request) {
	// 生成笔记列表HTML
	noteListHTML := ""
	for _, title := range noteTitles {
		note := notes[title]
		preview := getNotePreview(note.Body)
		noteListHTML += fmt.Sprintf(`
		<div class="note-card">
			<div class="note-header">
				<h3 class="note-title">%s</h3>
				<div class="note-actions">
					<a href="/note/%s" class="btn btn-edit">编辑</a>
					<a href="/delete-note/%s" class="btn btn-delete" onclick="return confirm('确定删除笔记吗？')">删除</a>
				</div>
			</div>
			<div class="note-preview">%s</div>
			<div class="note-meta">
				<span class="note-length">%d 字符</span>
				<span class="note-date">最后编辑: 刚刚</span>
			</div>
		</div>
		`, title, title, title, preview, len(note.Body))
	}
	
	if noteListHTML == "" {
		noteListHTML = `
		<div class="empty-state">
			<div class="empty-icon">📝</div>
			<h3>暂无笔记</h3>
			<p>创建您的第一个笔记开始记录</p>
			<a href="/note/new" class="btn btn-primary">创建笔记</a>
		</div>`
	}
	
	tmpl := fmt.Sprintf(`
	<!DOCTYPE html>
	<html lang="zh-CN">
	<head>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<title>笔记管理</title>
		<style>
			* { margin: 0; padding: 0; box-sizing: border-box; }
			body { font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; line-height: 1.6; color: #333; background-color: #f5f7fa; }
			.container { max-width: 1200px; margin: 0 auto; padding: 20px; }
			
			.notes-container {
				max-width: 1000px;
				margin: 0 auto;
			}
			.notes-header {
				display: flex;
				justify-content: space-between;
				align-items: center;
				margin-bottom: 2rem;
			}
			.notes-header h1 {
				font-size: 2.5rem;
				color: #333;
			}
			.notes-grid {
				display: grid;
				gap: 1.5rem;
				grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
			}
			.note-card {
				background: white;
				border-radius: 12px;
				padding: 1.5rem;
				box-shadow: 0 2px 10px rgba(0,0,0,0.05);
				transition: all 0.3s ease;
				border: 1px solid #f0f0f0;
				display: flex;
				flex-direction: column;
				height: 200px;
			}
			.note-card:hover {
				transform: translateY(-4px);
				box-shadow: 0 8px 25px rgba(0,0,0,0.1);
				border-color: #2575fc;
			}
			.note-header {
				display: flex;
				justify-content: space-between;
				align-items: flex-start;
				margin-bottom: 1rem;
			}
			.note-title {
				font-size: 1.2rem;
				font-weight: 600;
				color: #333;
				margin: 0;
				flex: 1;
				overflow: hidden;
				text-overflow: ellipsis;
				white-space: nowrap;
			}
			.note-actions {
				display: flex;
				gap: 0.5rem;
				margin-left: 1rem;
			}
			.note-preview {
				flex: 1;
				color: #666;
				font-size: 0.95rem;
				line-height: 1.4;
				overflow: hidden;
				display: -webkit-box;
				-webkit-line-clamp: 3;
				-webkit-box-orient: vertical;
			}
			.note-meta {
				display: flex;
				justify-content: space-between;
				align-items: center;
				margin-top: 1rem;
				padding-top: 1rem;
				border-top: 1px solid #f0f0f0;
				font-size: 0.85rem;
				color: #999;
			}
			.btn {
				padding: 6px 12px;
				border-radius: 6px;
				text-decoration: none;
				font-weight: 500;
				font-size: 0.85rem;
				transition: all 0.2s;
				border: none;
				cursor: pointer;
			}
			.btn-primary {
				background: #2575fc;
				color: white;
			}
			.btn-primary:hover {
				background: #1a5fd8;
			}
			.btn-edit {
				background: #28a745;
				color: white;
			}
			.btn-edit:hover {
				background: #218838;
			}
			.btn-delete {
				background: #dc3545;
				color: white;
			}
			.btn-delete:hover {
				background: #c82333;
			}
			.empty-state {
				text-align: center;
				padding: 4rem 2rem;
				grid-column: 1 / -1;
			}
			.empty-icon {
				font-size: 4rem;
				margin-bottom: 1rem;
				opacity: 0.5;
			}
			.empty-state h3 {
				color: #666;
				margin-bottom: 0.5rem;
			}
			.empty-state p {
				color: #999;
				margin-bottom: 2rem;
			}
			.create-note-btn {
				background: linear-gradient(135deg, #6a11cb 0%, #2575fc 100%);
				color: white;
				padding: 12px 24px;
				border-radius: 8px;
				text-decoration: none;
				font-weight: bold;
				display: inline-flex;
				align-items: center;
				gap: 0.5rem;
				transition: transform 0.2s;
			}
			.create-note-btn:hover {
				transform: translateY(-2px);
				color: white;
			}
		</style>
	</head>
	<body>
		<div class="container">
			<div class="notes-header">
				<h1>我的笔记</h1>
				<a href="/note/new" class="create-note-btn">
					<span>+</span> 新建笔记
				</a>
			</div>
			
			<div class="notes-container">
				<div class="notes-grid">
					%s
				</div>
			</div>
		</div>
	</body>
	</html>
	`, noteListHTML)
	
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, tmpl)
}

// 笔记编辑器处理器
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
	
	// 显示笔记编辑器
	tmpl := `
	<!DOCTYPE html>
	<html lang="zh-CN">
	<head>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<title>笔记编辑器</title>
		<style>
			* { margin: 0; padding: 0; box-sizing: border-box; }
			body { font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; line-height: 1.6; color: #333; background-color: #f5f7fa; }
			.container { max-width: 1200px; margin: 0 auto; padding: 20px; }
			header { background: linear-gradient(135deg, #6a11cb 0%, #2575fc 100%); color: white; padding: 2rem 0; text-align: center; border-radius: 10px; margin-bottom: 2rem; box-shadow: 0 4px 6px rgba(0,0,0,0.1); }
			h1 { font-size: 2.5rem; margin-bottom: 0.5rem; }
			.card { background: white; border-radius: 10px; padding: 1.5rem; box-shadow: 0 2px 10px rgba(0,0,0,0.05); margin-bottom: 2rem; }
			.btn { display: inline-block; background: #2575fc; color: white; padding: 10px 20px; border-radius: 5px; text-decoration: none; font-weight: bold; transition: background 0.3s; }
			.btn:hover { background: #1a5fd8; }
			.btn-secondary { background: #6c757d; }
			.btn-secondary:hover { background: #5a6268; }
			.btn-success { background: #28a745; }
			.btn-success:hover { background: #218838; }
			.form-group { margin-bottom: 1rem; }
			.form-group label { display: block; margin-bottom: 0.5rem; font-weight: bold; }
			.form-control { width: 100%; padding: 10px; border: 1px solid #ddd; border-radius: 5px; font-size: 1rem; }
			textarea.form-control { min-height: 500px; resize: vertical; font-family: monospace; line-height: 1.4; }
		</style>
	</head>
	<body>
		<div class="container">
			<header>
				<h1>笔记编辑器</h1>
				<div>
					<a href="/notes" class="btn btn-secondary">返回笔记列表</a>
				</div>
			</header>
			
			<div class="card">
				<form action="/save-note" method="post">
					<div class="form-group">
						<label for="title">标题:</label>
						<input type="text" name="title" id="title" class="form-control" value="{{.Title}}" {{if not .IsNew}}readonly{{end}} required>
					</div>
					<div class="form-group">
						<label for="body">内容:</label>
						<textarea name="body" id="body" class="form-control" required>{{.Body}}</textarea>
					</div>
					<input type="hidden" name="isNew" value="{{.IsNew}}">
					<input type="hidden" name="oldTitle" value="{{.Title}}">
					<button type="submit" class="btn btn-success">保存笔记</button>
				</form>
			</div>
		</div>
	</body>
	</html>
	`
	
	data := struct {
		Title  string
		Body   string
		IsNew  bool
	}{
		Title: note.Title,
		Body:  note.Body,
		IsNew: isNew,
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

// 获取笔记预览内容
func getNotePreview(body string) string {
	// 移除换行和多余空格
	preview := strings.TrimSpace(body)
	preview = strings.ReplaceAll(preview, "\n", " ")
	
	// 限制长度
	if len(preview) > 120 {
		preview = preview[:120] + "..."
	}
	
	if preview == "" {
		preview = "暂无内容"
	}
	
	return template.HTMLEscapeString(preview)
}

// 文件大小格式化函数
func formatFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}
