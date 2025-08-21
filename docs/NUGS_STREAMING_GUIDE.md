# Nugs.net Streaming Implementation Guide

This guide provides detailed implementation instructions for streaming audio and video content from Nugs.net.

## Table of Contents

1. [Streaming Overview](#streaming-overview)
2. [Audio Streaming](#audio-streaming)
3. [Video Streaming](#video-streaming)
4. [Quality Management](#quality-management)
5. [HLS Implementation](#hls-implementation)
6. [Encryption & Decryption](#encryption--decryption)
7. [Download Management](#download-management)
8. [FFmpeg Integration](#ffmpeg-integration)
9. [Error Handling](#error-handling)

## Streaming Overview

Nugs.net uses different streaming protocols based on content type:

- **Direct HTTP Streaming**: For high-quality audio formats (ALAC, FLAC, MQA)
- **HLS (HTTP Live Streaming)**: For adaptive audio and all video content
- **Encrypted Streams**: Some content uses AES-128-CBC encryption

### Content Type Detection

```go
func determineContentType(url string) ContentType {
    if strings.Contains(url, ".m3u8") {
        return HLSContent
    }
    if strings.Contains(url, ".alac16/") || strings.Contains(url, ".flac16/") {
        return DirectHTTPContent
    }
    return UnknownContent
}
```

## Audio Streaming

### Stream URL Generation

```go
func getStreamURL(trackID int, format int, streamParams *StreamParams) (string, error) {
    baseURL := "https://streamapi.nugs.net/bigriver/subPlayer.aspx"
    
    params := url.Values{}
    params.Set("platformID", strconv.Itoa(format))
    params.Set("trackID", strconv.Itoa(trackID))
    params.Set("app", "1")
    params.Set("subscriptionID", streamParams.SubscriptionID)
    params.Set("subCostplanIDAccessList", streamParams.SubCostplanIDAccessList)
    params.Set("nn_userID", streamParams.UserID)
    params.Set("startDateStamp", streamParams.StartStamp)
    params.Set("endDateStamp", streamParams.EndStamp)
    
    req, err := http.NewRequest(http.MethodGet, baseURL, nil)
    if err != nil {
        return "", err
    }
    
    req.URL.RawQuery = params.Encode()
    req.Header.Add("User-Agent", "nugsnetAndroid")
    
    resp, err := client.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return "", errors.New(resp.Status)
    }
    
    var streamMeta struct {
        StreamLink string `json:"streamLink"`
    }
    
    err = json.NewDecoder(resp.Body).Decode(&streamMeta)
    if err != nil {
        return "", err
    }
    
    return streamMeta.StreamLink, nil
}
```

### Quality Detection and Mapping

```go
var qualityMap = map[string]Quality{
    ".alac16/": {Specs: "16-bit / 44.1 kHz ALAC", Extension: ".m4a", Format: 1},
    ".flac16/": {Specs: "16-bit / 44.1 kHz FLAC", Extension: ".flac", Format: 2},
    ".mqa24/":  {Specs: "24-bit / 48 kHz MQA", Extension: ".flac", Format: 3},
    ".s360/":   {Specs: "360 Reality Audio", Extension: ".mp4", Format: 4},
    ".aac150/": {Specs: "150 Kbps AAC", Extension: ".m4a", Format: 5},
    ".m3u8":    {Extension: ".m4a", Format: 6}, // HLS
}

func queryQuality(streamURL string) *Quality {
    for pattern, quality := range qualityMap {
        if strings.Contains(streamURL, pattern) {
            quality.URL = streamURL
            return &quality
        }
    }
    return nil
}
```

### Multi-Format Quality Negotiation

```go
func getAvailableQualities(trackID int, streamParams *StreamParams) ([]*Quality, error) {
    var qualities []*Quality
    
    // Try multiple format IDs to get all available qualities
    formatIDs := []int{1, 4, 7, 10}
    
    for _, formatID := range formatIDs {
        streamURL, err := getStreamURL(trackID, formatID, streamParams)
        if err != nil {
            continue // Skip failed formats
        }
        
        if streamURL == "" {
            continue // Skip empty responses
        }
        
        quality := queryQuality(streamURL)
        if quality != nil {
            qualities = append(qualities, quality)
        }
    }
    
    return qualities, nil
}

func selectQuality(qualities []*Quality, wantedFormat int) *Quality {
    // Try to get exactly what was requested
    for _, quality := range qualities {
        if quality.Format == wantedFormat {
            return quality
        }
    }
    
    // Fallback chain
    fallbacks := map[int]int{
        1: 2, // ALAC → FLAC
        2: 5, // FLAC → AAC
        3: 2, // MQA → FLAC
        4: 3, // 360 → MQA
    }
    
    if fallback, exists := fallbacks[wantedFormat]; exists {
        return selectQuality(qualities, fallback)
    }
    
    // Return first available if no fallback
    if len(qualities) > 0 {
        return qualities[0]
    }
    
    return nil
}
```

## Video Streaming

### Video Stream URL Generation

```go
func getVideoStreamURL(containerID, skuID int, streamParams *StreamParams) (string, error) {
    baseURL := "https://streamapi.nugs.net/bigriver/subPlayer.aspx"
    
    params := url.Values{}
    params.Set("skuId", strconv.Itoa(skuID))
    params.Set("containerID", strconv.Itoa(containerID))
    params.Set("chap", "1") // Enable chapters
    params.Set("app", "1")
    params.Set("subscriptionID", streamParams.SubscriptionID)
    params.Set("subCostplanIDAccessList", streamParams.SubCostplanIDAccessList)
    params.Set("nn_userID", streamParams.UserID)
    params.Set("startDateStamp", streamParams.StartStamp)
    params.Set("endDateStamp", streamParams.EndStamp)
    
    // Implementation similar to audio stream URL generation
    // Returns HLS manifest URL
}
```

### Video Quality Selection

```go
func selectVideoVariant(manifestURL, wantedResolution string) (*Variant, string, error) {
    playlist, err := parseHLSMaster(manifestURL)
    if err != nil {
        return nil, "", err
    }
    
    // Sort by bandwidth (highest first)
    sort.Slice(playlist.Variants, func(i, j int) bool {
        return playlist.Variants[i].Bandwidth > playlist.Variants[j].Bandwidth
    })
    
    // For 4K, take highest bandwidth
    if wantedResolution == "2160" {
        variant := playlist.Variants[0]
        return variant, "4K", nil
    }
    
    // Find exact resolution match
    for _, variant := range playlist.Variants {
        if strings.HasSuffix(variant.Resolution, "x"+wantedResolution) {
            return variant, formatResolution(wantedResolution), nil
        }
    }
    
    // Fallback to lower resolution
    fallbacks := map[string]string{
        "1440": "1080",
        "1080": "720",
        "720":  "480",
    }
    
    if fallback, exists := fallbacks[wantedResolution]; exists {
        return selectVideoVariant(manifestURL, fallback)
    }
    
    return nil, "", errors.New("no suitable variant found")
}

func formatResolution(res string) string {
    if res == "2160" {
        return "4K"
    }
    return res + "p"
}
```

## Quality Management

### Format Priority System

```go
type QualityPreference struct {
    PreferredFormat int
    FallbackChain   []int
    MaxBitrate      int
}

func (qp *QualityPreference) SelectBest(available []*Quality) *Quality {
    // Try preferred format first
    for _, quality := range available {
        if quality.Format == qp.PreferredFormat {
            return quality
        }
    }
    
    // Try fallback chain
    for _, fallbackFormat := range qp.FallbackChain {
        for _, quality := range available {
            if quality.Format == fallbackFormat {
                return quality
            }
        }
    }
    
    // Return highest quality available
    if len(available) > 0 {
        return available[0]
    }
    
    return nil
}
```

### Adaptive Quality Selection

```go
func adaptiveQualitySelection(available []*Quality, bandwidth int) *Quality {
    // Estimate required bandwidth for each format
    bandwidthRequirements := map[int]int{
        1: 1000000, // ALAC ~1Mbps
        2: 800000,  // FLAC ~800kbps
        3: 2000000, // MQA ~2Mbps
        4: 3000000, // 360 Reality ~3Mbps
        5: 150000,  // AAC 150kbps
    }
    
    // Select highest quality that fits bandwidth
    for _, quality := range available {
        if required, exists := bandwidthRequirements[quality.Format]; exists {
            if bandwidth >= required {
                return quality
            }
        }
    }
    
    // Fallback to lowest quality
    for _, quality := range available {
        if quality.Format == 5 { // AAC
            return quality
        }
    }
    
    return nil
}
```

## HLS Implementation

### Master Playlist Parsing

```go
func parseHLSMaster(manifestURL string) (*MasterPlaylist, error) {
    resp, err := http.Get(manifestURL)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return nil, errors.New(resp.Status)
    }
    
    playlist, _, err := m3u8.DecodeFrom(resp.Body, true)
    if err != nil {
        return nil, err
    }
    
    master, ok := playlist.(*m3u8.MasterPlaylist)
    if !ok {
        return nil, errors.New("not a master playlist")
    }
    
    return master, nil
}
```

### Media Playlist Processing

```go
func parseMediaPlaylist(playlistURL string) (*MediaPlaylist, error) {
    resp, err := http.Get(playlistURL)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    playlist, _, err := m3u8.DecodeFrom(resp.Body, true)
    if err != nil {
        return nil, err
    }
    
    media, ok := playlist.(*m3u8.MediaPlaylist)
    if !ok {
        return nil, errors.New("not a media playlist")
    }
    
    return media, nil
}

func extractSegmentURLs(playlist *MediaPlaylist, baseURL string) []string {
    var segmentURLs []string
    
    for _, segment := range playlist.Segments {
        if segment == nil {
            break
        }
        segmentURLs = append(segmentURLs, baseURL+segment.URI)
    }
    
    return segmentURLs
}
```

### Manifest Base URL Extraction

```go
func getManifestBase(manifestURL string) (string, string, error) {
    u, err := url.Parse(manifestURL)
    if err != nil {
        return "", "", err
    }
    
    path := u.Path
    lastSlash := strings.LastIndex(path, "/")
    
    baseURL := u.Scheme + "://" + u.Host + path[:lastSlash+1]
    query := "?" + u.RawQuery
    
    return baseURL, query, nil
}
```

## Encryption & Decryption

### AES Key Retrieval

```go
func getEncryptionKey(keyURL string) ([]byte, error) {
    resp, err := http.Get(keyURL)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return nil, errors.New(resp.Status)
    }
    
    key := make([]byte, 16) // AES-128 key size
    _, err = io.ReadFull(resp.Body, key)
    if err != nil {
        return nil, err
    }
    
    return key, nil
}
```

### AES-CBC Decryption

```go
func decryptSegment(encryptedData, key, iv []byte) ([]byte, error) {
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, err
    }
    
    if len(encryptedData)%aes.BlockSize != 0 {
        return nil, errors.New("encrypted data is not a multiple of block size")
    }
    
    mode := cipher.NewCBCDecrypter(block, iv)
    decrypted := make([]byte, len(encryptedData))
    mode.CryptBlocks(decrypted, encryptedData)
    
    // Remove PKCS5 padding
    return pkcs5Unpad(decrypted), nil
}

func pkcs5Unpad(data []byte) []byte {
    padding := data[len(data)-1]
    return data[:len(data)-int(padding)]
}
```

### IV Extraction from HLS

```go
func extractIVFromHLS(ivHex string) ([]byte, error) {
    // IV format in HLS: "0x{hex}" or just "{hex}"
    ivHex = strings.TrimPrefix(ivHex, "0x")
    
    iv, err := hex.DecodeString(ivHex)
    if err != nil {
        return nil, err
    }
    
    if len(iv) != 16 {
        return nil, errors.New("IV must be 16 bytes")
    }
    
    return iv, nil
}
```

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
    elapsed := time.Since(dp.StartTime)
    speed := float64(dp.Downloaded) / elapsed.Seconds()
    
    fmt.Printf("\rProgress: %.1f%% @ %.2f KB/s", percentage, speed/1024)
    
    return n, nil
}
```

### Resumable Downloads

```go
func downloadWithResume(url, filepath string) error {
    // Check existing file size
    var startByte int64 = 0
    if info, err := os.Stat(filepath); err == nil {
        startByte = info.Size()
    }
    
    // Open file for appending
    file, err := os.OpenFile(filepath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
    if err != nil {
        return err
    }
    defer file.Close()
    
    // Create request with Range header
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return err
    }
    
    if startByte > 0 {
        req.Header.Set("Range", fmt.Sprintf("bytes=%d-", startByte))
    }
    
    req.Header.Set("Referer", "https://play.nugs.net/")
    req.Header.Set("User-Agent", userAgent)
    
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    // Copy with progress tracking
    progress := &DownloadProgress{
        Total:      resp.ContentLength + startByte,
        Downloaded: startByte,
        StartTime:  time.Now(),
    }
    
    _, err = io.Copy(file, io.TeeReader(resp.Body, progress))
    return err
}
```

### Concurrent Segment Downloads

```go
func downloadSegmentsConcurrently(segmentURLs []string, maxWorkers int) error {
    semaphore := make(chan struct{}, maxWorkers)
    errChan := make(chan error, len(segmentURLs))
    
    for i, segmentURL := range segmentURLs {
        go func(index int, url string) {
            semaphore <- struct{}{} // Acquire
            defer func() { <-semaphore }() // Release
            
            err := downloadSegment(url, fmt.Sprintf("segment_%04d.ts", index))
            errChan <- err
        }(i, segmentURL)
    }
    
    // Wait for all downloads to complete
    for i := 0; i < len(segmentURLs); i++ {
        if err := <-errChan; err != nil {
            return err
        }
    }
    
    return nil
}
```

## FFmpeg Integration

### TS to MP4 Conversion

```go
func convertTSToMP4(inputPath, outputPath string, ffmpegPath string) error {
    cmd := exec.Command(ffmpegPath, 
        "-hide_banner",
        "-i", inputPath,
        "-c", "copy", // Stream copy (no re-encoding)
        outputPath)
    
    var stderr bytes.Buffer
    cmd.Stderr = &stderr
    
    err := cmd.Run()
    if err != nil {
        return fmt.Errorf("ffmpeg error: %v\nStderr: %s", err, stderr.String())
    }
    
    return nil
}
```

### Chapter Integration

```go
func addChaptersToVideo(inputPath, outputPath, chaptersFile string, ffmpegPath string) error {
    cmd := exec.Command(ffmpegPath,
        "-hide_banner",
        "-i", inputPath,
        "-f", "ffmetadata",
        "-i", chaptersFile,
        "-map_metadata", "1",
        "-c", "copy",
        outputPath)
    
    return cmd.Run()
}

func writeChaptersFile(chapters []Chapter, duration int, filename string) error {
    file, err := os.Create(filename)
    if err != nil {
        return err
    }
    defer file.Close()
    
    // Write FFmetadata header
    fmt.Fprintln(file, ";FFMETADATA1")
    
    for i, chapter := range chapters {
        fmt.Fprintf(file, "\n[CHAPTER]\n")
        fmt.Fprintf(file, "TIMEBASE=1/1\n")
        fmt.Fprintf(file, "START=%d\n", int(chapter.StartTime))
        
        if i < len(chapters)-1 {
            fmt.Fprintf(file, "END=%d\n", int(chapters[i+1].StartTime)-1)
        } else {
            fmt.Fprintf(file, "END=%d\n", duration)
        }
        
        fmt.Fprintf(file, "TITLE=%s\n", chapter.Title)
    }
    
    return nil
}
```

### Audio Format Conversion

```go
func convertTSToAAC(inputPath, outputPath string, ffmpegPath string) error {
    cmd := exec.Command(ffmpegPath,
        "-hide_banner",
        "-i", inputPath,
        "-c:a", "copy", // Copy audio without re-encoding
        outputPath)
    
    return cmd.Run()
}
```

## Error Handling

### Retry Logic

```go
type RetryConfig struct {
    MaxRetries int
    BackoffBase time.Duration
    MaxBackoff  time.Duration
}

func (rc *RetryConfig) Execute(operation func() error) error {
    var lastErr error
    
    for attempt := 0; attempt <= rc.MaxRetries; attempt++ {
        err := operation()
        if err == nil {
            return nil
        }
        
        lastErr = err
        
        if attempt < rc.MaxRetries {
            backoff := time.Duration(float64(rc.BackoffBase) * math.Pow(2, float64(attempt)))
            if backoff > rc.MaxBackoff {
                backoff = rc.MaxBackoff
            }
            time.Sleep(backoff)
        }
    }
    
    return fmt.Errorf("operation failed after %d attempts: %w", rc.MaxRetries+1, lastErr)
}
```

### Network Error Handling

```go
func isRetryableError(err error) bool {
    if err == nil {
        return false
    }
    
    // Check for network timeouts
    if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
        return true
    }
    
    // Check for temporary network errors
    if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
        return true
    }
    
    // Check for specific HTTP status codes
    if strings.Contains(err.Error(), "500") || 
       strings.Contains(err.Error(), "502") ||
       strings.Contains(err.Error(), "503") ||
       strings.Contains(err.Error(), "504") {
        return true
    }
    
    return false
}
```

### Stream Validation

```go
func validateStreamURL(streamURL string) error {
    if streamURL == "" {
        return errors.New("empty stream URL")
    }
    
    resp, err := http.Head(streamURL)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("stream not available: %s", resp.Status)
    }
    
    return nil
}
```

---

*This streaming implementation guide provides the technical foundation for building a robust Nugs.net streaming client. Always respect the service's terms of use and implement appropriate rate limiting.*