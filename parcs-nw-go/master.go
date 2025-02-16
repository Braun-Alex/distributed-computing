package main

import (
	"fmt"
	"github.com/lionell/parcs/go/parcs"
	"log"
	"os"
	"strconv"
	"sync"
	"time"
)

type Program struct {
	*parcs.Runner
}

func (h *Program) Run() {
	a, err := strconv.Atoi(os.Getenv("A"))
	if err != nil {
		log.Fatal(err)
	}

	iterations, err := strconv.Atoi(os.Getenv("ITERATIONS"))
	if err != nil {
		iterations = 50
	}

	numWorkers, err := strconv.Atoi(os.Getenv("WORKERS"))
	if err != nil {
		numWorkers = 3
	}

	startTime := time.Now()

	r, s := 0, a-1

	for s%2 == 0 {
		r++
		s /= 2
	}

	chunkSize := iterations / numWorkers

	var wg sync.WaitGroup

	var (
		isComposite bool
		mutex       sync.Mutex
	)

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		start := i * chunkSize
		end := (i + 1) * chunkSize
		if i == numWorkers-1 {
			end = iterations
		}

		go func(start, end int) {
			defer wg.Done()
			task, err := h.Start("oleksiibraun/parcs-nw-py")

			if err != nil {
				log.Fatal(err)
			}

			defer func() {
				if err := task.Shutdown(); err != nil {
					log.Print(err)
				}
			}()

			if func() bool {
				mutex.Lock()
				defer mutex.Unlock()
				return isComposite
			}() {
				return
			}

			if err := task.SendAll(a, r, s, start, end); err != nil {
				log.Fatal(err)
			}

			var isPrime bool
			if err := task.Recv(&isPrime); err != nil {
				log.Fatal(err)
			}

			if !isPrime {
				mutex.Lock()
				isComposite = true
				mutex.Unlock()
			}
		}(start, end)
	}

	wg.Wait()

	finalIsPrime := !isComposite

	elapsedTime := time.Since(startTime)

	log.Printf("Time taken: %s", elapsedTime)

	fmt.Printf("%d is %s\n", a, map[bool]string{true: "probably prime", false: "composite"}[finalIsPrime])
}

func main() {
	parcs.Exec(&Program{parcs.DefaultRunner()})
}
