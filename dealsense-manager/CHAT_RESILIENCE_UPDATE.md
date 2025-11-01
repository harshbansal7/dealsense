# Chat Resilience Update

## Summary

Enhanced the chatbot to work gracefully with partial data, removing unnecessary predefined messages and letting the LLM handle edge cases intelligently.

## What Changed

### Problem: Overly Restrictive Chat Logic

**Before:** The chat would return predefined error messages in many scenarios:
- No documents available (even with transcripts)
- No transcripts available (even with documents)
- No relevant context found (even with data present)

This prevented users from getting value when only partial data was available.

### Solution: Intelligent Degradation

**Now:** The chat only returns a predefined message in ONE case:
- ❌ **BOTH documents AND transcripts are completely absent**

In all other cases, it proceeds with whatever data is available and lets the LLM provide intelligent responses.

## Updated Behavior

### Scenario 1: No Data At All ❌
```
User: What is the startup's revenue model?
Bot: I don't have any data to work with. No documents or meeting 
     transcripts have been uploaded for this agent yet. Please 
     upload some documents or start a meeting to enable me to 
     answer your questions.
```
**This is the ONLY predefined message.**

---

### Scenario 2: Documents Only ✅
```
User: What was discussed in the meeting?
Bot: I have access to the pitch documents but not the meeting 
     transcripts. Based on the documents, I can tell you about 
     [provides info from docs]. However, I cannot tell you what 
     was specifically discussed in meetings as those transcripts 
     are not available.
```
**LLM handles gracefully with clear boundaries.**

---

### Scenario 3: Transcripts Only ✅
```
User: What's the startup's revenue model?
Bot: Based on the meeting transcripts, the team discussed 
     [mentions from transcript]. However, I don't have access 
     to the pitch documents which would likely contain more 
     detailed information about the revenue model.
```
**LLM provides what it can, notes limitations.**

---

### Scenario 4: Both Available ✅
```
User: What's the revenue model?
Bot: According to the pitch deck (Page 5), the startup uses a 
     SaaS subscription model... In the meeting, the founder 
     mentioned [from transcript]... [comprehensive answer]
```
**Best case - uses all available data.**

---

### Scenario 5: Data Exists But No Relevant Context ✅
```
User: What's the quantum mechanics theory behind this?
Bot: I searched through the available documents and meeting 
     transcripts but didn't find specific information about 
     quantum mechanics. The data seems to focus on [actual topics]. 
     Is there something specific about the startup I can help with?
```
**LLM acknowledges limitation, offers to help differently.**

## Technical Changes

### 1. Removed Multiple Predefined Messages

**Before (Lines 112-131):**
```go
if len(allContext) == 0 {
    responseMsg := "I don't have enough context..."
    if docCount == 0 && meetingCount == 0 {
        responseMsg += " No documents or meeting transcripts..."
    } else if docCount == 0 {
        responseMsg += " No processed documents are available..."
    } else if meetingCount == 0 {
        responseMsg += " No meeting transcripts are available..."
    } else {
        responseMsg += " Found X documents but no relevant context..."
    }
    return predefinedResponse
}
```

**After (Lines 128-150):**
```go
// ONLY return predefined message if BOTH sources are absent
if len(allContext) == 0 && docCount == 0 && meetingCount == 0 {
    return predefinedResponse // Only one case!
}

// If data exists but no relevant context, let LLM handle it
if len(allContext) == 0 {
    // Add system note for LLM guidance
    infoChunk := ContextChunk{
        Text:   "System searched data but found no relevant matches",
        Source: "system",
    }
    allContext = append(allContext, infoChunk)
}
// Proceed to LLM with guidance
```

### 2. Enhanced Prompt Engineering

**Key Improvements:**
- LLM receives clear guidance about which data sources are available
- Explicitly instructed to work with partial data
- Told to acknowledge limitations but provide value where possible
- System context helps LLM understand search results

**New Prompt Structure:**
```
You are an intelligent assistant...

[Dynamic based on available data:]
- "You have access to documents only" OR
- "You have access to transcripts only" OR  
- "You have both" OR
- "Data exists but wasn't relevant"

Always be honest about what you have and don't have.
If you can partially answer, do so and explain what's missing.

[Context if available]
[System notes if needed]

USER QUESTION: ...
ANSWER: [LLM responds intelligently]
```

