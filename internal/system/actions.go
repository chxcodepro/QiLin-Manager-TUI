package system

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

type Action struct {
	Title   string
	Confirm string
	Command string
	Preview string
}

type NetworkConfig struct {
	Connection string
	Device     string
	Method     string
	Address    string
	Mask       string
	Gateway    string
	DNS        string
}

func OfficialSourceAction() Action {
	lines := []string{
		"deb http://archive.kylinos.cn/kylin/KYLIN-ALL 10.1 main restricted universe multiverse",
		"deb http://archive.kylinos.cn/kylin/partner juniper main",
	}

	script := fmt.Sprintf(`set -e
if [ ! -f /etc/apt/sources.list.bak ]; then
  cp /etc/apt/sources.list /etc/apt/sources.list.bak
fi
cat > /etc/apt/sources.list <<'EOF'
%s
EOF
apt-get update`, strings.Join(lines, "\n"))

	return Action{
		Title:   "切换到银河麒麟 V10 官方源",
		Confirm: "会备份当前 sources.list，并切换到内置官方源模板，继续吗？",
		Command: buildRootCommand(script),
		Preview: "cp sources.list{,.bak} && 写入官方源 && apt-get update",
	}
}

func RestoreSourceAction() Action {
	script := `set -e
test -f /etc/apt/sources.list.bak
cp /etc/apt/sources.list.bak /etc/apt/sources.list
apt-get update`

	return Action{
		Title:   "恢复备份源",
		Confirm: "会用 /etc/apt/sources.list.bak 覆盖当前源并执行更新，继续吗？",
		Command: buildRootCommand(script),
		Preview: "cp sources.list.bak sources.list && apt-get update",
	}
}

func AptUpdateAction() Action {
	return Action{
		Title:   "更新软件索引",
		Confirm: "会执行 apt-get update，继续吗？",
		Command: buildRootCommand("apt-get update"),
		Preview: "apt-get update",
	}
}

func CleanAptCacheAction() Action {
	return Action{
		Title:   "清理包缓存",
		Confirm: "会执行 apt-get clean，继续吗？",
		Command: buildRootCommand("apt-get clean"),
		Preview: "apt-get clean",
	}
}

func CleanLogsAction() Action {
	home := shellQuote(HomeDir())
	script := fmt.Sprintf(`set -e
find /var/log -type f -name '*.log' -exec truncate -s 0 {} +
find %s -type f -name '*.log' -exec truncate -s 0 {} + 2>/dev/null || true`, home)

	return Action{
		Title:   "清理日志文件",
		Confirm: "会把 /var/log 和当前用户目录下的 .log 文件清空内容，继续吗？",
		Command: buildRootCommand(script),
		Preview: "find /var/log -name '*.log' -exec truncate -s 0 {} +",
	}
}

func InstallAppsAction(packages []string) Action {
	quoted := make([]string, 0, len(packages))
	for _, pkg := range packages {
		pkg = strings.TrimSpace(pkg)
		if pkg == "" {
			continue
		}
		quoted = append(quoted, shellQuote(pkg))
	}

	return Action{
		Title:   "安装选中的软件",
		Confirm: "会通过 apt-get install 安装当前勾选的软件，继续吗？",
		Command: buildRootCommand("apt-get install -y " + strings.Join(quoted, " ")),
		Preview: "apt-get install -y " + strings.Join(quoted, " "),
	}
}

func UninstallAppsAction(packages []string) Action {
	quoted := make([]string, 0, len(packages))
	for _, pkg := range packages {
		pkg = strings.TrimSpace(pkg)
		if pkg == "" {
			continue
		}
		quoted = append(quoted, shellQuote(pkg))
	}

	return Action{
		Title:   "卸载选中的软件",
		Confirm: "会通过 apt-get remove 卸载当前勾选的已安装软件，继续吗？",
		Command: buildRootCommand("apt-get remove -y " + strings.Join(quoted, " ")),
		Preview: "apt-get remove -y " + strings.Join(quoted, " "),
	}
}

