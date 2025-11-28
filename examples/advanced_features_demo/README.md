# Advanced Features Demo - Decision API

This demo showcases the advanced features of the Flagship Decision API:

- **Experience Continuity (XPC)**: Consistent visitor assignments across sessions
- **1 Visitor 1 Test (1v1t)**: Single campaign assignment per visitor

## Prerequisites

1. **Running Decision API** instance with persistent cache (Redis or DynamoDB)
2. **Flagship environment** with campaigns configured
3. **Environment credentials**: ENV_ID and API_KEY

## Setup

### 1. Configure the Demo

Edit `main.go` and update these constants:

```go
const (
    decisionAPIURL = "http://localhost:8080"  // Your Decision API endpoint
    envID  = "your_env_id"                     // Your Flagship environment ID
    apiKey = "your_api_key"                    // Your Flagship API key
)
```

### 2. Start Decision API with Persistent Cache

For **Experience Continuity** to work, you need persistent storage:

#### Option A: Using Redis (Recommended)

```bash
# Start Redis
docker run -d -p 6379:6379 redis

# Start Decision API with Redis cache
docker run -p 8080:8080 \
  -e ENV_ID=your_env_id \
  -e API_KEY=your_api_key \
  -e CACHE_TYPE=redis \
  -e CACHE_OPTIONS_REDISHOST=host.docker.internal:6379 \
  flagshipio/decision-api
```

#### Option B: Using Docker Compose

```bash
# Use the provided docker-compose.yml in the root directory
cd ../..
docker compose up -d
```

#### Option C: Using Local Cache

```bash
docker run -p 8080:8080 \
  -e ENV_ID=your_env_id \
  -e API_KEY=your_api_key \
  -e CACHE_TYPE=local \
  -e CACHE_OPTIONS_DBPATH=/data/cache \
  -v $(pwd)/cache_data:/data/cache \
  flagshipio/decision-api
```

### 3. Configure Flagship Account Settings

In your Flagship dashboard, enable:

- **Experience Continuity (XPC)**: Settings → Advanced → Cross-Platform Consistency
- **1 Visitor 1 Test (1v1t)**: Settings → Advanced → Single Assignment

Create at least 2-3 campaigns with different targeting rules to see the features in action.

## Running the Demo

```bash
cd examples/advanced_features_demo
go run main.go
```

## What to Expect

### Scenario 1: Experience Continuity (XPC)

```
Visitor: alice_1234567890
Making first request to Decision API...
✓ First request: Received 1 campaign(s)
  1. Campaign ID: campaign_abc123
     Variation: Variation A (var_xyz789)
     Type: ab

Activating campaign: campaign_abc123 (variation: var_xyz789)
✓ Campaign activated - assignment now cached

Making second request (simulating return visit)...
✓ Second request: Received 1 campaign(s)
  1. Campaign ID: campaign_abc123
     Variation: Variation A (var_xyz789)
     Type: ab

✓ EXPERIENCE CONTINUITY VERIFIED: Same variation assigned across requests!
```

**Key Point**: The visitor receives the **same variation** on subsequent requests because the assignment is cached.

### Scenario 2: 1 Visitor 1 Test (1v1t)

```
Visitor: bob_1234567891
With 1v1t enabled, visitor should be in AT MOST 1 campaign

✓ Retrieved campaigns: 1
  1. Campaign ID: campaign_def456
     Variation: Control (var_control)
     Type: ab

✓ 1 VISITOR 1 TEST VERIFIED: Visitor assigned to at most 1 campaign
```

**Key Point**: Even if multiple campaigns are active and the visitor qualifies for several, they are assigned to **at most one** to avoid interaction effects.

### Scenario 3: Cross-Session Consistency

```
Simulating 3 sessions for visitor: charlie_1234567892

--- Session 1 ---
Campaigns received: 1
Assigned variation: var_xyz789
✓ Campaign activated

--- Session 2 ---
Campaigns received: 1
Assigned variation: var_xyz789

--- Session 3 ---
Campaigns received: 1
Assigned variation: var_xyz789

✓ CROSS-SESSION CONSISTENCY VERIFIED: Same variation across all sessions!
```

**Key Point**: Assignments persist across sessions (days, weeks) as long as the cache is maintained.

### Scenario 4: Context-Based Targeting

