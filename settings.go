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
	"runtime"
	"time"
)

// error for invalid settings
type InvalidSettingsErr struct {
	Message string
}

// returned when the supplied settings are incorrect
func (e *InvalidSettingsErr) Error() string {
	return e.Message
}

type Settings struct {
	maxMemory     uint64        // maximum amount of memory allowed before killing in bytes
	percentMemory float32       // maximum percentage of memory allowed before killing
	procName      string        // the process name to monitor
	killAll       bool          // kill all processes that match procName in the event threshold is exceeded
	debug         bool          // should we print debug statements
	interval      time.Duration // the interval to monitor the processes
	allowedTime   time.Duration // how long the process is allowed to go over the threshold
}

func NewSettings() *Settings {
	return &Settings{
		percentMemory: 70.0,             // allow 70%
		interval:      1 * time.Second,  // check every 1 second
		allowedTime:   10 * time.Second, // allow to be over threshold for 10 seconds
	}
}

// Sets the maximum amount of memory allowed in bytes.
func (s *Settings) SetMaxMemory(maxMemory uint64) error {
	total, err := totalMemory()
	if err != nil {
		return err
	}
	if maxMemory > total {
		return &InvalidSettingsErr{Message: fmt.Sprintf("maxMemory greater than system total of %d", total)}
	}
	s.maxMemory = maxMemory
	s.percentMemory = 0
	return nil
}

// Sets the percentage of memory allowed for a process for this particular system.
func (s *Settings) SetPercentMemory(percentMemory float32) error {
	if percentMemory == 0 || percentMemory > 100.0 {
		return &InvalidSettingsErr{Message: "percent of memory can not be 0 or > 100"}
	}
	s.percentMemory = percentMemory
	s.maxMemory = 0
	return nil
}

// The process name to monitor, note this will monitor *all* processes given this name
func (s *Settings) ProcName(procName string) error {
	if procName == "" {
		return &InvalidSettingsErr{Message: "process name must be set"}
	}
	// linux truncates to 15 characters
	if len(procName) > 15 && runtime.GOOS == "linux" {
		procName = procName[0:15]
	}
	s.procName = procName
	return nil
}

// if multiple processes are found, kill them all even if only 1 goes over the threshold
// for longer than allowedTime
func (s *Settings) KillAll() {
	s.killAll = true
}

// the duration to check our processes
func (s *Settings) SetInterval(interval time.Duration) error {
	if interval == 0 {
		return &InvalidSettingsErr{Message: "interval must be greater than 0"}
	}
	s.interval = interval
	return nil
}

// how long the process is allowed to exceed the threshold
func (s *Settings) SetAllowedTime(allowedTime time.Duration) error {
	if allowedTime <= s.interval {
		return &InvalidSettingsErr{Message: "allowedTime must be greater than interval"}
	}
	s.allowedTime = allowedTime
	return nil
}

func (s *Settings) Debug() {
	s.debug = true
}
