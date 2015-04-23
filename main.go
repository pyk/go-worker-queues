package main

import (
	"flag"
	"fmt"
	"net/http"
	"time"
)

var (
	NWorkers = flag.Int("n", 4, "The number of workers to start")
	HTTPAddr = flag.String("http", "127.0.0.1:8000", "Address to listen for HTTP requests on")
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
	fmt.Printf("Work request: %s with delay %f queued.\n", work.Name, work.Delay.Seconds())

	w.WriteHeader(http.StatusCreated)
	return
}

// A Worker represents a worker. Worker have a channel of its own that the
// dispatcher can use to give worker a WorkRequest.
type Worker struct {
	ID        int
	Work      chan WorkRequest
	WorkQueue chan chan WorkRequest
	QuitChan  chan bool
}

// NewWorker returns a new Worker.
func NewWorker(id int, workerQueue chan chan WorkRequest) Worker {
	return Worker{id, make(chan WorkRequest), workerQueue, make(chan bool)}
}

// Start starts the worker
func (w Worker) Start() {
	go func() {
		for {
			w.WorkQueue <- w.Work

			select {
			case work := <-w.Work:
				fmt.Printf("worker%d: Received work request, delaying for %f seconds\n", w.ID, work.Delay.Seconds())
				time.Sleep(work.Delay)
				fmt.Printf("worker%d: Hello, %s!\n", w.ID, work.Name)
			case <-w.QuitChan:
				fmt.Printf("worker%d stopping\n", w.ID)
			}
		}
	}()
}

func (w Worker) Stop() {
	go func() {
		w.QuitChan <- true
	}()
}

var WorkerQueue chan chan WorkRequest

func StartDispatcher(nworkers int) {
	// First, initialize the channel we are going to but the workers' work channels into.
	WorkerQueue = make(chan chan WorkRequest, nworkers)

	// Now, create all of our workers.
	for i := 0; i < nworkers; i++ {
		fmt.Println("Starting worker", i+1)
		worker := NewWorker(i+1, WorkerQueue)
		worker.Start()
	}

	go func() {
		for {
			select {
			case work := <-WorkQueue:
				fmt.Println("Received work requeust")
				go func() {
					worker := <-WorkerQueue

					fmt.Println("Dispatching work request")
					worker <- work
				}()
			}
		}
	}()
}

func main() {
	// Parse the command-line flags.
	flag.Parse()

	// Start the dispatcher.
	fmt.Println("Starting the dispatcher")
	StartDispatcher(*NWorkers)

	// Register our collector as an HTTP handler function.
	fmt.Println("Registering the collector")
	http.HandleFunc("/work", Collector)

	// Start the HTTP server!
	fmt.Println("HTTP server listening on", *HTTPAddr)
	if err := http.ListenAndServe(*HTTPAddr, nil); err != nil {
		fmt.Println(err.Error())
	}
}
