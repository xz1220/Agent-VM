package service

// Container bundles the application-layer services so the presentation
// layer depends on a single composition root. Construction is in
// cmd/avm/main.go (or a thin app bootstrapper); presentation never
// imports infra/runtime concrete packages directly.
type Container struct {
	Agents       AgentService
	Run          RunService
	Packages     PackageService
	Capabilities CapabilityService
	Diagnostics  DiagnosticsService
}
