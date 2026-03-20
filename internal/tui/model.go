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

type snapshotMsg struct {
	snapshot system.Snapshot
}

type actionDoneMsg struct {
	title string
	err   error
}

type tickMsg time.Time

type pendingAction struct {
	action system.Action
}

type formField struct {
	Label string
	Value string
	Kind  string
}

type networkForm struct {
	iface  system.NetworkInterface
	fields []formField
	cursor int
}

type model struct {
	version      string
	active       section
	width        int
	height       int
	ready        bool
	loading      bool
	showHelp     bool
	diskPath     string
	diskCursor   int
	networkCursor int
	apps         []system.AppInfo
	appCursor    int
	selectedApps map[string]bool
	snapshot     system.Snapshot
	status       string
	confirming   *pendingAction
	form         *networkForm
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
		if len(m.snapshot.Network.Interfaces) == 0 {
			m.networkCursor = 0
		} else if m.networkCursor >= len(m.snapshot.Network.Interfaces) {
			m.networkCursor = len(m.snapshot.Network.Interfaces) - 1
		}
		if len(m.snapshot.Disk.Entries) == 0 {
			m.diskCursor = 0
		} else if m.diskCursor >= len(m.snapshot.Disk.Entries) {
			m.diskCursor = len(m.snapshot.Disk.Entries) - 1
		}
		m.status = "数据已刷新"
		return m, nil

	case tickMsg:
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

	case tea.KeyMsg:
		if m.form != nil {
			return m.updateForm(msg)
		}

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

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "left", "h":
			m.prevSection()
			return m, nil
		case "right", "l":
			m.nextSection()
			return m, nil
		case "r":
			m.loading = true
			m.status = "正在手动刷新"
			return m, m.refreshCmd()
		case "?":
			m.showHelp = !m.showHelp
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
		if m.networkCursor < len(m.snapshot.Network.Interfaces)-1 {
			m.networkCursor++
		}
		return m, nil
	case "e":
		if len(m.snapshot.Network.Interfaces) == 0 {
			m.status = "当前没有可编辑的网卡"
			return m, nil
		}
		if !m.snapshot.Network.NMCLIAvailable {
			m.status = "当前系统没有 nmcli，暂不支持保存网卡配置"
			return m, nil
		}
		iface := m.snapshot.Network.Interfaces[m.networkCursor]
		if strings.TrimSpace(iface.Connection) == "" {
			m.status = "当前网卡没有可用的 NetworkManager 连接"
			return m, nil
		}
		m.form = newNetworkForm(iface)
		m.status = "正在编辑网卡配置"
		return m, nil
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
	switch msg.String() {
	case "up", "k":
		if m.appCursor > 0 {
			m.appCursor--
		}
		return m, nil
	case "down", "j":
		if m.appCursor < len(m.apps)-1 {
			m.appCursor++
		}
		return m, nil
	case " ":
		if len(m.apps) == 0 {
			return m, nil
		}
		pkg := m.apps[m.appCursor].Package
		m.selectedApps[pkg] = !m.selectedApps[pkg]
		m.status = "已更新软件勾选状态"
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
	}
	return m, nil
}

func (m model) updateForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.form == nil {
		return m, nil
	}

	field := m.form.current()
	switch msg.String() {
	case "esc":
		m.form = nil
		m.status = "已取消网卡编辑"
		return m, nil
	case "up", "k", "shift+tab":
		m.form.move(-1)
		return m, nil
	case "down", "j", "tab":
		m.form.move(1)
		return m, nil
	case "left", "right", " ":
		if field.Kind == "mode" {
			m.form.toggleMode()
		}
		return m, nil
	case "backspace":
		if field.Kind == "text" && field.Value != "" {
			runes := []rune(field.Value)
			field.Value = string(runes[:len(runes)-1])
		}
		return m, nil
	case "ctrl+s":
		cfg := m.form.config()
		action, err := system.ConfigureNetworkAction(cfg)
		if err != nil {
			m.status = err.Error()
			return m, nil
		}
		m.form = nil
		m.confirming = &pendingAction{action: action}
		return m, nil
	case "enter":
		m.form.move(1)
		return m, nil
	}

	if field.Kind == "text" {
		value := msg.String()
		if utf8.RuneCountInString(value) == 1 && value != "\x00" {
			field.Value += value
		}
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
	if m.form != nil {
		return overlay(content, m.viewNetworkForm())
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
	for _, app := range m.apps {
		if m.selectedApps[app.Package] {
			result = append(result, app.Package)
		}
	}
	return result
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

func newNetworkForm(iface system.NetworkInterface) *networkForm {
	mode := "静态"
	if strings.TrimSpace(iface.Method) == "auto" {
		mode = "DHCP"
	}
	return &networkForm{
		iface: iface,
		fields: []formField{
			{Label: "模式", Value: mode, Kind: "mode"},
			{Label: "IP 地址", Value: iface.IPv4, Kind: "text"},
			{Label: "子网掩码", Value: iface.Mask, Kind: "text"},
			{Label: "默认网关", Value: iface.Gateway, Kind: "text"},
			{Label: "DNS", Value: strings.Join(iface.DNS, " "), Kind: "text"},
		},
	}
}

func (f *networkForm) move(delta int) {
	if len(f.fields) == 0 {
		f.cursor = 0
		return
	}
	f.cursor = (f.cursor + delta + len(f.fields)) % len(f.fields)
}

func (f *networkForm) current() *formField {
	if len(f.fields) == 0 {
		return &formField{}
	}
	return &f.fields[f.cursor]
}

func (f *networkForm) toggleMode() {
	current := f.current()
	if current.Kind != "mode" {
		return
	}
	if current.Value == "DHCP" {
		current.Value = "静态"
		return
	}
	current.Value = "DHCP"
}

func (f *networkForm) config() system.NetworkConfig {
	method := "manual"
	if f.fields[0].Value == "DHCP" {
		method = "auto"
	}
	return system.NetworkConfig{
		Connection: f.iface.Connection,
		Device:     f.iface.Name,
		Method:     method,
		Address:    strings.TrimSpace(f.fields[1].Value),
		Mask:       strings.TrimSpace(f.fields[2].Value),
		Gateway:    strings.TrimSpace(f.fields[3].Value),
		DNS:        strings.TrimSpace(f.fields[4].Value),
	}
}
