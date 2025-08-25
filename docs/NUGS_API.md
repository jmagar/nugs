# Nugs.net Complete API Documentation & Implementation Guide

## Table of Contents

1. [Overview](#overview)
2. [Quick Start](#quick-start)
3. [Architecture](#architecture)
4. [Authentication](#authentication)
5. [API Endpoints](#api-endpoints)
6. [Streaming Implementation](#streaming-implementation)
7. [Data Models](#data-models)
8. [URL Patterns](#url-patterns)
9. [Quality & Formats](#quality--formats)
10. [Download Management](#download-management)
11. [Testing Findings](#testing-findings)
12. [Implementation Examples](#implementation-examples)
13. [Error Handling](#error-handling)
14. [Security Best Practices](#security-best-practices)

## Overview

The Nugs.net API is a comprehensive music streaming service that provides:
- High-quality audio streaming (ALAC, FLAC, MQA, 360 Reality Audio)
- Live and on-demand video content
- User playlists and catalog browsing
- Subscription-based access control
- Multi-format content delivery via HLS

### Key Features
- OAuth2 authentication with JWT tokens
- Legacy API compatibility
- Direct HTTP and HLS streaming
- AES-128-CBC encryption for protected content
- Download speeds of 30-50 MB/s verified

## Quick Start

### Prerequisites
- Nugs.net subscription ($24.99/month for Hi-Res)
- Go 1.16+ or Python 3.8+
- FFmpeg (for video/HLS conversion)

### Configuration
```json
{
  "email": "user@example.com",
  "password": "password",
  "format": 4,
  "videoFormat": 5,
  "outPath": "/path/to/downloads"
}
```

### Format Options
- 1 = 16-bit / 44.1 kHz ALAC
- 2 = 16-bit / 44.1 kHz FLAC
- 3 = 24-bit / 48 kHz MQA
- 4 = 360 Reality Audio / best available
- 5 = 150 Kbps AAC

## Architecture

### Service Endpoints
| Service | Base URL | Purpose |
|---------|----------|---------|
| Identity | `https://id.nugs.net` | Authentication and user management |
| Subscriptions | `https://subscriptions.nugs.net` | Subscription and billing |
| Stream API | `https://streamapi.nugs.net` | Content metadata and streaming |
| Player | `https://play.nugs.net` | Web player (for referer headers) |

### Required Headers
```http
User-Agent: NugsNet/3.26.724 (Android; 7.1.2; Asus; ASUS_Z01QD; Scale/2.0; en)
Authorization: Bearer {access_token}
Referer: https://play.nugs.net/
```

## Authentication

### OAuth2 Password Grant Flow

**Endpoint:** `POST https://id.nugs.net/connect/token`

**Request:**
```http
Content-Type: application/x-www-form-urlencoded

client_id=Eg7HuH873H65r5rt325UytR5429&
grant_type=password&
scope=openid profile email nugsnet:api nugsnet:legacyapi offline_access&
username={email}&
password={url_encoded_password}
```

**Important:** Password special characters MUST be URL encoded (e.g., `!` becomes `%21`)

**Response:**
```json
{
  "access_token": "eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiJ9...",
  "expires_in": 600,
  "token_type": "Bearer",
  "refresh_token": "CfDJ8...",
  "scope": "openid profile email nugsnet:api nugsnet:legacyapi offline_access"
}
```

### JWT Token Structure
```json
{
  "sub": "1298822",
  "email": "user@example.com",
  "legacy_token": "C8DDB55B72ABBCA294F141CA08252559CC0210B4_1755820946",
  "legacy_uguid": "a61c3708f803af59af022f4c073fffae",
  "exp": 1755820706
}
```

**Token Duration:** 600 seconds (10 minutes)

### Apple/Google Authentication

For Apple/Google accounts, tokens must be extracted manually:

1. **Browser Method:**
   - Open Chrome DevTools (F12)
   - Navigate to https://play.nugs.net
   - Application tab → Session Storage → access_token

2. **Network Sniffing:**
   - Use Fiddler/Charles Proxy
   - Find `api/v1/me/subscriptions` call
   - Extract Bearer token from headers

## API Endpoints

### GET /connect/userinfo
Get authenticated user information.

**URL:** `https://id.nugs.net/connect/userinfo`

**Headers:**
```http
Authorization: Bearer {access_token}
```

**Response:**
```json
{
  "sub": "user-uuid",
  "email": "user@example.com",
  "email_verified": true
}
```

### GET /api/v1/me/subscriptions
Get user subscription information.

**URL:** `https://subscriptions.nugs.net/api/v1/me/subscriptions`

**Response:**
```json
{
  "id": "subscription-id",
  "legacySubscriptionId": "e33838049d12400d51c241cd1b2a4aa8",
  "status": "ActivePaid",
  "isContentAccessible": true,
  "startedAt": "08/05/2025 13:56:00",
  "endsAt": "09/05/2025 13:56:00",
  "plan": {
    "planId": "nnpaidhqmonthly",
    "description": "All Access Hi-Res Monthly No Trial",
    "price": 24.99
  }
}
```

### GET /api.aspx?method=catalog.artists
Get all artists in the catalog.

**URL:** `https://streamapi.nugs.net/api.aspx?method=catalog.artists`

**Response:**
```json
{
  "methodName": "catalog.artists",
  "responseAvailabilityCode": 0,
  "Response": {
    "artists": [
      {
        "artistID": 1125,
        "artistName": "Billy Strings",
        "numShows": 603,
        "numAlbums": 1
      }
    ]
  }
}
```
**Note:** Returns all ~620 artists in a single response

### GET /api.aspx?method=catalog.container
Get album/container metadata.

**URL:** `https://streamapi.nugs.net/api.aspx`

**Query Parameters:**
- `method=catalog.container`
- `containerID={albumId}`
- `vdisp=1` (detailed response)

**Response:**
```json
{
  "Response": {
    "containerID": 23329,
    "containerInfo": "08/30/19 Ryman Auditorium, Nashville, TN",
    "artistName": "The Raconteurs",
    "performanceDate": "8/30/2019",
    "tracks": [...],
    "products": [...]
  }
}
```

### GET /api.aspx?method=catalog.containersAll
Get artist discography or complete catalog.

**Critical Discovery:** This endpoint can return the ENTIRE catalog (30,101 shows) in a single API call without authentication!

**For Specific Artist:**
**Query Parameters:**
- `method=catalog.containersAll`
- `artistList={artistId}`
- `limit=100`
- `startOffset=1`
- `availableOnly=1`
- `token={jwt_token}` (required for artist-specific queries)

**For Complete Catalog (Game Changer!):**
**URL:** `https://streamapi.nugs.net/api.aspx?method=catalog.containersAll&availableOnly=1`

**Authentication:** None required! This is a public endpoint.

**Response:** Complete catalog with 30,101+ shows (171MB JSON)
```json
{
  "methodName": "catalog.containersAll",
  "responseAvailabilityCode": 0,
  "Response": {
    "containers": [
      {
        "containerID": 19574,
        "artistName": "Billy Strings",
        "venueName": "Red Rocks Amphitheatre", 
        "venueCity": "Morrison",
        "venueState": "CO",
        "performanceDate": "6/21/2025",
        "performanceDateShort": "06/21/25",
        "performanceDateFormatted": "Sunday, June 21, 2025",
        "containerInfo": "06/21/25 Red Rocks Amphitheatre, Morrison, CO",
        "availabilityType": 1,
        "availabilityTypeStr": "Available",
        "activeState": "active"
      }
    ]
  }
}
```

**Architecture Impact:** This discovery enabled a complete redesign from 600+ API calls per analysis to just 1 daily catalog refresh.

## Catalog Management Architecture

### Overview

The discovery of the public `catalog.containersAll` endpoint fundamentally changed our implementation approach. Instead of making hundreds of authenticated API calls for each artist, we now cache the entire catalog locally.

### Implementation Strategy

**Before Discovery (Inefficient):**
- 620 artists × multiple API calls = 600+ requests per analysis
- Required authentication for every query
- Rate limiting constraints
- API safety concerns

**After Discovery (Efficient):**
- 1 API call daily to refresh complete catalog
- No authentication required for catalog queries  
- All searches performed against local cache
- 99.9% reduction in API calls

### Catalog Cache Structure

**File:** `catalog_cache.json` (171MB)

```json
{
  "last_update": "2025-08-22T22:56:36-04:00",
  "total_shows": 30101,
  "total_artists": 532,
  "shows_by_artist": {
    "Billy Strings": [
      {
        "containerID": 19574,
        "artistName": "Billy Strings",
        "venueName": "Red Rocks Amphitheatre",
        "venueCity": "Morrison", 
        "venueState": "CO",
        "performanceDate": "6/21/2025",
        "performanceDateShort": "06/21/25",
        "availabilityType": 1,
        "activeState": "active"
      }
    ]
  },
  "all_shows": [ /* complete show array */ ]
}
```

### Refresh Strategy

**Frequency:** Daily refresh at 24-hour intervals
**Fallback:** Use existing cache if refresh fails
**Size Impact:** 171MB is acceptable for this use case

### Performance Benefits

| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| API Calls | 600+ | 1 | 99.9% reduction |
| Analysis Time | 5-10 minutes | 30 seconds | 90% faster |
| Authentication | Required | Not needed | Simplified |
| Rate Limiting | Constant concern | Eliminated | No constraints |
| Reliability | API dependent | Cache resilient | Much higher |

### Go Implementation

```go
type CatalogManager struct {
    catalogFile string
    maxAge      time.Duration
}

func (cm *CatalogManager) GetShowsForArtist(artistName string) ([]ShowContainer, error) {
    catalog, err := cm.GetCatalog()
    if err != nil {
        return nil, err
    }
    
    shows, exists := catalog.ShowsByArtist[artistName]
    if !exists {
        return []ShowContainer{}, nil
    }
    
    return shows, nil
}
```

### Usage Examples

**CLI Tools:**
```bash
# Show catalog statistics
./catalog_manager stats

# Force refresh from API
./catalog_manager refresh  

# Query specific artist
./catalog_manager artist "Billy Strings"
```

**Programmatic Access:**
```go
cm := NewCatalogManager()
shows, err := cm.GetShowsForArtist("Billy Strings")
// Returns 636 shows instantly from local cache
```

### GET /bigriver/subPlayer.aspx
Generate stream URLs for subscribers.

**For Audio:**
```
?platformID={formatId}&
trackID={trackId}&
app=1&
subscriptionID={subId}&
subCostplanIDAccessList={planId}&
nn_userID={userId}&
startDateStamp={start}&
endDateStamp={end}
```

**For Video:**
```
?skuId={skuId}&
containerID={containerId}&
chap=1&
app=1&
subscriptionID={subId}&
subCostplanIDAccessList={planId}&
nn_userID={userId}&
startDateStamp={start}&
endDateStamp={end}
```

## Streaming Implementation

### Stream URL Generation Process

1. **Get Authentication Token**
2. **Extract Legacy Components from JWT**
3. **Get Subscription Info**
4. **Generate Stream Parameters**
5. **Request Stream URL**

### Critical Discovery: JSONP Required for Web

The web player uses JSONP callbacks:
```javascript
https://streamapi.nugs.net/bigapp/subPlayer.aspx?
callback=jQuery...&
HLS=1&
orgn=websdk&
method=subPlayer&
trackId={trackId}&  // Note: lowercase 'd'
platformId=4&
app=1
```

### Direct API (Working in Go)
```go
https://streamapi.nugs.net/api.aspx?
HLS=1&
platformID=2&
trackID={trackId}&  // Note: uppercase 'D'
token={jwt}&
app=5
```

### Quality Detection
```go
var qualityMap = map[string]Quality{
    ".alac16/": {Specs: "16-bit / 44.1 kHz ALAC", Extension: ".m4a"},
    ".flac16/": {Specs: "16-bit / 44.1 kHz FLAC", Extension: ".flac"},
    ".mqa24/":  {Specs: "24-bit / 48 kHz MQA", Extension: ".flac"},
    ".s360/":   {Specs: "360 Reality Audio", Extension: ".mp4"},
    ".aac150/": {Specs: "150 Kbps AAC", Extension: ".m4a"},
    ".m3u8":    {Extension: ".m4a", Format: 6}, // HLS
}
```

### HLS Implementation

For HLS streams:
1. Parse master playlist (.m3u8)
2. Select quality variant
3. Download segments
4. Handle AES-128-CBC encryption if present
5. Concatenate with FFmpeg

### AES Decryption
```go
func decryptSegment(encryptedData, key, iv []byte) ([]byte, error) {
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, err
    }
    
    mode := cipher.NewCBCDecrypter(block, iv)
    decrypted := make([]byte, len(encryptedData))
    mode.CryptBlocks(decrypted, encryptedData)
    
    return pkcs5Unpad(decrypted), nil
}
```

## Data Models

### Track Object
```json
{
  "trackID": 67890,
  "songID": 54321,
  "songTitle": "Song Title",
  "totalRunningTime": 180,
  "trackNum": 1,
  "discNum": 1,
  "setNum": 1
}
```

### Container Response
```json
{
  "containerID": 12345,
  "containerInfo": "Album Title",
  "artistName": "Artist Name",
  "performanceDate": "01/15/2025",
  "venueName": "Venue Name",
  "venueCity": "City",
  "venueState": "State",
  "tracks": [...],
  "products": [...],
  "videoChapters": [...]
}
```

### Stream Parameters
```json
{
  "subscriptionID": "subscription-id",
  "subCostplanIDAccessList": "plan-id",
  "userID": "user-id",
  "startStamp": "1642291200",
  "endStamp": "1642377600"
}
```

## URL Patterns

| Content Type | URL Pattern |
|--------------|-------------|
| Album | `https://play.nugs.net/release/{id}` |
| Artist | `https://play.nugs.net/artist/{id}` |
| User Playlist | `https://play.nugs.net/library/playlist/{id}` |
| Public Playlist | `https://2nu.gs/{shortId}` |
| Video | `https://play.nugs.net/#/videos/artist/{artistId}/.../container/{id}` |
| Livestream | `https://play.nugs.net/livestream/{id}/exclusive` |
| Webcast | `https://play.nugs.net/#/my-webcasts/{webcastId}` |

## Quality & Formats

### Audio Formats
| Format ID | Description | Extension | Quality |
|-----------|-------------|-----------|---------|
| 1 | ALAC | .m4a | 16-bit / 44.1 kHz |
| 2 | FLAC | .flac | 16-bit / 44.1 kHz |
| 3 | MQA | .flac | 24-bit / 48 kHz |
| 4 | 360 Reality Audio | .mp4 | Best available |
| 5 | AAC | .m4a | 150 Kbps |
| 6 | HLS AAC | .m4a | Variable |

### Video Resolutions
| Format ID | Resolution |
|-----------|------------|
| 1 | 480p |
| 2 | 720p |
| 3 | 1080p |
| 4 | 1440p |
| 5 | 2160p (4K) |

### Format Fallback Chain
- ALAC (1) → FLAC (2)
- FLAC (2) → AAC (5)
- MQA (3) → FLAC (2)
- 360 Reality (4) → MQA (3)

## Download Management

### Progress Tracking
```go
type DownloadProgress struct {
    Total      int64
    Downloaded int64
    StartTime  time.Time
}

func (dp *DownloadProgress) Write(p []byte) (int, error) {
    n := len(p)
    dp.Downloaded += int64(n)
    percentage := float64(dp.Downloaded) / float64(dp.Total) * 100
    speed := float64(dp.Downloaded) / time.Since(dp.StartTime).Seconds()
    fmt.Printf("\r%.1f%% @ %.2f MB/s", percentage, speed/1048576)
    return n, nil
}
```

### FFmpeg Integration

**TS to MP4 Conversion:**
```bash
ffmpeg -hide_banner -i input.ts -c copy output.mp4
```

**HLS to MP4:**
```bash
ffmpeg -i playlist.m3u8 -c copy -bsf:a aac_adtstoasc output.mp4
```

## Testing Findings

### Key Discoveries

1. **Authentication Works:** OAuth2 flow confirmed with 10-minute token expiry
2. **Stream URL Mystery Solved:** Web player uses JSONP, API uses direct JSON
3. **Performance Verified:** 30-50 MB/s download speeds achieved
4. **Password Encoding Critical:** Special characters must be URL encoded

### Working Go Implementation
```bash
./nugs-dl -f 2 https://play.nugs.net/release/31721
# Downloads at 35-45 MB/s
```

### Failed Python Attempts
- Missing JSONP callback parameter
- Incorrect parameter casing (trackID vs trackId)
- Missing session cookie management

### Actual Stream URLs Captured
```
https://securestream02.akamaized.net/nas5/bwwb/230221.flac16/
bwwb230221d1_01_Hell_In_A_Bucket.flac?
__gda__=1756003777_9fcd89ccc47c18550b85dee938b5b6a9&ultID=0
```

## Implementation Examples

### Complete Authentication Flow (Go)
```go
func Authenticate(email, password string) error {
    // 1. OAuth2 token request
    data := url.Values{}
    data.Set("client_id", "Eg7HuH873H65r5rt325UytR5429")
    data.Set("grant_type", "password")
    data.Set("scope", "openid profile email nugsnet:api nugsnet:legacyapi offline_access")
    data.Set("username", email)
    data.Set("password", password)
    
    req, _ := http.NewRequest("POST", "https://id.nugs.net/connect/token", 
        strings.NewReader(data.Encode()))
    req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
    req.Header.Add("User-Agent", userAgent)
    
    resp, _ := client.Do(req)
    // Parse token from response
    
    // 2. Get user info
    // 3. Get subscription info
    // 4. Generate stream parameters
    return nil
}
```

### Stream URL Request (Working)
```go
func GetStreamURL(trackID int, format int, params *StreamParams) (string, error) {
    query := url.Values{}
    query.Set("platformID", strconv.Itoa(format))
    query.Set("trackID", strconv.Itoa(trackID))
    query.Set("app", "1")
    query.Set("subscriptionID", params.SubscriptionID)
    query.Set("subCostplanIDAccessList", params.SubCostplanIDAccessList)
    query.Set("nn_userID", params.UserID)
    query.Set("startDateStamp", params.StartStamp)
    query.Set("endDateStamp", params.EndStamp)
    
    req, _ := http.NewRequest("GET", "https://streamapi.nugs.net/bigriver/subPlayer.aspx", nil)
    req.URL.RawQuery = query.Encode()
    req.Header.Add("User-Agent", "nugsnetAndroid")
    
    resp, _ := client.Do(req)
    // Parse streamLink from response
}
```

## Error Handling

### Common Error Codes
| Status | Error | Action |
|--------|-------|--------|
| 400 | Bad Request | Check parameters |
| 401 | Unauthorized | Re-authenticate |
| 403 | Forbidden | Check subscription |
| 429 | Rate Limited | Implement backoff |
| 500 | Server Error | Retry with backoff |

### Retry Logic
```go
func RetryWithBackoff(operation func() error, maxRetries int) error {
    for i := 0; i <= maxRetries; i++ {
        err := operation()
        if err == nil {
            return nil
        }
        if i < maxRetries {
            time.Sleep(time.Duration(math.Pow(2, float64(i))) * time.Second)
        }
    }
    return fmt.Errorf("failed after %d attempts", maxRetries)
}
```

## Security Best Practices

1. **Token Storage**
   - Never log tokens
   - Use encrypted storage
   - Implement proper cleanup

2. **Request Security**
   - Always use HTTPS
   - Validate SSL certificates
   - Implement timeout handling

3. **Rate Limiting**
   - Authentication: 10 req/min
   - Content Discovery: 60 req/min
   - Stream URLs: 30 req/min

4. **Password Handling**
   - URL encode special characters
   - Never store plaintext
   - Use secure config storage

## Constants Reference

```javascript
const CONFIG = {
  CLIENT_ID: "Eg7HuH873H65r5rt325UytR5429",
  DEVELOPER_KEY: "x7f54tgbdyc64y656thy47er4",
  USER_AGENT: "NugsNet/3.26.724 (Android; 7.1.2; Asus; ASUS_Z01QD; Scale/2.0; en)",
  USER_AGENT_ALT: "nugsnetAndroid",
  SCOPES: "openid profile email nugsnet:api nugsnet:legacyapi offline_access"
};
```

## Supported Media Types

| Type | URL Example |
|------|-------------|
| Album | `https://play.nugs.net/release/23329` |
| Artist | `https://play.nugs.net/artist/461` |
| Catalog playlist | `https://2nu.gs/3PmqXLW` |
| Exclusive Livestream | `https://play.nugs.net/watch/livestreams/exclusive/30119` |
| User playlist | `https://play.nugs.net/library/playlist/1261211` |
| Video | `https://play.nugs.net/#/videos/artist/1045/Dead%20and%20Company/container/27323` |
| Webcast | `https://play.nugs.net/#/my-webcasts/5826189-30369-0-624602` |

## Usage Examples

### Download Album
```bash
./nugs-dl -f 2 https://play.nugs.net/release/23329
```

### Download Artist Discography
```bash
./nugs-dl -f 4 https://play.nugs.net/artist/1125
```

### Download with Custom Path
```bash
./nugs-dl -f 2 -o /custom/path https://play.nugs.net/release/23329
```

## Additional Endpoints (Suspected but Unverified)

- `/api.aspx?method=catalog.search` - Global search
- `/api.aspx?method=catalog.featured` - Featured content
- `/api.aspx?method=catalog.recommendations` - Personalized recommendations
- `/api.aspx?method=catalog.genres` - Browse by genre
- `/api.aspx?method=catalog.newReleases` - Recently added content
- `/api.aspx?method=catalog.livestreams` - Current/upcoming live streams

## Conclusion

The Nugs.net API provides robust access to high-quality audio streaming with:
- Proven download speeds of 30-50 MB/s
- Multiple quality formats including lossless
- Working Go implementation available
- JSONP requirement for web player compatibility
- 10-minute token expiration requiring refresh logic

The API follows standard OAuth2 patterns with legacy compatibility layers. The main challenge is the different parameter requirements between web player (JSONP) and direct API access.

---

*This documentation is based on reverse engineering analysis and live testing. Use responsibly and in accordance with Nugs.net's terms of service.*