```
Visitor: premium_user_1234567893
Context: map[country:US plan:premium vip:true]
Eligible campaigns: 1
  1. Campaign ID: premium_feature_test
     Variation: Premium Flow (var_premium)
     Flags:
       - showNewUI: true
       - discountPercent: 20

Visitor: free_user_1234567894
Context: map[country:FR plan:free vip:false]
Eligible campaigns: 1
  1. Campaign ID: free_tier_test
     Variation: Standard Flow (var_standard)
     Flags:
       - showNewUI: false
       - discountPercent: 0

✓ Context-based targeting allows different campaigns for different user segments
```

**Key Point**: Visitor context determines campaign eligibility, enabling sophisticated audience targeting.

## Troubleshooting

### "Different variations - cache may not be configured"

**Problem**: Experience Continuity not working

**Solutions**:

1. Verify cache is configured (not using empty/memory-only cache)
2. Check Redis/DynamoDB is running and accessible
3. Ensure `CacheEnabled: true` in Flagship settings
4. Check Decision API logs for cache errors

### "⚠ Visitor in X campaigns - 1v1t may not be enabled"

**Problem**: 1 Visitor 1 Test not enforced

**Solutions**:

1. Enable "Single Assignment" in Flagship dashboard
2. Verify environment configuration is fetched correctly
3. Check that `Enabled1V1T: true` in account settings
4. Allow time for configuration to propagate (up to 1 minute polling interval)

### "API returned status 400/500"

**Problem**: Request errors

**Solutions**:

1. Verify ENV_ID and API_KEY are correct
2. Check Decision API is running (`curl http://localhost:8080/health`)
3. Ensure campaigns exist in your Flagship environment
4. Check Decision API logs for detailed errors

## Architecture Diagram

```
┌─────────────────┐
│   Demo App      │
│                 │
│  - alice (XPC)  │
│  - bob (1v1t)   │
│  - charlie      │
└────────┬────────┘
         │ HTTP Requests
         │ (POST /v2/campaigns)
         │
         ▼
┌─────────────────┐      ┌──────────────┐
│  Decision API   │◄────►│ Redis Cache  │
│  (Port 8080)    │      │ (Assignments)│
└────────┬────────┘      └──────────────┘
         │
         │ Poll Config (1min)
         ▼
┌─────────────────┐
│ Flagship CDN    │
│ (Configuration) │
└─────────────────┘
         │
         │ Send Tracking
         ▼
┌─────────────────┐
│ Data Collection │
│ (Analytics)     │
└─────────────────┘
```

## Key Takeaways

1. **XPC requires persistent storage** - Use Redis, DynamoDB, or local cache (not memory)
2. **1v1t reduces test pollution** - Visitors see consistent, single-test experiences
3. **Activation triggers caching** - Call `/activate` to persist assignments
4. **Context drives targeting** - Visitor attributes determine campaign eligibility
5. **Cache is customer-managed** - In self-hosted mode, you control cache availability

## Advanced Usage

### Custom Visitor Context

Add more context fields for sophisticated targeting:

```go
context := map[string]interface{}{
    "age":           28,
    "country":       "US",
    "plan":          "premium",
    "signupDate":    "2024-01-15",
    "totalSpent":    1250.50,
    "device":        "mobile",
    "appVersion":    "3.2.1",
    "experiments":   []string{"feature_x", "feature_y"},
}
```

### Anonymous to Authenticated Visitor

```go
// First visit (anonymous)
anonymousID := "anon_xyz123"
resp1, _ := client.GetCampaigns(anonymousID, context)

// User logs in (reconciliation)
authenticatedID := "user@example.com"
// Use anonymous_id in request to link identities
```

### Monitoring Cache Performance

Check if assignments are cached:

```bash
# Redis
redis-cli keys "*"
redis-cli get "env_id.visitor_id"

# Decision API metrics
curl http://localhost:8080/metrics
```

## Next Steps

1. **Integrate into your application** - Use this demo as a reference
2. **Set up monitoring** - Track cache hit rates and decision latency
3. **Configure backups** - Ensure cache data is backed up for continuity
4. **Test failover** - Verify behavior when cache is unavailable
5. **Review legal guidance** - See `SELF_HOSTING_LEGAL_GUIDANCE.md` for operational responsibilities
