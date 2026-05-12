package service

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/xz1220/agent-vm/internal/app/model"
	"github.com/xz1220/agent-vm/internal/infra/agentstore"
	"github.com/xz1220/agent-vm/internal/infra/capstore"
	"github.com/xz1220/agent-vm/internal/infra/packageio"
)

type remappingCapStore struct {
	inner capstore.Store
	ids   []model.CapabilityID
	next  int
	recs  map[model.CapabilityID]model.CapabilityRecord
}

func newRemappingCapStore(inner capstore.Store, ids ...model.CapabilityID) *remappingCapStore {
	return &remappingCapStore{inner: inner, ids: ids, recs: map[model.CapabilityID]model.CapabilityRecord{}}
}

func (s *remappingCapStore) List() ([]model.CapabilityRecord, error) {
	out := make([]model.CapabilityRecord, 0, len(s.recs))
	for _, rec := range s.recs {
		out = append(out, rec)
	}
	return out, nil
}

func (s *remappingCapStore) Get(id model.CapabilityID) (model.CapabilityRecord, error) {
	if rec, ok := s.recs[id]; ok {
		return rec, nil
	}
	return s.inner.Get(id)
}

func (s *remappingCapStore) Add(rec model.CapabilityRecord, payload io.Reader) (model.CapabilityID, error) {
	if s.next >= len(s.ids) {
		return s.inner.Add(rec, payload)
	}
	id := s.ids[s.next]
	s.next++
	if _, err := io.ReadAll(payload); err != nil {
		return "", err
	}
	full := rec
	full.ID = id
	s.recs[id] = full
	return id, nil
}

func (s *remappingCapStore) Materialize(ids []model.CapabilityID, target string) error {
	return s.inner.Materialize(ids, target)
}

func (s *remappingCapStore) ReadPayload(id model.CapabilityID) ([]byte, error) {
	return s.inner.ReadPayload(id)
}

func (s *remappingCapStore) Remove(id model.CapabilityID) error {
	delete(s.recs, id)
	return s.inner.Remove(id)
}

// buildTestPkg writes a package zip to disk and returns the path.
func buildTestPkg(t *testing.T, dir string, manifest *model.PackageManifest, files map[string][]byte) string {
	t.Helper()
	dst := filepath.Join(dir, manifest.Name+".avm.zip")
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, data := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := packageio.New().Write(manifest, bytes.NewReader(buf.Bytes()), dst); err != nil {
		t.Fatalf("Write: %v", err)
	}
	return dst
}

func sumHex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func TestPackages_Install_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	skill := []byte("# skill content\n")
	agentYAML := []byte(`identity:
  name: alpha
runtimes:
  - runtime: codex
    default: true
skills:
  - id: demo-skill
    kind: skill
`)
	manifest := &model.PackageManifest{
		SchemaVersion: "1",
		Name:          "demo",
		Version:       "0.1.0",
		CreatedAt:     time.Unix(1, 0).UTC(),
		Agents: []model.PackageAgentRef{
			{Name: "alpha", Path: "agents/alpha.yaml"},
		},
		Capabilities: []model.PackageCapBlob{
			{
				Kind:     model.CapabilityKindSkill,
				Name:     "demo-skill",
				Path:     "capabilities/demo-skill.md",
				Checksum: sumHex(skill),
			},
		},
	}
	pkgPath := buildTestPkg(t, dir, manifest, map[string][]byte{
		"agents/alpha.yaml":          agentYAML,
		"capabilities/demo-skill.md": skill,
	})

	agents := agentstore.New(t.TempDir())
	caps := capstore.New(t.TempDir())
	pkgs := NewPackages(agents, caps, packageio.New())

	res, err := pkgs.Install(context.Background(), model.InstallRequest{Source: pkgPath})
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if len(res.InstalledAgents) != 1 || res.InstalledAgents[0] != "alpha" {
		t.Fatalf("InstalledAgents=%+v", res.InstalledAgents)
	}
	if len(res.ImportedCaps) != 1 {
		t.Fatalf("ImportedCaps=%+v", res.ImportedCaps)
	}

	// CapabilityRef inside agent should now point at the local store ID.
	got, err := agents.Get("alpha")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got.Skills) != 1 {
		t.Fatalf("skills=%+v", got.Skills)
	}
	if got.Skills[0].ID != res.ImportedCaps[0] {
		t.Fatalf("skill id %q want imported %q", got.Skills[0].ID, res.ImportedCaps[0])
	}

	// Idempotent capability import: installing again should reuse ID.
	if err := agents.Delete("alpha"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	res2, err := pkgs.Install(context.Background(), model.InstallRequest{Source: pkgPath})
	if err != nil {
		t.Fatalf("Install #2: %v", err)
	}
	if res.ImportedCaps[0] != res2.ImportedCaps[0] {
		t.Fatalf("cap id changed across installs: %s vs %s", res.ImportedCaps[0], res2.ImportedCaps[0])
	}
}

