package cluster_test

import (
	"math"
	"testing"

	"gpuscheduler/cluster"
)

// The test fixture everywhere below: 2 racks x 2 nodes x 4 GPUs = 16 GPUs.
// Small enough that every expected value in this file was computed by hand.
const (
	tinyRacks       = 2
	tinyNodesPerRak = 2
	tinyGPUsPerNode = 4
	tinyNodes       = tinyRacks * tinyNodesPerRak // 4
	tinyGPUs        = tinyNodes * tinyGPUsPerNode // 16
)

func newTiny() *cluster.Cluster {
	return cluster.NewCluster(tinyRacks, tinyNodesPerRak, tinyGPUsPerNode)
}

// freeByNode snapshots per-node free counts so a test can assert the cluster was
// left untouched after a rejected Allocate.
func freeByNode(t *testing.T, c *cluster.Cluster) []int {
	t.Helper()
	free := make([]int, tinyNodes)
	for id := range free {
		free[id] = c.NodeFree(id)
	}
	return free
}

func assertFreeByNode(t *testing.T, c *cluster.Cluster, want []int) {
	t.Helper()
	got := freeByNode(t, c)
	for id, w := range want {
		if got[id] != w {
			t.Errorf("node %d free = %d, want %d (full state: got %v, want %v)", id, got[id], w, got, want)
		}
	}
}

func mustAllocate(t *testing.T, c *cluster.Cluster, job cluster.JobID, p cluster.Placement) {
	t.Helper()
	if err := c.Allocate(job, p); err != nil {
		t.Fatalf("Allocate(%v, %v) returned unexpected error: %v", job, p, err)
	}
}

func assertFragmentation(t *testing.T, c *cluster.Cluster, want float64) {
	t.Helper()
	got := c.Fragmentation()
	if math.IsNaN(got) {
		t.Fatalf("Fragmentation() = NaN, want %v (guard the zero-free-GPU case)", want)
	}
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("Fragmentation() = %v, want %v", got, want)
	}
}

// --- construction -----------------------------------------------------------

func TestNewClusterTopology(t *testing.T) {
	c := newTiny()

	if got := c.TotalGPUs(); got != tinyGPUs {
		t.Errorf("TotalGPUs() = %d, want %d", got, tinyGPUs)
	}
	if got := c.FreeGPUs(); got != tinyGPUs {
		t.Errorf("FreeGPUs() on a fresh cluster = %d, want %d", got, tinyGPUs)
	}
	if got := c.NumRacks(); got != tinyRacks {
		t.Errorf("NumRacks() = %d, want %d", got, tinyRacks)
	}
	if got := c.NumNodes(); got != tinyNodes {
		t.Errorf("NumNodes() = %d, want %d", got, tinyNodes)
	}

	// Nodes are laid out rack-major: nodes 0,1 in rack 0; nodes 2,3 in rack 1.
	wantRack := []int{0, 0, 1, 1}
	for nodeID, want := range wantRack {
		if got := c.RackOf(nodeID); got != want {
			t.Errorf("RackOf(%d) = %d, want %d", nodeID, got, want)
		}
	}

	for nodeID := 0; nodeID < tinyNodes; nodeID++ {
		if got := c.NodeFree(nodeID); got != tinyGPUsPerNode {
			t.Errorf("NodeFree(%d) on a fresh cluster = %d, want %d", nodeID, got, tinyGPUsPerNode)
		}
	}
}

// --- fragmentation ----------------------------------------------------------

func TestFragmentation(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, c *cluster.Cluster)
		want  float64
		why   string
	}{
		{
			name:  "empty cluster",
			setup: func(t *testing.T, c *cluster.Cluster) {},
			want:  0,
			why:   "all 16 free GPUs sit on fully-free nodes",
		},
		{
			name: "two whole nodes allocated",
			setup: func(t *testing.T, c *cluster.Cluster) {
				mustAllocate(t, c, 1, cluster.Placement{{NodeID: 0, Count: 4}})
				mustAllocate(t, c, 2, cluster.Placement{{NodeID: 1, Count: 4}})
			},
			want: 0,
			why:  "8 free GPUs remain, all of them on the two untouched nodes",
		},
		{
			name: "one GPU taken on every node",
			setup: func(t *testing.T, c *cluster.Cluster) {
				for nodeID := 0; nodeID < tinyNodes; nodeID++ {
					mustAllocate(t, c, cluster.JobID(10+nodeID), cluster.Placement{{NodeID: nodeID, Count: 1}})
				}
			},
			want: 1,
			why:  "12 free GPUs, zero of them on a fully-free node",
		},
		{
			name: "half of every node taken",
			setup: func(t *testing.T, c *cluster.Cluster) {
				for nodeID := 0; nodeID < tinyNodes; nodeID++ {
					mustAllocate(t, c, cluster.JobID(20+nodeID), cluster.Placement{{NodeID: nodeID, Count: 2}})
				}
			},
			want: 1,
			why:  "8 free GPUs spread across 4 dirty nodes; none on a whole node",
		},
		{
			name: "one dirty node, rest untouched",
			setup: func(t *testing.T, c *cluster.Cluster) {
				mustAllocate(t, c, 1, cluster.Placement{{NodeID: 0, Count: 1}})
			},
			// 15 free GPUs; 12 of them on the 3 fully-free nodes => 1 - 12/15 = 0.2
			want: 0.2,
			why:  "3 free GPUs stranded on node 0, 12 on whole nodes",
		},
		{
			name: "fully allocated",
			setup: func(t *testing.T, c *cluster.Cluster) {
				for nodeID := 0; nodeID < tinyNodes; nodeID++ {
					mustAllocate(t, c, cluster.JobID(30+nodeID), cluster.Placement{{NodeID: nodeID, Count: tinyGPUsPerNode}})
				}
			},
			want: 0,
			why:  "zero free GPUs must yield 0, not NaN",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := newTiny()
			tc.setup(t, c)
			t.Logf("expecting %v: %s", tc.want, tc.why)
			assertFragmentation(t, c, tc.want)
		})
	}
}

