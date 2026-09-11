package main

import (
	"flag"
	"fmt"
	"net/http"
	"sync"
	"time"
)

func main() {
	url := flag.String("url", "http://localhost:8080/readyz", "Target URL")
	concurrency := flag.Int("c", 10, "Number of concurrent workers")
	totalRequests := flag.Int("n", 100, "Total number of requests")
	flag.Parse()

	fmt.Printf("Starting load test: %d requests, %d concurrency, URL: %s\n", *totalRequests, *concurrency, *url)

	start := time.Now()
	results := make(chan time.Duration, *totalRequests)
	errors := make(chan error, *totalRequests)

	var wg sync.WaitGroup
	reqChan := make(chan struct{}, *totalRequests)

	// Fill request channel
	for i := 0; i < *totalRequests; i++ {
		reqChan <- struct{}{}
	}
	close(reqChan)

	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client := &http.Client{Timeout: 5 * time.Second}
			for range reqChan {
				reqStart := time.Now()
				resp, err := client.Get(*url)
				if err != nil {
					errors <- err
					continue
				}
				resp.Body.Close()
				if resp.StatusCode >= 400 {
					errors <- fmt.Errorf("status code: %d", resp.StatusCode)
					continue
				}
				results <- time.Since(reqStart)
			}
		}()
	}

	wg.Wait()
	close(results)
	close(errors)

	duration := time.Since(start)

	successCount := 0
	var totalLatency time.Duration
	for lat := range results {
		successCount++
		totalLatency += lat
	}

	errCount := 0
	for range errors {
		errCount++
	}

	fmt.Printf("\nSummary:\n")
	fmt.Printf("  Total time:     %v\n", duration)
	fmt.Printf("  Total requests: %d\n", *totalRequests)
	fmt.Printf("  Successful:     %d\n", successCount)
	fmt.Printf("  Failed:         %d\n", errCount)

	if successCount > 0 {
		avgLatency := totalLatency / time.Duration(successCount)
		fmt.Printf("  Avg Latency:    %v\n", avgLatency)
		fmt.Printf("  Requests/sec:   %.2f\n", float64(successCount)/duration.Seconds())
	}
}
