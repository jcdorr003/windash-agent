//go:build tray
// +build tray

package tray

import (
	"github.com/getlantern/systray"
	"go.uber.org/zap"
)

// TODO: System tray implementation for post-MVP
// This will allow users to:
// - Open Dashboard
// - Start with OS (autostart)
// - Pause/Resume metrics collection
// - Quit the agent

// Manager handles the system tray
type Manager struct {
	logger       *zap.SugaredLogger
	dashboardURL string
}

// NewManager creates a new tray manager
func NewManager(logger *zap.SugaredLogger, dashboardURL string) *Manager {
	return &Manager{
		logger:       logger,
		dashboardURL: dashboardURL,
	}
}

// Run starts the system tray (blocking)
func (m *Manager) Run(onQuit func()) {
	systray.Run(func() {
		m.onReady(onQuit)
	}, func() {
		m.logger.Info("System tray exiting")
	})
}

func (m *Manager) onReady(onQuit func()) {
	systray.SetTitle("WinDash")
	systray.SetTooltip("WinDash Agent")
	// TODO: Set icon (systray.SetIcon)

	mOpen := systray.AddMenuItem("Open Dashboard", "Open WinDash dashboard in browser")
	systray.AddSeparator()
	mAutostart := systray.AddMenuItemCheckbox("Start with Windows", "Launch agent when Windows starts", false)
	systray.AddSeparator()
	mPause := systray.AddMenuItem("Pause", "Pause metrics collection")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Exit WinDash Agent")

	go func() {
		for {
			select {
			case <-mOpen.ClickedCh:
				m.logger.Info("Opening dashboard...")
				// TODO: Open browser
			case <-mAutostart.ClickedCh:
				// TODO: Toggle autostart
				if mAutostart.Checked() {
					mAutostart.Uncheck()
				} else {
					mAutostart.Check()
				}
			case <-mPause.ClickedCh:
				// TODO: Toggle pause/resume
				if mPause.Disabled() {
					mPause.Enable()
					mPause.SetTitle("Pause")
				} else {
					mPause.Disable()
					mPause.SetTitle("Resume")
				}
			case <-mQuit.ClickedCh:
				m.logger.Info("Quit requested from tray")
				systray.Quit()
				if onQuit != nil {
					onQuit()
				}
				return
			}
		}
	}()
}
