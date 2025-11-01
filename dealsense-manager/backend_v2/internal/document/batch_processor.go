package document

import (
	"context"
	"fmt"
	"io"
	"time"

	documentai "cloud.google.com/go/documentai/apiv1"
	"cloud.google.com/go/documentai/apiv1/documentaipb"
	"cloud.google.com/go/storage"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/option"
)

// BatchProcessor handles async batch processing for large documents
type BatchProcessor struct {
	client        *documentai.DocumentProcessorClient
	storageClient *storage.Client
	config        ProcessorConfig
	ctx           context.Context
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(config ProcessorConfig, storageClient *storage.Client) (*BatchProcessor, error) {
	ctx := context.Background()

	var client *documentai.DocumentProcessorClient
	var err error

	if config.UseDefaultCreds {
		client, err = documentai.NewDocumentProcessorClient(ctx)
	} else if config.CredentialsJSON != "" {
		client, err = documentai.NewDocumentProcessorClient(ctx, option.WithCredentialsJSON([]byte(config.CredentialsJSON)))
	} else {
		return nil, fmt.Errorf("no credentials provided for Document AI batch processor")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create batch processor client: %w", err)
	}

	return &BatchProcessor{
		client:        client,
		storageClient: storageClient,
		config:        config,
		ctx:           ctx,
	}, nil
}

// BatchProcessResult represents the result of a batch processing operation
type BatchProcessResult struct {
	ProcessedDocument *ProcessedDocument
	OperationName     string
	Status            string // "processing", "completed", "failed"
	Error             error
}

// ProcessDocumentBatch processes a document using the batch API
// This is designed for documents > 15 pages with images
func (b *BatchProcessor) ProcessDocumentBatch(inputGCSURI, outputGCSPrefix string, mimeType string) (*documentaipb.BatchProcessResponse, error) {
	logrus.Infof("Starting batch processing for document: %s", inputGCSURI)

	processorName := fmt.Sprintf("projects/%s/locations/%s/processors/%s",
		b.config.ProjectID, b.config.Location, b.config.ProcessorID)

	// Create batch process request
	req := &documentaipb.BatchProcessRequest{
		Name: processorName,
		InputDocuments: &documentaipb.BatchDocumentsInputConfig{
			Source: &documentaipb.BatchDocumentsInputConfig_GcsDocuments{
				GcsDocuments: &documentaipb.GcsDocuments{
					Documents: []*documentaipb.GcsDocument{
						{
							GcsUri:   inputGCSURI,
							MimeType: mimeType,
						},
					},
				},
			},
		},
		DocumentOutputConfig: &documentaipb.DocumentOutputConfig{
			Destination: &documentaipb.DocumentOutputConfig_GcsOutputConfig_{
				GcsOutputConfig: &documentaipb.DocumentOutputConfig_GcsOutputConfig{
					GcsUri: outputGCSPrefix,
				},
			},
		},
	}

	// Start batch processing operation
	op, err := b.client.BatchProcessDocuments(b.ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to start batch processing: %w", err)
	}

	logrus.Infof("Batch processing started, operation name: %s", op.Name())

	// Wait for operation to complete (with timeout)
	ctx, cancel := context.WithTimeout(b.ctx, 30*time.Minute) // Batch can take a while
	defer cancel()

	resp, err := op.Wait(ctx)
	if err != nil {
		return nil, fmt.Errorf("batch processing operation failed: %w", err)
	}

	logrus.Info("Batch processing completed successfully")
	return resp, nil
}

// ProcessDocumentBatchAsync starts batch processing and returns operation name for polling
func (b *BatchProcessor) ProcessDocumentBatchAsync(inputGCSURI, outputGCSPrefix string, mimeType string) (string, error) {
	logrus.Infof("Starting async batch processing for document: %s", inputGCSURI)

	processorName := fmt.Sprintf("projects/%s/locations/%s/processors/%s",
		b.config.ProjectID, b.config.Location, b.config.ProcessorID)

	req := &documentaipb.BatchProcessRequest{
		Name: processorName,
		InputDocuments: &documentaipb.BatchDocumentsInputConfig{
			Source: &documentaipb.BatchDocumentsInputConfig_GcsDocuments{
				GcsDocuments: &documentaipb.GcsDocuments{
					Documents: []*documentaipb.GcsDocument{
						{
							GcsUri:   inputGCSURI,
							MimeType: mimeType,
						},
					},
				},
			},
		},
		DocumentOutputConfig: &documentaipb.DocumentOutputConfig{
			Destination: &documentaipb.DocumentOutputConfig_GcsOutputConfig_{
				GcsOutputConfig: &documentaipb.DocumentOutputConfig_GcsOutputConfig{
					GcsUri: outputGCSPrefix,
				},
			},
		},
	}

	op, err := b.client.BatchProcessDocuments(b.ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to start async batch processing: %w", err)
	}

	operationName := op.Name()
	logrus.Infof("Async batch processing started, operation: %s", operationName)
	return operationName, nil
}

// GetOperationStatus checks the status of a batch processing operation
func (b *BatchProcessor) GetOperationStatus(operationName string) (bool, error) {
	// Note: This would typically use the Operations API
	// For now, we'll implement a simple check
	logrus.Infof("Checking status of operation: %s", operationName)
	// Implementation would go here - this is a placeholder
	return true, nil
}

// RetrieveBatchResult retrieves and parses the batch processing result from GCS
func (b *BatchProcessor) RetrieveBatchResult(outputGCSPrefix string) (*ProcessedDocument, error) {
	// The batch processor outputs results to GCS in a specific format
	// We need to read the output JSON and convert it to ProcessedDocument
	logrus.Infof("Retrieving batch result from: %s", outputGCSPrefix)

	// List files in output directory
	bucket := b.storageClient.Bucket(extractBucketName(outputGCSPrefix))
	prefix := extractObjectPrefix(outputGCSPrefix)

	// The output is typically in format: {outputPrefix}/0/output-1-to-1.json
	outputPath := fmt.Sprintf("%s/0/output-1-to-1.json", prefix)
	
	obj := bucket.Object(outputPath)
	reader, err := obj.NewReader(b.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read batch output: %w", err)
	}
	defer reader.Close()

	// Read the document proto
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read batch output data: %w", err)
	}

	// Parse the Document proto from JSON
	// Note: The actual batch output format needs to be parsed properly
	// This is a simplified implementation
	logrus.Infof("Retrieved batch result, size: %d bytes", len(data))

	// TODO: Parse the actual Document proto from the batch output
	// For now, return a placeholder
	return &ProcessedDocument{
		Text:   string(data),
		Pages:  1,
		Metadata: map[string]interface{}{
			"source": "batch_processing",
		},
	}, nil
}

