package system

import (
	"fmt"
	"strings"
)

type Action struct {
	Title   string
	Confirm string
	Command string
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
	}
}

func AptUpdateAction() Action {
	return Action{
		Title:   "更新软件索引",
		Confirm: "会执行 apt-get update，继续吗？",
		Command: buildRootCommand("apt-get update"),
	}
}

func CleanAptCacheAction() Action {
	return Action{
		Title:   "清理包缓存",
		Confirm: "会执行 apt-get clean，继续吗？",
		Command: buildRootCommand("apt-get clean"),
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
	}
}

func buildRootCommand(script string) string {
	escaped := shellQuote(script)
	if commandExists("sudo") {
		return "sudo sh -lc " + escaped
	}
	return "sh -lc " + escaped
}
