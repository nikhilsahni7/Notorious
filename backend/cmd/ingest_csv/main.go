package main

import (
	"bufio"
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"notorious-backend/internal/config"
	"notorious-backend/internal/services"
	"os"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Command line flags
	csvFilePath := flag.String("file", "", "Path to CSV file (required)")
	region := flag.String("region", "delhi-ncr", "Region for the data (default: delhi-ncr)")
	offset := flag.Int("resume", 0, "Number of documents already ingested; skip this many")
	batchSize := flag.Int("batch", 25000, "Batch size for bulk indexing")
	flag.Parse()

	if *csvFilePath == "" {
		log.Fatal("Usage: go run cmd/ingest_csv/main.go -file=/path/to/data.csv [-region=delhi-ncr] [-resume=0] [-batch=5000]")
	}

	log.Printf("🚀 Starting CSV ingestion from: %s", *csvFilePath)
	log.Printf("📍 Region: %s", *region)
	log.Printf("📦 Batch size: %d", *batchSize)
	if *offset > 0 {
		log.Printf("⏭️  Resuming from offset: %d", *offset)
	}

	// Load configuration
	cfg := config.Load()
	cfg.IngestBatchSize = *batchSize // Override batch size if provided

	// Initialize OpenSearch service
	openSearchService := services.NewOpenSearchService(cfg)

	// Apply index template
	log.Println("📋 Applying index template...")
	if err := openSearchService.ApplyIndexTemplate(); err != nil {
		log.Fatalf("❌ Error applying index template: %v", err)
	}

	// Create index if it doesn't exist
	log.Println("🏗️  Creating index (if not exists)...")
	if err := openSearchService.CreateIndex(); err != nil {
		log.Printf("⚠️  Index might already exist: %v", err)
	}

	// Open CSV file
	file, err := os.Open(*csvFilePath)
	if err != nil {
		log.Fatalf("❌ Error opening CSV file: %v", err)
	}
	defer file.Close()

	// Process CSV file
	if err := processCSV(file, *region, *offset, cfg, openSearchService); err != nil {
		log.Fatalf("❌ Error processing CSV: %v", err)
	}

	// Finalize index (enable replicas and refresh)
	log.Println("✅ Finalizing index...")
	if err := openSearchService.FinalizeIndex(); err != nil {
		log.Fatalf("❌ Error finalizing index: %v", err)
	}

	log.Println("🎉 CSV ingestion completed successfully!")
}

