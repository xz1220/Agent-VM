package cli

import (
	"github.com/spf13/cobra"

	"github.com/xz1220/agent-vm/internal/app/model"
)

func newCapabilityCmd(deps Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "capability",
		Short: "Discover and import runtime-global capabilities (PRD §4.2)",
	}
	cmd.AddCommand(newCapabilityDiscoverCmd(deps))
	cmd.AddCommand(newCapabilityImportCmd(deps))
	cmd.AddCommand(newCapabilityBootstrapCmd(deps))
	return cmd
}

// ----------------------------------------------------------------------------
// avm capability discover
// ----------------------------------------------------------------------------

func newCapabilityDiscoverCmd(deps Deps) *cobra.Command {
	var (
		runtimes []string
		kinds    []string
	)
	cmd := &cobra.Command{
		Use:   "discover",
		Short: "List capabilities from AVM store and runtime globals",
		Long: `List every capability candidate AVM can see right now: AVM-managed
records plus live runtime-global discoveries. Same-(kind,name) across
sources is flagged Conflict; runtime-global candidates already imported
into capstore are flagged Imported.`,
		RunE: func(c *cobra.Command, args []string) error {
			req := model.DiscoverRequest{
				Runtimes: runtimes,
				Kinds:    capKindFlags(kinds),
			}
			cands, err := deps.Services.Capabilities.Discover(c.Context(), req)
			if err != nil {
				return err
			}
			g := globalFlags(c)
			if g.JSON {
				return jsonWrite(c.OutOrStdout(), cands)
			}
			return RenderCapabilityList(c.OutOrStdout(), cands)
		},
	}
	cmd.Flags().StringSliceVar(&runtimes, "runtime", nil, "filter by runtime name (repeatable)")
	cmd.Flags().StringSliceVar(&kinds, "kind", nil, "filter by kind: skill|mcp (repeatable)")
	return cmd
}

// ----------------------------------------------------------------------------
// avm capability import
// ----------------------------------------------------------------------------

func newCapabilityImportCmd(deps Deps) *cobra.Command {
	var (
		runtime    string
		kind       string
		name       string
		onConflict string
	)
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import a single runtime-global capability into capstore",
		Long: `Copy one runtime-global capability (skill or mcp) into the AVM
capability store so Agents can reference it.

Required: --runtime, --kind, --name
Conflicts: pass --on-conflict {skip|overwrite|cancel} when the same
(kind,name) already exists in capstore with different content.`,
		RunE: func(c *cobra.Command, args []string) error {
			req := model.ImportCapabilityRequest{
				Runtime:    runtime,
				Kind:       model.CapabilityKind(kind),
				Name:       name,
				OnConflict: model.ConflictResolution(onConflict),
			}
			res, err := deps.Services.Capabilities.Import(c.Context(), req)
			if err != nil {
				return err
			}
			g := globalFlags(c)
			if g.JSON {
				return jsonWrite(c.OutOrStdout(), res)
			}
			return RenderImportResult(c.OutOrStdout(), res)
		},
	}
	cmd.Flags().StringVar(&runtime, "runtime", "", "runtime name (required)")
	cmd.Flags().StringVar(&kind, "kind", "", "capability kind: skill|mcp (required)")
	cmd.Flags().StringVar(&name, "name", "", "capability name in the runtime (required)")
	cmd.Flags().StringVar(&onConflict, "on-conflict", "", "skip|overwrite|cancel (default cancel)")
	return cmd
}

// ----------------------------------------------------------------------------
// avm capability bootstrap
// ----------------------------------------------------------------------------

func newCapabilityBootstrapCmd(deps Deps) *cobra.Command {
	var (
		runtime    string
		kinds      []string
		onConflict string
	)
	cmd := &cobra.Command{
		Use:   "bootstrap",
		Short: "Import every runtime-global capability for a runtime",
		Long: `Run capability discovery for the named runtime and import every
runtime-global capability into the AVM capability store. Per-item
failures are collected as 'skipped' rather than aborting the whole run.

Typical use: first install of AVM on a machine that already has codex /
claude / opencode skills installed.`,
		RunE: func(c *cobra.Command, args []string) error {
			req := model.BootstrapCapabilitiesRequest{
				Runtime:    runtime,
				Kinds:      capKindFlags(kinds),
				OnConflict: model.ConflictResolution(onConflict),
			}
			res, err := deps.Services.Capabilities.Bootstrap(c.Context(), req)
			if err != nil {
				return err
			}
			g := globalFlags(c)
			if g.JSON {
				return jsonWrite(c.OutOrStdout(), res)
			}
			return RenderBootstrapResult(c.OutOrStdout(), res)
		},
	}
	cmd.Flags().StringVar(&runtime, "runtime", "", "runtime name (required)")
	cmd.Flags().StringSliceVar(&kinds, "kind", nil, "filter by kind: skill|mcp (repeatable; empty = all)")
	cmd.Flags().StringVar(&onConflict, "on-conflict", "", "skip|overwrite|cancel applied to every item")
	return cmd
}

// capKindFlags converts repeated --kind flags into typed model values.
// Unknown values are passed through as-is so the service layer surfaces
// the validation error consistently with other commands.
func capKindFlags(in []string) []model.CapabilityKind {
	if len(in) == 0 {
		return nil
	}
	out := make([]model.CapabilityKind, 0, len(in))
	for _, k := range in {
		out = append(out, model.CapabilityKind(k))
	}
	return out
}
