# Vector Search Configuration Update Guide

## Summary

This guide explains the recent update to Vector Search configuration that adds support for using the GCP **project number** instead of just the project ID.

## What Changed?

### Before
The configuration only supported `project_id`, which caused issues because Google Vector Search requires the **project number** (numeric identifier) rather than the project ID (alphanumeric with dashes).

```yaml
google:
  project_id: "genai-exchange-475318"
  
  vertex_ai:
    vector_search:
      enabled: true
      index_id: "3413880250151469056"
      index_endpoint_id: "6674108148367753216"
```

### After
A new optional field `project_number` has been added to the Vector Search configuration:

```yaml
google:
  project_id: "genai-exchange-475318"  # Still used for Document AI, Storage, etc.
  
  vertex_ai:
    vector_search:
      enabled: true
      project_number: "33593473489"  # NEW: Use your GCP project number here
      index_id: "3413880250151469056"
      index_endpoint_id: "6674108148367753216"
```

## Why This Change?

Google Vector Search resource paths use the **project number**, not the project ID:
```
projects/33593473489/locations/us-central1/indexes/3413880250151469056
         ^^^^^^^^^^^
         project number (numeric)
```

Previously, the system would construct paths like:
```
projects/genai-exchange-475318/locations/us-central1/indexes/3413880250151469056
         ^^^^^^^^^^^^^^^^^^^^^
         project ID (wrong for Vector Search!)
```

This caused 404 errors or permission denied errors when trying to access Vector Search resources.

## How to Update Your Configuration

### Step 1: Find Your Project Number

You have Vector Search URLs that look like this:
```
projects/33593473489/locations/us-central1/indexes/3413880250151469056
projects/33593473489/locations/us-central1/indexEndpoints/6674108148367753216
```

The first number (`33593473489`) is your **project number**.

Alternatively, you can find it using the `gcloud` command:
```bash
gcloud projects describe YOUR_PROJECT_ID --format="value(projectNumber)"
```

For example:
```bash
gcloud projects describe genai-exchange-475318 --format="value(projectNumber)"
# Output: 33593473489
```

### Step 2: Update Your `config.yaml`

Add the `project_number` field to your Vector Search configuration:

```yaml
google:
  project_id: "genai-exchange-475318"  # Keep this - used by other services
  
  vertex_ai:
    location: "us-central1"
    vector_search:
      enabled: true
      project_number: "33593473489"  # ADD THIS LINE
      index_id: "3413880250151469056"
      index_endpoint_id: "6674108148367753216"
```

### Step 3: Restart Your Server

After updating the configuration, restart the Dealsense backend:
```bash
# If running with Docker
docker-compose restart backend

# If running directly
./server
```

### Step 4: Verify

Check the logs at startup for this message:
```
INFO Vector Search Service initialized: projects/33593473489/locations/us-central1/indexEndpoints/6674108148367753216
```

The path should now use your **project number** (numeric), not project ID.

## Backward Compatibility

The change is **backward compatible**:
- If you don't specify `project_number`, the system will fall back to using `project_id`
- This maintains compatibility with any existing configurations

However, for Vector Search to work correctly, it's **highly recommended** to add the `project_number` field.

## Technical Details

### Code Changes

1. **Config Structure** (`internal/config/config.go`):
   ```go
   type VectorSearchConfig struct {
       Enabled         bool   `yaml:"enabled"`
       ProjectNumber   string `yaml:"project_number"`    // NEW FIELD
       IndexID         string `yaml:"index_id"`
       IndexEndpointID string `yaml:"index_endpoint_id"`
   }
   ```

2. **Vector Search Service** (`internal/document/vector_search.go`):
   - Added `getProjectIdentifier()` helper method
   - Returns `ProjectNumber` if set, otherwise falls back to `ProjectID`
   - All path construction now uses this method

3. **Main Server** (`cmd/server/main.go`):
   - Updated to pass `ProjectNumber` from config to Vector Search service

### Files Modified

- `/dealsense-manager/backend_v2/internal/config/config.go`
- `/dealsense-manager/backend_v2/internal/document/vector_search.go`
- `/dealsense-manager/backend_v2/cmd/server/main.go`
- `/dealsense-manager/config.example.yaml`
- `/dealsense-manager/CONFIG_DOCUMENTATION.md`
- `/dealsense-manager/VECTOR_SEARCH_TROUBLESHOOTING.md`

## Troubleshooting

### Issue: Vector Search still not working after update

**Check:**
1. Verify you're using the correct project number (numeric, not alphanumeric with dashes)
2. Ensure the location matches your Vector Search index location
3. Verify the index_id and index_endpoint_id are correct

**Logs to check:**
```bash
grep "Vector Search" logs/dealsense.log
```

Look for:
```
INFO Vector Search Service initialized: projects/YOUR_PROJECT_NUMBER/...
```

### Issue: 404 or permission denied errors

This usually means:
1. Wrong project number
2. Wrong index ID or endpoint ID
3. Service account doesn't have permissions

**Verify your resource paths:**
```bash
# List your Vector Search indexes
gcloud ai indexes list --region=us-central1

# List your Vector Search endpoints
gcloud ai index-endpoints list --region=us-central1
```

## Additional Resources

- [CONFIG_DOCUMENTATION.md](./CONFIG_DOCUMENTATION.md) - Full configuration reference
- [VECTOR_SEARCH_TROUBLESHOOTING.md](./VECTOR_SEARCH_TROUBLESHOOTING.md) - Detailed troubleshooting guide
- [Google Cloud Vector Search Documentation](https://cloud.google.com/vertex-ai/docs/vector-search/overview)

## Questions?

If you encounter issues:
1. Check the logs: `logs/dealsense.log`
2. Verify your configuration against the examples above
3. Consult the troubleshooting guide: `VECTOR_SEARCH_TROUBLESHOOTING.md`

