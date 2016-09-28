package jobadmin

import (
	"math/rand"
	"sync"
)

type schedJob struct {
	Job    *Job
	Master *LiveMaster
	Res    chan<- *LiveJob
	Err    chan<- error
}

type schedMasterReq struct {
	Res  chan<- []*LiveMaster
	Auto chan<- []bool
}

type schedAutoReq struct {
	Master *LiveMaster
	Auto   bool
}

// A Scheduler manages a pool of masters and automatically
// schedules jobs on them.
//
// All jobs should be launched through the scheduler, even
// jobs that are launched manually by the admin.
// This allows the scheduler to keep tabs on the current
// jobs and receive notifications when slaves or jobs are
// available.
type Scheduler struct {
	shutdownLock sync.Mutex
	shutdown     chan struct{}

	masterNote nextNotifier

	newJobs    chan []*Job
	newMaster  chan *LiveMaster
	runJob     chan *schedJob
	getJobs    chan chan<- []*Job
	getMasters chan *schedMasterReq
	setAuto    chan *schedAutoReq
}

// NewScheduler creates an active scheduler.
// When you are done with the scheduler, you should call
// Terminate on it.
func NewScheduler() *Scheduler {
	s := &Scheduler{
		shutdown:   make(chan struct{}),
		newJobs:    make(chan []*Job),
		newMaster:  make(chan *LiveMaster),
		runJob:     make(chan *schedJob),
		getJobs:    make(chan chan<- []*Job),
		getMasters: make(chan *schedMasterReq),
		setAuto:    make(chan *schedAutoReq),
	}
	go s.runLoop()
	return s
}

// Terminate triggers a shutdown process that will stop
// all the masters on the scheduler and prevent any new
// jobs/masters from being added.
func (s *Scheduler) Terminate() {
	s.shutdownLock.Lock()
	defer s.shutdownLock.Unlock()
	select {
	case <-s.shutdown:
	default:
		close(s.shutdown)
	}
}

func (s *Scheduler) runLoop() {
	var jobs []*Job
	var masters []*LiveMaster
	var auto []bool

	defer func() {
		s.masterNote.Close()
		for _, m := range masters {
			m.Cancel()
		}
	}()

	doneChan := make(chan struct{}, 1)

	for {
		select {
		case <-s.shutdown:
			return
		default:
		}

		select {
		case <-s.shutdown:
			return
		case j := <-s.newJobs:
			jobs = j
			s.reschedule(jobs, s.availableMasters(masters, auto), doneChan)
		case <-doneChan:
			s.reschedule(jobs, s.availableMasters(masters, auto), doneChan)
		case m := <-s.newMaster:
			masters = append(masters, m)
			auto = append(auto, false)
			s.masterNote.Notify()
		case j := <-s.runJob:
			s.startJob(j.Job, j.Master, doneChan)
		case r := <-s.getJobs:
			r <- jobs
		case r := <-s.getMasters:
			r.Res <- masters
			a := make([]bool, len(auto))
			copy(a, auto)
			r.Auto <- a
		case r := <-s.setAuto:
			for i, m := range masters {
				if m == r.Master {
					auto[i] = r.Auto
					if r.Auto {
						s.reschedule(jobs, s.availableMasters(masters, auto),
							doneChan)
					}
					break
				}
			}
		}
	}
}

func (s *Scheduler) reschedule(jobs []*Job, masters []*LiveMaster, doneChan chan<- struct{}) {
	jobCounts := map[string]int{}
	cpuCounts := make([]int, len(masters))
	for masterIdx, m := range masters {
		for _, j := range m.Jobs(0, m.JobCount()) {
			if j.Running() {
				job := j.Job()
				jobCounts[job.ID]++
				cpuCounts[masterIdx] += job.NumCPU
			}
		}
	}

	pl := newPriorityList(jobs, jobCounts)
	for {
		jobIdx := pl.Random()
		job := pl.Jobs[jobIdx]
		var master *LiveMaster
		for _, i := range rand.Perm(len(masters)) {
			if cpuCounts[i] < masters[i].SlaveInfo().MaxProcs {
				master = masters[i]
				cpuCounts[i] += job.NumCPU
			}
		}
		if master == nil {
			break
		}
		s.startJob(job, master, doneChan)
	}
}

func (s *Scheduler) availableMasters(m []*LiveMaster, auto []bool) []*LiveMaster {
	res := make([]*LiveMaster, 0, len(m))
	for i, x := range m {
		if auto[i] && x.Accepting() {
			res = append(res, x)
		}
	}
	return res
}

func (s *Scheduler) startJob(j *Job, m *LiveMaster, doneChan chan<- struct{}) {
	go func() {
		defer func() {
			doneChan <- struct{}{}
		}()
		lj, err := m.RunJob(j)
		if err != nil {
			return
		}
		lj.Wait(nil)
	}()
}

type priorityList struct {
	Jobs          []*Job
	TotalPriority int
}

func newPriorityList(j []*Job, counts map[string]int) *priorityList {
	var p priorityList
	for _, job := range j {
		if job.Priority > 0 && counts[job.ID] < job.MaxInstances {
			p.Jobs = append(p.Jobs, job)
			p.TotalPriority += job.Priority
		}
	}
	return &p
}

func (p *priorityList) Random() int {
	num := rand.Intn(p.TotalPriority)
	for i, j := range p.Jobs {
		if num == 0 {
			return i
		}
		num -= j.Priority
		if num < 0 {
			return i
		}
	}
	panic("unreachable code")
}

func (p *priorityList) Remove(idx int) {
	j := p.Jobs[idx]
	p.Jobs[idx] = p.Jobs[len(p.Jobs)-1]
	p.Jobs = p.Jobs[:len(p.Jobs)-1]
	p.TotalPriority -= j.Priority
}
