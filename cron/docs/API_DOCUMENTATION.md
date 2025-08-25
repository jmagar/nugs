# Nugs Collection API - Complete Documentation

## Table of Contents
- [Overview](#overview)
- [Authentication](#authentication)
- [Base URL & Versioning](#base-url--versioning)
- [Response Format](#response-format)
- [Pagination](#pagination)
- [Error Handling](#error-handling)
- [Rate Limiting](#rate-limiting)
- [API Endpoints](#api-endpoints)
  - [Authentication](#authentication-endpoints)
  - [Catalog Management](#catalog-management)
  - [Download Management](#download-management)
  - [Monitoring](#monitoring)
  - [Analytics](#analytics)
  - [Webhooks](#webhooks)
  - [Administration](#administration)
  - [Scheduler](#scheduler)
- [Data Models](#data-models)
- [Background Jobs](#background-jobs)
- [Webhook Events](#webhook-events)
- [Examples](#examples)

## Overview

The Nugs Collection API is a comprehensive REST API for managing music collections, downloads, monitoring, and analytics. It provides enterprise-grade features including background job processing, webhook notifications, user management, and detailed analytics.

### Key Features
- **Catalog Management**: Browse artists, shows, and venues
- **Download Queue**: Manage high-quality audio downloads
- **Monitoring**: Track new shows and releases
- **Analytics**: Comprehensive collection and system analytics
- **Webhooks**: Real-time event notifications
- **Scheduler**: Automated background tasks
- **User Management**: Role-based access control

## Authentication

The API uses JWT (JSON Web Token) based authentication. All protected endpoints require a valid JWT token.

### Obtaining a Token
```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "your_username",
  "password": "your_password"
}
```

### Using the Token
Include the token in the Authorization header for all protected requests:
```http
Authorization: Bearer <your_jwt_token>
```

### Token Expiration
Tokens expire after 24 hours. You'll receive a 401 Unauthorized response when the token expires.

## Base URL & Versioning

- **Base URL**: `http://localhost:8080`
- **API Version**: `v1`
- **Full Base URL**: `http://localhost:8080/api/v1`

## Response Format

All API responses follow a consistent JSON format:

### Success Response
```json
{
  "data": {
    // Response data here
  },
  "pagination": {  // Only for paginated responses
    "page": 1,
    "page_size": 20,
    "total": 150,
    "has_next": true,
    "has_prev": false
  }
}
```

### Error Response
```json
{
  "success": false,
  "error": "Error message here",
  "details": {
    // Additional error details if applicable
  }
}
```

## Pagination

Paginated endpoints support the following query parameters:
- `page`: Page number (default: 1)
- `page_size`: Items per page (default: 20, max: 100)

## Error Handling

### HTTP Status Codes
- `200 OK`: Successful request
- `201 Created`: Resource created successfully
- `400 Bad Request`: Invalid request parameters
- `401 Unauthorized`: Authentication required or failed
- `403 Forbidden`: Insufficient permissions
- `404 Not Found`: Resource not found
- `409 Conflict`: Resource conflict
- `422 Unprocessable Entity`: Validation errors
- `500 Internal Server Error`: Server error

### Error Response Format
```json
{
  "success": false,
  "error": "Human readable error message",
  "code": "ERROR_CODE",
  "details": {
    "field": "Additional context"
  }
}
```

## Rate Limiting

- **General Endpoints**: 1000 requests per hour per user
- **Download Endpoints**: 100 requests per hour per user
- **Authentication Endpoints**: 10 requests per minute per IP

Rate limit headers are included in responses:
- `X-RateLimit-Limit`: Request limit
- `X-RateLimit-Remaining`: Remaining requests
- `X-RateLimit-Reset`: Reset timestamp

---

# API Endpoints

## Authentication Endpoints

### Login
Authenticate and receive a JWT token.

**Endpoint**: `POST /api/v1/auth/login`

**Request Body**:
```json
{
  "username": "string",
  "password": "string"
}
```

**Response (200)**:
```json
{
  "success": true,
  "token": "jwt_token_here",
  "user": {
    "id": 1,
    "username": "admin",
    "email": "admin@example.com",
    "role": "admin",
    "active": true
  }
}
```

**Errors**:
- `400`: Missing username or password
- `401`: Invalid credentials

---

### Logout
Invalidate the current session.

**Endpoint**: `POST /api/v1/auth/logout`

**Headers**: `Authorization: Bearer <token>`

**Response (200)**:
```json
{
  "success": true,
  "message": "Logged out successfully"
}
```

---

### Verify Token
Verify the current JWT token is valid.

**Endpoint**: `GET /api/v1/auth/verify`

**Headers**: `Authorization: Bearer <token>`

**Response (200)**:
```json
{
  "success": true,
  "user": {
    "id": 1,
    "username": "admin",
    "email": "admin@example.com",
    "role": "admin",
    "active": true
  }
}
```

---

## Catalog Management

### Get Artists
Retrieve a paginated list of artists.

**Endpoint**: `GET /api/v1/catalog/artists`

**Headers**: `Authorization: Bearer <token>`

**Query Parameters**:
- `page` (int): Page number (default: 1)
- `page_size` (int): Items per page (default: 20, max: 100)
- `search` (string): Search by artist name
- `sort_by` (string): Sort field (name, show_count, last_show_date)
- `sort_order` (string): asc or desc (default: asc)

**Response (200)**:
```json
{
  "artists": [
    {
      "id": 1,
      "name": "Grateful Dead",
      "slug": "grateful-dead",
      "show_count": 2847,
      "first_show_date": "1965-05-05",
      "last_show_date": "1995-07-09",
      "is_active": false,
      "genres": ["Rock", "Psychedelic"],
      "bio": "American rock band formed in 1965...",
      "image_url": "https://example.com/artists/grateful-dead.jpg"
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total": 150,
    "has_next": true,
    "has_prev": false
  }
}
```

---

### Get Single Artist
Retrieve detailed information about a specific artist.

**Endpoint**: `GET /api/v1/catalog/artists/{id}`

**Headers**: `Authorization: Bearer <token>`

**Path Parameters**:
- `id` (int): Artist ID

**Response (200)**:
```json
{
  "artist": {
    "id": 1,
    "name": "Grateful Dead",
    "slug": "grateful-dead",
    "show_count": 2847,
    "first_show_date": "1965-05-05",
    "last_show_date": "1995-07-09",
    "is_active": false,
    "genres": ["Rock", "Psychedelic"],
    "bio": "American rock band formed in 1965 in Palo Alto, California...",
    "image_url": "https://example.com/artists/grateful-dead.jpg",
    "social_links": {
      "website": "https://dead.net",
      "wikipedia": "https://en.wikipedia.org/wiki/Grateful_Dead"
    },
    "stats": {
      "total_shows": 2847,
      "total_downloads": 15420,
      "top_venues": [
        "Fillmore West",
        "Madison Square Garden"
      ]
    }
  }
}
```

**Errors**:
- `400`: Invalid artist ID format
- `404`: Artist not found

---

### Get Artist Shows
Retrieve shows for a specific artist.

**Endpoint**: `GET /api/v1/catalog/artists/{id}/shows`

**Headers**: `Authorization: Bearer <token>`

**Path Parameters**:
- `id` (int): Artist ID

**Query Parameters**:
- `page` (int): Page number
- `page_size` (int): Items per page
- `year` (int): Filter by year
- `venue` (string): Filter by venue name
- `sort_by` (string): date, venue, downloads
- `sort_order` (string): asc or desc

**Response (200)**:
```json
{
  "shows": [
    {
      "id": 12345,
      "artist_id": 1,
      "artist_name": "Grateful Dead",
      "date": "1977-05-08",
      "venue": "Barton Hall, Cornell University",
      "city": "Ithaca",
      "state": "NY",
      "country": "USA",
      "container_id": 67890,
      "duration_minutes": 180,
      "set_list": [
        "Set 1: Minglewood Blues, Loser, El Paso...",
        "Set 2: Scarlet Begonias > Fire on the Mountain..."
      ],
      "download_count": 5420,
      "rating": 4.9,
      "is_available": true,
      "formats": ["MP3", "FLAC", "ALAC"],
      "qualities": ["320kbps", "16bit/44.1kHz", "24bit/96kHz"]
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total": 2847,
    "has_next": true,
    "has_prev": false
  }
}
```

---

### Search Shows
Search for shows across all artists.

**Endpoint**: `GET /api/v1/catalog/shows/search`

**Headers**: `Authorization: Bearer <token>`

**Query Parameters**:
- `artist` (string): Artist name search
- `venue` (string): Venue name search
- `city` (string): City search
- `date_from` (string): Start date (YYYY-MM-DD)
- `date_to` (string): End date (YYYY-MM-DD)
- `year` (int): Specific year
- `page` (int): Page number
- `page_size` (int): Items per page
- `sort_by` (string): date, artist, venue, downloads
- `sort_order` (string): asc or desc

**Response (200)**:
```json
{
  "shows": [
    {
      "id": 12345,
      "artist_id": 1,
      "artist_name": "Grateful Dead",
      "date": "1977-05-08",
      "venue": "Barton Hall, Cornell University",
      "city": "Ithaca",
      "state": "NY",
      "country": "USA",
      "container_id": 67890,
      "duration_minutes": 180,
      "download_count": 5420,
      "rating": 4.9,
      "is_available": true,
      "formats": ["MP3", "FLAC", "ALAC"]
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total": 450,
    "has_next": true,
    "has_prev": false
  }
}
```

---

### Get Single Show
Retrieve detailed information about a specific show.

**Endpoint**: `GET /api/v1/catalog/shows/{id}`

**Headers**: `Authorization: Bearer <token>`

**Path Parameters**:
- `id` (int): Show ID

**Response (200)**:
```json
{
  "show": {
    "id": 12345,
    "artist_id": 1,
    "artist_name": "Grateful Dead",
    "date": "1977-05-08",
    "venue": "Barton Hall, Cornell University",
    "city": "Ithaca",
    "state": "NY",
    "country": "USA",
    "container_id": 67890,
    "duration_minutes": 180,
    "set_list": [
      "Set 1: Minglewood Blues, Loser, El Paso, They Love Each Other, Jack Straw, Deal, Looks Like Rain > Deal, Around and Around, U.S. Blues",
      "Set 2: Scarlet Begonias > Fire on the Mountain, Estimated Prophet, St. Stephen > Not Fade Away > St. Stephen > Seastones > Drums > The Other One > Comes a Time > Sugar Magnolia",
      "Encore: Uncle John's Band"
    ],
    "notes": "Legendary Cornell '77 show, often considered one of the greatest Dead shows ever recorded.",
    "download_count": 5420,
    "rating": 4.9,
    "is_available": true,
    "formats": {
      "MP3": ["320kbps", "256kbps"],
      "FLAC": ["16bit/44.1kHz"],
      "ALAC": ["16bit/44.1kHz"]
    },
    "file_info": {
      "source": "Audience recording",
      "taper": "Betty Cantor-Jackson",
      "equipment": "Unknown mics > Nakamichi 550 > FLAC"
    },
    "tags": ["audience", "excellent", "legendary"],
    "reviews": [
      {
        "user": "deadhead77",
        "rating": 5,
        "comment": "Absolutely incredible show. The Scarlet > Fire is transcendent."
      }
    ]
  }
}
```

---

### Start Catalog Refresh
Initiate a background job to refresh the catalog from the Nugs API.

**Endpoint**: `POST /api/v1/catalog/refresh`

**Headers**: `Authorization: Bearer <token>`

**Request Body**:
```json
{
  "force": false,
  "artists": [1, 2, 3],  // Optional: specific artist IDs
  "since_date": "2024-01-01"  // Optional: only refresh shows after this date
}
```

**Response (202)**:
```json
{
  "success": true,
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "message": "Catalog refresh started",
  "estimated_duration": "5-10 minutes"
}
```

---

### Get Refresh Status
Check the status of a catalog refresh job.

**Endpoint**: `GET /api/v1/catalog/refresh/status/{job_id}`

**Headers**: `Authorization: Bearer <token>`

**Path Parameters**:
- `job_id` (string): Job ID from refresh request

**Response (200)**:
```json
{
  "job": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "type": "catalog_refresh",
    "status": "running",
    "progress": 75,
    "message": "Processing artist 150 of 200",
    "created_at": "2024-01-15T10:30:00Z",
    "started_at": "2024-01-15T10:30:05Z",
    "estimated_completion": "2024-01-15T10:35:00Z",
    "result": null
  },
  "status": "Job is running"
}
```

**Job Statuses**:
- `pending`: Job queued but not started
- `running`: Job currently executing
- `completed`: Job finished successfully
- `failed`: Job encountered an error
- `canceled`: Job was canceled

---

### List Refresh Jobs
Get a list of recent catalog refresh jobs.

**Endpoint**: `GET /api/v1/catalog/refresh/jobs`

**Headers**: `Authorization: Bearer <token>`

**Query Parameters**:
- `status` (string): Filter by job status
- `limit` (int): Number of jobs to return (default: 50)

**Response (200)**:
```json
{
  "jobs": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "type": "catalog_refresh",
      "status": "completed",
      "progress": 100,
      "message": "Refresh completed successfully",
      "created_at": "2024-01-15T10:30:00Z",
      "completed_at": "2024-01-15T10:37:22Z",
      "duration_seconds": 442,
      "result": {
        "artists_processed": 200,
        "new_shows": 45,
        "updated_shows": 12,
        "errors": 0
      }
    }
  ]
}
```

---

### Cancel Refresh Job
Cancel a running catalog refresh job.

**Endpoint**: `DELETE /api/v1/catalog/refresh/{job_id}`

**Headers**: `Authorization: Bearer <token>`

**Path Parameters**:
- `job_id` (string): Job ID to cancel

**Response (200)**:
```json
{
  "success": true,
  "message": "Job canceled successfully"
}
```

---

### Get Refresh Info
Get information about the catalog refresh system.

**Endpoint**: `GET /api/v1/catalog/refresh/info`

**Headers**: `Authorization: Bearer <token>`

**Response (200)**:
```json
{
  "system_info": {
    "last_full_refresh": "2024-01-15T10:37:22Z",
    "last_incremental_refresh": "2024-01-16T03:15:44Z",
    "total_artists": 200,
    "total_shows": 45680,
    "refresh_frequency": "daily",
    "auto_refresh_enabled": true,
    "next_scheduled_refresh": "2024-01-17T03:00:00Z"
  },
  "recent_activity": {
    "shows_added_today": 12,
    "shows_updated_today": 5,
    "artists_added_this_week": 2,
    "total_refresh_jobs_this_month": 31
  }
}
```

---

## Download Management

### Get Downloads
Retrieve a paginated list of downloads.

**Endpoint**: `GET /api/v1/downloads`

**Headers**: `Authorization: Bearer <token>`

**Query Parameters**:
- `page` (int): Page number
- `page_size` (int): Items per page
- `status` (string): Filter by status (pending, in_progress, completed, failed, cancelled)
- `format` (string): Filter by format (mp3, flac, alac)
- `artist` (string): Filter by artist name
- `date_from` (string): Filter downloads created after date
- `date_to` (string): Filter downloads created before date

**Response (200)**:
```json
{
  "downloads": [
    {
      "id": 1001,
      "show_id": 12345,
      "container_id": 67890,
      "artist_name": "Grateful Dead",
      "show_date": "1977-05-08",
      "venue": "Barton Hall, Cornell University",
      "format": "FLAC",
      "quality": "16bit/44.1kHz",
      "status": "completed",
      "progress": 100,
      "file_path": "/downloads/grateful-dead/1977-05-08/gd77-05-08.flac",
      "file_size": 687194624,
      "downloaded_at": "2024-01-15T14:22:30Z",
      "created_at": "2024-01-15T14:15:12Z"
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total": 89,
    "has_next": true,
    "has_prev": false
  }
}
```

---

### Queue Download
Add a new download to the queue.

**Endpoint**: `POST /api/v1/downloads/queue`

**Headers**: `Authorization: Bearer <token>`

**Request Body**:
```json
{
  "container_id": 67890,
  "format": "FLAC",
  "quality": "16bit/44.1kHz",
  "download_path": "/custom/path"  // Optional
}
```

**Available Formats**:
- `MP3`: 320kbps, 256kbps
- `FLAC`: 16bit/44.1kHz, 24bit/96kHz
- `ALAC`: 16bit/44.1kHz

**Response (201)**:
```json
{
  "success": true,
  "download_id": 1002,
  "message": "Download queued successfully",
  "queue_position": 3,
  "estimated_start": "2024-01-16T09:15:00Z"
}
```

**Errors**:
- `400`: Invalid container_id or format
- `409`: Download already exists for this container

---

### Get Download Queue
View the current download queue.

**Endpoint**: `GET /api/v1/downloads/queue`

**Headers**: `Authorization: Bearer <token>`

**Response (200)**:
```json
{
  "queue": [
    {
      "id": 1002,
      "container_id": 67891,
      "artist_name": "Phish",
      "show_date": "1997-12-31",
      "venue": "Madison Square Garden",
      "format": "FLAC",
      "quality": "16bit/44.1kHz",
      "status": "pending",
      "queue_position": 1,
      "estimated_start": "2024-01-16T09:15:00Z",
      "priority": "normal",
      "created_at": "2024-01-16T08:45:22Z"
    }
  ],
  "stats": {
    "total_items": 5,
    "pending_items": 3,
    "processing_items": 2,
    "estimated_completion": "2024-01-16T12:30:00Z",
    "total_size_gb": 12.4
  }
}
```

---

### Reorder Queue
Change the order of items in the download queue.

**Endpoint**: `POST /api/v1/downloads/queue/reorder`

**Headers**: `Authorization: Bearer <token>`

**Request Body**:
```json
{
  "download_ids": [1005, 1002, 1003, 1004]
}
```

**Response (200)**:
```json
{
  "success": true,
  "message": "Queue reordered successfully",
  "new_order": [
    {
      "id": 1005,
      "queue_position": 1
    },
    {
      "id": 1002,
      "queue_position": 2
    }
  ]
}
```

---

### Get Download Statistics
Get comprehensive download statistics.

**Endpoint**: `GET /api/v1/downloads/stats`

**Headers**: `Authorization: Bearer <token>`

**Response (200)**:
```json
{
  "total_downloads": 1542,
  "completed_downloads": 1489,
  "failed_downloads": 43,
  "pending_downloads": 10,
  "total_size_gb": 2847.5,
  "queue_length": 10,
  "active_downloads": 2,
  "average_speed_mbps": 15.7,
  "success_rate": 96.6,
  "by_format": {
    "FLAC": 892,
    "MP3": 534,
    "ALAC": 116
  },
  "by_quality": {
    "16bit/44.1kHz": 1245,
    "320kbps": 534,
    "24bit/96kHz": 89
  },
  "top_artists": [
    {
      "artist": "Grateful Dead",
      "downloads": 245,
      "total_size_gb": 428.7
    }
  ],
  "recent_activity": {
    "downloads_today": 12,
    "downloads_this_week": 89,
    "size_downloaded_today_gb": 21.4
  }
}
```

---

### Get Single Download
Get details of a specific download.

**Endpoint**: `GET /api/v1/downloads/{id}`

**Headers**: `Authorization: Bearer <token>`

**Path Parameters**:
- `id` (int): Download ID

**Response (200)**:
```json
{
  "download": {
    "id": 1001,
    "show_id": 12345,
    "container_id": 67890,
    "artist_name": "Grateful Dead",
    "show_date": "1977-05-08",
    "venue": "Barton Hall, Cornell University",
    "format": "FLAC",
    "quality": "16bit/44.1kHz",
    "status": "completed",
    "progress": 100,
    "file_path": "/downloads/grateful-dead/1977-05-08/gd77-05-08.flac",
    "file_size": 687194624,
    "download_speed_mbps": 12.4,
    "downloaded_at": "2024-01-15T14:22:30Z",
    "created_at": "2024-01-15T14:15:12Z",
    "updated_at": "2024-01-15T14:22:30Z",
    "error_message": null,
    "retry_count": 0,
    "job_id": "download-550e8400-e29b-41d4-a716-446655440000"
  }
}
```

---

### Cancel Download
Cancel a pending or in-progress download.

**Endpoint**: `DELETE /api/v1/downloads/{id}`

**Headers**: `Authorization: Bearer <token>`

**Path Parameters**:
- `id` (int): Download ID

**Response (200)**:
```json
{
  "success": true,
  "message": "Download canceled successfully"
}
```

**Errors**:
- `404`: Download not found
- `409`: Cannot cancel completed download

---

## Monitoring

### Create Monitor
Create a new artist monitor to track new shows.

**Endpoint**: `POST /api/v1/monitoring/monitors`

**Headers**: `Authorization: Bearer <token>`

**Request Body**:
```json
{
  "artist_id": 1,
  "settings": {
    "check_frequency": "hourly",  // hourly, daily, weekly
    "notify_new": true,
    "notify_updates": false,
    "auto_download": false,
    "download_format": "FLAC",
    "download_quality": "16bit/44.1kHz",
    "min_rating": 4.0,
    "venue_filters": ["Madison Square Garden", "Red Rocks"]
  }
}
```

**Response (201)**:
```json
{
  "success": true,
  "monitor_id": 501,
  "message": "Monitor created successfully"
}
```

---

### Create Bulk Monitors
Create multiple monitors at once.

**Endpoint**: `POST /api/v1/monitoring/monitors/bulk`

**Headers**: `Authorization: Bearer <token>`

**Request Body**:
```json
{
  "monitors": [
    {
      "artist_id": 1,
      "settings": {
        "check_frequency": "daily",
        "notify_new": true
      }
    },
    {
      "artist_id": 2,
      "settings": {
        "check_frequency": "hourly",
        "notify_new": true,
        "auto_download": true
      }
    }
  ]
}
```

**Response (201)**:
```json
{
  "success": true,
  "created_count": 2,
  "failed_count": 0,
  "monitor_ids": [501, 502],
  "message": "2 monitors created successfully"
}
```

---

### Get Monitors
Retrieve a list of active monitors.

**Endpoint**: `GET /api/v1/monitoring/monitors`

**Headers**: `Authorization: Bearer <token>`

**Query Parameters**:
- `page` (int): Page number
- `page_size` (int): Items per page
- `artist_id` (int): Filter by artist
- `status` (string): Filter by status (active, paused, error)
- `frequency` (string): Filter by check frequency

**Response (200)**:
```json
{
  "monitors": [
    {
      "id": 501,
      "artist_id": 1,
      "artist_name": "Grateful Dead",
      "status": "active",
      "settings": {
        "check_frequency": "daily",
        "notify_new": true,
        "notify_updates": false,
        "auto_download": false
      },
      "stats": {
        "shows_found": 15,
        "alerts_sent": 8,
        "last_check": "2024-01-16T03:00:00Z",
        "next_check": "2024-01-17T03:00:00Z"
      },
      "created_at": "2024-01-10T15:30:00Z",
      "updated_at": "2024-01-16T03:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total": 25,
    "has_next": true,
    "has_prev": false
  }
}
```

---

### Get Single Monitor
Get details of a specific monitor.

**Endpoint**: `GET /api/v1/monitoring/monitors/{id}`

**Headers**: `Authorization: Bearer <token>`

**Path Parameters**:
- `id` (int): Monitor ID

**Response (200)**:
```json
{
  "monitor": {
    "id": 501,
    "artist_id": 1,
    "artist_name": "Grateful Dead",
    "status": "active",
    "settings": {
      "check_frequency": "daily",
      "notify_new": true,
      "notify_updates": false,
      "auto_download": false,
      "download_format": "FLAC",
      "download_quality": "16bit/44.1kHz",
      "min_rating": 4.0,
      "venue_filters": []
    },
    "stats": {
      "shows_found": 15,
      "alerts_sent": 8,
      "auto_downloads": 0,
      "last_check": "2024-01-16T03:00:00Z",
      "next_check": "2024-01-17T03:00:00Z",
      "check_count": 45,
      "error_count": 2
    },
    "recent_finds": [
      {
        "show_id": 12346,
        "date": "1978-05-08",
        "venue": "Barton Hall",
        "found_at": "2024-01-15T14:30:00Z",
        "action_taken": "alert_sent"
      }
    ],
    "created_at": "2024-01-10T15:30:00Z",
    "updated_at": "2024-01-16T03:00:00Z"
  }
}
```

---

### Update Monitor
Update monitor settings.

**Endpoint**: `PUT /api/v1/monitoring/monitors/{id}`

**Headers**: `Authorization: Bearer <token>`

**Path Parameters**:
- `id` (int): Monitor ID

**Request Body**:
```json
{
  "status": "active",
  "settings": {
    "check_frequency": "hourly",
    "notify_new": true,
    "auto_download": true,
    "download_format": "MP3",
    "download_quality": "320kbps"
  }
}
```

**Response (200)**:
```json
{
  "success": true,
  "message": "Monitor updated successfully"
}
```

---

### Delete Monitor
Delete a monitor.

**Endpoint**: `DELETE /api/v1/monitoring/monitors/{id}`

**Headers**: `Authorization: Bearer <token>`

**Path Parameters**:
- `id` (int): Monitor ID

**Response (200)**:
```json
{
  "success": true,
  "message": "Monitor deleted successfully"
}
```

---

### Check All Monitors
Manually trigger a check of all active monitors.

**Endpoint**: `POST /api/v1/monitoring/check/all`

**Headers**: `Authorization: Bearer <token>`

**Request Body** (optional):
```json
{
  "force": true  // Force check even if recently checked
}
```

**Response (202)**:
```json
{
  "success": true,
  "job_id": "monitor-550e8400-e29b-41d4-a716-446655440000",
  "message": "Monitor check started",
  "monitors_to_check": 25
}
```

---

### Check Specific Artist
Manually check for new shows for a specific artist.

**Endpoint**: `POST /api/v1/monitoring/check/artist/{id}`

**Headers**: `Authorization: Bearer <token>`

**Path Parameters**:
- `id` (int): Artist ID

**Response (202)**:
```json
{
  "success": true,
  "job_id": "artist-check-550e8400-e29b-41d4-a716-446655440000",
  "message": "Artist check started",
  "artist_name": "Grateful Dead"
}
```

---

### Get Alerts
Retrieve monitoring alerts.

**Endpoint**: `GET /api/v1/monitoring/alerts`

**Headers**: `Authorization: Bearer <token>`

**Query Parameters**:
- `page` (int): Page number
- `page_size` (int): Items per page
- `status` (string): Filter by status (unread, read, acknowledged)
- `type` (string): Filter by type (new_show, show_update, error)
- `artist_id` (int): Filter by artist

**Response (200)**:
```json
{
  "alerts": [
    {
      "id": 1001,
      "monitor_id": 501,
      "artist_id": 1,
      "artist_name": "Grateful Dead",
      "type": "new_show",
      "title": "New Show Found",
      "message": "New Grateful Dead show found: 1978-05-08 at Barton Hall",
      "status": "unread",
      "data": {
        "show_id": 12346,
        "show_date": "1978-05-08",
        "venue": "Barton Hall",
        "container_id": 67892
      },
      "created_at": "2024-01-16T14:30:00Z",
      "acknowledged_at": null
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total": 45,
    "has_next": true,
    "has_prev": false
  }
}
```

---

### Acknowledge Alert
Mark an alert as acknowledged.

**Endpoint**: `PUT /api/v1/monitoring/alerts/{id}/acknowledge`

**Headers**: `Authorization: Bearer <token>`

**Path Parameters**:
- `id` (int): Alert ID

**Response (200)**:
```json
{
  "success": true,
  "message": "Alert acknowledged"
}
```

---

### Get Monitoring Statistics
Get comprehensive monitoring statistics.

**Endpoint**: `GET /api/v1/monitoring/stats`

**Headers**: `Authorization: Bearer <token>`

**Response (200)**:
```json
{
  "total_monitors": 25,
  "active_monitors": 23,
  "paused_monitors": 2,
  "error_monitors": 0,
  "total_alerts": 145,
  "unread_alerts": 12,
  "shows_found_today": 3,
  "shows_found_this_week": 18,
  "auto_downloads_today": 1,
  "check_frequency_breakdown": {
    "hourly": 5,
    "daily": 18,
    "weekly": 2
  },
  "top_monitored_artists": [
    {
      "artist_id": 1,
      "artist_name": "Grateful Dead",
      "monitor_count": 1,
      "shows_found": 15,
      "last_find": "2024-01-15T14:30:00Z"
    }
  ],
  "recent_activity": {
    "checks_today": 89,
    "errors_today": 2,
    "alerts_sent_today": 3
  }
}
```

---

## Analytics

### Generate Report
Generate a custom analytics report.

**Endpoint**: `POST /api/v1/analytics/reports`

**Headers**: `Authorization: Bearer <token>`

**Request Body**:
```json
{
  "type": "collection_summary",  // collection_summary, download_analysis, artist_breakdown
  "period": "last_30_days",      // last_7_days, last_30_days, last_year, all_time
  "filters": {
    "artists": [1, 2, 3],
    "formats": ["FLAC", "MP3"],
    "date_from": "2024-01-01",
    "date_to": "2024-01-31"
  },
  "format": "json"  // json, csv
}
```

**Response (202)**:
```json
{
  "success": true,
  "report_id": "report-550e8400-e29b-41d4-a716-446655440000",
  "message": "Report generation started",
  "estimated_completion": "2024-01-16T15:35:00Z"
}
```

---

### Get Collection Statistics
Get comprehensive collection statistics.

**Endpoint**: `GET /api/v1/analytics/collection`

**Headers**: `Authorization: Bearer <token>`

**Query Parameters**:
- `period` (string): Time period for statistics (week, month, year, all)

**Response (200)**:
```json
{
  "totals": {
    "total_artists": 200,
    "total_shows": 45680,
    "total_venues": 3420,
    "total_downloads": 1542,
    "total_storage_gb": 2847.5
  },
  "growth": {
    "artists_added_this_month": 5,
    "shows_added_this_month": 234,
    "downloads_this_month": 89
  },
  "top_artists": [
    {
      "id": 1,
      "name": "Grateful Dead",
      "show_count": 2847,
      "download_count": 245,
      "storage_gb": 428.7
    }
  ],
  "top_venues": [
    {
      "name": "Madison Square Garden",
      "show_count": 342,
      "artist_count": 45,
      "download_count": 89
    }
  ],
  "format_breakdown": {
    "FLAC": {
      "count": 892,
      "percentage": 57.8,
      "total_size_gb": 1654.3
    },
    "MP3": {
      "count": 534,
      "percentage": 34.6,
      "total_size_gb": 156.7
    }
  },
  "timeline": {
    "downloads_by_month": [
      {
        "month": "2024-01",
        "downloads": 89,
        "size_gb": 156.7
      }
    ]
  }
}
```

---

### Get Artist Analytics
Get detailed analytics for artists.

**Endpoint**: `GET /api/v1/analytics/artists`

**Headers**: `Authorization: Bearer <token>`

**Query Parameters**:
- `limit` (int): Number of artists to return (default: 20)
- `sort_by` (string): Sort by (downloads, shows, storage, popularity)
- `period` (string): Time period for analytics

**Response (200)**:
```json
{
  "artists": [
    {
      "id": 1,
      "name": "Grateful Dead",
      "slug": "grateful-dead",
      "stats": {
        "total_shows": 2847,
        "available_shows": 2840,
        "download_count": 245,
        "total_size_gb": 428.7,
        "average_rating": 4.3,
        "popularity_score": 89.5
      },
      "trends": {
        "downloads_trend": "+15%",
        "new_shows_this_month": 0,
        "top_year": "1977"
      },
      "top_shows": [
        {
          "show_id": 12345,
          "date": "1977-05-08",
          "venue": "Barton Hall",
          "download_count": 5420,
          "rating": 4.9
        }
      ]
    }
  ],
  "summary": {
    "total_artists_analyzed": 200,
    "total_downloads": 1542,
    "average_shows_per_artist": 228.4
  }
}
```

---

### Get Download Analytics
Get detailed download analytics.

**Endpoint**: `GET /api/v1/analytics/downloads`

**Headers**: `Authorization: Bearer <token>`

**Query Parameters**:
- `period` (string): Time period for analytics
- `group_by` (string): Group by (artist, format, quality, month)

**Response (200)**:
```json
{
  "overview": {
    "total_downloads": 1542,
    "successful_downloads": 1489,
    "failed_downloads": 43,
    "canceled_downloads": 10,
    "success_rate": 96.6,
    "average_download_time_minutes": 12.5,
    "total_bandwidth_gb": 2847.5
  },
  "by_format": {
    "FLAC": {
      "count": 892,
      "success_rate": 97.2,
      "average_size_mb": 650,
      "total_size_gb": 580.2
    }
  },
  "by_quality": {
    "16bit/44.1kHz": {
      "count": 1245,
      "percentage": 80.7
    }
  },
  "timeline": [
    {
      "date": "2024-01-15",
      "downloads": 12,
      "success_rate": 100.0,
      "total_size_gb": 8.4
    }
  ],
  "peak_hours": [
    {
      "hour": 20,
      "downloads": 89,
      "average_speed_mbps": 15.7
    }
  ]
}
```

---

### Get System Metrics
Get system performance and health metrics.

**Endpoint**: `GET /api/v1/analytics/system`

**Headers**: `Authorization: Bearer <token>`

**Response (200)**:
```json
{
  "storage": {
    "total_storage_gb": 5000.0,
    "available_storage_gb": 2152.5,
    "used_percentage": 56.95,
    "download_storage_gb": 2847.5
  },
  "performance": {
    "average_response_time_ms": 145,
    "requests_per_minute": 23.5,
    "active_connections": 12,
    "error_rate": 0.8
  },
  "database": {
    "total_records": 48720,
    "database_size_mb": 234.7,
    "query_performance_ms": 12.3,
    "last_backup": "2024-01-16T02:00:00Z"
  },
  "jobs": {
    "active_jobs": 2,
    "pending_jobs": 8,
    "failed_jobs_today": 1,
    "average_job_duration_minutes": 8.5
  },
  "system": {
    "uptime": "15 days, 8 hours",
    "cpu_usage_percent": 23.5,
    "memory_usage_percent": 45.2,
    "active_monitors": 23,
    "last_catalog_refresh": "2024-01-16T03:00:00Z"
  }
}
```

---

### Get Performance Metrics
Get detailed performance analytics.

**Endpoint**: `GET /api/v1/analytics/performance`

**Headers**: `Authorization: Bearer <token>`

**Response (200)**:
```json
{
  "api_performance": {
    "average_response_time_ms": 145,
    "p95_response_time_ms": 450,
    "p99_response_time_ms": 890,
    "requests_per_second": 23.5,
    "error_rate_percent": 0.8,
    "slowest_endpoints": [
      {
        "endpoint": "/api/v1/analytics/collection",
        "average_time_ms": 890,
        "call_count": 45
      }
    ]
  },
  "download_performance": {
    "average_speed_mbps": 15.7,
    "concurrent_downloads": 2,
    "queue_processing_time_minutes": 8.5,
    "success_rate": 96.6,
    "retry_rate": 12.3
  },
  "background_jobs": {
    "average_execution_time_minutes": 8.5,
    "success_rate": 94.2,
    "queue_size": 8,
    "failed_jobs_last_24h": 3
  },
  "database_performance": {
    "average_query_time_ms": 12.3,
    "slow_queries_count": 2,
    "connection_pool_usage": 45.2,
    "index_efficiency": 89.5
  }
}
```

---

### Get Top Artists
Get top artists by various metrics.

**Endpoint**: `GET /api/v1/analytics/top/artists`

**Headers**: `Authorization: Bearer <token>`

**Query Parameters**:
- `metric` (string): Metric to sort by (downloads, shows, size, popularity)
- `limit` (int): Number of artists to return (default: 10)
- `period` (string): Time period to analyze

**Response (200)**:
```json
{
  "top_artists": [
    {
      "rank": 1,
      "artist_id": 1,
      "name": "Grateful Dead",
      "metric_value": 245,
      "metric_name": "downloads",
      "percentage_of_total": 15.9,
      "change_from_previous_period": "+12%"
    }
  ],
  "metric_info": {
    "metric": "downloads",
    "period": "last_30_days",
    "total_value": 1542
  }
}
```

---

### Get Top Venues
Get top venues by various metrics.

**Endpoint**: `GET /api/v1/analytics/top/venues`

**Headers**: `Authorization: Bearer <token>`

**Query Parameters**:
- `metric` (string): Metric to sort by (shows, downloads, artists)
- `limit` (int): Number of venues to return (default: 10)

**Response (200)**:
```json
{
  "top_venues": [
    {
      "rank": 1,
      "venue_name": "Madison Square Garden",
      "city": "New York",
      "state": "NY",
      "metric_value": 342,
      "metric_name": "shows",
      "artist_count": 45,
      "download_count": 89,
      "top_artists": ["Phish", "Grateful Dead"]
    }
  ]
}
```

---

### Get Download Trends
Get download trends over time.

**Endpoint**: `GET /api/v1/analytics/trends/downloads`

**Headers**: `Authorization: Bearer <token>`

**Query Parameters**:
- `period` (string): Time period granularity (day, week, month)
- `range` (string): Time range (last_week, last_month, last_year)

**Response (200)**:
```json
{
  "trends": [
    {
      "period": "2024-01-15",
      "downloads": 12,
      "size_gb": 8.4,
      "success_rate": 100.0,
      "formats": {
        "FLAC": 8,
        "MP3": 4
      }
    }
  ],
  "summary": {
    "trend_direction": "increasing",
    "trend_percentage": "+15%",
    "peak_period": "2024-01-15",
    "average_per_period": 8.7
  }
}
```

---

### Get Dashboard Summary
Get a comprehensive dashboard summary.

**Endpoint**: `GET /api/v1/analytics/summary`

**Headers**: `Authorization: Bearer <token>`

**Response (200)**:
```json
{
  "collection": {
    "total_artists": 200,
    "total_shows": 45680,
    "new_shows_this_week": 18
  },
  "downloads": {
    "total_downloads": 1542,
    "pending_downloads": 10,
    "active_downloads": 2,
    "downloads_today": 12
  },
  "monitoring": {
    "active_monitors": 23,
    "alerts_today": 3,
    "shows_found_this_week": 18
  },
  "system": {
    "storage_used_gb": 2847.5,
    "storage_available_gb": 2152.5,
    "system_health_score": 94.5,
    "active_jobs": 2
  },
  "recent_activity": [
    {
      "type": "download_completed",
      "message": "Downloaded: Grateful Dead 1977-05-08",
      "timestamp": "2024-01-16T14:22:30Z"
    }
  ],
  "quick_stats": {
    "downloads_success_rate": 96.6,
    "monitoring_coverage": 85.2,
    "storage_efficiency": 78.9
  }
}
```

---

### Get Health Score
Get system health score and details.

**Endpoint**: `GET /api/v1/analytics/health`

**Headers**: `Authorization: Bearer <token>`

**Response (200)**:
```json
{
  "overall_score": 94.5,
  "status": "excellent",
  "components": {
    "api_performance": {
      "score": 92.0,
      "status": "good",
      "details": "Average response time: 145ms"
    },
    "download_system": {
      "score": 96.5,
      "status": "excellent",
      "details": "96.6% success rate"
    },
    "monitoring": {
      "score": 98.0,
      "status": "excellent",
      "details": "All monitors active"
    },
    "storage": {
      "score": 89.0,
      "status": "good",
      "details": "56.95% storage used"
    },
    "database": {
      "score": 95.0,
      "status": "excellent",
      "details": "Query performance optimal"
    }
  },
  "recommendations": [
    "Consider archiving old download logs to free up storage",
    "Monitor API performance during peak hours"
  ],
  "last_updated": "2024-01-16T15:30:00Z"
}
```

---

## Webhooks

### Create Webhook
Create a new webhook endpoint.

**Endpoint**: `POST /api/v1/webhooks`

**Headers**: `Authorization: Bearer <token>`

**Request Body**:
```json
{
  "url": "https://your-server.com/webhooks/nugs",
  "events": ["new_show", "download_complete", "monitor_alert"],
  "secret": "your_webhook_secret",
  "timeout": 30,
  "retry_count": 3,
  "active": true,
  "description": "Main webhook for new show notifications"
}
```

**Available Events**:
- `new_show`: New show found by monitoring
- `download_complete`: Download finished (success or failure)
- `download_started`: Download started
- `monitor_alert`: Monitor generated an alert
- `catalog_refresh`: Catalog refresh completed
- `system_error`: System error occurred

**Response (201)**:
```json
{
  "success": true,
  "webhook_id": 101,
  "message": "Webhook created successfully"
}
```

---

### Get Webhooks
List all webhooks.

**Endpoint**: `GET /api/v1/webhooks`

**Headers**: `Authorization: Bearer <token>`

**Query Parameters**:
- `page` (int): Page number
- `page_size` (int): Items per page
- `status` (string): Filter by status (active, disabled, failed)

**Response (200)**:
```json
{
  "webhooks": [
    {
      "id": 101,
      "url": "https://your-server.com/webhooks/nugs",
      "events": ["new_show", "download_complete"],
      "status": "active",
      "timeout": 30,
      "retry_count": 3,
      "description": "Main webhook for notifications",
      "stats": {
        "total_deliveries": 156,
        "successful_deliveries": 154,
        "failed_deliveries": 2,
        "success_rate": 98.7,
        "last_delivery": "2024-01-16T14:30:00Z",
        "last_success": "2024-01-16T14:30:00Z",
        "last_failure": "2024-01-15T08:22:15Z"
      },
      "created_at": "2024-01-10T12:00:00Z",
      "updated_at": "2024-01-16T14:30:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total": 5,
    "has_next": false,
    "has_prev": false
  }
}
```

---

### Get Single Webhook
Get details of a specific webhook.

**Endpoint**: `GET /api/v1/webhooks/{id}`

**Headers**: `Authorization: Bearer <token>`

**Path Parameters**:
- `id` (int): Webhook ID

**Response (200)**:
```json
{
  "webhook": {
    "id": 101,
    "url": "https://your-server.com/webhooks/nugs",
    "events": ["new_show", "download_complete", "monitor_alert"],
    "status": "active",
    "timeout": 30,
    "retry_count": 3,
    "secret": "***hidden***",
    "description": "Main webhook for new show notifications",
    "stats": {
      "total_deliveries": 156,
      "successful_deliveries": 154,
      "failed_deliveries": 2,
      "success_rate": 98.7,
      "average_response_time_ms": 245,
      "last_delivery": "2024-01-16T14:30:00Z",
      "last_success": "2024-01-16T14:30:00Z",
      "last_failure": "2024-01-15T08:22:15Z",
      "consecutive_failures": 0
    },
    "created_at": "2024-01-10T12:00:00Z",
    "updated_at": "2024-01-16T14:30:00Z"
  }
}
```

---

### Update Webhook
Update an existing webhook.

**Endpoint**: `PUT /api/v1/webhooks/{id}`

**Headers**: `Authorization: Bearer <token>`

**Path Parameters**:
- `id` (int): Webhook ID

**Request Body**:
```json
{
  "url": "https://new-server.com/webhooks/nugs",
  "events": ["new_show", "download_complete"],
  "active": true,
  "timeout": 45,
  "retry_count": 5,
  "description": "Updated webhook endpoint"
}
```

**Response (200)**:
```json
{
  "success": true,
  "message": "Webhook updated successfully"
}
```

---

### Delete Webhook
Delete a webhook.

**Endpoint**: `DELETE /api/v1/webhooks/{id}`

**Headers**: `Authorization: Bearer <token>`

**Path Parameters**:
- `id` (int): Webhook ID

**Response (200)**:
```json
{
  "success": true,
  "message": "Webhook deleted successfully"
}
```

---

### Test Webhook
Send a test payload to a webhook.

**Endpoint**: `POST /api/v1/webhooks/{id}/test`

**Headers**: `Authorization: Bearer <token>`

**Path Parameters**:
- `id` (int): Webhook ID

**Request Body**:
```json
{
  "event": "new_show",
  "test_data": {
    "custom_field": "test_value"
  }
}
```

**Response (200)**:
```json
{
  "success": true,
  "delivery_id": 5001,
  "response_code": 200,
  "response_time_ms": 245,
  "message": "Test webhook delivered successfully"
}
```

---

### Get Webhook Deliveries
Get delivery history for a specific webhook.

**Endpoint**: `GET /api/v1/webhooks/{id}/deliveries`

**Headers**: `Authorization: Bearer <token>`

**Path Parameters**:
- `id` (int): Webhook ID

**Query Parameters**:
- `page` (int): Page number
- `page_size` (int): Items per page
- `status` (string): Filter by success/failure
- `event` (string): Filter by event type

**Response (200)**:
```json
{
  "deliveries": [
    {
      "id": 5001,
      "event": "new_show",
      "status_code": 200,
      "success": true,
      "duration_ms": 245,
      "attempt": 1,
      "payload": {
        "event": "new_show",
        "data": {
          "show_id": 12346,
          "artist_name": "Grateful Dead",
          "date": "1978-05-08",
          "venue": "Barton Hall"
        }
      },
      "response": "OK",
      "error": null,
      "created_at": "2024-01-16T14:30:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total": 156,
    "has_next": true,
    "has_prev": false
  }
}
```

---

### Get All Deliveries
Get delivery history across all webhooks.

**Endpoint**: `GET /api/v1/webhooks/deliveries`

**Headers**: `Authorization: Bearer <token>`

**Query Parameters**:
- `page` (int): Page number
- `page_size` (int): Items per page
- `webhook_id` (int): Filter by webhook
- `status` (string): Filter by success/failure
- `event` (string): Filter by event type

**Response (200)**:
```json
{
  "deliveries": [
    {
      "id": 5001,
      "webhook_id": 101,
      "webhook_url": "https://your-server.com/webhooks/nugs",
      "event": "new_show",
      "status_code": 200,
      "success": true,
      "duration_ms": 245,
      "attempt": 1,
      "created_at": "2024-01-16T14:30:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total": 567,
    "has_next": true,
    "has_prev": false
  }
}
```

---

### Get Available Events
List all available webhook events.

**Endpoint**: `GET /api/v1/webhooks/events`

**Headers**: `Authorization: Bearer <token>`

**Response (200)**:
```json
{
  "events": [
    {
      "name": "new_show",
      "description": "New show found by monitoring system",
      "payload_example": {
        "event": "new_show",
        "timestamp": "2024-01-16T14:30:00Z",
        "data": {
          "show_id": 12346,
          "artist_id": 1,
          "artist_name": "Grateful Dead",
          "date": "1978-05-08",
          "venue": "Barton Hall",
          "container_id": 67892
        }
      }
    },
    {
      "name": "download_complete",
      "description": "Download finished (success or failure)",
      "payload_example": {
        "event": "download_complete",
        "timestamp": "2024-01-16T14:30:00Z",
        "data": {
          "download_id": 1001,
          "show_id": 12345,
          "artist_name": "Grateful Dead",
          "status": "completed",
          "file_path": "/downloads/gd77-05-08.flac",
          "file_size": 687194624
        }
      }
    }
  ]
}
```

---

### Get Webhook Statistics
Get overall webhook statistics.

**Endpoint**: `GET /api/v1/webhooks/stats`

**Headers**: `Authorization: Bearer <token>`

**Response (200)**:
```json
{
  "total_webhooks": 5,
  "active_webhooks": 4,
  "disabled_webhooks": 1,
  "total_deliveries": 1245,
  "successful_deliveries": 1198,
  "failed_deliveries": 47,
  "overall_success_rate": 96.2,
  "deliveries_today": 23,
  "average_response_time_ms": 245,
  "by_event": {
    "new_show": {
      "deliveries": 456,
      "success_rate": 98.5
    },
    "download_complete": {
      "deliveries": 789,
      "success_rate": 94.8
    }
  },
  "recent_activity": [
    {
      "webhook_id": 101,
      "event": "new_show",
      "status": "success",
      "timestamp": "2024-01-16T14:30:00Z"
    }
  ]
}
```

---

## Administration

### Create User
Create a new user account.

**Endpoint**: `POST /api/v1/admin/users`

**Headers**: `Authorization: Bearer <token>`

**Required Role**: Admin

**Request Body**:
```json
{
  "username": "newuser",
  "email": "newuser@example.com",
  "password": "securepassword123",
  "role": "user",
  "active": true
}
```

**Available Roles**:
- `admin`: Full access to all features
- `user`: Standard user access
- `readonly`: Read-only access

**Response (201)**:
```json
{
  "success": true,
  "user_id": 25,
  "message": "User created successfully"
}
```

---

### Get Users
List all users.

**Endpoint**: `GET /api/v1/admin/users`

**Headers**: `Authorization: Bearer <token>`

**Required Role**: Admin

**Query Parameters**:
- `page` (int): Page number
- `page_size` (int): Items per page
- `role` (string): Filter by role
- `active` (bool): Filter by active status

**Response (200)**:
```json
{
  "users": [
    {
      "id": 1,
      "username": "admin",
      "email": "admin@example.com",
      "role": "admin",
      "active": true,
      "last_login": "2024-01-16T14:30:00Z",
      "login_count": 245,
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-16T14:30:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total": 25,
    "has_next": true,
    "has_prev": false
  }
}
```

---

### Update User
Update an existing user.

**Endpoint**: `PUT /api/v1/admin/users/{id}`

**Headers**: `Authorization: Bearer <token>`

**Required Role**: Admin

**Path Parameters**:
- `id` (int): User ID

**Request Body**:
```json
{
  "email": "newemail@example.com",
  "role": "admin",
  "active": false
}
```

**Response (200)**:
```json
{
  "success": true,
  "message": "User updated successfully"
}
```

---

### Delete User
Delete a user account.

**Endpoint**: `DELETE /api/v1/admin/users/{id}`

**Headers**: `Authorization: Bearer <token>`

**Required Role**: Admin

**Path Parameters**:
- `id` (int): User ID

**Response (200)**:
```json
{
  "success": true,
  "message": "User deleted successfully"
}
```

---

### Get System Configuration
Get system configuration settings.

**Endpoint**: `GET /api/v1/admin/config`

**Headers**: `Authorization: Bearer <token>`

**Required Role**: Admin

**Response (200)**:
```json
{
  "config": {
    "download_settings": {
      "max_concurrent_downloads": 5,
      "default_download_path": "/downloads",
      "auto_retry_failed": true,
      "retry_count": 3
    },
    "monitoring_settings": {
      "default_check_frequency": "daily",
      "max_monitors_per_user": 100,
      "alert_retention_days": 30
    },
    "system_settings": {
      "log_retention_days": 30,
      "backup_frequency": "daily",
      "maintenance_window": "02:00-04:00"
    },
    "api_settings": {
      "rate_limit_enabled": true,
      "max_requests_per_hour": 1000,
      "jwt_expiry_hours": 24
    }
  }
}
```

---

### Update Configuration
Update a system configuration setting.

**Endpoint**: `PUT /api/v1/admin/config/{key}`

**Headers**: `Authorization: Bearer <token>`

**Required Role**: Admin

**Path Parameters**:
- `key` (string): Configuration key

**Request Body**:
```json
{
  "value": "new_value"
}
```

**Response (200)**:
```json
{
  "success": true,
  "message": "Configuration updated successfully"
}
```

---

### Get System Status
Get comprehensive system status.

**Endpoint**: `GET /api/v1/admin/status`

**Headers**: `Authorization: Bearer <token>`

**Required Role**: Admin

**Response (200)**:
```json
{
  "system": {
    "status": "healthy",
    "uptime": "15 days, 8 hours",
    "version": "1.0.0",
    "environment": "production"
  },
  "services": {
    "database": {
      "status": "healthy",
      "connection_count": 5,
      "response_time_ms": 12
    },
    "download_manager": {
      "status": "healthy",
      "active_downloads": 2,
      "queue_length": 8
    },
    "scheduler": {
      "status": "healthy",
      "active_schedules": 15,
      "next_execution": "2024-01-17T03:00:00Z"
    }
  },
  "resources": {
    "cpu_usage": 23.5,
    "memory_usage": 45.2,
    "disk_usage": 56.9,
    "network_io": "normal"
  },
  "alerts": [
    {
      "level": "warning",
      "message": "Disk usage above 50%",
      "timestamp": "2024-01-16T14:30:00Z"
    }
  ]
}
```

---

### Get Admin Statistics
Get administrative statistics and metrics.

**Endpoint**: `GET /api/v1/admin/stats`

**Headers**: `Authorization: Bearer <token>`

**Required Role**: Admin

**Response (200)**:
```json
{
  "users": {
    "total_users": 25,
    "active_users": 22,
    "admin_users": 3,
    "users_created_this_month": 5,
    "last_login_activity": "2024-01-16T14:30:00Z"
  },
  "system_usage": {
    "total_api_requests": 15420,
    "requests_today": 1234,
    "unique_users_today": 18,
    "peak_concurrent_users": 45,
    "error_rate": 0.8
  },
  "storage": {
    "total_downloads": 1542,
    "total_storage_gb": 2847.5,
    "average_file_size_mb": 650,
    "storage_growth_gb_per_day": 45.2
  },
  "background_jobs": {
    "total_jobs_executed": 5420,
    "jobs_today": 89,
    "average_job_duration_minutes": 8.5,
    "failed_jobs_percentage": 5.8
  }
}
```

---

### Run System Cleanup
Run system maintenance cleanup.

**Endpoint**: `POST /api/v1/admin/maintenance/cleanup`

**Headers**: `Authorization: Bearer <token>`

**Required Role**: Admin

**Request Body**:
```json
{
  "cleanup_logs": true,
  "cleanup_old_jobs": true,
  "cleanup_temp_files": true,
  "older_than_days": 30,
  "dry_run": false
}
```

**Response (202)**:
```json
{
  "success": true,
  "job_id": "cleanup-550e8400-e29b-41d4-a716-446655440000",
  "message": "Cleanup job started",
  "estimated_duration": "5-10 minutes"
}
```

---

### Get Audit Logs
Get system audit logs.

**Endpoint**: `GET /api/v1/admin/audit`

**Headers**: `Authorization: Bearer <token>`

**Required Role**: Admin

**Query Parameters**:
- `page` (int): Page number
- `page_size` (int): Items per page
- `user_id` (int): Filter by user
- `action` (string): Filter by action type
- `date_from` (string): Filter from date
- `date_to` (string): Filter to date

**Response (200)**:
```json
{
  "logs": [
    {
      "id": 10001,
      "user_id": 1,
      "username": "admin",
      "action": "user_created",
      "resource": "user",
      "resource_id": 25,
      "details": {
        "username": "newuser",
        "role": "user"
      },
      "ip_address": "192.168.1.100",
      "user_agent": "Mozilla/5.0...",
      "created_at": "2024-01-16T14:30:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 50,
    "total": 15420,
    "has_next": true,
    "has_prev": false
  }
}
```

---

### Get Jobs
Get all background jobs.

**Endpoint**: `GET /api/v1/admin/jobs`

**Headers**: `Authorization: Bearer <token>`

**Required Role**: Admin

**Query Parameters**:
- `status` (string): Filter by job status
- `type` (string): Filter by job type
- `limit` (int): Number of jobs to return

**Response (200)**:
```json
{
  "jobs": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "type": "catalog_refresh",
      "status": "completed",
      "progress": 100,
      "message": "Refresh completed successfully",
      "created_at": "2024-01-16T10:30:00Z",
      "started_at": "2024-01-16T10:30:05Z",
      "completed_at": "2024-01-16T10:37:22Z",
      "duration_seconds": 437,
      "result": {
        "artists_processed": 200,
        "new_shows": 12
      }
    }
  ]
}
```

---

### Get Single Job
Get details of a specific job.

**Endpoint**: `GET /api/v1/admin/jobs/{id}`

**Headers**: `Authorization: Bearer <token>`

**Required Role**: Admin

**Path Parameters**:
- `id` (string): Job ID

**Response (200)**:
```json
{
  "job": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "type": "catalog_refresh",
    "status": "completed",
    "progress": 100,
    "message": "Refresh completed successfully",
    "created_at": "2024-01-16T10:30:00Z",
    "started_at": "2024-01-16T10:30:05Z",
    "completed_at": "2024-01-16T10:37:22Z",
    "duration_seconds": 437,
    "result": {
      "artists_processed": 200,
      "new_shows": 12,
      "updated_shows": 5,
      "errors": []
    },
    "logs": [
      {
        "timestamp": "2024-01-16T10:30:05Z",
        "level": "info",
        "message": "Started catalog refresh"
      }
    ]
  }
}
```

---

### Cancel Job
Cancel a running job.

**Endpoint**: `DELETE /api/v1/admin/jobs/{id}`

**Headers**: `Authorization: Bearer <token>`

**Required Role**: Admin

**Path Parameters**:
- `id` (string): Job ID

**Response (200)**:
```json
{
  "success": true,
  "message": "Job canceled successfully"
}
```

---

### Create Database Backup
Create a database backup.

**Endpoint**: `POST /api/v1/admin/database/backup`

**Headers**: `Authorization: Bearer <token>`

**Required Role**: Admin

**Request Body**:
```json
{
  "include_downloads": false,
  "compress": true,
  "description": "Manual backup before maintenance"
}
```

**Response (202)**:
```json
{
  "success": true,
  "backup_id": "backup-20240116-143000",
  "message": "Database backup started",
  "estimated_size_mb": 234.7
}
```

---

### Optimize Database
Optimize database performance.

**Endpoint**: `POST /api/v1/admin/database/optimize`

**Headers**: `Authorization: Bearer <token>`

**Required Role**: Admin

**Response (200)**:
```json
{
  "success": true,
  "message": "Database optimization completed",
  "results": {
    "tables_optimized": 15,
    "indexes_rebuilt": 8,
    "space_reclaimed_mb": 45.2,
    "performance_improvement": "12%"
  }
}
```

---

### Get Database Statistics
Get database statistics and health information.

**Endpoint**: `GET /api/v1/admin/database/stats`

**Headers**: `Authorization: Bearer <token>`

**Required Role**: Admin

**Response (200)**:
```json
{
  "database": {
    "size_mb": 234.7,
    "tables": 12,
    "total_records": 48720,
    "largest_table": "shows",
    "growth_rate_mb_per_day": 5.2
  },
  "performance": {
    "average_query_time_ms": 12.3,
    "slow_queries": 2,
    "index_usage": 89.5,
    "connection_pool_usage": 45.2
  },
  "maintenance": {
    "last_backup": "2024-01-16T02:00:00Z",
    "last_optimization": "2024-01-15T03:15:00Z",
    "next_scheduled_backup": "2024-01-17T02:00:00Z"
  },
  "tables": [
    {
      "name": "shows",
      "records": 45680,
      "size_mb": 156.8,
      "last_updated": "2024-01-16T14:30:00Z"
    }
  ]
}
```

---

## Scheduler

### Start Scheduler
Start the background scheduler service.

**Endpoint**: `POST /api/v1/scheduler/start`

**Headers**: `Authorization: Bearer <token>`

**Response (200)**:
```json
{
  "success": true,
  "message": "Scheduler started successfully"
}
```

---

### Stop Scheduler
Stop the background scheduler service.

**Endpoint**: `POST /api/v1/scheduler/stop`

**Headers**: `Authorization: Bearer <token>`

**Response (200)**:
```json
{
  "success": true,
  "message": "Scheduler stopped successfully"
}
```

---

### Get Scheduler Status
Get the current status of the scheduler.

**Endpoint**: `GET /api/v1/scheduler/status`

**Headers**: `Authorization: Bearer <token>`

**Response (200)**:
```json
{
  "is_running": true,
  "start_time": "2024-01-16T10:00:00Z",
  "uptime": "6 hours, 30 minutes",
  "active_schedules": 15,
  "paused_schedules": 2,
  "total_schedules": 17,
  "running_jobs": 1,
  "next_execution": "2024-01-16T17:00:00Z",
  "last_execution": "2024-01-16T16:00:00Z",
  "executions_today": 23,
  "failures_today": 1
}
```

---

### Get Scheduler Statistics
Get comprehensive scheduler statistics.

**Endpoint**: `GET /api/v1/scheduler/stats`

**Headers**: `Authorization: Bearer <token>`

**Response (200)**:
```json
{
  "total_schedules": 17,
  "active_schedules": 15,
  "paused_schedules": 2,
  "disabled_schedules": 0,
  "total_executions": 5420,
  "successful_executions": 5198,
  "failed_executions": 222,
  "average_success_rate": 95.9,
  "executions_last_24h": 89,
  "failures_last_24h": 3,
  "type_breakdown": {
    "catalog_refresh": 2,
    "monitor_check": 8,
    "system_cleanup": 3,
    "database_backup": 2,
    "health_check": 2
  },
  "popular_schedules": [
    {
      "schedule_id": 1,
      "schedule_name": "Hourly Monitor Check",
      "execution_count": 1245,
      "success_rate": 98.2,
      "average_runtime_ms": 5420
    }
  ],
  "recent_activity": [
    {
      "schedule_id": 1,
      "schedule_name": "Hourly Monitor Check",
      "status": "completed",
      "duration_ms": 5200,
      "started_at": "2024-01-16T16:00:00Z"
    }
  ]
}
```

---

### Create Schedule
Create a new scheduled task.

**Endpoint**: `POST /api/v1/scheduler/schedules`

**Headers**: `Authorization: Bearer <token>`

**Request Body**:
```json
{
  "name": "Daily Catalog Refresh",
  "description": "Refresh the catalog daily at 3 AM",
  "type": "catalog_refresh",
  "cron_expr": "0 3 * * *",
  "parameters": {
    "force": false,
    "artists": []
  }
}
```

**Schedule Types**:
- `catalog_refresh`: Refresh catalog data
- `monitor_check`: Check all monitors
- `system_cleanup`: Clean up old data
- `database_backup`: Create database backup
- `health_check`: System health check
- `custom`: Custom task

**Response (201)**:
```json
{
  "success": true,
  "schedule_id": 18,
  "message": "Schedule created successfully"
}
```

---

### Get Schedules
List all schedules.

**Endpoint**: `GET /api/v1/scheduler/schedules`

**Headers**: `Authorization: Bearer <token>`

**Query Parameters**:
- `page` (int): Page number
- `page_size` (int): Items per page
- `status` (string): Filter by status
- `type` (string): Filter by type

**Response (200)**:
```json
{
  "schedules": [
    {
      "id": 1,
      "name": "Daily Catalog Refresh",
      "description": "Refresh the catalog daily at 3 AM",
      "type": "catalog_refresh",
      "cron_expr": "0 3 * * *",
      "status": "active",
      "next_run": "2024-01-17T03:00:00Z",
      "last_run": "2024-01-16T03:00:00Z",
      "last_status": "completed",
      "run_count": 45,
      "fail_count": 2,
      "created_at": "2024-01-01T00:00:00Z",
      "created_by": "admin"
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total": 17,
    "has_next": false,
    "has_prev": false
  }
}
```

---

### Get Single Schedule
Get details of a specific schedule.

**Endpoint**: `GET /api/v1/scheduler/schedules/{id}`

**Headers**: `Authorization: Bearer <token>`

**Path Parameters**:
- `id` (int): Schedule ID

**Response (200)**:
```json
{
  "schedule": {
    "id": 1,
    "name": "Daily Catalog Refresh",
    "description": "Refresh the catalog daily at 3 AM",
    "type": "catalog_refresh",
    "cron_expr": "0 3 * * *",
    "status": "active",
    "parameters": {
      "force": false,
      "artists": []
    },
    "next_run": "2024-01-17T03:00:00Z",
    "last_run": "2024-01-16T03:00:00Z",
    "last_job_id": "schedule-1-20240116-030000",
    "last_status": "completed",
    "run_count": 45,
    "fail_count": 2,
    "is_running": false,
    "failure_rate": 4.4,
    "average_runtime": "7 minutes",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-16T03:00:00Z",
    "created_by": "admin"
  }
}
```

---

### Update Schedule
Update an existing schedule.

**Endpoint**: `PUT /api/v1/scheduler/schedules/{id}`

**Headers**: `Authorization: Bearer <token>`

**Path Parameters**:
- `id` (int): Schedule ID

**Request Body**:
```json
{
  "name": "Updated Catalog Refresh",
  "cron_expr": "0 2 * * *",
  "status": "paused",
  "parameters": {
    "force": true
  }
}
```

**Response (200)**:
```json
{
  "success": true,
  "message": "Schedule updated successfully"
}
```

---

### Delete Schedule
Delete a schedule.

**Endpoint**: `DELETE /api/v1/scheduler/schedules/{id}`

**Headers**: `Authorization: Bearer <token>`

**Path Parameters**:
- `id` (int): Schedule ID

**Response (200)**:
```json
{
  "success": true,
  "message": "Schedule deleted successfully"
}
```

---

### Bulk Schedule Operations
Perform bulk operations on multiple schedules.

**Endpoint**: `POST /api/v1/scheduler/schedules/bulk`

**Headers**: `Authorization: Bearer <token>`

**Request Body**:
```json
{
  "schedule_ids": [1, 2, 3, 4],
  "operation": "pause",  // enable, disable, pause, resume, delete
  "parameters": {}
}
```

**Response (200)**:
```json
{
  "success": true,
  "processed_count": 4,
  "success_count": 4,
  "failed_count": 0,
  "errors": [],
  "message": "Bulk operation completed successfully"
}
```

---

### Get Schedule Executions
Get execution history for a specific schedule.

**Endpoint**: `GET /api/v1/scheduler/schedules/{id}/executions`

**Headers**: `Authorization: Bearer <token>`

**Path Parameters**:
- `id` (int): Schedule ID

**Query Parameters**:
- `page` (int): Page number
- `page_size` (int): Items per page
- `status` (string): Filter by execution status

**Response (200)**:
```json
{
  "executions": [
    {
      "id": 5001,
      "schedule_id": 1,
      "job_id": "schedule-1-20240116-030000",
      "status": "completed",
      "started_at": "2024-01-16T03:00:00Z",
      "completed_at": "2024-01-16T03:07:22Z",
      "duration_ms": 442000,
      "result": {
        "artists_processed": 200,
        "new_shows": 12
      },
      "error": null
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total": 45,
    "has_next": true,
    "has_prev": false
  }
}
```

---

### Get All Executions
Get execution history across all schedules.

**Endpoint**: `GET /api/v1/scheduler/executions`

**Headers**: `Authorization: Bearer <token>`

**Query Parameters**:
- `page` (int): Page number
- `page_size` (int): Items per page
- `schedule_id` (int): Filter by schedule
- `status` (string): Filter by execution status

**Response (200)**:
```json
{
  "executions": [
    {
      "id": 5001,
      "schedule_id": 1,
      "schedule_name": "Daily Catalog Refresh",
      "schedule_type": "catalog_refresh",
      "job_id": "schedule-1-20240116-030000",
      "status": "completed",
      "started_at": "2024-01-16T03:00:00Z",
      "completed_at": "2024-01-16T03:07:22Z",
      "duration_ms": 442000
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total": 5420,
    "has_next": true,
    "has_prev": false
  }
}
```

---

### Get Schedule Templates
Get predefined schedule templates.

**Endpoint**: `GET /api/v1/scheduler/templates`

**Headers**: `Authorization: Bearer <token>`

**Response (200)**:
```json
{
  "templates": [
    {
      "name": "Daily Catalog Refresh",
      "description": "Refresh the catalog daily at 3 AM",
      "type": "catalog_refresh",
      "cron_expr": "0 3 * * *",
      "parameters": {
        "force": false
      },
      "category": "Data Management"
    },
    {
      "name": "Hourly Monitor Check",
      "description": "Check all active monitors every hour",
      "type": "monitor_check",
      "cron_expr": "0 * * * *",
      "parameters": {},
      "category": "Monitoring"
    }
  ]
}
```

---

### Get Cron Patterns
Get common cron expression patterns.

**Endpoint**: `GET /api/v1/scheduler/cron-patterns`

**Headers**: `Authorization: Bearer <token>`

**Response (200)**:
```json
{
  "patterns": [
    {
      "expression": "* * * * *",
      "description": "Every minute",
      "example": "Runs every minute"
    },
    {
      "expression": "0 * * * *",
      "description": "Every hour",
      "example": "Runs at the start of every hour"
    },
    {
      "expression": "0 0 * * *",
      "description": "Daily at midnight",
      "example": "Runs once per day at 00:00"
    },
    {
      "expression": "0 0 * * 0",
      "description": "Weekly on Sunday",
      "example": "Runs once per week on Sunday at midnight"
    },
    {
      "expression": "0 0 1 * *",
      "description": "Monthly",
      "example": "Runs once per month on the 1st at midnight"
    }
  ]
}
```

---

# Data Models

## User Model
```json
{
  "id": 1,
  "username": "admin",
  "email": "admin@example.com",
  "role": "admin",
  "active": true,
  "last_login": "2024-01-16T14:30:00Z",
  "login_count": 245,
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-16T14:30:00Z"
}
```

## Artist Model
```json
{
  "id": 1,
  "name": "Grateful Dead",
  "slug": "grateful-dead",
  "show_count": 2847,
  "first_show_date": "1965-05-05",
  "last_show_date": "1995-07-09",
  "is_active": false,
  "genres": ["Rock", "Psychedelic"],
  "bio": "American rock band formed in 1965...",
  "image_url": "https://example.com/artists/grateful-dead.jpg",
  "social_links": {
    "website": "https://dead.net",
    "wikipedia": "https://en.wikipedia.org/wiki/Grateful_Dead"
  }
}
```

## Show Model
```json
{
  "id": 12345,
  "artist_id": 1,
  "artist_name": "Grateful Dead",
  "date": "1977-05-08",
  "venue": "Barton Hall, Cornell University",
  "city": "Ithaca",
  "state": "NY",
  "country": "USA",
  "container_id": 67890,
  "duration_minutes": 180,
  "set_list": ["Set 1: ...", "Set 2: ..."],
  "notes": "Legendary Cornell '77 show...",
  "download_count": 5420,
  "rating": 4.9,
  "is_available": true,
  "formats": {
    "MP3": ["320kbps", "256kbps"],
    "FLAC": ["16bit/44.1kHz"],
    "ALAC": ["16bit/44.1kHz"]
  }
}
```

## Download Model
```json
{
  "id": 1001,
  "show_id": 12345,
  "container_id": 67890,
  "artist_name": "Grateful Dead",
  "show_date": "1977-05-08",
  "venue": "Barton Hall, Cornell University",
  "format": "FLAC",
  "quality": "16bit/44.1kHz",
  "status": "completed",
  "progress": 100,
  "file_path": "/downloads/grateful-dead/1977-05-08/gd77-05-08.flac",
  "file_size": 687194624,
  "downloaded_at": "2024-01-15T14:22:30Z",
  "created_at": "2024-01-15T14:15:12Z"
}
```

## Monitor Model
```json
{
  "id": 501,
  "artist_id": 1,
  "artist_name": "Grateful Dead",
  "status": "active",
  "settings": {
    "check_frequency": "daily",
    "notify_new": true,
    "notify_updates": false,
    "auto_download": false,
    "download_format": "FLAC",
    "download_quality": "16bit/44.1kHz",
    "min_rating": 4.0
  },
  "stats": {
    "shows_found": 15,
    "alerts_sent": 8,
    "last_check": "2024-01-16T03:00:00Z",
    "next_check": "2024-01-17T03:00:00Z"
  },
  "created_at": "2024-01-10T15:30:00Z"
}
```

## Job Model
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "type": "catalog_refresh",
  "status": "running",
  "progress": 75,
  "message": "Processing artist 150 of 200",
  "created_at": "2024-01-15T10:30:00Z",
  "started_at": "2024-01-15T10:30:05Z",
  "estimated_completion": "2024-01-15T10:35:00Z",
  "result": null,
  "error": null
}
```

## Webhook Model
```json
{
  "id": 101,
  "url": "https://your-server.com/webhooks/nugs",
  "events": ["new_show", "download_complete"],
  "status": "active",
  "timeout": 30,
  "retry_count": 3,
  "secret": "your_webhook_secret",
  "description": "Main webhook for notifications",
  "stats": {
    "total_deliveries": 156,
    "successful_deliveries": 154,
    "failed_deliveries": 2,
    "success_rate": 98.7
  },
  "created_at": "2024-01-10T12:00:00Z"
}
```

## Schedule Model
```json
{
  "id": 1,
  "name": "Daily Catalog Refresh",
  "description": "Refresh the catalog daily at 3 AM",
  "type": "catalog_refresh",
  "cron_expr": "0 3 * * *",
  "status": "active",
  "parameters": {
    "force": false,
    "artists": []
  },
  "next_run": "2024-01-17T03:00:00Z",
  "last_run": "2024-01-16T03:00:00Z",
  "run_count": 45,
  "fail_count": 2,
  "created_at": "2024-01-01T00:00:00Z",
  "created_by": "admin"
}
```

---

# Background Jobs

The API uses a comprehensive background job system for long-running operations.

## Job Types
- `catalog_refresh`: Refresh catalog data from Nugs API
- `download`: Download audio files
- `monitor_check`: Check monitors for new content
- `system_cleanup`: System maintenance tasks
- `database_backup`: Create database backups
- `webhook_delivery`: Deliver webhook notifications

## Job Statuses
- `pending`: Job queued but not started
- `running`: Job currently executing
- `completed`: Job finished successfully
- `failed`: Job encountered an error
- `canceled`: Job was canceled

## Job Management
- Jobs can be monitored via `/api/v1/admin/jobs`
- Running jobs can be canceled
- Failed jobs can be retried
- Job logs and results are preserved

---

# Webhook Events

## Available Events

### new_show
Triggered when a new show is found by the monitoring system.

**Payload**:
```json
{
  "event": "new_show",
  "timestamp": "2024-01-16T14:30:00Z",
  "data": {
    "show_id": 12346,
    "artist_id": 1,
    "artist_name": "Grateful Dead",
    "date": "1978-05-08",
    "venue": "Barton Hall",
    "city": "Ithaca",
    "state": "NY",
    "container_id": 67892,
    "monitor_id": 501
  }
}
```

### download_complete
Triggered when a download finishes (success or failure).

**Payload**:
```json
{
  "event": "download_complete",
  "timestamp": "2024-01-16T14:30:00Z",
  "data": {
    "download_id": 1001,
    "show_id": 12345,
    "artist_name": "Grateful Dead",
    "show_date": "1977-05-08",
    "status": "completed",
    "file_path": "/downloads/gd77-05-08.flac",
    "file_size": 687194624,
    "duration_seconds": 437
  }
}
```

### download_started
Triggered when a download begins.

**Payload**:
```json
{
  "event": "download_started",
  "timestamp": "2024-01-16T14:15:00Z",
  "data": {
    "download_id": 1001,
    "show_id": 12345,
    "artist_name": "Grateful Dead",
    "format": "FLAC",
    "quality": "16bit/44.1kHz",
    "estimated_size": 687194624
  }
}
```

### monitor_alert
Triggered when a monitor generates an alert.

**Payload**:
```json
{
  "event": "monitor_alert",
  "timestamp": "2024-01-16T14:30:00Z",
  "data": {
    "alert_id": 1001,
    "monitor_id": 501,
    "artist_id": 1,
    "artist_name": "Grateful Dead",
    "alert_type": "new_show",
    "message": "New show found: 1978-05-08 at Barton Hall"
  }
}
```

### catalog_refresh
Triggered when a catalog refresh completes.

**Payload**:
```json
{
  "event": "catalog_refresh",
  "timestamp": "2024-01-16T03:07:22Z",
  "data": {
    "job_id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "completed",
    "duration_seconds": 442,
    "artists_processed": 200,
    "new_shows": 12,
    "updated_shows": 5
  }
}
```

### system_error
Triggered when a system error occurs.

**Payload**:
```json
{
  "event": "system_error",
  "timestamp": "2024-01-16T14:30:00Z",
  "data": {
    "error_type": "download_failed",
    "error_message": "Connection timeout",
    "component": "download_manager",
    "severity": "warning",
    "details": {
      "download_id": 1001,
      "retry_count": 3
    }
  }
}
```

## Webhook Security

All webhook payloads include a signature header for verification:

```
X-Nugs-Signature: sha256=1a2b3c4d5e6f...
```

The signature is calculated as:
```
HMAC-SHA256(webhook_secret, payload_body)
```

## Webhook Delivery

- **Timeout**: Configurable per webhook (default: 30 seconds)
- **Retries**: Configurable retry count with exponential backoff
- **Verification**: SSL certificate verification enforced
- **Delivery Tracking**: Full delivery history and statistics

---

# Examples

## Authentication Flow
```bash
# 1. Login
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'

# Response: {"success":true,"token":"eyJ...","user":{...}}

# 2. Use token for subsequent requests
export TOKEN="eyJ..."
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/catalog/artists
```

## Complete Download Workflow
```bash
# 1. Search for shows
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/api/v1/catalog/shows/search?artist=grateful+dead&date_from=1977-01-01&date_to=1977-12-31"

# 2. Queue a download
curl -X POST \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"container_id":67890,"format":"FLAC","quality":"16bit/44.1kHz"}' \
  http://localhost:8080/api/v1/downloads/queue

# 3. Check download status
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/downloads/1001

# 4. Monitor queue
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/downloads/queue
```

## Setting Up Monitoring
```bash
# 1. Create a monitor
curl -X POST \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "artist_id": 1,
    "settings": {
      "check_frequency": "daily",
      "notify_new": true,
      "auto_download": true,
      "download_format": "FLAC"
    }
  }' \
  http://localhost:8080/api/v1/monitoring/monitors

# 2. Trigger manual check
curl -X POST \
  -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/monitoring/check/artist/1

# 3. View alerts
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/monitoring/alerts
```

## Webhook Setup
```bash
# 1. Create webhook
curl -X POST \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://your-server.com/webhooks",
    "events": ["new_show", "download_complete"],
    "secret": "your_secret_key",
    "timeout": 30
  }' \
  http://localhost:8080/api/v1/webhooks

# 2. Test webhook
curl -X POST \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"event": "new_show"}' \
  http://localhost:8080/api/v1/webhooks/101/test
```

## Analytics Queries
```bash
# Collection overview
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/analytics/collection

# Download statistics
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/api/v1/analytics/downloads?period=last_30_days"

# Top artists by downloads
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/api/v1/analytics/top/artists?metric=downloads&limit=10"

# System health
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/analytics/health
```

## Scheduled Tasks
```bash
# 1. Create daily catalog refresh
curl -X POST \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Daily Catalog Refresh",
    "type": "catalog_refresh",
    "cron_expr": "0 3 * * *",
    "parameters": {"force": false}
  }' \
  http://localhost:8080/api/v1/scheduler/schedules

# 2. Check scheduler status
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/scheduler/status

# 3. View execution history
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/scheduler/executions
```

---

## Support

For questions, issues, or feature requests:
- GitHub Issues: https://github.com/your-org/nugs-api/issues
- Documentation: https://docs.your-domain.com/nugs-api
- API Status: https://status.your-domain.com

## Changelog

### v1.0.0 (2024-01-16)
- Initial release
- Complete catalog management
- Download queue system
- Monitoring and alerts
- Comprehensive analytics
- Webhook notifications
- Background scheduler
- User management
- Full API documentation

---

*Last updated: January 16, 2024*