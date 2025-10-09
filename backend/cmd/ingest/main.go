package main

import (
	"bufio"
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

	// Get file path from command line argument
	args := flag.Args()
	if len(args) < 1 {
		log.Fatal("Usage: go run cmd/ingest/main.go [--resume=N] <path-to-json-file>")
	}
	filePath := args[0]

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Fatalf("File does not exist: %s", filePath)
	}

	log.Printf("Starting ingestion of file: %s", filePath)

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
	if err := processFile(filePath, *offset, openSearchService); err != nil {
		log.Fatalf("Error processing file: %v", err)
	}

	// Finalize index (enable replicas and refresh)
	log.Println("Finalizing index...")
	if err := openSearchService.FinalizeIndex(); err != nil {
		log.Fatalf("Error finalizing index: %v", err)
	}

	log.Println("Ingestion completed successfully!")
}

func processFile(filePath string, alreadyProcessed int, openSearchService *services.OpenSearchService) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	totalProcessed := 0
	startTime := time.Now()
	skippedMalformed := 0

	numWorkers := runtime.NumCPU()
	docChan := make(chan map[string]interface{}, 1000)
	doneChan := make(chan bool, numWorkers)

	skipUntil := alreadyProcessed
	if skipUntil > 0 {
		log.Printf("Skipping first %d previously ingested documents...", skipUntil)
	}

	for i := 0; i < numWorkers; i++ {
		go func() {
			defer func() { doneChan <- true }()

			batch := make([]services.Document, 0, 5000)
			for rawDoc := range docChan {
				transformedDoc := openSearchService.TransformDocument(rawDoc)
				batch = append(batch, transformedDoc)

				if len(batch) >= 5000 {
					if err := openSearchService.BulkIndex(batch); err != nil {
						log.Printf("Error bulk indexing batch: %v", err)
					}
					batch = batch[:0]
				}
			}

			if len(batch) > 0 {
				if err := openSearchService.BulkIndex(batch); err != nil {
					log.Printf("Error bulk indexing final batch: %v", err)
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

	logProgress := func() {
		if totalProcessed%10000 == 0 {
			elapsed := time.Since(startTime)
			rate := float64(totalProcessed) / elapsed.Seconds()
			log.Printf("Processed %d documents (%.2f docs/sec)", totalProcessed, rate)
		}
	}

	skipDocIfNeeded := func() bool {
		if skipUntil > 0 {
			skipUntil--
			return true
		}
		return false
	}

	switch firstByte {
	case '[':
		dec := json.NewDecoder(reader)
		if _, err := dec.Token(); err != nil {
			return fmt.Errorf("error reading JSON array start: %w", err)
		}
		for dec.More() {
			var rawDoc map[string]interface{}
			if err := dec.Decode(&rawDoc); err != nil {
				log.Printf("Error decoding JSON object: %v", err)
				skippedMalformed++
				continue
			}

			if skipDocIfNeeded() {
				continue
			}

			docChan <- rawDoc
			totalProcessed++
			logProgress()
		}
		if _, err := dec.Token(); err != nil {
			return fmt.Errorf("error reading JSON array end: %w", err)
		}
	default:
		if err := streamBareObjects(reader, func(rawDoc map[string]interface{}) {
			if skipDocIfNeeded() {
				return
			}
			docChan <- rawDoc
			totalProcessed++
			logProgress()
		}, func(err error) {
			skippedMalformed++
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

	totalTime := time.Since(startTime)
	log.Printf("Total documents processed: %d in %v (%.2f docs/sec)",
		totalProcessed, totalTime, float64(totalProcessed)/totalTime.Seconds())
	if skippedMalformed > 0 {
		log.Printf("Skipped %d malformed documents", skippedMalformed)
	}

	return nil
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

func streamBareObjects(r *bufio.Reader, emit func(map[string]interface{}), onMalformed func(error)) error {
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
		emit(rawDoc)
		builder.Reset()
		return nil
	}

	for {
		b, err := r.ReadByte()
		if err != nil {
			if errors.Is(err, io.EOF) {
				if err := flush(); err != nil {
					return err
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
