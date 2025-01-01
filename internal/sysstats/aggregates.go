package sysstats

import (
	"fmt"
	"time"

	"github.com/fudanchii/szb/internal/humanreadable"
	"github.com/mackerelio/go-osstat/cpu"
	"github.com/mackerelio/go-osstat/memory"
	"github.com/mackerelio/go-osstat/uptime"
)

type Aggregates struct {
	prevCPUStats    *cpu.Stats
	currentCPUStats *cpu.Stats
	memStats        *memory.Stats
	uptime          time.Duration
}

func NewAggregates() (*Aggregates, error) {
	cpuStats, err := cpu.Get()
	if err != nil {
		return nil, err
	}

	memStats, err := memory.Get()
	if err != nil {
		return nil, err
	}

	uptime, err := uptime.Get()
	if err != nil {
		return nil, err
	}

	aggr := &Aggregates{
		currentCPUStats: cpuStats,
		prevCPUStats:    cpuStats,
		memStats:        memStats,
		uptime:          uptime,
	}

	go aggr.populateStatsInfo()

	return aggr, nil
}

func (aggr *Aggregates) populateStatsInfo() {
	for {
		aggr.prevCPUStats = aggr.currentCPUStats
		aggr.currentCPUStats, _ = cpu.Get()
		aggr.uptime, _ = uptime.Get()

		time.Sleep(1 * time.Second)
	}
}

func (aggr *Aggregates) String() string {
	cpuTotal := float64(aggr.currentCPUStats.Total - aggr.prevCPUStats.Total)

	usrCpu := float64(0)
	sysCpu := float64(0)
	idlCpu := float64(0)

	if cpuTotal != 0 {
		usrCpu = float64(aggr.currentCPUStats.User-aggr.prevCPUStats.User) / cpuTotal * 100
		sysCpu = float64(aggr.currentCPUStats.System-aggr.prevCPUStats.System) / cpuTotal * 100
		idlCpu = float64(aggr.currentCPUStats.Idle-aggr.prevCPUStats.Idle) / cpuTotal * 100
	}

	return fmt.Sprintf("mem.total:%s, mem.avail:%s, mem.cached:%s, mem.act:%s, mem.inact:%s, mem.free:%s, cpu.usr:%.1f%%, cpu.sys:%.1f%%, cpu.idle:%.1f%%, up:%v",
		humanreadable.BiBytes(aggr.memStats.Total),
		humanreadable.BiBytes(aggr.memStats.Available),
		humanreadable.BiBytes(aggr.memStats.Cached),
		humanreadable.BiBytes(aggr.memStats.Active),
		humanreadable.BiBytes(aggr.memStats.Inactive),
		humanreadable.BiBytes(aggr.memStats.Free),
		usrCpu, sysCpu, idlCpu,
		humanreadable.Second(aggr.uptime))
}
