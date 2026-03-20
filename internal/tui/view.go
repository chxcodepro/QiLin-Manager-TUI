package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/chxcodepro/qilin-manager-tui/internal/system"
)

var (
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
		m.renderTab(sectionOverview, "系统/网络"),
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
	case sectionOverview:
		return panelStyle.Width(width).Render(m.viewOverview(bodyWidth))
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

func (m model) viewOverview(width int) string {
	systemCard := renderInfoCard("系统概览", m.snapshot.System.Items, width/3-2)
	networkList := cardStyle.Width(width/3 - 2).Render(
		highlightStyle.Render("网卡列表") + "\n" +
			renderNetworkList(m.snapshot.Network.Interfaces, m.networkCursor),
	)
	networkDetail := cardStyle.Width(width-width/3-width/3-4).Render(
		highlightStyle.Render("网络详情") + "\n" +
			renderNetworkDetail(m.snapshot.Network, m.currentInterface()),
	)
	return lipgloss.JoinHorizontal(lipgloss.Top, systemCard, "  ", networkList, "  ", networkDetail)
}

func (m model) viewDisk(width int) string {
	info := []system.InfoItem{
		{Label: "当前路径", Value: m.snapshot.Disk.Target},
		{Label: "上一级", Value: firstText(m.snapshot.Disk.Parent, "无")},
		{Label: "操作", Value: "Enter 进入 / Backspace 返回"},
	}
	left := renderInfoCard("分析目标", info, width/3)
	center := cardStyle.Width(width/3).Render(
		highlightStyle.Render("挂载情况") + "\n" +
			renderList(m.snapshot.Disk.Filesystems, "暂无数据"),
	)
	right := cardStyle.Width(width-width/3-width/3-4).Render(
		highlightStyle.Render("当前目录子项占用") + "\n" +
			renderDiskEntries(m.snapshot.Disk.Entries, m.diskCursor),
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
	lines := []string{"状态: " + m.status}
	if m.showHelp {
		lines = append(lines, "全局: ←/→ 切页 | r 刷新 | ? 帮助 | q 退出")
		switch m.active {
		case sectionOverview:
			lines = append(lines, "系统/网络页: ↑/↓ 选网卡 | e 编辑并保存")
		case sectionDisk:
			lines = append(lines, "磁盘页: ↑/↓ 选项 | Enter 进入目录 | Backspace 返回")
		case sectionPackage:
			lines = append(lines, "软件页: ↑/↓ 选中 | 空格勾选")
		}
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

func (m model) viewNetworkForm() string {
	if m.form == nil {
		return ""
	}
	lines := []string{
		highlightStyle.Render("编辑网卡配置"),
		labelStyle.Render("网卡: "+m.form.iface.Name+" | 连接: "+m.form.iface.Connection),
		"",
	}
	for idx, field := range m.form.fields {
		line := fmt.Sprintf("%-10s %s", field.Label, field.Value)
		if idx == m.form.cursor {
			line = selectedRowStyle.Render(line)
		}
		lines = append(lines, line)
	}
	lines = append(lines, "", "↑/↓ 或 Tab 切换字段，左右切换模式，Ctrl+S 保存，Esc 取消")
	return cardStyle.
		Width(min(max(m.width/2, 48), 86)).
		BorderForeground(lipgloss.Color("#F59E0B")).
		Render(strings.Join(lines, "\n"))
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

func renderNetworkList(items []system.NetworkInterface, cursor int) string {
	if len(items) == 0 {
		return "暂无网卡"
	}
	lines := make([]string, 0, len(items))
	for idx, item := range items {
		line := fmt.Sprintf("%-10s %-10s %-15s %-15s", item.Name, item.State, firstText(item.IPv4, "-"), firstText(item.Mask, "-"))
		if idx == cursor {
			line = selectedRowStyle.Render(line)
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func renderNetworkDetail(network system.NetworkSection, iface *system.NetworkInterface) string {
	lines := []string{
		fmt.Sprintf("默认网关: %s", network.DefaultGateway),
		fmt.Sprintf("全局 DNS: %s", strings.Join(network.DNS, ", ")),
	}
	if network.NMCLIAvailable {
		lines = append(lines, "可编辑方式: nmcli 持久化保存")
	} else {
		lines = append(lines, "可编辑方式: 当前系统缺少 nmcli，只能查看")
	}
	lines = append(lines, "")
	if iface == nil {
		lines = append(lines, "请先选中一块网卡")
		return strings.Join(lines, "\n")
	}
	lines = append(lines,
		fmt.Sprintf("网卡: %s", iface.Name),
		fmt.Sprintf("连接名: %s", firstText(iface.Connection, "无")),
		fmt.Sprintf("状态: %s", firstText(iface.State, "-")),
		fmt.Sprintf("IPv4: %s", firstText(iface.IPv4, "-")),
		fmt.Sprintf("子网掩码: %s", firstText(iface.Mask, "-")),
		fmt.Sprintf("默认网关: %s", firstText(iface.Gateway, "-")),
		fmt.Sprintf("DNS: %s", firstText(strings.Join(iface.DNS, ", "), "-")),
		fmt.Sprintf("模式: %s", firstText(iface.Method, "未知")),
	)
	return strings.Join(lines, "\n")
}

func renderDiskEntries(entries []system.DiskEntry, cursor int) string {
	if len(entries) == 0 {
		return "当前目录没有子项，或需要更高权限"
	}
	lines := make([]string, 0, len(entries))
	for idx, entry := range entries {
		kind := "文件"
		if entry.IsDir {
			kind = "目录"
		}
		line := fmt.Sprintf("%-8s %-8s %s", entry.Size, kind, entry.Name)
		if idx == cursor {
			line = selectedRowStyle.Render(line)
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func firstText(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func (m model) currentInterface() *system.NetworkInterface {
	if len(m.snapshot.Network.Interfaces) == 0 || m.networkCursor < 0 || m.networkCursor >= len(m.snapshot.Network.Interfaces) {
		return nil
	}
	iface := m.snapshot.Network.Interfaces[m.networkCursor]
	return &iface
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
