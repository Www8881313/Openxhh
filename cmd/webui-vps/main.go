package main

import (
	"bufio"
	"bytes"
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
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

const defaultAddr = ":29173"
const journalName = "__journal__"

var indexTemplate = template.Must(template.New("index").Parse(indexHTML))
var serviceNamePattern = regexp.MustCompile(`^[A-Za-z0-9_.@-]+$`)

type authStore struct {
	Salt string `json:"salt"`
	Hash string `json:"hash"`
}

type serverState struct {
	rootDir    string
	authPath   string
	listenAddr string
	service    string
	sessions   map[string]time.Time
	loginFails map[string]loginFail
	mu         sync.Mutex
}

type loginFail struct {
	Count     int
	LockedTil time.Time
}

type logFile struct {
	Name    string `json:"name"`
	Label   string `json:"label"`
	Size    int64  `json:"size"`
	ModTime string `json:"modTime"`
}

func main() {
	addr := flag.String("addr", defaultAddr, "public listen address for VPS web ui")
	root := flag.String("root", "/opt/xhhRobot", "xhhRobot working directory")
	service := flag.String("service", "xhhRobot", "systemd service name")
	flag.Parse()

	if err := validateServiceName(*service); err != nil {
		log.Fatal(err)
	}
	rootDir, err := filepath.Abs(*root)
	if err != nil {
		log.Fatal(err)
	}
	state := &serverState{
		rootDir:    rootDir,
		authPath:   filepath.Join(rootDir, "webui_auth.json"),
		service:    *service,
		sessions:   map[string]time.Time{},
		loginFails: map[string]loginFail{},
	}

	password, created, err := ensureAuth(state.authPath)
	if err != nil {
		log.Fatal(err)
	}
	if created {
		fmt.Println("xhhRobot VPS Web UI 已生成随机强密码")
		fmt.Println("登录密码:", password)
		fmt.Println("请立即保存；如需重置，停止 Web UI 后删除 webui_auth.json 再启动。")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", state.handleIndex)
	mux.HandleFunc("/login", state.handleLogin)
	mux.HandleFunc("/logout", state.requireAuth(state.handleLogout))
	mux.HandleFunc("/api/status", state.requireAuth(state.handleStatus))
	mux.HandleFunc("/api/start", state.requireAuth(state.handleStart))
	mux.HandleFunc("/api/stop", state.requireAuth(state.handleStop))
	mux.HandleFunc("/api/restart", state.requireAuth(state.handleRestart))
	mux.HandleFunc("/api/logs", state.requireAuth(state.handleLogs))
	mux.HandleFunc("/api/logs/read", state.requireAuth(state.handleReadLog))

	listener, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("监听 %s 失败: %v", *addr, err)
	}
	state.listenAddr = listener.Addr().String()
	fmt.Printf("xhhRobot VPS Web UI: http://%s\n", publicAddr(state.listenAddr))
	fmt.Printf("服务名: %s\n", state.service)
	fmt.Printf("工作目录: %s\n", state.rootDir)
	log.Fatal(http.Serve(listener, withSecurityHeaders(mux)))
}

func validateServiceName(service string) error {
	if service == "" || strings.HasPrefix(service, "-") || !serviceNamePattern.MatchString(service) {
		return fmt.Errorf("systemd 服务名无效: %q", service)
	}
	return nil
}

func ensureAuth(path string) (string, bool, error) {
	if _, err := os.Stat(path); err == nil {
		return "", false, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", false, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
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
	store := authStore{Salt: salt, Hash: hashPassword(password, salt)}
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

func (s *serverState) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = indexTemplate.Execute(w, map[string]any{"Authed": s.validSession(r), "Service": s.service})
}

func (s *serverState) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ip := clientIP(r)
	if lockedUntil := s.lockedUntil(ip); !lockedUntil.IsZero() {
		writeJSON(w, http.StatusTooManyRequests, map[string]any{"ok": false, "error": "登录失败过多，请稍后再试"})
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
		s.recordLoginFailure(ip)
		writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "密码错误"})
		return
	}
	s.clearLoginFailure(ip)
	token, err := randomPassword(48)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "无法创建会话"})
		return
	}
	s.mu.Lock()
	s.sessions[token] = time.Now().Add(24 * time.Hour)
	s.mu.Unlock()
	http.SetCookie(w, &http.Cookie{
		Name:     "xhh_vps_webui_session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(24 * time.Hour),
	})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *serverState) lockedUntil(ip string) time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	fail := s.loginFails[ip]
	if fail.LockedTil.After(time.Now()) {
		return fail.LockedTil
	}
	return time.Time{}
}

