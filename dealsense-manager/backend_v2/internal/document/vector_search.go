package document

import (
	"context"
	"fmt"
	"time"

	aiplatform "cloud.google.com/go/aiplatform/apiv1"
	"cloud.google.com/go/aiplatform/apiv1/aiplatformpb"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/option"
)

// VectorSearchConfig holds Google Vertex AI Vector Search configuration
type VectorSearchConfig struct {
	ProjectID             string // Project ID (e.g., "genai-exchange-475318")
	ProjectNumber         string // Optional: Project number (e.g., "33593473489"). If set, overrides ProjectID for Vector Search paths.
	Location              string // e.g., "us-central1"
	IndexID               string // The deployed index ID (just the numeric ID)
	IndexEndpointID       string // The index endpoint ID (just the numeric ID)
	DeployedIndexID       string // The deployed index ID used when deploying to endpoint (e.g., "dealsense_deployed")
	PublicEndpointDomain  string // Optional: Public endpoint domain for queries (e.g., "266063970.us-central1-33593473489.vdb.vertexai.goog")
	CredentialsJSON       string
	UseDefaultCreds       bool
	Enabled               bool
}

// VectorSearchService handles vector search operations using Google Vertex AI Vector Search
type VectorSearchService struct {
	indexClient    *aiplatform.IndexClient
	matchClient    *aiplatform.MatchClient
	config         VectorSearchConfig
	indexEndpoint  string
	ctx            context.Context
}

