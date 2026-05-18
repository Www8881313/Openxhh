package main

import (
	"bufio"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

const defaultAddr = "127.0.0.1:29173"

var indexTemplate = template.Must(template.New("index").Parse(indexHTML))

type authStore struct {
	Salt string `json:"salt"`
	Hash string `json:"hash"`
}

type serverState struct {
	rootDir     string
	authPath    string
	listenAddr  string
	sessions    map[string]time.Time
	sessionsMu  sync.Mutex
	processMu   sync.Mutex
	process     *exec.Cmd
	processDone chan error
	startedAt   time.Time
	stdoutPath  string
}

type logFile struct {
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	ModTime string `json:"modTime"`
}

func main() {
	addr := flag.String("addr", defaultAddr, "web ui listen address")
	root := flag.String("root", ".", "xhhRobot working directory")
	flag.Parse()

	rootDir, err := filepath.Abs(*root)
	if err != nil {
		log.Fatal(err)
	}

	state := &serverState{
		rootDir:  rootDir,
		authPath: filepath.Join(rootDir, "webui_auth.json"),
		sessions: map[string]time.Time{},
	}

	password, created, err := ensureAuth(state.authPath)
	if err != nil {
		log.Fatal(err)
	}
	if created {
		fmt.Println("xhhRobot Web UI 已生成随机强密码")
		fmt.Println("登录密码:", password)
		fmt.Println("请保存该密码；如需重置，停止 Web UI 后删除 webui_auth.json 再启动。")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", state.handleIndex)
	mux.HandleFunc("/login", state.handleLogin)
	mux.HandleFunc("/logout", state.requireAuth(state.handleLogout))
	mux.HandleFunc("/api/status", state.requireAuth(state.handleStatus))
	mux.HandleFunc("/api/start", state.requireAuth(state.handleStart))
	mux.HandleFunc("/api/stop", state.requireAuth(state.handleStop))
	mux.HandleFunc("/api/logs", state.requireAuth(state.handleLogs))
	mux.HandleFunc("/api/logs/read", state.requireAuth(state.handleReadLog))

	listener, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("监听 %s 失败: %v", *addr, err)
	}

	state.listenAddr = listener.Addr().String()
	fmt.Printf("xhhRobot Web UI: http://%s\n", state.listenAddr)
	fmt.Printf("工作目录: %s\n", rootDir)
	log.Fatal(http.Serve(listener, withSecurityHeaders(mux)))
}

func ensureAuth(path string) (string, bool, error) {
	if _, err := os.Stat(path); err == nil {
		return "", false, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", false, err
	}

	password, err := randomPassword(32)
	if err != nil {
		return "", false, err
	}
	saltBytes := make([]byte, 32)
	if _, err := rand.Read(saltBytes); err != nil {
		return "", false, err
	}
	salt := hex.EncodeToString(saltBytes)
	store := authStore{
		Salt: salt,
		Hash: hashPassword(password, salt),
	}
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return "", false, err
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return "", false, err
	}
	return password, true, nil
}

func randomPassword(length int) (string, error) {
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func hashPassword(password, salt string) string {
	digest := sha256.Sum256([]byte(salt + ":" + password))
	for i := 0; i < 120000; i++ {
		h := sha256.New()
		_, _ = h.Write([]byte(salt))
		_, _ = h.Write([]byte(password))
		_, _ = h.Write(digest[:])
		copy(digest[:], h.Sum(nil))
	}
	return hex.EncodeToString(digest[:])
}

func (s *serverState) validPassword(password string) bool {
	data, err := os.ReadFile(s.authPath)
	if err != nil {
		return false
	}
	var store authStore
	if err := json.Unmarshal(data, &store); err != nil {
		return false
	}
	actual := hashPassword(password, store.Salt)
	return subtle.ConstantTimeCompare([]byte(actual), []byte(store.Hash)) == 1
}

func (s *serverState) createSession() (string, error) {
	token, err := randomPassword(48)
	if err != nil {
		return "", err
	}
	s.sessionsMu.Lock()
	defer s.sessionsMu.Unlock()
	s.sessions[token] = time.Now().Add(24 * time.Hour)
	return token, nil
}

func (s *serverState) validSession(r *http.Request) bool {
	cookie, err := r.Cookie("xhh_webui_session")
	if err != nil || cookie.Value == "" {
		return false
	}
	s.sessionsMu.Lock()
	defer s.sessionsMu.Unlock()
	expiresAt, ok := s.sessions[cookie.Value]
	if !ok || time.Now().After(expiresAt) {
		delete(s.sessions, cookie.Value)
		return false
	}
	return true
}

func (s *serverState) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = indexTemplate.Execute(w, map[string]any{"Authed": s.validSession(r)})
}

