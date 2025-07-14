package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// CacheItem represents a cached item with expiration
type CacheItem struct {
	Data       interface{} `json:"data"`
	ExpiresAt  time.Time   `json:"expires_at"`
	CreatedAt  time.Time   `json:"created_at"`
	AccessedAt time.Time   `json:"accessed_at"`
	HitCount   int64       `json:"hit_count"`
}

// IsExpired checks if the cache item has expired
func (c *CacheItem) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}

// Touch updates the access time and increments hit count
func (c *CacheItem) Touch() {
	c.AccessedAt = time.Now()
	c.HitCount++
}

// MultiLayerCache implements a 3-tier caching system:
// 1. L1: In-memory LRU cache (fastest, limited size)
// 2. L2: Disk-based cache (moderate speed, larger capacity)
// 3. L3: Query result cache (persistent, optimized for search results)
type MultiLayerCache struct {
	// L1 Cache - In-memory LRU
	l1Cache   map[string]*CacheItem
	l1Order   []string // LRU order tracking
	l1Mutex   sync.RWMutex
	l1MaxSize int
	l1TTL     time.Duration

	// L2 Cache - Disk-based
	l2Dir         string
	l2Mutex       sync.RWMutex
	l2TTL         time.Duration
	l2MaxSize     int64 // in bytes
	l2CurrentSize int64

	// L3 Cache - Query results
	l3Cache map[string]*CacheItem
	l3Mutex sync.RWMutex
	l3TTL   time.Duration

	// Statistics
	stats      CacheStats
	statsMutex sync.RWMutex
}

// CacheStats tracks cache performance metrics
type CacheStats struct {
	L1Hits    int64 `json:"l1_hits"`
	L1Misses  int64 `json:"l1_misses"`
	L2Hits    int64 `json:"l2_hits"`
	L2Misses  int64 `json:"l2_misses"`
	L3Hits    int64 `json:"l3_hits"`
	L3Misses  int64 `json:"l3_misses"`
	Evictions int64 `json:"evictions"`
}

// CacheConfig holds configuration for the cache system
type CacheConfig struct {
	L1MaxItems  int           `json:"l1_max_items"`
	L1TTL       time.Duration `json:"l1_ttl"`
	L2Dir       string        `json:"l2_dir"`
	L2MaxSizeMB int64         `json:"l2_max_size_mb"`
	L2TTL       time.Duration `json:"l2_ttl"`
	L3TTL       time.Duration `json:"l3_ttl"`
}

// DefaultCacheConfig returns sensible defaults
func DefaultCacheConfig() *CacheConfig {
	return &CacheConfig{
		L1MaxItems:  1000,
		L1TTL:       30 * time.Minute,
		L2Dir:       "./cache/disk",
		L2MaxSizeMB: 500, // 500MB
		L2TTL:       24 * time.Hour,
		L3TTL:       1 * time.Hour,
	}
}

// NewMultiLayerCache creates a new multi-layer cache
func NewMultiLayerCache(config *CacheConfig) (*MultiLayerCache, error) {
	if config == nil {
		config = DefaultCacheConfig()
	}

	// Ensure L2 cache directory exists
	if err := os.MkdirAll(config.L2Dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create L2 cache directory: %w", err)
	}

	cache := &MultiLayerCache{
		l1Cache:   make(map[string]*CacheItem),
		l1Order:   make([]string, 0, config.L1MaxItems),
		l1MaxSize: config.L1MaxItems,
		l1TTL:     config.L1TTL,

		l2Dir:     config.L2Dir,
		l2TTL:     config.L2TTL,
		l2MaxSize: config.L2MaxSizeMB * 1024 * 1024, // Convert MB to bytes

		l3Cache: make(map[string]*CacheItem),
		l3TTL:   config.L3TTL,

		stats: CacheStats{},
	}

	// Start cleanup goroutine
	go cache.cleanupLoop()

	// Load existing L2 cache size
	cache.calculateL2Size()

	log.Printf("Multi-layer cache initialized: L1=%d items, L2=%s (max %dMB), L3=query results",
		config.L1MaxItems, config.L2Dir, config.L2MaxSizeMB)

	return cache, nil
}

