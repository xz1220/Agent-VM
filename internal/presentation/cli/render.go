package cli

import (
	"fmt"
	"io"

	"github.com/xz1220/agent-vm/internal/app/model"
	"github.com/xz1220/agent-vm/internal/presentation/render"
)

// These are intentionally simple stubs in the skeleton. Real layouts
// (tables, diff blocks, mapping markers) come during the presentation
// implementation pass.

func RenderAgentList(w io.Writer, items []model.AgentSummary) error {
	if len(items) == 0 {
		return render.Linef(w, "(no agents)")
	}
	for _, it := range items {
		if err := render.Linef(w, "%s\t%s", it.Name, it.Description); err != nil {
			return err
		}
	}
	return nil
}

func RenderAgentDetail(w io.Writer, d *model.AgentDetail) error {
	return render.JSON(w, d)
}

func RenderRunPreview(w io.Writer, p *model.RunPreview) error {
	return render.JSON(w, p)
}

func RenderRunResult(w io.Writer, r *model.RunResult) error {
	return render.Linef(w, "exit=%d", r.ExitCode)
}

func RenderPackageList(w io.Writer, items []model.PackageSummary) error {
	if len(items) == 0 {
		return render.Linef(w, "(no packages)")
	}
	for _, it := range items {
		if err := render.Linef(w, "%s\t%s\t%s", it.Name, it.Version, it.Description); err != nil {
			return err
		}
	}
	return nil
}

func RenderPackageDetail(w io.Writer, d *model.PackageDetail) error {
	return render.JSON(w, d)
}

func RenderDoctor(w io.Writer, r *model.DoctorReport) error {
	return render.JSON(w, r)
}

func RenderStatus(w io.Writer, r *model.StatusReport) error {
	return render.JSON(w, r)
}

// ensure fmt is referenced even when render uses Linef only
var _ = fmt.Sprintf
