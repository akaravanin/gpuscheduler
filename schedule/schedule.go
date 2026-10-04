package schedule

import (
	"errors"
	"gpuscheduler/cluster"
	"gpuscheduler/job"
)

var (
	ErrNoFit   = errors.New("schedule: not enough free gpus for gang")
	ErrBadGang = errors.New("schedule: gang size must be positive")
)

type Policy interface {
	Place(cl *cluster.Cluster, j job.Job) (cluster.Placement, error)
}

type FirstFit struct{}

func (FirstFit) Place(cl *cluster.Cluster, j job.Job) (cluster.Placement, error) {
	if j.GPUs <= 0 {
		return nil, ErrBadGang
	}
	// fail before building anything: no partial placements
	if j.GPUs > cl.FreeGPUs() {
		return nil, ErrNoFit
	}
	var p cluster.Placement
	need := j.GPUs
	for id := 0; id < cl.NumNodes() && need > 0; id++ {
		take := min(cl.NodeFree(id), need)
		if take > 0 {
			p = append(p, cluster.NodeAlloc{NodeID: id, Count: take})
			need -= take
		}
	}
	return p, nil
}