func ConfigureNetworkAction(cfg NetworkConfig) (Action, error) {
	if !commandExists("nmcli") {
		return Action{}, fmt.Errorf("系统里没有 nmcli")
	}
	if strings.TrimSpace(cfg.Connection) == "" {
		return Action{}, fmt.Errorf("未找到可保存的网络连接")
	}

	method := strings.TrimSpace(cfg.Method)
	if method != "auto" && method != "manual" {
		return Action{}, fmt.Errorf("不支持的配置模式")
	}

	lines := []string{"set -e"}
	connection := shellQuote(cfg.Connection)

	if method == "auto" {
		lines = append(lines,
			"nmcli con mod "+connection+" ipv4.method auto ipv4.addresses \"\" ipv4.gateway \"\" ipv4.dns \"\" connection.autoconnect yes",
			"nmcli con up "+connection,
		)
		return Action{
			Title:   "保存网卡配置",
			Confirm: fmt.Sprintf("会把网卡 %s 切换到 DHCP 并立即重连，继续吗？", cfg.Device),
			Command: buildRootCommand(strings.Join(lines, "\n")),
		}, nil
	}

	address := strings.TrimSpace(cfg.Address)
	mask := strings.TrimSpace(cfg.Mask)
	if address == "" || mask == "" {
		return Action{}, fmt.Errorf("静态模式下 IP 和子网掩码不能为空")
	}

	if net.ParseIP(address) == nil {
		return Action{}, fmt.Errorf("IP 地址格式不对: %s", address)
	}

	prefix, err := maskToPrefix(mask)
	if err != nil {
		return Action{}, err
	}

	lines = append(lines,
		"nmcli con mod "+connection+" ipv4.method manual ipv4.addresses "+shellQuote(address+"/"+prefix)+" connection.autoconnect yes",
	)

	if strings.TrimSpace(cfg.Gateway) != "" {
		if net.ParseIP(strings.TrimSpace(cfg.Gateway)) == nil {
			return Action{}, fmt.Errorf("网关地址格式不对: %s", cfg.Gateway)
		}
		lines = append(lines, "nmcli con mod "+connection+" ipv4.gateway "+shellQuote(strings.TrimSpace(cfg.Gateway)))
	} else {
		lines = append(lines, "nmcli con mod "+connection+" ipv4.gateway \"\"")
	}

	dns := strings.Join(splitDNS(cfg.DNS), " ")
	if dns != "" {
		for _, d := range splitDNS(cfg.DNS) {
			if net.ParseIP(d) == nil {
				return Action{}, fmt.Errorf("DNS 地址格式不对: %s", d)
			}
		}
		lines = append(lines, "nmcli con mod "+connection+" ipv4.dns "+shellQuote(dns))
	} else {
		lines = append(lines, "nmcli con mod "+connection+" ipv4.dns \"\"")
	}

	lines = append(lines, "nmcli con up "+connection)

	return Action{
		Title:   "保存网卡配置",
		Confirm: fmt.Sprintf("会把网卡 %s 改成静态地址 %s/%s，并立即重连，继续吗？", cfg.Device, address, mask),
		Command: buildRootCommand(strings.Join(lines, "\n")),
	}, nil
}

func buildRootCommand(script string) string {
	escaped := shellQuote(script)
	if commandExists("sudo") {
		return "sudo sh -lc " + escaped
	}
	return "sh -lc " + escaped
}

func splitDNS(value string) []string {
	value = strings.ReplaceAll(value, ",", " ")
	fields := strings.Fields(value)
	result := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		result = append(result, field)
	}
	return result
}

func maskToPrefix(mask string) (string, error) {
	ip := net.ParseIP(strings.TrimSpace(mask)).To4()
	if ip == nil {
		return "", fmt.Errorf("子网掩码格式不对")
	}
	ones, bits := net.IPMask(ip).Size()
	if bits != 32 || ones < 0 {
		return "", fmt.Errorf("子网掩码格式不对")
	}
	return strconv.Itoa(ones), nil
}
