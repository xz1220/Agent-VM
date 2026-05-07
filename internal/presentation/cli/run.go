package cli

import (
	"github.com/spf13/cobra"

	"github.com/xz1220/agent-vm/internal/app/model"
)

func newRunCmd(deps Deps) *cobra.Command {
	var (
		runtime        string
		preview        bool
		nonInteractive bool
	)
	cmd := &cobra.Command{
		Use:   "run <agent>",
		Short: "Run an Agent through a runtime (PRD §4.4)",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			req := model.RunRequest{
				Agent:          args[0],
				Runtime:        runtime,
				NonInteractive: nonInteractive,
			}
			if preview {
				p, err := deps.Services.Run.Preview(c.Context(), req)
				if err != nil {
					return err
				}
				return RenderRunPreview(c.OutOrStdout(), p)
			}
			res, err := deps.Services.Run.Run(c.Context(), req)
			if err != nil {
				return err
			}
			return RenderRunResult(c.OutOrStdout(), res)
		},
	}
	cmd.Flags().StringVar(&runtime, "runtime", "", "target runtime")
	cmd.Flags().BoolVar(&preview, "preview", false, "show plan, do not launch")
	cmd.Flags().BoolVar(&nonInteractive, "non-interactive", false, "fail instead of prompting")
	return cmd
}
