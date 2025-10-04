package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strings"
	"sync"
	"syscall"

	"golang.org/x/sys/windows"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx    context.Context
	cmd    *exec.Cmd
	mu     sync.Mutex
	status string
	ip     string
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// Connect starts the npc connection
func (a *App) Connect(address, port, vkey string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.cmd != nil {
		return fmt.Errorf("already connected")
	}

	// 获取 npc.exe 路径
	npcPath, err := getNpcPath()
	if err != nil {
		return fmt.Errorf("获取 npc.exe 路径失败: %v", err)
	}

	// 构建npc命令，格式：npc.exe -server=地址:端口 -vkey=xxx -type=tcp
	args := []string{
		"-server=" + address + ":" + port,
		"-vkey=" + vkey,
		"-type=tcp",
	}

	log.Printf("执行命令: %s %v", npcPath, args)

	a.cmd = exec.Command(npcPath, args...)
	a.cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: windows.CREATE_NEW_PROCESS_GROUP | windows.CREATE_NO_WINDOW,
	}

	// 设置输出管道
	stdout, err := a.cmd.StdoutPipe()
	if err != nil {
		log.Printf("创建stdout管道失败: %v", err)
		return err
	}

	stderr, err := a.cmd.StderrPipe()
	if err != nil {
		log.Printf("创建stderr管道失败: %v", err)
		return err
	}

	// 启动进程
	if err := a.cmd.Start(); err != nil {
		log.Printf("启动进程失败: %v", err)
		a.cmd = nil
		return err
	}

	// 更新状态
	a.status = "正在连接..."
	a.ip = ""
	wailsRuntime.EventsEmit(a.ctx, "status-update", a.status)
	wailsRuntime.EventsEmit(a.ctx, "connection-state", true)

	// TODO: 根据 npc.exe 输出调整正则和状态解析

	go func() {
		// 读取stdout
		go func() {
			buf := make([]byte, 1024)
			for {
				n, err := stdout.Read(buf)
				if err != nil {
					if err != io.EOF && !strings.Contains(err.Error(), "file already closed") {
						log.Printf("读取stdout失败: %v", err)
					}
					break
				}
				output := string(buf[:n])
				log.Printf("npc输出: %s", output)
				if strings.Contains(output, "Successful connection with server") {
					a.mu.Lock()
					a.status = "已连接"
					a.mu.Unlock()
					wailsRuntime.EventsEmit(a.ctx, "status-update", a.status)
				}
				if strings.Contains(output, "The connection server failed and will be reconnected in five seconds") {
					a.mu.Lock()
					a.status = "已断开"
					a.mu.Unlock()
					wailsRuntime.EventsEmit(a.ctx, "status-update", a.status)
					wailsRuntime.EventsEmit(a.ctx, "connection-state", false)
					a.Disconnect()
				}
			}
		}()

		// 读取stderr
		go func() {
			buf := make([]byte, 1024)
			for {
				n, err := stderr.Read(buf)
				if err != nil {
					if err != io.EOF && !strings.Contains(err.Error(), "file already closed") {
						log.Printf("读取stderr失败: %v", err)
					}
					break
				}
				output := string(buf[:n])
				log.Printf("npc输出: %s", output)
				// TODO: 可根据 npc.exe 输出内容调整状态
			}
		}()

		if err := a.cmd.Wait(); err != nil {
			if a.cmd != nil {
				log.Printf("进程退出: %v", err)
				a.mu.Lock()
				a.cmd = nil
				a.status = "已断开"
				a.ip = ""
				wailsRuntime.EventsEmit(a.ctx, "status-update", a.status)
				a.mu.Unlock()
			}
		}
	}()

	return nil
}

// Disconnect stops the n3n connection
func (a *App) Disconnect() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.cmd == nil {
		return fmt.Errorf("not connected")
	}

	log.Printf("正在断开连接，进程ID: %d", a.cmd.Process.Pid)

	// 在Windows上使用taskkill强制终止进程，并隐藏命令行窗口
	cmd := exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprint(a.cmd.Process.Pid))
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: windows.CREATE_NEW_PROCESS_GROUP | windows.CREATE_NO_WINDOW,
	}
	if err := cmd.Run(); err != nil {
		log.Printf("终止进程失败: %v", err)
		return err
	}

	// 等待进程完全退出
	a.cmd.Wait()

	a.cmd = nil
	a.status = "已断开"
	a.ip = ""
	wailsRuntime.EventsEmit(a.ctx, "status-update", a.status)
	wailsRuntime.EventsEmit(a.ctx, "connection-state", false)

	return nil
}

// GetStatus returns the current connection status
func (a *App) GetStatus() (string, string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.status, a.ip
}
