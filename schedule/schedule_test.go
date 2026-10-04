package schedule_test

import (
	"errors"
	"reflect"
	"testing"

	"gpuscheduler/cluster"
	"gpuscheduler/job"
	"gpuscheduler/schedule"
)

// API surface these tests assume (rename freely, then update this file):
//
//   schedule.Policy           interface with Place(*cluster.Cluster, job.Job) (cluster.Placement, error)
//   schedule.FirstFit         zero-value struct type implementing Policy
//   schedule.ErrNoFit         sentinel error: not enough free GPUs to place the gang
//   job.Job                   struct with a GPUs int field (gang size)
//
// First-fit as defined here: walk nodes in ID order, take min(free, still-needed) from
// each, stop when the gang is satisfied. All-or-nothing: on failure, no placement.

// Compile-time check that FirstFit satisfies Policy.
var _ schedule.Policy = schedule.FirstFit{}

// Same fixture as the cluster tests: 2 racks x 2 nodes x 4 GPUs = 16 GPUs.
// Nodes 0,1 are in rack 0; nodes 2,3 are in rack 1.
const (
	tinyRacks        = 2
	tinyNodesPerRack = 2
	tinyGPUsPerNode  = 4
	tinyNodes        = tinyRacks * tinyNodesPerRack // 4
	tinyGPUs         = tinyNodes * tinyGPUsPerNode  // 16
)

func newTiny() *cluster.Cluster {
	return cluster.NewCluster(tinyRacks, tinyNodesPerRack, tinyGPUsPerNode)
}

// occupy places a pre-existing job directly on the cluster, so a test can start from a
// partially-used state. Depends on cluster.Allocate working.
func occupy(t *testing.T, c *cluster.Cluster, id cluster.JobID, p cluster.Placement) {
	t.Helper()
	if err := c.Allocate(id, p); err != nil {
		t.Fatalf("setup: Allocate(%v, %v) failed: %v (finish cluster.Allocate first)", id, p, err)
	}
}

func freeByNode(c *cluster.Cluster) []int {
	free := make([]int, c.NumNodes())
	for id := range free {
		free[id] = c.NodeFree(id)
	}
	return free
}

// assertValid checks invariants every placement must satisfy, whatever the policy:
// it is exactly the gang size (never partial), each entry is positive and in range,
// no node is listed twice, and no node is asked for more than it has free.
func assertValid(t *testing.T, c *cluster.Cluster, p cluster.Placement, gang int) {
	t.Helper()
	if got := p.TotalGPUs(); got != gang {
		t.Fatalf("placement %v totals %d GPUs, want exactly %d (gang is all-or-nothing)", p, got, gang)
	}
	seen := map[int]bool{}
	for _, na := range p {
		if na.NodeID < 0 || na.NodeID >= c.NumNodes() {
			t.Fatalf("placement %v names node %d, out of range", p, na.NodeID)
		}
		if na.Count <= 0 {
			t.Fatalf("placement %v has non-positive count on node %d", p, na.NodeID)
		}
		if seen[na.NodeID] {
			t.Fatalf("placement %v lists node %d twice", p, na.NodeID)
		}
		seen[na.NodeID] = true
		if free := c.NodeFree(na.NodeID); na.Count > free {
			t.Fatalf("placement %v asks node %d for %d GPUs, only %d free", p, na.NodeID, na.Count, free)
		}
	}
}

// span reports the shape of a placement: distinct nodes and distinct racks touched.
// Tests assert on shape, not node IDs, so they survive iteration-order changes.
func span(c *cluster.Cluster, p cluster.Placement) (nodes, racks int) {
	ns, rs := map[int]bool{}, map[int]bool{}
	for _, na := range p {
		ns[na.NodeID] = true
		rs[c.RackOf(na.NodeID)] = true
	}
	return len(ns), len(rs)
}

// --- fits -------------------------------------------------------------------

func TestFirstFitPlacesOnEmptyCluster(t *testing.T) {
	tests := []struct {
		gang      int
		wantNodes int
		wantRacks int
		why       string
	}{
		{1, 1, 1, "one GPU fits on the first node"},
		{4, 1, 1, "exactly one node's worth"},
		{6, 2, 1, "fills node 0, spills to node 1, same rack"},
		{8, 2, 1, "both nodes of rack 0"},
		{10, 3, 2, "rack 0 is not enough; spills into rack 1"},
		{16, 4, 2, "the whole cluster"},
	}
	for _, tc := range tests {
		t.Run(tc.why, func(t *testing.T) {
			c := newTiny()
			p, err := schedule.FirstFit{}.Place(c, job.Job{GPUs: tc.gang})
			if err != nil {
				t.Fatalf("Place(gang=%d) returned error %v, want a placement", tc.gang, err)
			}
			assertValid(t, c, p, tc.gang)
			if n, r := span(c, p); n != tc.wantNodes || r != tc.wantRacks {
				t.Errorf("gang=%d spans %d nodes / %d racks, want %d / %d", tc.gang, n, r, tc.wantNodes, tc.wantRacks)
			}
		})
	}
}

