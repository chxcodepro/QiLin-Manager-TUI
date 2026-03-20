package tui

import (
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

	activeTabStyle = tabStyle.
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

	selectedCellStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#0F172A")).
				Background(lipgloss.Color("#F59E0B")).
				Bold(true)

	editingCellStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F8FAFC")).
				Background(lipgloss.Color("#2563EB")).
				Bold(true)
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
	top := lipgloss.JoinHorizontal(
		lipgloss.Top,
		renderInfoCard("系统概览", m.snapshot.System.Items, width/3-2),
		"  ",
		cardStyle.Width(width-width/3-2).Render(
			highlightStyle.Render("网络总览")+"\n"+
				labelStyle.Render("默认网关:")+" "+valueStyle.Render(m.snapshot.Network.DefaultGateway)+"\n"+
				labelStyle.Render("全局 DNS:")+" "+valueStyle.Render(firstText(strings.Join(m.snapshot.Network.DNS, ", "), "-"))+"\n"+
				labelStyle.Render("保存方式:")+" "+valueStyle.Render(m.networkSaveModeText()),
		),
	)

	table := cardStyle.Width(width).Render(
		highlightStyle.Render("网卡配置表") + "\n" +
			"光标可直接选中单元格；Enter 开始编辑；Ctrl+S 保存当前行\n" +
			renderNetworkTable(m),
	)

	return lipgloss.JoinVertical(lipgloss.Left, top, table)
}

func (m model) viewDisk(width int) string {
	top := cardStyle.Width(width).Render(
		highlightStyle.Render("当前路径") + "\n" +
			valueStyle.Render(m.snapshot.Disk.Target) + "\n" +
			labelStyle.Render("上一级:") + " " + valueStyle.Render(firstText(m.snapshot.Disk.Parent, "无")) + "\n" +
			labelStyle.Render("操作:  ") + " Enter 进入 | Backspace 返回",
	)
	bottomLeft := cardStyle.Width(width/3 - 2).Render(
		highlightStyle.Render("挂载") + "\n" +
			renderList(m.snapshot.Disk.Filesystems, "暂无数据"),
	)
	bottomRight := cardStyle.Width(width-width/3-2).Render(
		highlightStyle.Render("子项占用") + "\n" +
			renderDiskEntries(m.snapshot.Disk.Entries, m.diskCursor, width-width/3-18),
	)
	return lipgloss.JoinVertical(
		lipgloss.Left,
		top,
		lipgloss.JoinHorizontal(lipgloss.Top, bottomLeft, "  ", bottomRight),
	)
}

func (m model) viewPerf(width int) string {
	summary := renderInfoCard("资源总览", m.snapshot.Perf.Summary, width/3-2)
	table := cardStyle.Width(width-width/3-2).Render(
		highlightStyle.Render("进程资源表") + "\n" +
			renderProcessTable(m.snapshot.Perf.Top),
	)
	return lipgloss.JoinHorizontal(lipgloss.Top, summary, "  ", table)
}

