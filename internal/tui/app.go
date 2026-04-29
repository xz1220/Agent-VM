package tui

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/xz1220/agent-vm/internal/config"
	"github.com/xz1220/agent-vm/internal/state"
	"gopkg.in/yaml.v3"
)

type ActivateFunc func(kind, name string) error
type RuntimeLoader func(cwd string) ([]RuntimeRow, string, error)
type RuntimeScanner func() error
type ImportAgentCreator func(ref string) (string, error)
type MemoryImporter func(source string) (string, error)

type Options struct {
	CWD                string
	Input              io.Reader
	Output             io.Writer
	Activate           ActivateFunc
	LoadRuntimes       RuntimeLoader
	ScanRuntimes       RuntimeScanner
	CreateImportAgent  ImportAgentCreator
	ImportMemoryDryRun MemoryImporter
	DisableAltScreen   bool
}

type RuntimeRow struct {
	Runtime    string
	Found      bool
	ConfigDir  string
	Candidates []RuntimeCandidate
	Warnings   []string
}

type RuntimeCandidate struct {
	Runtime     string
	Name        string
	Description string
}

type Snapshot struct {
	Active             config.ActiveRef
	ConfigWarning      string
	SyncState          *state.SyncState
	Agents             []AgentRow
	Envs               []EnvRow
	HasProjectOverride bool
	ProjectOverride    *config.ProjectOverride
	Runtimes           []RuntimeRow
	RuntimeReportPath  string
	Skills             []SkillRow
	MCPs               []MCPRow
	Memory             []MemoryRow
}

type AgentRow struct {
	Name        string
	Scope       config.Scope
	Description string
	Version     string
	Path        string
}

type EnvRow struct {
	Name        string
	Description string
	Version     string
	Path        string
	Local       bool
}

type SkillRow struct {
	Name        string
	Description string
	Path        string
}

type MCPRow struct {
	Name        string
	Description string
	Type        string
	Command     string
	URL         string
	Path        string
}

type MemoryRow struct {
	ID          string
	Scope       config.Scope
	Format      string
	Description string
	Path        string
}

type tab int

const (
	tabStatus tab = iota
	tabAgents
	tabEnvs
	tabRuntimes
	tabSkills
	tabMCP
	tabMemory
	tabCount
)

var tabNames = []string{"Status", "Agents", "Envs", "Runtimes", "Skills", "MCP", "Memory"}

type mode int

const (
	modeNormal mode = iota
	modeForm
	modeConfirm
	modeHelp
	modePreview
)

type Model struct {
	opts         Options
	width        int
	height       int
	tab          tab
	selected     [tabCount]int
	snapshot     Snapshot
	err          string
	message      string
	mode         mode
	form         formModel
	confirm      confirmModel
	previewTitle string
	previewBody  string
	styles       styles
}

type styles struct {
	header      lipgloss.Style
	tab         lipgloss.Style
	activeTab   lipgloss.Style
	panelTitle  lipgloss.Style
	row         lipgloss.Style
	selectedRow lipgloss.Style
	muted       lipgloss.Style
	error       lipgloss.Style
	success     lipgloss.Style
	footer      lipgloss.Style
	label       lipgloss.Style
	inputLabel  lipgloss.Style
}

type formField struct {
	Key   string
	Label string
	Help  string
	Input textinput.Model
}

type formModel struct {
	Kind   string
	Title  string
	Fields []formField
	Focus  int
	Meta   map[string]string
	Err    string
}

type confirmModel struct {
	Title  string
	Body   string
	Kind   string
	Target map[string]string
}

func Run(opts Options) error {
	if opts.CWD == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		opts.CWD = cwd
	}
	model, err := NewModel(opts)
	if err != nil {
		return err
	}
	programOpts := []tea.ProgramOption{}
	if opts.Input != nil {
		programOpts = append(programOpts, tea.WithInput(opts.Input))
	}
	if opts.Output != nil {
		programOpts = append(programOpts, tea.WithOutput(opts.Output))
	}
	if !opts.DisableAltScreen {
		programOpts = append(programOpts, tea.WithAltScreen())
	}
	_, err = tea.NewProgram(model, programOpts...).Run()
	return err
}

func NewModel(opts Options) (Model, error) {
	if opts.CWD == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return Model{}, err
		}
		opts.CWD = cwd
	}
	m := Model{
		opts:   opts,
		width:  100,
		height: 30,
		styles: newStyles(),
	}
	if err := m.refresh(); err != nil {
		m.err = err.Error()
	}
	return m, nil
}

func newStyles() styles {
	return styles{
		header:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Background(lipgloss.Color("62")).Padding(0, 1),
		tab:         lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Padding(0, 1),
		activeTab:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Background(lipgloss.Color("24")).Padding(0, 1),
		panelTitle:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81")),
		row:         lipgloss.NewStyle().Foreground(lipgloss.Color("252")),
		selectedRow: lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Background(lipgloss.Color("238")),
		muted:       lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
		error:       lipgloss.NewStyle().Foreground(lipgloss.Color("203")),
		success:     lipgloss.NewStyle().Foreground(lipgloss.Color("84")),
		footer:      lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Background(lipgloss.Color("235")).Padding(0, 1),
		label:       lipgloss.NewStyle().Foreground(lipgloss.Color("117")).Bold(true),
		inputLabel:  lipgloss.NewStyle().Foreground(lipgloss.Color("250")),
	}
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = max(60, msg.Width)
		m.height = max(20, msg.Height)
		m.resizeFormInputs()
		return m, nil
	case tea.KeyMsg:
		switch m.mode {
		case modeForm:
			return m.updateForm(msg)
		case modeConfirm:
			return m.updateConfirm(msg)
		case modeHelp:
			if keyString(msg) == "esc" || keyString(msg) == "?" || keyString(msg) == "q" {
				m.mode = modeNormal
			}
			return m, nil
		case modePreview:
			if keyString(msg) == "esc" || keyString(msg) == "q" {
				m.mode = modeNormal
			}
			return m, nil
		default:
			return m.updateNormal(msg)
		}
	}
	return m, nil
}

func (m Model) updateNormal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.err = ""
	key := keyString(msg)
	switch key {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "?":
		m.mode = modeHelp
	case "left", "h", "shift+tab":
		m.tab = tab((int(m.tab) + int(tabCount) - 1) % int(tabCount))
	case "right", "l", "tab":
		m.tab = tab((int(m.tab) + 1) % int(tabCount))
	case "up", "k":
		m.moveSelection(-1)
	case "down", "j":
		m.moveSelection(1)
	case "r":
		if err := m.refresh(); err != nil {
			m.err = err.Error()
		} else {
			m.message = "refreshed"
		}
	case "x":
		if m.tab == tabRuntimes {
			m.scanRuntimes()
		}
	case "n":
		m.startNew()
	case "e", "enter":
		m.startEdit()
	case "d":
		m.startDelete()
	case "u":
		m.activateSelected()
	case "i":
		if m.tab == tabMemory {
			m.startMemoryImport()
		}
	}
	return m, nil
}

