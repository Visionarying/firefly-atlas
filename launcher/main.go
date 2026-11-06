package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

const (
	appName    = "Firefly Atlas"
	port       = "3002"
	healthURL  = "http://127.0.0.1:" + port + "/api/v1/health"
	appURL     = "http://127.0.0.1:" + port
	startWait  = 35 * time.Second
	checkEvery = 750 * time.Millisecond
)

var (
	user32       = syscall.NewLazyDLL("user32.dll")
	shell32      = syscall.NewLazyDLL("shell32.dll")
	messageBoxW  = user32.NewProc("MessageBoxW")
	shellExecute = shell32.NewProc("ShellExecuteW")
)

func utf16(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(s)
	return p
}

func notify(title, text string) {
	_, _, _ = messageBoxW.Call(0,
		uintptr(unsafe.Pointer(utf16(text))),
		uintptr(unsafe.Pointer(utf16(title))),
		0x00000010, // MB_ICONERROR
	)
}

func openBrowser(url string) {
	op := utf16("open")
	file := utf16(url)
	_, _, _ = shellExecute.Call(0,
		uintptr(unsafe.Pointer(op)),
		uintptr(unsafe.Pointer(file)),
		0, 0, 1,
	)
}

func healthOK() bool {
	client := &http.Client{Timeout: 1200 * time.Millisecond}
	resp, err := client.Get(healthURL)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode == http.StatusOK
}

func main() {
	exe, err := os.Executable()
	if err != nil {
		notify(appName, "无法定位启动器路径："+err.Error())
		return
	}
	root := filepath.Dir(exe)
	server := filepath.Join(root, "service", "firefly-atlas.exe")
	if _, err := os.Stat(server); err != nil {
		notify(appName, "没有找到 service\\firefly-atlas.exe。\n\n请先运行 scripts\\build.ps1，或确认启动器和 service 文件夹在同一目录。")
		return
	}

	// Already running: just open the browser. This is also what happens when the
	// user double-clicks the shortcut a second time.
	if healthOK() {
		openBrowser(appURL)
		return
	}

	logDir := filepath.Join(root, "data", "logs")
	_ = os.MkdirAll(logDir, 0o755)
	logPath := filepath.Join(logDir, "server.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		notify(appName, "无法创建日志文件：\n"+logPath+"\n\n"+err.Error())
		return
	}
	defer logFile.Close()

	cmd := exec.Command(server)
	cmd.Dir = root
	cmd.Env = append(os.Environ(),
		"DATA_DIR="+filepath.Join(root, "data"),
		"PORT="+port,
		"GIN_MODE=release",
	)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
	if err := cmd.Start(); err != nil {
		notify(appName, "启动服务失败：\n"+err.Error())
		return
	}
	waitCh := make(chan error, 1)
	go func() { waitCh <- cmd.Wait() }()

	deadline := time.Now().Add(startWait)
	for time.Now().Before(deadline) {
		if healthOK() {
			openBrowser(appURL)
			return
		}
		select {
		case err := <-waitCh:
			notify(appName, fmt.Sprintf("服务启动后立即退出：\n%v\n\n请查看日志：\n%s", err, logPath))
			return
		default:
			time.Sleep(checkEvery)
		}
	}

	notify(appName, fmt.Sprintf("服务在 %d 秒内没有就绪。\n\n请查看日志：\n%s\n\n常见原因：端口 %s 被占用，或卡巴斯基/防火墙拦截了 %s。", int(startWait.Seconds()), logPath, port, strings.TrimSpace(server)))
}
