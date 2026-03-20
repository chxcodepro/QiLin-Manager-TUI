package tui

import (
	"fmt"
	"strings"

	"github.com/chxcodepro/qilin-manager-tui/internal/system"
	"github.com/charmbracelet/lipgloss"
)

var (
	pageStyle = lipgloss.NewStyle().
			Padding(1, 2).
			Width(120)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#F8FAFC")).
			Background(lipgloss.Color("#0F172A")).
			Padding(0, 1)

	tabStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(lipgloss.Color("#CBD5E1"))

	activeTabStyle = tabStyle.Copy().
			Bold(true).
			Foreground(lipgloss.Color("#0F172A")).
			Background(lipgloss.Color("#F59E0B"))

	panelStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#334155")).
			Padding(1, 2)

	cardStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#475569")).
			Padding(1, 2)

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#94A3B8"))

	valueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E2E8F0"))

	highlightStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F59E0B")).
			Bold(true)

	selectedRowStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#0F172A")).
				Background(lipgloss.Color("#FCD34D"))
)

func (m model) viewHeader() string {
	tabs := []string{
		m.renderTab(sectionSystem, "系统信息"),
		m.renderTab(sectionNetwork, "网络信息"),
		m.renderTab(sectionDisk, "磁盘分析"),
		m.renderTab(sectionPerf, "CPU/内存"),
		m.renderTab(sectionPackage, "软件维护"),
	}

	right := "版本 " + m.version
	if !m.snapshot.GeneratedAt.IsZero() {
		right += " | 更新于 " + m.snapshot.GeneratedAt.Format("15:04:05")
	}
	if m.loading {
		right += " | 刷新中"
	}

	headerLine := lipgloss.JoinHorizontal(lipgloss.Top, titleStyle.Render("银河麒麟 TUI 管理面板"), "  ", strings.Join(tabs, " "))
	return panelStyle.Width(max(m.width-4, 60)).Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			headerLine,
			labelStyle.Render(right),
		),
	)
}

func (m model) renderTab(target section, title string) string {
	if m.active == target {
		return activeTabStyle.Render(title)
	}
	return tabStyle.Render(title)
}

func (m model) viewBody() string {
	width := max(m.width-4, 60)
	bodyWidth := width - 6
	switch m.active {
	case sectionSystem:
		return panelStyle.Width(width).Render(m.viewSystem(bodyWidth))
	case sectionNetwork:
		return panelStyle.Width(width).Render(m.viewNetwork(bodyWidth))
	case sectionDisk:
		return panelStyle.Width(width).Render(m.viewDisk(bodyWidth))
	case sectionPerf:
		return panelStyle.Width(width).Render(m.viewPerf(bodyWidth))
	case sectionPackage:
		return panelStyle.Width(width).Render(m.viewPackages(bodyWidth))
	default:
		return panelStyle.Width(width).Render("未知页面")
	}
}

func (m model) viewSystem(width int) string {
	left := renderInfoCard("系统概览", m.snapshot.System.Items, width/2-2)
	rightLines := []string{
		"这个页面主要看整机基本信息。",
		"如果系统识别不准，先检查 /etc/os-release。",
		"数据来源以系统命令和 /proc 为主。",
	}
	right := cardStyle.Width(width - width/2 - 2).Render(renderList(rightLines, "暂无说明"))
	return lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", right)
}

func (m model) viewNetwork(width int) string {
	left := cardStyle.Width(width/2 - 2).Render(
		highlightStyle.Render("网卡与地址") + "\n" +
			renderList(m.snapshot.Network.Interfaces, "暂无数据"),
	)
	rightTop := cardStyle.Width(width-width/2-2).Render(
		highlightStyle.Render("默认路由") + "\n" +
			renderList(m.snapshot.Network.Routes, "暂无数据"),
	)
	rightBottom := cardStyle.Width(width-width/2-2).Render(
		highlightStyle.Render("DNS 配置") + "\n" +
			renderList(m.snapshot.Network.DNS, "暂无数据"),
	)
	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		left,
		"  ",
		lipgloss.JoinVertical(lipgloss.Left, rightTop, rightBottom),
	)
}

func (m model) viewDisk(width int) string {
	info := []system.InfoItem{
		{Label: "当前路径", Value: m.snapshot.Disk.Target},
		{Label: "切换按键", Value: "[ / ]"},
	}
	left := renderInfoCard("分析目标", info, width/3)
	center := cardStyle.Width(width/3).Render(
		highlightStyle.Render("挂载情况") + "\n" +
			renderList(m.snapshot.Disk.Filesystems, "暂无数据"),
	)
	right := cardStyle.Width(width-width/3-width/3-4).Render(
		highlightStyle.Render("目录占用 Top") + "\n" +
			renderList(m.snapshot.Disk.TopDirs, "暂无数据"),
	)
	return lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", center, "  ", right)
}

