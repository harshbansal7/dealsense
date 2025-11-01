package document

import (
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"joinly-manager/internal/database"
)

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ServiceConfig holds configuration for the document service
type ServiceConfig struct {
	Storage      StorageConfig
	Processor    ProcessorConfig
	Embedding    EmbeddingConfig
	VectorSearch *VectorSearchConfig // Optional: if nil, uses PostgreSQL JSONB
}

// Service orchestrates document management, processing, and search
type Service struct {
	db               *database.Database
	storage          *StorageClient
	processor        *DocumentProcessor
	embeddingService *EmbeddingService
	vectorSearch     *VectorSearchService // Optional: nil if not configured
	useVectorSearch  bool
}

// NewService creates a new document service
func NewService(db *database.Database, config ServiceConfig) (*Service, error) {
	// Initialize Google Cloud Storage
	storage, err := NewStorageClient(config.Storage)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage client: %w", err)
	}

	// Initialize Document AI processor
	processor, err := NewDocumentProcessor(config.Processor)
	if err != nil {
		storage.Close()
		return nil, fmt.Errorf("failed to initialize document processor: %w", err)
	}

	// Initialize Vertex AI embedding service
	embeddingService, err := NewEmbeddingService(config.Embedding)
	if err != nil {
		storage.Close()
		processor.Close()
		return nil, fmt.Errorf("failed to initialize embedding service: %w", err)
	}

	// Initialize Vector Search if configured
	var vectorSearch *VectorSearchService
	useVectorSearch := false
	if config.VectorSearch != nil && config.VectorSearch.Enabled {
		vectorSearch, err = NewVectorSearchService(*config.VectorSearch)
		if err != nil {
			logrus.Warnf("Failed to initialize Vector Search: %v. Falling back to PostgreSQL JSONB", err)
		} else {
			useVectorSearch = true
			logrus.Info("Vector Search enabled for low-latency semantic search")
		}
	}

	logrus.Info("Document service initialized successfully")

	return &Service{
		db:               db,
		storage:          storage,
		processor:        processor,
		embeddingService: embeddingService,
		vectorSearch:     vectorSearch,
		useVectorSearch:  useVectorSearch,
	}, nil
}

// UploadAndProcessDocument handles the complete document pipeline:
// 1. Upload to GCS
// 2. Process with Document AI (intelligently chooses sync/async based on size)
// 3. Generate embeddings
// 4. Store in database
func (s *Service) UploadAndProcessDocument(agentID uuid.UUID, fileName string, fileData io.Reader, contentType string) (*database.Document, error) {
	logrus.Infof("Starting document upload and processing for agent: %s, file: %s", agentID.String(), fileName)

	// 1. Upload to Google Cloud Storage
	storagePath, fileSize, err := s.storage.UploadDocument(agentID, fileName, fileData, contentType)
	if err != nil {
		return nil, fmt.Errorf("failed to upload document: %w", err)
	}

	// 2. Get actual page count by downloading and analyzing the file
	// Download from GCS to count pages
	reader, err := s.storage.DownloadDocument(storagePath)
	if err != nil {
		logrus.Warnf("Failed to download document for page counting: %v. Using fallback estimation.", err)
	}
	
	var pageCount int
	var isActual bool
	
	if reader != nil {
		defer reader.Close()
		pageCount, isActual = GetPageCountWithFallback(reader, fileSize, contentType)
		if isActual {
			logrus.Infof("Actual page count: %d", pageCount)
		} else {
			logrus.Infof("Estimated page count: %d (fallback)", pageCount)
		}
	} else {
		// Fallback to size-based estimation if download failed
		pageCount = EstimatePageCountFallback(fileSize, contentType)
		isActual = false
		logrus.Infof("Estimated page count: %d (download failed, using fallback)", pageCount)
	}

	// 3. Create database record
	doc := &database.Document{
		AgentID:      agentID,
		Name:         fileName,
		OriginalName: fileName,
		FileType:     contentType,
		FileSize:     fileSize,
		StoragePath:  storagePath,
		GCSBucket:    s.storage.bucketName,
		Status:       "processing",
		Metadata:     "{}", // Initialize with empty JSON object
	}

	if err := s.db.Create(doc).Error; err != nil {
		// Attempt to cleanup uploaded file
		s.storage.DeleteDocument(storagePath)
		return nil, fmt.Errorf("failed to create document record: %w", err)
	}

	// 4. Process document asynchronously (uses intelligent processing internally)
	go s.processDocumentAsyncIntelligent(doc.ID, storagePath, contentType, pageCount, fileSize)

	return doc, nil
}

