package tui

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/chxcodepro/qilin-manager-tui/internal/system"
)

type section int

const (
	sectionSystem section = iota
	sectionNetwork
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

type model struct {
	version       string
	active        section
	width         int
	height        int
	ready         bool
	loading       bool
	showHelp      bool
	diskTargets   []string
	diskTargetIdx int
	apps          []system.AppInfo
	appCursor     int
	selectedApps  map[string]bool
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
		active:       sectionSystem,
		diskTargets:  []string{"/", "/home", "/var", "/opt"},
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
	if m.active == sectionSystem {
		m.active = sectionPackage
		return
	}
	m.active--
}

func (m *model) nextSection() {
	if m.active == sectionPackage {
		m.active = sectionSystem
		return
	}
	m.active++
}

func (m model) updateDisk(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "[":
		if m.diskTargetIdx > 0 {
			m.diskTargetIdx--
		} else {
			m.diskTargetIdx = len(m.diskTargets) - 1
		}
		m.loading = true
		m.status = "已切换磁盘分析路径"
		return m, m.refreshCmd()
	case "]":
		if m.diskTargetIdx < len(m.diskTargets)-1 {
			m.diskTargetIdx++
		} else {
			m.diskTargetIdx = 0
		}
		m.loading = true
		m.status = "已切换磁盘分析路径"
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
	target := m.diskTargets[m.diskTargetIdx]
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
