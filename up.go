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
		<title>上传文件</title>
		<link rel="stylesheet" href="/static/style.css">
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
					<a href="/delete-file/%s" class="btn btn-secondary" onclick="return confirm('确定删除文件 %s 吗？')">删除</a>
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
		<link rel="stylesheet" href="/static/style.css">
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
				<a href="/delete-note/%s" class="btn btn-secondary" onclick="return confirm('确定删除笔记 %s 吗？')">删除</a>
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
		<link rel="stylesheet" href="/static/style.css">
	</head>
	<body>
		<div class="container">
			<header>
				<h1>笔记管理</h1>
				<div>
					<a href="/" class="btn btn-secondary">返回主页</a>
					<a href="/note/new" class="btn">新建笔记</a>
				</div>
			</header>
			
			<div class="card">
				<h2>笔记列表</h2>
				<ul class="note-list">
					%s
				</ul>
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
		<link rel="stylesheet" href="/static/style.css">
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
	
	// 如果是编辑现有笔记且标题改变，需要删除旧笔记
	if !isNew && oldTitle != title {
		delete(notes, oldTitle)
		// 从noteTitles中移除旧标题
		for i, t := range noteTitles {
			if t == oldTitle {
				noteTitles = append(noteTitles[:i], noteTitles[i+1:]...)
				break
			}
		}
	}
	
	// 保存笔记
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
	
	// 保存到文件
	saveNotes()
	
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
	
	// 删除笔记
	delete(notes, title)
	
	// 从noteTitles中移除
	for i, t := range noteTitles {
		if t == title {
			noteTitles = append(noteTitles[:i], noteTitles[i+1:]...)
			break
		}
	}
	
	// 保存到文件
	saveNotes()
	
	// 重定向到笔记列表
	http.Redirect(w, r, "/notes", http.StatusSeeOther)
}

// 加载笔记
func loadNotes() {
	// 这里简化处理，实际应用中应该从文件或数据库加载
	// 添加一些示例笔记
	notes["欢迎使用"] = &Note{
		Title: "欢迎使用",
		Body:  "这是一个在线笔记应用的示例。您可以创建、编辑和删除笔记。",
	}
	noteTitles = append(noteTitles, "欢迎使用")
	
	notes["使用说明"] = &Note{
		Title: "使用说明",
		Body:  "1. 点击'新建笔记'创建新笔记\n2. 点击笔记标题编辑现有笔记\n3. 使用删除按钮删除笔记",
	}
	noteTitles = append(noteTitles, "使用说明")
}

// 保存笔记到文件
func saveNotes() {
	// 这里简化处理，实际应用中应该保存到文件或数据库
	// 在这个示例中，我们只是将笔记保存在内存中
}
