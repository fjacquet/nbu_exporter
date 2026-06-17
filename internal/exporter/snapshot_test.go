package exporter

import "testing"

func TestSnapshotStoreLoadStore(t *testing.T) {
	var s SnapshotStore
	if s.Load() != nil {
		t.Fatal("zero-value store should Load() nil")
	}
	snap := &Snapshot{Sites: map[string]*SiteSnapshot{"paris": {Site: "paris", Up: true}}}
	s.Store(snap)
	got := s.Load()
	if got == nil || got.Sites["paris"] == nil || !got.Sites["paris"].Up {
		t.Fatal("Load did not return the stored snapshot")
	}
}
