package tui

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/chxcodepro/qilin-manager-tui/internal/system"
)

type section int

const (
	sectionOverview section = iota
	sectionDisk
	sectionPerf
	sectionPackage
)

const (
	networkColName = iota
	networkColState
	networkColMode
	networkColIP
	networkColMask
	networkColGateway
	networkColDNS
	networkColConnection
)

type snapshotMsg struct {
	snapshot system.Snapshot
}

type actionDoneMsg struct {
	title string
	err   error
}

type tickMsg time.Time

type searchResultsMsg struct {
	results []system.AppState
}

type pendingAction struct {
	action system.Action
}

type networkDraft struct {
	Device     string
	Connection string
	State      string
	Mode       string
	Address    string
	Mask       string
	Gateway    string
	DNS        string
}

type networkEdit struct {
	Active bool
	Value  string
}

type model struct {
	version       string
	active        section
	width         int
	height        int
	ready         bool
	loading       bool
	showHelp      bool
	diskPath      string
	diskCursor    int
	networkCursor int
	networkCol    int
	networkDrafts []networkDraft
	networkEdit   networkEdit
	apps          []system.AppInfo
	appCursor     int
	selectedApps  map[string]bool
	searchMode    bool
	searchInput   string
	searchResults []system.AppState
	showSearch    bool
	snapshot      system.Snapshot
	status        string
	confirming    *pendingAction
}

func Run(version string) error {
	p := tea.NewProgram(newModel(version), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func newModel(version string) model {
	return model{
		version:      version,
		active:       sectionOverview,
		diskPath:     "/",
		apps:         system.DefaultApps(),
		selectedApps: map[string]bool{},
		showHelp:     true,
		loading:      true,
		status:       "正在加载系统信息",
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.refreshCmd(), tickCmd())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		return m, nil

	case snapshotMsg:
		m.snapshot = msg.snapshot
		m.loading = false
		m.syncNetworkDrafts()
		if len(m.snapshot.Disk.Entries) == 0 {
			m.diskCursor = 0
		} else if m.diskCursor >= len(m.snapshot.Disk.Entries) {
			m.diskCursor = len(m.snapshot.Disk.Entries) - 1
		}
		m.status = "数据已刷新"
		return m, nil

	case tickMsg:
		if m.confirming != nil || m.networkEdit.Active || m.searchMode {
			return m, tickCmd()
		}
		m.loading = true
		return m, tea.Batch(m.refreshCmd(), tickCmd())

	case actionDoneMsg:
		m.confirming = nil
		if msg.err != nil {
			m.status = fmt.Sprintf("%s失败: %v", msg.title, msg.err)
		} else {
			m.status = msg.title + "完成"
		}
		m.loading = true
		return m, m.refreshCmd()

	case searchResultsMsg:
		m.searchResults = msg.results
		m.showSearch = true
		m.appCursor = 0
		if len(msg.results) == 0 {
			m.status = "未找到匹配的软件包"
		} else {
			m.status = fmt.Sprintf("找到 %d 个结果", len(msg.results))
		}
		return m, nil

	case tea.KeyMsg:
		if m.confirming != nil {
			switch msg.String() {
			case "y", "Y", "enter":
				m.status = "正在执行: " + m.confirming.action.Title
				return m, execActionCmd(m.confirming.action)
			case "n", "N", "esc":
				m.confirming = nil
				m.status = "已取消操作"
				return m, nil
			}
			return m, nil
		}

		if m.networkEdit.Active {
			return m.updateNetworkEdit(msg)
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab":
			m.nextSection()
			return m, nil
		case "shift+tab":
			m.prevSection()
			return m, nil
		case "left", "h":
			if m.active == sectionOverview {
				return m.updateOverview(msg)
			}
			m.prevSection()
			return m, nil
		case "right", "l":
			if m.active == sectionOverview {
				return m.updateOverview(msg)
			}
			m.nextSection()
			return m, nil
		case "r":
			m.loading = true
			m.status = "正在手动刷新"
			return m, m.refreshCmd()
		case "?":
			m.showHelp = !m.showHelp
			if m.showHelp {
				m.status = "已显示帮助"
			} else {
				m.status = "已隐藏帮助"
			}
			return m, nil
		}

		switch m.active {
		case sectionOverview:
			return m.updateOverview(msg)
		case sectionDisk:
			return m.updateDisk(msg)
		case sectionPackage:
			return m.updatePackages(msg)
		default:
			return m, nil
		}
	}

	return m, nil
}

func (m *model) prevSection() {
	if m.active == sectionOverview {
		m.active = sectionPackage
		return
	}
	m.active--
}

func (m *model) nextSection() {
	if m.active == sectionPackage {
		m.active = sectionOverview
		return
	}
	m.active++
}

func (m model) updateOverview(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.networkCursor > 0 {
			m.networkCursor--
		}
		return m, nil
	case "down", "j":
		if m.networkCursor < len(m.networkDrafts)-1 {
			m.networkCursor++
		}
		return m, nil
	case "left", "h":
		if m.networkCol > 0 {
			m.networkCol--
		}
		return m, nil
	case "right", "l":
		if m.networkCol < networkColConnection {
			m.networkCol++
		}
		return m, nil
	case "enter", "e":
		if !m.canEditCurrentCell() {
			m.status = "当前单元格不可编辑"
			return m, nil
		}
		m.networkEdit.Active = true
		m.networkEdit.Value = m.currentNetworkCellValue()
		m.status = "开始编辑当前单元格"
		return m, nil
	case "ctrl+s":
		action, err := m.currentNetworkAction()
		if err != nil {
			m.status = err.Error()
			return m, nil
		}
		m.confirming = &pendingAction{action: action}
		return m, nil
	}
	return m, nil
}