func (m Model) updateForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := keyString(msg)
	switch key {
	case "esc":
		m.mode = modeNormal
		return m, nil
	case "ctrl+c":
		return m, tea.Quit
	case "tab", "down":
		m.focusField(m.form.Focus + 1)
		return m, nil
	case "shift+tab", "up":
		m.focusField(m.form.Focus - 1)
		return m, nil
	case "enter":
		if m.form.Focus < len(m.form.Fields)-1 {
			m.focusField(m.form.Focus + 1)
			return m, nil
		}
		if err := m.saveForm(); err != nil {
			m.form.Err = err.Error()
			return m, nil
		}
		if m.mode == modePreview {
			return m, nil
		}
		m.mode = modeNormal
		m.message = "saved"
		if err := m.refresh(); err != nil {
			m.err = err.Error()
		}
		return m, nil
	case "ctrl+s":
		if err := m.saveForm(); err != nil {
			m.form.Err = err.Error()
			return m, nil
		}
		if m.mode == modePreview {
			return m, nil
		}
		m.mode = modeNormal
		m.message = "saved"
		if err := m.refresh(); err != nil {
			m.err = err.Error()
		}
		return m, nil
	}

	if len(m.form.Fields) == 0 {
		return m, nil
	}
	input, cmd := m.form.Fields[m.form.Focus].Input.Update(msg)
	m.form.Fields[m.form.Focus].Input = input
	return m, cmd
}

func (m Model) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch keyString(msg) {
	case "esc", "n", "q":
		m.mode = modeNormal
	case "y", "enter":
		if err := m.runConfirm(); err != nil {
			m.err = err.Error()
			m.mode = modeNormal
			return m, nil
		}
		m.mode = modeNormal
		m.message = "deleted"
		if err := m.refresh(); err != nil {
			m.err = err.Error()
		}
	case "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

func (m *Model) refresh() error {
	snapshot, err := LoadSnapshot(m.opts)
	if err != nil {
		return err
	}
	m.snapshot = snapshot
	m.clampSelection()
	return nil
}

func LoadSnapshot(opts Options) (Snapshot, error) {
	var out Snapshot
	cfg, err := config.ReadGlobalConfig()
	if err != nil {
		if os.IsNotExist(err) {
			out.ConfigWarning = "AVM home is not initialized"
		} else {
			return out, err
		}
	} else {
		out.Active = cfg.Active
	}

	if syncState, err := state.LoadSyncState(""); err == nil {
		out.SyncState = &syncState
	}

	globalAgents, err := config.ListAgents(config.ScopeGlobal, opts.CWD)
	if err != nil {
		return out, err
	}
	projectAgents, err := config.ListAgents(config.ScopeProject, opts.CWD)
	if err != nil {
		return out, err
	}
	out.Agents = append(out.Agents, agentRows(globalAgents, config.ScopeGlobal)...)
	out.Agents = append(out.Agents, agentRows(projectAgents, config.ScopeProject)...)

	envs, err := config.ListEnvironments()
	if err != nil {
		return out, err
	}
	for _, env := range envs {
		out.Envs = append(out.Envs, EnvRow{
			Name:        env.Name,
			Description: env.Description,
			Version:     env.Version,
			Path:        env.Path,
		})
	}
	if override, err := config.ReadProjectOverride(opts.CWD); err == nil {
		out.HasProjectOverride = true
		out.ProjectOverride = override
		out.Envs = append(out.Envs, EnvRow{
			Name:        override.Extends,
			Description: "current project override",
			Path:        config.ProjectEnvPath(opts.CWD),
			Local:       true,
		})
	} else if err != nil && !os.IsNotExist(err) {
		return out, err
	}

	if opts.LoadRuntimes != nil {
		rows, path, err := opts.LoadRuntimes(opts.CWD)
		if err != nil {
			return out, err
		}
		out.Runtimes = rows
		out.RuntimeReportPath = path
	}

	skills, err := listInstalledSkills()
	if err != nil {
		return out, err
	}
	out.Skills = skills

	mcps, err := config.ListMCPRegistryEntries()
	if err != nil {
		return out, err
	}
	for _, mcp := range mcps {
		out.MCPs = append(out.MCPs, MCPRow{
			Name:        mcp.Name,
			Description: mcp.Description,
			Type:        mcp.Type,
			Command:     mcp.Command,
			URL:         mcp.URL,
			Path:        mcp.Path,
		})
	}

	memories, err := config.ListPortableMemory("")
	if err != nil {
		return out, err
	}
	for _, memory := range memories {
		out.Memory = append(out.Memory, MemoryRow{
			ID:          memory.ID,
			Scope:       config.Scope(memory.Scope),
			Format:      memory.Format,
			Description: memory.Description,
			Path:        memory.Path,
		})
	}
	return out, nil
}

func agentRows(summaries []config.AgentSummary, scope config.Scope) []AgentRow {
	rows := make([]AgentRow, 0, len(summaries))
	for _, agent := range summaries {
		rows = append(rows, AgentRow{
			Name:        agent.Name,
			Scope:       scope,
			Description: agent.Description,
			Version:     agent.Version,
			Path:        agent.Path,
		})
	}
	return rows
}

func listInstalledSkills() ([]SkillRow, error) {
	root := config.RegistryKindDir("skills")
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	rows := make([]SkillRow, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(root, entry.Name(), "SKILL.md")
		if _, err := os.Stat(path); err != nil {
			continue
		}
		rows = append(rows, SkillRow{
			Name:        entry.Name(),
			Description: skillDescription(path),
			Path:        path,
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].Name < rows[j].Name
	})
	return rows, nil
}

func skillDescription(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || line == "---" {
			continue
		}
		if strings.HasPrefix(line, "description:") {
			return strings.Trim(strings.TrimSpace(strings.TrimPrefix(line, "description:")), `"'`)
		}
		return line
	}
	return ""
}

func (m *Model) moveSelection(delta int) {
	rows := m.rowCount()
	if rows == 0 {
		m.selected[m.tab] = 0
		return
	}
	next := m.selected[m.tab] + delta
	if next < 0 {
		next = 0
	}
	if next >= rows {
		next = rows - 1
	}
	m.selected[m.tab] = next
}

