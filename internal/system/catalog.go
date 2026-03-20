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
		{Name: "奔图打印机驱动", Package: "pantum", Description: "国产打印机驱动", InstallMode: "apt"},
		{Name: "搜狗输入法", Package: "sogoupinyin", Description: "输入法", InstallMode: "apt"},
		{Name: "360安全浏览器", Package: "browser360-cn-stable", Description: "浏览器", InstallMode: "apt"},
		{Name: "VLC", Package: "vlc", Description: "视频播放器", InstallMode: "apt"},
		{Name: "GIMP", Package: "gimp", Description: "图片编辑", InstallMode: "apt"},
		{Name: "向日葵远程控制", Package: "sunloginclient", Description: "远程桌面", InstallMode: "apt"},
		{Name: "LibreOffice", Package: "libreoffice", Description: "开源办公套件", InstallMode: "apt"},
		{Name: "GParted", Package: "gparted", Description: "磁盘分区工具", InstallMode: "apt"},
		{Name: "远程桌面", Package: "remmina", Description: "远程连接客户端", InstallMode: "apt"},
		{Name: "文本编辑器", Package: "pluma", Description: "系统文本编辑器", InstallMode: "apt"},
	}
}