func (m model) updateNetworkEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.networkEdit = networkEdit{}
		m.status = "已取消单元格编辑"
		return m, nil
	case "enter":
		m.applyNetworkEditValue(strings.TrimSpace(m.networkEdit.Value))
		m.networkEdit = networkEdit{}
		m.status = "已应用当前单元格"
		return m, nil
	case "backspace":
		if m.networkCol == networkColMode {
			m.toggleModeValue()
			return m, nil
		}
		runes := []rune(m.networkEdit.Value)
		if len(runes) > 0 {
			m.networkEdit.Value = string(runes[:len(runes)-1])
		}
		return m, nil
	case "left", "right", " ":
		if m.networkCol == networkColMode {
			m.toggleModeValue()
		}
		return m, nil
	case "ctrl+s":
		m.applyNetworkEditValue(strings.TrimSpace(m.networkEdit.Value))
		m.networkEdit = networkEdit{}
		action, err := m.currentNetworkAction()
		if err != nil {
			m.status = err.Error()
			return m, nil
		}
		m.confirming = &pendingAction{action: action}
		return m, nil
	}

	if m.networkCol != networkColMode {
		value := msg.String()
		if utf8.RuneCountInString(value) == 1 && value != "\x00" {
			m.networkEdit.Value += value
		}
	}
	return m, nil
}

func (m model) updateDisk(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.diskCursor > 0 {
			m.diskCursor--
		}
		return m, nil
	case "down", "j":
		if m.diskCursor < len(m.snapshot.Disk.Entries)-1 {
			m.diskCursor++
		}
		return m, nil
	case "enter":
		if len(m.snapshot.Disk.Entries) == 0 {
			m.status = "当前目录没有可进入的子项"
			return m, nil
		}
		entry := m.snapshot.Disk.Entries[m.diskCursor]
		if !entry.IsDir {
			m.status = "当前选中项不是目录"
			return m, nil
		}
		m.diskPath = entry.Path
		m.diskCursor = 0
		m.loading = true
		m.status = "已进入目录"
		return m, m.refreshCmd()
	case "backspace":
		if strings.TrimSpace(m.snapshot.Disk.Parent) == "" {
			m.status = "已经在最上层目录"
			return m, nil
		}
		m.diskPath = m.snapshot.Disk.Parent
		m.diskCursor = 0
		m.loading = true
		m.status = "已返回上一级目录"
		return m, m.refreshCmd()
	}
	return m, nil
}

