# CRITICAL FIX: Public Endpoint for Vector Search Queries

## The Root Cause (Finally Found!) 🎯

The "Operation is not implemented" error was caused by using the **wrong endpoint** for the `MatchClient`.

**Note:** The error was NOT about restricts - restricts ARE fully supported in Vector Search! See `VECTOR_SEARCH_RESTRICTS.md` for details on metadata filtering.

### Two Different Endpoints Required

Google Vector Search has **two separate endpoints** for different operations:

| Operation | Client | Endpoint Type | Example |
|-----------|--------|---------------|---------|
| **Upsert/Remove** | `IndexClient` | Regional aiplatform | `us-central1-aiplatform.googleapis.com:443` |
| **Search/Query** | `MatchClient` | Public VDB | `266063970.us-central1-33593473489.vdb.vertexai.goog:443` |

**We were using the regional aiplatform endpoint for queries, which doesn't support FindNeighbors!**

## How to Get Your Public Endpoint

Run this command:

```bash
gcloud ai index-endpoints describe INDEX_ENDPOINT_ID \
  --region=REGION \
  --project=PROJECT_ID \
  --format=json | grep publicEndpointDomainName
```

For your setup:

```bash
gcloud ai index-endpoints describe 8766030175281348608 \
  --region=us-central1 \
  --project=genai-exchange-475318 \
  --format=json | grep publicEndpointDomainName
```

**Output:**
```json
"publicEndpointDomainName": "266063970.us-central1-33593473489.vdb.vertexai.goog"
```

## Required Configuration Update

### 1. Update your `config.yaml`

Add the `public_endpoint_domain` field:

```yaml
google:
  vertex_ai:
    vector_search:
      enabled: true
      project_number: "33593473489"
      index_id: "3629630819800186880"
      index_endpoint_id: "8766030175281348608"
      deployed_index_id: "dealsense_deployed"
      public_endpoint_domain: "266063970.us-central1-33593473489.vdb.vertexai.goog"  # ← ADD THIS
```

**CRITICAL:** Do NOT include `https://` or port in the config. The code adds `:443` automatically.

### 2. Your Specific Values

From the `gcloud` output, here are your exact values:

```yaml
vector_search:
  enabled: true
  project_number: "33593473489"
  index_id: "3629630819800186880"
  index_endpoint_id: "8766030175281348608"
  deployed_index_id: "dealsense_deployed"
  public_endpoint_domain: "266063970.us-central1-33593473489.vdb.vertexai.goog"
```

## What Changed in the Code

### Before (❌ Wrong):

```go
// Both clients used the same regional endpoint
endpoint := fmt.Sprintf("%s-aiplatform.googleapis.com:443", config.Location)

indexClient, _ = aiplatform.NewIndexClient(ctx, option.WithEndpoint(endpoint))
matchClient, _ = aiplatform.NewMatchClient(ctx, option.WithEndpoint(endpoint))  // ❌ WRONG!
```

### After (✅ Correct):

```go
// Different endpoints for different operations
managementEndpoint := fmt.Sprintf("%s-aiplatform.googleapis.com:443", config.Location)
queryEndpoint := fmt.Sprintf("%s:443", config.PublicEndpointDomain)

// IndexClient: Uses regional endpoint for upsert/remove
indexClient, _ = aiplatform.NewIndexClient(ctx, option.WithEndpoint(managementEndpoint))

// MatchClient: Uses PUBLIC VDB endpoint for queries
matchClient, _ = aiplatform.NewMatchClient(ctx, option.WithEndpoint(queryEndpoint))  // ✅ CORRECT!
```

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│ Your Application                                            │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  IndexClient                     MatchClient                │
│       │                               │                     │
│       │ Upsert/Remove                 │ FindNeighbors       │
│       │                               │                     │
│       ▼                               ▼                     │
│  us-central1-                    266063970.                 │
│  aiplatform.googleapis.com       us-central1-33593473489.   │
│                                  vdb.vertexai.goog          │
│       │                               │                     │
│       │                               │                     │
└───────┼───────────────────────────────┼─────────────────────┘
        │                               │
        │                               │
        ▼                               ▼
┌──────────────────┐          ┌──────────────────┐
│  AI Platform     │          │  Vector DB       │
│  Management API  │          │  Query Endpoint  │
│                  │          │  (Public)        │
│  - Create Index  │          │  - Search        │
│  - Upsert Data   │          │  - Find Nearest  │
│  - Delete Data   │          │                  │
└──────────────────┘          └──────────────────┘
```

## Why This Happens

1. **Management Operations**: Need to go through the centralized AI Platform API (regional endpoint)
2. **Query Operations**: Are routed to the specific deployed index's public endpoint for low latency

The public VDB endpoint is a **direct connection** to your deployed index, optimized for fast queries.

## Verification

After updating the config and restarting, you should see these logs:

```
INFO Vector Search IndexClient endpoint: us-central1-aiplatform.googleapis.com:443
INFO Vector Search MatchClient endpoint: 266063970.us-central1-33593473489.vdb.vertexai.goog:443
```

Then when you upload a document and try to chat, it should work! ✅

## Files Modified

1. **`internal/document/vector_search.go`**:
   - Added `PublicEndpointDomain` field to `VectorSearchConfig`
   - Split endpoint initialization: `managementEndpoint` vs `queryEndpoint`
   - `IndexClient` → management endpoint
   - `MatchClient` → public VDB endpoint

2. **`internal/config/config.go`**:
   - Added `PublicEndpointDomain string` field to `VectorSearchConfig`

3. **`cmd/server/main.go`**:
   - Pass `PublicEndpointDomain` from config to service

## Common Issues

### Issue 1: Still getting "Operation not implemented"

**Cause:** `public_endpoint_domain` not set in config.yaml

**Fix:** Add the field with your exact public endpoint domain (no https://, no port)

### Issue 2: "Connection refused" or "Name resolution failed"

**Cause:** Incorrect public endpoint domain or network issue

**Fix:** 
- Verify the domain with `gcloud ai index-endpoints describe ...`
- Ensure your network can reach `*.vdb.vertexai.goog` domains
- Check firewall rules

### Issue 3: Upserts work but queries fail

**Cause:** This was the exact problem! Queries need the public endpoint.

**Fix:** Already fixed in the code. Just update your config.

## Summary

✅ **Root Cause Identified**: Using wrong endpoint for MatchClient  
✅ **Solution Implemented**: Separate endpoints for IndexClient and MatchClient  
✅ **Configuration Required**: Add `public_endpoint_domain` to config.yaml  
✅ **Your Value**: `266063970.us-central1-33593473489.vdb.vertexai.goog`  

**Update your config.yaml and restart the server!** 🚀

