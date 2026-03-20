package system

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type InfoItem struct {
	Label string
	Value string
}

type ProcessItem struct {
	PID     string
	Name    string
	CPU     string
	Memory  string
}

type AppState struct {
	AppInfo
	Installed bool
}

type SystemSection struct {
	Items []InfoItem
}

type NetworkSection struct {
	Interfaces []string
	Routes     []string
	DNS        []string
}

type DiskSection struct {
	Target      string
	Filesystems []string
	TopDirs     []string
}

type PerfSection struct {
	Summary  []InfoItem
	TopCPU   []ProcessItem
	TopMemory []ProcessItem
}

type PackageSection struct {
	SourceLines  []string
	BackupExists bool
	AptReady     bool
	SudoReady    bool
	Apps         []AppState
}

type Snapshot struct {
	GeneratedAt time.Time
	System      SystemSection
	Network     NetworkSection
	Disk        DiskSection
	Perf        PerfSection
	Packages    PackageSection
}

func CollectSnapshot(diskTarget string, apps []AppInfo) Snapshot {
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()

	return Snapshot{
		GeneratedAt: time.Now(),
		System:      collectSystem(ctx),
		Network:     collectNetwork(ctx),
		Disk:        collectDisk(ctx, diskTarget),
		Perf:        collectPerf(ctx),
		Packages:    collectPackages(ctx, apps),
	}
}

func collectSystem(ctx context.Context) SystemSection {
	osName := firstNonEmpty(
		parseOSRelease("PRETTY_NAME"),
		strings.TrimSpace(runShell(ctx, "uname -o 2>/dev/null")),
		runtime.GOOS,
	)

	hostname := strings.TrimSpace(runShell(ctx, "hostname 2>/dev/null"))
	if hostname == "" {
		hostname = "未知"
	}

	kernel := strings.TrimSpace(runShell(ctx, "uname -sr 2>/dev/null"))
	if kernel == "" {
		kernel = "未知"
	}

	arch := strings.TrimSpace(runShell(ctx, "uname -m 2>/dev/null"))
	if arch == "" {
		arch = runtime.GOARCH
	}

	uptime := strings.TrimSpace(runShell(ctx, "uptime -p 2>/dev/null"))
	if uptime == "" {
		uptime = readUptime()
	}

	currentUser := strings.TrimSpace(runShell(ctx, "whoami 2>/dev/null"))
	if currentUser == "" {
		currentUser = "未知"
	}

	desktop := strings.TrimSpace(os.Getenv("XDG_CURRENT_DESKTOP"))
	if desktop == "" {
		desktop = "未检测到"
	}

	return SystemSection{
		Items: []InfoItem{
			{Label: "系统", Value: osName},
			{Label: "主机名", Value: hostname},
			{Label: "内核", Value: kernel},
			{Label: "架构", Value: arch},
			{Label: "运行时长", Value: uptime},
			{Label: "当前用户", Value: currentUser},
			{Label: "桌面环境", Value: desktop},
		},
	}
}

func collectNetwork(ctx context.Context) NetworkSection {
	interfaces := cleanLines(runShell(ctx, "ip -brief addr 2>/dev/null || ifconfig 2>/dev/null"))
	if len(interfaces) == 0 {
		interfaces = []string{"未获取到网卡信息"}
	}

	routes := cleanLines(runShell(ctx, "ip route 2>/dev/null"))
	if len(routes) == 0 {
		routes = []string{"未获取到路由信息"}
	}

	dns := readFileLines("/etc/resolv.conf", 8, false)
	if len(dns) == 0 {
		dns = []string{"未获取到 DNS 配置"}
	}

	return NetworkSection{
		Interfaces: interfaces,
		Routes:     routes,
		DNS:        dns,
	}
}

func collectDisk(ctx context.Context, target string) DiskSection {
	if strings.TrimSpace(target) == "" {
		target = "/"
	}

	filesystems := cleanLines(runShell(ctx, fmt.Sprintf("df -h %s 2>/dev/null || true", shellQuote(target))))
	if len(filesystems) == 0 {
		filesystems = []string{"未获取到磁盘挂载信息"}
	}

	topDirs := cleanLines(runShell(ctx, fmt.Sprintf("du -xh --max-depth=1 %s 2>/dev/null | sort -hr | head -n 12", shellQuote(target))))
	if len(topDirs) == 0 {
		topDirs = []string{"未获取到目录占用信息，可能需要更高权限"}
	}

	return DiskSection{
		Target:      target,
		Filesystems: filesystems,
		TopDirs:     topDirs,
	}
}