// processDocumentAsyncIntelligent processes document intelligently based on size
func (s *Service) processDocumentAsyncIntelligent(documentID uuid.UUID, storagePath string, contentType string, estimatedPages int, fileSize int64) {
	logrus.Infof("Intelligent processing for document %s: %d estimated pages", documentID.String(), estimatedPages)

	const syncPageLimit = 15

	// For small documents (≤15 pages), use sync processing
	if estimatedPages <= syncPageLimit {
		logrus.Info("Using sync processing for small document")
		s.processDocumentSync(documentID, storagePath, contentType)
		return
	}

	// For large documents (>15 pages), use batch processing
	logrus.Info("Using batch processing for large document")
	s.processDocumentBatch(documentID, storagePath, contentType)
}

// processDocumentSync processes small documents synchronously
func (s *Service) processDocumentSync(documentID uuid.UUID, storagePath string, contentType string) {
	logrus.Infof("Sync processing document: %s", documentID.String())

	// Download from GCS
	reader, err := s.storage.DownloadDocument(storagePath)
	if err != nil {
		s.updateDocumentStatus(documentID, "failed", fmt.Sprintf("Failed to download: %v", err))
		return
	}
	defer reader.Close()

	// Process with Document AI
	processed, err := s.processor.ProcessDocument(reader, contentType)
	if err != nil {
		s.updateDocumentStatus(documentID, "failed", fmt.Sprintf("Failed to process: %v", err))
		return
	}

	// Store processed document
	s.storeProcessedDocument(documentID, processed, false, "", "")
}

// processDocumentBatch processes large documents using batch API
func (s *Service) processDocumentBatch(documentID uuid.UUID, storagePath string, contentType string) {
	logrus.Infof("Batch processing document: %s", documentID.String())

	// Update status to indicate batch processing
	s.updateDocumentStatus(documentID, "processing_batch", "")

	// Construct GCS URI
	gcsURI := fmt.Sprintf("gs://%s/%s", s.storage.bucketName, storagePath)
	outputPrefix := fmt.Sprintf("gs://%s/batch_output/%s", s.storage.bucketName, documentID.String())

	// Initialize batch processor if not exists
	if s.processor.batchProcessor == nil {
		storageClient := s.storage.client
		batchProcessor, err := NewBatchProcessor(s.processor.config, storageClient)
		if err != nil {
			s.updateDocumentStatus(documentID, "failed", fmt.Sprintf("Failed to init batch processor: %v", err))
			return
		}
		s.processor.batchProcessor = batchProcessor
	}

	// Start batch processing
	operationName, err := s.processor.batchProcessor.ProcessDocumentBatchAsync(gcsURI, outputPrefix, contentType)
	if err != nil {
		s.updateDocumentStatus(documentID, "failed", fmt.Sprintf("Failed to start batch processing: %v", err))
		return
	}

	// Update document with batch operation info
	updateData := map[string]interface{}{
		"used_batch_processing": true,
		"batch_operation_name":  operationName,
		"batch_output_path":     outputPrefix,
	}
	s.db.Model(&database.Document{}).Where("id = ?", documentID).Updates(updateData)

	logrus.Infof("Batch processing started for document %s, operation: %s", documentID.String(), operationName)

	// Wait for batch processing to complete (in background)
	// Poll operation status every 30 seconds
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	timeout := time.After(30 * time.Minute) // 30 minute timeout for batch processing

	for {
		select {
		case <-ticker.C:
			// Check if operation is complete by trying to retrieve result
			processed, err := s.processor.batchProcessor.RetrieveBatchResult(outputPrefix)
			if err == nil && processed != nil {
				logrus.Info("Batch processing completed successfully")
				s.storeProcessedDocument(documentID, processed, true, operationName, outputPrefix)
				return
			}
			logrus.Debugf("Batch operation still processing...")
		case <-timeout:
			s.updateDocumentStatus(documentID, "failed", "Batch processing timeout")
			return
		}
	}
}

