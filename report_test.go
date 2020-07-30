package sharedforeststore

import (
	"context"
	"testing"

	leveldb "github.com/ipfs/go-ds-leveldb"
)

func TestReport(t *testing.T) {
	cids, getter := setup(t)
	db, err := leveldb.NewDatastore("", nil)
	fatalIfErr(t, err)
	defer db.Close()
	store := NewProgressiveTagCountedStore(db, nil)
	ctx := context.Background()

	expectedReports := []ProgressReport{
		{initalized: true},
		{initalized: true, HaveBytes: 116, KnownBytes: 116},
	}

	r := &ProgressReport{}
	fatalIfErr(t, store.GetProgressReport(ctx, cids[0], r))
	if *r != expectedReports[0] {
		t.Errorf("expected report %v, but got %v", expectedReports[0], *r)
	}
	_, err = store.Increment(ctx, cids[0], getter)
	fatalIfErr(t, err)
	fatalIfErr(t, store.GetProgressReport(ctx, cids[0], r))
	if *r != expectedReports[1] {
		t.Errorf("expected report %v, but got %v", expectedReports[1], *r)
	}
}