func TestFirstFitSkipsFullNodes(t *testing.T) {
	c := newTiny()
	occupy(t, c, 100, cluster.Placement{{NodeID: 0, Count: 4}})

	p, err := schedule.FirstFit{}.Place(c, job.Job{GPUs: 2})
	if err != nil {
		t.Fatalf("Place returned error %v, want a placement", err)
	}
	assertValid(t, c, p, 2) // catches any GPUs taken from full node 0
	if n, _ := span(c, p); n != 1 {
		t.Errorf("spans %d nodes, want 1 (node 1 has room for all 2)", n)
	}
}

// TestFirstFitScatters pins the baseline weakness that best-fit and topology-aware
// exist to fix: first-fit grabs the leftover GPU on node 0 and splits a job that
// would have fit entirely on node 1.
func TestFirstFitScatters(t *testing.T) {
	c := newTiny()
	occupy(t, c, 100, cluster.Placement{{NodeID: 0, Count: 3}}) // node 0: 1 free

	p, err := schedule.FirstFit{}.Place(c, job.Job{GPUs: 4})
	if err != nil {
		t.Fatalf("Place returned error %v, want a placement", err)
	}
	assertValid(t, c, p, 4)
	if n, _ := span(c, p); n != 2 {
		t.Errorf("spans %d nodes, want 2 (1 GPU from node 0 + 3 from node 1)", n)
	}
}

func TestFirstFitUsesScatteredFreeGPUs(t *testing.T) {
	c := newTiny()
	for id := 0; id < tinyNodes; id++ {
		occupy(t, c, cluster.JobID(100+id), cluster.Placement{{NodeID: id, Count: 1}})
	}
	// 12 free GPUs, 3 on each node. No topology constraint yet, so it must fit.
	p, err := schedule.FirstFit{}.Place(c, job.Job{GPUs: 12})
	if err != nil {
		t.Fatalf("Place returned error %v, want a placement", err)
	}
	assertValid(t, c, p, 12)
	if n, r := span(c, p); n != 4 || r != 2 {
		t.Errorf("spans %d nodes / %d racks, want 4 / 2", n, r)
	}
}

// --- does not fit -----------------------------------------------------------

func TestFirstFitNoFitIsAllOrNothing(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, c *cluster.Cluster)
		gang  int
	}{
		{
			name:  "bigger than the cluster",
			setup: func(t *testing.T, c *cluster.Cluster) {},
			gang:  tinyGPUs + 1,
		},
		{
			name: "one short after other jobs",
			setup: func(t *testing.T, c *cluster.Cluster) {
				occupy(t, c, 100, cluster.Placement{{NodeID: 0, Count: 4}, {NodeID: 1, Count: 4}})
				occupy(t, c, 101, cluster.Placement{{NodeID: 2, Count: 1}})
			},
			gang: 8, // 7 free
		},
		{
			name: "cluster full",
			setup: func(t *testing.T, c *cluster.Cluster) {
				for id := 0; id < tinyNodes; id++ {
					occupy(t, c, cluster.JobID(100+id), cluster.Placement{{NodeID: id, Count: tinyGPUsPerNode}})
				}
			},
			gang: 1,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := newTiny()
			tc.setup(t, c)

			p, err := schedule.FirstFit{}.Place(c, job.Job{GPUs: tc.gang})
			if !errors.Is(err, schedule.ErrNoFit) {
				t.Errorf("Place(gang=%d) error = %v, want ErrNoFit", tc.gang, err)
			}
			if len(p) != 0 {
				t.Errorf("Place returned partial placement %v alongside failure, want none", p)
			}
		})
	}
}

func TestFirstFitRejectsNonPositiveGang(t *testing.T) {
	for _, gang := range []int{0, -1} {
		c := newTiny()
		p, err := schedule.FirstFit{}.Place(c, job.Job{GPUs: gang})
		if err == nil {
			t.Errorf("Place(gang=%d) returned nil error, want an error", gang)
		}
		if len(p) != 0 {
			t.Errorf("Place(gang=%d) returned placement %v, want none", gang, p)
		}
	}
}

// --- purity -----------------------------------------------------------------

// Place evaluates; it must not allocate. Only the caller (the sim) mutates the cluster.
func TestFirstFitDoesNotMutateCluster(t *testing.T) {
	c := newTiny()
	occupy(t, c, 100, cluster.Placement{{NodeID: 1, Count: 2}})
	before := freeByNode(c)

	for _, gang := range []int{3, 14, 99} { // fits, fits exactly, doesn't fit
		_, _ = schedule.FirstFit{}.Place(c, job.Job{GPUs: gang})
		if got := freeByNode(c); !reflect.DeepEqual(got, before) {
			t.Fatalf("after Place(gang=%d), free by node = %v, want unchanged %v", gang, got, before)
		}
	}
}

func TestFirstFitIsDeterministic(t *testing.T) {
	c := newTiny()
	occupy(t, c, 100, cluster.Placement{{NodeID: 0, Count: 1}, {NodeID: 2, Count: 3}})

	first, err := schedule.FirstFit{}.Place(c, job.Job{GPUs: 9})
	if err != nil {
		t.Fatalf("Place returned error %v", err)
	}
	for i := 0; i < 20; i++ {
		again, _ := schedule.FirstFit{}.Place(c, job.Job{GPUs: 9})
		if !reflect.DeepEqual(again, first) {
			t.Fatalf("run %d gave %v, first run gave %v (seeded sims need deterministic policies)", i, again, first)
		}
	}
}