func (m model) viewPerf(width int) string {
	summary := renderInfoCard("资源总览", m.snapshot.Perf.Summary, width/3)
	topCPU := cardStyle.Width(width/3).Render(
		highlightStyle.Render("CPU Top 进程") + "\n" +
			renderProcessTable(m.snapshot.Perf.TopCPU),
	)
	topMem := cardStyle.Width(width-width/3-width/3-4).Render(
		highlightStyle.Render("内存 Top 进程") + "\n" +
			renderProcessTable(m.snapshot.Perf.TopMemory),
	)
	return lipgloss.JoinHorizontal(lipgloss.Top, summary, "  ", topCPU, "  ", topMem)
}

func (m model) viewPackages(width int) string {
	sourceCard := cardStyle.Width(width/2 - 2).Render(
		highlightStyle.Render("软件源状态") + "\n" +
			renderPackageState(m.snapshot.Packages),
	)
	actionCard := cardStyle.Width(width-width/2-2).Render(
		highlightStyle.Render("维护动作") + "\n" +
			"o 切换官方源\n" +
			"b 恢复备份源\n" +
			"u 更新索引\n" +
			"c 清理包缓存\n" +
			"g 清理 .log 文件\n" +
			"i 安装勾选的软件",
	)

	appLines := make([]string, 0, len(m.snapshot.Packages.Apps))
	for idx, app := range m.snapshot.Packages.Apps {
		selected := " "
		if m.selectedApps[app.Package] {
			selected = "x"
		}

		installed := "未安装"
		if app.Installed {
			installed = "已安装"
		}

		line := fmt.Sprintf("[%s] %-18s %-22s %-8s %s", selected, app.Name, app.Package, installed, app.Description)
		if idx == m.appCursor {
			line = selectedRowStyle.Render(line)
		}
		appLines = append(appLines, line)
	}

	appCard := cardStyle.Width(width).Render(
		highlightStyle.Render("软件清单") + "\n" +
			"上下移动，空格勾选\n" +
			renderList(appLines, "暂无软件"),
	)

	return lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Top, sourceCard, "  ", actionCard),
		appCard,
	)
}

func (m model) viewFooter() string {
	width := max(m.width-4, 60)
	lines := []string{
		"状态: " + m.status,
	}
	if m.showHelp {
		lines = append(lines, "全局: ←/→ 切页 | r 刷新 | ? 帮助 | q 退出")
		lines = append(lines, "磁盘页: [ / ] 切换路径 | 软件页: ↑/↓ 选中 | 空格勾选")
	}
	return panelStyle.Width(width).Render(strings.Join(lines, "\n"))
}

func (m model) viewConfirmDialog() string {
	if m.confirming == nil {
		return ""
	}
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		highlightStyle.Render("请确认"),
		m.confirming.action.Title,
		labelStyle.Render(m.confirming.action.Confirm),
		"",
		"按 y 或 Enter 确认，按 n 或 Esc 取消",
	)
	return cardStyle.
		Width(min(max(m.width/2, 40), 80)).
		BorderForeground(lipgloss.Color("#F59E0B")).
		Render(content)
}

func renderInfoCard(title string, items []system.InfoItem, width int) string {
	lines := make([]string, 0, len(items)+1)
	lines = append(lines, highlightStyle.Render(title))
	for _, item := range items {
		lines = append(lines, fmt.Sprintf("%s %s", labelStyle.Render(item.Label+":"), valueStyle.Render(item.Value)))
	}
	return cardStyle.Width(width).Render(strings.Join(lines, "\n"))
}

func renderProcessTable(items []system.ProcessItem) string {
	lines := []string{
		fmt.Sprintf("%-8s %-18s %-8s %-8s", "PID", "进程", "CPU%", "内存%"),
	}
	for _, item := range items {
		lines = append(lines, fmt.Sprintf("%-8s %-18s %-8s %-8s", item.PID, item.Name, item.CPU, item.Memory))
	}
	return strings.Join(lines, "\n")
}

func renderPackageState(state system.PackageSection) string {
	lines := []string{
		fmt.Sprintf("apt 可用: %t", state.AptReady),
		fmt.Sprintf("sudo 可用: %t", state.SudoReady),
		fmt.Sprintf("备份源存在: %t", state.BackupExists),
		"",
		"当前 sources.list 预览:",
	}
	lines = append(lines, state.SourceLines...)
	return strings.Join(lines, "\n")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