func (s *serverState) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var payload struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "请求格式错误"})
		return
	}
	if !s.validPassword(payload.Password) {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "密码错误"})
		return
	}
	token, err := s.createSession()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "无法创建会话"})
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "xhh_webui_session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(24 * time.Hour),
	})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *serverState) handleLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("xhh_webui_session"); err == nil {
		s.sessionsMu.Lock()
		delete(s.sessions, cookie.Value)
		s.sessionsMu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{Name: "xhh_webui_session", Path: "/", MaxAge: -1})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *serverState) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.validSession(r) {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "未登录"})
			return
		}
		next(w, r)
	}
}

func (s *serverState) handleStatus(w http.ResponseWriter, r *http.Request) {
	running, startedAt, stdoutPath := s.processStatus()
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":         true,
		"running":    running,
		"startedAt":  formatTime(startedAt),
		"rootDir":    s.rootDir,
		"listenPort": s.listenAddr,
		"stdoutLog":  baseName(stdoutPath),
	})
}

func (s *serverState) handleStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.processMu.Lock()
	defer s.processMu.Unlock()
	if s.process != nil && s.process.Process != nil {
		writeJSON(w, http.StatusConflict, map[string]any{"ok": false, "error": "机器人已在 Web UI 中运行"})
		return
	}
	if err := os.MkdirAll(filepath.Join(s.rootDir, "log"), 0775); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	stdoutPath := filepath.Join(s.rootDir, "log", "webui-robot-"+time.Now().Format("2006-01-02_15_04_05")+".log")
	stdoutFile, err := os.Create(stdoutPath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	cmd := buildRobotCommand(s.rootDir)
	cmd.Stdout = io.MultiWriter(stdoutFile)
	cmd.Stderr = io.MultiWriter(stdoutFile)
	if err := cmd.Start(); err != nil {
		_ = stdoutFile.Close()
		writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	s.process = cmd
	s.processDone = make(chan error, 1)
	s.startedAt = time.Now()
	s.stdoutPath = stdoutPath
	go func() {
		err := cmd.Wait()
		_ = stdoutFile.Close()
		s.processDone <- err
		s.processMu.Lock()
		if s.process == cmd {
			s.process = nil
			s.startedAt = time.Time{}
		}
		s.processMu.Unlock()
	}()

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func buildRobotCommand(rootDir string) *exec.Cmd {
	if custom := strings.TrimSpace(os.Getenv("XHHROBOT_COMMAND")); custom != "" {
		return shellCommand(custom, rootDir)
	}
	if runtime.GOOS == "windows" {
		if fileExists(filepath.Join(rootDir, "xhhRobot.exe")) {
			cmd := exec.Command(filepath.Join(rootDir, "xhhRobot.exe"), "-mode", "start")
			cmd.Dir = rootDir
			return cmd
		}
	}
	if fileExists(filepath.Join(rootDir, "xhhRobot")) {
		cmd := exec.Command(filepath.Join(rootDir, "xhhRobot"), "-mode", "start")
		cmd.Dir = rootDir
		return cmd
	}
	cmd := exec.Command("go", "run", ".", "-mode", "start")
	cmd.Dir = rootDir
	return cmd
}

func shellCommand(command, dir string) *exec.Cmd {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}
	cmd.Dir = dir
	return cmd
}

func (s *serverState) handleStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.processMu.Lock()
	cmd := s.process
	done := s.processDone
	s.processMu.Unlock()
	if cmd == nil || cmd.Process == nil || done == nil {
		writeJSON(w, http.StatusConflict, map[string]any{"ok": false, "error": "机器人未由 Web UI 启动"})
		return
	}
	if err := stopProcess(cmd); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		if err := cmd.Process.Kill(); err != nil && !strings.Contains(err.Error(), "process already finished") {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func stopProcess(cmd *exec.Cmd) error {
	if runtime.GOOS == "windows" {
		return exec.Command("taskkill", "/T", "/F", "/PID", fmt.Sprint(cmd.Process.Pid)).Run()
	}
	return cmd.Process.Signal(os.Interrupt)
}

func (s *serverState) processStatus() (bool, time.Time, string) {
	s.processMu.Lock()
	defer s.processMu.Unlock()
	return s.process != nil && s.process.Process != nil, s.startedAt, s.stdoutPath
}

func (s *serverState) handleLogs(w http.ResponseWriter, r *http.Request) {
	files, err := listLogFiles(filepath.Join(s.rootDir, "log"))
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "files": []logFile{}})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "files": files})
}

