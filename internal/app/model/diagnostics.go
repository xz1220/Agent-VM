package model

// DoctorReport is the diagnostic output of `avm doctor`.
type DoctorReport struct {
	AVMHome          CheckResult    `json:"avm_home"`
	PATH             CheckResult    `json:"path"`
	ShellIntegration CheckResult    `json:"shell_integration"`
	Runtimes         []RuntimeCheck `json:"runtimes,omitempty"`
}

// RuntimeCheck reports per-runtime detection state.
type RuntimeCheck struct {
	Runtime   string   `json:"runtime"`
	Available bool     `json:"available"`
	Binary    string   `json:"binary,omitempty"`
	Version   string   `json:"version,omitempty"`
	Issues    []string `json:"issues,omitempty"`
}

// CheckResult is a generic boolean+detail pair used by DoctorReport.
type CheckResult struct {
	OK     bool   `json:"ok"`
	Detail string `json:"detail,omitempty"`
}

// StatusReport answers "what is AVM's current state?". It is a
// snapshot of agents, runtimes and recent run history.
type StatusReport struct {
	Agents     []AgentSummary `json:"agents,omitempty"`
	Runtimes   []RuntimeCheck `json:"runtimes,omitempty"`
	RecentRuns []RunRecord    `json:"recent_runs,omitempty"`
}
