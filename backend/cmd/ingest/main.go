package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"notorious-backend/internal/config"
	"notorious-backend/internal/services"
	"os"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"github.com/joho/godotenv"
 )

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	offset := flag.Int("resume", 0, "number of documents already ingested; skip this many")
	flag.Parse()

	// Load configuration
	cfg := config.Load()

	// Initialize OpenSearch service
	openSearchService := services.NewOpenSearchService(cfg)

	// Get input path from command line argument
	args := flag.Args()
	if len(args) < 1 {
		log.Fatal("Usage: go run cmd/ingest/main.go [--resume=N] <path-to-json-file|s3://bucket/key|->")
	}
	inputPath := args[0]

	// Resolve input reader (local file, S3 object, or stdin)
	inputReader, err := resolveInput(inputPath, cfg)
	if err != nil {
		log.Fatalf("Error resolving input %s: %v", inputPath, err)
	}
	defer inputReader.Close()

	log.Printf("Starting ingestion of input: %s", inputPath)

	// Apply index template
	log.Println("Applying index template...")
	if err := openSearchService.ApplyIndexTemplate(); err != nil {
		log.Fatalf("Error applying index template: %v", err)
	}

	// Create index
	log.Println("Creating index...")
	if err := openSearchService.CreateIndex(); err != nil {
		log.Fatalf("Error creating index: %v", err)
	}

	// Process file
	if err := processFile(inputReader, *offset, openSearchService); err != nil {
		log.Fatalf("Error processing file: %v", err)
	}

	// Finalize index (enable replicas and refresh)
	log.Println("Finalizing index...")
	if err := openSearchService.FinalizeIndex(); err != nil {
		log.Fatalf("Error finalizing index: %v", err)
	}

	log.Println("Ingestion completed successfully!")
}

func processFile(input io.Reader, alreadyProcessed int, openSearchService *services.OpenSearchService) error {
	reader := bufio.NewReader(input)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var totalProcessed int64
	startTime := time.Now()
	var skippedMalformed int64

	numWorkers := runtime.NumCPU()
	docChan := make(chan map[string]interface{}, 1000)
	doneChan := make(chan struct{}, numWorkers)
	firstErr := make(chan error, 1)

	skipUntil := alreadyProcessed
	if skipUntil > 0 {
		log.Printf("Skipping first %d previously ingested documents...", skipUntil)
	}

	for i := 0; i < numWorkers; i++ {
		workerID := i
		go func() {
			defer func() { doneChan <- struct{}{} }()

			batch := make([]services.Document, 0, 5000)

			flush := func() bool {
				if len(batch) == 0 {
					return true
				}
				if err := openSearchService.BulkIndex(batch); err != nil {
					select {
					case firstErr <- fmt.Errorf("worker %d bulk index error: %w", workerID, err):
					default:
					}
					cancel()
					return false
				}
				batch = batch[:0]
				return true
			}

			for {
				select {
				case <-ctx.Done():
					return
				case rawDoc, ok := <-docChan:
					if !ok {
						flush()
						return
					}

					transformedDoc := openSearchService.TransformDocument(rawDoc)
					batch = append(batch, transformedDoc)

					if len(batch) >= 5000 {
						if !flush() {
							return
						}
					}
				}
			}
		}()
	}

	firstByte, err := peekFirstNonWhitespace(reader)
	if err != nil {
		close(docChan)
		for i := 0; i < numWorkers; i++ {
			<-doneChan
		}
		if errors.Is(err, io.EOF) {
			return nil
		}
		return fmt.Errorf("unable to inspect file format: %w", err)
	}

	const logEvery = int64(10000)

	skipDocIfNeeded := func() bool {
		if skipUntil > 0 {
			skipUntil--
			return true
		}
		return false
	}

	enqueueDocument := func(rawDoc map[string]interface{}) error {
		if skipDocIfNeeded() {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case docChan <- rawDoc:
			total := atomic.AddInt64(&totalProcessed, 1)
			if total%logEvery == 0 {
				elapsed := time.Since(startTime)
				rate := float64(total) / elapsed.Seconds()
				log.Printf("Processed %d documents (%.2f docs/sec)", total, rate)
			}
		}

		return nil
	}

	monitorTicker := time.NewTicker(30 * time.Second)
	defer monitorTicker.Stop()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-monitorTicker.C:
				processed := atomic.LoadInt64(&totalProcessed)
				skipped := atomic.LoadInt64(&skippedMalformed)
				elapsed := time.Since(startTime)
				rate := float64(0)
				if elapsed.Seconds() > 0 {
					rate = float64(processed) / elapsed.Seconds()
				}
				log.Printf("[monitor] processed=%d skipped=%d queue=%d elapsed=%s rate=%.2f docs/sec",
					processed, skipped, len(docChan), elapsed.Round(time.Second), rate)
			}
		}
	}()

	switch firstByte {
	case '[':
		dec := json.NewDecoder(reader)
		if _, err := dec.Token(); err != nil {
			return fmt.Errorf("error reading JSON array start: %w", err)
		}
		for dec.More() {
			if ctx.Err() != nil {
				break
			}
			var rawDoc map[string]interface{}
			if err := dec.Decode(&rawDoc); err != nil {
				log.Printf("Error decoding JSON object: %v", err)
				atomic.AddInt64(&skippedMalformed, 1)
				continue
			}

			if err := enqueueDocument(rawDoc); err != nil {
				break
			}
		}
		if _, err := dec.Token(); err != nil {
			return fmt.Errorf("error reading JSON array end: %w", err)
		}
	default:
		if err := streamBareObjects(ctx, reader, func(rawDoc map[string]interface{}) error {
			if err := enqueueDocument(rawDoc); err != nil {
				return err
			}
			return nil
		}, func(err error) {
			atomic.AddInt64(&skippedMalformed, 1)
			log.Printf("Malformed document skipped: %v", err)
		}); err != nil {
			close(docChan)
			for i := 0; i < numWorkers; i++ {
				<-doneChan
			}
			return err
		}
	}

	close(docChan)
	for i := 0; i < numWorkers; i++ {
		<-doneChan
	}

	select {
	case err := <-firstErr:
		if err != nil {
			return err
		}
	default:
	}

	totalTime := time.Since(startTime)
	finalTotal := atomic.LoadInt64(&totalProcessed)
	finalSkipped := atomic.LoadInt64(&skippedMalformed)
	rate := float64(0)
	if totalTime.Seconds() > 0 {
		rate = float64(finalTotal) / totalTime.Seconds()
	}

	log.Printf("Total documents processed: %d in %v (%.2f docs/sec)",
		finalTotal, totalTime, rate)
	if finalSkipped > 0 {
		log.Printf("Skipped %d malformed documents", finalSkipped)
	}

	return nil
}