func TestPackages_Install_RequiresRuntime(t *testing.T) {
	dir := t.TempDir()
	manifest := &model.PackageManifest{
		SchemaVersion: "1", Name: "demo", Version: "0",
		Agents: []model.PackageAgentRef{{Name: "alpha", Path: "agents/alpha.yaml"}},
	}
	pkgPath := buildTestPkg(t, dir, manifest, map[string][]byte{
		"agents/alpha.yaml": []byte("identity:\n  name: alpha\n"),
	})
	agents := agentstore.New(t.TempDir())
	pkgs := NewPackages(agents, capstore.New(t.TempDir()), packageio.New())
	_, err := pkgs.Install(context.Background(), model.InstallRequest{Source: pkgPath})
	if err == nil {
		t.Fatal("expected MISSING_INPUT for package agent with no runtimes")
	}
	se := AsError(err)
	if se == nil || se.Code != CodeMissingInput {
		t.Fatalf("expected MISSING_INPUT, got %T %v", err, err)
	}
	if got, _ := se.Details["field"].(string); got != "runtime" {
		t.Fatalf("details.field=%v want runtime", se.Details["field"])
	}
	if agents.Exists("alpha") {
		t.Fatal("agent should not be persisted on validation failure")
	}
}

func TestPackages_Install_ConflictAskNonInteractive(t *testing.T) {
	dir := t.TempDir()
	manifest := &model.PackageManifest{
		SchemaVersion: "1", Name: "demo", Version: "0",
		Agents: []model.PackageAgentRef{{Name: "alpha", Path: "agents/alpha.yaml"}},
	}
	pkgPath := buildTestPkg(t, dir, manifest, map[string][]byte{
		"agents/alpha.yaml": []byte("identity:\n  name: alpha\nruntimes:\n  - runtime: codex\n"),
	})
	agents := agentstore.New(t.TempDir())
	if err := agents.Save(&model.Agent{Identity: model.Identity{Name: "alpha"}}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	pkgs := NewPackages(agents, capstore.New(t.TempDir()), packageio.New())
	_, err := pkgs.Install(context.Background(), model.InstallRequest{
		Source:     pkgPath,
		Resolution: model.ResolveAsk,
	})
	if !errors.Is(err, agentstore.ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

func TestPackages_Install_ConflictRename(t *testing.T) {
	dir := t.TempDir()
	manifest := &model.PackageManifest{
		SchemaVersion: "1", Name: "demo", Version: "0",
		Agents: []model.PackageAgentRef{{Name: "alpha", Path: "agents/alpha.yaml"}},
	}
	pkgPath := buildTestPkg(t, dir, manifest, map[string][]byte{
		"agents/alpha.yaml": []byte("identity:\n  name: alpha\nruntimes:\n  - runtime: codex\n"),
	})
	agents := agentstore.New(t.TempDir())
	if err := agents.Save(&model.Agent{Identity: model.Identity{Name: "alpha"}}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	pkgs := NewPackages(agents, capstore.New(t.TempDir()), packageio.New())
	res, err := pkgs.Install(context.Background(), model.InstallRequest{
		Source:     pkgPath,
		Resolution: model.ResolveRename,
	})
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if res.Renamed["alpha"] != "alpha-1" {
		t.Fatalf("Renamed=%+v", res.Renamed)
	}
	if !agents.Exists("alpha-1") {
		t.Fatal("renamed agent not on disk")
	}
}

func TestPackages_Install_ConflictSkip(t *testing.T) {
	dir := t.TempDir()
	manifest := &model.PackageManifest{
		SchemaVersion: "1", Name: "demo", Version: "0",
		Agents: []model.PackageAgentRef{{Name: "alpha", Path: "agents/alpha.yaml"}},
	}
	pkgPath := buildTestPkg(t, dir, manifest, map[string][]byte{
		"agents/alpha.yaml": []byte("identity:\n  name: alpha\n  description: pkg\nruntimes:\n  - runtime: codex\n"),
	})
	agents := agentstore.New(t.TempDir())
	if err := agents.Save(&model.Agent{Identity: model.Identity{Name: "alpha", Description: "original"}}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	pkgs := NewPackages(agents, capstore.New(t.TempDir()), packageio.New())
	res, err := pkgs.Install(context.Background(), model.InstallRequest{
		Source:     pkgPath,
		Resolution: model.ResolveSkip,
	})
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if len(res.Skipped) != 1 {
		t.Fatalf("Skipped=%+v", res.Skipped)
	}
	got, err := agents.Get("alpha")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Identity.Description != "original" {
		t.Fatalf("agent was overwritten by skip: %+v", got.Identity)
	}
}

func TestPackages_Install_ConflictOverwrite(t *testing.T) {
	dir := t.TempDir()
	manifest := &model.PackageManifest{
		SchemaVersion: "1", Name: "demo", Version: "0",
		Agents: []model.PackageAgentRef{{Name: "alpha", Path: "agents/alpha.yaml"}},
	}
	pkgPath := buildTestPkg(t, dir, manifest, map[string][]byte{
		"agents/alpha.yaml": []byte("identity:\n  name: alpha\n  description: pkg\nruntimes:\n  - runtime: codex\n"),
	})
	agents := agentstore.New(t.TempDir())
	if err := agents.Save(&model.Agent{Identity: model.Identity{Name: "alpha", Description: "original"}}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	pkgs := NewPackages(agents, capstore.New(t.TempDir()), packageio.New())
	if _, err := pkgs.Install(context.Background(), model.InstallRequest{
		Source:     pkgPath,
		Resolution: model.ResolveOverwrite,
	}); err != nil {
		t.Fatalf("Install: %v", err)
	}
	got, err := agents.Get("alpha")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Identity.Description != "pkg" {
		t.Fatalf("agent not overwritten: %+v", got.Identity)
	}
}

func TestPackages_Inspect(t *testing.T) {
	dir := t.TempDir()
	manifest := &model.PackageManifest{
		SchemaVersion: "1", Name: "demo", Version: "0",
	}
	pkgPath := buildTestPkg(t, dir, manifest, map[string][]byte{
		"some/extra.txt": []byte("hi"),
	})
	pkgs := NewPackages(nil, nil, packageio.New())
	det, err := pkgs.Inspect(context.Background(), pkgPath)
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if det.Manifest.Name != "demo" {
		t.Fatalf("manifest=%+v", det.Manifest)
	}
	if det.Source != pkgPath {
		t.Fatalf("source=%q", det.Source)
	}
	if len(det.Files) == 0 {
		t.Fatal("Files empty")
	}
}

func TestPackages_Uninstall(t *testing.T) {
	agents := agentstore.New(t.TempDir())
	if err := agents.Save(&model.Agent{Identity: model.Identity{Name: "alpha"}}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	pkgs := NewPackages(agents, capstore.New(t.TempDir()), packageio.New())
	if err := pkgs.Uninstall(context.Background(), "alpha"); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if agents.Exists("alpha") {
		t.Fatal("agent still exists after uninstall")
	}
}

func TestPackages_Export_RoundTrip(t *testing.T) {
	agentDir := t.TempDir()
	capDir := t.TempDir()
	outDir := t.TempDir()

	agents := agentstore.New(agentDir)
	caps := capstore.New(capDir)

	// Add a capability and reference it from an agent.
	id, err := caps.Add(model.CapabilityRecord{
		Kind: model.CapabilityKindSkill, Name: "demo-skill",
	}, bytes.NewReader([]byte("# skill body\n")))
	if err != nil {
		t.Fatalf("caps.Add: %v", err)
	}
	if err := agents.Save(&model.Agent{
		Identity: model.Identity{Name: "alpha", Description: "exp"},
		Skills:   []model.CapabilityRef{{ID: id, Kind: model.CapabilityKindSkill}},
	}); err != nil {
		t.Fatalf("agents.Save: %v", err)
	}

	pkgs := NewPackages(agents, caps, packageio.New())
	out := filepath.Join(outDir, "alpha.avm.zip")
	res, err := pkgs.Export(context.Background(), model.ExportRequest{
		Agent:         "alpha",
		IncludeSkills: true,
		OutputPath:    out,
	})
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if res.Path != out {
		t.Fatalf("path=%q", res.Path)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("stat: %v", err)
	}

	// Verify the package: should be readable, contain the agent and skill.
	pio := packageio.New()
	if err := pio.Verify(out); err != nil {
		t.Fatalf("Verify: %v", err)
	}
	mf, h, err := pio.Read(out)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	defer h.Close()
	if mf.Name != "alpha" {
		t.Fatalf("manifest name=%q", mf.Name)
	}
	if len(mf.Agents) != 1 || len(mf.Capabilities) != 1 {
		t.Fatalf("manifest=%+v", mf)
	}

	// Read agent yaml and confirm it parses.
	rc, err := h.Open(mf.Agents[0].Path)
	if err != nil {
		t.Fatalf("Open agent: %v", err)
	}
	data, _ := io.ReadAll(rc)
	rc.Close()
	var got model.Agent
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("agent yaml: %v", err)
	}
	if got.Identity.Name != "alpha" {
		t.Fatalf("agent=%+v", got.Identity)
	}
}

func TestPackages_Export_UsesCapabilityNameDirectory(t *testing.T) {
	agents := agentstore.New(t.TempDir())
	caps := capstore.New(t.TempDir())
	out := filepath.Join(t.TempDir(), "alpha.avm.zip")

	reviewID, err := caps.Add(model.CapabilityRecord{
		Kind:   model.CapabilityKindSkill,
		Name:   "review",
		Format: model.PayloadFormatSkillMD,
	}, bytes.NewReader([]byte("# review\n")))
	if err != nil {
		t.Fatalf("add review: %v", err)
	}
	shipID, err := caps.Add(model.CapabilityRecord{
		Kind:   model.CapabilityKindSkill,
		Name:   "ship",
		Format: model.PayloadFormatSkillMD,
	}, bytes.NewReader([]byte("# ship\n")))
	if err != nil {
		t.Fatalf("add ship: %v", err)
	}
	if err := agents.Save(&model.Agent{
		Identity: model.Identity{Name: "alpha"},
		Skills: []model.CapabilityRef{
			{ID: reviewID, Kind: model.CapabilityKindSkill},
			{ID: shipID, Kind: model.CapabilityKindSkill},
		},
	}); err != nil {
		t.Fatalf("save agent: %v", err)
	}

	pkgs := NewPackages(agents, caps, packageio.New())
	res, err := pkgs.Export(context.Background(), model.ExportRequest{
		Agent:         "alpha",
		IncludeSkills: true,
		OutputPath:    out,
	})
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if len(res.Manifest.Capabilities) != 2 {
		t.Fatalf("capabilities=%+v", res.Manifest.Capabilities)
	}

	want := map[string][]byte{
		"capabilities/skill/review/SKILL.md": []byte("# review\n"),
		"capabilities/skill/ship/SKILL.md":   []byte("# ship\n"),
	}
	seen := map[string]bool{}
	pio := packageio.New()
	mf, h, err := pio.Read(out)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	defer h.Close()
	for _, blob := range mf.Capabilities {
		body, ok := want[blob.Path]
		if !ok {
			t.Fatalf("unexpected capability path %q in manifest %+v", blob.Path, mf.Capabilities)
		}
		if seen[blob.Path] {
			t.Fatalf("duplicate capability path %q", blob.Path)
		}
		seen[blob.Path] = true
		rc, err := h.Open(blob.Path)
		if err != nil {
			t.Fatalf("open %s: %v", blob.Path, err)
		}
		got, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatalf("read %s: %v", blob.Path, err)
		}
		if !bytes.Equal(got, body) {
			t.Fatalf("payload %s = %q want %q", blob.Path, got, body)
		}
	}
	if len(seen) != len(want) {
		t.Fatalf("paths seen=%v want=%v", seen, want)
	}
}

func TestPackages_Install_RewritesExportedSourceIDs(t *testing.T) {
	srcAgents := agentstore.New(t.TempDir())
	srcCaps := capstore.New(t.TempDir())
	out := filepath.Join(t.TempDir(), "alpha.avm.zip")

	reviewID, err := srcCaps.Add(model.CapabilityRecord{
		Kind:   model.CapabilityKindSkill,
		Name:   "review",
		Format: model.PayloadFormatSkillMD,
	}, bytes.NewReader([]byte("# review\n")))
	if err != nil {
		t.Fatalf("add review: %v", err)
	}
	shipID, err := srcCaps.Add(model.CapabilityRecord{
		Kind:   model.CapabilityKindSkill,
		Name:   "ship",
		Format: model.PayloadFormatSkillMD,
	}, bytes.NewReader([]byte("# ship\n")))
	if err != nil {
		t.Fatalf("add ship: %v", err)
	}
	if err := srcAgents.Save(&model.Agent{
		Identity: model.Identity{Name: "alpha"},
		Skills: []model.CapabilityRef{
			{ID: reviewID, Kind: model.CapabilityKindSkill},
			{ID: shipID, Kind: model.CapabilityKindSkill},
		},
		Runtimes: []model.RuntimePref{{Runtime: "codex"}},
	}); err != nil {
		t.Fatalf("save source agent: %v", err)
	}

	srcPkgs := NewPackages(srcAgents, srcCaps, packageio.New())
	exported, err := srcPkgs.Export(context.Background(), model.ExportRequest{
		Agent:         "alpha",
		IncludeSkills: true,
		OutputPath:    out,
	})
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if len(exported.Manifest.Capabilities) != 2 {
		t.Fatalf("manifest capabilities=%+v", exported.Manifest.Capabilities)
	}
	for _, blob := range exported.Manifest.Capabilities {
		if blob.SourceID == "" {
			t.Fatalf("missing source_id in blob %+v", blob)
		}
		if blob.Format != model.PayloadFormatSkillMD {
			t.Fatalf("format=%q want %q in blob %+v", blob.Format, model.PayloadFormatSkillMD, blob)
		}
	}

	dstAgents := agentstore.New(t.TempDir())
	localReviewID := model.CapabilityID("cap_localreview000000000000000000")
	localShipID := model.CapabilityID("cap_localship00000000000000000000")
	dstCaps := newRemappingCapStore(capstore.New(t.TempDir()), localReviewID, localShipID)
	dstPkgs := NewPackages(dstAgents, dstCaps, packageio.New())
	res, err := dstPkgs.Install(context.Background(), model.InstallRequest{Source: out})
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if len(res.ImportedCaps) != 2 {
		t.Fatalf("ImportedCaps=%+v", res.ImportedCaps)
	}
	got, err := dstAgents.Get("alpha")
	if err != nil {
		t.Fatalf("Get installed agent: %v", err)
	}
	want := map[model.CapabilityID]bool{localReviewID: true, localShipID: true}
	for _, ref := range got.Skills {
		if !want[ref.ID] {
			t.Fatalf("installed ref %s was not rewritten to local IDs; skills=%+v", ref.ID, got.Skills)
		}
		delete(want, ref.ID)
	}
	if len(want) != 0 {
		t.Fatalf("missing local refs after install: %v", want)
	}
}

func TestPackages_ExportInstall_MultiSkillRoundTrip(t *testing.T) {
	srcAgents := agentstore.New(t.TempDir())
	srcCaps := capstore.New(t.TempDir())
	out := filepath.Join(t.TempDir(), "alpha.avm.zip")

	skills := map[string][]byte{
		"review":      []byte("# review\n"),
		"investigate": []byte("# investigate\n"),
		"ship":        []byte("# ship\n"),
	}
	var refs []model.CapabilityRef
	for name, body := range skills {
		id, err := srcCaps.Add(model.CapabilityRecord{
			Kind:   model.CapabilityKindSkill,
			Name:   name,
			Format: model.PayloadFormatSkillMD,
		}, bytes.NewReader(body))
		if err != nil {
			t.Fatalf("add %s: %v", name, err)
		}
		refs = append(refs, model.CapabilityRef{ID: id, Kind: model.CapabilityKindSkill})
	}
	if err := srcAgents.Save(&model.Agent{
		Identity: model.Identity{Name: "alpha"},
		Skills:   refs,
		Runtimes: []model.RuntimePref{{Runtime: "codex"}},
	}); err != nil {
		t.Fatalf("save source agent: %v", err)
	}

	srcPkgs := NewPackages(srcAgents, srcCaps, packageio.New())
	if _, err := srcPkgs.Export(context.Background(), model.ExportRequest{
		Agent:         "alpha",
		IncludeSkills: true,
		OutputPath:    out,
	}); err != nil {
		t.Fatalf("Export: %v", err)
	}

	pio := packageio.New()
	mf, h, err := pio.Read(out)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	paths := map[string]bool{}
	for _, blob := range mf.Capabilities {
		if paths[blob.Path] {
			t.Fatalf("duplicate capability path %q", blob.Path)
		}
		paths[blob.Path] = true
		rc, err := h.Open(blob.Path)
		if err != nil {
			t.Fatalf("open %s: %v", blob.Path, err)
		}
		body, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatalf("read %s: %v", blob.Path, err)
		}
		if !bytes.Equal(body, skills[blob.Name]) {
			t.Fatalf("payload %s=%q want %q", blob.Name, body, skills[blob.Name])
		}
	}
	if err := h.Close(); err != nil {
		t.Fatalf("close package: %v", err)
	}

	dstAgents := agentstore.New(t.TempDir())
	dstCaps := capstore.New(t.TempDir())
	dstPkgs := NewPackages(dstAgents, dstCaps, packageio.New())
	if _, err := dstPkgs.Install(context.Background(), model.InstallRequest{Source: out}); err != nil {
		t.Fatalf("Install: %v", err)
	}
	installed, err := dstAgents.Get("alpha")
	if err != nil {
		t.Fatalf("Get installed agent: %v", err)
	}
	if len(installed.Skills) != len(skills) {
		t.Fatalf("installed skills=%+v", installed.Skills)
	}
	for _, ref := range installed.Skills {
		rec, err := dstCaps.Get(ref.ID)
		if err != nil {
			t.Fatalf("installed ref %s not in capstore: %v", ref.ID, err)
		}
		body, err := dstCaps.ReadPayload(ref.ID)
		if err != nil {
			t.Fatalf("read installed payload %s: %v", ref.ID, err)
		}
		if !bytes.Equal(body, skills[rec.Name]) {
			t.Fatalf("installed payload %s=%q want %q", rec.Name, body, skills[rec.Name])
		}
	}
}

func TestPackageCapabilityPath_EncodesUnsafeNameSegment(t *testing.T) {
	got, err := packageCapabilityPath(model.CapabilityKindSkill, "../foo bar", "SKILL.md")
	if err != nil {
		t.Fatalf("packageCapabilityPath: %v", err)
	}
	want := "capabilities/skill/%2E%2E%2Ffoo%20bar/SKILL.md"
	if got != want {
		t.Fatalf("path=%q want %q", got, want)
	}
}

func TestPackages_Export_NoCapsWhenFlagsFalse(t *testing.T) {
	agents := agentstore.New(t.TempDir())
	caps := capstore.New(t.TempDir())
	id, err := caps.Add(model.CapabilityRecord{
		Kind: model.CapabilityKindSkill, Name: "demo",
	}, bytes.NewReader([]byte("x")))
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := agents.Save(&model.Agent{
		Identity: model.Identity{Name: "alpha"},
		Skills:   []model.CapabilityRef{{ID: id, Kind: model.CapabilityKindSkill}},
	}); err != nil {
		t.Fatalf("save: %v", err)
	}
	out := filepath.Join(t.TempDir(), "alpha.avm.zip")
	pkgs := NewPackages(agents, caps, packageio.New())
	res, err := pkgs.Export(context.Background(), model.ExportRequest{
		Agent:      "alpha",
		OutputPath: out,
		// IncludeSkills false
	})
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if len(res.Manifest.Capabilities) != 0 {
		t.Fatalf("expected no capabilities, got %+v", res.Manifest.Capabilities)
	}
}