func listLogFiles(logDir string) ([]logFile, error) {
	entries, err := os.ReadDir(logDir)
	if err != nil {
		return nil, err
	}
	files := make([]logFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".log") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, logFile{
			Name:    entry.Name(),
			Size:    info.Size(),
			ModTime: info.ModTime().Format("2006-01-02 15:04:05"),
		})
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].ModTime > files[j].ModTime
	})
	return files, nil
}

func (s *serverState) handleReadLog(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("file")
	if name == "" || strings.Contains(name, "/") || strings.Contains(name, "\\") || strings.Contains(name, "..") {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "日志文件名无效"})
		return
	}
	path := filepath.Join(s.rootDir, "log", name)
	content, err := tailFile(path, 800)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "content": content})
}

func tailFile(path string, maxLines int) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) > maxLines {
			copy(lines, lines[1:])
			lines = lines[:maxLines]
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return strings.Join(lines, "\n"), nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func withSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline'; connect-src 'self'; img-src 'self' data:")
		next.ServeHTTP(w, r)
	})
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02 15:04:05")
}

func baseName(path string) string {
	if path == "" {
		return ""
	}
	return filepath.Base(path)
}

const indexHTML = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>xhhRobot 控制台</title>
  <style>
    :root {
      color-scheme: dark;
      --bg: #080b10;
      --panel: rgba(17, 23, 33, .84);
      --panel-strong: rgba(23, 31, 44, .96);
      --line: rgba(141, 170, 202, .18);
      --text: #e8eef8;
      --muted: #7f8da3;
      --cyan: #52e0ff;
      --green: #7dffb2;
      --amber: #ffcf67;
      --red: #ff6b7a;
      --shadow: 0 24px 80px rgba(0, 0, 0, .42);
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      min-height: 100vh;
      font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, "Liberation Mono", monospace;
      color: var(--text);
      background:
        radial-gradient(circle at 12% 18%, rgba(82, 224, 255, .16), transparent 34rem),
        radial-gradient(circle at 78% 6%, rgba(125, 255, 178, .10), transparent 30rem),
        linear-gradient(135deg, #05070b 0%, #0b111b 48%, #071018 100%);
      overflow-x: hidden;
    }
    body::before {
      content: "";
      position: fixed;
      inset: 0;
      pointer-events: none;
      opacity: .24;
      background-image: linear-gradient(rgba(255,255,255,.045) 1px, transparent 1px), linear-gradient(90deg, rgba(255,255,255,.035) 1px, transparent 1px);
      background-size: 44px 44px;
      mask-image: linear-gradient(to bottom, black, transparent 82%);
    }
    .shell { width: min(1180px, calc(100vw - 40px)); margin: 0 auto; padding: 34px 0 40px; position: relative; }
    .topbar { display: flex; align-items: center; justify-content: space-between; gap: 18px; margin-bottom: 24px; }
    .brand { display: flex; align-items: center; gap: 14px; }
    .mark { width: 42px; height: 42px; border: 1px solid rgba(82,224,255,.45); background: linear-gradient(145deg, rgba(82,224,255,.2), rgba(125,255,178,.08)); box-shadow: 0 0 28px rgba(82,224,255,.2); transform: rotate(45deg); }
    .brand h1 { font-size: clamp(24px, 3.4vw, 44px); letter-spacing: -.06em; line-height: .95; margin: 0; }
    .brand p { margin: 6px 0 0; color: var(--muted); font-size: 13px; }
    .status-pill { display: inline-flex; align-items: center; gap: 8px; padding: 10px 13px; border: 1px solid var(--line); border-radius: 999px; background: rgba(10,14,21,.62); color: var(--muted); }
    .dot { width: 8px; height: 8px; border-radius: 50%; background: var(--red); box-shadow: 0 0 14px currentColor; color: var(--red); }
    .dot.on { background: var(--green); color: var(--green); }
    .login { min-height: 72vh; display: grid; place-items: center; }
    .login-card { width: min(440px, 100%); padding: 30px; border: 1px solid var(--line); border-radius: 26px; background: linear-gradient(180deg, rgba(23,31,44,.92), rgba(10,14,21,.9)); box-shadow: var(--shadow); }
    .login-card h2 { margin: 0 0 8px; font-size: 28px; letter-spacing: -.04em; }
    .login-card p { margin: 0 0 22px; color: var(--muted); line-height: 1.65; }
    input, select, button { font: inherit; }
    .input { width: 100%; border: 1px solid var(--line); background: rgba(2,5,9,.62); color: var(--text); border-radius: 16px; padding: 14px 15px; outline: none; }
    .input:focus, select:focus { border-color: rgba(82,224,255,.65); box-shadow: 0 0 0 4px rgba(82,224,255,.09); }
    .button-row { display: flex; gap: 10px; flex-wrap: wrap; }
    button { border: 0; cursor: pointer; color: #041016; background: var(--cyan); padding: 12px 16px; border-radius: 14px; font-weight: 800; transition: transform .18s ease, filter .18s ease, background .18s ease; }
    button:hover { transform: translateY(-1px); filter: brightness(1.08); }
    button.secondary { color: var(--text); background: rgba(255,255,255,.08); border: 1px solid var(--line); }
    button.danger { color: #220206; background: var(--red); }
    button:disabled { opacity: .45; cursor: not-allowed; transform: none; }
    .grid { display: grid; grid-template-columns: 360px 1fr; gap: 18px; }
    .panel { border: 1px solid var(--line); border-radius: 26px; background: var(--panel); box-shadow: var(--shadow); backdrop-filter: blur(18px); }
    .panel-pad { padding: 22px; }
    .section-title { margin: 0 0 16px; color: var(--muted); font-size: 12px; letter-spacing: .18em; text-transform: uppercase; }
    .metric { display: grid; gap: 6px; padding: 16px 0; border-bottom: 1px solid var(--line); }
    .metric:last-child { border-bottom: 0; }
    .metric span { color: var(--muted); font-size: 12px; }
    .metric strong { font-size: 16px; word-break: break-all; }
    .actions { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; margin-top: 18px; }
    .log-head { display: flex; align-items: center; justify-content: space-between; gap: 14px; padding: 18px; border-bottom: 1px solid var(--line); }
    select { min-width: 280px; max-width: 100%; color: var(--text); background: rgba(2,5,9,.72); border: 1px solid var(--line); border-radius: 14px; padding: 11px 12px; outline: none; }
    .terminal { height: min(66vh, 680px); overflow: auto; padding: 20px; background: rgba(0,0,0,.42); border-radius: 0 0 26px 26px; }
    pre { margin: 0; white-space: pre-wrap; word-break: break-word; line-height: 1.55; color: #c7d6e8; font-size: 13px; }
    .empty { color: var(--muted); display: grid; place-items: center; min-height: 320px; text-align: center; line-height: 1.7; }
    .toast { min-height: 22px; margin-top: 14px; color: var(--amber); font-size: 13px; }
    .hidden { display: none !important; }
    @media (max-width: 860px) {
      .grid { grid-template-columns: 1fr; }
      .topbar { align-items: flex-start; flex-direction: column; }
      .log-head { align-items: stretch; flex-direction: column; }
      select { width: 100%; min-width: 0; }
    }
  </style>
</head>
<body data-authed="{{.Authed}}">
  <main class="shell">
    <div class="topbar">
      <div class="brand">
        <div class="mark"></div>
        <div>
          <h1>xhhRobot</h1>
          <p>本地运维控制台 · 默认监听 127.0.0.1:29173</p>
        </div>
      </div>
      <div id="topStatus" class="status-pill"><span class="dot"></span><span>未连接</span></div>
    </div>

    <section id="loginView" class="login hidden">
      <form id="loginForm" class="login-card">
        <h2>进入控制台</h2>
        <p>密码在 Web UI 首次启动时随机生成并打印到终端。默认只绑定本机地址。</p>
        <input id="password" class="input" type="password" placeholder="随机强密码" autocomplete="current-password" autofocus>
        <div class="button-row" style="margin-top:14px"><button type="submit">登录</button></div>
        <div id="loginToast" class="toast"></div>
      </form>
    </section>

    <section id="appView" class="grid hidden">
      <aside class="panel panel-pad">
        <p class="section-title">Runtime</p>
        <div class="metric"><span>运行状态</span><strong id="robotState">读取中</strong></div>
        <div class="metric"><span>启动时间</span><strong id="startedAt">—</strong></div>
        <div class="metric"><span>工作目录</span><strong id="rootDir">—</strong></div>
        <div class="metric"><span>捕获日志</span><strong id="stdoutLog">—</strong></div>
        <div class="actions">
          <button id="startBtn">启动</button>
          <button id="stopBtn" class="danger">停止</button>
        </div>
        <div class="button-row" style="margin-top:10px">
          <button id="refreshBtn" class="secondary" type="button">刷新日志</button>
          <button id="logoutBtn" class="secondary" type="button">退出</button>
        </div>
        <div id="appToast" class="toast"></div>
      </aside>

      <section class="panel">
        <div class="log-head">
          <div>
            <p class="section-title" style="margin-bottom:6px">Live Logs</p>
            <strong>日志监控</strong>
          </div>
          <select id="logSelect"></select>
        </div>
        <div class="terminal"><pre id="logOutput" class="empty">等待日志文件...</pre></div>
      </section>
    </section>
  </main>

  <script>
    const authed = document.body.dataset.authed === 'true';
    const loginView = document.querySelector('#loginView');
    const appView = document.querySelector('#appView');
    const topStatus = document.querySelector('#topStatus');
    const loginToast = document.querySelector('#loginToast');
    const appToast = document.querySelector('#appToast');
    const logSelect = document.querySelector('#logSelect');
    const logOutput = document.querySelector('#logOutput');
    let currentLog = '';
    let logTimer = null;

    function showApp(isAuthed) {
      loginView.classList.toggle('hidden', isAuthed);
      appView.classList.toggle('hidden', !isAuthed);
      if (isAuthed) bootstrap();
    }

    async function api(path, options = {}) {
      const res = await fetch(path, {
        headers: { 'Content-Type': 'application/json' },
        credentials: 'same-origin',
        ...options
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) throw new Error(data.error || '请求失败');
      return data;
    }

    document.querySelector('#loginForm').addEventListener('submit', async (event) => {
      event.preventDefault();
      loginToast.textContent = '';
      try {
        await api('/login', { method: 'POST', body: JSON.stringify({ password: document.querySelector('#password').value }) });
        showApp(true);
      } catch (error) {
        loginToast.textContent = error.message;
      }
    });

    document.querySelector('#startBtn').addEventListener('click', () => action('/api/start', '启动命令已发送'));
    document.querySelector('#stopBtn').addEventListener('click', () => action('/api/stop', '停止命令已发送'));
    document.querySelector('#refreshBtn').addEventListener('click', loadLogs);
    document.querySelector('#logoutBtn').addEventListener('click', async () => {
      await api('/logout', { method: 'POST' });
      location.reload();
    });
    logSelect.addEventListener('change', () => {
      currentLog = logSelect.value;
      loadCurrentLog();
    });

    async function action(path, okText) {
      appToast.textContent = '';
      try {
        await api(path, { method: 'POST' });
        appToast.textContent = okText;
        await refreshStatus();
        await loadLogs();
      } catch (error) {
        appToast.textContent = error.message;
      }
    }

    async function bootstrap() {
      await refreshStatus();
      await loadLogs();
      clearInterval(logTimer);
      setInterval(refreshStatus, 4000);
      logTimer = setInterval(loadCurrentLog, 1800);
    }

    async function refreshStatus() {
      try {
        const data = await api('/api/status');
        document.querySelector('#robotState').textContent = data.running ? '运行中' : '未运行';
        document.querySelector('#startedAt').textContent = data.startedAt || '—';
        document.querySelector('#rootDir').textContent = data.rootDir || '—';
        document.querySelector('#stdoutLog').textContent = data.stdoutLog || '—';
        document.querySelector('#startBtn').disabled = data.running;
        document.querySelector('#stopBtn').disabled = !data.running;
        topStatus.innerHTML = '<span class="dot ' + (data.running ? 'on' : '') + '"></span><span>' + (data.running ? '运行中' : '待机') + '</span>';
      } catch (error) {
        topStatus.innerHTML = '<span class="dot"></span><span>认证失效</span>';
      }
    }

    async function loadLogs() {
      const data = await api('/api/logs');
      const files = data.files || [];
      const previous = currentLog;
      logSelect.innerHTML = '';
      for (const file of files) {
        const option = document.createElement('option');
        option.value = file.name;
        option.textContent = file.name + ' · ' + formatBytes(file.size) + ' · ' + file.modTime;
        logSelect.appendChild(option);
      }
      currentLog = files.some(file => file.name === previous) ? previous : (files[0]?.name || '');
      logSelect.value = currentLog;
      await loadCurrentLog();
    }

    async function loadCurrentLog() {
      if (!currentLog) {
        logOutput.textContent = '暂无日志文件。启动机器人后会自动创建 log/*.log。';
        logOutput.classList.add('empty');
        return;
      }
      const data = await api('/api/logs/read?file=' + encodeURIComponent(currentLog));
      logOutput.textContent = data.content || '日志文件为空。';
      logOutput.classList.toggle('empty', !data.content);
      const terminal = logOutput.parentElement;
      terminal.scrollTop = terminal.scrollHeight;
    }

    function formatBytes(size) {
      if (size < 1024) return size + ' B';
      if (size < 1024 * 1024) return (size / 1024).toFixed(1) + ' KB';
      return (size / 1024 / 1024).toFixed(1) + ' MB';
    }

    showApp(authed);
  </script>
</body>
</html>`
