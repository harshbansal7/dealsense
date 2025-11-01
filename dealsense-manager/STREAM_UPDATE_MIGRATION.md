# Migration to STREAM_UPDATE Vector Search Index

## Quick Summary

You need to create a **new** Vector Search index with STREAM_UPDATE mode to support real-time RPC operations. The existing BATCH_UPDATE index cannot be converted.

## Why STREAM_UPDATE?

| Feature | BATCH_UPDATE (Old) | STREAM_UPDATE (New) |
|---------|-------------------|---------------------|
| Update Method | Upload files to GCS | Direct RPC calls |
| Latency | Minutes (batch job) | Instant |
| GCS Requirement | Yes (contentsDeltaUri) | No |
| Use Case | Bulk imports | Real-time ingestion |
| Your Code | Already using RPC! ✅ | Perfect match ✅ |

**Your code already uses `UpsertDatapoints()` RPC**, so STREAM_UPDATE is the right choice!

## Migration Steps

### Option 1: Minimal Downtime (Recommended)

#### Step 1: Create New STREAM Index

```bash
# Create the index config (NO contentsDeltaUri = STREAM mode)
cat > vector-index-stream-config.json <<EOF
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
  --display-name="dealsense-embeddings-stream" \
  --description="Document embeddings for Dealsense RAG (STREAM_UPDATE)" \
  --metadata-file=vector-index-stream-config.json \
  --region=us-central1 \
  --project=33593473489

# Wait 5-10 minutes for creation
# Note the NEW_INDEX_ID from output
```

#### Step 2: Undeploy Old Index from Endpoint

```bash
gcloud ai index-endpoints undeploy-index 6674108148367753216 \
  --deployed-index-id=dealsense_deployed \
  --region=us-central1 \
  --project=33593473489

# This takes ~10-15 minutes
```

#### Step 3: Deploy New STREAM Index to Endpoint

```bash
gcloud ai index-endpoints deploy-index 6674108148367753216 \
  --deployed-index-id=dealsense_deployed \
  --display-name=dealsense_deployed \
  --index=NEW_INDEX_ID \
  --region=us-central1 \
  --project=33593473489

# This takes 30-60 minutes
```

#### Step 4: Update Your Config

Edit your `config.yaml`:

```yaml
google:
  project_id: "genai-exchange-475318"
  
  vertex_ai:
    location: "us-central1"
    vector_search:
      enabled: true
      project_number: "33593473489"
      index_id: "NEW_INDEX_ID"  # ← Update this!
      index_endpoint_id: "6674108148367753216"  # Same endpoint
```

#### Step 5: Restart Server

```bash
# Your server will now use STREAM_UPDATE mode
./server
```

#### Step 6: Test Upload

```bash
# Upload a document
curl -X POST http://localhost:8001/agents/{agent_id}/documents \
  -F "file=@test.pdf"

# Check logs - should see instant upsert:
# "Upserting X datapoints to vector index"
# "Successfully upserted datapoints"
```

#### Step 7: Clean Up Old Index (After Verification)

```bash
# After confirming everything works, delete the old BATCH index
gcloud ai indexes delete 3413880250151469056 \
  --region=us-central1 \
  --project=33593473489
```

**Total Downtime:** ~45-75 minutes (while deploying new index)

---

### Option 2: Zero Downtime (More Resources)

If you need zero downtime, create a new endpoint too:

#### Step 1: Create New STREAM Index (same as above)

#### Step 2: Create New Endpoint

```bash
gcloud ai index-endpoints create \
  --display-name="dealsense-endpoint-stream" \
  --region=us-central1 \
  --project=33593473489

# Note the NEW_ENDPOINT_ID
```

#### Step 3: Deploy New Index to New Endpoint

```bash
gcloud ai index-endpoints deploy-index NEW_ENDPOINT_ID \
  --deployed-index-id=dealsense_deployed \
  --display-name=dealsense_deployed \
  --index=NEW_INDEX_ID \
  --region=us-central1 \
  --project=33593473489

# Takes 30-60 minutes, but old system still running
```

#### Step 4: Switch Config and Restart

```yaml
vector_search:
  enabled: true
  project_number: "33593473489"
  index_id: "NEW_INDEX_ID"
  index_endpoint_id: "NEW_ENDPOINT_ID"
```

#### Step 5: Clean Up Old Resources

```bash
# Undeploy old index
gcloud ai index-endpoints undeploy-index 6674108148367753216 \
  --deployed-index-id=dealsense_deployed \
  --region=us-central1 \
  --project=33593473489

# Delete old endpoint
gcloud ai index-endpoints delete 6674108148367753216 \
  --region=us-central1 \
  --project=33593473489

# Delete old index
gcloud ai indexes delete 3413880250151469056 \
  --region=us-central1 \
  --project=33593473489
```

