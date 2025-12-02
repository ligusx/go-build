package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// 配置项
const (
	UploadDir = "./up"
	NoteDir   = "./notes"
	Port      = ":8080"
)

// Note 结构体
type Note struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Updated int64  `json:"updated"`
}

// FileInfo 结构体
type FileInfo struct {
	Name    string `json:"name"`
	Size    string `json:"size"`
	ModTime string `json:"modTime"`
}

func main() {
	// 1. 初始化目录
	initDirs()

	// 2. 注册路由
	http.HandleFunc("/", handleIndex)           // 主页
	http.HandleFunc("/api/upload", handleUpload) // 上传接口
	http.HandleFunc("/api/files", handleListFiles) // 文件列表接口
	http.HandleFunc("/download/", handleDownload)  // 下载接口
	http.HandleFunc("/api/notes", handleNotes)     // 笔记查/改接口
	http.HandleFunc("/api/note/del", handleDelNote) // 删除笔记

	// 3. 启动服务
	fmt.Printf("🚀 服务已启动: http://localhost%s\n", Port)
	fmt.Printf("📂 文件存储目录: %s\n", UploadDir)
	fmt.Printf("📝 笔记存储目录: %s\n", NoteDir)
	
	if err := http.ListenAndServe(Port, nil); err != nil {
		log.Fatal("启动失败: ", err)
	}
}

// 初始化文件夹
func initDirs() {
	dirs := []string{UploadDir, NoteDir}
	for _, d := range dirs {
		if _, err := os.Stat(d); os.IsNotExist(err) {
			os.Mkdir(d, 0755)
		}
	}
}

// --- 处理器 ---

// 渲染主页
func handleIndex(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.New("index").Parse(htmlTemplate))
	tmpl.Execute(w, nil)
}

// 获取文件列表
func handleListFiles(w http.ResponseWriter, r *http.Request) {
	files, err := ioutil.ReadDir(UploadDir)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	var fileList []FileInfo
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		fileList = append(fileList, FileInfo{
			Name:    f.Name(),
			Size:    formatSize(f.Size()),
			ModTime: f.ModTime().Format("2006-01-02 15:04"),
		})
	}
	// 按时间倒序
	sort.Slice(fileList, func(i, j int) bool {
		return fileList[i].Name > fileList[j].Name 
	})

	jsonResponse(w, fileList)
}

// 上传文件
func handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	
	// 限制大小 (例如 1GB)
	r.Body = http.MaxBytesReader(w, r.Body, 1<<30) 
	if err := r.ParseMultipartForm(1 << 30); err != nil {
		http.Error(w, "文件太大", 400)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "无效文件", 400)
		return
	}
	defer file.Close()

	// 保存文件
	dstPath := filepath.Join(UploadDir, filepath.Base(header.Filename))
	dst, err := os.Create(dstPath)
	if err != nil {
		http.Error(w, "无法保存文件", 500)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, "保存失败", 500)
		return
	}

	jsonResponse(w, map[string]string{"status": "success"})
}

// 下载文件
func handleDownload(w http.ResponseWriter, r *http.Request) {
	fname := strings.TrimPrefix(r.URL.Path, "/download/")
	fpath := filepath.Join(UploadDir, fname)
	
	// 防止路径遍历攻击
	if !strings.HasPrefix(fpath, UploadDir) && !strings.Contains(fname, "..") {
		http.NotFound(w, r)
		return
	}
	
	http.ServeFile(w, r, fpath)
}

// 笔记处理 (GET: 列表, POST: 保存)
func handleNotes(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		// 读取所有笔记
		files, _ := ioutil.ReadDir(NoteDir)
		var notes []Note
		for _, f := range files {
			if strings.HasSuffix(f.Name(), ".json") {
				data, _ := ioutil.ReadFile(filepath.Join(NoteDir, f.Name()))
				var n Note
				json.Unmarshal(data, &n)
				notes = append(notes, n)
			}
		}
		// 按更新时间倒序
		sort.Slice(notes, func(i, j int) bool {
			return notes[i].Updated > notes[j].Updated
		})
		jsonResponse(w, notes)
	} else if r.Method == "POST" {
		// 保存笔记
		var n Note
		if err := json.NewDecoder(r.Body).Decode(&n); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		if n.ID == "" {
			n.ID = fmt.Sprintf("%d", time.Now().UnixNano())
		}
		n.Updated = time.Now().Unix()
		
		data, _ := json.Marshal(n)
		ioutil.WriteFile(filepath.Join(NoteDir, n.ID+".json"), data, 0644)
		jsonResponse(w, n)
	}
}

// 删除笔记
func handleDelNote(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id != "" {
		os.Remove(filepath.Join(NoteDir, id+".json"))
	}
	jsonResponse(w, map[string]string{"status": "deleted"})
}