// generateKey creates a consistent hash key from the input
func (c *MultiLayerCache) generateKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:16]) // Use first 16 bytes for shorter keys
}

// Get retrieves an item from the cache, checking all layers
func (c *MultiLayerCache) Get(key string) (interface{}, bool) {
	hashKey := c.generateKey(key)

	// L1 Cache check (fastest)
	if data, found := c.getFromL1(hashKey); found {
		c.incrementStat("l1_hits")
		return data, true
	}
	c.incrementStat("l1_misses")

	// L2 Cache check (disk)
	if data, found := c.getFromL2(hashKey); found {
		c.incrementStat("l2_hits")
		// Promote to L1
		c.setToL1(hashKey, data)
		return data, true
	}
	c.incrementStat("l2_misses")

	// L3 Cache check (query results)
	if data, found := c.getFromL3(hashKey); found {
		c.incrementStat("l3_hits")
		// Promote to L1 and L2
		c.setToL1(hashKey, data)
		c.setToL2(hashKey, data)
		return data, true
	}
	c.incrementStat("l3_misses")

	return nil, false
}

// Set stores an item in all appropriate cache layers
func (c *MultiLayerCache) Set(key string, data interface{}) error {
	hashKey := c.generateKey(key)

	// Store in L1 (always)
	c.setToL1(hashKey, data)

	// Store in L2 for persistence
	if err := c.setToL2(hashKey, data); err != nil {
		log.Printf("Failed to store in L2 cache: %v", err)
	}

	return nil
}

// SetQueryResult stores a query result in L3 cache
func (c *MultiLayerCache) SetQueryResult(query string, results interface{}) error {
	hashKey := c.generateKey("query:" + query)
	c.setToL3(hashKey, results)

	// Also store in L1 for fast access
	c.setToL1(hashKey, results)

	return nil
}

// GetQueryResult retrieves a query result from cache
func (c *MultiLayerCache) GetQueryResult(query string) (interface{}, bool) {
	// hashKey := c.generateKey("query:" + query)
	return c.Get("query:" + query) // This will check all layers
}

// L1 Cache methods (in-memory LRU)
func (c *MultiLayerCache) getFromL1(key string) (interface{}, bool) {
	c.l1Mutex.RLock()
	defer c.l1Mutex.RUnlock()

	item, exists := c.l1Cache[key]
	if !exists || item.IsExpired() {
		return nil, false
	}

	item.Touch()
	c.moveToFront(key)
	return item.Data, true
}

func (c *MultiLayerCache) setToL1(key string, data interface{}) {
	c.l1Mutex.Lock()
	defer c.l1Mutex.Unlock()

	// Remove existing item if present
	if _, exists := c.l1Cache[key]; exists {
		c.removeFromOrder(key)
	}

	// Create new cache item
	item := &CacheItem{
		Data:       data,
		ExpiresAt:  time.Now().Add(c.l1TTL),
		CreatedAt:  time.Now(),
		AccessedAt: time.Now(),
		HitCount:   0,
	}

	c.l1Cache[key] = item
	c.l1Order = append([]string{key}, c.l1Order...)

	// Evict if over capacity
	if len(c.l1Cache) > c.l1MaxSize {
		c.evictFromL1()
	}
}

func (c *MultiLayerCache) evictFromL1() {
	if len(c.l1Order) == 0 {
		return
	}

	// Remove least recently used item
	oldestKey := c.l1Order[len(c.l1Order)-1]
	delete(c.l1Cache, oldestKey)
	c.l1Order = c.l1Order[:len(c.l1Order)-1]
	c.incrementStat("evictions")
}

func (c *MultiLayerCache) moveToFront(key string) {
	for i, k := range c.l1Order {
		if k == key {
			// Move to front
			c.l1Order = append([]string{key}, append(c.l1Order[:i], c.l1Order[i+1:]...)...)
			break
		}
	}
}