### 3. Better Logging

```go
if hasDocContext && !hasMeetingContext {
    logrus.Infof("Using document context only (%d chunks)", len(documentContext))
} else if !hasDocContext && hasMeetingContext {
    logrus.Infof("Using meeting transcript context only (%d chunks)", len(meetingContext))
} else if hasDocContext && hasMeetingContext {
    logrus.Infof("Using both document (%d) and meeting (%d) context", ...)
} else if hasAnyData {
    logrus.Infof("Data available but no relevant context found")
} else {
    logrus.Infof("No data available for this agent")
}
```

## Benefits

### 1. Better User Experience
- ✅ Users get value even with partial data
- ✅ Clear communication about what's available/missing
- ✅ No frustrating "not enough context" errors when data exists

### 2. More Intelligent Responses
- ✅ LLM can provide partial answers
- ✅ LLM can suggest what additional data would help
- ✅ LLM adapts response based on available sources

### 3. Flexibility
- ✅ Works during document upload (transcripts available)
- ✅ Works during meetings (documents available)
- ✅ Works with either/both data sources
- ✅ Gracefully handles search misses

### 4. Production Ready
- ✅ No hard failures on edge cases
- ✅ Comprehensive logging for debugging
- ✅ LLM-driven degradation instead of code-driven blockers

## Testing Scenarios

### Test 1: Documents Only
```bash
# Upload documents, don't start meetings
curl -X POST http://localhost:8001/agents/{agent_id}/documents -F "file=@pitch.pdf"

# Ask meeting-related question
curl -X POST http://localhost:8001/agents/{agent_id}/chat \
  -d '{"query": "What was discussed about revenue?", "top_k": 5}'

# Expected: LLM provides info from documents, notes no transcripts
```

### Test 2: Transcripts Only
```bash
# Start meeting (generates transcripts), no documents

# Ask document-related question
curl -X POST http://localhost:8001/agents/{agent_id}/chat \
  -d '{"query": "What's on page 5 of the pitch deck?", "top_k": 5}'

# Expected: LLM explains no documents available, provides meeting info if relevant
```

### Test 3: No Relevant Context
```bash
# Upload tech startup docs

# Ask unrelated question
curl -X POST http://localhost:8001/agents/{agent_id}/chat \
  -d '{"query": "What is the recipe for chocolate cake?", "top_k": 5}'

# Expected: LLM acknowledges no relevant data, may offer to help with actual content
```

### Test 4: No Data At All
```bash
# New agent, nothing uploaded

curl -X POST http://localhost:8001/agents/{agent_id}/chat \
  -d '{"query": "Tell me about the startup", "top_k": 5}'

# Expected: Predefined message asking to upload data
```

## Monitoring

Check logs for chat behavior:

```bash
# See what data sources are being used
grep "Using.*context" logs/dealsense.log

# Should show patterns like:
# "Using document context only (3 chunks)"
# "Using meeting transcript context only (5 chunks)"
# "Using both document (3) and meeting (2) context"
# "Data available but no relevant context found"
```

## Rollback

If issues arise, the key change to revert is in `chatbot.go` lines 128-150. Simply restore the old predefined message logic. However, the new approach is more robust and should handle all cases better.

## Future Enhancements

1. **Hybrid Search**: Combine semantic (embeddings) + keyword search for better recall
2. **Query Expansion**: Rephrase user queries automatically to improve matches
3. **Source Weighting**: Prioritize more recent transcripts or specific documents
4. **Feedback Loop**: Learn from user feedback which responses were helpful
5. **Streaming**: Stream responses for better UX on long answers

## Files Modified

- `backend_v2/internal/document/chatbot.go`:
  - Simplified predefined message logic (lines 128-150)
  - Enhanced prompt engineering (lines 282-362)
  - Better logging (lines 111-126)

## Key Principle

**Let the LLM handle ambiguity, don't block users with code.**

The LLM is better at:
- Understanding context limitations
- Providing partial answers
- Explaining what's missing
- Offering alternative help

The code should:
- Gather available data
- Inform LLM about data sources
- Get out of the way

This is the modern RAG approach: **intelligent degradation via LLM, not code-level blocking.**