func (s *serverState) recordLoginFailure(ip string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fail := s.loginFails[ip]
	fail.Count++
	if fail.Count >= 5 {
		fail.Count = 0
		fail.LockedTil = time.Now().Add(5 * time.Minute)
	}
	s.loginFails[ip] = fail
}

func (s *serverState) clearLoginFailure(ip string) {
	s.mu.Lock()
	delete(s.loginFails, ip)
	s.mu.Unlock()
}

func (s *serverState) validSession(r *http.Request) bool {
	cookie, err := r.Cookie("xhh_vps_webui_session")
	if err != nil || cookie.Value == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	expiresAt, ok := s.sessions[cookie.Value]
	if !ok || time.Now().After(expiresAt) {
		delete(s.sessions, cookie.Value)
		return false
	}
	return true
}

func (s *serverState) handleLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("xhh_vps_webui_session"); err == nil {
		s.mu.Lock()
		delete(s.sessions, cookie.Value)
		s.mu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{Name: "xhh_vps_webui_session", Path: "/", MaxAge: -1})
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
	active, activeErr := s.systemctl("is-active")
	status, statusErr := s.systemctl("status", "--no-pager")
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":         true,
		"service":    s.service,
		"running":    strings.TrimSpace(active) == "active",
		"active":     strings.TrimSpace(active),
		"detail":     errorText(activeErr),
		"rootDir":    s.rootDir,
		"listenAddr": s.listenAddr,
		"statusText": trimStatus(firstNonEmpty(status, errorText(statusErr))),
	})
}

func (s *serverState) handleStart(w http.ResponseWriter, r *http.Request) {
	s.handleSystemctlAction(w, r, "start")
}

func (s *serverState) handleStop(w http.ResponseWriter, r *http.Request) {
	s.handleSystemctlAction(w, r, "stop")
}

func (s *serverState) handleRestart(w http.ResponseWriter, r *http.Request) {
	s.handleSystemctlAction(w, r, "restart")
}

func (s *serverState) handleSystemctlAction(w http.ResponseWriter, r *http.Request, action string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	out, err := s.systemctl(action)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": out})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *serverState) systemctl(args ...string) (string, error) {
	cmdArgs := append(args, s.service)
	cmd := exec.Command("systemctl", cmdArgs...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return strings.TrimSpace(buf.String()), err
}

func (s *serverState) handleLogs(w http.ResponseWriter, r *http.Request) {
	files := []logFile{{Name: journalName, Label: "systemd journal · " + s.service}}
	logFiles, _ := listLogFiles(filepath.Join(s.rootDir, "log"))
	files = append(files, logFiles...)
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
			Label:   entry.Name(),
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
	if name == journalName || name == "" {
		content, err := s.readJournal()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": content})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "content": content})
		return
	}
	if strings.Contains(name, "/") || strings.Contains(name, "\\") || strings.Contains(name, "..") {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "日志文件名无效"})
		return
	}
	content, err := tailFile(filepath.Join(s.rootDir, "log", name), 800)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "content": content})
}

func (s *serverState) readJournal() (string, error) {
	cmd := exec.Command("journalctl", "-u", s.service, "-n", "800", "--no-pager", "-o", "short-iso")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return strings.TrimRight(buf.String(), "\n"), err
}