func (c *MultiLayerCache) removeFromOrder(key string) {
	for i, k := range c.l1Order {
		if k == key {
			c.l1Order = append(c.l1Order[:i], c.l1Order[i+1:]...)
			break
		}
	}
}

// L2 Cache methods (disk-based)
func (c *MultiLayerCache) getFromL2(key string) (interface{}, bool) {
	c.l2Mutex.RLock()
	defer c.l2Mutex.RUnlock()

	filePath := filepath.Join(c.l2Dir, key+".cache")

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, false
	}

	var item CacheItem
	if err := json.Unmarshal(data, &item); err != nil {
		log.Printf("Failed to unmarshal L2 cache item: %v", err)
		return nil, false
	}

	if item.IsExpired() {
		os.Remove(filePath) // Clean up expired file
		return nil, false
	}

	item.Touch()

	// Update file with new access time
	go func() {
		if newData, err := json.Marshal(item); err == nil {
			os.WriteFile(filePath, newData, 0644)
		}
	}()

	return item.Data, true
}

func (c *MultiLayerCache) setToL2(key string, data interface{}) error {
	c.l2Mutex.Lock()
	defer c.l2Mutex.Unlock()

	item := &CacheItem{
		Data:       data,
		ExpiresAt:  time.Now().Add(c.l2TTL),
		CreatedAt:  time.Now(),
		AccessedAt: time.Now(),
		HitCount:   0,
	}

	jsonData, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("failed to marshal cache item: %w", err)
	}

	filePath := filepath.Join(c.l2Dir, key+".cache")

	// Check if we need to evict old files
	c.checkL2Capacity(int64(len(jsonData)))

	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	c.l2CurrentSize += int64(len(jsonData))
	return nil
}

func (c *MultiLayerCache) checkL2Capacity(newItemSize int64) {
	if c.l2CurrentSize+newItemSize <= c.l2MaxSize {
		return
	}

	// Need to evict old files
	files, err := filepath.Glob(filepath.Join(c.l2Dir, "*.cache"))
	if err != nil {
		return
	}

	// Sort by modification time (oldest first)
	type fileInfo struct {
		path    string
		modTime time.Time
		size    int64
	}

	var fileList []fileInfo
	for _, file := range files {
		if info, err := os.Stat(file); err == nil {
			fileList = append(fileList, fileInfo{
				path:    file,
				modTime: info.ModTime(),
				size:    info.Size(),
			})
		}
	}

	// Sort by modification time
	for i := 0; i < len(fileList)-1; i++ {
		for j := i + 1; j < len(fileList); j++ {
			if fileList[i].modTime.After(fileList[j].modTime) {
				fileList[i], fileList[j] = fileList[j], fileList[i]
			}
		}
	}

	// Remove oldest files until we have enough space
	spaceNeeded := (c.l2CurrentSize + newItemSize) - c.l2MaxSize
	for _, file := range fileList {
		if spaceNeeded <= 0 {
			break
		}
		os.Remove(file.path)
		c.l2CurrentSize -= file.size
		spaceNeeded -= file.size
		c.incrementStat("evictions")
	}
}

func (c *MultiLayerCache) calculateL2Size() {
	files, err := filepath.Glob(filepath.Join(c.l2Dir, "*.cache"))
	if err != nil {
		return
	}

	var totalSize int64
	for _, file := range files {
		if info, err := os.Stat(file); err == nil {
			totalSize += info.Size()
		}
	}
	c.l2CurrentSize = totalSize
}

// L3 Cache methods (query results)
func (c *MultiLayerCache) getFromL3(key string) (interface{}, bool) {
	c.l3Mutex.RLock()
	defer c.l3Mutex.RUnlock()

	item, exists := c.l3Cache[key]
	if !exists || item.IsExpired() {
		return nil, false
	}

	item.Touch()
	return item.Data, true
}

func (c *MultiLayerCache) setToL3(key string, data interface{}) {
	c.l3Mutex.Lock()
	defer c.l3Mutex.Unlock()

	item := &CacheItem{
		Data:       data,
		ExpiresAt:  time.Now().Add(c.l3TTL),
		CreatedAt:  time.Now(),
		AccessedAt: time.Now(),
		HitCount:   0,
	}

	c.l3Cache[key] = item
}