// storeProcessedDocument stores the processed document and generates embeddings
func (s *Service) storeProcessedDocument(documentID uuid.UUID, processed *ProcessedDocument, wasBatch bool, operationName string, outputPath string) {
	// Sanitize extracted text
	sanitizedText := sanitizeUTF8(processed.Text)
	metadataJSON, _ := processed.GetMetadataJSON()
	now := time.Now()

	updateData := map[string]interface{}{
		"extracted_text": sanitizedText,
		"page_count":     processed.Pages,
		"metadata":       metadataJSON,
		"processed_at":   now,
		"status":         "processed",
	}

	if wasBatch {
		updateData["used_batch_processing"] = true
		if operationName != "" {
			updateData["batch_operation_name"] = operationName
		}
		if outputPath != "" {
			updateData["batch_output_path"] = outputPath
		}
	}

	if err := s.db.Model(&database.Document{}).Where("id = ?", documentID).Updates(updateData).Error; err != nil {
		logrus.Errorf("Failed to update document %s: %v", documentID.String(), err)
		return
	}

	// Generate and store embeddings
	if err := s.generateAndStoreEmbeddings(documentID, processed); err != nil {
		logrus.Errorf("Failed to generate embeddings for document %s: %v", documentID.String(), err)
		logrus.Infof("Document %s processed successfully despite embedding failure", documentID.String())
	} else {
		logrus.Infof("Document %s processed successfully with embeddings", documentID.String())
	}
}