func (m *Model) clampSelection() {
	for t := tab(0); t < tabCount; t++ {
		rows := m.rowCountForTab(t)
		if rows == 0 {
			m.selected[t] = 0
		} else if m.selected[t] >= rows {
			m.selected[t] = rows - 1
		}
	}
}

func (m Model) rowCount() int {
	return m.rowCountForTab(m.tab)
}

func (m Model) rowCountForTab(t tab) int {
	switch t {
	case tabStatus:
		if m.snapshot.SyncState == nil || len(m.snapshot.SyncState.Runtimes) == 0 {
			return 1
		}
		return len(m.snapshot.SyncState.Runtimes)
	case tabAgents:
		return len(m.snapshot.Agents)
	case tabEnvs:
		return len(m.snapshot.Envs)
	case tabRuntimes:
		count := 0
		for _, runtime := range m.snapshot.Runtimes {
			count++
			count += len(runtime.Candidates)
		}
		return count
	case tabSkills:
		return len(m.snapshot.Skills)
	case tabMCP:
		return len(m.snapshot.MCPs)
	case tabMemory:
		return len(m.snapshot.Memory)
	default:
		return 0
	}
}

func (m Model) View() string {
	switch m.mode {
	case modeForm:
		return m.viewFrame(m.viewForm())
	case modeConfirm:
		return m.viewFrame(m.viewConfirm())
	case modeHelp:
		return m.viewFrame(m.viewHelp())
	case modePreview:
		return m.viewFrame(m.viewPreview())
	default:
		return m.viewFrame(m.viewNormal())
	}
}

func (m Model) viewFrame(body string) string {
	width := max(60, m.width)
	headerText := "AVM Console"
	if active := formatActive(m.snapshot.Active); active != "none" {
		headerText += "  active " + active
	}
	header := m.styles.header.Width(width).Render(headerText)
	return lipgloss.JoinVertical(lipgloss.Left, header, body)
}

func (m Model) viewNormal() string {
	width := max(60, m.width)
	height := max(20, m.height)
	tabs := m.viewTabs(width)
	footer := m.viewFooter(width)
	available := height - 3
	if m.err != "" || m.message != "" || m.snapshot.ConfigWarning != "" {
		available--
	}
	leftW := clamp(width/3, 24, 38)
	rightW := max(20, width-leftW-1)
	list := m.renderList(leftW, available)
	detail := m.renderDetail(rightW, available)
	body := lipgloss.JoinHorizontal(lipgloss.Top, list, detail)
	lines := []string{tabs, body}
	if m.err != "" {
		lines = append(lines, m.styles.error.Render(m.err))
	} else if m.message != "" {
		lines = append(lines, m.styles.success.Render(m.message))
	} else if m.snapshot.ConfigWarning != "" {
		lines = append(lines, m.styles.error.Render(m.snapshot.ConfigWarning))
	}
	lines = append(lines, footer)
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m Model) viewTabs(width int) string {
	parts := make([]string, 0, len(tabNames))
	for i, name := range tabNames {
		style := m.styles.tab
		if tab(i) == m.tab {
			style = m.styles.activeTab
		}
		parts = append(parts, style.Render(name))
	}
	line := strings.Join(parts, "")
	return fitLine(line, width)
}

func (m Model) viewFooter(width int) string {
	keys := "left/right tabs  j/k move  n new  e edit  d delete  u activate  r refresh  ? help  q quit"
	switch m.tab {
	case tabRuntimes:
		keys = "left/right tabs  j/k move  x scan  n create from candidate  r refresh  ? help  q quit"
	case tabMemory:
		keys = "left/right tabs  j/k move  n new  e edit  d delete  i import dry-run  r refresh  ? help  q quit"
	case tabSkills:
		keys = "left/right tabs  j/k move  r refresh  ? help  q quit"
	}
	return m.styles.footer.Width(width).Render(fitLine(keys, width-2))
}

func (m Model) renderList(width, height int) string {
	title := m.styles.panelTitle.Render(tabNames[m.tab])
	rows := m.listRows()
	lines := []string{title}
	if len(rows) == 0 {
		lines = append(lines, m.styles.muted.Render("No items"))
	} else {
		for i, row := range rows {
			line := fitLine(row, width)
			if i == m.selected[m.tab] {
				line = m.styles.selectedRow.Width(width).Render(line)
			} else {
				line = m.styles.row.Width(width).Render(line)
			}
			lines = append(lines, line)
		}
	}
	return fitBlock(strings.Join(lines, "\n"), width, height)
}

func (m Model) listRows() []string {
	switch m.tab {
	case tabStatus:
		if m.snapshot.SyncState == nil || len(m.snapshot.SyncState.Runtimes) == 0 {
			return []string{"runtime status unavailable"}
		}
		runtimes := make([]string, 0, len(m.snapshot.SyncState.Runtimes))
		for runtime := range m.snapshot.SyncState.Runtimes {
			runtimes = append(runtimes, runtime)
		}
		sort.Strings(runtimes)
		rows := make([]string, 0, len(runtimes))
		for _, runtime := range runtimes {
			state := m.snapshot.SyncState.Runtimes[runtime]
			rows = append(rows, fmt.Sprintf("%s  %s", runtime, state.Status))
		}
		return rows
	case tabAgents:
		rows := make([]string, 0, len(m.snapshot.Agents))
		for _, agent := range m.snapshot.Agents {
			rows = append(rows, fmt.Sprintf("%s  [%s]", agent.Name, agent.Scope))
		}
		return rows
	case tabEnvs:
		rows := make([]string, 0, len(m.snapshot.Envs))
		for _, env := range m.snapshot.Envs {
			label := env.Name
			if env.Local {
				label += "  [local override]"
			}
			rows = append(rows, label)
		}
		return rows
	case tabRuntimes:
		rows := []string{}
		for _, runtime := range m.snapshot.Runtimes {
			status := "missing"
			if runtime.Found {
				status = "found"
			}
			rows = append(rows, fmt.Sprintf("%s  %s  candidates:%d", runtime.Runtime, status, len(runtime.Candidates)))
			for _, candidate := range runtime.Candidates {
				rows = append(rows, fmt.Sprintf("  %s/%s", candidate.Runtime, candidate.Name))
			}
		}
		return rows
	case tabSkills:
		rows := make([]string, 0, len(m.snapshot.Skills))
		for _, skill := range m.snapshot.Skills {
			rows = append(rows, skill.Name)
		}
		return rows
	case tabMCP:
		rows := make([]string, 0, len(m.snapshot.MCPs))
		for _, mcp := range m.snapshot.MCPs {
			rows = append(rows, mcp.Name)
		}
		return rows
	case tabMemory:
		rows := make([]string, 0, len(m.snapshot.Memory))
		for _, memory := range m.snapshot.Memory {
			rows = append(rows, fmt.Sprintf("%s  [%s]", memory.ID, memory.Scope))
		}
		return rows
	default:
		return nil
	}
}

