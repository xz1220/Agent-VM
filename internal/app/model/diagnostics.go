package model

// DoctorReport is the diagnostic output of `avm doctor`.
type DoctorReport struct {
	AVMHome          CheckResult
	PATH             CheckResult
	ShellIntegration CheckResult
	Runtimes         []RuntimeCheck
}

// RuntimeCheck reports per-runtime detection state.
type RuntimeCheck struct {
	Runtime   string
	Available bool
	Binary    string
	Version   string
	Issues    []string
}

// CheckResult is a generic boolean+detail pair used by DoctorReport.
type CheckResult struct {
	OK     bool
	Detail string
}

// StatusReport answers "what is AVM's current state?". It is a
// snapshot of agents, runtimes and recent run history.
type StatusReport struct {
	Agents     []AgentSummary
	Runtimes   []RuntimeCheck
	RecentRuns []RunRecord
}
