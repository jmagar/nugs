# Nugs.net API Reference

This document provides a complete API reference for all Nugs.net endpoints, parameters, and response formats.

## Table of Contents

1. [Base URLs](#base-urls)
2. [Authentication Endpoints](#authentication-endpoints)
3. [User Information Endpoints](#user-information-endpoints)
4. [Subscription Endpoints](#subscription-endpoints)
5. [Content Discovery Endpoints](#content-discovery-endpoints)
6. [Streaming Endpoints](#streaming-endpoints)
7. [Data Structures](#data-structures)
8. [Error Responses](#error-responses)
9. [Rate Limits](#rate-limits)

## Base URLs

| Service | Base URL | Purpose |
|---------|----------|---------|
| Identity | `https://id.nugs.net` | Authentication and user management |
| Subscriptions | `https://subscriptions.nugs.net` | Subscription and billing |
| Stream API | `https://streamapi.nugs.net` | Content metadata and streaming |
| Player | `https://play.nugs.net` | Web player (for referer headers) |

## Authentication Endpoints

### POST /connect/token

Authenticate user and obtain access token.

**URL:** `https://id.nugs.net/connect/token`

**Headers:**
```http
Content-Type: application/x-www-form-urlencoded
User-Agent: NugsNet/3.26.724 (Android; 7.1.2; Asus; ASUS_Z01QD; Scale/2.0; en)
```

**Body Parameters:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `client_id` | string | Yes | `Eg7HuH873H65r5rt325UytR5429` |
| `grant_type` | string | Yes | `password` |
| `scope` | string | Yes | `openid profile email nugsnet:api nugsnet:legacyapi offline_access` |
| `username` | string | Yes | User email address |
| `password` | string | Yes | User password |

**Success Response (200):**
```json
{
  "access_token": "eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiJ9...",
  "expires_in": 36000,
  "token_type": "Bearer",
  "refresh_token": "CfDJ8...",
  "scope": "openid profile email nugsnet:api nugsnet:legacyapi offline_access"
}
```

**Error Response (400):**
```json
{
  "error": "invalid_grant",
  "error_description": "Invalid username or password."
}
```

## User Information Endpoints

### GET /connect/userinfo

Get authenticated user information.

**URL:** `https://id.nugs.net/connect/userinfo`

**Headers:**
```http
Authorization: Bearer {access_token}
User-Agent: NugsNet/3.26.724 (Android; 7.1.2; Asus; ASUS_Z01QD; Scale/2.0; en)
```

**Success Response (200):**
```json
{
  "sub": "user-uuid-string",
  "preferred_username": "username",
  "name": "Full Name",
  "email": "user@example.com",
  "email_verified": true
}
```

## Subscription Endpoints

### GET /api/v1/me/subscriptions

Get user subscription information.

**URL:** `https://subscriptions.nugs.net/api/v1/me/subscriptions`

**Headers:**
```http
Authorization: Bearer {access_token}
User-Agent: NugsNet/3.26.724 (Android; 7.1.2; Asus; ASUS_Z01QD; Scale/2.0; en)
```

**Success Response (200):**
```json
{
  "id": "subscription-id",
  "legacySubscriptionId": "legacy-sub-id",
  "status": "active",
  "isContentAccessible": true,
  "startedAt": "01/15/2025 12:00:00",
  "endsAt": "02/15/2025 12:00:00",
  "plan": {
    "id": "plan-id",
    "planId": "plan-access-id",
    "description": "Premium Plan",
    "serviceLevel": "premium",
    "price": 19.99,
    "period": 30
  },
  "promo": {
    "promoCode": "PROMO123",
    "plan": {
      "planId": "promo-plan-id",
      "description": "Promotional Plan"
    }
  }
}
```

## Content Discovery Endpoints

### GET /api.aspx?method=catalog.container

Get album or container metadata.

**URL:** `https://streamapi.nugs.net/api.aspx`

**Headers:**
```http
User-Agent: NugsNet/3.26.724 (Android; 7.1.2; Asus; ASUS_Z01QD; Scale/2.0; en)
```

**Query Parameters:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `method` | string | Yes | `catalog.container` |
| `containerID` | integer | Yes | Container/Album ID |
| `vdisp` | integer | No | `1` for detailed response |

**Success Response (200):**
```json
{
  "methodName": "catalog.container",
  "responseAvailabilityCode": 1,
  "responseAvailabilityCodeStr": "AVAILABLE",
  "Response": {
    "containerID": 12345,
    "containerInfo": "Album Title",
    "artistName": "Artist Name",
    "performanceDate": "01/15/2025",
    "tracks": [...],
    "products": [...],
    "videoChapters": [...]
  }
}
```

### GET /api.aspx?method=catalog.containersAll

Get artist discography.

**URL:** `https://streamapi.nugs.net/api.aspx`

**Query Parameters:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `method` | string | Yes | `catalog.containersAll` |
| `artistList` | integer | Yes | Artist ID |
| `limit` | integer | No | Results per page (default: 100) |
| `startOffset` | integer | No | Pagination offset (default: 1) |
| `availType` | integer | No | `1` for available content |
| `vdisp` | integer | No | `1` for detailed response |

**Success Response (200):**
```json
{
  "methodName": "catalog.containersAll",
  "Response": {
    "containers": [...],
    "totalMatchedRecords": 150,
    "artistID": 12345,
    "artistName": "Artist Name"
  }
}
```

### GET /api.aspx?method=catalog.playlist

Get public playlist metadata.

**URL:** `https://streamapi.nugs.net/api.aspx`

**Query Parameters:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `method` | string | Yes | `catalog.playlist` |
| `plGUID` | string | Yes | Playlist GUID |

### GET /secureApi.aspx?method=user.playlist

Get user playlist metadata.

**URL:** `https://streamapi.nugs.net/secureApi.aspx`

**Headers:**
```http
User-Agent: nugsnetAndroid
```

**Query Parameters:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `method` | string | Yes | `user.playlist` |
| `playlistID` | integer | Yes | Playlist ID |
| `developerKey` | string | Yes | `x7f54tgbdyc64y656thy47er4` |
| `user` | string | Yes | User email |
| `token` | string | Yes | Legacy token from JWT |

**Success Response (200):**
```json
{
  "methodName": "user.playlist",
  "Response": {
    "ID": 12345,
    "playListName": "My Playlist",
    "items": [
      {
        "track": {
          "trackID": 67890,
          "songTitle": "Song Title",
          "totalRunningTime": 180
        }
      }
    ],
    "numTracks": 10
  }
}
```

## Streaming Endpoints

### GET /bigriver/subPlayer.aspx

Generate stream URLs for subscribers.

**URL:** `https://streamapi.nugs.net/bigriver/subPlayer.aspx`

**Headers:**
```http
User-Agent: nugsnetAndroid
```

**Query Parameters:**

**For Audio Tracks:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `platformID` | integer | Yes | Format ID (1,4,7,10) |
| `trackID` | integer | Yes | Track ID |
| `app` | integer | Yes | `1` |
| `subscriptionID` | string | Yes | Subscription ID |
| `subCostplanIDAccessList` | string | Yes | Plan access ID |
| `nn_userID` | string | Yes | User ID |
| `startDateStamp` | string | Yes | Subscription start timestamp |
| `endDateStamp` | string | Yes | Subscription end timestamp |

**For Video Content:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `skuId` | integer | Yes | Video SKU ID |
| `containerID` | integer | Yes | Container ID |
| `chap` | integer | No | `1` for chapters |
| `app` | integer | Yes | `1` |
| `subscriptionID` | string | Yes | Subscription ID |
| `subCostplanIDAccessList` | string | Yes | Plan access ID |
| `nn_userID` | string | Yes | User ID |
| `startDateStamp` | string | Yes | Subscription start timestamp |
| `endDateStamp` | string | Yes | Subscription end timestamp |

**Success Response (200):**
```json
{
  "streamLink": "https://content.nugs.net/path/to/stream.m3u8?token=...",
  "streamer": "server-name",
  "userID": "user-id",
  "subContentAccess": 1,
  "stashContentAccess": 1
}
```

### GET /bigriver/vidPlayer.aspx

Generate URLs for purchased video content.

**URL:** `https://streamapi.nugs.net/bigriver/vidPlayer.aspx`

**Headers:**
```http
User-Agent: nugsnetAndroid
```

**Query Parameters:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `skuId` | integer | Yes | Video SKU ID |
| `showId` | string | Yes | Show ID |
| `uguid` | string | Yes | Legacy user GUID |
| `nn_userID` | string | Yes | User ID |
| `app` | integer | Yes | `1` |

**Success Response (200):**
```json
{
  "fileURL": "https://content.nugs.net/path/to/video.m3u8",
  "responseCode": 1
}
```

## Data Structures

### Track Object

```json
{
  "trackID": 67890,
  "songID": 54321,
  "songTitle": "Song Title",
  "totalRunningTime": 180,
  "trackNum": 1,
  "discNum": 1,
  "setNum": 1,
  "trackLabel": "Track Label",
  "hhmmssTotalRunningTime": "00:03:00",
  "products": [...],
  "accessList": [...]
}
```

### Product Object

```json
{
  "skuID": 12345,
  "formatStr": "LIVE HD VIDEO",
  "cost": 1999,
  "costplanID": 100,
  "numDiscs": 1,
  "isSubStreamOnly": 1,
  "liveEventInfo": {
    "isEventLive": false,
    "eventStartDateStr": "01/15/2025 20:00:00",
    "eventEndDateStr": "01/15/2025 23:00:00"
  }
}
```

### Container Response Object

```json
{
  "containerID": 12345,
  "containerInfo": "Album or Show Title",
  "artistName": "Artist Name",
  "artistID": 456,
  "performanceDate": "01/15/2025",
  "performanceDateFormatted": "January 15, 2025",
  "venueName": "Venue Name",
  "venueCity": "City",
  "venueState": "State",
  "tracks": [...],
  "songs": [...],
  "products": [...],
  "videoChapters": [...],
  "totalContainerRunningTime": 7200,
  "hhmmssTotalRunningTime": "02:00:00"
}
```

### Quality Object

```json
{
  "specs": "16-bit / 44.1 kHz FLAC",
  "extension": ".flac",
  "format": 2,
  "url": "stream-url"
}
```

### Stream Parameters Object

```json
{
  "subscriptionID": "subscription-id",
  "subCostplanIDAccessList": "plan-id",
  "userID": "user-id",
  "startStamp": "1642291200",
  "endStamp": "1642377600"
}
```

### Video Chapter Object

```json
{
  "chapterSeconds": 120.5,
  "chaptername": "Chapter Title"
}
```

## Error Responses

### Standard Error Format

```json
{
  "error": "error_code",
  "error_description": "Human readable error description"
}
```

### Common Error Codes

| HTTP Status | Error Code | Description |
|-------------|------------|-------------|
| 400 | `invalid_request` | Missing or invalid parameters |
| 401 | `invalid_grant` | Invalid credentials |
| 401 | `invalid_token` | Expired or invalid access token |
| 403 | `access_denied` | Insufficient permissions |
| 404 | `not_found` | Resource not found |
| 429 | `rate_limit_exceeded` | Too many requests |
| 500 | `internal_error` | Server error |

### API Response Codes

| Code | String | Description |
|------|--------|-------------|
| 1 | `AVAILABLE` | Content is available |
| 2 | `UNAVAILABLE` | Content is not available |
| 3 | `RESTRICTED` | Content is restricted |

## Rate Limits

### Recommended Limits

- **Authentication**: 10 requests per minute
- **Content Discovery**: 60 requests per minute
- **Stream URL Generation**: 30 requests per minute
- **General API**: 100 requests per minute

### Rate Limit Headers

Check for these response headers:
- `X-RateLimit-Limit`: Request limit per window
- `X-RateLimit-Remaining`: Remaining requests
- `X-RateLimit-Reset`: Window reset time

### Best Practices

1. **Implement exponential backoff** for failed requests
2. **Cache responses** when possible
3. **Batch requests** for multiple items
4. **Monitor rate limit headers** and adjust accordingly
5. **Use appropriate user agents** as specified

### Request Headers

Always include these headers:

```http
User-Agent: NugsNet/3.26.724 (Android; 7.1.2; Asus; ASUS_Z01QD; Scale/2.0; en)
```

For secure endpoints:
```http
Authorization: Bearer {access_token}
```

For streaming requests:
```http
Referer: https://play.nugs.net/
```

## Constants Reference

### Client Configuration

```javascript
const CONFIG = {
  CLIENT_ID: "Eg7HuH873H65r5rt325UytR5429",
  DEVELOPER_KEY: "x7f54tgbdyc64y656thy47er4",
  USER_AGENT: "NugsNet/3.26.724 (Android; 7.1.2; Asus; ASUS_Z01QD; Scale/2.0; en)",
  USER_AGENT_ALT: "nugsnetAndroid",
  SCOPES: "openid profile email nugsnet:api nugsnet:legacyapi offline_access"
};
```

### Format Codes

```javascript
const FORMATS = {
  ALAC_16: 1,
  FLAC_16: 2,
  MQA_24: 3,
  REALITY_360: 4,
  AAC_150: 5,
  HLS: 6
};

const VIDEO_FORMATS = {
  "480P": 1,
  "720P": 2,
  "1080P": 3,
  "1440P": 4,
  "4K": 5
};
```

### URL Patterns

```javascript
const URL_PATTERNS = [
  /^https:\/\/play\.nugs\.net\/release\/(\d+)$/,                    // Album
  /^https:\/\/play\.nugs\.net\/#\/playlists\/playlist\/(\d+)$/,    // Playlist
  /^https:\/\/play\.nugs\.net\/library\/playlist\/(\d+)$/,         // Library Playlist
  /^https:\/\/2nu\.gs\/([a-zA-Z\d]+)$/,                           // Short URL
  /^https:\/\/play\.nugs\.net\/#\/videos\/artist\/\d+\/.+\/(\d+)$/, // Video
  /^https:\/\/play\.nugs\.net\/artist\/(\d+)(?:\/albums|\/latest|)$/, // Artist
  /^https:\/\/play\.nugs\.net\/livestream\/(\d+)\/exclusive$/,     // Livestream
  // ... additional patterns
];
```

---

*This API reference is based on analysis of the Nugs-Downloader implementation. Use responsibly and in accordance with Nugs.net's terms of service.*