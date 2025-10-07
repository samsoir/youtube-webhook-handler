# Webhook Authentication & Security Design

## Overview

Implement hybrid authentication to protect the YouTube Webhook Service from unauthorized access while maintaining compatibility with YouTube's PubSubHubbub notifications.

## Current State

- All endpoints are publicly accessible (`allUsers` has `roles/run.invoker`)
- No authentication or verification
- Potential for abuse/cost escalation
- Cloud Scheduler uses OIDC (already secure)

## Security Architecture

### 1. YouTube Webhook Endpoint (POST /)

**Method:** HMAC Signature Verification

YouTube's PubSubHubbub sends a signature in the `X-Hub-Signature` header using HMAC-SHA1.

**Implementation:**
- Parse `X-Hub-Signature` header (format: `sha1=<hex-digest>`)
- Compute HMAC-SHA1 of request body using secret
- Compare signatures in constant-time
- Reject invalid/missing signatures

**Secret Storage:**
- Store webhook secret in Google Secret Manager
- Load at function startup
- Rotate periodically

**References:**
- [PubSubHubbub Spec](https://pubsubhubbub.github.io/PubSubHubbub/pubsubhubbub-core-0.4.html#authednotify)

### 2. Management Endpoints

**Method:** Cloud IAM with OIDC Tokens

Endpoints requiring protection:
- `POST /subscribe` - Creates subscriptions
- `DELETE /unsubscribe` - Removes subscriptions
- `GET /subscriptions` - Lists subscriptions
- `POST /renew` - Triggers renewal

**Implementation:**
- Remove `allUsers` from IAM policy (keep scheduler service account)
- Require `Authorization: Bearer <token>` header
- Validate tokens using Google's IAM library
- Return 401/403 for invalid/missing tokens

**CLI Changes:**
- Use `gcloud auth print-identity-token` to get token
- Send token in Authorization header
- Handle token expiration/refresh

### 3. Cloud Scheduler (POST /renew)

**Status:** Already Secure ✓

- Uses OIDC authentication
- Service account: `yt-sched-prod-*@*.iam.gserviceaccount.com`
- No changes needed

### 4. Verification Challenge (GET /)

**Status:** Keep Public ✓

- YouTube sends verification challenges during subscription
- Must remain publicly accessible
- Low risk (read-only, deterministic response)
- No changes needed

## Implementation Tasks

### Phase 1: YouTube Webhook Signature Verification

- [ ] Add webhook secret to Secret Manager
  - Secret name: `youtube-webhook-secret-${environment}`
  - Generate cryptographically random secret
  - Document secret generation process

- [ ] Update Terraform
  - Create Secret Manager secret resource
  - Grant function service account `secretAccessor` role
  - Add secret name to function environment variables

- [ ] Implement signature verification
  - Create `pkg/auth/signature.go`
  - Implement `VerifyHubSignature(secret, body, signature string) bool`
  - Use constant-time comparison
  - Add comprehensive tests

- [ ] Update webhook handler
  - Extract `X-Hub-Signature` header
  - Verify signature before processing
  - Return 403 for invalid signatures
  - Log verification failures

- [ ] Update subscription creation
  - Include webhook secret when subscribing
  - Pass secret to YouTube hub
  - Document secret in subscription state

### Phase 2: IAM Authentication for Management Endpoints

- [ ] Update Terraform IAM
  - Remove `allUsers` from `roles/run.invoker`
  - Keep scheduler service account
  - Document required permissions for CLI users

- [ ] Implement token verification
  - Create `pkg/auth/iam.go`
  - Implement `VerifyIDToken(ctx, token string) (*TokenInfo, error)`
  - Use Google IAM libraries
  - Add comprehensive tests

- [ ] Update router middleware
  - Create authentication middleware
  - Apply to protected endpoints only
  - Exclude webhook and verification endpoints
  - Return 401 for missing tokens, 403 for invalid

- [ ] Update CLI client
  - Add `getAuthToken()` function
  - Call `gcloud auth print-identity-token`
  - Include token in request headers
  - Handle authentication errors gracefully

- [ ] Update documentation
  - CLI README: authentication requirements
  - Add troubleshooting for auth failures
  - Document `gcloud auth login` requirement

### Phase 3: Testing & Validation

- [ ] Unit tests
  - Signature verification with valid/invalid signatures
  - Token verification with valid/invalid tokens
  - Middleware authentication logic

- [ ] Integration tests
  - Test YouTube webhook with signature
  - Test management endpoints with auth
  - Test scheduler can still access /renew
  - Test verification challenge still public

- [ ] Manual testing
  - Subscribe to test channel
  - Receive webhook notification
  - Verify signature validation works
  - Test CLI with authenticated user
  - Test CLI without authentication (should fail)

### Phase 4: Documentation & Deployment

- [ ] Update documentation
  - Architecture docs with security model
  - API reference with auth requirements
  - Deployment guide with Secret Manager setup
  - Operations guide for secret rotation

- [ ] Deployment checklist
  - Create secret in Secret Manager
  - Update Terraform configuration
  - Deploy function with new code
  - Test all endpoints
  - Monitor for authentication failures

## Configuration

### Environment Variables

```bash
# New variables to add
WEBHOOK_SECRET_NAME=youtube-webhook-secret-production
REQUIRE_AUTH=true  # Feature flag for gradual rollout
```

### Secret Manager

**Secret Name:** `youtube-webhook-secret-${environment}`

**Generation:**
```bash
# Generate 32-byte random secret
openssl rand -hex 32
```

**Storage:**
```bash
# Store in Secret Manager
echo -n "YOUR_SECRET_HERE" | gcloud secrets create youtube-webhook-secret-production \
  --data-file=- \
  --replication-policy="automatic" \
  --project=youtube-webhook-handler
```

## Security Considerations

### Signature Verification
- **Constant-time comparison:** Prevent timing attacks
- **Secret rotation:** Document rotation procedure
- **Secret scope:** Different secrets per environment
- **Logging:** Log verification failures for monitoring

### IAM Authentication
- **Token validation:** Use official Google libraries
- **Scope verification:** Ensure token is for correct project
- **Audience validation:** Verify token audience matches function URL
- **Error handling:** Don't leak information in error messages

### Rate Limiting (Future Enhancement)
- Consider adding rate limiting per IP/token
- Prevent abuse even with valid credentials
- Use Cloud Armor or custom middleware

## Migration Strategy

### Phase 1: Add Signature Verification (Breaking Change)
1. Deploy code with signature verification
2. Existing subscriptions will fail until re-subscribed
3. Re-subscribe to all channels with new secret

### Phase 2: Add IAM Auth (Breaking Change)
1. Deploy code with IAM requirement
2. Update CLI with auth support
3. Notify users of authentication requirement
4. Remove `allUsers` from IAM policy after code deployment

### Rollback Plan
- Keep feature flags to disable auth if needed
- Document rollback steps in deployment guide
- Monitor error rates after deployment

## Testing Strategy

### Local Testing
```bash
# Test signature verification
curl -X POST http://localhost:8080/ \
  -H "X-Hub-Signature: sha1=$(echo -n 'test body' | openssl dgst -sha1 -hmac 'secret' | cut -d' ' -f2)" \
  -d 'test body'

# Test IAM auth
TOKEN=$(gcloud auth print-identity-token)
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/subscriptions
```

### Production Testing
- Subscribe to test YouTube channel
- Trigger webhook notification
- Verify signature validation
- Test all CLI commands with auth

## Success Criteria

- [ ] YouTube webhooks validate signatures correctly
- [ ] Management endpoints require authentication
- [ ] Cloud Scheduler can still trigger renewals
- [ ] Verification challenges remain public
- [ ] CLI works with authenticated users
- [ ] Unauthorized requests are rejected
- [ ] All tests pass
- [ ] Documentation is complete
- [ ] No disruption to existing functionality

## Open Questions

1. **Secret rotation:** How often should we rotate the webhook secret?
   - Impacts existing subscriptions
   - Need to re-subscribe all channels after rotation

2. **Grace period:** Should we allow a grace period for migration?
   - Consider feature flag approach
   - Log warnings before enforcing

3. **Alternative auth:** Should we support API keys as alternative to IAM?
   - Simpler for some use cases
   - More complexity to maintain

## References

- [PubSubHubbub Authentication](https://pubsubhubbub.github.io/PubSubHubbub/pubsubhubbub-core-0.4.html#authednotify)
- [Google Cloud IAM Authentication](https://cloud.google.com/functions/docs/securing/authenticating)
- [Google Secret Manager](https://cloud.google.com/secret-manager/docs)
- [HMAC Signature Verification](https://en.wikipedia.org/wiki/HMAC)

---

**Status:** Design Phase
**Priority:** High
**Complexity:** Medium
**Breaking Change:** Yes (requires re-subscription)
