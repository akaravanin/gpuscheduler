package cluster

import "errors"

type Cluster struct {
	Nodes []Node
	Racks []Rack
}

type Rack struct {
	ID      int
	NodeIDs []int
}

type Node struct {
	ID        int
	RackID    int
	TotalGPUs int
	Alloc     map[JobID]int
}

type NodeAlloc struct {
	NodeID int
	Count  int
}

type Placement []NodeAlloc

var (
	ErrUnknownNode  = errors.New("cluster: node id out of range")
	ErrBadCount     = errors.New("cluster: gpu count must be positive")
	ErrInsufficient = errors.New("cluster: not enough free gpus on node")
	ErrUnknownJob   = errors.New("cluster: job holds no gpus")
	ErrInvalidJob   = errors.New("cluster: job id must not be NoJob")
)

type JobID int

func NewCluster(racks int, nodesPerRack int, gpusPerNode int) *Cluster {
	totalNodes := racks * nodesPerRack
	nodes := make([]Node, totalNodes)
	rs := make([]Rack, racks)
	for rack := 0; rack < racks; rack++ {
		rs[rack].ID = rack
		rs[rack].NodeIDs = make([]int, 0, nodesPerRack)
		for node := 0; node < nodesPerRack; node++ {
			id := rack*nodesPerRack + node
			nodes[id] = Node{
				ID:        id,
				RackID:    rack,
				TotalGPUs: gpusPerNode,
				Alloc:     make(map[JobID]int),
			}
			rs[rack].NodeIDs = append(rs[rack].NodeIDs, id)
		}
	}
	return &Cluster{Nodes: nodes, Racks: rs}
}

func (cl *Cluster) NodeFree(id int) int {
	used := 0
	for _, n := range cl.Nodes[id].Alloc {
		used += n
	}
	return cl.Nodes[id].TotalGPUs - used
}

func (c *Cluster) NumNodes() int {
	return len(c.Nodes)
}
func (c *Cluster) NumRacks() int {
	return len(c.Racks)
}
func (c *Cluster) TotalGPUs() int
func (c *Cluster) FreeGPUs() int
func (c *Cluster) RackOf(id int) int
func (p Placement) TotalGPUs() int

func (c *Cluster) Allocate(job JobID, p Placement) error {
	if job < 1 {
		return ErrInvalidJob
	}
	if len(p) <= 0 {
		return ErrBadCount
	}
	// 3. tally: build map[nodeID]int of total GPUs wanted per node,
	//    validating each entry as you go:
	//       0 <= NodeID < NumNodes   → ErrUnknownNode
	//       Count > 0                → ErrBadCount
	//
	// 4. for each nodeID in the tally: wanted <= c.NodeFree(nodeID),
	//    else ErrInsufficient
	//
	// 5. only now mutate: for each tally entry, alloc[job] += count
	return ErrUnknownNode
}
func (c *Cluster) Free(job JobID) error {
	// 1. job != NoJob
	// 2. scan nodes; where alloc has the job: delete(alloc, job), count a hit
	//    ── delete the key. Never set the count to 0.
	// 3. hits == 0 → ErrUnknownJob
	return ErrUnknownJob
}

// Fragmentation is the fraction of free GPUs that are not on a fully-free node.
// Returns 0 when there are no free GPUs.
func (c *Cluster) Fragmentation() float64 {
	// 1. freeTotal = sum of NodeFree over all nodes
	// 2. freeWhole = sum of TotalGPUs over nodes where len(alloc) == 0
	// 3. freeTotal == 0 → return 0
	// 4. return 1 - float64(freeWhole)/float64(freeTotal)
	return 1
}
