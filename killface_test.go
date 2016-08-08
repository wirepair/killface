package killface

import (
	"flag"
	"fmt"
	"os/exec"
	"testing"
	"time"
)

var testProc string

func init() {
	flag.StringVar(&testProc, "proc", "memconsume", "a process you know is running")
	flag.Parse()
}

func TestSettingsMaxMemory(t *testing.T) {
	s := NewSettings()
	err := s.SetMaxMemory(29359203522345932)
	if err == nil {
		t.Fatalf("didn't get error setting ridic amount of memory: %s\n", err)
	}
}

func TestFindPidsByName(t *testing.T) {
	rdy := make(chan struct{})
	go testStart(t, 5, 1, rdy)
	<-rdy
	pids, err := findPidsByName(testProc)
	if err != nil {
		t.Fatalf("error finding process(es) %s: %s\n", testProc, err)
	}
	t.Logf("%#v\n", pids)
}

func TestKillMaxMemory(t *testing.T) {
	done := make(chan struct{})
	rdy := make(chan struct{})
	go testStart(t, 5, 10, rdy)
	<-rdy
	s := NewSettings()
	s.SetMaxMemory(100 * 1024 * 4)
	s.SetAllowedTime(time.Duration(5) * time.Second)
	s.ProcName(testProc)

	k := NewKiller(s)
	go testWaitKill(t, k.KilledCh, done)
	go func() {
		err := k.Monitor()
		if err != nil {
			t.Fatalf("error while monitoring: %s\n", err)
		}
	}()
	<-done
	k.Stop()
}

func TestKillMemPercent(t *testing.T) {
	done := make(chan struct{})
	rdy := make(chan struct{})
	go testStart(t, 5, 10, rdy)
	<-rdy
	total, err := totalMemory()
	if err != nil {
		t.Fatalf("error getting memory: %s\n", err)
	}
	percentage := 100 * float32(100000000*5) / float32(total)
	fmt.Printf("%f\n", percentage)
	s := NewSettings()
	s.SetPercentMemory(percentage)
	s.SetAllowedTime(time.Duration(5) * time.Second)
	s.ProcName(testProc)

	k := NewKiller(s)
	go testWaitKill(t, k.KilledCh, done)
	go func() {
		err := k.Monitor()
		if err != nil {
			t.Fatalf("error while monitoring: %s\n", err)
		}
	}()
	<-done
	k.Stop()
}

func TestKillStop(t *testing.T) {
	done := make(chan struct{})
	rdy := make(chan struct{})
	go testStart(t, 5, 15, rdy)
	<-rdy
	s := NewSettings()
	s.SetMaxMemory(100 * 1024 * 4)
	s.SetAllowedTime(time.Duration(5) * time.Second)
	s.ProcName(testProc)

	k := NewKiller(s)
	go testWaitKill(t, k.KilledCh, done)
	go k.Monitor()
	k.Stop()
	<-done
}

func TestKillReset(t *testing.T) {
	done := make(chan struct{})
	rdy := make(chan struct{})
	go testStart(t, 5, 15, rdy)
	<-rdy
	s := NewSettings()
	s.SetMaxMemory(100 * 1024 * 4)
	s.SetAllowedTime(time.Duration(5) * time.Second)
	s.ProcName(testProc)

	k := NewKiller(s)
	go k.Monitor()
	k.Stop()
	k.Reset()
	go testWaitKill(t, k.KilledCh, done)
	go func() {
		err := k.Monitor()
		if err != nil {
			t.Fatalf("error while monitoring: %s\n", err)
		}
	}()
	<-done
	k.Stop()
}

func testWaitKill(t *testing.T, killed <-chan *KillMsg, done chan<- struct{}) {
	fmt.Printf("killed; %#v\n", <-killed)
	done <- struct{}{}
}

func testStart(t *testing.T, count int, timeout int, rdy chan<- struct{}) {
	cmd := exec.Command("testdata/memconsume", "-mem", fmt.Sprintf("%d", count), "-timeout", fmt.Sprintf("%d", timeout))
	err := cmd.Start()
	if err != nil {
		t.Fatalf("error starting test process: %s\n", err)
	}
	rdy <- struct{}{}
	cmd.Wait()
}
