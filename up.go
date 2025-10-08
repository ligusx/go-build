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
	
	// 显示上传表单
	tmpl := `
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>文件上传</title>
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.4.0/css/all.min.css">
    <style>
        :root {
            --primary-color: #4361ee;
            --primary-light: #4895ef;
            --secondary-color: #3f37c9;
            --success-color: #4cc9f0;
            --text-color: #333;
            --text-light: #6c757d;
            --bg-color: #f8f9fa;
            --card-bg: #ffffff;
            --border-color: #e0e0e0;
            --shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
            --shadow-hover: 0 8px 24px rgba(0, 0, 0, 0.12);
        }

        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
        }

        body {
            background-color: var(--bg-color);
            color: var(--text-color);
            line-height: 1.6;
            padding: 20px;
            min-height: 100vh;
            display: flex;
            justify-content: center;
            align-items: center;
            background: linear-gradient(135deg, #f5f7fa 0%, #c3cfe2 100%);
        }

        .container {
            width: 100%;
            max-width: 600px;
        }

        header {
            text-align: center;
            margin-bottom: 40px;
        }

        h1 {
            font-size: 2.5rem;
            margin-bottom: 15px;
            color: var(--primary-color);
            font-weight: 700;
        }

        .btn {
            display: inline-block;
            padding: 10px 24px;
            background-color: var(--primary-color);
            color: white;
            text-decoration: none;
            border-radius: 8px;
            font-weight: 600;
            transition: all 0.3s ease;
            border: none;
            cursor: pointer;
            font-size: 1rem;
            box-shadow: var(--shadow);
        }

        .btn:hover {
            background-color: var(--secondary-color);
            transform: translateY(-2px);
            box-shadow: var(--shadow-hover);
        }

        .btn-secondary {
            background-color: var(--text-light);
        }

        .btn-secondary:hover {
            background-color: #5a6268;
        }

        .card {
            background-color: var(--card-bg);
            border-radius: 16px;
            padding: 40px;
            box-shadow: var(--shadow);
            transition: all 0.3s ease;
        }

        .card:hover {
            box-shadow: var(--shadow-hover);
        }

        .form-group {
            margin-bottom: 30px;
        }

        label {
            display: block;
            margin-bottom: 10px;
            font-weight: 600;
            color: var(--text-color);
            font-size: 1.1rem;
        }

        .file-input-container {
            position: relative;
            margin-bottom: 20px;
        }

        .file-input {
            width: 100%;
            padding: 15px;
            border: 2px dashed var(--border-color);
            border-radius: 12px;
            background-color: #fafafa;
            transition: all 0.3s ease;
            cursor: pointer;
            text-align: center;
            color: var(--text-light);
        }

        .file-input:hover {
            border-color: var(--primary-light);
            background-color: #f0f5ff;
        }

        .file-input.highlight {
            border-color: var(--primary-color);
            background-color: #e8f0fe;
        }

        .file-input input[type="file"] {
            position: absolute;
            left: 0;
            top: 0;
            width: 100%;
            height: 100%;
            opacity: 0;
            cursor: pointer;
        }

        .file-info {
            margin-top: 15px;
            padding: 12px;
            background-color: #f8f9fa;
            border-radius: 8px;
            display: none;
        }

        .file-info.show {
            display: block;
            animation: fadeIn 0.3s ease;
        }

        .file-name {
            font-weight: 600;
            color: var(--primary-color);
        }

        .file-size {
            color: var(--text-light);
            font-size: 0.9rem;
        }

        .submit-btn {
            width: 100%;
            padding: 15px;
            background-color: var(--primary-color);
            color: white;
            border: none;
            border-radius: 12px;
            font-size: 1.1rem;
            font-weight: 600;
            cursor: pointer;
            transition: all 0.3s ease;
            box-shadow: var(--shadow);
        }

        .submit-btn:hover {
            background-color: var(--secondary-color);
            transform: translateY(-2px);
            box-shadow: var(--shadow-hover);
        }

        .submit-btn:active {
            transform: translateY(0);
        }

        .upload-icon {
            font-size: 3rem;
            color: var(--primary-light);
            margin-bottom: 20px;
            text-align: center;
        }

        .progress-container {
            margin-top: 20px;
            display: none;
        }

        .progress-container.show {
            display: block;
            animation: fadeIn 0.3s ease;
        }

        .progress-bar {
            height: 10px;
            background-color: #e0e0e0;
            border-radius: 5px;
            overflow: hidden;
            margin-bottom: 10px;
        }

        .progress {
            height: 100%;
            background-color: var(--primary-color);
            width: 0%;
            transition: width 0.3s ease;
        }

        .progress-text {
            text-align: center;
            font-size: 0.9rem;
            color: var(--text-light);
        }

        .upload-result {
            margin-top: 20px;
            padding: 15px;
            border-radius: 8px;
            text-align: center;
            display: none;
        }

        .upload-result.show {
            display: block;
            animation: fadeIn 0.5s ease;
        }

        .upload-result.success {
            background-color: #d4edda;
            color: #155724;
            border: 1px solid #c3e6cb;
        }

        .upload-result.error {
            background-color: #f8d7da;
            color: #721c24;
            border: 1px solid #f5c6cb;
        }

        .upload-result i {
            margin-right: 8px;
        }

        @keyframes fadeIn {
            from { opacity: 0; }
            to { opacity: 1; }
        }

        @media (max-width: 768px) {
            .card {
                padding: 30px 20px;
            }
            
            h1 {
                font-size: 2rem;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>上传文件</h1>
            <a href="/files" class="btn btn-secondary">返回文件列表</a>
        </header>
        
        <div class="card">
            <div class="upload-icon">
                <i class="fas fa-cloud-upload-alt"></i>
            </div>
            <form action="/upload" method="post" enctype="multipart/form-data" id="uploadForm">
                <div class="form-group">
                    <label for="file">选择文件</label>
                    <div class="file-input-container">
                        <div class="file-input" id="fileDropArea">
                            <span>点击选择文件或拖放文件到此处</span>
                            <input type="file" name="file" id="file" required>
                        </div>
                    </div>
                    <div class="file-info" id="fileInfo">
                        <div class="file-name" id="fileName"></div>
                        <div class="file-size" id="fileSize"></div>
                    </div>
                </div>
                
                <div class="progress-container" id="progressContainer">
                    <div class="progress-bar">
                        <div class="progress" id="progressBar"></div>
                    </div>
                    <div class="progress-text" id="progressText">0%</div>
                </div>
                
                <div class="upload-result" id="uploadResult"></div>
                
                <button type="submit" class="submit-btn" id="submitBtn">上传文件</button>
            </form>
        </div>
    </div>

    <script>
        document.addEventListener('DOMContentLoaded', function() {
            const fileInput = document.getElementById('file');
            const fileDropArea = document.getElementById('fileDropArea');
            const fileInfo = document.getElementById('fileInfo');
            const fileName = document.getElementById('fileName');
            const fileSize = document.getElementById('fileSize');
            const uploadForm = document.getElementById('uploadForm');
            const submitBtn = document.getElementById('submitBtn');
            const progressContainer = document.getElementById('progressContainer');
            const progressBar = document.getElementById('progressBar');
            const progressText = document.getElementById('progressText');
            const uploadResult = document.getElementById('uploadResult');
            
            // 处理文件选择
            fileInput.addEventListener('change', function() {
                if (this.files.length > 0) {
                    displayFileInfo(this.files[0]);
                }
            });
            
            // 拖放功能
            fileDropArea.addEventListener('dragover', function(e) {
                e.preventDefault();
                this.classList.add('highlight');
            });
            
            fileDropArea.addEventListener('dragleave', function() {
                this.classList.remove('highlight');
            });
            
            fileDropArea.addEventListener('drop', function(e) {
                e.preventDefault();
                this.classList.remove('highlight');
                
                if (e.dataTransfer.files.length > 0) {
                    fileInput.files = e.dataTransfer.files;
                    displayFileInfo(e.dataTransfer.files[0]);
                }
            });
            
            // 显示文件信息
            function displayFileInfo(file) {
                fileName.textContent = file.name;
                fileSize.textContent = formatFileSize(file.size);
                fileInfo.classList.add('show');
                
                // 重置上传状态
                resetUploadState();
            }
            
            // 格式化文件大小
            function formatFileSize(bytes) {
                if (bytes === 0) return '0 Bytes';
                
                const k = 1024;
                const sizes = ['Bytes', 'KB', 'MB', 'GB'];
                const i = Math.floor(Math.log(bytes) / Math.log(k));
                
                return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
            }
            
            // 重置上传状态
            function resetUploadState() {
                progressContainer.classList.remove('show');
                uploadResult.classList.remove('show');
                submitBtn.disabled = false;
                submitBtn.textContent = '上传文件';
                progressBar.style.width = '0%';
                progressText.textContent = '0%';
            }
            
            // 表单提交
            uploadForm.addEventListener('submit', function(e) {
                e.preventDefault();
                
                if (!fileInput.files.length) {
                    showUploadResult('请先选择一个文件', 'error');
                    return;
                }
                
                // 禁用提交按钮
                submitBtn.disabled = true;
                submitBtn.textContent = '上传中...';
                
                // 显示进度条
                progressContainer.classList.add('show');
                
                // 模拟上传过程
                simulateUpload(fileInput.files[0]);
            });
            
            // 模拟上传过程
            function simulateUpload(file) {
                let progress = 0;
                const interval = setInterval(() => {
                    progress += Math.random() * 10;
                    if (progress >= 100) {
                        progress = 100;
                        clearInterval(interval);
                        
                        // 上传完成
                        setTimeout(() => {
                            showUploadResult(`文件 "${file.name}" "上传文件成功!"`, 'success');
                            submitBtn.textContent = '上传完成';
                        }, 500);
                    }
                    
                    // 更新进度条
                    progressBar.style.width = progress + '%';
                    progressText.textContent = Math.round(progress) + '%';
                }, 200);
            }
            
            // 显示上传结果
            function showUploadResult(message, type) {
                uploadResult.textContent = message;
                uploadResult.className = 'upload-result show ' + type;
                
                if (type === 'success') {
                    uploadResult.innerHTML = '<i class="fas fa-check-circle"></i>' + message;
                } else {
                    uploadResult.innerHTML = '<i class="fas fa-exclamation-circle"></i>' + message;
                }
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

// 文件列表处理器
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
			fileListHTML += fmt.Sprintf(`
			<li>
				<span>%s</span>
				<div class="file-actions">
					<a href="/download/%s" class="btn">下载</a>
					<a href="/delete-file/%s" class="btn btn-danger" onclick="return confirm('确定删除文件 %s 吗？')">删除</a>
				</div>
			</li>
			`, file.Name(), file.Name(), file.Name(), file.Name())
		}
	}
	
	if fileListHTML == "" {
		fileListHTML = "<li>暂无文件</li>"
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
	</head>
	<body>
		<div class="container">
			<header>
				<h1>文件管理</h1>
				<div>
					<a href="/" class="btn btn-secondary">返回主页</a>
					<a href="/upload" class="btn">上传文件</a>
				</div>
			</header>
			
			%s
			
			<div class="card">
				<h2>文件列表</h2>
				<ul class="file-list">
					%s
				</ul>
			</div>
		</div>
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

// 笔记列表处理器
func notesHandler(w http.ResponseWriter, r *http.Request) {
	// 生成笔记列表HTML
	noteListHTML := ""
	for _, title := range noteTitles {
		noteListHTML += fmt.Sprintf(`
		<li>
			<span>%s</span>
			<div class="note-actions">
				<a href="/note/%s" class="btn">编辑</a>
				<a href="/delete-note/%s" class="btn btn-danger" onclick="return confirm('确定删除笔记 %s 吗？')">删除</a>
			</div>
		</li>
		`, title, title, title, title)
	}
	
	if noteListHTML == "" {
		noteListHTML = "<li>暂无笔记</li>"
	}
	
	tmpl := fmt.Sprintf(`
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>笔记管理</title>
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.4.0/css/all.min.css">
    <style>
        :root {
            --primary-color: #4361ee;
            --primary-light: #4895ef;
            --secondary-color: #3f37c9;
            --success-color: #4cc9f0;
            --text-color: #333;
            --text-light: #6c757d;
            --bg-color: #f8f9fa;
            --card-bg: #ffffff;
            --border-color: #e0e0e0;
            --shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
            --shadow-hover: 0 8px 24px rgba(0, 0, 0, 0.12);
        }

        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
        }

        body {
            background-color: var(--bg-color);
            color: var(--text-color);
            line-height: 1.6;
            padding: 20px;
            min-height: 100vh;
            background: linear-gradient(135deg, #f5f7fa 0%, #c3cfe2 100%);
        }

        .container {
            width: 100%;
            max-width: 1000px;
            margin: 0 auto;
        }

        header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 30px;
            padding-bottom: 15px;
            border-bottom: 1px solid var(--border-color);
        }

        h1 {
            font-size: 2.5rem;
            color: var(--primary-color);
            font-weight: 700;
        }

        h2 {
            font-size: 1.8rem;
            margin-bottom: 20px;
            color: var(--text-color);
            font-weight: 600;
        }

        .btn {
            display: inline-flex;
            align-items: center;
            gap: 8px;
            padding: 10px 20px;
            background-color: var(--primary-color);
            color: white;
            text-decoration: none;
            border-radius: 8px;
            font-weight: 600;
            transition: all 0.3s ease;
            border: none;
            cursor: pointer;
            font-size: 1rem;
            box-shadow: var(--shadow);
        }

        .btn:hover {
            background-color: var(--secondary-color);
            transform: translateY(-2px);
            box-shadow: var(--shadow-hover);
        }

        .btn-secondary {
            background-color: var(--text-light);
        }

        .btn-secondary:hover {
            background-color: #5a6268;
        }

        .header-actions {
            display: flex;
            gap: 15px;
        }

        .card {
            background-color: var(--card-bg);
            border-radius: 16px;
            padding: 30px;
            box-shadow: var(--shadow);
            transition: all 0.3s ease;
        }

        .card:hover {
            box-shadow: var(--shadow-hover);
        }

        .note-list {
            list-style: none;
        }

        .note-item {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 20px;
            border-bottom: 1px solid var(--border-color);
            transition: all 0.3s ease;
        }

        .note-item:hover {
            background-color: #f8f9fa;
            border-radius: 8px;
            transform: translateY(-2px);
            box-shadow: 0 4px 8px rgba(0, 0, 0, 0.05);
        }

        .note-item:last-child {
            border-bottom: none;
        }

        .note-content {
            flex: 1;
        }

        .note-title {
            font-size: 1.3rem;
            font-weight: 600;
            color: var(--text-color);
            margin-bottom: 8px;
        }

        .note-preview {
            color: var(--text-light);
            font-size: 0.95rem;
            line-height: 1.5;
            display: -webkit-box;
            -webkit-line-clamp: 2;
            -webkit-box-orient: vertical;
            overflow: hidden;
        }

        .note-meta {
            display: flex;
            gap: 15px;
            margin-top: 10px;
            font-size: 0.85rem;
            color: var(--text-light);
        }

        .note-actions {
            display: flex;
            gap: 10px;
        }

        .action-btn {
            display: flex;
            align-items: center;
            justify-content: center;
            width: 40px;
            height: 40px;
            border-radius: 50%;
            background-color: #f8f9fa;
            color: var(--text-light);
            text-decoration: none;
            transition: all 0.3s ease;
        }

        .action-btn:hover {
            background-color: var(--primary-color);
            color: white;
            transform: scale(1.1);
        }

        .empty-state {
            text-align: center;
            padding: 60px 20px;
            color: var(--text-light);
        }

        .empty-icon {
            font-size: 4rem;
            margin-bottom: 20px;
            color: #dee2e6;
        }

        .empty-state h3 {
            font-size: 1.5rem;
            margin-bottom: 10px;
            color: var(--text-light);
        }

        .empty-state p {
            margin-bottom: 25px;
        }

        .search-bar {
            display: flex;
            margin-bottom: 25px;
            gap: 10px;
        }

        .search-input {
            flex: 1;
            padding: 12px 15px;
            border: 1px solid var(--border-color);
            border-radius: 8px;
            font-size: 1rem;
            transition: all 0.3s ease;
        }

        .search-input:focus {
            outline: none;
            border-color: var(--primary-color);
            box-shadow: 0 0 0 3px rgba(67, 97, 238, 0.2);
        }

        .search-btn {
            padding: 12px 20px;
            background-color: var(--primary-color);
            color: white;
            border: none;
            border-radius: 8px;
            cursor: pointer;
            transition: all 0.3s ease;
        }

        .search-btn:hover {
            background-color: var(--secondary-color);
        }

        @keyframes fadeIn {
            from { opacity: 0; transform: translateY(10px); }
            to { opacity: 1; transform: translateY(0); }
        }

        .note-item {
            animation: fadeIn 0.4s ease forwards;
        }

        @media (max-width: 768px) {
            .card {
                padding: 20px 15px;
            }
            
            h1 {
                font-size: 2rem;
            }
            
            header {
                flex-direction: column;
                align-items: flex-start;
                gap: 15px;
            }
            
            .header-actions {
                width: 100%;
                justify-content: space-between;
            }
            
            .note-item {
                flex-direction: column;
                align-items: flex-start;
                gap: 15px;
            }
            
            .note-actions {
                align-self: flex-end;
            }
            
            .search-bar {
                flex-direction: column;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>笔记管理</h1>
            <div class="header-actions">
                <a href="/" class="btn btn-secondary">
                    <i class="fas fa-home"></i> 返回主页
                </a>
                <a href="/note/new" class="btn">
                    <i class="fas fa-plus"></i> 新建笔记
                </a>
            </div>
        </header>
        
        <div class="card">
            <h2>笔记列表</h2>
            
            <div class="search-bar">
                <input type="text" class="search-input" placeholder="搜索笔记...">
                <button class="search-btn">
                    <i class="fas fa-search"></i>
                </button>
            </div>
            
            <ul class="note-list">
                <!-- 示例笔记项目 -->
                <li class="note-item">
                    <div class="note-content">
                        <div class="note-title">我的第一篇笔记</div>
                        <div class="note-preview">这是笔记的预览内容，显示笔记的前面部分文字，让用户能够快速了解笔记的大致内容...</div>
                        <div class="note-meta">
                            <span><i class="far fa-calendar"></i> 2023-10-15</span>
                            <span><i class="far fa-file-alt"></i> 256 字</span>
                        </div>
                    </div>
                    <div class="note-actions">
                        <a href="/note/edit/1" class="action-btn" title="编辑笔记">
                            <i class="fas fa-edit"></i>
                        </a>
                        <a href="/note/delete/1" class="action-btn" title="删除笔记">
                            <i class="fas fa-trash"></i>
                        </a>
                    </div>
                </li>
                
                <li class="note-item">
                    <div class="note-content">
                        <div class="note-title">项目计划</div>
                        <div class="note-preview">项目目标和里程碑：1. 完成需求分析 2. 设计系统架构 3. 开发核心功能 4. 测试与优化...</div>
                        <div class="note-meta">
                            <span><i class="far fa-calendar"></i> 2023-10-10</span>
                            <span><i class="far fa-file-alt"></i> 512 字</span>
                        </div>
                    </div>
                    <div class="note-actions">
                        <a href="/note/edit/2" class="action-btn" title="编辑笔记">
                            <i class="fas fa-edit"></i>
                        </a>
                        <a href="/note/delete/2" class="action-btn" title="删除笔记">
                            <i class="fas fa-trash"></i>
                        </a>
                    </div>
                </li>
                
                <li class="note-item">
                    <div class="note-content">
                        <div class="note-title">学习笔记</div>
                        <div class="note-preview">关于现代Web开发技术的学习总结，包括HTML5、CSS3、JavaScript、React等前端技术...</div>
                        <div class="note-meta">
                            <span><i class="far fa-calendar"></i> 2023-10-05</span>
                            <span><i class="far fa-file-alt"></i> 1024 字</span>
                        </div>
                    </div>
                    <div class="note-actions">
                        <a href="/note/edit/3" class="action-btn" title="编辑笔记">
                            <i class="fas fa-edit"></i>
                        </a>
                        <a href="/note/delete/3" class="action-btn" title="删除笔记">
                            <i class="fas fa-trash"></i>
                        </a>
                    </div>
                </li>
            </ul>
            
            <!-- 空状态示例（当没有笔记时显示） -->
            <!--
            <div class="empty-state">
                <div class="empty-icon">
                    <i class="far fa-file-alt"></i>
                </div>
                <h3>暂无笔记</h3>
                <p>您还没有创建任何笔记，开始记录您的想法吧！</p>
                <a href="/note/new" class="btn">
                    <i class="fas fa-plus"></i> 创建第一份笔记
                </a>
            </div>
            -->
        </div>
    </div>

    <script>
        document.addEventListener('DOMContentLoaded', function() {
            const searchInput = document.querySelector('.search-input');
            const noteItems = document.querySelectorAll('.note-item');
            
            // 搜索功能
            searchInput.addEventListener('input', function() {
                const searchTerm = this.value.toLowerCase();
                
                noteItems.forEach(item => {
                    const title = item.querySelector('.note-title').textContent.toLowerCase();
                    const preview = item.querySelector('.note-preview').textContent.toLowerCase();
                    
                    if (title.includes(searchTerm) || preview.includes(searchTerm)) {
                        item.style.display = 'flex';
                    } else {
                        item.style.display = 'none';
                    }
                });
            });
            
            // 添加删除确认
            const deleteButtons = document.querySelectorAll('.action-btn[href*="delete"]');
            deleteButtons.forEach(button => {
                button.addEventListener('click', function(e) {
                    if (!confirm('确定要删除这个笔记吗？此操作不可撤销。')) {
                        e.preventDefault();
                    }
                });
            });
        });
    </script>
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
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.4.0/css/all.min.css">
    <style>
        :root {
            --primary-color: #4361ee;
            --primary-light: #4895ef;
            --secondary-color: #3f37c9;
            --success-color: #4cc9f0;
            --text-color: #333;
            --text-light: #6c757d;
            --bg-color: #f8f9fa;
            --card-bg: #ffffff;
            --border-color: #e0e0e0;
            --shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
            --shadow-hover: 0 8px 24px rgba(0, 0, 0, 0.12);
        }

        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
        }

        body {
            background-color: var(--bg-color);
            color: var(--text-color);
            line-height: 1.6;
            padding: 20px;
            min-height: 100vh;
            background: linear-gradient(135deg, #f5f7fa 0%, #c3cfe2 100%);
        }

        .container {
            width: 100%;
            max-width: 900px;
            margin: 0 auto;
        }

        header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 30px;
            padding-bottom: 15px;
            border-bottom: 1px solid var(--border-color);
        }

        h1 {
            font-size: 2.5rem;
            color: var(--primary-color);
            font-weight: 700;
        }

        .btn {
            display: inline-block;
            padding: 10px 24px;
            background-color: var(--primary-color);
            color: white;
            text-decoration: none;
            border-radius: 8px;
            font-weight: 600;
            transition: all 0.3s ease;
            border: none;
            cursor: pointer;
            font-size: 1rem;
            box-shadow: var(--shadow);
        }

        .btn:hover {
            background-color: var(--secondary-color);
            transform: translateY(-2px);
            box-shadow: var(--shadow-hover);
        }

        .btn-secondary {
            background-color: var(--text-light);
        }

        .btn-secondary:hover {
            background-color: #5a6268;
        }

        .btn-success {
            background-color: #28a745;
        }

        .btn-success:hover {
            background-color: #218838;
        }

        .card {
            background-color: var(--card-bg);
            border-radius: 16px;
            padding: 40px;
            box-shadow: var(--shadow);
            transition: all 0.3s ease;
        }

        .card:hover {
            box-shadow: var(--shadow-hover);
        }

        .form-group {
            margin-bottom: 30px;
        }

        label {
            display: block;
            margin-bottom: 10px;
            font-weight: 600;
            color: var(--text-color);
            font-size: 1.1rem;
        }

        .form-control {
            width: 100%;
            padding: 15px;
            border: 1px solid var(--border-color);
            border-radius: 8px;
            font-size: 1rem;
            transition: all 0.3s ease;
            background-color: #fff;
        }

        .form-control:focus {
            outline: none;
            border-color: var(--primary-color);
            box-shadow: 0 0 0 3px rgba(67, 97, 238, 0.2);
        }

        .form-control:read-only {
            background-color: #f8f9fa;
            color: var(--text-light);
            cursor: not-allowed;
        }

        textarea.form-control {
            min-height: 300px;
            resize: vertical;
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            line-height: 1.6;
        }

        .editor-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 20px;
        }

        .editor-icon {
            font-size: 2.5rem;
            color: var(--primary-light);
        }

        .char-count {
            text-align: right;
            color: var(--text-light);
            font-size: 0.9rem;
            margin-top: 5px;
        }

        .form-actions {
            display: flex;
            justify-content: flex-end;
            gap: 15px;
            margin-top: 30px;
        }

        .save-indicator {
            display: flex;
            align-items: center;
            color: var(--text-light);
            font-size: 0.9rem;
            margin-top: 10px;
            opacity: 0;
            transition: opacity 0.3s ease;
        }

        .save-indicator.show {
            opacity: 1;
        }

        .save-indicator i {
            margin-right: 5px;
        }

        @keyframes fadeIn {
            from { opacity: 0; }
            to { opacity: 1; }
        }

        @media (max-width: 768px) {
            .card {
                padding: 30px 20px;
            }
            
            h1 {
                font-size: 2rem;
            }
            
            header {
                flex-direction: column;
                align-items: flex-start;
                gap: 15px;
            }
            
            .form-actions {
                flex-direction: column;
            }
            
            .btn {
                width: 100%;
                text-align: center;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>笔记编辑器</h1>
            <div>
                <a href="/notes" class="btn btn-secondary">
                    <i class="fas fa-arrow-left"></i> 返回笔记列表
                </a>
            </div>
        </header>
        
        <div class="card">
            <div class="editor-header">
                <div class="editor-icon">
                    <i class="fas fa-edit"></i>
                </div>
            </div>
            
            <form action="/save-note" method="post" id="noteForm">
                <div class="form-group">
                    <label for="title">标题</label>
                    <input type="text" name="title" id="title" class="form-control" value="{{.Title}}" {{if not .IsNew}}readonly{{end}} required>
                </div>
                
                <div class="form-group">
                    <label for="body">内容</label>
                    <textarea name="body" id="body" class="form-control" required>{{.Body}}</textarea>
                    <div class="char-count">
                        <span id="charCount">0</span> 字符
                    </div>
                </div>
                
                <input type="hidden" name="isNew" value="{{.IsNew}}">
                <input type="hidden" name="oldTitle" value="{{.Title}}">
                
                <div class="form-actions">
                    <a href="/notes" class="btn btn-secondary">取消</a>
                    <button type="submit" class="btn btn-success">
                        <i class="fas fa-save"></i> 保存笔记
                    </button>
                </div>
                
                <div class="save-indicator" id="saveIndicator">
                    <i class="fas fa-check-circle"></i> 笔记已保存
                </div>
            </form>
        </div>
    </div>

    <script>
        document.addEventListener('DOMContentLoaded', function() {
            const noteForm = document.getElementById('noteForm');
            const bodyTextarea = document.getElementById('body');
            const charCount = document.getElementById('charCount');
            const saveIndicator = document.getElementById('saveIndicator');
            
            // 初始化字符计数
            updateCharCount();
            
            // 更新字符计数
            bodyTextarea.addEventListener('input', updateCharCount);
            
            function updateCharCount() {
                charCount.textContent = bodyTextarea.value.length;
            }
            
            // 表单提交
            noteForm.addEventListener('submit', function(e) {
                e.preventDefault();
                
                // 禁用提交按钮
                const submitBtn = this.querySelector('button[type="submit"]');
                const originalText = submitBtn.innerHTML;
                submitBtn.disabled = true;
                submitBtn.innerHTML = '<i class="fas fa-spinner fa-spin"></i> 保存中...';
                
                // 在实际应用中，这里会发送表单数据到服务器
                // 这里模拟一个保存请求
                setTimeout(function() {
                    // 显示保存成功提示
                    saveIndicator.classList.add('show');
                    
                    // 恢复按钮状态
                    submitBtn.disabled = false;
                    submitBtn.innerHTML = originalText;
                    
                    // 3秒后隐藏保存提示
                    setTimeout(function() {
                        saveIndicator.classList.remove('show');
                    }, 3000);
                }, 1000);
            });
            
            // 自动保存功能（可选）
            let autoSaveTimer;
            bodyTextarea.addEventListener('input', function() {
                clearTimeout(autoSaveTimer);
                
                // 在实际应用中，这里会发送自动保存请求
                // autoSaveTimer = setTimeout(function() {
                //     // 自动保存逻辑
                //     console.log('自动保存...');
                // }, 2000);
            });
        });
    </script>
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
