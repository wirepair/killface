/*
The MIT License (MIT)
Copyright (c) 2016 isaac dawson
Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:
The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.
THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package killface

import (
	"fmt"
	gops "github.com/mitchellh/go-ps"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/process"
	"log"
	"time"
)

// an error occurred while monitoring processes to kill.
type MonitorErr struct {
	Message string
}

// the error that occurred while monitoring processes to kill
func (e *MonitorErr) Error() string {
	return e.Message
}

// Our process killer object. Notifies implementers of killed processes over the KilledCh.
//
type Killer struct {
	KilledCh     chan []int            // a channel that notifies caller of the processes that were killed
	settings     *Settings             // how this killer object is configured
	systemMemory uint64                // how much system memory exists
	pids         map[int]time.Duration // our current pid list
	ticker       *time.Ticker          // how often to look for processes over the threshold
	done         chan struct{}         // signals completion of Monitor
}

// Creates a new Killer object with the given settings
func NewKiller(settings *Settings) *Killer {
	k := &Killer{settings: settings}
	k.Reset()
	return k
}

// Resets the Killer properties, assumes Stop has already been called
func (k *Killer) Reset() {
	k.pids = make(map[int]time.Duration)
	k.KilledCh = make(chan []int)
	k.ticker = time.NewTicker(k.settings.interval)
	k.done = make(chan struct{})
}

// Monitors processes to see if they go over a certain memory threshold
// greater than allowed time. If they do, we kill them and send the pid
// values over the KilledCh.
func (k *Killer) Monitor() error {
	var err error

	k.systemMemory, err = totalMemory()
	if err != nil {
		return err
	}

	if k.settings.procName == "" {
		return &MonitorErr{Message: "procname not properly supplied"}
	}

	k.debugf("monitoring: %s\n", k.settings.procName)
	for {
		select {
		case <-k.ticker.C:
			if err := k.refreshPids(); err != nil {
				return err
			}
			k.debugf("refreshed pids %v\n", k.pids)
			pids := k.killCheck(k.pids)
			if len(pids) > 0 {
				k.debugf("killing pids: %v\n", pids)
				k.KilledCh <- pids
			}
		case <-k.done:
			return nil
		}
	}
	return nil
}

// Stops monitoring by stopping our ticker, closing the killed channel and signaling
// the done channel for Monitor to return.
// Call Reset() to prior to re-running Monitor if you wish to re-use this object
func (k *Killer) Stop() {
	k.ticker.Stop()
	close(k.KilledCh)
	close(k.done)
}

// Refreshes our pid list, updates and checks if any have exceeded threshold for > allowedTime
// then kills either all the pids found (if settings.killAll) or just kills the exceeded ones.
func (k *Killer) killCheck(pids map[int]time.Duration) []int {
	exceeded := k.checkExceeded(pids)
	if len(exceeded) == 0 {
		return exceeded // return an empty array
	}

	pidsToKill := make([]int, len(pids))
	// kill all
	if k.settings.killAll {
		i := 0
		for pid := range pids {
			pidsToKill[i] = pid
		}
	} else {
		// just kill exceeded
		pidsToKill = exceeded
	}

	for _, pid := range pidsToKill {
		err := k.kill(pid)
		if err != nil {
			k.debugf("error killing process: %d, %s", pid, err)
		}
	}

	return pidsToKill
}

// Kills the pid provided it still exists
func (k *Killer) kill(pid int) error {
	proc, err := process.NewProcess(int32(pid))
	if err != nil {
		return err
	}
	return proc.Kill()
}

// Finds the pids by name and updates our k.pid list.
func (k *Killer) refreshPids() error {
	foundPids, err := findPidsByName(k.settings.procName)
	if err != nil {
		return &MonitorErr{Message: fmt.Sprintf("error trying to find pids: %s\n", err)}
	}

	// update our k.pids list with the new procs
	for pid, exceededTime := range foundPids {
		if _, ok := k.pids[pid]; !ok {
			k.pids[pid] = exceededTime
		}
	}
	return nil
}

// Checks that our process still exists. If not, remove it from the map. Then check
// if it is over the allowed threshold, add its time + the interval we check at.
// If that time is greater than allowedTime we should kill it. Otherwise, reset it to 0.
func (k *Killer) checkExceeded(pids map[int]time.Duration) []int {
	pidsToKill := make([]int, 0)
	for pid, exceededTime := range k.pids {
		proc, err := process.NewProcess(int32(pid))
		// process no longer exists
		if err != nil {
			k.debugf("pid %d no longer exists", pid)
			delete(pids, pid)
			continue
		}
		// it's over our threshold, add the interval duration to the current overThreshold time
		if k.isOverThreshold(proc) {
			k.pids[pid] += k.settings.interval
			k.debugf("pid %d is over the limit for %v\n", pid, k.pids[pid])
			// it's been overthreshold for the max allowed time, add it to our kill list
			if exceededTime >= k.settings.allowedTime {
				pidsToKill = append(pidsToKill, pid)
			}
		} else {
			// reset
			exceededTime = 0
		}
	}
	return pidsToKill
}

// If we are killing by max memory, check the processes RSS value is >= maxMemory allowed.
// If we are using percentages, check the percentage this process is at.
func (k *Killer) isOverThreshold(proc *process.Process) bool {
	info, err := proc.MemoryInfo()
	if err != nil {
		return false
	}
	if k.settings.maxMemory != 0 && (info.RSS >= k.settings.maxMemory) {
		return true
	} else if k.settings.percentMemory != 0 {
		percent, err := proc.MemoryPercent()
		if err != nil {
			return false
		}

		if percent >= k.settings.percentMemory {
			return true
		}
	}
	return false
}

// Show debug messages if settings.Debug() was called
func (k *Killer) debugf(format string, args ...interface{}) {
	if k.settings.debug {
		log.Printf(format, args...)
	}
}

// Returns the systems total virtual memory
func totalMemory() (uint64, error) {
	v, err := mem.VirtualMemory()
	return v.Total, err
}

// Finds pids by name by iterating over all processes and extracting those that match
func findPidsByName(name string) (map[int]time.Duration, error) {
	mpids := make(map[int]time.Duration)
	procs, err := gops.Processes()
	if err != nil {
		return nil, err
	}

	for _, proc := range procs {
		if proc.Executable() == name {
			mpids[proc.Pid()] = 0
		}
	}
	return mpids, nil
}
