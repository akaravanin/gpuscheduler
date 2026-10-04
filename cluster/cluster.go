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

func (cl *Cluster) NumNodes() int {
	return len(cl.Nodes)
}
func (cl *Cluster) NumRacks() int {
	return len(cl.Racks)
}
func (cl *Cluster) TotalGPUs() int {
	total := 0
	for _, n := range cl.Nodes {
		total += n.TotalGPUs
	}
	return total
}
func (cl *Cluster) FreeGPUs() int {
	freeGPUs := 0
	for _, n := range cl.Nodes {
		freeGPUs += cl.NodeFree(n.ID)
	}
	return freeGPUs
}
func (cl *Cluster) RackOf(id int) int {
	return cl.Nodes[id].RackID
}
func (p Placement) TotalGPUs() int {
	total := 0
	for _, n := range p {
		total += n.Count
	}
	return total
}

func (cl *Cluster) Allocate(job JobID, p Placement) error {
	if job < 1 {
		return ErrInvalidJob
	}
	if len(p) <= 0 {
		return ErrBadCount
	}
	tally := make(map[int]int, len(p))

	for _, n := range p {
		if n.NodeID < 0 || n.NodeID >= cl.NumNodes() {
			return ErrUnknownNode
		}
		if n.Count <= 0 {
			return ErrBadCount
		}
		tally[n.NodeID] += n.Count
	}
	for id, n := range tally {
		if n > cl.NodeFree(id) {
			return ErrInsufficient
		}
	}
	for id, n := range tally {
		cl.Nodes[id].Alloc[job] += n
	}
	return nil
}
func (cl *Cluster) Free(job JobID) error {
	if job < 1 {
		return ErrInvalidJob
	}
	hits := 0
	for i := range cl.Nodes {
		if _, ok := cl.Nodes[i].Alloc[job]; ok {
			delete(cl.Nodes[i].Alloc, job)
			hits++
		}
	}
	if hits == 0 {
		return ErrUnknownJob
	}
	return nil
}

// Fragmentation is the fraction of free GPUs that are not on a fully-free node.
// Returns 0 when there are no free GPUs.
func (cl *Cluster) Fragmentation() float64 {
	freeTotal, freeWhole := 0, 0
	for i := range cl.Nodes {
		free := cl.NodeFree(i)
		freeTotal += free
		if len(cl.Nodes[i].Alloc) == 0 {
			freeWhole += free
		}
	}
	if freeTotal == 0 {
		return 0
	}
	return 1 - float64(freeWhole)/float64(freeTotal)
}