func (m Model) renderDetail(width, height int) string {
	var detail string
	switch m.tab {
	case tabStatus:
		detail = m.statusDetail()
	case tabAgents:
		detail = m.agentDetail()
	case tabEnvs:
		detail = m.envDetail()
	case tabRuntimes:
		detail = m.runtimeDetail()
	case tabSkills:
		detail = m.skillDetail()
	case tabMCP:
		detail = m.mcpDetail()
	case tabMemory:
		detail = m.memoryDetail()
	}
	return fitBlock(detail, width, height)
}

func (m Model) statusDetail() string {
	lines := []string{
		m.styles.panelTitle.Render("Status"),
		kv("active", formatActive(m.snapshot.Active)),
	}
	if m.snapshot.RuntimeReportPath != "" {
		lines = append(lines, kv("runtime report", m.snapshot.RuntimeReportPath))
	}
	if m.snapshot.SyncState == nil {
		lines = append(lines, "", m.styles.muted.Render("sync-state not found"))
		return strings.Join(lines, "\n")
	}
	lines = append(lines, kv("last active", formatActive(m.snapshot.SyncState.LastActive)))
	lines = append(lines, kv("updated", m.snapshot.SyncState.UpdatedAt.Format("2006-01-02 15:04:05 MST")))
	lines = append(lines, "", m.styles.label.Render("Runtimes"))
	runtimes := make([]string, 0, len(m.snapshot.SyncState.Runtimes))
	for runtime := range m.snapshot.SyncState.Runtimes {
		runtimes = append(runtimes, runtime)
	}
	sort.Strings(runtimes)
	for _, runtime := range runtimes {
		row := m.snapshot.SyncState.Runtimes[runtime]
		lines = append(lines, fmt.Sprintf("- %s: %s agent=%s", runtime, row.Status, empty(row.AgentName, "none")))
		if row.Error != "" {
			lines = append(lines, "  error: "+row.Error)
		}
		for _, warning := range row.Warnings {
			lines = append(lines, "  warning: "+warning)
		}
	}
	return strings.Join(lines, "\n")
}

func (m Model) agentDetail() string {
	row, ok := m.selectedAgent()
	if !ok {
		return emptyDetail("Agents", "No agent profiles found. Press n to create one.")
	}
	agent, err := config.ReadAgent(row.Name, row.Scope, m.opts.CWD)
	if err != nil {
		return m.styles.error.Render(err.Error())
	}
	raw, _ := yaml.Marshal(agent)
	return m.styles.panelTitle.Render("Agent "+row.Name) + "\n" +
		kv("scope", string(row.Scope)) + "\n" +
		kv("path", row.Path) + "\n\n" +
		string(raw)
}

func (m Model) envDetail() string {
	row, ok := m.selectedEnv()
	if !ok {
		return emptyDetail("Envs", "No environments found. Press n to create one.")
	}
	if row.Local {
		raw, _ := yaml.Marshal(m.snapshot.ProjectOverride)
		return m.styles.panelTitle.Render("Project Override") + "\n" +
			kv("path", row.Path) + "\n\n" + string(raw)
	}
	env, err := config.ReadEnvironment(row.Name)
	if err != nil {
		return m.styles.error.Render(err.Error())
	}
	raw, _ := yaml.Marshal(env)
	return m.styles.panelTitle.Render("Env "+row.Name) + "\n" +
		kv("path", row.Path) + "\n\n" + string(raw)
}

func (m Model) runtimeDetail() string {
	item, ok := m.selectedRuntimeItem()
	if !ok {
		return emptyDetail("Runtimes", "No runtime import report found. Press x to scan.")
	}
	lines := []string{m.styles.panelTitle.Render("Runtime")}
	if item.Candidate != nil {
		c := item.Candidate
		lines = append(lines,
			kv("candidate", c.Runtime+"/"+c.Name),
			kv("summary", c.Description),
			"",
			"Press n to create an AVM agent from this candidate.",
		)
		return strings.Join(lines, "\n")
	}
	r := item.Runtime
	lines = append(lines,
		kv("runtime", r.Runtime),
		kv("found", yesNo(r.Found)),
		kv("config dir", r.ConfigDir),
		kv("candidates", fmt.Sprintf("%d", len(r.Candidates))),
	)
	if len(r.Warnings) > 0 {
		lines = append(lines, "", m.styles.label.Render("Warnings"))
		for _, warning := range r.Warnings {
			lines = append(lines, "- "+warning)
		}
	}
	return strings.Join(lines, "\n")
}

func (m Model) skillDetail() string {
	row, ok := m.selectedSkill()
	if !ok {
		return emptyDetail("Skills", "No installed skills found.")
	}
	return m.styles.panelTitle.Render("Skill "+row.Name) + "\n" +
		kv("summary", row.Description) + "\n" +
		kv("path", row.Path) + "\n\n" +
		"Attach skills from the Agents tab by editing an agent's skills field."
}

func (m Model) mcpDetail() string {
	row, ok := m.selectedMCP()
	if !ok {
		return emptyDetail("MCP", "No MCP registry entries found. Press n to create one.")
	}
	entry, _, err := config.ReadMCPRegistryEntry(row.Name)
	if err != nil {
		return m.styles.error.Render(err.Error())
	}
	raw, _ := yaml.Marshal(entry)
	return m.styles.panelTitle.Render("MCP "+row.Name) + "\n" +
		kv("path", row.Path) + "\n\n" + string(raw)
}

func (m Model) memoryDetail() string {
	row, ok := m.selectedMemory()
	if !ok {
		return emptyDetail("Memory", "No portable memory metadata found. Press n to create one.")
	}
	memory, err := config.ReadPortableMemory(row.ID, row.Scope)
	if err != nil {
		return m.styles.error.Render(err.Error())
	}
	raw, _ := yaml.Marshal(memory)
	return m.styles.panelTitle.Render("Memory "+row.ID) + "\n" +
		kv("path", row.Path) + "\n\n" + string(raw)
}

type runtimeItem struct {
	Runtime   RuntimeRow
	Candidate *RuntimeCandidate
}

func (m Model) selectedRuntimeItem() (runtimeItem, bool) {
	selected := m.selected[tabRuntimes]
	index := 0
	for _, runtime := range m.snapshot.Runtimes {
		if index == selected {
			return runtimeItem{Runtime: runtime}, true
		}
		index++
		for _, candidate := range runtime.Candidates {
			if index == selected {
				c := candidate
				return runtimeItem{Runtime: runtime, Candidate: &c}, true
			}
			index++
		}
	}
	return runtimeItem{}, false
}

