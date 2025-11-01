# Vector Search Troubleshooting Guide

## Common Issues and Solutions

### Issue 1: Empty String in JSONB Field

**Symptom:** Errors when uploading documents with Vector Search enabled

**Root Cause:** 
When Vector Search is enabled, the code was storing an empty string `""` in the `Embedding` JSONB field, which causes PostgreSQL errors because JSONB expects valid JSON.

**Fix Applied:**
Changed to store empty JSON array `"[]"` instead of empty string when using Vector Search:

```go
// Before (WRONG)
var embeddingJSON string  // Empty string = ""
if !s.useVectorSearch {
    embeddingJSON, err = EmbeddingToJSON(embResult.Embedding)
}

// After (CORRECT)
var embeddingJSON string
if s.useVectorSearch {
    embeddingJSON = "[]"  // Valid empty JSON array
} else {
    embeddingJSON, err = EmbeddingToJSON(embResult.Embedding)
}
```

---

### Issue 2: Wrong Project Identifier (Project ID vs Project Number)

**Symptom:** 
- Vector Search initialization fails
- Errors mentioning resource not found or permission denied
- 404 errors when accessing Vector Search endpoints

**Root Cause:** 
Google Vector Search requires the **project number** (e.g., "33593473489"), not the project ID (e.g., "genai-exchange-475318").

**How to Identify:**
Vector Search resource paths look like:
```
projects/33593473489/locations/us-central1/indexes/3413880250151469056
projects/33593473489/locations/us-central1/indexEndpoints/6674108148367753216
```

The first number (`33593473489`) is your **project number**, not your project ID.

**Fix:**
1. Find your project number:
```bash
gcloud projects describe YOUR_PROJECT_ID --format="value(projectNumber)"
```

2. Add it to your `config.yaml`:
```yaml
google:
  project_id: "genai-exchange-475318"  # Still needed for other GCP services
  
  vertex_ai:
    vector_search:
      enabled: true
      project_number: "33593473489"  # Add this field with your project number
      index_id: "3413880250151469056"
      index_endpoint_id: "6674108148367753216"
```

**Note:** If `project_number` is not specified, the system will fall back to using `project_id`, which may work in some cases but is not recommended for Vector Search.

---

## Diagnostic Steps

### Step 1: Check if Vector Search is Actually Enabled

Look for this log message at startup:
```
INFO Vector Search enabled for low-latency semantic search
```

If you see this instead, Vector Search is disabled:
```
WARN Failed to initialize Vector Search: ... Falling back to PostgreSQL JSONB
```

### Step 2: Verify Configuration

Check your `config.yaml`:
```yaml
google:
  project_id: "genai-exchange-475318"  # Your GCP project ID

vertex_ai:
  location: "us-central1"  # Must match your Vector Search location
  vector_search:
    enabled: true  # Must be true
    project_number: "33593473489"  # IMPORTANT: Use project number, not project ID
    index_id: "3413880250151469056"  # Just the numeric ID
    index_endpoint_id: "6674108148367753216"  # Just the numeric ID
```

**Important:** Vector Search requires the **project number** (numeric), not the project ID (alphanumeric with dashes). 

To find your project number:
```bash
gcloud projects describe YOUR_PROJECT_ID --format="value(projectNumber)"
```

Or look at your Vector Search resource paths:
- Full path: `projects/33593473489/locations/us-central1/indexes/3413880250151469056`
- Project number: `33593473489`
- Index ID: `3413880250151469056`

### Step 3: Check Logs During Document Upload

**Successful Vector Search Storage:**
```
DEBUG Attempting to store chunk 0 in Vector Search with ID: {uuid}_chunk_0
DEBUG Successfully stored embedding in Vector Search for chunk 0
DEBUG Successfully stored embedding for chunk 0 of document {uuid}
```

**Vector Search Failure (falls back to PostgreSQL):**
```
ERROR Failed to store embedding in Vector Search for chunk 0: <error details>
DEBUG Successfully stored embedding for chunk 0 of document {uuid}
```

**Vector Search Not Initialized:**
```
WARN Vector Search is enabled but service is nil - using PostgreSQL fallback
```

### Step 4: Check Database Records

After document processing, check the database:

```sql
SELECT 
    id,
    chunk_index,
    vector_search_id,
    stored_in_vector_search,
    LENGTH(embedding) as embedding_size
FROM document_embeddings 
WHERE document_id = '<your-doc-id>'
LIMIT 5;
```

**Expected Results:**

**With Vector Search:**
```
vector_search_id: {uuid}_chunk_0
stored_in_vector_search: true
embedding_size: 2  (just "[]")
```

**Without Vector Search (fallback):**
```
vector_search_id: (null)
stored_in_vector_search: false
embedding_size: >1000  (full embedding JSON)
```

---

## Common Error Messages

### Error: "invalid input syntax for type json"

**Cause:** Empty string `""` being stored in JSONB field

**Solution:** Fixed in latest version - now stores `"[]"`

### Error: "Vector Search is enabled but service is nil"

**Causes:**
1. Index ID or Endpoint ID not configured
2. Invalid credentials
3. Index not deployed