// generateAndStoreEmbeddings generates embeddings for document chunks and stores them
func (s *Service) generateAndStoreEmbeddings(documentID uuid.UUID, processed *ProcessedDocument) error {
	logrus.Infof("Generating embeddings for document: %s", documentID.String())

	// Fetch the document to get agent_id (needed for Vector Search filtering)
	var doc database.Document
	if err := s.db.Where("id = ?", documentID).First(&doc).Error; err != nil {
		return fmt.Errorf("failed to fetch document for agent_id: %w", err)
	}

	// Chunk the document (uses visual context for pitch decks)
	chunks := s.processor.ChunkDocument(processed, 1000, 100)
	if len(chunks) == 0 {
		logrus.Warnf("No chunks generated from document %s", documentID.String())
		return nil // Not an error, just no chunks to process
	}

	logrus.Infof("Generated %d chunks for embedding (agent_id: %s)", len(chunks), doc.AgentID.String())

	// Generate embeddings in batches (Vertex AI supports batch processing)
	batchSize := 5 // Process 5 chunks at a time
	totalStored := 0

	for i := 0; i < len(chunks); i += batchSize {
		end := i + batchSize
		if end > len(chunks) {
			end = len(chunks)
		}

		batchChunks := chunks[i:end]
		validTexts := make([]string, 0, len(batchChunks))
		validIndices := make([]int, 0, len(batchChunks))

		// Filter out problematic chunks and sanitize the rest
		for j, chunk := range batchChunks {
			// Skip chunks that are too short
			if len(strings.TrimSpace(chunk.Text)) < 10 {
				logrus.Debugf("Skipping chunk %d: too short (%d chars)", j, len(chunk.Text))
				continue
			}

			// Try to sanitize the text
			sanitized := sanitizeUTF8(chunk.Text)

			// Skip chunks that still have invalid UTF-8 after sanitization
			if !utf8.ValidString(sanitized) {
				logrus.Warnf("Skipping chunk %d: invalid UTF-8 after sanitization", j)
				continue
			}

			validTexts = append(validTexts, sanitized)
			validIndices = append(validIndices, j)
		}

		if len(validTexts) == 0 {
			logrus.Warnf("No valid chunks in batch %d, skipping", i/batchSize)
			continue
		}

		// Generate embeddings for valid texts
		logrus.Infof("Generating embeddings for %d valid texts out of %d chunks", len(validTexts), len(batchChunks))
		embeddings, err := s.embeddingService.GenerateEmbeddings(validTexts)
		if err != nil {
			logrus.Errorf("Failed to generate embeddings for batch %d: %v", i/batchSize, err)
			// Continue with other batches instead of failing completely
			continue
		}

		// Store embeddings in Vector Search and/or database
		for j, embResult := range embeddings {
			originalIndex := validIndices[j]
			chunkData := batchChunks[originalIndex]

			// Convert chunk metadata to JSON
			metadataJSON := "{}"
			if len(chunkData.Metadata) > 0 {
				if data, err := processed.GetMetadataJSON(); err == nil {
					metadataJSON = data
				}
			}

			// Store in Vector Search if enabled
			var vectorSearchID string
			storedInVectorSearch := false
			
			if s.useVectorSearch && s.vectorSearch != nil {
				// Create datapoint for Vector Search
				datapointID := CreateDatapointID(documentID, chunkData.ChunkIndex)
				datapoint := VectorSearchDatapoint{
					ID:        datapointID,
					Embedding: embResult.Embedding,
					Metadata: map[string]interface{}{
						"agent_id":     doc.AgentID.String(), // CRITICAL: Include agent_id for filtering
						"document_id":  documentID.String(),
						"chunk_index":  chunkData.ChunkIndex,
						"page_number":  chunkData.PageNumber,
						"chunk_length": len(chunkData.Text),
					},
				}

				// Upsert to Vector Search
				logrus.Debugf("Attempting to store chunk %d in Vector Search with ID: %s", chunkData.ChunkIndex, datapointID)
				if err := s.vectorSearch.UpsertDatapoints([]VectorSearchDatapoint{datapoint}); err != nil {
					logrus.Errorf("Failed to store embedding in Vector Search for chunk %d: %v", chunkData.ChunkIndex, err)
					// Continue with PostgreSQL storage even if Vector Search fails
				} else {
					vectorSearchID = datapointID
					storedInVectorSearch = true
					logrus.Debugf("Successfully stored embedding in Vector Search for chunk %d", chunkData.ChunkIndex)
				}
			} else {
				if s.useVectorSearch && s.vectorSearch == nil {
					logrus.Warnf("Vector Search is enabled but service is nil - using PostgreSQL fallback")
				}
			}

			// Always store reference in PostgreSQL (with or without embedding data)
			var embeddingJSON string
			if s.useVectorSearch {
				// When using Vector Search, store empty JSON array instead of full embedding
				embeddingJSON = "[]"
			} else {
				// Only store full embedding data in PostgreSQL if not using Vector Search
				embeddingJSON, err = EmbeddingToJSON(embResult.Embedding)
				if err != nil {
					logrus.Warnf("Failed to marshal embedding: %v", err)
					continue
				}
			}

			docEmbedding := &database.DocumentEmbedding{
				DocumentID:           documentID,
				ChunkIndex:           chunkData.ChunkIndex,
				ChunkText:            chunkData.Text,
				ChunkMetadata:        metadataJSON,
				Embedding:            embeddingJSON, // Full embedding or empty array
				EmbeddingModel:       s.embeddingService.config.Model,
				VectorSearchID:       vectorSearchID,
				StoredInVectorSearch: storedInVectorSearch,
			}

			if err := s.db.Create(docEmbedding).Error; err != nil {
				logrus.Errorf("Failed to store embedding for chunk %d: %v", chunkData.ChunkIndex, err)
			} else {
				logrus.Debugf("Successfully stored embedding for chunk %d of document %s", chunkData.ChunkIndex, documentID.String())
				totalStored++
			}
		}

		logrus.Infof("Stored %d embeddings for batch %d/%d", len(embeddings), (i/batchSize)+1, (len(chunks)+batchSize-1)/batchSize)
	}

	logrus.Infof("Embedding generation completed for document %s: %d embeddings stored successfully", documentID.String(), totalStored)
	return nil // Always return success - embeddings are optional
}

