package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"resin/pkg/autostart"
	"resin/pkg/config"
	"resin/pkg/logging"
	"resin/pkg/util"
	"time"

	"github.com/energye/systray"
	"github.com/skratchdot/open-golang/open"
	"golang.org/x/sys/windows"
)

const (
	defaultTheme    = 0
	allowDarkTheme  = 1
	forceDarkTheme  = 2
	forceLightTheme = 3
	maxTheme        = 4
)

var (
	ux                   = windows.NewLazySystemDLL("uxtheme.dll")
	pSetPreferredAppMode = util.NewProcByOrdinal(ux, 135)
	pFlushMenuThemes     = util.NewProcByOrdinal(ux, 136)
)

type CommonMenu struct {
	Refresh   *systray.MenuItem
	Quit      *systray.MenuItem
	Advanced  *systray.MenuItem
	Logs      *systray.MenuItem
	Login     *systray.MenuItem
	DarkMode  *systray.MenuItem
	Autostart *systray.MenuItem
}

func CreateMenuItem(title string, icon []byte) *systray.MenuItem {
	item := systray.AddMenuItem(title, "")
	item.SetIcon(icon)
	return item
}

func refreshLoop[T any](mgr *config.Manager, menu *T, refresh func(*config.Config, *T)) {
	for {
		cfg := mgr.Get()
		duration_secs := 60
		if cfg != nil {
			duration_secs = cfg.RefreshInterval
		}
		duration := time.Duration(duration_secs) * time.Second

		refresh(cfg, menu)
		time.Sleep(duration)
	}
}

func watchEvents[T any](cm *CommonMenu, mgr *config.Manager, menu *T, auto *autostart.App, logFile string, configFile string, app string, refresh func(*config.Config, *T)) {
	cm.Quit.Click(func() {
		systray.Quit()
	})
	cm.Refresh.Click(func() {
		logging.Info("User clicked refresh")
		refresh(mgr.Get(), menu)
	})
	cm.Logs.Click(func() {
		logging.Info(fmt.Sprintf("Opening \"%s\"", logFile))
		open.Start(logFile)
	})
	cm.Login.Click(func() {
		var err error
		cfg, err := login(app, configFile, mgr.Get(), menu, refresh)
		if err != nil {
			logging.Fail("Failed to login:\n%s", err)
			return
		}
		mgr.Set(cfg)
	})
	cm.DarkMode.Click(func() {
		if cm.DarkMode.Checked() {
			cm.DarkMode.Uncheck()
			SetTheme(forceLightTheme)
		} else {
			cm.DarkMode.Check()
			SetTheme(forceDarkTheme)
		}
	})
	cm.Autostart.Click(func() {
		if cm.Autostart.Checked() {
			cm.Autostart.Uncheck()
			auto.Disable()
		} else {
			cm.Autostart.Check()
			auto.Enable()
		}
	})
}

func login[T any](app string, configFile string, cfg *config.Config, menu *T, refresh func(*config.Config, *T)) (*config.Config, error) {
	wd, err := os.Getwd()
	if err != nil {
		logging.Fail("Failed to get working directory: %v", err)
		return nil, err
	}
	exeName := fmt.Sprintf("WebViewLogin-%s.exe", config.VERSION)
	exe := filepath.Join(wd, "login", exeName)
	
	logging.Info("Launching login helper: %s %s", exe, app)
	cmd := exec.Command(exe, app)
	cmd.Dir = "."
	
	// Block until finished
	output, err := cmd.CombinedOutput()
	if err != nil {
		logging.Fail("Login helper exited with error: %v\nOutput: %s", err, string(output))
		windows.MessageBox(0, 
			windows.StringToUTF16Ptr(fmt.Sprintf("Failed to launch login window.\n\nPlease ensure you have the .NET 8.0 Desktop Runtime installed.\n\nError: %v", err)), 
			windows.StringToUTF16Ptr("HoyoLAB Monitor Error"), 
			windows.MB_OK|windows.MB_ICONERROR)
		return nil, err
	}
	logging.Info("Login helper finished successfully")

	cookies, err := config.LoadConfig(configFile)
	if err != nil {
		logging.Fail("Failed to load new config after login: %v", err)
		return nil, err
	}
	logging.Info("Successfully refreshed credentials from login helper")
	refresh(cookies, menu)
	return cookies, nil
}

func SetTheme(code uintptr) {
	pSetPreferredAppMode.Call(code)
	pFlushMenuThemes.Call()
}

func InitApp[T any](title string, tooltip string, icon []byte, logFile string, configFile string, menu *T, app string, refresh func(*config.Config, *T)) *config.Manager {
	systray.SetOnClick(func(menu systray.IMenu) {
		menu.ShowMenu()
	})
	logging.Info("Application start")

	systray.SetIcon(icon)
	systray.SetTitle(title)
	systray.SetTooltip(tooltip)

	systray.AddSeparator()

	cm := &CommonMenu{}

	cm.Advanced = systray.AddMenuItem("Advanced", "Advanced options")
	cm.Logs = cm.Advanced.AddSubMenuItem("Logs", "Show logs")
	cm.Login = cm.Advanced.AddSubMenuItem("Login", "Login To Hoyolab")

	cm.Refresh = systray.AddMenuItem("Refresh", "Refresh data")
	cm.Quit = systray.AddMenuItem("Quit", "Exit the application")

	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		cfg, err = login(app, configFile, cfg, menu, refresh)
		if err != nil {
			logging.Fail("Failed to login")
			os.Exit(1)
			return nil
		}
	}

	mgr := config.NewManager(cfg)

	if cfg.DarkMode {
		SetTheme(forceDarkTheme)
	}
	cm.DarkMode = cm.Advanced.AddSubMenuItemCheckbox("Dark Mode", "Dark Mode", cfg.DarkMode)

	exec, err := os.Executable()
	var auto *autostart.App
	enabled := false
	if err == nil {
		auto = &autostart.App{
			Name:             app,
			FileName:         fmt.Sprintf("%s.lnk", app),
			Exec:             []string{exec},
			WorkingDirectory: filepath.Dir(exec),
		}
		enabled = auto.IsEnabled()
	}
	cm.Autostart = cm.Advanced.AddSubMenuItemCheckbox("Autostart", "Autostart", enabled)

	go refreshLoop(mgr, menu, refresh)

	watchEvents(cm, mgr, menu, auto, logFile, configFile, app, refresh)
	return mgr
}