**Solutions:**
1. Verify config.yaml has correct IDs
2. Check `GOOGLE_APPLICATION_CREDENTIALS` environment variable
3. Verify index is deployed:
```bash
gcloud ai index-endpoints describe YOUR_ENDPOINT_ID --region=us-central1
```

### Error: "failed to search vector index: rpc error"

**Causes:**
1. Index endpoint not accessible
2. Service account lacks permissions
3. Index not ready

**Solutions:**
1. Check network connectivity to Google Cloud
2. Verify service account has "Vertex AI User" role
3. Wait for index deployment to complete (can take 30-60 minutes)

---

## Testing Vector Search

### Test 1: Upload a Small Document

```bash
curl -X POST http://localhost:8001/agents/{agent_id}/documents \
  -F "file=@test.pdf"
```

Check logs for:
- "Attempting to store chunk X in Vector Search"
- "Successfully stored embedding in Vector Search"

### Test 2: Search Documents

```bash
curl -X POST http://localhost:8001/agents/{agent_id}/documents/search \
  -H "Content-Type: application/json" \
  -d '{"query": "test query", "top_k": 5}'
```

Check logs for:
```
INFO Using Vector Search for semantic search
INFO Vector Search returned X results
```

If you see this, Vector Search is NOT being used:
```
INFO Using PostgreSQL JSONB for semantic search (fallback)
```

### Test 3: Performance Check

**Vector Search should be fast:**
- Search latency: 50-150ms
- Log message: "Using Vector Search for semantic search"

**PostgreSQL is slower:**
- Search latency: 2-10 seconds
- Log message: "Using PostgreSQL JSONB for semantic search (fallback)"

---

## Fallback Behavior

The system is designed to gracefully degrade:

1. **Vector Search Unavailable at Startup:**
   - Logs warning
   - Uses PostgreSQL JSONB for all operations
   - No errors, just slower search

2. **Vector Search Fails During Upload:**
   - Logs error for specific chunk
   - Stores full embedding in PostgreSQL
   - Document processing continues
   - Search will use PostgreSQL for that document

3. **Vector Search Fails During Search:**
   - Logs error
   - Automatically falls back to PostgreSQL
   - Returns results (slower but works)

---

## Monitoring Vector Search Health

### Add Health Check Endpoint

The Vector Search service has a `HealthCheck()` method:

```go
if vectorSearch != nil {
    if err := vectorSearch.HealthCheck(); err != nil {
        logrus.Errorf("Vector Search health check failed: %v", err)
    }
}
```

### Key Metrics to Monitor

1. **Upsert Success Rate:**
   - Count of "Successfully stored embedding in Vector Search"
   - Should be >99%

2. **Search Latency:**
   - Vector Search: <150ms
   - PostgreSQL fallback: >2s

3. **Fallback Rate:**
   - Count of "Using PostgreSQL JSONB for semantic search (fallback)"
   - Should be 0% if Vector Search is working

---

## Best Practices

### 1. Always Enable Logging

Set log level to DEBUG during initial setup:
```yaml
logging:
  level: "debug"
```

### 2. Test Without Vector Search First

Start with `vector_search.enabled: false` to verify:
- Document processing works
- Embeddings are generated
- Search works (even if slow)

Then enable Vector Search incrementally.

### 3. Monitor Database Size

With Vector Search, the `document_embeddings` table should be small:
- Only metadata and IDs stored
- `embedding` field is just `"[]"`

Without Vector Search:
- Full embeddings stored
- Table can be 50-100x larger

### 4. Set Up Alerts

Alert on:
- Vector Search errors in logs
- Search latency >500ms
- Fallback rate >5%

---

## Quick Fixes

### Fix 1: Restart with Correct Config

```bash
# Verify config
cat config.yaml | grep -A 5 "vector_search"

# Restart server
./server
```

### Fix 2: Reprocess Failed Documents

If documents were processed before fix:

```sql
-- Find documents with empty embedding strings
SELECT d.id, d.name 
FROM documents d
JOIN document_embeddings de ON d.id = de.document_id
WHERE de.embedding = '' OR de.embedding IS NULL;

-- Delete and reupload these documents
```

### Fix 3: Disable Vector Search Temporarily

If Vector Search is causing issues:

```yaml
vertex_ai:
  vector_search:
    enabled: false  # Temporarily disable
```

System will use PostgreSQL fallback (slower but reliable).

---

## Support

If you encounter persistent issues:

1. **Collect logs:**
```bash
tail -n 1000 logs/dealsense.log > issue-logs.txt
```

2. **Check database state:**
```sql
SELECT COUNT(*), 
       stored_in_vector_search,
       AVG(LENGTH(embedding)) as avg_embedding_size
FROM document_embeddings
GROUP BY stored_in_vector_search;
```

3. **Verify Vector Search index:**
```bash
gcloud ai indexes describe YOUR_INDEX_ID --region=us-central1
```

---

## Summary of Latest Fix

**Problem:** Empty string in JSONB field caused database errors

**Solution:** 
- Store `"[]"` instead of `""` when using Vector Search
- Added better error logging
- Improved diagnostic messages

**Status:** ✅ Fixed and tested

**Compatibility:** Backward compatible - old documents still work

