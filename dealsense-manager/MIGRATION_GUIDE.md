# Dealsense Migration Guide: Enhanced Document Processing & Vector Search

This guide helps you migrate from the old document processing system to the new intelligent, production-grade system.

## What's New

### 1. Intelligent Document Processing
- **Automatic Strategy Selection**: System chooses sync (≤15 pages) or async batch (>15 pages)
- **No Page Limits**: Process pitch decks with 100+ pages
- **Full Image Support**: Images extracted and used in all processing modes
- **Better Error Handling**: Detailed status tracking and error messages

### 2. Google Vector Search Integration
- **50-100x Faster Search**: Milliseconds instead of seconds
- **Infinite Scalability**: Handle millions of documents
- **Lower Memory**: Only stores IDs in PostgreSQL
- **Production Ready**: Google-managed infrastructure

### 3. Resilient Chat
- **Partial Data Support**: Works with docs OR transcripts OR both
- **Better UX**: Informative messages about data availability
- **No Hard Failures**: Degrades gracefully when data is missing

## Migration Steps

### Step 1: Update Database Schema

The system will automatically migrate the database schema when you restart the server. New fields added:

**Documents Table:**
- `used_batch_processing` (boolean)
- `batch_operation_name` (string)
- `batch_output_path` (string)

**DocumentEmbeddings Table:**
- `vector_search_id` (string)
- `stored_in_vector_search` (boolean)

**Action Required:** None - GORM AutoMigrate handles this automatically.

### Step 2: Update Configuration

Copy the example configuration:
```bash
cp config.example.yaml config.yaml
```

Edit `config.yaml` with your values:

```yaml
google:
  project_id: "your-project-id"
  
  # Storage - No changes needed
  storage:
    bucket_name: "your-bucket"
    use_default_credentials: true
  
  # Document AI - No changes needed
  document_ai:
    location: "us"
    processor_id: "your-processor-id"
    use_default_credentials: true
  
  # Vertex AI - New: Vector Search section
  vertex_ai:
    location: "us-central1"
    embedding_model: "text-embedding-004"
    use_default_credentials: true
    vector_search:
      enabled: false  # Start with false, enable after setup
      index_id: ""
      index_endpoint_id: ""
```

### Step 3: (Optional) Set Up Vector Search

**Highly recommended for production, but optional.**

#### 3.1 Create Vector Search Index (STREAM_UPDATE Mode)

**Important:** Use STREAM_UPDATE mode for real-time RPC operations. This is recommended for incremental document uploads.

```bash
# Create index metadata file (STREAM_UPDATE - no contentsDeltaUri)
cat > vector-index-config.json <<EOF
{
  "config": {
    "dimensions": 768,
    "approximateNeighborsCount": 150,
    "distanceMeasureType": "DOT_PRODUCT_DISTANCE",
    "shardSize": "SHARD_SIZE_SMALL",
    "algorithm_config": {
      "treeAhConfig": {
        "leafNodeEmbeddingCount": 500,
        "leafNodesToSearchPercent": 7
      }
    }
  }
}
EOF

# Create the index
gcloud ai indexes create \
  --display-name="dealsense-embeddings" \
  --description="Document embeddings for Dealsense RAG (STREAM_UPDATE)" \
  --metadata-file=vector-index-config.json \
  --region=us-central1 \
  --project=your-project-number
```

**Note the INDEX_ID** from the output.

**Why STREAM_UPDATE?**
- Real-time updates via RPC (UpsertDatapoints, RemoveDatapoints)
- No GCS bucket required for updates
- Perfect for incremental document ingestion
- Simpler architecture

**Alternative - BATCH_UPDATE Mode:**
If you need bulk imports from GCS, add `"contentsDeltaUri": "gs://your-bucket/vector-data"` to the config. However, STREAM_UPDATE is recommended for this application.

#### 3.2 Create Index Endpoint

```bash
gcloud ai index-endpoints create \
  --display-name="dealsense-endpoint" \
  --region=us-central1 \
  --project=your-project-number
```

