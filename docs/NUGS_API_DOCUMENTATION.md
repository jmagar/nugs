# Nugs.net API Documentation

This document provides comprehensive documentation for the Nugs.net streaming API, based on analysis of the Nugs-Downloader implementation.

## Table of Contents

1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Authentication](#authentication)
4. [Content Discovery](#content-discovery)
5. [Streaming Protocol](#streaming-protocol)
6. [Data Models](#data-models)
7. [URL Patterns](#url-patterns)
8. [Quality & Formats](#quality--formats)
9. [Implementation Notes](#implementation-notes)

## Overview

The Nugs.net API is a comprehensive music streaming service that provides:
- High-quality audio streaming (ALAC, FLAC, MQA, 360 Reality Audio)
- Live and on-demand video content
- User playlists and catalog browsing
- Subscription-based access control
- Multi-format content delivery via HLS

## Architecture

The API consists of several microservices:

### Identity Service
- **Base URL:** `https://id.nugs.net`
- **Purpose:** User authentication and profile management
- **Authentication:** OAuth2 with password grant

### Subscription Service
- **Base URL:** `https://subscriptions.nugs.net`
- **Purpose:** Subscription management and access validation
- **Authentication:** Bearer token required

### Stream API Service
- **Base URL:** `https://streamapi.nugs.net`
- **Purpose:** Content metadata and stream URL generation
- **Authentication:** Mixed (public catalog + authenticated streams)

### Player Service
- **Base URL:** `https://play.nugs.net`
- **Purpose:** Frontend reference and web player integration

## Authentication

### OAuth2 Password Grant Flow

```http
POST https://id.nugs.net/connect/token
Content-Type: application/x-www-form-urlencoded

client_id=Eg7HuH873H65r5rt325UytR5429&
grant_type=password&
scope=openid profile email nugsnet:api nugsnet:legacyapi offline_access&
username={email}&
password={password}
```

**Response:**
```json
{
  "access_token": "eyJ...",
  "expires_in": 36000,
  "token_type": "Bearer",
  "refresh_token": "...",
  "scope": "openid profile email nugsnet:api nugsnet:legacyapi offline_access"
}
```

### Token Structure

The access token is a JWT containing:
- `sub`: User ID
- `email`: User email
- `legacy_token`: Legacy API token
- `legacy_uguid`: Legacy user GUID for purchased content
- `exp`: Expiration timestamp (600 minutes)

### Alternative Authentication

For Apple/Google accounts, tokens must be extracted manually:
1. Log into https://play.nugs.net via browser
2. Use browser dev tools or Fiddler to capture the access_token
3. Extract from session storage or API calls

## Content Discovery

### Album Metadata

```http
GET https://streamapi.nugs.net/api.aspx?method=catalog.container&containerID={albumId}&vdisp=1
```

### Artist Discography

```http
GET https://streamapi.nugs.net/api.aspx?method=catalog.containersAll&limit=100&artistList={artistId}&availType=1&vdisp=1&startOffset={offset}
```

### User Playlists

```http
GET https://streamapi.nugs.net/secureApi.aspx?method=user.playlist&playlistID={playlistId}&developerKey=x7f54tgbdyc64y656thy47er4&user={email}&token={legacyToken}
```

### Public Playlists

```http
GET https://streamapi.nugs.net/api.aspx?method=catalog.playlist&plGUID={playlistId}
```

## Streaming Protocol

### Stream URL Generation

```http
GET https://streamapi.nugs.net/bigriver/subPlayer.aspx?
platformID={formatId}&
trackID={trackId}&
app=1&
subscriptionID={subscriptionId}&
subCostplanIDAccessList={planId}&
nn_userID={userId}&
startDateStamp={startStamp}&
endDateStamp={endStamp}
```

### Video Stream URLs

```http
GET https://streamapi.nugs.net/bigriver/subPlayer.aspx?
skuId={skuId}&
containerID={containerId}&
chap=1&
app=1&
subscriptionID={subscriptionId}&
subCostplanIDAccessList={planId}&
nn_userID={userId}&
startDateStamp={startStamp}&
endDateStamp={endStamp}
```

### Purchased Content URLs

```http
GET https://streamapi.nugs.net/bigriver/vidPlayer.aspx?
skuId={skuId}&
showId={showId}&
uguid={userGuid}&
nn_userID={userId}&
app=1
```

## Data Models

### Subscription Information

```json
{
  "id": "subscription-id",
  "legacySubscriptionId": "legacy-id",
  "status": "active",
  "isContentAccessible": true,
  "startedAt": "01/15/2025 12:00:00",
  "endsAt": "02/15/2025 12:00:00",
  "plan": {
    "id": "plan-id",
    "planId": "plan-access-id",
    "description": "Premium Plan",
    "serviceLevel": "premium"
  }
}
```

### Album/Container Metadata

```json
{
  "containerID": 12345,
  "containerInfo": "Album Title",
  "artistName": "Artist Name",
  "performanceDate": "01/15/2025",
  "tracks": [...],
  "products": [...],
  "videoChapters": [...]
}
```

### Track Information

```json
{
  "trackID": 67890,
  "songTitle": "Song Title",
  "songID": 54321,
  "totalRunningTime": 180,
  "trackNum": 1,
  "discNum": 1,
  "setNum": 1
}
```

### Quality Specifications

```json
{
  "specs": "16-bit / 44.1 kHz FLAC",
  "extension": ".flac",
  "format": 2,
  "url": "stream-url"
}
```

## URL Patterns

The API supports these URL patterns for content identification:

1. **Albums:** `https://play.nugs.net/release/{id}`
2. **User Playlists:** `https://play.nugs.net/library/playlist/{id}`
3. **Public Playlists:** `https://2nu.gs/{shortId}`
4. **Videos:** `https://play.nugs.net/#/videos/artist/{artistId}/.../container/{id}`
5. **Artists:** `https://play.nugs.net/artist/{id}`
6. **Livestreams:** `https://play.nugs.net/livestream/{id}/exclusive`
7. **Webcasts:** `https://play.nugs.net/#/my-webcasts/{webcastId}`
8. **Purchased Content:** Complex URLs with SKU and show parameters

## Quality & Formats

### Audio Formats

| Format ID | Description | Extension | Bitrate/Quality |
|-----------|-------------|-----------|-----------------|
| 1 | ALAC | .m4a | 16-bit / 44.1 kHz |
| 2 | FLAC | .flac | 16-bit / 44.1 kHz |
| 3 | MQA | .flac | 24-bit / 48 kHz |
| 4 | 360 Reality Audio | .mp4 | Best available |
| 5 | AAC | .m4a | 150 Kbps |
| 6 | HLS AAC | .m4a | Variable |

### Video Resolutions

| Format ID | Resolution | Description |
|-----------|------------|-------------|
| 1 | 480p | Standard Definition |
| 2 | 720p | HD |
| 3 | 1080p | Full HD |
| 4 | 1440p | 2K |
| 5 | 2160p | 4K/Best Available |

### Format Fallback Chain

When a preferred format is unavailable:
- ALAC (1) → FLAC (2)
- FLAC (2) → AAC (5)
- MQA (3) → FLAC (2)
- 360 Reality (4) → MQA (3)

## Implementation Notes

### User Agent Requirements

The API requires specific user agents:
- **Primary:** `NugsNet/3.26.724 (Android; 7.1.2; Asus; ASUS_Z01QD; Scale/2.0; en)`
- **Secondary:** `nugsnetAndroid`

### Rate Limiting

- No explicit rate limits documented in the implementation
- Recommend implementing reasonable delays between requests

### Error Handling

- Standard HTTP status codes
- Check `responseAvailabilityCode` in JSON responses
- Handle subscription expiration gracefully

### Dependencies

For full implementation, you'll need:
- HTTP client with cookie jar support
- JSON parsing capabilities
- HLS manifest parsing (m3u8)
- AES-128-CBC decryption for encrypted streams
- FFmpeg for format conversion (optional)

### Security Considerations

- Tokens contain personal information - handle securely
- Implement proper token refresh mechanisms
- Validate subscription status before stream access
- Use HTTPS for all requests

---

*This documentation is based on analysis of the open-source Nugs-Downloader implementation and is intended for educational purposes.*