func (m Model) selectedAgent() (AgentRow, bool) {
	i := m.selected[tabAgents]
	if i < 0 || i >= len(m.snapshot.Agents) {
		return AgentRow{}, false
	}
	return m.snapshot.Agents[i], true
}

func (m Model) selectedEnv() (EnvRow, bool) {
	i := m.selected[tabEnvs]
	if i < 0 || i >= len(m.snapshot.Envs) {
		return EnvRow{}, false
	}
	return m.snapshot.Envs[i], true
}

func (m Model) selectedSkill() (SkillRow, bool) {
	i := m.selected[tabSkills]
	if i < 0 || i >= len(m.snapshot.Skills) {
		return SkillRow{}, false
	}
	return m.snapshot.Skills[i], true
}

func (m Model) selectedMCP() (MCPRow, bool) {
	i := m.selected[tabMCP]
	if i < 0 || i >= len(m.snapshot.MCPs) {
		return MCPRow{}, false
	}
	return m.snapshot.MCPs[i], true
}

func (m Model) selectedMemory() (MemoryRow, bool) {
	i := m.selected[tabMemory]
	if i < 0 || i >= len(m.snapshot.Memory) {
		return MemoryRow{}, false
	}
	return m.snapshot.Memory[i], true
}

func (m *Model) startNew() {
	m.err = ""
	switch m.tab {
	case tabAgents:
		m.form = newAgentForm("new agent", nil, config.ScopeGlobal, m.width)
		m.mode = modeForm
	case tabEnvs:
		m.form = newEnvForm("new env", nil, false, m.width)
		m.mode = modeForm
	case tabRuntimes:
		item, ok := m.selectedRuntimeItem()
		if !ok || item.Candidate == nil {
			m.err = "select an import candidate first"
			return
		}
		if m.opts.CreateImportAgent == nil {
			m.err = "runtime import creation is not wired"
			return
		}
		ref := item.Candidate.Runtime + "/" + item.Candidate.Name
		name, err := m.opts.CreateImportAgent(ref)
		if err != nil {
			m.err = err.Error()
			return
		}
		m.message = "created agent " + name
		_ = m.refresh()
	case tabMCP:
		m.form = newMCPForm("new mcp", nil, m.width)
		m.mode = modeForm
	case tabMemory:
		m.form = newMemoryForm("new memory", nil, m.width)
		m.mode = modeForm
	}
}

func (m *Model) startEdit() {
	m.err = ""
	switch m.tab {
	case tabAgents:
		row, ok := m.selectedAgent()
		if !ok {
			return
		}
		agent, err := config.ReadAgent(row.Name, row.Scope, m.opts.CWD)
		if err != nil {
			m.err = err.Error()
			return
		}
		m.form = newAgentForm("edit agent", agent, row.Scope, m.width)
		m.form.Meta["old_name"] = row.Name
		m.form.Meta["old_scope"] = string(row.Scope)
		m.mode = modeForm
	case tabEnvs:
		row, ok := m.selectedEnv()
		if !ok {
			return
		}
		if row.Local {
			m.form = newEnvForm("edit project override", nil, true, m.width)
			m.form.Meta["local"] = "true"
			if m.snapshot.ProjectOverride != nil {
				setFormValue(&m.form, "name", m.snapshot.ProjectOverride.Extends)
				for runtime, agent := range m.snapshot.ProjectOverride.RuntimeAgents {
					setFormValue(&m.form, runtime, agent.Primary)
				}
			}
			m.mode = modeForm
			return
		}
		env, err := config.ReadEnvironment(row.Name)
		if err != nil {
			m.err = err.Error()
			return
		}
		m.form = newEnvForm("edit env", env, false, m.width)
		m.form.Meta["old_name"] = row.Name
		m.mode = modeForm
	case tabMCP:
		row, ok := m.selectedMCP()
		if !ok {
			return
		}
		entry, _, err := config.ReadMCPRegistryEntry(row.Name)
		if err != nil {
			m.err = err.Error()
			return
		}
		m.form = newMCPForm("edit mcp", entry, m.width)
		m.form.Meta["old_name"] = row.Name
		m.mode = modeForm
	case tabMemory:
		row, ok := m.selectedMemory()
		if !ok {
			return
		}
		memory, err := config.ReadPortableMemory(row.ID, row.Scope)
		if err != nil {
			m.err = err.Error()
			return
		}
		m.form = newMemoryForm("edit memory", memory, m.width)
		m.form.Meta["old_id"] = row.ID
		m.form.Meta["old_scope"] = string(row.Scope)
		m.mode = modeForm
	}
}

func (m *Model) startDelete() {
	m.err = ""
	switch m.tab {
	case tabAgents:
		row, ok := m.selectedAgent()
		if !ok {
			return
		}
		m.confirm = confirmModel{
			Title:  "Delete agent?",
			Body:   fmt.Sprintf("Delete %s [%s]?", row.Name, row.Scope),
			Kind:   "agent",
			Target: map[string]string{"name": row.Name, "scope": string(row.Scope)},
		}
		m.mode = modeConfirm
	case tabEnvs:
		row, ok := m.selectedEnv()
		if !ok {
			return
		}
		kind := "env"
		body := "Delete env " + row.Name + "?"
		if row.Local {
			kind = "project_override"
			body = "Delete current project override?"
		}
		m.confirm = confirmModel{Title: "Delete environment?", Body: body, Kind: kind, Target: map[string]string{"name": row.Name}}
		m.mode = modeConfirm
	case tabMCP:
		row, ok := m.selectedMCP()
		if !ok {
			return
		}
		m.confirm = confirmModel{Title: "Delete MCP?", Body: "Delete MCP registry entry " + row.Name + "?", Kind: "mcp", Target: map[string]string{"name": row.Name}}
		m.mode = modeConfirm
	case tabMemory:
		row, ok := m.selectedMemory()
		if !ok {
			return
		}
		m.confirm = confirmModel{Title: "Delete memory?", Body: fmt.Sprintf("Delete memory %s [%s]?", row.ID, row.Scope), Kind: "memory", Target: map[string]string{"id": row.ID, "scope": string(row.Scope)}}
		m.mode = modeConfirm
	}
}

func (m *Model) activateSelected() {
	m.err = ""
	if m.opts.Activate == nil {
		m.err = "activation is not wired"
		return
	}
	switch m.tab {
	case tabAgents:
		row, ok := m.selectedAgent()
		if !ok {
			return
		}
		if err := m.opts.Activate(config.ActiveKindProfile, row.Name); err != nil {
			m.err = err.Error()
			return
		}
		m.message = "activated profile " + row.Name
		_ = m.refresh()
	case tabEnvs:
		row, ok := m.selectedEnv()
		if !ok || row.Local {
			return
		}
		if err := m.opts.Activate(config.ActiveKindEnv, row.Name); err != nil {
			m.err = err.Error()
			return
		}
		m.message = "activated env " + row.Name
		_ = m.refresh()
	}
}