**Note the ENDPOINT_ID** from the output.

#### 3.3 Deploy Index to Endpoint

```bash
gcloud ai index-endpoints deploy-index ENDPOINT_ID \
  --deployed-index-id=dealsense_deployed \
  --display-name=dealsense_deployed \
  --index=INDEX_ID \
  --region=us-central1 \
  --project=your-project-number
```

This takes 30-60 minutes.

**Note:** If you're switching from an existing BATCH_UPDATE index to STREAM_UPDATE, you'll need to:
1. Create the new STREAM index (Step 3.1)
2. Undeploy the old index: `gcloud ai index-endpoints undeploy-index ENDPOINT_ID --deployed-index-id=OLD_DEPLOYED_ID --region=us-central1 --project=your-project-number`
3. Deploy the new STREAM index (this step)
4. Update config.yaml with the new index_id

#### 3.4 Update Configuration

```yaml
google:
  project_id: "your-project-id"  # e.g., "genai-exchange-475318"
  
  vertex_ai:
    location: "us-central1"
    vector_search:
      enabled: true
      project_number: "your-project-number"  # e.g., "33593473489" (REQUIRED for Vector Search)
      index_id: "YOUR_INDEX_ID"
      index_endpoint_id: "YOUR_ENDPOINT_ID"
```

**Important:** Use your project **number** (numeric), not project ID. Find it with:
```bash
gcloud projects describe YOUR_PROJECT_ID --format="value(projectNumber)"
```

### Step 4: Migrate Existing Embeddings (Optional)

If you have existing embeddings in PostgreSQL JSONB:

```bash
# Run migration script (creates datapoints in Vector Search)
go run scripts/migrate_embeddings_to_vector_search.go
```

**Note:** This is optional. New documents will use Vector Search automatically. Old documents work fine with PostgreSQL JSONB.

### Step 5: Test the System

#### Test 1: Small Document (Sync Processing)

```bash
# Upload a document ≤15 pages
curl -X POST http://localhost:8001/agents/{agent_id}/documents \
  -F "file=@small-deck.pdf" \
  -H "Content-Type: multipart/form-data"

# Response includes document ID
# Status should be "processing" then quickly become "processed"
```

#### Test 2: Large Document (Batch Processing)

```bash
# Upload a document >15 pages
curl -X POST http://localhost:8001/agents/{agent_id}/documents \
  -F "file=@large-deck.pdf" \
  -H "Content-Type: multipart/form-data"

# Get document ID from response
# Check status
curl http://localhost:8001/documents/{document_id}/status

# Response:
{
  "status": "processing_batch",
  "used_batch_processing": true,
  "estimated_completion_minutes": 12
}

# Poll every 30 seconds until status is "processed"
```

#### Test 3: Chat with Partial Data

```bash
# Test with only documents (no transcripts)
curl -X POST http://localhost:8001/agents/{agent_id}/chat \
  -H "Content-Type: application/json" \
  -d '{
    "query": "What is the startup'\''s revenue model?",
    "session_id": "test-session",
    "top_k": 5
  }'

# Should work and note that transcripts are not available
```

#### Test 4: Vector Search Performance

```bash
# Search documents (uses Vector Search if enabled)
curl -X POST http://localhost:8001/agents/{agent_id}/documents/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "revenue projections",
    "top_k": 10
  }'

# Check logs for:
# "Using Vector Search for semantic search" (if enabled)
# or
# "Using PostgreSQL JSONB for semantic search (fallback)" (if disabled)
```

## Rollback Plan

If you need to rollback:

1. **Disable Vector Search:**
```yaml
vector_search:
  enabled: false
```

2. **Restart Server:** System automatically uses PostgreSQL JSONB

3. **Data Safety:** All embeddings are still in PostgreSQL, no data loss

## Monitoring

### Key Metrics to Track

1. **Document Processing**
   - Sync success rate: Should be >99%
   - Batch success rate: Should be >98%
   - Average batch time: 10-25 minutes

