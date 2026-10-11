package main

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

type Spinner struct {
	chars   []rune
	message string
	active  bool
	stop    chan bool
	done    chan struct{}
	output  io.Writer
	mu      sync.Mutex
}

func NewSpinner(message string) *Spinner {
	return &Spinner{
		chars:   []rune("⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏"),
		message: message,
		stop:    make(chan bool),
		done:    make(chan struct{}),
		output:  os.Stderr,
	}
}

func (s *Spinner) Start() {
	if s.active {
		return
	}
	s.active = true
	go func() {
		defer close(s.done)
		i := 0
		for {
			select {
			case <-s.stop:
				return
			default:
				s.mu.Lock()
				message := s.message
				s.mu.Unlock()
				fmt.Fprintf(s.output, "\r%c %s", s.chars[i%len(s.chars)], message)
				time.Sleep(100 * time.Millisecond)
				i++
			}
		}
	}()
}

func (s *Spinner) Stop() {
	if !s.active {
		return
	}
	s.active = false
	s.stop <- true
	<-s.done
	fmt.Fprint(s.output, "\r\033[K") // Clear the line
}

func (s *Spinner) UpdateMessage(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.message = message
}
