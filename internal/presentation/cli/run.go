package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/xz1220/agent-vm/internal/app/model"
)

func newRunCmd(deps Deps) *cobra.Command {
	var (
		runtime string
		preview bool
		drift   string
	)
	cmd := &cobra.Command{
		Use:   "run <agent>",
		Short: "Run an Agent through a runtime (PRD §4.4)",
		Long: `Run an Agent on a runtime.

Pass --runtime when the Agent has multiple runtimes configured (the service
returns RUNTIME_AMBIGUOUS otherwise).
Pass --drift {keep|merge|discard} to acknowledge drift between AVM Agent
definition and existing managed config (the service returns DRIFT_DETECTED
when drift exists and --drift is unset).
Pass --preview to render the plan without launching.`,
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			req := model.RunRequest{
				Agent:       args[0],
				Runtime:     runtime,
				DriftPolicy: model.DriftPolicy(drift),
			}
			g := globalFlags(c)

			pv, err := deps.Services.Run.Preview(c.Context(), req)
			if err != nil {
				return err
			}
			if preview {
				if g.JSON {
					return jsonWrite(c.OutOrStdout(), pv)
				}
				return RenderRunPreview(c.OutOrStdout(), pv)
			}

			res, err := deps.Services.Run.Run(c.Context(), req)
			if err != nil {
				return err
			}
			if g.JSON {
				if err := jsonWrite(c.OutOrStdout(), res); err != nil {
					return err
				}
			} else if err := RenderRunResult(c.OutOrStdout(), res); err != nil {
				return err
			}
			if res != nil && res.ExitCode != 0 {
				return &exitCodeError{code: res.ExitCode}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&runtime, "runtime", "", "target runtime")
	cmd.Flags().BoolVar(&preview, "preview", false, "show plan, do not launch")
	cmd.Flags().StringVar(&drift, "drift", "", "keep|merge|discard (required when drift detected)")
	return cmd
}

// exitCodeError carries a non-zero exit code through cobra so main.go
// can map it to os.Exit.
type exitCodeError struct{ code int }

func (e *exitCodeError) Error() string {
	return fmt.Sprintf("run: exit code %d", e.code)
}

// ExitCode lets main.go translate to os.Exit.
func (e *exitCodeError) ExitCode() int { return e.code }
