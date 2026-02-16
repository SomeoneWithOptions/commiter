package main

import (
	"fmt"
	"time"
)

type Spinner struct {
	chars   []rune
	message string
	active  bool
	stop    chan bool
}

func NewSpinner(message string) *Spinner {
	return &Spinner{
		chars:   []rune("⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏"),
		message: message,
		stop:    make(chan bool),
	}
}

func (s *Spinner) Start() {
	if s.active {
		return
	}
	s.active = true
	go func() {
		i := 0
		for {
			select {
			case <-s.stop:
				return
			default:
				fmt.Printf("\r%c %s", s.chars[i%len(s.chars)], s.message)
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
	fmt.Print("\r\033[K") // Clear the line
}

func (s *Spinner) UpdateMessage(message string) {
	s.message = message
}
