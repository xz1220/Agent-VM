package state

import (
	"encoding/json"
	"testing"

	"github.com/xz1220/agent-vm/internal/adapter"
	"github.com/xz1220/agent-vm/internal/config"
)

func TestSyncStateJSONContract(t *testing.T) {
	state := NewSyncState(config.ActiveRef{Kind: config.ActiveKindProfile, Name: "backend"})
	state.Runtimes["fake"] = RuntimeState{
		Runtime: "fake",
		Status:  RuntimeStatusSynced,
		Active:  state.LastActive,
		Mappings: MappingStates([]adapter.FieldMapping{
			{SourcePath: "agent.name", TargetPath: "/fake#name", Status: adapter.MappingNative},
		}),
	}

	raw, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded SyncState
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if decoded.Version != StateVersion || decoded.Runtimes["fake"].Mappings[0].Status != string(adapter.MappingNative) {
		t.Fatalf("unexpected decoded state: %#v", decoded)
	}
}
