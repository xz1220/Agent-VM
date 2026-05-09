package model

// InitResult is the outcome of `avm init`. AlreadyExists means the
// AVM home was already laid out; CreatedPaths is empty in that case.
type InitResult struct {
	Root          string   `json:"root"`
	AlreadyExists bool     `json:"already_exists,omitempty"`
	CreatedPaths  []string `json:"created_paths,omitempty"`
}

// UninstallResult is the outcome of `avm uninstall` for the AVM home.
// CLI is responsible for removing the binary itself (which lives
// outside the home directory and is process-self-aware via os.Executable).
type UninstallResult struct {
	Root    string `json:"root"`
	Removed bool   `json:"removed,omitempty"`
}