func (m model) viewPackages(width int) string {
	sourceCard := cardStyle.Width(width/2 - 2).Render(
		highlightStyle.Render("软件源状态") + "\n" +
			renderPackageState(m.snapshot.Packages),
	)
	actionCard := cardStyle.Width(width-width/2-2).Render(
		highlightStyle.Render("维护动作") + "\n" +
			highlightStyle.Render("[o]") + " 切换官方源\n" +
			highlightStyle.Render("[b]") + " 恢复备份源\n" +
			highlightStyle.Render("[u]") + " 更新索引\n" +
			highlightStyle.Render("[c]") + " 清理包缓存\n" +
			highlightStyle.Render("[g]") + " 清理 .log 文件\n" +
			highlightStyle.Render("[i]") + " 安装勾选的软件",
	)

	nameW, pkgW := 0, 0
	for _, app := range m.snapshot.Packages.Apps {
		if w := displayWidth(app.Name); w > nameW {
			nameW = w
		}
		if w := displayWidth(app.Package); w > pkgW {
			pkgW = w
		}
	}
	nameW += 2
	pkgW += 1

	appLines := make([]string, 0, len(m.snapshot.Packages.Apps)+1)
	appLines = append(appLines, "    "+padRight("名称", nameW)+" "+padRight("包名", pkgW)+" "+padRight("状态", 6)+" 说明")
	for idx, app := range m.snapshot.Packages.Apps {
		selected := " "
		if m.selectedApps[app.Package] {
			selected = "x"
		}

		installed := "未安装"
		if app.Installed {
			installed = "已安装"
		}

		line := "[" + selected + "] " + padRight(app.Name, nameW) + " " + padRight(app.Package, pkgW) + " " + padRight(installed, 6) + " " + app.Description
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
		lines = append(lines, "全局: Tab/Shift+Tab 切页 | r 刷新 | ? 帮助开关 | q 退出")
		switch m.active {
		case sectionOverview:
			lines = append(lines, "系统/网络页: ↑/↓ 选行 | ←/→ 选列 | Enter 编辑 | Ctrl+S 保存当前行 | Esc 取消编辑")
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

func renderInfoCard(title string, items []system.InfoItem, width int) string {
	lines := make([]string, 0, len(items)+1)
	lines = append(lines, highlightStyle.Render(title))
	maxLW := 0
	for _, item := range items {
		if w := displayWidth(item.Label); w > maxLW {
			maxLW = w
		}
	}
	for _, item := range items {
		label := padRight(item.Label, maxLW)
		lines = append(lines, labelStyle.Render(label+":")+" "+valueStyle.Render(item.Value))
	}
	return cardStyle.Width(width).Render(strings.Join(lines, "\n"))
}

func renderProcessTable(items []system.ProcessItem) string {
	lines := []string{
		padRight("PID", 8) + " " + padRight("进程", 22) + " " + padRight("CPU%", 8) + " " + "内存%",
	}
	for _, item := range items {
		lines = append(lines, padRight(item.PID, 8)+" "+padRight(truncateText(item.Name, 22), 22)+" "+padRight(item.CPU, 8)+" "+item.Memory)
	}
	return strings.Join(lines, "\n")
}

func renderPackageState(state system.PackageSection) string {
	type kv struct{ label, value string }
	items := []kv{
		{"apt 可用", boolText(state.AptReady)},
		{"sudo 可用", boolText(state.SudoReady)},
		{"备份源存在", boolText(state.BackupExists)},
	}
	maxLW := 0
	for _, item := range items {
		if w := displayWidth(item.label); w > maxLW {
			maxLW = w
		}
	}
	lines := make([]string, 0, len(items)+3)
	for _, item := range items {
		lines = append(lines, padRight(item.label, maxLW)+": "+item.value)
	}
	lines = append(lines, "", "当前 sources.list 预览:")
	lines = append(lines, state.SourceLines...)
	return strings.Join(lines, "\n")
}

func renderNetworkTable(m model) string {
	headers := []string{"网卡", "状态", "模式", "IP", "掩码", "网关", "DNS", "连接"}
	widths := []int{8, 8, 6, 15, 15, 15, 18, 12}

	lines := []string{renderTableRow(headers, widths, -1, -1, true)}
	if len(m.networkDrafts) == 0 {
		lines = append(lines, "暂无网卡")
		return strings.Join(lines, "\n")
	}

	for rowIdx, draft := range m.networkDrafts {
		values := []string{
			draft.Device,
			draft.State,
			draft.Mode,
			draft.Address,
			draft.Mask,
			draft.Gateway,
			draft.DNS,
			draft.Connection,
		}

		cells := make([]string, 0, len(values))
		for colIdx, value := range values {
			cellValue := value
			if rowIdx == m.networkCursor && colIdx == m.networkCol {
				if m.networkEdit.Active {
					cellValue = m.networkEdit.Value + "_"
				}
				cellValue = truncateText(firstText(cellValue, "-"), widths[colIdx])
				if m.networkEdit.Active {
					cells = append(cells, editingCellStyle.Width(widths[colIdx]).Render(cellValue))
				} else {
					cells = append(cells, selectedCellStyle.Width(widths[colIdx]).Render(cellValue))
				}
				continue
			}
			cells = append(cells, lipgloss.NewStyle().Width(widths[colIdx]).Render(truncateText(firstText(cellValue, "-"), widths[colIdx])))
		}
		lines = append(lines, strings.Join(cells, " "))
	}

	return strings.Join(lines, "\n")
}

func renderTableRow(values []string, widths []int, rowIdx int, cursorRow int, header bool) string {
	cells := make([]string, 0, len(values))
	for idx, value := range values {
		text := truncateText(firstText(value, "-"), widths[idx])
		style := lipgloss.NewStyle().Width(widths[idx])
		if header {
			style = style.Bold(true)
		}
		if rowIdx == cursorRow {
			style = selectedRowStyle.Width(widths[idx])
		}
		cells = append(cells, style.Render(text))
	}
	return strings.Join(cells, " ")
}

func renderDiskEntries(entries []system.DiskEntry, cursor int, nameWidth int) string {
	if len(entries) == 0 {
		return "当前目录没有子项，或需要更高权限"
	}
	lines := []string{padRight("大小", 8) + " " + padRight("类型", 4) + " " + "名称"}
	for idx, entry := range entries {
		kind := "文件"
		if entry.IsDir {
			kind = "目录"
		}
		line := padRight(entry.Size, 8) + " " + padRight(kind, 4) + " " + truncateText(entry.Name, nameWidth)
		if idx == cursor {
			line = selectedRowStyle.Render(line)
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func truncateText(value string, width int) string {
	value = strings.TrimSpace(value)
	if width <= 0 {
		return ""
	}
	if displayWidth(value) <= width {
		return value
	}
	if width <= 1 {
		return "…"
	}
	var b strings.Builder
	used := 0
	for _, r := range value {
		rw := 1
		if isWide(r) {
			rw = 2
		}
		if used+rw > width-1 {
			break
		}
		b.WriteRune(r)
		used += rw
	}
	b.WriteRune('…')
	return b.String()
}

func firstText(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func isWide(r rune) bool {
	return (r >= 0x2E80 && r <= 0x9FFF) ||
		(r >= 0xF900 && r <= 0xFAFF) ||
		(r >= 0xFE30 && r <= 0xFE4F) ||
		(r >= 0xFF01 && r <= 0xFF60) ||
		(r >= 0xFFE0 && r <= 0xFFE6)
}

func displayWidth(s string) int {
	w := 0
	for _, r := range s {
		if isWide(r) {
			w += 2
		} else {
			w++
		}
	}
	return w
}

func padRight(s string, width int) string {
	gap := width - displayWidth(s)
	if gap <= 0 {
		return s
	}
	return s + strings.Repeat(" ", gap)
}

func boolText(v bool) string {
	if v {
		return "是"
	}
	return "否"
}

func (m model) networkSaveModeText() string {
	if m.snapshot.Network.NMCLIAvailable {
		return "nmcli 持久化保存"
	}
	return "只读，缺少 nmcli"
}
