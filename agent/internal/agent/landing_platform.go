package agent

import "strings"

// Platforms the phone page ("Get tailbond") has a section for; the order
// is the switcher's.
const (
	platAndroid = "android"
	platMac     = "mac"
	platIPhone  = "iphone"
	platLinux   = "linux"
	platWindows = "windows" // no section: a note, and the Android steps
)

// landingPlatform guesses the visitor's OS from the User-Agent (and the
// Sec-CH-UA-Platform client hint, when the browser sends it). The page's
// script refines it (an iPad asks for the desktop site and says
// "Macintosh"); the switcher always lets the visitor pick another.
func landingPlatform(ua, chPlatform string) string {
	switch strings.ToLower(strings.Trim(chPlatform, `" `)) {
	case "android":
		return platAndroid
	case "macos":
		return platMac
	case "ios":
		return platIPhone
	case "windows":
		return platWindows
	case "linux", "chrome os", "chromeos":
		return platLinux
	}
	switch {
	case strings.Contains(ua, "Android"):
		return platAndroid
	case strings.Contains(ua, "iPhone"), strings.Contains(ua, "iPad"), strings.Contains(ua, "iPod"):
		return platIPhone
	case strings.Contains(ua, "Macintosh"), strings.Contains(ua, "Mac OS X"):
		return platMac
	case strings.Contains(ua, "Windows"):
		return platWindows
	case strings.Contains(ua, "Linux"), strings.Contains(ua, "X11"), strings.Contains(ua, "CrOS"):
		return platLinux
	}
	return platAndroid
}

// landingSection is the section shown first for a detected platform
// (Windows has none: its note sits above the Android steps).
func landingSection(p string) string {
	if p == platWindows {
		return platAndroid
	}
	return p
}