// 辅助函数：JSON响应
func jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// 辅助函数：格式化大小
func formatSize(size int64) string {
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

// --- 前端资源 (HTML/CSS/JS) ---
// 为了美观，使用了 CDN 引入 Pico.css 和 Vue.js (轻量化实现逻辑)
const htmlTemplate = `
<!DOCTYPE html>
<html lang="zh-CN" data-theme="light">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>云端空间 | Cloud Space</title>
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/@picocss/pico@1/css/pico.min.css">
    <script src="https://unpkg.com/vue@3/dist/vue.global.js"></script>
    <style>
        :root {
            --primary: #4361ee;
            --background: #f8f9fa;
            --card-bg: #ffffff;
        }
        [data-theme="dark"] {
            --background: #11191f;
            --card-bg: #1e262e;
        }
        body { background-color: var(--background); transition: all 0.3s; }
        .container { max-width: 1000px; padding-top: 2rem; }
        
        /* 顶部导航 */
        .nav-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 2rem; }
        .nav-tabs { display: flex; gap: 1rem; }
        .nav-btn { cursor: pointer; padding: 0.5rem 1.5rem; border-radius: 50px; font-weight: bold; transition: 0.2s; border: none; background: transparent; color: var(--contrast); }
        .nav-btn.active { background: var(--primary); color: white; box-shadow: 0 4px 15px rgba(67, 97, 238, 0.3); }

        /* 卡片风格 */
        .card { background: var(--card-bg); padding: 1.5rem; border-radius: 16px; box-shadow: 0 10px 30px rgba(0,0,0,0.05); border: 1px solid rgba(0,0,0,0.05); }

        /* 文件列表 */
        .file-item { display: flex; justify-content: space-between; align-items: center; padding: 0.8rem 0; border-bottom: 1px solid var(--muted-border-color); }
        .file-item:last-child { border-bottom: none; }
        .file-info { display: flex; flex-direction: column; }
        .file-meta { font-size: 0.8rem; opacity: 0.7; }
        .action-btn { padding: 0.2rem 0.8rem; font-size: 0.9rem; text-decoration: none; border-radius: 8px; }

        /* 笔记布局 */
        .note-layout { display: grid; grid-template-columns: 30% 70%; gap: 1.5rem; height: 70vh; }
        .note-list { overflow-y: auto; border-right: 1px solid var(--muted-border-color); padding-right: 1rem; }
        .note-item { padding: 1rem; cursor: pointer; border-radius: 8px; margin-bottom: 0.5rem; transition: 0.2s; border: 1px solid transparent; }
        .note-item:hover { background: rgba(0,0,0,0.03); }
        .note-item.active { border-color: var(--primary); background: rgba(67, 97, 238, 0.05); }
        .note-editor { display: flex; flex-direction: column; height: 100%; }
        .note-title-input { font-size: 1.5rem; font-weight: bold; border: none !important; background: transparent !important; box-shadow: none !important; padding: 0; }
        .note-content-area { flex-grow: 1; resize: none; border: none !important; background: transparent !important; box-shadow: none !important; line-height: 1.6; }
        
        /* 拖拽上传区 */
        .upload-zone { border: 2px dashed var(--muted-border-color); border-radius: 12px; padding: 2rem; text-align: center; transition: 0.3s; cursor: pointer; margin-bottom: 1.5rem; }
        .upload-zone:hover, .upload-zone.dragover { border-color: var(--primary); background: rgba(67, 97, 238, 0.05); }

        /* 移动端适配 */
        @media (max-width: 768px) {
            .note-layout { grid-template-columns: 1fr; grid-template-rows: auto 1fr; height: auto; }
            .note-list { height: 200px; border-right: none; border-bottom: 1px solid var(--muted-border-color); }
        }
    </style>
</head>
<body>
    <div id="app" class="container">
        <div class="nav-header">
            <h2 style="margin:0;">☁️ Cloud Space</h2>
            <div class="nav-tabs">
                <button class="nav-btn" :class="{active: tab === 'files'}" @click="tab='files'">文件传输</button>
                <button class="nav-btn" :class="{active: tab === 'notes'}" @click="tab='notes'">在线笔记</button>
            </div>
        </div>

        <div v-if="tab === 'files'" class="card">
            <div class="upload-zone" 
                 @click="$refs.fileInput.click()" 
                 @dragover.prevent="dragover = true" 
                 @dragleave="dragover = false" 
                 @drop.prevent="handleDrop"
                 :class="{dragover: dragover}">
                <h4 style="margin-bottom:0.5rem;">点击或拖拽文件至此上传</h4>
                <small class="secondary">支持任意格式文件，自动保存到服务器 ./up 目录</small>
                <input type="file" ref="fileInput" @change="uploadFile" style="display: none;">
            </div>

            <article v-if="uploading">
                上传中... <progress></progress>
            </article>

            <h5 style="margin-bottom:1rem;">文件列表 ({{files.length}})</h5>
            <div v-if="files.length === 0" style="text-align:center; opacity:0.5; padding:2rem;">暂无文件</div>
            
            <div class="file-item" v-for="f in files" :key="f.name">
                <div class="file-info">
                    <strong>{{ f.name }}</strong>
                    <span class="file-meta">{{ f.size }} · {{ f.modTime }}</span>
                </div>
                <div>
                    <a :href="'/download/' + f.name" class="action-btn contrast" role="button">⬇ 下载</a>
                </div>
            </div>
        </div>

        <div v-if="tab === 'notes'" class="card">
            <div class="note-layout">
                <div class="note-list">
                    <button @click="createNote" class="outline" style="width:100%; margin-bottom:1rem;">+ 新建页面</button>
                    <div v-for="n in notes" :key="n.id" 
                         class="note-item" 
                         :class="{active: currentNote && currentNote.id === n.id}"
                         @click="selectNote(n)">
                        <div style="font-weight:bold;">{{ n.title || '无标题' }}</div>
                        <small style="opacity:0.6;">{{ formatDate(n.updated) }}</small>
                    </div>
                </div>

                <div class="note-editor" v-if="currentNote">
                    <div style="display:flex; justify-content:space-between; align-items:center;">
                        <input type="text" v-model="currentNote.title" placeholder="输入标题..." class="note-title-input" @input="debouncedSave">
                        <small style="cursor:pointer; color:red;" @click="deleteNote(currentNote.id)">删除</small>
                    </div>
                    <hr style="margin: 1rem 0;">
                    <textarea v-model="currentNote.content" placeholder="开始输入内容..." class="note-content-area" @input="debouncedSave"></textarea>
                    <small style="text-align:right; opacity:0.5;">{{ saveStatus }}</small>
                </div>
                <div v-else style="display:flex; align-items:center; justify-content:center; opacity:0.5;">
                    选择或新建一个笔记
                </div>
            </div>
        </div>
    </div>

    <script>
        const { createApp } = Vue;

        createApp({
            data() {
                return {
                    tab: 'files',
                    dragover: false,
                    uploading: false,
                    files: [],
                    notes: [],
                    currentNote: null,
                    saveTimer: null,
                    saveStatus: '已同步'
                }
            },
            mounted() {
                this.refreshFiles();
                this.refreshNotes();
            },
            methods: {
                // --- 文件逻辑 ---
                async refreshFiles() {
                    const res = await fetch('/api/files');
                    this.files = await res.json() || [];
                },
                handleDrop(e) {
                    this.dragover = false;
                    const files = e.dataTransfer.files;
                    if(files.length > 0) this.doUpload(files[0]);
                },
                uploadFile(e) {
                    if(e.target.files.length > 0) this.doUpload(e.target.files[0]);
                },
                async doUpload(file) {
                    this.uploading = true;
                    const formData = new FormData();
                    formData.append('file', file);
                    try {
                        await fetch('/api/upload', { method: 'POST', body: formData });
                        await this.refreshFiles();
                    } catch(e) { alert('上传失败'); }
                    this.uploading = false;
                },

                // --- 笔记逻辑 ---
                async refreshNotes() {
                    const res = await fetch('/api/notes');
                    this.notes = await res.json() || [];
                },
                createNote() {
                    const newNote = { id: '', title: '新笔记', content: '' };
                    this.currentNote = newNote;
                    this.saveNote(); // 立即保存以获取ID
                },
                selectNote(note) {
                    this.currentNote = note;
                },
                async saveNote() {
                    this.saveStatus = '保存中...';
                    const res = await fetch('/api/notes', {
                        method: 'POST',
                        body: JSON.stringify(this.currentNote)
                    });
                    const saved = await res.json();
                    
                    // 如果是新笔记，更新ID
                    if (!this.currentNote.id) {
                        this.currentNote.id = saved.id;
                        this.notes.unshift(this.currentNote);
                    } else {
                        // 更新列表中的显示
                        const idx = this.notes.findIndex(n => n.id === saved.id);
                        if(idx !== -1) {
                            this.notes[idx].title = saved.title;
                            this.notes[idx].updated = saved.updated;
                            // 重新排序
                            this.notes.sort((a,b) => b.updated - a.updated);
                        }
                    }
                    this.saveStatus = '已保存 ' + new Date().toLocaleTimeString();
                },
                debouncedSave() {
                    this.saveStatus = '输入中...';
                    clearTimeout(this.saveTimer);
                    this.saveTimer = setTimeout(this.saveNote, 1000); // 1秒后自动保存
                },
                async deleteNote(id) {
                    if(!confirm('确定删除吗？')) return;
                    await fetch('/api/note/del?id=' + id);
                    this.currentNote = null;
                    await this.refreshNotes();
                },
                formatDate(timestamp) {
                    return new Date(timestamp * 1000).toLocaleString();
                }
            }
        }).mount('#app');
    </script>
</body>
</html>
`