// VectorSearchDatapoint represents a vector datapoint for indexing
type VectorSearchDatapoint struct {
	ID        string                 `json:"id"`
	Embedding []float32              `json:"embedding"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// VectorSearchMatch represents a search result
type VectorSearchMatch struct {
	ID         string                 `json:"id"`
	Distance   float32                `json:"distance"`   // Lower is more similar
	Similarity float32                `json:"similarity"` // Converted to 0-1 scale
	Metadata   map[string]interface{} `json:"metadata"`
}

// getProjectIdentifier returns the project number if set, otherwise returns project ID
func (config VectorSearchConfig) getProjectIdentifier() string {
	if config.ProjectNumber != "" {
		return config.ProjectNumber
	}
	return config.ProjectID
}

// NewVectorSearchService creates a new Vector Search service
func NewVectorSearchService(config VectorSearchConfig) (*VectorSearchService, error) {
	ctx := context.Background()

	var indexClient *aiplatform.IndexClient
	var matchClient *aiplatform.MatchClient
	var err error

	// CRITICAL: Different endpoints for different operations
	// - IndexClient (upsert/remove): uses regional aiplatform endpoint
	// - MatchClient (search/query): uses public VDB endpoint
	
	// Regional endpoint for index management operations
	managementEndpoint := fmt.Sprintf("%s-aiplatform.googleapis.com:443", config.Location)
	
	// Public endpoint for query operations
	queryEndpoint := config.PublicEndpointDomain
	if queryEndpoint != "" {
		queryEndpoint = fmt.Sprintf("%s:443", queryEndpoint)
	} else {
		// Fallback to regional endpoint (may not work for queries)
		queryEndpoint = managementEndpoint
		logrus.Warn("PublicEndpointDomain not configured, using regional endpoint for queries (may cause errors)")
	}
	
	if config.UseDefaultCreds {
		indexClient, err = aiplatform.NewIndexClient(ctx, option.WithEndpoint(managementEndpoint))
		if err != nil {
			return nil, fmt.Errorf("failed to create index client: %w", err)
		}

		matchClient, err = aiplatform.NewMatchClient(ctx, option.WithEndpoint(queryEndpoint))
		if err != nil {
			indexClient.Close()
			return nil, fmt.Errorf("failed to create match client: %w", err)
		}
	} else if config.CredentialsJSON != "" {
		indexClient, err = aiplatform.NewIndexClient(ctx, 
			option.WithCredentialsJSON([]byte(config.CredentialsJSON)),
			option.WithEndpoint(managementEndpoint))
		if err != nil {
			return nil, fmt.Errorf("failed to create index client: %w", err)
		}

		matchClient, err = aiplatform.NewMatchClient(ctx, 
			option.WithCredentialsJSON([]byte(config.CredentialsJSON)),
			option.WithEndpoint(queryEndpoint))
		if err != nil {
			indexClient.Close()
			return nil, fmt.Errorf("failed to create match client: %w", err)
		}
	} else {
		return nil, fmt.Errorf("no credentials provided for Vector Search")
	}

	logrus.Infof("Vector Search IndexClient endpoint: %s", managementEndpoint)
	logrus.Infof("Vector Search MatchClient endpoint: %s", queryEndpoint)

	// Construct the index endpoint path using project number if available
	projectIdentifier := config.getProjectIdentifier()
	indexEndpoint := fmt.Sprintf("projects/%s/locations/%s/indexEndpoints/%s",
		projectIdentifier, config.Location, config.IndexEndpointID)

	logrus.Infof("Vector Search Service initialized: %s", indexEndpoint)

	return &VectorSearchService{
		indexClient:   indexClient,
		matchClient:   matchClient,
		config:        config,
		indexEndpoint: indexEndpoint,
		ctx:           ctx,
	}, nil
}

// UpsertDatapoints upserts (inserts or updates) datapoints in the vector index
func (v *VectorSearchService) UpsertDatapoints(datapoints []VectorSearchDatapoint) error {
	if len(datapoints) == 0 {
		return nil
	}

	logrus.Infof("Upserting %d datapoints to vector index", len(datapoints))

	projectIdentifier := v.config.getProjectIdentifier()
	indexPath := fmt.Sprintf("projects/%s/locations/%s/indexes/%s",
		projectIdentifier, v.config.Location, v.config.IndexID)

	logrus.Debugf("Using index path for upsert: %s", indexPath)

	// Convert datapoints to protobuf format
	protoDatapoints := make([]*aiplatformpb.IndexDatapoint, 0, len(datapoints))
	for _, dp := range datapoints {
		// Convert embedding to float values
		featureVector := make([]float32, len(dp.Embedding))
		copy(featureVector, dp.Embedding)

		// Convert metadata to Restricts for filtering
		// This allows us to filter by agent_id and other metadata at query time
		var restricts []*aiplatformpb.IndexDatapoint_Restriction
		for namespace, value := range dp.Metadata {
			// Convert value to string
			valueStr, ok := value.(string)
			if !ok {
				valueStr = fmt.Sprintf("%v", value)
			}
			restricts = append(restricts, &aiplatformpb.IndexDatapoint_Restriction{
				Namespace: namespace,
				AllowList: []string{valueStr},
			})
		}
		
		protoDatapoints = append(protoDatapoints, &aiplatformpb.IndexDatapoint{
			DatapointId:   dp.ID,
			FeatureVector: featureVector,
			Restricts:     restricts,
		})

		logrus.Debugf("Prepared datapoint: ID=%s, Vector dim=%d, Restricts: %v", 
			dp.ID, len(featureVector), dp.Metadata)
	}

	// Upsert datapoints using the Index API
	// For STREAM_UPDATE indexes, the path should be: projects/.../indexes/INDEX_ID
	req := &aiplatformpb.UpsertDatapointsRequest{
		Index:      indexPath,
		Datapoints: protoDatapoints,
	}

	logrus.Debugf("Sending UpsertDatapointsRequest with %d datapoints to: %s", len(protoDatapoints), indexPath)

	resp, err := v.indexClient.UpsertDatapoints(v.ctx, req)
	if err != nil {
		logrus.Errorf("Upsert failed - Index path: %s, Error: %v", indexPath, err)
		return fmt.Errorf("failed to upsert datapoints to path %s: %w", indexPath, err)
	}

	logrus.Infof("Successfully upserted %d datapoints, response: %+v", len(protoDatapoints), resp)
	return nil
}

// Search searches for similar vectors in the index
func (v *VectorSearchService) Search(queryEmbedding []float32, topK int, filters map[string]string) ([]VectorSearchMatch, error) {
	if topK <= 0 {
		topK = 10
	}

	if len(filters) > 0 {
		logrus.Infof("Searching vector index with topK=%d, filters=%v", topK, filters)
	} else {
		logrus.Infof("Searching vector index with topK=%d (no filters)", topK)
	}

	// Convert filters to Restricts for Vector Search filtering
	var restricts []*aiplatformpb.IndexDatapoint_Restriction
	for namespace, value := range filters {
		restricts = append(restricts, &aiplatformpb.IndexDatapoint_Restriction{
			Namespace: namespace,
			AllowList: []string{value},
		})
		logrus.Debugf("Adding restrict filter: %s=%s", namespace, value)
	}

	// Build query with restricts
	queries := []*aiplatformpb.FindNeighborsRequest_Query{
		{
			Datapoint: &aiplatformpb.IndexDatapoint{
				DatapointId:   fmt.Sprintf("query_%d", time.Now().UnixNano()),
				FeatureVector: queryEmbedding,
				Restricts:     restricts,
			},
			NeighborCount: int32(topK),
		},
	}

	// Create find neighbors request
	// For STREAM_UPDATE, we need to specify the deployed index ID
	deployedIndexID := v.config.DeployedIndexID
	if deployedIndexID == "" {
		deployedIndexID = "dealsense_deployed" // Default fallback
	}
	
	req := &aiplatformpb.FindNeighborsRequest{
		IndexEndpoint:   v.indexEndpoint,
		DeployedIndexId: deployedIndexID,
		Queries:         queries,
	}
	
	logrus.Debugf("Querying deployed index: %s on endpoint: %s", deployedIndexID, v.indexEndpoint)

	// Execute search
	resp, err := v.matchClient.FindNeighbors(v.ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to search vector index: %w", err)
	}

	// Parse results
	if len(resp.NearestNeighbors) == 0 {
		logrus.Warn("No neighbors found in search results")
		return []VectorSearchMatch{}, nil
	}

	neighbors := resp.NearestNeighbors[0].Neighbors
	matches := make([]VectorSearchMatch, 0, len(neighbors))

	for _, neighbor := range neighbors {
		// Convert distance to similarity (assuming cosine distance)
		// For cosine distance: similarity = 1 - distance/2
		// Distance range is typically [0, 2] for cosine distance
		distance := float32(neighbor.Distance)
		similarity := float32(1.0) - (distance / 2.0)
		if similarity < 0 {
			similarity = 0
		}
		if similarity > 1 {
			similarity = 1
		}

		match := VectorSearchMatch{
			ID:         neighbor.Datapoint.DatapointId,
			Distance:   distance,
			Similarity: similarity,
			Metadata:   make(map[string]interface{}), // Empty - no metadata from STREAM_UPDATE
		}

		// NOTE: STREAM_UPDATE doesn't support Restricts, so no metadata is returned
		// Filtering must happen at the service layer using database queries
		matches = append(matches, match)
	}

	logrus.Infof("Vector Search returned %d matches", len(matches))
	
	// Note about filters: STREAM_UPDATE doesn't support metadata/restricts
	// The filters parameter is accepted for API compatibility but filtering
	// must be done at the service layer (service.go) using database queries
	if len(filters) > 0 {
		logrus.Debugf("Note: Filters %v will be applied at service layer via database", filters)
	}
	
	return matches, nil
}

// BatchUpsert upserts datapoints in batches to handle large datasets efficiently
func (v *VectorSearchService) BatchUpsert(datapoints []VectorSearchDatapoint, batchSize int) error {
	if batchSize <= 0 {
		batchSize = 100 // Default batch size
	}

	totalBatches := (len(datapoints) + batchSize - 1) / batchSize
	logrus.Infof("Upserting %d datapoints in %d batches", len(datapoints), totalBatches)

	for i := 0; i < len(datapoints); i += batchSize {
		end := i + batchSize
		if end > len(datapoints) {
			end = len(datapoints)
		}

		batch := datapoints[i:end]
		logrus.Infof("Upserting batch %d/%d (%d datapoints)", (i/batchSize)+1, totalBatches, len(batch))

		if err := v.UpsertDatapoints(batch); err != nil {
			return fmt.Errorf("failed to upsert batch %d: %w", (i/batchSize)+1, err)
		}

		// Small delay between batches to avoid rate limiting
		if i+batchSize < len(datapoints) {
			time.Sleep(100 * time.Millisecond)
		}
	}

	logrus.Info("Batch upsert completed successfully")
	return nil
}

// RemoveDatapoints removes datapoints from the index
func (v *VectorSearchService) RemoveDatapoints(datapointIDs []string) error {
	if len(datapointIDs) == 0 {
		return nil
	}

	logrus.Infof("Removing %d datapoints from vector index", len(datapointIDs))

	projectIdentifier := v.config.getProjectIdentifier()
	indexPath := fmt.Sprintf("projects/%s/locations/%s/indexes/%s",
		projectIdentifier, v.config.Location, v.config.IndexID)

	req := &aiplatformpb.RemoveDatapointsRequest{
		Index:        indexPath,
		DatapointIds: datapointIDs,
	}

	resp, err := v.indexClient.RemoveDatapoints(v.ctx, req)
	if err != nil {
		return fmt.Errorf("failed to remove datapoints: %w", err)
	}

	logrus.Infof("Successfully removed datapoints, response: %+v", resp)
	return nil
}

// GetIndexInfo retrieves information about the vector index
func (v *VectorSearchService) GetIndexInfo() (map[string]interface{}, error) {
	projectIdentifier := v.config.getProjectIdentifier()
	indexPath := fmt.Sprintf("projects/%s/locations/%s/indexes/%s",
		projectIdentifier, v.config.Location, v.config.IndexID)

	req := &aiplatformpb.GetIndexRequest{
		Name: indexPath,
	}

	index, err := v.indexClient.GetIndex(v.ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get index info: %w", err)
	}

	info := map[string]interface{}{
		"name":                   index.Name,
		"display_name":           index.DisplayName,
		"description":            index.Description,
		"deployed_indexes_count": len(index.DeployedIndexes),
		"index_stats":            index.IndexStats,
	}

	return info, nil
}

// CreateDatapointID creates a unique datapoint ID for a document chunk
func CreateDatapointID(documentID uuid.UUID, chunkIndex int) string {
	return fmt.Sprintf("%s_chunk_%d", documentID.String(), chunkIndex)
}

// ParseDatapointID parses a datapoint ID to extract document ID and chunk index
func ParseDatapointID(datapointID string) (uuid.UUID, int, error) {
	var docID uuid.UUID
	var chunkIndex int

	// Format: {documentID}_chunk_{chunkIndex}
	n, err := fmt.Sscanf(datapointID, "%s_chunk_%d", &docID, &chunkIndex)
	if err != nil || n != 2 {
		return uuid.Nil, 0, fmt.Errorf("invalid datapoint ID format: %s", datapointID)
	}

	return docID, chunkIndex, nil
}

// Close closes all clients
func (v *VectorSearchService) Close() error {
	var errors []error

	if v.indexClient != nil {
		if err := v.indexClient.Close(); err != nil {
			errors = append(errors, fmt.Errorf("index client close error: %w", err))
		}
	}

	if v.matchClient != nil {
		if err := v.matchClient.Close(); err != nil {
			errors = append(errors, fmt.Errorf("match client close error: %w", err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors closing vector search service: %v", errors)
	}

	return nil
}

// HealthCheck verifies the vector search service is operational
func (v *VectorSearchService) HealthCheck() error {
	_, err := v.GetIndexInfo()
	if err != nil {
		return fmt.Errorf("vector search health check failed: %w", err)
	}
	return nil
}