// SearchDocuments searches for similar content across documents
func (s *Service) SearchDocuments(agentID uuid.UUID, query string, topK int) ([]SimilarityResult, error) {
	logrus.Infof("Searching documents for agent %s with query: %s", agentID.String(), query)

	// Check how many processed documents exist for this agent
	var docCount int64
	err := s.db.Model(&database.Document{}).Where("agent_id = ? AND status = ?", agentID, "processed").Count(&docCount).Error
	if err != nil {
		logrus.Errorf("Failed to count documents: %v", err)
	} else {
		logrus.Infof("Found %d processed documents for agent %s", docCount, agentID.String())
	}

	// Generate embedding for query
	queryEmbedding, err := s.embeddingService.GenerateEmbedding(query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Use Vector Search if available, otherwise fall back to PostgreSQL
	if s.useVectorSearch && s.vectorSearch != nil {
		return s.searchWithVectorSearch(agentID, queryEmbedding.Embedding, topK)
	}

	return s.searchWithPostgreSQL(agentID, queryEmbedding.Embedding, topK)
}

// searchWithVectorSearch performs search using Google Vector Search
func (s *Service) searchWithVectorSearch(agentID uuid.UUID, queryEmbedding []float32, topK int) ([]SimilarityResult, error) {
	logrus.Infof("Using Vector Search for semantic search (agent_id: %s)", agentID.String())

	// Use restricts to filter by agent_id at Vector Search level
	filters := map[string]string{
		"agent_id": agentID.String(),
	}

	matches, err := s.vectorSearch.Search(queryEmbedding, topK, filters)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}

	logrus.Infof("Vector Search returned %d matches (filtered by agent_id)", len(matches))

	// Retrieve chunk text from database using vector search IDs
	results := make([]SimilarityResult, 0, len(matches))
	for _, match := range matches {
		// Parse document ID and chunk index from vector search ID
		docID, chunkIndex, err := ParseDatapointID(match.ID)
		if err != nil {
			logrus.Warnf("Failed to parse datapoint ID %s: %v", match.ID, err)
			continue
		}

		// Fetch chunk from database
		var embedding database.DocumentEmbedding
		err = s.db.
			Where("document_id = ? AND chunk_index = ?", docID, chunkIndex).
			First(&embedding).Error

		if err != nil {
			logrus.Warnf("Failed to fetch chunk %s from database: %v", match.ID, err)
			continue
		}

		// Get page number from embedding metadata
		pageNumber := 0
		// Could parse from embedding.ChunkMetadata if needed

		results = append(results, SimilarityResult{
			ChunkText:  embedding.ChunkText,
			ChunkIndex: embedding.ChunkIndex,
			PageNumber: pageNumber,
			Similarity: match.Similarity,
			Metadata:   map[string]interface{}{},
		})
	}

	logrus.Infof("Vector Search returned %d results", len(results))
	return results, nil
}

// searchWithPostgreSQL performs search using PostgreSQL JSONB (fallback)
func (s *Service) searchWithPostgreSQL(agentID uuid.UUID, queryEmbedding []float32, topK int) ([]SimilarityResult, error) {
	logrus.Info("Using PostgreSQL JSONB for semantic search (fallback)")

	// Check how many embeddings exist
	var embeddingCount int64
	err := s.db.Model(&database.DocumentEmbedding{}).
		Joins("JOIN documents ON documents.id = document_embeddings.document_id").
		Where("documents.agent_id = ? AND documents.status = ?", agentID, "processed").
		Count(&embeddingCount).Error
	if err != nil {
		logrus.Errorf("Failed to count embeddings: %v", err)
	} else {
		logrus.Infof("Found %d embeddings for agent %s", embeddingCount, agentID.String())
	}

	// Fetch all embeddings for the agent's documents
	var dbEmbeddings []database.DocumentEmbedding
	err = s.db.
		Joins("JOIN documents ON documents.id = document_embeddings.document_id").
		Where("documents.agent_id = ? AND documents.status = ?", agentID, "processed").
		Find(&dbEmbeddings).Error
	if err != nil {
		return nil, fmt.Errorf("failed to fetch embeddings: %w", err)
	}

	logrus.Infof("Fetched %d embeddings from database for agent %s", len(dbEmbeddings), agentID.String())

	if len(dbEmbeddings) == 0 {
		return []SimilarityResult{}, nil
	}

	// Convert to ChunkWithEmbedding format
	chunks := make([]ChunkWithEmbedding, 0, len(dbEmbeddings))
	for _, dbEmb := range dbEmbeddings {
		embedding, err := EmbeddingFromJSON(dbEmb.Embedding)
		if err != nil {
			logrus.Warnf("Failed to parse embedding: %v", err)
			continue
		}

		chunks = append(chunks, ChunkWithEmbedding{
			Text:       dbEmb.ChunkText,
			ChunkIndex: dbEmb.ChunkIndex,
			PageNumber: 0, // Can be extracted from metadata if needed
			Embedding:  embedding,
		})
	}

	// Search for similar chunks using in-memory cosine similarity
	results, err := s.embeddingService.SearchSimilarChunks(queryEmbedding, chunks, topK)
	if err != nil {
		return nil, fmt.Errorf("failed to search similar chunks: %w", err)
	}

	logrus.Infof("Found %d similar chunks for query", len(results))
	return results, nil
}

