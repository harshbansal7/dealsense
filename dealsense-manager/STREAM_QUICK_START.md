# STREAM_UPDATE Quick Start Guide

## TL;DR - 3 Commands to Switch

```bash
# 1. Create STREAM index (use config from MIGRATION_GUIDE.md)
gcloud ai indexes create \
  --display-name="dealsense-embeddings-stream" \
  --metadata-file=vector-index-stream-config.json \
  --region=us-central1 \
  --project=33593473489

# 2. Switch indexes on your endpoint (45-75 min downtime)
gcloud ai index-endpoints undeploy-index 6674108148367753216 \
  --deployed-index-id=dealsense_deployed \
  --region=us-central1 \
  --project=33593473489

gcloud ai index-endpoints deploy-index 6674108148367753216 \
  --deployed-index-id=dealsense_deployed \
  --index=NEW_INDEX_ID \
  --region=us-central1 \
  --project=33593473489

# 3. Update config.yaml and restart
# Change: index_id: "NEW_INDEX_ID"
```

## What Changes?

### Index Config File

**BEFORE (BATCH_UPDATE):**
```json
{
  "contentsDeltaUri": "gs://your-bucket/vector-data",  ← Remove this
  "config": {
    "dimensions": 768,
    ...
  }
}
```

**AFTER (STREAM_UPDATE):**
```json
{
  "config": {
    "dimensions": 768,
    "shardSize": "SHARD_SIZE_SMALL",  ← Add this
    ...
  }
}
```

### Your config.yaml

**Only this line changes:**
```yaml
vector_search:
  index_id: "NEW_INDEX_ID"  # ← Update this after creating STREAM index
```

### Your Code

**No changes needed!** ✅

Your code already uses:
- `UpsertDatapoints()` - Perfect for STREAM ✅
- `RemoveDatapoints()` - Perfect for STREAM ✅
- `FindNeighbors()` - Works with both ✅

## Why STREAM_UPDATE?

| Your Current Code | BATCH Index | STREAM Index |
|-------------------|-------------|--------------|
| `UpsertDatapoints()` RPC | ⚠️ Wrong mode | ✅ Correct mode |
| Real-time uploads | ⚠️ Requires GCS | ✅ Direct RPC |
| Instant embeddings | ❌ Batch delay | ✅ Instant |

**Bottom line:** Your code was designed for STREAM mode, but your index is in BATCH mode. This mismatch works but isn't optimal.

## Benefits After Switch

✅ **Real-time** - Embeddings available instantly  
✅ **Simpler** - No GCS bucket needed for updates  
✅ **Correct** - Matches your code's design  
✅ **Same cost** - No price increase  
✅ **Same code** - Zero code changes  

## Full Documentation

- **Detailed steps:** [STREAM_UPDATE_MIGRATION.md](./STREAM_UPDATE_MIGRATION.md)
- **Step-by-step:** [MIGRATION_GUIDE.md](./MIGRATION_GUIDE.md) (updated)
- **Interactive script:** `./STREAM_COMMANDS.sh`

## Quick Checklist

- [ ] Create `vector-index-stream-config.json` (see MIGRATION_GUIDE.md)
- [ ] Run `gcloud ai indexes create ...` (5-10 min)
- [ ] Note NEW_INDEX_ID
- [ ] Undeploy old index (10-15 min) ⚠️ Downtime starts
- [ ] Deploy new index (30-60 min) ⚠️ Downtime continues
- [ ] Update config.yaml: `index_id: "NEW_INDEX_ID"`
- [ ] Restart server
- [ ] Test upload
- [ ] Delete old BATCH index

**Total time:** ~60-90 minutes  
**Downtime:** ~45-75 minutes (or 0 with new endpoint)

## Need Help?

See [STREAM_UPDATE_MIGRATION.md](./STREAM_UPDATE_MIGRATION.md) for:
- Zero-downtime option
- Troubleshooting
- FAQ
- Verification steps

