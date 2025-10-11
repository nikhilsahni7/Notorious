package utils

import (
	"fmt"

	"github.com/mileusna/useragent"
)

type DeviceInfo struct {
	DeviceType     string
	Browser        string
	BrowserVersion string
	OS             string
	OSVersion      string
}

// ParseUserAgent extracts device, browser, and OS information from User-Agent string
// Uses production-grade mileusna/useragent library for accurate parsing
func ParseUserAgent(userAgentString string) DeviceInfo {
	ua := useragent.Parse(userAgentString)

	info := DeviceInfo{
		DeviceType: "Desktop",
		Browser:    "Unknown",
		OS:         "Unknown",
	}

	// Detect Device Type
	if ua.Mobile {
		info.DeviceType = "Mobile"
	} else if ua.Tablet {
		info.DeviceType = "Tablet"
	} else if ua.Desktop {
		info.DeviceType = "Desktop"
	} else if ua.Bot {
		info.DeviceType = "Bot"
	}

	// Browser detection
	if ua.Name != "" {
		info.Browser = ua.Name
		info.BrowserVersion = ua.Version
	}

	// OS detection with version
	if ua.OS != "" {
		info.OS = ua.OS
		info.OSVersion = ua.OSVersion
	}

	// Handle specific cases
	if ua.IsAndroid() {
		info.OS = "Android"
	} else if ua.IsIOS() {
		info.OS = "iOS"
	} else if ua.IsWindows() {
		info.OS = "Windows"
	} else if ua.IsMacOS() {
		info.OS = "macOS"
	} else if ua.IsLinux() {
		info.OS = "Linux"
	}

	return info
}

// GetBrowserFullName returns browser name with version
func (d *DeviceInfo) GetBrowserFullName() string {
	if d.BrowserVersion != "" {
		return fmt.Sprintf("%s %s", d.Browser, d.BrowserVersion)
	}
	return d.Browser
}

// GetOSFullName returns OS name with version
func (d *DeviceInfo) GetOSFullName() string {
	if d.OSVersion != "" {
		return fmt.Sprintf("%s %s", d.OS, d.OSVersion)
	}
	return d.OS
}