// GetDocument retrieves a document by ID
func (s *Service) GetDocument(documentID uuid.UUID) (*database.Document, error) {
	var doc database.Document
	if err := s.db.First(&doc, "id = ?", documentID).Error; err != nil {
		return nil, err
	}
	return &doc, nil
}

// GetDocumentsByAgent retrieves all documents for an agent
func (s *Service) GetDocumentsByAgent(agentID uuid.UUID) ([]database.Document, error) {
	var docs []database.Document
	if err := s.db.Where("agent_id = ?", agentID).Order("created_at DESC").Find(&docs).Error; err != nil {
		return nil, err
	}
	return docs, nil
}

// DeleteDocument deletes a document and its embeddings
func (s *Service) DeleteDocument(documentID uuid.UUID) error {
	var doc database.Document
	if err := s.db.First(&doc, "id = ?", documentID).Error; err != nil {
		return err
	}

	// Delete from GCS
	if err := s.storage.DeleteDocument(doc.StoragePath); err != nil {
		logrus.Warnf("Failed to delete document from GCS: %v", err)
	}

	// Delete from database (cascades to embeddings)
	if err := s.db.Delete(&doc).Error; err != nil {
		return fmt.Errorf("failed to delete document from database: %w", err)
	}

	logrus.Infof("Document deleted: %s", documentID.String())
	return nil
}

// GetDocumentDownloadURL generates a signed URL for downloading a document
func (s *Service) GetDocumentDownloadURL(documentID uuid.UUID, expiration time.Duration) (string, error) {
	var doc database.Document
	if err := s.db.First(&doc, "id = ?", documentID).Error; err != nil {
		return "", err
	}

	url, err := s.storage.GetSignedURL(doc.StoragePath, expiration)
	if err != nil {
		return "", fmt.Errorf("failed to generate signed URL: %w", err)
	}

	return url, nil
}

// updateDocumentStatus updates the status and error message of a document
func (s *Service) updateDocumentStatus(documentID uuid.UUID, status string, errorMessage string) {
	updateData := map[string]interface{}{
		"status": status,
	}
	if errorMessage != "" {
		updateData["error_message"] = errorMessage
	}

	if err := s.db.Model(&database.Document{}).Where("id = ?", documentID).Updates(updateData).Error; err != nil {
		logrus.Errorf("Failed to update document status: %v", err)
	}
}

// Close closes all service clients
func (s *Service) Close() error {
	var errors []error

	if err := s.storage.Close(); err != nil {
		errors = append(errors, fmt.Errorf("storage close error: %w", err))
	}

	if err := s.processor.Close(); err != nil {
		errors = append(errors, fmt.Errorf("processor close error: %w", err))
	}

	if err := s.embeddingService.Close(); err != nil {
		errors = append(errors, fmt.Errorf("embedding service close error: %w", err))
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors closing document service: %v", errors)
	}

	return nil
}