func resolveInput(path string, cfg *config.Config) (io.ReadCloser, error) {
	if path == "-" {
		log.Println("Reading data from stdin")
		return io.NopCloser(os.Stdin), nil
	}

	if strings.HasPrefix(path, "s3://") {
		bucket, key, err := parseS3URI(path)
		if err != nil {
			return nil, err
		}

		s3Service, err := services.NewS3StreamService(cfg)
		if err != nil {
			return nil, fmt.Errorf("error creating S3 stream service: %w", err)
		}

		log.Printf("Streaming input from S3: s3://%s/%s", bucket, key)
		reader, err := s3Service.GetObject(context.Background(), bucket, key)
		if err != nil {
			return nil, err
		}
		return reader, nil
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("error opening file %s: %w", path, err)
	}
	log.Printf("Reading input from local file: %s", path)
	return file, nil
}

func parseS3URI(uri string) (string, string, error) {
	trimmed := strings.TrimPrefix(uri, "s3://")
	parts := strings.SplitN(trimmed, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid S3 URI: %s", uri)
	}
	return parts[0], parts[1], nil
}

func peekFirstNonWhitespace(r *bufio.Reader) (byte, error) {
	for {
		b, err := r.ReadByte()
		if err != nil {
			return 0, err
		}

		// Handle UTF-8 BOM (0xEF 0xBB 0xBF)
		if b == 0xEF {
			next1, err := r.ReadByte()
			if err != nil {
				return 0, err
			}
			next2, err := r.ReadByte()
			if err != nil {
				return 0, err
			}

			if next1 == 0xBB && next2 == 0xBF {
				continue
			}

			if err := r.UnreadByte(); err != nil {
				return 0, err
			}
			if err := r.UnreadByte(); err != nil {
				return 0, err
			}
			if err := r.UnreadByte(); err != nil {
				return 0, err
			}
			return b, nil
		}

		if b == ' ' || b == '\n' || b == '\r' || b == '\t' {
			continue
		}
		if err := r.UnreadByte(); err != nil {
			return 0, err
		}
		return b, nil
	}
}

func streamBareObjects(ctx context.Context, r *bufio.Reader, emit func(map[string]interface{}) error, onMalformed func(error)) error {
	var (
		builder strings.Builder
		depth   int
		inObj   bool
	)

	flush := func() error {
		if builder.Len() == 0 {
			return nil
		}
		var rawDoc map[string]interface{}
		if err := json.Unmarshal([]byte(builder.String()), &rawDoc); err != nil {
			onMalformed(fmt.Errorf("error decoding JSON object: %w", err))
			return nil
		}
		if err := emit(rawDoc); err != nil {
			return err
		}
		builder.Reset()
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		b, err := r.ReadByte()
		if err != nil {
			if errors.Is(err, io.EOF) {
				if err := flush(); err != nil {
					return err
				}
				if ctxErr := ctx.Err(); ctxErr != nil {
					return ctxErr
				}
				return nil
			}
			return err
		}

		if b == '{' {
			depth++
			inObj = true
		}
		if inObj {
			builder.WriteByte(b)
		}
		if b == '}' {
			depth--
			if depth == 0 && inObj {
				if err := flush(); err != nil {
					return err
				}
				inObj = false
			}
		}
	}
}