**Total Downtime:** ~0 minutes (but costs 2x during migration)

---

## Key Configuration Differences

### BATCH_UPDATE Index Config (Old - Don't Use)
```json
{
  "contentsDeltaUri": "gs://your-bucket/vector-data",  ← This makes it BATCH
  "config": {
    "dimensions": 768,
    ...
  }
}
```

### STREAM_UPDATE Index Config (New - Use This)
```json
{
  "config": {  ← No contentsDeltaUri = STREAM mode
    "dimensions": 768,
    "shardSize": "SHARD_SIZE_SMALL",  ← Optional but recommended
    ...
  }
}
```

## Verification Checklist

After migration, verify:

- [ ] Server starts without errors
- [ ] Logs show: `Vector Search Service initialized: projects/33593473489/...`
- [ ] Upload a test document
- [ ] Logs show instant upsert (no GCS operations)
- [ ] Search returns results
- [ ] Chat works with document context

## Troubleshooting

### Issue: "Index is not in STREAM_UPDATE mode"

**Cause:** Index was created with `contentsDeltaUri`

**Fix:** Create a new index without `contentsDeltaUri` (follow Step 1)

### Issue: "Failed to upsert datapoints"

**Possible causes:**
1. Index not fully deployed (check: `gcloud ai index-endpoints describe ENDPOINT_ID --region=us-central1`)
2. Wrong project number (use `33593473489`, not `genai-exchange-475318`)
3. Service account lacks permissions

**Check permissions:**
```bash
# Your service account needs:
# - Vertex AI User
# - AI Platform Developer
```

### Issue: Old embeddings lost

**Not a problem!** Embeddings are still in PostgreSQL. The system will:
1. Fall back to PostgreSQL for old documents (graceful)
2. New documents go to the new STREAM index
3. Optionally re-upload old documents to populate new index

## Cost Impact

### Old Setup (BATCH_UPDATE)
- Index: ~$0.30/hour
- GCS storage: ~$0.02/GB/month
- Update operations: Via GCS (included)

### New Setup (STREAM_UPDATE)
- Index: ~$0.30/hour (same)
- GCS storage: $0 (not needed for updates)
- Update operations: ~$0.001 per 1K calls

**Net change:** Slightly cheaper (no GCS for updates), much faster!

## FAQ

**Q: Can I convert my existing BATCH index to STREAM?**  
A: No, you must create a new index. This is a Google limitation.

**Q: Will I lose data?**  
A: No. Embeddings are in PostgreSQL. New index starts empty, gets populated as documents are uploaded.

**Q: Do I need to reprocess all documents?**  
A: Not required, but recommended. Old documents work with PostgreSQL fallback. Reprocess to populate new index.

**Q: How long does migration take?**  
A: ~60-90 minutes total (10 min create + 15 min undeploy + 45 min deploy)

**Q: Can I test STREAM before switching?**  
A: Yes! Use Option 2 (new endpoint). Test in parallel, then switch config when ready.

**Q: Does the code need changes?**  
A: No! Your code already uses `UpsertDatapoints()`, which is perfect for STREAM mode.

## Your Specific Commands

Based on your setup:

```bash
# 1. Create new STREAM index
gcloud ai indexes create \
  --display-name="dealsense-embeddings-stream" \
  --metadata-file=vector-index-stream-config.json \
  --region=us-central1 \
  --project=33593473489

# 2. Undeploy old index
gcloud ai index-endpoints undeploy-index 6674108148367753216 \
  --deployed-index-id=dealsense_deployed \
  --region=us-central1 \
  --project=33593473489

# 3. Deploy new index (use NEW_INDEX_ID from step 1)
gcloud ai index-endpoints deploy-index 6674108148367753216 \
  --deployed-index-id=dealsense_deployed \
  --index=NEW_INDEX_ID \
  --region=us-central1 \
  --project=33593473489

# 4. Update config.yaml with NEW_INDEX_ID

# 5. Restart server
```

## Timeline

| Step | Duration | Can Work in Parallel? |
|------|----------|-----------------------|
| Create STREAM index | 5-10 min | ✅ Yes (old system runs) |
| Undeploy old index | 10-15 min | ❌ No (downtime starts) |
| Deploy new index | 30-60 min | ❌ No (downtime continues) |
| Update config + restart | 1 min | ❌ No |
| **Total downtime** | **45-75 min** | (Option 1) |
| **Total with new endpoint** | **0 min** | (Option 2) |

## Support

- Full guide: [MIGRATION_GUIDE.md](./MIGRATION_GUIDE.md)
- Config help: [QUICK_CONFIG_REFERENCE.md](./QUICK_CONFIG_REFERENCE.md)
- Troubleshooting: [VECTOR_SEARCH_TROUBLESHOOTING.md](./VECTOR_SEARCH_TROUBLESHOOTING.md)

