package runtime

import (
	"errors"
	"fmt"
	"sort"
	"sync"
)

// Registry resolves a runtime name to a Driver and lists known drivers.
// Application layer never imports a specific driver package directly.
type Registry interface {
	Resolve(name string) (Driver, error)
	List() []DriverInfo
}

// DriverInfo is a non-Driver projection used by listings/diagnostics.
type DriverInfo struct {
	Name string
}

// ErrUnknownRuntime is returned by Registry.Resolve for unknown names.
var ErrUnknownRuntime = errors.New("runtime: unknown")

// MapRegistry is a default in-memory Registry implementation.
type MapRegistry struct {
	mu      sync.RWMutex
	drivers map[string]Driver
}

// NewRegistry constructs an empty MapRegistry. Callers register
// drivers with Register before resolving.
func NewRegistry() *MapRegistry { return &MapRegistry{drivers: map[string]Driver{}} }

// Register associates name -> driver. It is an error to register the
// same name twice; that signals a wiring bug.
func (r *MapRegistry) Register(d Driver) error {
	if d == nil {
		return errors.New("registry: nil driver")
	}
	name := d.Name()
	if name == "" {
		return errors.New("registry: empty driver name")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.drivers[name]; exists {
		return fmt.Errorf("registry: %q already registered", name)
	}
	r.drivers[name] = d
	return nil
}

// Resolve implements Registry.
func (r *MapRegistry) Resolve(name string) (Driver, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	d, ok := r.drivers[name]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownRuntime, name)
	}
	return d, nil
}

// List implements Registry.
func (r *MapRegistry) List() []DriverInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]DriverInfo, 0, len(r.drivers))
	for name := range r.drivers {
		out = append(out, DriverInfo{Name: name})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
