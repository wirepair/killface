![alt text](http://wirepair.github.io/images/frisky-dingo-killface-postcard1.jpg "Vote Killface")
# Killface
This library is for monitoring processes by name and kills them if they consume more memory than allowed. You can configure it to kill all processes that match the name or only pids that are over the allowed amount.  

## Usage
```Go
package main

import (
	"flag"
	"github.com/wirepair/killface"
	"log"
	"time"
)

var (
	procName    string  // process(es) to monitor by name
	useBytes    bool    // should we use bytes instead of percentage?
	debug       bool    // show debug messages
	killAll     bool    // killall processes
	percentage  float64 // percentage value
	maxMemory   uint64  // max memory in bytes to allow
	interval    int     // how often to check
	allowedTime int     // how long we allow process(es) to be over the threshold
)

func init() {
	flag.StringVar(&procName, "proc", "", "the process to monitor")
	flag.BoolVar(&useBytes, "usebytes", false, "use bytes instead of percentages")
	flag.BoolVar(&debug, "debug", false, "print out debug statements")
	flag.BoolVar(&killAll, "all", false, "kill all processes over the limits")
	flag.Float64Var(&percentage, "percent", 5, "the percentage of memory the process(es) are allowed to consume")
	flag.Uint64Var(&maxMemory, "maxmem", 100000000, "how many bytes these processes are allowed to consume")
	flag.IntVar(&interval, "interval", 1, "time in seconds to check on processes")
	flag.IntVar(&allowedTime, "allowed", 10, "time in seconds processes are allowed to be over threshold")
}

// Easy test:
// $ ./example -proc chromium-browser -debug -allowed 5 &
// $ chromium-browser http://crashsafari.com
func main() {
	flag.Parse()
	if procName == "" {
		log.Fatal("process name is required (use -proc)")
	}

	settings := killface.NewSettings()
	settings.ProcName(procName)
	settings.SetInterval(time.Duration(interval) * time.Second)
	settings.SetAllowedTime(time.Duration(allowedTime) * time.Second)

	if killAll {
		settings.KillAll() // kill all processes if even 1 is over the threshold
	}

	if debug {
		settings.Debug() // prints debug messages
	}

	if useBytes {
		settings.SetMaxMemory(maxMemory) // uses a byte value
	} else {
		settings.SetPercentMemory(float32(percentage)) // use a percentage
	}

	killer := killface.NewKiller(settings) // create new killer using our settings object
	go killer.Monitor()                    // start monitoring

	// wait for pids to be killed
	for {
		select {
		case pids := <-killer.KilledCh:
			log.Printf("killed the following pids: %v\n", pids)
			killer.Stop()
			return
		}
	}
}
```

##License
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