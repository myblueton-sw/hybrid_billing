package accesshistory

import "strings"

const (
	DeviceUnknown = "unknown"
	DeviceOther   = "other"
	uaPrefixLimit = 512
)

var (
	Browsers = []string{"edge", "chrome", "firefox", "safari", DeviceOther, DeviceUnknown}
	Systems  = []string{"windows", "macos", "linux", "ios", "android", "chromeos", DeviceOther, DeviceUnknown}
)

var browserTokens = []struct{ token, name string }{
	{"edg/", "edge"}, {"edge/", "edge"}, {"edga/", "edge"}, {"edgios/", "edge"},
	{"firefox/", "firefox"}, {"fxios/", "firefox"},
	{"crios/", "chrome"}, {"chrome/", "chrome"}, {"chromium/", "chrome"},
	{"safari/", "safari"},
}

var systemTokens = []struct{ token, name string }{
	{"iphone", "ios"}, {"ipad", "ios"}, {"ipod", "ios"},
	{"android", "android"},
	{"cros", "chromeos"},
	{"windows", "windows"},
	{"macintosh", "macos"}, {"mac os x", "macos"},
	{"linux", "linux"},
}

func Classify(userAgent string) (browser, system string) {
	if strings.TrimSpace(userAgent) == "" {
		return DeviceUnknown, DeviceUnknown
	}
	if len(userAgent) > uaPrefixLimit {
		userAgent = userAgent[:uaPrefixLimit]
	}
	ua := strings.ToLower(userAgent)
	browser, system = DeviceOther, DeviceOther
	for _, t := range browserTokens {
		if strings.Contains(ua, t.token) {
			browser = t.name
			break
		}
	}
	for _, t := range systemTokens {
		if strings.Contains(ua, t.token) {
			system = t.name
			break
		}
	}
	return browser, system
}
