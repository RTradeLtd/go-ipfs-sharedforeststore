// Copyright 2020 RTrade Technologies Ltd
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
		{initialized: true},
		{initialized: true, HaveBytes: 116, KnownBytes: 116},
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