func processCSV(file *os.File, region string, offset int, cfg *config.Config, openSearchService *services.OpenSearchService) error {
	reader := csv.NewReader(bufio.NewReader(file))
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var totalProcessed int64
	var skippedRows int64
	startTime := time.Now()

	numWorkers := runtime.NumCPU() * cfg.IngestWorkerMultiplier
	if numWorkers < 1 {
		numWorkers = 1
	}
	batchSize := cfg.IngestBatchSize

	log.Printf("⚙️  Using %d workers", numWorkers)

	// Channels for worker pool
	docChan := make(chan map[string]interface{}, batchSize*numWorkers)
	doneChan := make(chan struct{}, numWorkers)

	// Track last checkpoint for resume capability
	var lastCheckpoint int64

	// Start workers
	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("❌ PANIC in worker %d: %v", workerID, r)
					log.Printf("💾 CHECKPOINT: Last successful count = %d (use -resume=%d to continue)",
						atomic.LoadInt64(&totalProcessed), atomic.LoadInt64(&totalProcessed))
				}
				doneChan <- struct{}{}
			}()

			batch := make([]services.Document, 0, batchSize)
			consecutiveErrors := 0
			maxConsecutiveErrors := 10

			for doc := range docChan {
				transformed := openSearchService.TransformDocument(doc)
				transformed.Region = region // Set region for all documents
				batch = append(batch, transformed)

				if len(batch) >= batchSize {
					if err := openSearchService.BulkIndex(batch); err != nil {
						consecutiveErrors++
						log.Printf("⚠️  Worker %d bulk index error (%d/%d): %v", workerID, consecutiveErrors, maxConsecutiveErrors, err)
						if consecutiveErrors >= maxConsecutiveErrors {
							log.Printf("❌ Worker %d: Too many consecutive errors, stopping", workerID)
							log.Printf("💾 CHECKPOINT: Processed ~%d documents (use -resume=%d to continue)",
								atomic.LoadInt64(&totalProcessed), atomic.LoadInt64(&totalProcessed))
							return
						}
						// Wait before retry
						time.Sleep(5 * time.Second)
					} else {
						consecutiveErrors = 0 // Reset on success
						processed := atomic.AddInt64(&totalProcessed, int64(len(batch)))

						// Log progress every 50k
						if processed%50000 == 0 {
							elapsed := time.Since(startTime)
							rate := float64(processed) / elapsed.Seconds()
							log.Printf("📊 Progress: %d documents | %.0f docs/sec | %s elapsed",
								processed, rate, elapsed.Round(time.Second))
						}

						// Checkpoint every 1 million
						if processed/1000000 > lastCheckpoint/1000000 {
							atomic.StoreInt64(&lastCheckpoint, processed)
							log.Printf("💾 CHECKPOINT: %d million documents processed (use -resume=%d to continue)",
								processed/1000000, processed)
						}
					}
					batch = batch[:0]
				}
			}

			// Process remaining batch
			if len(batch) > 0 {
				if err := openSearchService.BulkIndex(batch); err != nil {
					log.Printf("⚠️  Worker %d final batch error: %v", workerID, err)
				} else {
					atomic.AddInt64(&totalProcessed, int64(len(batch)))
				}
			}
		}(i)
	}

	// Read CSV header
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("error reading CSV header: %v", err)
	}

	log.Printf("📄 CSV Headers: %v", header)

	// Map column names to indices (no strict validation - ingest whatever columns exist)
	expectedCols := []string{"mobile", "name", "fname", "address", "id", "alt", "email", "oid"}
	colIndices := make(map[string]int)
	for i, col := range header {
		colIndices[col] = i
	}

	// Log which expected columns are present/missing (but don't fail)
	for _, col := range expectedCols {
		if _, exists := colIndices[col]; !exists {
			log.Printf("⚠️  Column '%s' not found in CSV (will be empty for all rows)", col)
		}
	}

	log.Printf("✅ Found %d columns in CSV, proceeding with ingestion...", len(header))

	// Skip offset rows if resuming
	rowNum := 0
	if offset > 0 {
		log.Printf("⏭️  Skipping first %d rows...", offset)
		for rowNum < offset {
			if _, err := reader.Read(); err != nil {
				if err == io.EOF {
					log.Println("⚠️  Reached EOF during offset skip")
					return nil
				}
				return fmt.Errorf("error skipping rows: %v", err)
			}
			rowNum++
		}
		log.Printf("✅ Skipped %d rows, starting ingestion...", offset)
	}

	// Process CSV rows
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			atomic.AddInt64(&skippedRows, 1)
			log.Printf("⚠️  Error reading row %d: %v (skipping)", rowNum+1, err)
			rowNum++
			continue
		}

		rowNum++

		// Build document from CSV row
		doc := make(map[string]interface{})
		for colName, colIdx := range colIndices {
			if colIdx < len(record) {
				value := record[colIdx]
				if value != "" { // Only add non-empty values
					doc[colName] = value
				}
			}
		}

		// Note: oid, year_of_registration, and alt_address are set in TransformDocument()
		// All rows are ingested regardless of null/empty fields

		select {
		case docChan <- doc:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	close(docChan)

	// Wait for all workers to finish
	for i := 0; i < numWorkers; i++ {
		<-doneChan
	}

	elapsed := time.Since(startTime)
	rate := float64(totalProcessed) / elapsed.Seconds()

	log.Printf("\n"+
		"═══════════════════════════════════════════════════════\n"+
		"  📊 INGESTION SUMMARY\n"+
		"═══════════════════════════════════════════════════════\n"+
		"  ✅ Total processed: %d documents\n"+
		"  ⚠️  Skipped rows: %d\n"+
		"  ⏱️  Time elapsed: %s\n"+
		"  🚀 Average rate: %.0f docs/sec\n"+
		"  📍 Region: %s\n"+
		"═══════════════════════════════════════════════════════\n",
		totalProcessed, skippedRows, elapsed.Round(time.Second), rate, region)

	return nil
}
