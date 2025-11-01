# Vector Search Upsert Troubleshooting Guide

## Current Status

✅ **Code is using the correct path format:** `projects/.../indexes/INDEX_ID` (not indexEndpoints)  
✅ **Code is using the correct client:** `IndexClient.UpsertDatapoints()` (not IndexEndpointClient)  
✅ **Enhanced logging** has been added for better diagnostics

## Correct vs Incorrect Paths

### ✅ CORRECT (for STREAM_UPDATE upserts)
```
projects/33593473489/locations/us-central1/indexes/NEW_INDEX_ID
```

This uses:
- **indexes/** (not indexEndpoints)
- Your **INDEX_ID** (the STREAM index you created)
- `IndexClient.UpsertDatapoints()` method

### ❌ INCORRECT (common mistakes)
```
projects/33593473489/locations/us-central1/indexEndpoints/6674108148367753216
```

This would be wrong because:
- Uses `indexEndpoints/` instead of `indexes/`
- Uses endpoint ID instead of index ID
- Endpoint is for **querying**, not upserting

## Understanding the Architecture

```
Your Setup:
┌─────────────────────────────────────────────┐
│ Index (STREAM_UPDATE)                       │
│ ID: NEW_INDEX_ID                            │
│ Path: .../indexes/NEW_INDEX_ID             │
│ Used for: UpsertDatapoints, RemoveDatapoints│
└─────────────┬───────────────────────────────┘
              │ deployed to
              ↓
┌─────────────────────────────────────────────┐
│ Index Endpoint                              │
│ ID: 6674108148367753216                     │
│ Path: .../indexEndpoints/6674...           │
│ Used for: FindNeighbors (search/query)     │
└─────────────────────────────────────────────┘
```

**Key Point:** 
- **Upserts** → Send to INDEX path
- **Searches** → Send to INDEX ENDPOINT path

## Diagnostic Steps

### Step 1: Check Your Config

```yaml
vector_search:
  enabled: true
  project_number: "33593473489"
  index_id: "NEW_INDEX_ID"  # ← This is your STREAM index ID
  index_endpoint_id: "6674108148367753216"
```

**Verify:**
```bash
# The index ID should be your NEW STREAM index
gcloud ai indexes list --region=us-central1 --project=33593473489

# Look for the index with display-name "dealsense-embeddings-stream"
# Note its ID - this should match your config
```

### Step 2: Check Index Status

```bash
# Get detailed index info
gcloud ai indexes describe NEW_INDEX_ID \
  --region=us-central1 \
  --project=33593473489

# Check:
# - updateMethod: Should be "STREAM_UPDATE"
# - deployedIndexes: Should show it's deployed
```

### Step 3: Check Deployment Status

```bash
# Verify index is deployed to endpoint
gcloud ai index-endpoints describe 6674108148367753216 \
  --region=us-central1 \
  --project=33593473489

# Look for:
# - deployedIndexes[].index: Should reference your STREAM index
# - state: Should be "DEPLOYED"
```

### Step 4: Check Logs with Enhanced Logging

Set log level to debug in `config.yaml`:
```yaml
logging:
  level: "debug"
```

Restart server and upload a document. Look for:

```bash
# Check what path is being used
grep "Using index path for upsert" logs/dealsense.log

# Should show:
# DEBUG Using index path for upsert: projects/33593473489/locations/us-central1/indexes/NEW_INDEX_ID

# Check for errors
grep "Upsert failed" logs/dealsense.log

# Should show specific error if failing
```

### Step 5: Verify Permissions

Your service account needs these roles:

```bash
# Check current roles
gcloud projects get-iam-policy 33593473489 \
  --flatten="bindings[].members" \
  --filter="bindings.members:serviceAccount:YOUR_SERVICE_ACCOUNT"

# Required roles:
# - roles/aiplatform.user
# - roles/aiplatform.developer (for upserts)
```

## Common Issues and Fixes

### Issue 1: "Index not found" or 404 Error

**Symptom:** Error mentions index path not found

**Possible causes:**
1. Using old BATCH index ID instead of new STREAM index ID
2. Wrong project number
3. Index not fully created yet

**Fix:**
```bash
# Verify your STREAM index exists
gcloud ai indexes list --region=us-central1 --project=33593473489 | grep STREAM

# Update config with correct NEW_INDEX_ID
```

### Issue 2: "Method not allowed" or Permission Denied

**Symptom:** Permission errors when upserting

**Possible causes:**
1. Index is BATCH_UPDATE, not STREAM_UPDATE
2. Service account lacks permissions
3. Index not deployed

**Fix:**
```bash
# Check index update method
gcloud ai indexes describe YOUR_INDEX_ID \
  --region=us-central1 \
  --project=33593473489 \
  --format="value(metadata.config.updateMethod)"

# Should output: STREAM_UPDATE

# If it says BATCH_UPDATE or nothing, you need to create a new STREAM index
```

### Issue 3: Using IndexEndpoint path instead of Index path

**Symptom:** Your concern about using endpoint ID

**Clarification:**
- ✅ The code is **correct** - it uses `indexes/INDEX_ID` for upserts
- ❌ Do NOT change to `indexEndpoints/ENDPOINT_ID` for upserts
- The endpoint ID is stored in config but only used for **search/query** operations

**Code verification:**
```go
// Line 118-119 in vector_search.go (CORRECT)
indexPath := fmt.Sprintf("projects/%s/locations/%s/indexes/%s",
    projectIdentifier, v.config.Location, v.config.IndexID)  // Uses IndexID ✅

// Line 95 in vector_search.go (for search - different operation)
indexEndpoint := fmt.Sprintf("projects/%s/locations/%s/indexEndpoints/%s",
    projectIdentifier, config.Location, config.IndexEndpointID)  // Uses EndpointID ✅
```

## Testing Upserts

### Manual Test via REST API

```bash
# Get access token
TOKEN=$(gcloud auth print-access-token)

# Test upsert directly
curl -X POST \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  https://us-central1-aiplatform.googleapis.com/v1/projects/33593473489/locations/us-central1/indexes/NEW_INDEX_ID:upsertDatapoints \
  -d '{
    "datapoints": [{
      "datapoint_id": "test-123",
      "feature_vector": [0.1, 0.2, 0.3, ... (768 dimensions)]
    }]
  }'

# Success response:
# {}

# Error response will show specific issue
```

### Test via Application

```bash
# Upload a small document
curl -X POST http://localhost:8001/agents/AGENT_ID/documents \
  -F "file=@test.pdf"

# Check logs
tail -f logs/dealsense.log | grep -E "Upsert|upsert|Using index path"

# You should see:
# DEBUG Using index path for upsert: projects/33593473489/locations/us-central1/indexes/NEW_INDEX_ID
# DEBUG Prepared datapoint: ID=..., Vector dim=768
# DEBUG Sending UpsertDatapointsRequest with X datapoints to: projects/.../indexes/...
# INFO Successfully upserted X datapoints
```

## Verification Checklist

After fixing, verify:

- [ ] Index is STREAM_UPDATE mode (check with gcloud)
- [ ] Config has correct NEW_INDEX_ID (not old BATCH index)
- [ ] Index is deployed to endpoint (check with gcloud)
- [ ] Service account has aiplatform.developer role
- [ ] Logs show correct path: `indexes/NEW_INDEX_ID`
- [ ] No 404 or permission errors in logs
- [ ] Upserts succeed (see "Successfully upserted" in logs)
- [ ] Search still works (uses endpoint path)

## Expected Log Flow

```
# When document is uploaded:
INFO Generating embeddings for document...
INFO Generating embeddings for 5 valid texts...
INFO Attempting to store chunk 0 in Vector Search...
DEBUG Using index path for upsert: projects/33593473489/locations/us-central1/indexes/NEW_INDEX_ID
DEBUG Prepared datapoint: ID=doc-uuid_chunk_0, Vector dim=768
DEBUG Sending UpsertDatapointsRequest with 1 datapoints to: projects/.../indexes/NEW_INDEX_ID
INFO Successfully upserted 1 datapoints, response: {}
DEBUG Successfully stored embedding in Vector Search for chunk 0
```

## If Still Failing

1. **Share the exact error** from logs (grep "Upsert failed")
2. **Verify index ID**: `gcloud ai indexes list --region=us-central1 --project=33593473489`
3. **Check index update method**: Should be STREAM_UPDATE
4. **Confirm config**: `index_id` should match the STREAM index ID
5. **Test manually** with curl command above

## Key Takeaways

✅ **Code is correct** - Uses `indexes/INDEX_ID` for upserts  
✅ **Architecture is correct** - IndexClient for upserts, MatchClient for searches  
❓ **Most likely issue** - Config has wrong index ID or index not fully deployed  
🔍 **Next step** - Check logs with enhanced debugging to see exact error  

## Quick Fix Commands

```bash
# 1. List your indexes
gcloud ai indexes list --region=us-central1 --project=33593473489

# 2. Find your STREAM index ID (look for display-name with "stream")
# Note the ID

# 3. Update config.yaml
# Change: index_id: "YOUR_STREAM_INDEX_ID"

# 4. Restart server with debug logging
# In config.yaml: logging.level: "debug"

# 5. Test upload and check logs
tail -f logs/dealsense.log | grep -A 5 "Upsert"
```

The code is using the **correct** path pattern. The issue is most likely:
1. Wrong index ID in config (old BATCH instead of new STREAM)
2. Index not fully deployed
3. Permission issue

The enhanced logging will help identify which one!

