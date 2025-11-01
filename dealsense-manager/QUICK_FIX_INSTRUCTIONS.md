# Quick Fix: Add Public Endpoint to Config

## What You Need to Do

Add ONE line to your `config.yaml` file in the `vector_search` section:

```yaml
google:
  vertex_ai:
    vector_search:
      enabled: true
      project_number: "33593473489"
      index_id: "3629630819800186880"
      index_endpoint_id: "8766030175281348608"
      deployed_index_id: "dealsense_deployed"
      public_endpoint_domain: "266063970.us-central1-33593473489.vdb.vertexai.goog"  # ← ADD THIS LINE
```

## Your Exact Value

Based on your deployment, your `public_endpoint_domain` is:

```
266063970.us-central1-33593473489.vdb.vertexai.goog
```

**Do NOT include:**
- `https://`
- Port number (`:443`)
- Trailing slash

Just the domain name as shown above.

## How to Get This Value (For Future Reference)

```bash
gcloud ai index-endpoints describe 8766030175281348608 \
  --region=us-central1 \
  --project=genai-exchange-475318 \
  --format="value(publicEndpointDomainName)"
```

Output: `266063970.us-central1-33593473489.vdb.vertexai.goog`

## After Updating

1. Save your `config.yaml`
2. Restart the server
3. The error should be gone! ✅

## What This Fixes

The "Operation is not implemented" error was happening because:
- **IndexClient** (upsert) uses: `us-central1-aiplatform.googleapis.com` ✅ Already working
- **MatchClient** (search) needs: `266063970.us-central1-33593473489.vdb.vertexai.goog` ❌ Was missing

Now both will work correctly.

**Bonus:** We've also implemented **restricts** (metadata filtering) for efficient multi-tenant isolation! See `VECTOR_SEARCH_RESTRICTS.md` for details.

## Verification

After restart, check logs for:

```
INFO Vector Search IndexClient endpoint: us-central1-aiplatform.googleapis.com:443
INFO Vector Search MatchClient endpoint: 266063970.us-central1-33593473489.vdb.vertexai.goog:443
```

Then upload a document and try to chat - it should work! 🎉

## Full Details

See `PUBLIC_ENDPOINT_FIX.md` for complete technical explanation.