func (m model) updatePackages(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.searchMode {
		return m.updateSearchInput(msg)
	}

	visibleApps := m.visibleApps()

	switch msg.String() {
	case "up", "k":
		if m.appCursor > 0 {
			m.appCursor--
		}
		return m, nil
	case "down", "j":
		if m.appCursor < len(visibleApps)-1 {
			m.appCursor++
		}
		return m, nil
	case " ":
		if len(visibleApps) == 0 {
			return m, nil
		}
		pkg := visibleApps[m.appCursor].Package
		m.selectedApps[pkg] = !m.selectedApps[pkg]
		m.status = "已更新软件勾选状态"
		return m, nil
	case "/":
		m.searchMode = true
		m.searchInput = ""
		m.status = "输入关键词搜索软件包，Enter 执行，Esc 取消"
		return m, nil
	case "esc":
		if m.showSearch {
			m.showSearch = false
			m.searchResults = nil
			m.appCursor = 0
			m.status = "已返回默认列表"
			return m, nil
		}
		return m, nil
	case "o":
		m.confirming = &pendingAction{action: system.OfficialSourceAction()}
		return m, nil
	case "b":
		m.confirming = &pendingAction{action: system.RestoreSourceAction()}
		return m, nil
	case "u":
		m.confirming = &pendingAction{action: system.AptUpdateAction()}
		return m, nil
	case "c":
		m.confirming = &pendingAction{action: system.CleanAptCacheAction()}
		return m, nil
	case "g":
		m.confirming = &pendingAction{action: system.CleanLogsAction()}
		return m, nil
	case "i":
		packages := m.selectedPackageNames()
		if len(packages) == 0 {
			m.status = "请先勾选要安装的软件"
			return m, nil
		}
		m.confirming = &pendingAction{action: system.InstallAppsAction(packages)}
		return m, nil
	case "d":
		packages := m.selectedInstalledPackageNames()
		if len(packages) == 0 {
			m.status = "请先勾选已安装的软件再卸载"
			return m, nil
		}
		m.confirming = &pendingAction{action: system.UninstallAppsAction(packages)}
		return m, nil
	}
	return m, nil
}

func (m model) updateSearchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.searchMode = false
		m.searchInput = ""
		m.status = "已取消搜索"
		return m, nil
	case "enter":
		keyword := strings.TrimSpace(m.searchInput)
		m.searchMode = false
		if keyword == "" {
			m.status = "搜索关键词为空"
			return m, nil
		}
		m.status = "正在搜索: " + keyword
		return m, searchCmd(keyword)
	case "backspace":
		runes := []rune(m.searchInput)
		if len(runes) > 0 {
			m.searchInput = string(runes[:len(runes)-1])
		}
		return m, nil
	}

	value := msg.String()
	if utf8.RuneCountInString(value) == 1 && value != "\x00" {
		m.searchInput += value
	}
	return m, nil
}

func (m model) View() string {
	if !m.ready {
		return "正在准备界面..."
	}

	header := m.viewHeader()
	body := m.viewBody()
	footer := m.viewFooter()
	content := lipgloss.JoinVertical(lipgloss.Left, header, body, footer)

	if m.confirming != nil {
		return overlay(content, m.viewConfirmDialog())
	}

	return content
}

func (m model) refreshCmd() tea.Cmd {
	target := m.diskPath
	apps := append([]system.AppInfo(nil), m.apps...)
	return func() tea.Msg {
		return snapshotMsg{snapshot: system.CollectSnapshot(target, apps)}
	}
}

func (m model) selectedPackageNames() []string {
	result := make([]string, 0, len(m.selectedApps))
	for _, app := range m.visibleApps() {
		if m.selectedApps[app.Package] {
			result = append(result, app.Package)
		}
	}
	return result
}

func (m model) selectedInstalledPackageNames() []string {
	result := make([]string, 0, len(m.selectedApps))
	for _, app := range m.visibleApps() {
		if m.selectedApps[app.Package] && app.Installed {
			result = append(result, app.Package)
		}
	}
	return result
}

func (m model) visibleApps() []system.AppState {
	if m.showSearch {
		return m.searchResults
	}
	return m.snapshot.Packages.Apps
}

