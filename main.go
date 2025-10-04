package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"

	// "syscall"
	// "unsafe"
	// "golang.org/x/sys/windows"
	"github.com/getlantern/systray"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend
var assets embed.FS

//go:embed npc.exe
var npcBinary []byte

//go:embed assets/icon.ico
var iconData []byte

// 全局变量
var app *App

const (
	mutexName   = "Global\\KistuTunnelSingleInstanceMutex"
	windowTitle = "KistuTunnel"
)

func main() {
	// 创建应用实例
	app = NewApp()

	// 启动系统托盘
	go systray.Run(onReady, onExit)

	// 创建应用配置
	err := wails.Run(&options.App{
		Title:            "KistuTunnel",
		Width:            400,
		Height:           600,
		MinWidth:         400,
		MinHeight:        600,
		MaxWidth:         400,
		MaxHeight:        600,
		DisableResize:    true,
		Assets:           assets,
		BackgroundColour: &options.RGBA{R: 245, G: 245, B: 245, A: 1},
		OnStartup:        app.startup,
		OnBeforeClose: func(ctx context.Context) (prevent bool) {
			runtime.Hide(ctx)
			return true // 阻止默认关闭，改为隐藏
		},
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		log.Fatal(err)
	}
}

// 托盘准备函数
func onReady() {
	// 设置托盘图标
	systray.SetIcon(getIcon())
	systray.SetTitle("KistuTunnel")
	systray.SetTooltip("KistuTunnel")

	// 创建菜单项
	mOpen := systray.AddMenuItem("打开主界面", "打开主界面")
	mQuit := systray.AddMenuItem("退出", "退出应用")

	// 处理菜单点击事件
	go func() {
		for {
			select {
			case <-mOpen.ClickedCh:
				runtime.Show(app.ctx)
			case <-mQuit.ClickedCh:
				// 先停止连接
				if app != nil {
					app.Disconnect()
				}
				// 退出系统托盘
				systray.Quit()
				// 退出应用
				os.Exit(0)
			}
		}
	}()
}

// 托盘退出函数
func onExit() {
	// 清理托盘图标
	systray.Quit()
	// 确保应用退出
	os.Exit(0)
}

// 获取图标数据
func getIcon() []byte {
	if len(iconData) == 0 {
		log.Printf("嵌入的图标数据为空")
	}
	return iconData
}

// 释放并获取 npc.exe 路径
func getNpcPath() (string, error) {
	// 获取临时目录
	tempDir := os.TempDir()
	kistuTunnelDir := filepath.Join(tempDir, "KistuTunnel")

	// 创建目录（如果不存在）
	if err := os.MkdirAll(kistuTunnelDir, 0755); err != nil {
		return "", fmt.Errorf("创建临时目录失败: %v", err)
	}

	// 设置 npc.exe 路径
	npcPath := filepath.Join(kistuTunnelDir, "npc.exe")

	// 如果文件已存在，先删除
	if _, err := os.Stat(npcPath); err == nil {
		if err := os.Remove(npcPath); err != nil {
			return "", fmt.Errorf("删除已存在的 npc.exe 失败: %v", err)
		}
	}

	// 写入文件
	if err := os.WriteFile(npcPath, npcBinary, 0755); err != nil {
		return "", fmt.Errorf("写入 npc.exe 失败: %v", err)
	}

	return npcPath, nil
}

// 激活已存在的主界面窗口
// func activateExistingWindow() {
// 	title, _ := syscall.UTF16PtrFromString(windowTitle)
// 	hwnd, _, _ := windows.NewLazySystemDLL("user32.dll").NewProc("FindWindowW").Call(0, uintptr(unsafe.Pointer(title)))
// 	if hwnd != 0 {
// 		windows.NewLazySystemDLL("user32.dll").NewProc("ShowWindow").Call(hwnd, 9) // SW_RESTORE
// 		windows.NewLazySystemDLL("user32.dll").NewProc("SetForegroundWindow").Call(hwnd)
// 	}
// }