func (m *Model) scanRuntimes() {
	if m.opts.ScanRuntimes == nil {
		m.err = "runtime scan is not wired"
		return
	}
	if err := m.opts.ScanRuntimes(); err != nil {
		m.err = err.Error()
		return
	}
	m.message = "runtime scan completed"
	_ = m.refresh()
}

func (m *Model) startMemoryImport() {
	if m.opts.ImportMemoryDryRun == nil {
		m.err = "memory import dry-run is not wired"
		return
	}
	m.form = newBasicForm("memory import dry-run", "memory_import", m.width, []basicField{
		{Key: "source", Label: "Source path or runtime", Value: ""},
	})
	m.mode = modeForm
}

func (m *Model) saveForm() error {
	values := m.formValues()
	switch m.form.Kind {
	case "agent":
		return m.saveAgent(values)
	case "env":
		return m.saveEnv(values)
	case "mcp":
		return m.saveMCP(values)
	case "memory":
		return m.saveMemory(values)
	case "memory_import":
		return m.runMemoryImport(values)
	default:
		return fmt.Errorf("unknown form kind %q", m.form.Kind)
	}
}

func (m *Model) saveAgent(values map[string]string) error {
	scope, err := parseAgentScope(values["scope"])
	if err != nil {
		return err
	}
	name := strings.TrimSpace(values["name"])
	if name == "" {
		return fmt.Errorf("agent name is required")
	}
	var agent *config.AgentProfile
	if oldName := m.form.Meta["old_name"]; oldName != "" {
		oldScope := config.Scope(m.form.Meta["old_scope"])
		agent, err = config.ReadAgent(oldName, oldScope, m.opts.CWD)
		if err != nil {
			return err
		}
	} else {
		agent = &config.AgentProfile{}
	}
	agent.Name = name
	agent.SourceScope = string(scope)
	agent.Description = strings.TrimSpace(values["description"])
	agent.Identity.DisplayName = strings.TrimSpace(values["display_name"])
	agent.Runtime.Preferred = strings.TrimSpace(values["runtime"])
	agent.Runtime.Fallback = commaList(values["fallback"])
	agent.ModelRun.Model = strings.TrimSpace(values["model"])
	agent.ModelRun.ReasoningEffort = strings.TrimSpace(values["reasoning"])
	agent.Capabilities.Skills = commaList(values["skills"])
	agent.Capabilities.MCPs = commaList(values["mcps"])
	refs, err := parseMemoryRefs(values["memory_refs"])
	if err != nil {
		return err
	}
	agent.MemoryRefs = refs
	agent.Instructions.Developer = strings.TrimSpace(values["developer"])
	if err := config.WriteAgent(agent, scope, m.opts.CWD); err != nil {
		return err
	}
	oldName := m.form.Meta["old_name"]
	oldScope := config.Scope(m.form.Meta["old_scope"])
	if oldName != "" && (oldName != name || oldScope != scope) {
		if err := config.DeleteAgent(oldName, oldScope, m.opts.CWD); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func (m *Model) saveEnv(values map[string]string) error {
	if m.form.Meta["local"] == "true" {
		extends := strings.TrimSpace(values["name"])
		if extends == "" {
			return fmt.Errorf("extends env name is required")
		}
		if _, err := config.ReadEnvironment(extends); err != nil {
			return err
		}
		override := &config.ProjectOverride{
			Extends:       extends,
			RuntimeAgents: runtimeAgentsFromValues(values),
		}
		return config.WriteProjectOverride(m.opts.CWD, override)
	}

	name := strings.TrimSpace(values["name"])
	if name == "" {
		return fmt.Errorf("env name is required")
	}
	env := &config.Environment{
		Name:          name,
		Description:   strings.TrimSpace(values["description"]),
		RuntimeAgents: runtimeAgentsFromValues(values),
	}
	for _, runtime := range knownRuntimeOrder() {
		if _, ok := env.RuntimeAgents[runtime]; ok {
			env.Targets = append(env.Targets, runtime)
		}
	}
	if err := config.WriteEnvironment(env); err != nil {
		return err
	}
	oldName := m.form.Meta["old_name"]
	if oldName != "" && oldName != name {
		if err := config.DeleteEnvironment(oldName); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func (m *Model) saveMCP(values map[string]string) error {
	name := strings.TrimSpace(values["name"])
	if name == "" {
		return fmt.Errorf("mcp name is required")
	}
	env, err := keyValueMap(values["env"])
	if err != nil {
		return err
	}
	headers, err := keyValueMap(values["headers"])
	if err != nil {
		return err
	}
	entry := &config.MCPRegistryEntry{
		Name:        name,
		Kind:        "mcp",
		Description: strings.TrimSpace(values["description"]),
		Server: config.MCPServerConfig{
			Type:    strings.TrimSpace(values["type"]),
			Command: strings.TrimSpace(values["command"]),
			Args:    commaList(values["args"]),
			Env:     env,
			URL:     strings.TrimSpace(values["url"]),
			Headers: headers,
		},
	}
	if err := config.WriteMCPRegistryEntry(entry); err != nil {
		return err
	}
	oldName := m.form.Meta["old_name"]
	if oldName != "" && oldName != name {
		if err := config.DeleteMCPRegistryEntry(oldName); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func (m *Model) saveMemory(values map[string]string) error {
	id := strings.TrimSpace(values["id"])
	scope := config.Scope(strings.TrimSpace(values["scope"]))
	if id == "" {
		return fmt.Errorf("memory id is required")
	}
	path := strings.TrimSpace(values["path"])
	if path == "" {
		path = config.MemoryPath(id, scope)
	}
	memory := &config.PortableMemory{
		ID:          id,
		Scope:       string(scope),
		Description: strings.TrimSpace(values["description"]),
		Format:      strings.TrimSpace(values["format"]),
		Path:        path,
		Mode:        strings.TrimSpace(values["mode"]),
		Tags:        commaList(values["tags"]),
	}
	if err := config.WritePortableMemory(memory); err != nil {
		return err
	}
	oldID := m.form.Meta["old_id"]
	oldScope := config.Scope(m.form.Meta["old_scope"])
	if oldID != "" && (oldID != id || oldScope != scope) {
		if err := config.DeletePortableMemory(oldID, oldScope); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func (m *Model) runMemoryImport(values map[string]string) error {
	source := strings.TrimSpace(values["source"])
	if source == "" {
		return fmt.Errorf("source is required")
	}
	report, err := m.opts.ImportMemoryDryRun(source)
	if err != nil {
		return err
	}
	m.previewTitle = "Memory import dry-run"
	m.previewBody = report
	m.mode = modePreview
	return nil
}

func (m *Model) runConfirm() error {
	switch m.confirm.Kind {
	case "agent":
		return config.DeleteAgent(m.confirm.Target["name"], config.Scope(m.confirm.Target["scope"]), m.opts.CWD)
	case "env":
		return config.DeleteEnvironment(m.confirm.Target["name"])
	case "project_override":
		return config.DeleteProjectOverride(m.opts.CWD)
	case "mcp":
		return config.DeleteMCPRegistryEntry(m.confirm.Target["name"])
	case "memory":
		return config.DeletePortableMemory(m.confirm.Target["id"], config.Scope(m.confirm.Target["scope"]))
	default:
		return fmt.Errorf("unknown delete target %q", m.confirm.Kind)
	}
}

func newAgentForm(title string, agent *config.AgentProfile, scope config.Scope, width int) formModel {
	if agent == nil {
		agent = &config.AgentProfile{
			SourceScope: string(scope),
			Runtime: config.RuntimePreferences{
				Preferred: "codex",
			},
			ModelRun: config.ModelRun{
				ReasoningEffort: "medium",
			},
		}
	}
	return newBasicForm(title, "agent", width, []basicField{
		{Key: "name", Label: "Name", Value: agent.Name},
		{Key: "scope", Label: "Scope", Value: empty(string(scope), config.ScopeGlobal)},
		{Key: "description", Label: "Description", Value: agent.Description},
		{Key: "display_name", Label: "Display name", Value: agent.Identity.DisplayName},
		{Key: "runtime", Label: "Preferred runtime", Value: agent.Runtime.Preferred},
		{Key: "fallback", Label: "Fallback runtimes", Value: strings.Join(agent.Runtime.Fallback, ",")},
		{Key: "model", Label: "Model", Value: agent.ModelRun.Model},
		{Key: "reasoning", Label: "Reasoning", Value: agent.ModelRun.ReasoningEffort},
		{Key: "skills", Label: "Skills", Value: strings.Join(agent.Capabilities.Skills, ",")},
		{Key: "mcps", Label: "MCP servers", Value: strings.Join(agent.Capabilities.MCPs, ",")},
		{Key: "memory_refs", Label: "Memory refs", Value: formatMemoryRefs(agent.MemoryRefs)},
		{Key: "developer", Label: "Developer instructions", Value: agent.Instructions.Developer},
	})
}

func newEnvForm(title string, env *config.Environment, local bool, width int) formModel {
	values := map[string]string{}
	if env != nil {
		values["name"] = env.Name
		values["description"] = env.Description
		for runtime, agent := range env.RuntimeAgents {
			values[runtime] = agent.Primary
		}
	}
	label := "Name"
	if local {
		label = "Extends env"
	}
	fields := []basicField{
		{Key: "name", Label: label, Value: values["name"]},
	}
	if !local {
		fields = append(fields, basicField{Key: "description", Label: "Description", Value: values["description"]})
	}
	for _, runtime := range knownRuntimeOrder() {
		fields = append(fields, basicField{Key: runtime, Label: runtime + " agent", Value: values[runtime]})
	}
	return newBasicForm(title, "env", width, fields)
}

func newMCPForm(title string, entry *config.MCPRegistryEntry, width int) formModel {
	if entry == nil {
		entry = &config.MCPRegistryEntry{Kind: "mcp"}
	}
	return newBasicForm(title, "mcp", width, []basicField{
		{Key: "name", Label: "Name", Value: entry.Name},
		{Key: "description", Label: "Description", Value: entry.Description},
		{Key: "type", Label: "Type", Value: entry.Server.Type},
		{Key: "command", Label: "Command", Value: entry.Server.Command},
		{Key: "args", Label: "Args", Value: strings.Join(entry.Server.Args, ",")},
		{Key: "env", Label: "Env K=V", Value: formatKeyValueMap(entry.Server.Env)},
		{Key: "url", Label: "URL", Value: entry.Server.URL},
		{Key: "headers", Label: "Headers K=V", Value: formatKeyValueMap(entry.Server.Headers)},
	})
}

func newMemoryForm(title string, memory *config.PortableMemory, width int) formModel {
	if memory == nil {
		memory = &config.PortableMemory{
			Scope:  string(config.ScopeProject),
			Format: "markdown",
			Mode:   "read",
		}
	}
	return newBasicForm(title, "memory", width, []basicField{
		{Key: "id", Label: "ID", Value: memory.ID},
		{Key: "scope", Label: "Scope", Value: memory.Scope},
		{Key: "description", Label: "Description", Value: memory.Description},
		{Key: "format", Label: "Format", Value: memory.Format},
		{Key: "path", Label: "Path", Value: memory.Path},
		{Key: "mode", Label: "Mode", Value: memory.Mode},
		{Key: "tags", Label: "Tags", Value: strings.Join(memory.Tags, ",")},
	})
}

type basicField struct {
	Key   string
	Label string
	Value string
}

func newBasicForm(title, kind string, width int, fields []basicField) formModel {
	out := formModel{
		Title: title,
		Kind:  kind,
		Meta:  map[string]string{},
	}
	inputWidth := clamp(width-28, 20, 90)
	for i, field := range fields {
		input := textinput.New()
		input.Width = inputWidth
		input.SetValue(field.Value)
		input.Prompt = ""
		if i == 0 {
			input.Focus()
		}
		out.Fields = append(out.Fields, formField{
			Key:   field.Key,
			Label: field.Label,
			Input: input,
		})
	}
	return out
}

func (m *Model) resizeFormInputs() {
	width := clamp(m.width-28, 20, 90)
	for i := range m.form.Fields {
		m.form.Fields[i].Input.Width = width
	}
}

func (m *Model) focusField(index int) {
	if len(m.form.Fields) == 0 {
		return
	}
	if index < 0 {
		index = len(m.form.Fields) - 1
	}
	if index >= len(m.form.Fields) {
		index = 0
	}
	for i := range m.form.Fields {
		m.form.Fields[i].Input.Blur()
	}
	m.form.Focus = index
	m.form.Fields[index].Input.Focus()
}

func (m Model) formValues() map[string]string {
	values := map[string]string{}
	for _, field := range m.form.Fields {
		values[field.Key] = field.Input.Value()
	}
	return values
}

func setFormValue(form *formModel, key, value string) {
	for i := range form.Fields {
		if form.Fields[i].Key == key {
			form.Fields[i].Input.SetValue(value)
			return
		}
	}
}

func (m Model) viewForm() string {
	width := max(60, m.width)
	lines := []string{m.styles.panelTitle.Render(m.form.Title), ""}
	for i, field := range m.form.Fields {
		label := m.styles.inputLabel.Width(22).Render(field.Label)
		line := label + field.Input.View()
		if i == m.form.Focus {
			line = m.styles.selectedRow.Width(width).Render(fitLine(line, width))
		}
		lines = append(lines, line)
	}
	if m.form.Err != "" {
		lines = append(lines, "", m.styles.error.Render(m.form.Err))
	}
	lines = append(lines, "", m.styles.muted.Render("Enter advances/saves, Ctrl+S saves, Esc cancels"))
	return fitBlock(strings.Join(lines, "\n"), width, m.height-1)
}

func (m Model) viewConfirm() string {
	lines := []string{
		m.styles.panelTitle.Render(m.confirm.Title),
		"",
		m.confirm.Body,
		"",
		m.styles.error.Render("Press y to confirm, Esc to cancel."),
	}
	return fitBlock(strings.Join(lines, "\n"), m.width, m.height-1)
}

func (m Model) viewPreview() string {
	lines := []string{
		m.styles.panelTitle.Render(m.previewTitle),
		"",
		m.previewBody,
		"",
		m.styles.muted.Render("Esc closes preview."),
	}
	return fitBlock(strings.Join(lines, "\n"), m.width, m.height-1)
}

func (m Model) viewHelp() string {
	lines := []string{
		m.styles.panelTitle.Render("Help"),
		"",
		"Navigation",
		"  left/right, h/l, Tab/Shift+Tab  switch top-level tabs",
		"  up/down, j/k                       move selection",
		"",
		"Actions",
		"  n  create item or create from selected runtime candidate",
		"  e  edit selected item",
		"  d  delete selected item after confirmation",
		"  u  activate selected agent profile or env",
		"  r  refresh data",
		"  x  scan runtimes from the Runtimes tab",
		"  i  memory import dry-run from the Memory tab",
		"  q  quit",
		"",
		"Forms",
		"  Enter advances fields and saves on the last field",
		"  Ctrl+S saves immediately",
		"  Esc cancels",
		"",
		"Press Esc, ?, or q to close help.",
	}
	return fitBlock(strings.Join(lines, "\n"), m.width, m.height-1)
}

func parseAgentScope(value string) (config.Scope, error) {
	scope := config.Scope(strings.TrimSpace(value))
	if scope == "" {
		return config.ScopeGlobal, nil
	}
	switch scope {
	case config.ScopeGlobal, config.ScopeProject, config.ScopeLocal:
		return scope, nil
	default:
		return "", fmt.Errorf("invalid scope %q", value)
	}
}

func runtimeAgentsFromValues(values map[string]string) map[string]config.RuntimeAgent {
	out := map[string]config.RuntimeAgent{}
	for _, runtime := range knownRuntimeOrder() {
		name := strings.TrimSpace(values[runtime])
		if name == "" {
			continue
		}
		out[runtime] = config.RuntimeAgent{Primary: name}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func knownRuntimeOrder() []string {
	return []string{"codex", "claude-code", "opencode", "cline", "cursor"}
}

func commaList(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func keyValueMap(value string) (map[string]string, error) {
	items := commaList(value)
	if len(items) == 0 {
		return nil, nil
	}
	out := map[string]string{}
	for _, item := range items {
		key, val, ok := strings.Cut(item, "=")
		if !ok {
			return nil, fmt.Errorf("expected K=V pair, got %q", item)
		}
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("empty key in %q", item)
		}
		out[key] = strings.TrimSpace(val)
	}
	return out, nil
}

func formatKeyValueMap(values map[string]string) string {
	if len(values) == 0 {
		return ""
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+values[key])
	}
	return strings.Join(parts, ",")
}

func parseMemoryRefs(value string) ([]config.MemoryRef, error) {
	items := commaList(value)
	refs := make([]config.MemoryRef, 0, len(items))
	for _, item := range items {
		parts := strings.Split(item, ":")
		if len(parts) > 4 {
			return nil, fmt.Errorf("invalid memory ref %q", item)
		}
		id := strings.TrimSpace(parts[0])
		if id == "" {
			return nil, fmt.Errorf("memory ref id is required")
		}
		scope := string(config.ScopeProject)
		path := ""
		mode := "read"
		if len(parts) > 1 && strings.TrimSpace(parts[1]) != "" {
			scope = strings.TrimSpace(parts[1])
		}
		if len(parts) > 2 {
			path = strings.TrimSpace(parts[2])
		}
		if len(parts) > 3 && strings.TrimSpace(parts[3]) != "" {
			mode = strings.TrimSpace(parts[3])
		}
		if path == "" {
			path = config.MemoryPath(id, config.Scope(scope))
		}
		refs = append(refs, config.MemoryRef{ID: id, Scope: scope, Path: path, Mode: mode})
	}
	return refs, nil
}

func formatMemoryRefs(refs []config.MemoryRef) string {
	parts := make([]string, 0, len(refs))
	for _, ref := range refs {
		parts = append(parts, strings.Join([]string{ref.ID, ref.Scope, ref.Path, ref.Mode}, ":"))
	}
	return strings.Join(parts, ",")
}

func fitBlock(text string, width, height int) string {
	width = max(20, width)
	height = max(1, height)
	lines := strings.Split(text, "\n")
	out := make([]string, 0, height)
	for i := 0; i < len(lines) && i < height; i++ {
		out = append(out, fitLine(lines[i], width))
	}
	for len(out) < height {
		out = append(out, strings.Repeat(" ", width))
	}
	return strings.Join(out, "\n")
}

func fitLine(value string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(value) > width {
		runes := []rune(value)
		if width <= 1 {
			return string(runes[:1])
		}
		if len(runes) > width-1 {
			return string(runes[:width-1]) + "..."
		}
	}
	padding := width - lipgloss.Width(value)
	if padding > 0 {
		return value + strings.Repeat(" ", padding)
	}
	return value
}

func emptyDetail(title, message string) string {
	return title + "\n\n" + message
}

func kv(key, value string) string {
	return key + ": " + empty(value, "none")
}

func formatActive(ref config.ActiveRef) string {
	if ref.Kind == "" || ref.Name == "" {
		return "none"
	}
	return ref.Kind + ":" + ref.Name
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func keyString(msg tea.KeyMsg) string {
	return msg.String()
}

func empty(value string, fallback any) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fmt.Sprint(fallback)
}

func clamp(value, low, high int) int {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
