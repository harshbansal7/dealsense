# Quick Configuration Reference

## Your Vector Search Configuration

Based on your resource paths, here's exactly what you need in your `config.yaml`:

```yaml
google:
  project_id: "genai-exchange-475318"
  
  vertex_ai:
    location: "us-central1"
    embedding_model: "text-embedding-004"
    use_default_credentials: true
    
    vector_search:
      enabled: true
      project_number: "33593473489"
      index_id: "3413880250151469056"
      index_endpoint_id: "6674108148367753216"
```

## How I Got These Values

From your resource paths:

**Index Path:**
```
projects/33593473489/locations/us-central1/indexes/3413880250151469056
         └─────────┘                                └──────────────────┘
      project_number                                    index_id
```

**Index Endpoint Path:**
```
projects/33593473489/locations/us-central1/indexEndpoints/6674108148367753216
         └─────────┘                                      └──────────────────┘
      project_number                                   index_endpoint_id
```

## Key Points

1. **`project_number`**: `"33593473489"` - This is your GCP project number (numeric)
2. **`project_id`**: `"genai-exchange-475318"` - Keep this for other GCP services (Document AI, Storage, etc.)
3. **`location`**: `"us-central1"` - Must match your Vector Search index location
4. **`index_id`**: Just the numeric ID, not the full path
5. **`index_endpoint_id`**: Just the numeric ID, not the full path

## What Changed

**Before:** The system was trying to use `project_id` for Vector Search paths, which is wrong.

**Now:** The system uses `project_number` for Vector Search paths, which is correct.

## Verify It Works

After updating your config and restarting, check logs for:
```
INFO Vector Search Service initialized: projects/33593473489/locations/us-central1/indexEndpoints/6674108148367753216
```

If you see your **project number** (33593473489) in the path, you're good! ✅

If you see your **project ID** (genai-exchange-475318) in the path, something's wrong. ❌

## Complete Example Config

Here's a more complete example with all sections:

```yaml
server:
  host: "0.0.0.0"
  port: 8001
  read_timeout: 30s
  write_timeout: 30s

logging:
  level: "info"
  format: "text"
  file:
    enabled: true
    path: "logs/dealsense.log"

google:
  project_id: "genai-exchange-475318"
  
  storage:
    bucket_name: "your-bucket-name"
    use_default_credentials: true
  
  document_ai:
    location: "us"
    processor_id: "your-processor-id"
    use_default_credentials: true
  
  vertex_ai:
    location: "us-central1"
    embedding_model: "text-embedding-004"
    use_default_credentials: true
    
    vector_search:
      enabled: true
      project_number: "33593473489"
      index_id: "3413880250151469056"
      index_endpoint_id: "6674108148367753216"

database:
  type: "postgres"
  host: "localhost"
  port: 5432
  user: "dealsense"
  password: "your-password"
  dbname: "dealsense"
  sslmode: "disable"
```

## Need More Help?

- Full documentation: [CONFIG_DOCUMENTATION.md](./CONFIG_DOCUMENTATION.md)
- Troubleshooting: [VECTOR_SEARCH_TROUBLESHOOTING.md](./VECTOR_SEARCH_TROUBLESHOOTING.md)
- Update guide: [VECTOR_SEARCH_CONFIG_UPDATE.md](./VECTOR_SEARCH_CONFIG_UPDATE.md)

