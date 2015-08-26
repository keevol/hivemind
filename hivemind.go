package main

import (
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Hivemind struct {
	procs       []*Process
	procWg      sync.WaitGroup
	done        chan bool
	interrupted chan os.Signal
}

func NewHivemind() (h *Hivemind) {
	h = &Hivemind{}
	h.createProcesses()
	return
}

func (h *Hivemind) createProcesses() {
	color := 32
	port := config.PortBase

	for _, entry := range parseProcfile("Procfile") {
		h.procs = append(
			h.procs,
			NewProcess(
				entry.Name,
				strings.Replace(entry.Command, "$PORT", strconv.Itoa(port), -1),
				color,
			),
		)

		color++
		port += config.PortStep
	}

	return
}

func (h *Hivemind) runProcess(proc *Process) {
	h.procWg.Add(1)

	go func() {
		defer h.procWg.Done()
		defer func() { h.done <- true }()

		proc.Run()
	}()
}

func (h *Hivemind) waitForDoneOrInterupt() {
	select {
	case <-h.done:
	case <-h.interrupted:
	}
}

func (h *Hivemind) waitForTimeoutOrInterrupt() {
	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(1 * time.Second)
		timeout <- true
	}()

	select {
	case <-timeout:
	case <-h.interrupted:
	}
}

func (h *Hivemind) waitForExit() {
	h.waitForDoneOrInterupt()

	for _, proc := range h.procs {
		go proc.Interrupt()
	}

	h.waitForTimeoutOrInterrupt()

	for _, proc := range h.procs {
		go proc.Kill()
	}
}

func (h *Hivemind) Run() {
	h.done = make(chan bool, len(h.procs))

	h.interrupted = make(chan os.Signal)
	signal.Notify(h.interrupted, os.Interrupt, os.Kill)

	for _, proc := range h.procs {
		h.runProcess(proc)
	}

	go h.waitForExit()

	h.procWg.Wait()
}