func searchCmd(keyword string) tea.Cmd {
	return func() tea.Msg {
		return searchResultsMsg{results: system.SearchPackages(keyword)}
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(8*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func execActionCmd(action system.Action) tea.Cmd {
	cmd := exec.Command("sh", "-lc", action.Command)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return actionDoneMsg{title: action.Title, err: err}
	})
}

func overlay(base string, dialog string) string {
	return lipgloss.Place(
		lipgloss.Width(base),
		lipgloss.Height(base),
		lipgloss.Center,
		lipgloss.Center,
		dialog,
	)
}

func renderList(lines []string, empty string) string {
	if len(lines) == 0 {
		return empty
	}
	return strings.Join(lines, "\n")
}

func (m *model) syncNetworkDrafts() {
	if len(m.snapshot.Network.Interfaces) == 0 {
		m.networkDrafts = nil
		m.networkCursor = 0
		return
	}

	next := make([]networkDraft, 0, len(m.snapshot.Network.Interfaces))
	for idx, iface := range m.snapshot.Network.Interfaces {
		mode := "静态"
		if strings.TrimSpace(iface.Method) == "auto" {
			mode = "DHCP"
		}
		draft := networkDraft{
			Device:     iface.Name,
			Connection: iface.Connection,
			State:      iface.State,
			Mode:       mode,
			Address:    iface.IPv4,
			Mask:       iface.Mask,
			Gateway:    iface.Gateway,
			DNS:        strings.Join(iface.DNS, " "),
		}
		if idx < len(m.networkDrafts) && m.networkDrafts[idx].Device == iface.Name {
			old := m.networkDrafts[idx]
			draft.Mode = firstText(old.Mode, draft.Mode)
			draft.Address = firstText(old.Address, draft.Address)
			draft.Mask = firstText(old.Mask, draft.Mask)
			draft.Gateway = firstText(old.Gateway, draft.Gateway)
			draft.DNS = firstText(old.DNS, draft.DNS)
		}
		next = append(next, draft)
	}
	m.networkDrafts = next
	if m.networkCursor >= len(m.networkDrafts) {
		m.networkCursor = len(m.networkDrafts) - 1
	}
	if m.networkCursor < 0 {
		m.networkCursor = 0
	}
	if m.networkCol > networkColConnection {
		m.networkCol = networkColConnection
	}
}

func (m model) currentNetworkDraft() *networkDraft {
	if len(m.networkDrafts) == 0 || m.networkCursor < 0 || m.networkCursor >= len(m.networkDrafts) {
		return nil
	}
	return &m.networkDrafts[m.networkCursor]
}

func (m model) canEditCurrentCell() bool {
	draft := m.currentNetworkDraft()
	if draft == nil {
		return false
	}
	if !m.snapshot.Network.NMCLIAvailable {
		return false
	}
	if strings.TrimSpace(draft.Connection) == "" {
		return false
	}
	return m.networkCol >= networkColMode && m.networkCol <= networkColDNS
}

func (m model) currentNetworkCellValue() string {
	draft := m.currentNetworkDraft()
	if draft == nil {
		return ""
	}
	switch m.networkCol {
	case networkColName:
		return draft.Device
	case networkColState:
		return draft.State
	case networkColMode:
		return draft.Mode
	case networkColIP:
		return draft.Address
	case networkColMask:
		return draft.Mask
	case networkColGateway:
		return draft.Gateway
	case networkColDNS:
		return draft.DNS
	case networkColConnection:
		return draft.Connection
	default:
		return ""
	}
}

func (m *model) applyNetworkEditValue(value string) {
	draft := m.currentNetworkDraft()
	if draft == nil {
		return
	}
	switch m.networkCol {
	case networkColMode:
		if value == "DHCP" {
			draft.Mode = "DHCP"
		} else {
			draft.Mode = "静态"
		}
	case networkColIP:
		draft.Address = value
		draft.Mode = "静态"
	case networkColMask:
		draft.Mask = value
		draft.Mode = "静态"
	case networkColGateway:
		draft.Gateway = value
		draft.Mode = "静态"
	case networkColDNS:
		draft.DNS = value
		draft.Mode = "静态"
	}
}

func (m *model) toggleModeValue() {
	if m.networkCol != networkColMode {
		return
	}
	if m.networkEdit.Value == "DHCP" {
		m.networkEdit.Value = "静态"
	} else {
		m.networkEdit.Value = "DHCP"
	}
}

func (m model) currentNetworkAction() (system.Action, error) {
	draft := m.currentNetworkDraft()
	if draft == nil {
		return system.Action{}, fmt.Errorf("当前没有可保存的网卡")
	}
	method := "manual"
	if draft.Mode == "DHCP" {
		method = "auto"
	}
	return system.ConfigureNetworkAction(system.NetworkConfig{
		Connection: draft.Connection,
		Device:     draft.Device,
		Method:     method,
		Address:    draft.Address,
		Mask:       draft.Mask,
		Gateway:    draft.Gateway,
		DNS:        draft.DNS,
	})
}
