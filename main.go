package main

import (
	"fmt"
	"net/http"
	"time"
)

// A buffered channel that we can send work request on
var WorkQueue = make(chan WorkRequest, 100)

// A WorkRequest represent work request that sends to the
// worker, via the dispatcher.
type WorkRequest struct {
	Name  string
	Delay time.Duration
}

// Collector receives client requests for work. Create a work request that
// worker can understand, and pushes the work onto the end of the work queue.
func Collector(w http.ResponseWriter, r *http.Request) {
	// Only accept POST request
	if r.Method != "POST" {
		w.Header().Set("Allow", "POST")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	delay, err := time.ParseDuration(r.FormValue("delay"))
	if err != nil {
		http.Error(w, "Bad delay value: "+err.Error(), http.StatusBadRequest)
	}

	if delay.Seconds() < 1 || delay.Seconds() > 10 {
		http.Error(w, "The delay must be between 1 and 10 seconds, inclusively.", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	if name == "" {
		http.Error(w, "You must specify a name.", http.StatusBadRequest)
		return
	}

	work := WorkRequest{Name: name, Delay: delay}

	WorkQueue <- work
	fmt.Printf("Work request: %s with delay %v queued.\n", work.Name, work.Delay)

	w.WriteHeader(http.StatusCreated)
	return
}
