package ui

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-toast/toast"
)

// Notify sends a Windows Toast notification.
// Since Windows requires a file path for the icon, we extract it if needed.
func Notify(title, message, game string, iconData []byte) error {
	iconPath := ""
	if len(iconData) > 0 {
		tempDir := os.TempDir()
		iconFile := filepath.Join(tempDir, fmt.Sprintf("hoyo_%s.ico", game))
		
		// Always write/extract to ensure the latest icon is available
		err := os.WriteFile(iconFile, iconData, 0644)
		if err == nil {
			iconPath = iconFile
		}
	}

	notification := toast.Notification{
		AppID:   "HoyoLAB Unified Monitor",
		Title:   title,
		Message: message,
		Icon:    iconPath,
		Audio:   toast.Default,
	}

	return notification.Push()
}
