package cli

import (
	"github.com/spf13/cobra"

	"github.com/xz1220/agent-vm/internal/app/model"
)

func newAgentCmd(deps Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Manage AVM Agents (PRD §4.2)",
	}
	cmd.AddCommand(newAgentCreateCmd(deps))
	cmd.AddCommand(newAgentListCmd(deps))
	cmd.AddCommand(newAgentShowCmd(deps))
	cmd.AddCommand(newAgentEditCmd(deps))
	cmd.AddCommand(newAgentDeleteCmd(deps))
	cmd.AddCommand(newAgentCloneCmd(deps))
	cmd.AddCommand(newAgentRenameCmd(deps))
	return cmd
}

func newAgentCreateCmd(deps Deps) *cobra.Command {
	var nonInteractive bool
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new Agent",
		RunE: func(c *cobra.Command, args []string) error {
			req := model.CreateAgentRequest{NonInteractive: nonInteractive}
			_, err := deps.Services.Agents.Create(c.Context(), req)
			return err
		},
	}
	cmd.Flags().BoolVar(&nonInteractive, "non-interactive", false, "fail instead of prompting")
	return cmd
}

func newAgentListCmd(deps Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List Agents",
		RunE: func(c *cobra.Command, args []string) error {
			summaries, err := deps.Services.Agents.List(c.Context())
			if err != nil {
				return err
			}
			return RenderAgentList(c.OutOrStdout(), summaries)
		},
	}
}

func newAgentShowCmd(deps Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "show <name>",
		Short: "Show one Agent",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			detail, err := deps.Services.Agents.Show(c.Context(), args[0])
			if err != nil {
				return err
			}
			return RenderAgentDetail(c.OutOrStdout(), detail)
		},
	}
}

func newAgentEditCmd(deps Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "edit <name>",
		Short: "Edit an Agent",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			_, err := deps.Services.Agents.Edit(c.Context(), model.EditAgentRequest{Name: args[0]})
			return err
		},
	}
}

func newAgentDeleteCmd(deps Deps) *cobra.Command {
	var confirm bool
	cmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete an Agent",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return deps.Services.Agents.Delete(c.Context(), model.DeleteAgentRequest{
				Name:    args[0],
				Confirm: confirm,
			})
		},
	}
	cmd.Flags().BoolVar(&confirm, "yes", false, "confirm deletion (required in non-interactive mode)")
	return cmd
}

func newAgentCloneCmd(deps Deps) *cobra.Command {
	var newName string
	cmd := &cobra.Command{
		Use:   "clone <name>",
		Short: "Clone an Agent",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			_, err := deps.Services.Agents.Clone(c.Context(), args[0], newName)
			return err
		},
	}
	cmd.Flags().StringVar(&newName, "name", "", "new Agent name")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newAgentRenameCmd(deps Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "rename <old> <new>",
		Short: "Rename an Agent",
		Args:  cobra.ExactArgs(2),
		RunE: func(c *cobra.Command, args []string) error {
			_, err := deps.Services.Agents.Rename(c.Context(), args[0], args[1])
			return err
		},
	}
}