// Utility methods
func (c *MultiLayerCache) incrementStat(stat string) {
	c.statsMutex.Lock()
	defer c.statsMutex.Unlock()

	switch stat {
	case "l1_hits":
		c.stats.L1Hits++
	case "l1_misses":
		c.stats.L1Misses++
	case "l2_hits":
		c.stats.L2Hits++
	case "l2_misses":
		c.stats.L2Misses++
	case "l3_hits":
		c.stats.L3Hits++
	case "l3_misses":
		c.stats.L3Misses++
	case "evictions":
		c.stats.Evictions++
	}
}

// GetStats returns cache performance statistics
func (c *MultiLayerCache) GetStats() CacheStats {
	c.statsMutex.RLock()
	defer c.statsMutex.RUnlock()
	return c.stats
}

// Clear removes all items from all cache layers
func (c *MultiLayerCache) Clear() error {
	// Clear L1
	c.l1Mutex.Lock()
	c.l1Cache = make(map[string]*CacheItem)
	c.l1Order = c.l1Order[:0]
	c.l1Mutex.Unlock()

	// Clear L2
	c.l2Mutex.Lock()
	files, _ := filepath.Glob(filepath.Join(c.l2Dir, "*.cache"))
	for _, file := range files {
		os.Remove(file)
	}
	c.l2CurrentSize = 0
	c.l2Mutex.Unlock()

	// Clear L3
	c.l3Mutex.Lock()
	c.l3Cache = make(map[string]*CacheItem)
	c.l3Mutex.Unlock()

	log.Println("All cache layers cleared")
	return nil
}

// cleanupLoop periodically removes expired items
func (c *MultiLayerCache) cleanupLoop() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanup()
	}
}

func (c *MultiLayerCache) cleanup() {
	now := time.Now()

	// Cleanup L1
	c.l1Mutex.Lock()
	for key, item := range c.l1Cache {
		if item.IsExpired() {
			delete(c.l1Cache, key)
			c.removeFromOrder(key)
		}
	}
	c.l1Mutex.Unlock()

	// Cleanup L2
	c.l2Mutex.Lock()
	files, _ := filepath.Glob(filepath.Join(c.l2Dir, "*.cache"))
	for _, file := range files {
		if info, err := os.Stat(file); err == nil {
			// If file is older than L2 TTL, remove it
			if now.Sub(info.ModTime()) > c.l2TTL {
				os.Remove(file)
				if info.Size() > 0 {
					c.l2CurrentSize -= info.Size()
				}
			}
		}
	}
	c.l2Mutex.Unlock()

	// Cleanup L3
	c.l3Mutex.Lock()
	for key, item := range c.l3Cache {
		if item.IsExpired() {
			delete(c.l3Cache, key)
		}
	}
	c.l3Mutex.Unlock()
}

// GetCacheInfo returns detailed information about cache status
func (c *MultiLayerCache) GetCacheInfo() map[string]interface{} {
	c.l1Mutex.RLock()
	l1Size := len(c.l1Cache)
	c.l1Mutex.RUnlock()

	c.l3Mutex.RLock()
	l3Size := len(c.l3Cache)
	c.l3Mutex.RUnlock()

	stats := c.GetStats()

	return map[string]interface{}{
		"l1_items":      l1Size,
		"l1_max_items":  c.l1MaxSize,
		"l2_size_bytes": c.l2CurrentSize,
		"l2_max_bytes":  c.l2MaxSize,
		"l3_items":      l3Size,
		"stats":         stats,
		"hit_rates": map[string]float64{
			"l1": float64(stats.L1Hits) / float64(stats.L1Hits+stats.L1Misses+1),
			"l2": float64(stats.L2Hits) / float64(stats.L2Hits+stats.L2Misses+1),
			"l3": float64(stats.L3Hits) / float64(stats.L3Hits+stats.L3Misses+1),
		},
	}
}