2. **Search Performance**
   - Vector Search latency: 50-150ms
   - PostgreSQL JSONB latency: 2-10 seconds
   - Query success rate: >99%

3. **Chat Quality**
   - Response time: <3 seconds
   - Context relevance: Monitor user feedback
   - Partial data handling: Check logs for warnings

### Log Monitoring

```bash
# Watch for errors
tail -f logs/dealsense.log | grep ERROR

# Watch batch processing
tail -f logs/dealsense.log | grep "Batch processing"

# Watch Vector Search usage
tail -f logs/dealsense.log | grep "Vector Search"
```

### Health Checks

```bash
# Overall health
curl http://localhost:8001/health

# Document processing health (check for stuck documents)
curl http://localhost:8001/agents/{agent_id}/documents | jq '.[] | select(.status != "processed")'

# Vector Search health (check initialization)
# Should see in logs at startup:
# "Vector Search enabled for low-latency semantic search"
```

## Troubleshooting

### Issue: Large documents failing

**Symptom:** Documents >15 pages show status "failed"

**Solution:**
1. Check batch processing is working:
```bash
curl http://localhost:8001/documents/{document_id}/status
```

2. Verify GCS permissions:
```bash
gsutil ls gs://your-bucket/batch_output/
```

3. Check Document AI quota:
```bash
gcloud ai quota list --filter="metric:documentai.googleapis.com"
```

### Issue: Vector Search not working

**Symptom:** Search is slow, logs show PostgreSQL usage

**Solution:**
1. Check config:
```yaml
vector_search:
  enabled: true  # Must be true
  index_id: "..."  # Must be set
  index_endpoint_id: "..."  # Must be set
```

2. Verify index is deployed:
```bash
gcloud ai index-endpoints describe ENDPOINT_ID --region=us-central1
```

3. Check service account permissions:
   - Vertex AI User
   - AI Platform Developer

### Issue: Chat returns "no context"

**Symptom:** Chat says no context available even with documents

**Solution:**
1. Check document status:
```bash
curl http://localhost:8001/documents/{document_id}/status
# Should show "processed"
```

2. Verify embeddings generated:
```sql
SELECT COUNT(*) FROM document_embeddings WHERE document_id = '{document_id}';
-- Should be > 0
```

3. Check search works:
```bash
curl -X POST http://localhost:8001/agents/{agent_id}/documents/search \
  -H "Content-Type: application/json" \
  -d '{"query": "test", "top_k": 5}'
# Should return results
```

## Performance Comparison

### Before Migration

| Metric | Value |
|--------|-------|
| Max document size | 15 pages |
| Search latency | 2-10 seconds |
| Chat with missing data | Hard failure |
| Scalability | ~1,000 docs |

### After Migration

| Metric | Value |
|--------|-------|
| Max document size | 500+ pages |
| Search latency | 50-150ms |
| Chat with missing data | Graceful degradation |
| Scalability | Millions of docs |

## FAQ

**Q: Do I need Vector Search?**
A: No, it's optional. But highly recommended for production (50-100x faster).

**Q: Can I migrate gradually?**
A: Yes. Old docs use PostgreSQL, new docs use Vector Search. No breaking changes.

**Q: What happens if Vector Search fails?**
A: Automatic fallback to PostgreSQL JSONB. No data loss, just slower search.

**Q: How much does Vector Search cost?**
A: ~$0.75 per 1M queries + $0.30/hour per node. Usually <$50/month for typical usage.

**Q: Do I need to reprocess existing documents?**
A: No. They work as-is. But reprocessing with batch API may extract better image data.

**Q: Can I test without Vector Search first?**
A: Yes. Set `vector_search.enabled: false` and everything works with PostgreSQL.

## Support

- **Documentation:** See `CONFIG_DOCUMENTATION.md`
- **Issues:** Check logs in `logs/dealsense.log`
- **Debug mode:** Set `logging.level: "debug"` in config