// extractBucketName extracts bucket name from GCS URI
func extractBucketName(gcsURI string) string {
	// Format: gs://bucket-name/path/to/object
	if len(gcsURI) > 5 && gcsURI[:5] == "gs://" {
		rest := gcsURI[5:]
		for i, c := range rest {
			if c == '/' {
				return rest[:i]
			}
		}
		return rest
	}
	return ""
}

// extractObjectPrefix extracts object prefix from GCS URI
func extractObjectPrefix(gcsURI string) string {
	// Format: gs://bucket-name/path/to/object
	if len(gcsURI) > 5 && gcsURI[:5] == "gs://" {
		rest := gcsURI[5:]
		for i, c := range rest {
			if c == '/' {
				return rest[i+1:]
			}
		}
	}
	return ""
}

// Close closes the batch processor client
func (b *BatchProcessor) Close() error {
	if b.client != nil {
		return b.client.Close()
	}
	return nil
}

// IntelligentProcessDocument automatically chooses sync or async based on document size
// Returns: processed document, isAsync flag, operation name (if async), error
func (p *DocumentProcessor) IntelligentProcessDocument(fileData io.Reader, mimeType string, estimatedPages int, gcsURI string, outputPrefix string) (*ProcessedDocument, bool, string, error) {
	const syncPageLimit = 15
	
	logrus.Infof("Intelligent processing: estimated pages=%d, mime=%s", estimatedPages, mimeType)

	// For documents ≤15 pages, use sync processing (supports images)
	if estimatedPages <= syncPageLimit {
		logrus.Info("Using sync processing (≤15 pages)")
		doc, err := p.ProcessDocument(fileData, mimeType)
		return doc, false, "", err
	}

	// For documents >15 pages, we MUST use async batch processing
	logrus.Info("Using async batch processing (>15 pages)")
	
	// Note: Batch processing requires the document to be in GCS already
	// The caller should have uploaded it to GCS first
	if gcsURI == "" {
		return nil, false, "", fmt.Errorf("batch processing requires GCS URI for documents >%d pages", syncPageLimit)
	}

	// Create batch processor if not exists
	if p.batchProcessor == nil {
		// We need storage client for batch processor
		return nil, false, "", fmt.Errorf("batch processor not initialized")
	}

	// Start async batch processing
	operationName, err := p.batchProcessor.ProcessDocumentBatchAsync(gcsURI, outputPrefix, mimeType)
	if err != nil {
		return nil, false, "", fmt.Errorf("failed to start batch processing: %w", err)
	}

	// Return nil document with operation name for polling
	return nil, true, operationName, nil
}