func collectPerf(ctx context.Context) PerfSection {
	loadAvg := strings.TrimSpace(runShell(ctx, "cat /proc/loadavg 2>/dev/null | awk '{print $1\" / \"$2\" / \"$3}'"))
	if loadAvg == "" {
		loadAvg = "未知"
	}

	cpuUsage := strings.TrimSpace(runShell(ctx, `top -bn1 2>/dev/null | awk -F',' '/Cpu\(s\)/ {gsub(/ id.*/, "", $4); gsub(/%?us/, "", $1); gsub(/^.*: */, "", $1); print $1 + $2}'`))
	if cpuUsage == "" {
		cpuUsage = "未知"
	} else {
		cpuUsage += "%"
	}

	memUsage := strings.TrimSpace(runShell(ctx, `free -m 2>/dev/null | awk '/Mem:/ {printf "%sMB / %sMB (%.1f%%)", $3, $2, ($3/$2)*100}'`))
	if memUsage == "" {
		memUsage = "未知"
	}

	swapUsage := strings.TrimSpace(runShell(ctx, `free -m 2>/dev/null | awk '/Swap:/ {printf "%sMB / %sMB", $3, $2}'`))
	if swapUsage == "" {
		swapUsage = "未知"
	}

	topCPU := parseProcessLines(cleanLines(runShell(ctx, "ps -eo pid,comm,%cpu,%mem --sort=-%cpu | head -n 8")))
	if len(topCPU) == 0 {
		topCPU = []ProcessItem{{PID: "-", Name: "未获取到进程数据", CPU: "-", Memory: "-"}}
	}

	topMemory := parseProcessLines(cleanLines(runShell(ctx, "ps -eo pid,comm,%cpu,%mem --sort=-%mem | head -n 8")))
	if len(topMemory) == 0 {
		topMemory = []ProcessItem{{PID: "-", Name: "未获取到进程数据", CPU: "-", Memory: "-"}}
	}

	return PerfSection{
		Summary: []InfoItem{
			{Label: "CPU 总览", Value: cpuUsage},
			{Label: "负载", Value: loadAvg},
			{Label: "内存", Value: memUsage},
			{Label: "交换分区", Value: swapUsage},
		},
		TopCPU:    topCPU,
		TopMemory: topMemory,
	}
}

func collectPackages(ctx context.Context, apps []AppInfo) PackageSection {
	sourceLines := readFileLines("/etc/apt/sources.list", 8, true)
	if len(sourceLines) == 0 {
		sourceLines = []string{"未检测到 /etc/apt/sources.list"}
	}

	appStates := make([]AppState, 0, len(apps))
	for _, app := range apps {
		appStates = append(appStates, AppState{
			AppInfo:   app,
			Installed: packageInstalled(ctx, app.Package),
		})
	}

	return PackageSection{
		SourceLines:  sourceLines,
		BackupExists: fileExists("/etc/apt/sources.list.bak"),
		AptReady:     commandExists("apt-get"),
		SudoReady:    commandExists("sudo") || strings.TrimSpace(runShell(ctx, "id -u 2>/dev/null")) == "0",
		Apps:         appStates,
	}
}

func runShell(ctx context.Context, script string) string {
	cmd := exec.CommandContext(ctx, "sh", "-lc", script)
	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil && text == "" {
		return ""
	}
	return text
}

func cleanLines(text string) []string {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		result = append(result, line)
	}
	return result
}

func parseOSRelease(key string) string {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, key+"=") {
			continue
		}
		value := strings.TrimPrefix(line, key+"=")
		return strings.Trim(value, `"`)
	}
	return ""
}

func readUptime() string {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return "未知"
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return "未知"
	}
	secondsText := fields[0]
	dot := strings.IndexByte(secondsText, '.')
	if dot > 0 {
		secondsText = secondsText[:dot]
	}
	seconds, err := time.ParseDuration(secondsText + "s")
	if err != nil {
		return "未知"
	}
	days := int(seconds.Hours()) / 24
	hours := int(seconds.Hours()) % 24
	minutes := int(seconds.Minutes()) % 60
	parts := make([]string, 0, 3)
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%d天", days))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%d小时", hours))
	}
	if minutes > 0 {
		parts = append(parts, fmt.Sprintf("%d分钟", minutes))
	}
	if len(parts) == 0 {
		return "不到 1 分钟"
	}
	return strings.Join(parts, "")
}

func parseProcessLines(lines []string) []ProcessItem {
	result := make([]ProcessItem, 0, len(lines))
	for _, line := range lines {
		if strings.HasPrefix(strings.ToUpper(line), "PID ") || strings.EqualFold(line, "PID COMMAND %CPU %MEM") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		result = append(result, ProcessItem{
			PID:    fields[0],
			Name:   fields[1],
			CPU:    fields[2],
			Memory: fields[3],
		})
	}
	return result
}

func packageInstalled(ctx context.Context, name string) bool {
	if name == "" {
		return false
	}
	cmd := exec.CommandContext(ctx, "sh", "-lc", fmt.Sprintf("dpkg -s %s >/dev/null 2>&1", shellQuote(name)))
	return cmd.Run() == nil
}

func readFileLines(path string, limit int, skipComments bool) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	result := make([]string, 0, limit)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if skipComments && strings.HasPrefix(line, "#") {
			continue
		}
		result = append(result, line)
		if limit > 0 && len(result) >= limit {
			break
		}
	}
	return result
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func HomeDir() string {
	if dir, err := os.UserHomeDir(); err == nil {
		return filepath.Clean(dir)
	}
	return "/root"
}
