# Nugs.net Authentication Guide

This guide provides detailed implementation instructions for authenticating with the Nugs.net API.

## Table of Contents

1. [Authentication Overview](#authentication-overview)
2. [OAuth2 Password Grant](#oauth2-password-grant)
3. [Token Management](#token-management)
4. [Apple/Google Authentication](#applegoogle-authentication)
5. [Subscription Validation](#subscription-validation)
6. [Implementation Examples](#implementation-examples)
7. [Error Handling](#error-handling)
8. [Security Best Practices](#security-best-practices)

## Authentication Overview

Nugs.net uses OAuth2 with two authentication methods:

1. **Email/Password Authentication** - Standard OAuth2 password grant flow
2. **Apple/Google Authentication** - Requires manual token extraction

Both methods result in JWT access tokens containing:
- User identification
- Legacy authentication components
- Subscription access parameters

## OAuth2 Password Grant

### Required Constants

```go
const (
    clientId  = "Eg7HuH873H65r5rt325UytR5429"
    devKey    = "x7f54tgbdyc64y656thy47er4"
    authUrl   = "https://id.nugs.net/connect/token"
    userAgent = "NugsNet/3.26.724 (Android; 7.1.2; Asus; ASUS_Z01QD; Scale/2.0; en)"
)
```

### Authentication Request

```http
POST https://id.nugs.net/connect/token
Content-Type: application/x-www-form-urlencoded
User-Agent: NugsNet/3.26.724 (Android; 7.1.2; Asus; ASUS_Z01QD; Scale/2.0; en)

client_id=Eg7HuH873H65r5rt325UytR5429&
grant_type=password&
scope=openid profile email nugsnet:api nugsnet:legacyapi offline_access&
username={email}&
password={password}
```

### Response Structure

```json
{
  "access_token": "eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiIsImtpZCI6IjEifQ...",
  "expires_in": 36000,
  "token_type": "Bearer",
  "refresh_token": "CfDJ8...",
  "scope": "openid profile email nugsnet:api nugsnet:legacyapi offline_access"
}
```

### Implementation Example (Go)

```go
func authenticate(email, password string) (string, error) {
    data := url.Values{}
    data.Set("client_id", clientId)
    data.Set("grant_type", "password")
    data.Set("scope", "openid profile email nugsnet:api nugsnet:legacyapi offline_access")
    data.Set("username", email)
    data.Set("password", password)
    
    req, err := http.NewRequest(http.MethodPost, authUrl, strings.NewReader(data.Encode()))
    if err != nil {
        return "", err
    }
    
    req.Header.Add("User-Agent", userAgent)
    req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
    
    resp, err := client.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return "", errors.New(resp.Status)
    }
    
    var authResp struct {
        AccessToken string `json:"access_token"`
    }
    
    err = json.NewDecoder(resp.Body).Decode(&authResp)
    if err != nil {
        return "", err
    }
    
    return authResp.AccessToken, nil
}
```

## Token Management

### JWT Token Structure

The access token is a JWT with this payload structure:

```json
{
  "nbf": 1642291200,
  "exp": 1642327200,
  "iss": "https://id.nugs.net",
  "aud": ["nugsnet:api", "nugsnet:legacyapi"],
  "client_id": "Eg7HuH873H65r5rt325UytR5429",
  "sub": "user-uuid",
  "auth_time": 1642291200,
  "email": "user@example.com",
  "legacy_token": "legacy-auth-token",
  "legacy_uguid": "legacy-user-guid",
  "scope": ["openid", "profile", "email", "nugsnet:api", "nugsnet:legacyapi", "offline_access"]
}
```

### Extracting Legacy Components

```go
func extractLegacyToken(tokenStr string) (string, string, error) {
    // Split JWT parts
    parts := strings.SplitN(tokenStr, ".", 3)
    if len(parts) != 3 {
        return "", "", errors.New("invalid JWT format")
    }
    
    // Decode payload (base64url)
    payload, err := base64.RawURLEncoding.DecodeString(parts[1])
    if err != nil {
        return "", "", err
    }
    
    var tokenPayload struct {
        LegacyToken string `json:"legacy_token"`
        LegacyUguid string `json:"legacy_uguid"`
    }
    
    err = json.Unmarshal(payload, &tokenPayload)
    if err != nil {
        return "", "", err
    }
    
    return tokenPayload.LegacyToken, tokenPayload.LegacyUguid, nil
}
```

### User Information Retrieval

```go
func getUserInfo(token string) (string, error) {
    req, err := http.NewRequest(http.MethodGet, "https://id.nugs.net/connect/userinfo", nil)
    if err != nil {
        return "", err
    }
    
    req.Header.Add("Authorization", "Bearer "+token)
    req.Header.Add("User-Agent", userAgent)
    
    resp, err := client.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return "", errors.New(resp.Status)
    }
    
    var userInfo struct {
        Sub string `json:"sub"`
    }
    
    err = json.NewDecoder(resp.Body).Decode(&userInfo)
    if err != nil {
        return "", err
    }
    
    return userInfo.Sub, nil
}
```

## Apple/Google Authentication

For accounts created via Apple or Google OAuth:

### Method 1: Browser Developer Tools

1. Open Chrome Developer Tools (`Ctrl+Shift+I`)
2. Navigate to `https://play.nugs.net`
3. Go to `Application` tab → `Session Storage` → `https://play.nugs.net/`
4. Copy the `access_token` value

### Method 2: Network Sniffing

1. Start Fiddler Classic or similar proxy tool
2. Navigate to `https://www.nugs.net/stream/music/`
3. Find the `api/v1/me/subscriptions` call
4. Extract `Authorization: Bearer {token}` header

### Token Duration

- Apple/Google tokens last **600 minutes** (10 hours)
- Must be manually refreshed when expired
- No automatic refresh mechanism available

## Subscription Validation

### Subscription Info Request

```go
func getSubscriptionInfo(token string) (*SubscriptionInfo, error) {
    req, err := http.NewRequest(http.MethodGet, 
        "https://subscriptions.nugs.net/api/v1/me/subscriptions", nil)
    if err != nil {
        return nil, err
    }
    
    req.Header.Add("Authorization", "Bearer "+token)
    req.Header.Add("User-Agent", userAgent)
    
    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return nil, errors.New(resp.Status)
    }
    
    var subInfo SubscriptionInfo
    err = json.NewDecoder(resp.Body).Decode(&subInfo)
    if err != nil {
        return nil, err
    }
    
    return &subInfo, nil
}
```

### Stream Parameters Generation

```go
func parseStreamParams(userId string, subInfo *SubscriptionInfo, isPromo bool) *StreamParams {
    startStamp, endStamp := parseTimestamps(subInfo.StartedAt, subInfo.EndsAt)
    
    params := &StreamParams{
        SubscriptionID: subInfo.LegacySubscriptionID,
        UserID:        userId,
        StartStamp:    startStamp,
        EndStamp:      endStamp,
    }
    
    if isPromo {
        params.SubCostplanIDAccessList = subInfo.Promo.Plan.PlanID
    } else {
        params.SubCostplanIDAccessList = subInfo.Plan.PlanID
    }
    
    return params
}

func parseTimestamps(start, end string) (string, string) {
    layout := "01/02/2006 15:04:05"
    startTime, _ := time.Parse(layout, start)
    endTime, _ := time.Parse(layout, end)
    
    startStamp := strconv.FormatInt(startTime.Unix(), 10)
    endStamp := strconv.FormatInt(endTime.Unix(), 10)
    
    return startStamp, endStamp
}
```

## Implementation Examples

### Complete Authentication Flow

```go
type AuthClient struct {
    httpClient *http.Client
    token      string
    userID     string
    subInfo    *SubscriptionInfo
    streamParams *StreamParams
}

func NewAuthClient() *AuthClient {
    jar, _ := cookiejar.New(nil)
    return &AuthClient{
        httpClient: &http.Client{Jar: jar},
    }
}

func (c *AuthClient) Authenticate(email, password string) error {
    // Step 1: Get access token
    token, err := c.authenticate(email, password)
    if err != nil {
        return fmt.Errorf("authentication failed: %w", err)
    }
    c.token = token
    
    // Step 2: Get user info
    userID, err := c.getUserInfo(token)
    if err != nil {
        return fmt.Errorf("failed to get user info: %w", err)
    }
    c.userID = userID
    
    // Step 3: Get subscription info
    subInfo, err := c.getSubscriptionInfo(token)
    if err != nil {
        return fmt.Errorf("failed to get subscription: %w", err)
    }
    c.subInfo = subInfo
    
    // Step 4: Validate subscription
    if !subInfo.IsContentAccessible {
        return errors.New("no active subscription")
    }
    
    // Step 5: Generate stream parameters
    _, isPromo := c.getPlan(subInfo)
    c.streamParams = parseStreamParams(userID, subInfo, isPromo)
    
    return nil
}

func (c *AuthClient) getPlan(subInfo *SubscriptionInfo) (string, bool) {
    if subInfo.Plan.Description != "" {
        return subInfo.Plan.Description, false
    }
    return subInfo.Promo.Plan.Description, true
}
```

## Error Handling

### Common Error Responses

| Status Code | Description | Action |
|-------------|-------------|---------|
| 400 | Bad Request | Check request parameters |
| 401 | Unauthorized | Invalid credentials |
| 403 | Forbidden | Account locked or suspended |
| 429 | Too Many Requests | Implement rate limiting |
| 500 | Internal Server Error | Retry with exponential backoff |

### Subscription Status Validation

```go
func validateSubscription(subInfo *SubscriptionInfo) error {
    if !subInfo.IsContentAccessible {
        return errors.New("subscription not accessible")
    }
    
    endTime, err := time.Parse("01/02/2006 15:04:05", subInfo.EndsAt)
    if err != nil {
        return err
    }
    
    if time.Now().After(endTime) {
        return errors.New("subscription expired")
    }
    
    return nil
}
```

## Security Best Practices

### Token Storage

- **Never log tokens** - they contain personal information
- Store tokens securely (encrypted storage, secure keychain)
- Implement proper token cleanup on logout

### Request Security

- Always use HTTPS for API requests
- Validate SSL certificates
- Implement request timeout handling

### Rate Limiting

```go
type RateLimiter struct {
    lastRequest time.Time
    minInterval time.Duration
}

func NewRateLimiter(requestsPerSecond float64) *RateLimiter {
    return &RateLimiter{
        minInterval: time.Duration(float64(time.Second) / requestsPerSecond),
    }
}

func (rl *RateLimiter) Wait() {
    elapsed := time.Since(rl.lastRequest)
    if elapsed < rl.minInterval {
        time.Sleep(rl.minInterval - elapsed)
    }
    rl.lastRequest = time.Now()
}
```

### Error Logging

```go
func logAuthError(err error, context string) {
    // Log error without exposing sensitive data
    log.Printf("Auth error in %s: %v", context, err)
    // Don't log tokens, passwords, or personal info
}
```

---

*This authentication guide is based on analysis of the Nugs-Downloader implementation. Always handle user credentials securely and in compliance with relevant privacy laws.*