package main

import (
	"fmt"
	"github.com/lionell/parcs/go/parcs"
	"log"
	"math"
	"os"
	"strconv"
	"sync"
	"time"
)

type Program struct {
	*parcs.Runner
}

func sequentialFactorization(n int) ([]int, int, time.Duration) {
	start := time.Now()
	factors := make([]int, 0)
	tempN := n

	for tempN % 2 == 0 {
		factors = append(factors, 2)
		tempN /= 2
	}

	sqrtN := int(math.Sqrt(float64(n))) + 1
	for i := 3; i <= sqrtN; i += 2 {
		for tempN % i == 0 {
			factors = append(factors, i)
			tempN /= i
		}
	}

	if tempN > 1 && tempN != n {
		factors = append(factors, tempN)
	}

	elapsed := time.Since(start)
	return factors, tempN, elapsed
}

func (h *Program) Run() {
	n, err := strconv.Atoi(os.Getenv("N"))
	if err != nil {
		log.Fatal(err)
	}

	numWorkers, err := strconv.Atoi(os.Getenv("WORKERS"))
	if err != nil {
		numWorkers = 0
		log.Printf("WORKERS environment variable not set or invalid. Using default value: %d (sequential)", numWorkers)
	}

	if numWorkers <= 0 {
		// Sequential execution
		factors, remainder, elapsed := sequentialFactorization(n)

		log.Printf("(Sequential) Factors of %d: %v", n, factors)
		log.Printf("(Sequential) Remainder: %d", remainder)
		log.Printf("(Sequential) Time taken: %s", elapsed)

		fmt.Printf("Factors of %d: ", n)
		for i, factor := range factors {
			fmt.Printf("%d", factor)
			if i < len(factors)-1 {
				fmt.Printf(", ")
			}
		}
		fmt.Printf("\nRemainder: %d\n", remainder)

	} else {
		start := time.Now()

		sqrtN := int(math.Sqrt(float64(n))) + 1
		chunkSize := sqrtN / numWorkers

		var wg sync.WaitGroup
		allFactors := make([]int, 0)
		var allFactorsMutex sync.Mutex
		remainingN := n

		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			startRange := i*chunkSize + 1
			endRange := (i + 1)*chunkSize + 1

			if i == numWorkers - 1 {
				endRange = sqrtN
			}

			go func(workerID int, startRange int, endRange int) {
				defer wg.Done()

				t, err := h.Start("oleksiibraun/parcs-nw-py")
				if err != nil {
					log.Fatalf("(Worker %d) Error starting task: %v", workerID, err)
				}

				log.Printf("(Worker %d) Sending data (n = %d, start = %d, end = %d)", workerID, n, startRange, endRange)
				if err := t.SendAll(n, startRange, endRange); err != nil {
					log.Fatalf("(Worker %d) Error sending data: %v", workerID, err)
				}

				var factors []int
				if err := t.Recv(&factors); err != nil {
					log.Fatalf("(Worker %d) Error receiving factors: %v", workerID, err)
				}

				var remainder int
				if err := t.Recv(&remainder); err != nil {
					log.Fatalf("(Worker %d) Error receiving remainder: %v", workerID, err)
				}

				t.Shutdown()

				log.Printf("(Worker %d) Factors found: %v", workerID, factors)
				log.Printf("(Worker %d) Remainder: %v", workerID, remainder)

				allFactorsMutex.Lock()
				allFactors = append(allFactors, factors...)
				if len(factors) > 0 {
					remainingN = remainder
				}
				allFactorsMutex.Unlock()
			}(i, startRange, endRange)
		}

		wg.Wait()

		if remainingN > 1 && remainingN != n {
			allFactors = append(allFactors, remainingN)
		}

		elapsed := time.Since(start)

		log.Printf("All factors found: %v", allFactors)
		log.Printf("Final remainder: %v", remainingN)
		log.Printf("(Parallel) Time taken: %s", elapsed)

		fmt.Printf("Factors of %d: ", n)
		for i, factor := range allFactors {
			fmt.Printf("%d", factor)
			if i < len(allFactors)-1 {
				fmt.Printf(", ")
			}
		}
		fmt.Printf("\nRemainder: %d\n", remainingN)
	}
}

func main() {
	parcs.Exec(&Program{parcs.DefaultRunner()})
}
