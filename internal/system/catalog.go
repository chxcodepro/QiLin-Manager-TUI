package system

type AppInfo struct {
	Name        string
	Package     string
	Description string
	InstallMode string
}

func DefaultApps() []AppInfo {
	return []AppInfo{
		{Name: "WPS Office", Package: "wps-office", Description: "办公套件", InstallMode: "apt"},
		{Name: "微信", Package: "electronic-wechat", Description: "微信桌面端", InstallMode: "apt"},
		{Name: "QQ", Package: "linuxqq", Description: "QQ Linux 客户端", InstallMode: "apt"},
		{Name: "百度网盘", Package: "netdisk", Description: "网盘客户端", InstallMode: "apt"},
		{Name: "麒麟软件中心", Package: "kylin-software-center", Description: "系统应用商店", InstallMode: "apt"},
	}
}