// --- allocate ---------------------------------------------------------------

func TestAllocateSpansMultipleNodes(t *testing.T) {
	c := newTiny()

	mustAllocate(t, c, 1, cluster.Placement{
		{NodeID: 0, Count: 3},
		{NodeID: 2, Count: 2},
	})

	assertFreeByNode(t, c, []int{1, 4, 2, 4})
	if got, want := c.FreeGPUs(), tinyGPUs-5; got != want {
		t.Errorf("FreeGPUs() = %d, want %d", got, want)
	}
}

// TestAllocateIsAtomic is the important one: a rejected placement must leave the
// cluster exactly as it was, not half applied.
func TestAllocateIsAtomic(t *testing.T) {
	tests := []struct {
		name string
		p    cluster.Placement
		why  string
	}{
		{
			name: "over node capacity",
			p:    cluster.Placement{{NodeID: 0, Count: 5}},
			why:  "5 GPUs requested on a 4-GPU node",
		},
		{
			name: "same node listed twice, together overflowing",
			p: cluster.Placement{
				{NodeID: 0, Count: 3},
				{NodeID: 0, Count: 3},
			},
			why: "each entry fits alone; summed they exceed the node",
		},
		{
			name: "valid entry followed by an invalid one",
			p: cluster.Placement{
				{NodeID: 1, Count: 2},
				{NodeID: 2, Count: 99},
			},
			why: "validation must happen before any mutation",
		},
		{
			name: "node id out of range",
			p:    cluster.Placement{{NodeID: tinyNodes, Count: 1}},
			why:  "there is no node with that id",
		},
		{
			name: "negative node id",
			p:    cluster.Placement{{NodeID: -1, Count: 1}},
			why:  "negative index must not panic",
		},
		{
			name: "zero count",
			p:    cluster.Placement{{NodeID: 0, Count: 0}},
			why:  "an empty entry is a caller bug, not a no-op",
		},
		{
			name: "negative count",
			p:    cluster.Placement{{NodeID: 0, Count: -1}},
			why:  "a negative count would manufacture free GPUs",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := newTiny()
			before := freeByNode(t, c)

			if err := c.Allocate(1, tc.p); err == nil {
				t.Fatalf("Allocate(%v) returned nil error, want an error (%s)", tc.p, tc.why)
			}

			assertFreeByNode(t, c, before)
			if got := c.FreeGPUs(); got != tinyGPUs {
				t.Errorf("FreeGPUs() after rejected Allocate = %d, want %d (cluster was mutated)", got, tinyGPUs)
			}
		})
	}
}

func TestAllocateExactlyFillsNode(t *testing.T) {
	c := newTiny()
	mustAllocate(t, c, 1, cluster.Placement{{NodeID: 0, Count: tinyGPUsPerNode}})

	if got := c.NodeFree(0); got != 0 {
		t.Errorf("NodeFree(0) = %d, want 0", got)
	}
	if err := c.Allocate(2, cluster.Placement{{NodeID: 0, Count: 1}}); err == nil {
		t.Error("Allocate on a full node returned nil error, want an error")
	}
}

// --- free -------------------------------------------------------------------

func TestFreeRestoresCapacity(t *testing.T) {
	c := newTiny()
	p := cluster.Placement{
		{NodeID: 0, Count: 3},
		{NodeID: 3, Count: 4},
	}
	mustAllocate(t, c, 7, p)

	if err := c.Free(7); err != nil {
		t.Fatalf("Free(7) returned unexpected error: %v", err)
	}

	assertFreeByNode(t, c, []int{4, 4, 4, 4})
	if got := c.FreeGPUs(); got != tinyGPUs {
		t.Errorf("FreeGPUs() after Free = %d, want %d", got, tinyGPUs)
	}
	assertFragmentation(t, c, 0)
}

func TestFreeLeavesOtherJobsAlone(t *testing.T) {
	c := newTiny()
	mustAllocate(t, c, 1, cluster.Placement{{NodeID: 0, Count: 2}})
	mustAllocate(t, c, 2, cluster.Placement{{NodeID: 0, Count: 2}})

	if err := c.Free(1); err != nil {
		t.Fatalf("Free(1) returned unexpected error: %v", err)
	}

	// Job 2 still holds 2 GPUs on node 0.
	assertFreeByNode(t, c, []int{2, 4, 4, 4})
}

func TestFreeUnknownJob(t *testing.T) {
	c := newTiny()
	if err := c.Free(404); err == nil {
		t.Error("Free of an unknown job returned nil error, want an error")
	}
}

func TestFreeTwiceIsAnError(t *testing.T) {
	c := newTiny()
	mustAllocate(t, c, 1, cluster.Placement{{NodeID: 0, Count: 1}})

	if err := c.Free(1); err != nil {
		t.Fatalf("first Free(1) returned unexpected error: %v", err)
	}
	if err := c.Free(1); err == nil {
		t.Error("second Free(1) returned nil error, want an error (a double free means the caller is confused)")
	}
}