func tailFile(path string, maxLines int) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	var lines []string
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
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
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline'; connect-src 'self'; img-src 'self' data:")
		next.ServeHTTP(w, r)
	})
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func trimStatus(status string) string {
	lines := strings.Split(status, "\n")
	if len(lines) > 24 {
		lines = lines[:24]
	}
	return strings.Join(lines, "\n")
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func publicAddr(addr string) string {
	if strings.HasPrefix(addr, "[::]:") {
		return "服务器IP" + strings.TrimPrefix(addr, "[::]")
	}
	if strings.HasPrefix(addr, "0.0.0.0:") {
		return "服务器IP" + strings.TrimPrefix(addr, "0.0.0.0")
	}
	return addr
}

const indexHTML = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>xhhRobot VPS 控制台</title>
  <style>
    :root{color-scheme:dark;--bg:#07090d;--card:rgba(16,22,32,.88);--card2:rgba(23,32,47,.96);--line:rgba(133,158,188,.2);--text:#e9f0fb;--muted:#8592a6;--cyan:#43dcff;--green:#7dffb2;--amber:#ffd36e;--red:#ff6678;--shadow:0 28px 90px rgba(0,0,0,.46)}
    *{box-sizing:border-box}body{margin:0;min-height:100vh;font-family:ui-monospace,SFMono-Regular,Menlo,Consolas,"Liberation Mono",monospace;color:var(--text);background:radial-gradient(circle at 15% 12%,rgba(67,220,255,.17),transparent 32rem),radial-gradient(circle at 84% 0%,rgba(125,255,178,.11),transparent 28rem),linear-gradient(135deg,#05070a,#0b111b 52%,#071017)}
    body:before{content:"";position:fixed;inset:0;pointer-events:none;opacity:.22;background-image:linear-gradient(rgba(255,255,255,.05) 1px,transparent 1px),linear-gradient(90deg,rgba(255,255,255,.04) 1px,transparent 1px);background-size:46px 46px;mask-image:linear-gradient(to bottom,black,transparent 85%)}
    .shell{width:min(1200px,calc(100vw - 38px));margin:0 auto;padding:34px 0 42px;position:relative}.topbar{display:flex;align-items:flex-start;justify-content:space-between;gap:16px;margin-bottom:22px}.brand{display:flex;gap:14px;align-items:center}.mark{width:42px;height:42px;border:1px solid rgba(67,220,255,.48);background:linear-gradient(145deg,rgba(67,220,255,.22),rgba(125,255,178,.09));box-shadow:0 0 28px rgba(67,220,255,.22);transform:rotate(45deg)}
    h1{margin:0;font-size:clamp(25px,3.3vw,44px);line-height:.95;letter-spacing:-.06em}.brand p{margin:7px 0 0;color:var(--muted);font-size:13px}.pill{display:inline-flex;align-items:center;gap:8px;padding:10px 13px;border:1px solid var(--line);border-radius:999px;background:rgba(8,12,18,.66);color:var(--muted)}.dot{width:8px;height:8px;border-radius:50%;background:var(--red);box-shadow:0 0 14px currentColor;color:var(--red)}.dot.on{background:var(--green);color:var(--green)}
    .login{min-height:72vh;display:grid;place-items:center}.login-card{width:min(450px,100%);padding:30px;border:1px solid var(--line);border-radius:26px;background:linear-gradient(180deg,rgba(23,32,47,.94),rgba(10,14,21,.9));box-shadow:var(--shadow)}.login-card h2{margin:0 0 8px;font-size:28px;letter-spacing:-.04em}.login-card p{margin:0 0 20px;color:var(--muted);line-height:1.65}
    input,select,button{font:inherit}.input,select{width:100%;border:1px solid var(--line);background:rgba(2,5,9,.66);color:var(--text);border-radius:15px;padding:13px 14px;outline:none}.input:focus,select:focus{border-color:rgba(67,220,255,.65);box-shadow:0 0 0 4px rgba(67,220,255,.1)}button{border:0;cursor:pointer;color:#031016;background:var(--cyan);padding:12px 16px;border-radius:14px;font-weight:800;transition:.18s ease}button:hover{transform:translateY(-1px);filter:brightness(1.07)}button.secondary{color:var(--text);background:rgba(255,255,255,.08);border:1px solid var(--line)}button.danger{color:#260106;background:var(--red)}button.warn{color:#211400;background:var(--amber)}button:disabled{opacity:.45;cursor:not-allowed;transform:none}
    .grid{display:grid;grid-template-columns:370px 1fr;gap:18px}.panel{border:1px solid var(--line);border-radius:26px;background:var(--card);box-shadow:var(--shadow);backdrop-filter:blur(18px)}.pad{padding:22px}.section{margin:0 0 16px;color:var(--muted);font-size:12px;letter-spacing:.18em;text-transform:uppercase}.metric{display:grid;gap:6px;padding:15px 0;border-bottom:1px solid var(--line)}.metric:last-child{border-bottom:0}.metric span{color:var(--muted);font-size:12px}.metric strong{font-size:15px;white-space:pre-wrap;word-break:break-word}.actions{display:grid;grid-template-columns:1fr 1fr 1fr;gap:9px;margin-top:18px}.row{display:flex;gap:9px;flex-wrap:wrap;margin-top:10px}.log-head{display:flex;justify-content:space-between;align-items:center;gap:14px;padding:18px;border-bottom:1px solid var(--line)}.log-head select{max-width:430px}.terminal{height:min(67vh,700px);overflow:auto;padding:20px;background:rgba(0,0,0,.44);border-radius:0 0 26px 26px}pre{margin:0;white-space:pre-wrap;word-break:break-word;line-height:1.55;color:#c8d7ea;font-size:13px}.empty{color:var(--muted);display:grid;place-items:center;min-height:320px;text-align:center;line-height:1.7}.toast{min-height:22px;margin-top:14px;color:var(--amber);font-size:13px}.hidden{display:none!important}.warnbox{border:1px solid rgba(255,211,110,.28);background:rgba(255,211,110,.08);color:#ffe2a0;border-radius:16px;padding:12px 13px;line-height:1.55;font-size:12px;margin-bottom:14px}
    @media(max-width:900px){.grid{grid-template-columns:1fr}.topbar,.log-head{flex-direction:column;align-items:stretch}.actions{grid-template-columns:1fr}.log-head select{max-width:none}}
  </style>
</head>
<body data-authed="{{.Authed}}">
  <main class="shell">
    <div class="topbar"><div class="brand"><div class="mark"></div><div><h1>xhhRobot VPS</h1><p>公网控制台 · 默认端口 29173 · systemd 服务：{{.Service}}</p></div></div><div id="topStatus" class="pill"><span class="dot"></span><span>未连接</span></div></div>
    <section id="loginView" class="login hidden"><form id="loginForm" class="login-card"><h2>进入 VPS 控制台</h2><p>密码在首次启动时随机生成并打印到终端。公网使用时建议只开放给可信 IP 或放到 HTTPS 反代后面。</p><input id="password" class="input" type="password" placeholder="随机强密码" autocomplete="current-password" autofocus><div class="row"><button type="submit">登录</button></div><div id="loginToast" class="toast"></div></form></section>
    <section id="appView" class="grid hidden"><aside class="panel pad"><p class="section">Systemd</p><div class="warnbox">VPS 版通过 systemctl 控制机器人服务，通过 journalctl 读取服务日志。</div><div class="metric"><span>服务状态</span><strong id="serviceState">读取中</strong></div><div class="metric"><span>监听地址</span><strong id="listenAddr">—</strong></div><div class="metric"><span>工作目录</span><strong id="rootDir">—</strong></div><div class="metric"><span>systemctl status</span><strong id="statusText">—</strong></div><div class="actions"><button id="startBtn">启动</button><button id="restartBtn" class="warn">重启</button><button id="stopBtn" class="danger">停止</button></div><div class="row"><button id="refreshBtn" class="secondary" type="button">刷新日志</button><button id="logoutBtn" class="secondary" type="button">退出</button></div><div id="appToast" class="toast"></div></aside><section class="panel"><div class="log-head"><div><p class="section" style="margin-bottom:6px">VPS Logs</p><strong>日志监控</strong></div><select id="logSelect"></select></div><div class="terminal"><pre id="logOutput" class="empty">等待日志...</pre></div></section></section>
  </main>
<script>
const authed=document.body.dataset.authed==='true';const loginView=document.querySelector('#loginView');const appView=document.querySelector('#appView');const topStatus=document.querySelector('#topStatus');const loginToast=document.querySelector('#loginToast');const appToast=document.querySelector('#appToast');const logSelect=document.querySelector('#logSelect');const logOutput=document.querySelector('#logOutput');let currentLog='';let logTimer=null;let statusTimer=null;
function showApp(ok){loginView.classList.toggle('hidden',ok);appView.classList.toggle('hidden',!ok);if(ok)bootstrap()}async function api(path,options={}){const res=await fetch(path,{headers:{'Content-Type':'application/json'},credentials:'same-origin',...options});const data=await res.json().catch(()=>({}));if(!res.ok)throw new Error(data.error||'请求失败');return data}
document.querySelector('#loginForm').addEventListener('submit',async e=>{e.preventDefault();loginToast.textContent='';try{await api('/login',{method:'POST',body:JSON.stringify({password:document.querySelector('#password').value})});showApp(true)}catch(err){loginToast.textContent=err.message}});
document.querySelector('#startBtn').addEventListener('click',()=>action('/api/start','启动命令已发送'));document.querySelector('#stopBtn').addEventListener('click',()=>action('/api/stop','停止命令已发送'));document.querySelector('#restartBtn').addEventListener('click',()=>action('/api/restart','重启命令已发送'));document.querySelector('#refreshBtn').addEventListener('click',loadLogs);document.querySelector('#logoutBtn').addEventListener('click',async()=>{await api('/logout',{method:'POST'});location.reload()});logSelect.addEventListener('change',()=>{currentLog=logSelect.value;loadCurrentLog()});
async function action(path,text){appToast.textContent='';try{await api(path,{method:'POST'});appToast.textContent=text;setTimeout(refreshStatus,900);setTimeout(loadCurrentLog,1200)}catch(err){appToast.textContent=err.message}}async function bootstrap(){await refreshStatus();await loadLogs();clearInterval(logTimer);clearInterval(statusTimer);statusTimer=setInterval(refreshStatus,4000);logTimer=setInterval(loadCurrentLog,1800)}
async function refreshStatus(){try{const d=await api('/api/status');const running=d.running;document.querySelector('#serviceState').textContent=(d.active||'unknown')+(d.detail?'\n'+d.detail:'');document.querySelector('#listenAddr').textContent=d.listenAddr||'—';document.querySelector('#rootDir').textContent=d.rootDir||'—';document.querySelector('#statusText').textContent=d.statusText||'—';document.querySelector('#startBtn').disabled=running;document.querySelector('#stopBtn').disabled=!running;topStatus.innerHTML='<span class="dot '+(running?'on':'')+'"></span><span>'+(running?'运行中':'待机')+'</span>'}catch(err){topStatus.innerHTML='<span class="dot"></span><span>认证失效</span>'}}
async function loadLogs(){const d=await api('/api/logs');const files=d.files||[];const previous=currentLog;logSelect.innerHTML='';for(const file of files){const option=document.createElement('option');option.value=file.name;option.textContent=(file.label||file.name)+(file.size?' · '+formatBytes(file.size):'')+(file.modTime?' · '+file.modTime:'');logSelect.appendChild(option)}currentLog=files.some(f=>f.name===previous)?previous:(files[0]?.name||'');logSelect.value=currentLog;await loadCurrentLog()}async function loadCurrentLog(){if(!currentLog){logOutput.textContent='暂无日志。';logOutput.classList.add('empty');return}const d=await api('/api/logs/read?file='+encodeURIComponent(currentLog));logOutput.textContent=d.content||'日志为空。';logOutput.classList.toggle('empty',!d.content);const box=logOutput.parentElement;box.scrollTop=box.scrollHeight}function formatBytes(n){if(n<1024)return n+' B';if(n<1024*1024)return(n/1024).toFixed(1)+' KB';return(n/1024/1024).toFixed(1)+' MB'}showApp(authed);
</script>
</body>
</html>`
