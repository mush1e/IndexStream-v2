package handler

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strconv"

	"github.com/mush1e/IndexStream-v2/internal/service"
)

const searchPageHTML = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>🔍 IndexStream - Local Search Engine</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            color: #333;
        }

        .container {
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
        }

        .header {
            text-align: center;
            color: white;
            margin-bottom: 40px;
        }

        .header h1 {
            font-size: 3rem;
            font-weight: 700;
            margin-bottom: 10px;
            text-shadow: 2px 2px 4px rgba(0,0,0,0.3);
        }

        .header p {
            font-size: 1.2rem;
            opacity: 0.9;
        }

        .search-container {
            background: rgba(255, 255, 255, 0.95);
            backdrop-filter: blur(10px);
            border-radius: 20px;
            padding: 40px;
            box-shadow: 0 20px 40px rgba(0,0,0,0.1);
            margin-bottom: 30px;
        }

        .search-form {
            display: flex;
            gap: 15px;
            margin-bottom: 20px;
        }

        .search-input {
            flex: 1;
            padding: 15px 20px;
            border: 2px solid #e1e5e9;
            border-radius: 50px;
            font-size: 16px;
            outline: none;
            transition: all 0.3s ease;
        }

        .search-input:focus {
            border-color: #667eea;
            box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
        }

        .search-btn {
            padding: 15px 30px;
            background: linear-gradient(135deg, #667eea, #764ba2);
            color: white;
            border: none;
            border-radius: 50px;
            font-size: 16px;
            font-weight: 600;
            cursor: pointer;
            transition: all 0.3s ease;
            min-width: 120px;
        }

        .search-btn:hover {
            transform: translateY(-2px);
            box-shadow: 0 10px 20px rgba(102, 126, 234, 0.3);
        }

        .search-options {
            display: flex;
            gap: 20px;
            align-items: center;
            font-size: 14px;
            color: #666;
        }

        .search-options label {
            display: flex;
            align-items: center;
            gap: 5px;
        }

        .search-options input[type="number"] {
            width: 60px;
            padding: 5px;
            border: 1px solid #ddd;
            border-radius: 5px;
        }

        .crawl-section {
            background: rgba(255, 255, 255, 0.95);
            backdrop-filter: blur(10px);
            border-radius: 20px;
            padding: 30px;
            box-shadow: 0 20px 40px rgba(0,0,0,0.1);
            margin-bottom: 30px;
        }

        .crawl-section h2 {
            color: #333;
            margin-bottom: 20px;
            font-size: 1.5rem;
        }

        .crawl-form {
            display: flex;
            gap: 15px;
            margin-bottom: 15px;
        }

        .crawl-input {
            flex: 1;
            padding: 12px 15px;
            border: 2px solid #e1e5e9;
            border-radius: 10px;
            font-size: 14px;
        }

        .crawl-btn {
            padding: 12px 25px;
            background: linear-gradient(135deg, #764ba2, #667eea);
            color: white;
            border: none;
            border-radius: 10px;
            font-weight: 600;
            cursor: pointer;
            transition: all 0.3s ease;
        }

        .crawl-btn:hover {
            transform: translateY(-1px);
            box-shadow: 0 5px 15px rgba(118, 75, 162, 0.3);
        }

        .results-container {
            background: rgba(255, 255, 255, 0.95);
            backdrop-filter: blur(10px);
            border-radius: 20px;
            padding: 30px;
            box-shadow: 0 20px 40px rgba(0,0,0,0.1);
            margin-bottom: 30px;
            display: none;
        }

        .result-item {
            border-bottom: 1px solid #eee;
            padding: 20px 0;
        }

        .result-item:last-child {
            border-bottom: none;
        }

        .result-title {
            font-size: 1.2rem;
            font-weight: 600;
            color: #667eea;
            margin-bottom: 8px;
        }

        .result-url {
            font-size: 0.9rem;
            color: #28a745;
            margin-bottom: 8px;
            word-break: break-all;
        }

        .result-score {
            font-size: 0.8rem;
            color: #666;
            background: #f8f9fa;
            padding: 4px 8px;
            border-radius: 4px;
            display: inline-block;
        }

        .stats-container {
            background: rgba(255, 255, 255, 0.95);
            backdrop-filter: blur(10px);
            border-radius: 20px;
            padding: 30px;
            box-shadow: 0 20px 40px rgba(0,0,0,0.1);
        }

        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 20px;
        }

        .stat-card {
            background: linear-gradient(135deg, #f8f9fa, #e9ecef);
            padding: 20px;
            border-radius: 15px;
            border-left: 4px solid #667eea;
        }

        .stat-card h3 {
            color: #333;
            font-size: 0.9rem;
            margin-bottom: 10px;
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }

        .stat-value {
            font-size: 1.8rem;
            font-weight: 700;
            color: #667eea;
        }

        .loading {
            text-align: center;
            padding: 40px;
            color: #666;
        }

        .spinner {
            border: 3px solid #f3f3f3;
            border-top: 3px solid #667eea;
            border-radius: 50%;
            width: 30px;
            height: 30px;
            animation: spin 1s linear infinite;
            margin: 0 auto 20px;
        }

        @keyframes spin {
            0% { transform: rotate(0deg); }
            100% { transform: rotate(360deg); }
        }

        .cache-controls {
            margin-top: 20px;
            display: flex;
            gap: 10px;
            flex-wrap: wrap;
        }

        .cache-btn {
            padding: 8px 16px;
            background: #6c757d;
            color: white;
            border: none;
            border-radius: 6px;
            font-size: 12px;
            cursor: pointer;
            transition: all 0.2s ease;
        }

        .cache-btn:hover {
            background: #5a6268;
        }

        .cache-btn.danger {
            background: #dc3545;
        }

        .cache-btn.danger:hover {
            background: #c82333;
        }

        .message {
            padding: 15px;
            border-radius: 10px;
            margin: 15px 0;
            display: none;
        }

        .message.success {
            background: #d4edda;
            color: #155724;
            border: 1px solid #c3e6cb;
        }

        .message.error {
            background: #f8d7da;
            color: #721c24;
            border: 1px solid #f5c6cb;
        }

        @media (max-width: 768px) {
            .header h1 {
                font-size: 2rem;
            }
            
            .search-form {
                flex-direction: column;
            }
            
            .search-options {
                flex-direction: column;
                align-items: flex-start;
            }
            
            .stats-grid {
                grid-template-columns: 1fr;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🔍 IndexStream</h1>
            <p>Fast Local Search Engine with Multi-Layer Caching</p>
        </div>

        <div class="search-container">
            <form class="search-form" onsubmit="performSearch(event)">
                <input type="text" class="search-input" id="searchQuery" placeholder="Enter your search query..." required>
                <button type="submit" class="search-btn">Search</button>
            </form>
            
            <div class="search-options">
                <label>
                    Results: 
                    <input type="number" id="resultLimit" value="10" min="1" max="100">
                </label>
                <div class="cache-controls">
                    <button class="cache-btn" onclick="clearCache()">Clear Cache</button>
                    <button class="cache-btn" onclick="prewarmCache()">Prewarm Cache</button>
                    <button class="cache-btn" onclick="optimizeCache()">Optimize Cache</button>
                    <button class="cache-btn" onclick="refreshStats()">Refresh Stats</button>
                </div>
            </div>
        </div>

        <div class="crawl-section">
            <h2>🕷️ Web Crawler</h2>
            <form class="crawl-form" onsubmit="startCrawl(event)">
                <input type="url" class="crawl-input" id="crawlUrl" placeholder="https://example.com" required>
                <button type="submit" class="crawl-btn">Start Crawl</button>
            </form>
            <p style="font-size: 12px; color: #666; margin-top: 10px;">
                This will crawl the website and add pages to the search index.
            </p>
        </div>

        <div id="message" class="message"></div>

        <div id="resultsContainer" class="results-container">
            <h2>Search Results</h2>
            <div id="searchResults"></div>
        </div>

        <div class="stats-container">
            <h2>📊 System Statistics</h2>
            <div id="statsGrid" class="stats-grid">
                <div class="loading">
                    <div class="spinner"></div>
                    Loading statistics...
                </div>
            </div>
        </div>
    </div>

    <script>
        let searchTimeout;

        async function performSearch(event) {
            event.preventDefault();
            
            const query = document.getElementById('searchQuery').value.trim();
            const limit = document.getElementById('resultLimit').value;
            
            if (!query) return;

            const resultsContainer = document.getElementById('resultsContainer');
            const searchResults = document.getElementById('searchResults');
            
            // Show loading state
            searchResults.innerHTML = '<div class="loading"><div class="spinner"></div>Searching...</div>';
            resultsContainer.style.display = 'block';

            try {
                const response = await fetch('/search?' + new URLSearchParams({
                    'search-query': query,
                    'k': limit
                }));

                if (!response.ok) {
                    throw new Error('Search failed');
                }

                const results = await response.json();
                displayResults(results, query);
                
                // Refresh stats after search
                setTimeout(refreshStats, 1000);
                
            } catch (error) {
                searchResults.innerHTML = '<div class="error">Search failed: ' + error.message + '</div>';
                showMessage('Search failed: ' + error.message, 'error');
            }
        }

        function displayResults(results, query) {
            const searchResults = document.getElementById('searchResults');
            
            if (!results || results.length === 0) {
                searchResults.innerHTML = '<div class="no-results">No results found for "' + query + '"</div>';
                return;
            }

            let html = '<div style="margin-bottom: 20px; color: #666;">Found ' + results.length + ' results for "' + query + '"</div>';
            
            results.forEach((result, index) => {
                html += '<div class="result-item">';
                html += '<div class="result-title">' + (result.title || result.url || 'Untitled') + '</div>';
                html += '<div class="result-url">' + (result.url || result.doc_id) + '</div>';
                html += '<div class="result-score">Relevance Score: ' + result.score.toFixed(4) + '</div>';
                html += '</div>';
            });

            searchResults.innerHTML = html;
        }

        async function startCrawl(event) {
            event.preventDefault();
            
            const url = document.getElementById('crawlUrl').value.trim();
            if (!url) return;

            try {
                const response = await fetch('/crawl?' + new URLSearchParams({
                    'url': url
                }), {
                    method: 'POST'
                });

                if (!response.ok) {
                    throw new Error('Crawl failed');
                }

                const message = await response.text();
                showMessage('Crawl started: ' + message, 'success');
                
                // Clear the input
                document.getElementById('crawlUrl').value = '';
                
                // Refresh stats after a delay to show new documents
                setTimeout(refreshStats, 3000);
                
            } catch (error) {
                showMessage('Crawl failed: ' + error.message, 'error');
            }
        }

        async function refreshStats() {
            const statsGrid = document.getElementById('statsGrid');
            
            try {
                const response = await fetch('/stats');
                if (!response.ok) throw new Error('Failed to load stats');
                
                const stats = await response.json();
                displayStats(stats);
                
            } catch (error) {
                statsGrid.innerHTML = '<div class="error">Failed to load statistics</div>';
            }
        }

        function displayStats(stats) {
            const statsGrid = document.getElementById('statsGrid');
            
            let html = '';
            
            // Index stats
            if (stats.index) {
                html += '<div class="stat-card"><h3>Documents</h3><div class="stat-value">' + (stats.index.total_documents || 0) + '</div></div>';
                html += '<div class="stat-card"><h3>Unique Terms</h3><div class="stat-value">' + (stats.index.unique_terms || 0) + '</div></div>';
                html += '<div class="stat-card"><h3>Avg Doc Length</h3><div class="stat-value">' + (stats.index.average_doc_length || 0).toFixed(1) + '</div></div>';
            }
            
            // Cache stats
            if (stats.cache) {
                const cacheStats = stats.cache.stats || {};
                const hitRates = stats.cache.hit_rates || {};
                
                html += '<div class="stat-card"><h3>L1 Cache Hit Rate</h3><div class="stat-value">' + (hitRates.l1 * 100).toFixed(1) + '%</div></div>';
                html += '<div class="stat-card"><h3>L2 Cache Hit Rate</h3><div class="stat-value">' + (hitRates.l2 * 100).toFixed(1) + '%</div></div>';
                html += '<div class="stat-card"><h3>L3 Cache Hit Rate</h3><div class="stat-value">' + (hitRates.l3 * 100).toFixed(1) + '%</div></div>';
                html += '<div class="stat-card"><h3>Cache Evictions</h3><div class="stat-value">' + (cacheStats.evictions || 0) + '</div></div>';
                html += '<div class="stat-card"><h3>L1 Items</h3><div class="stat-value">' + (stats.cache.l1_items || 0) + '</div></div>';
                html += '<div class="stat-card"><h3>L2 Size (MB)</h3><div class="stat-value">' + ((stats.cache.l2_size_bytes || 0) / 1024 / 1024).toFixed(1) + '</div></div>';
            }
            
            statsGrid.innerHTML = html;
        }

        async function clearCache() {
            try {
                const response = await fetch('/cache/clear', { method: 'POST' });
                if (!response.ok) throw new Error('Failed to clear cache');
                
                showMessage('Cache cleared successfully', 'success');
                refreshStats();
                
            } catch (error) {
                showMessage('Failed to clear cache: ' + error.message, 'error');
            }
        }

        async function prewarmCache() {
            try {
                const response = await fetch('/cache/prewarm', { method: 'POST' });
                if (!response.ok) throw new Error('Failed to prewarm cache');
                
                showMessage('Cache prewarmed successfully', 'success');
                refreshStats();
                
            } catch (error) {
                showMessage('Failed to prewarm cache: ' + error.message, 'error');
            }
        }

        async function optimizeCache() {
            try {
                const response = await fetch('/cache/optimize', { method: 'POST' });
                if (!response.ok) throw new Error('Failed to optimize cache');
                
                showMessage('Cache optimized successfully', 'success');
                refreshStats();
                
            } catch (error) {
                showMessage('Failed to optimize cache: ' + error.message, 'error');
            }
        }

        function showMessage(text, type) {
            const messageEl = document.getElementById('message');
            messageEl.textContent = text;
            messageEl.className = 'message ' + type;
            messageEl.style.display = 'block';
            
            setTimeout(() => {
                messageEl.style.display = 'none';
            }, 5000);
        }

        // Load stats on page load
        document.addEventListener('DOMContentLoaded', function() {
            refreshStats();
            
            // Auto-refresh stats every 30 seconds
            setInterval(refreshStats, 30000);
        });

        // Auto-search as user types (with debouncing)
        document.getElementById('searchQuery').addEventListener('input', function() {
            clearTimeout(searchTimeout);
            const query = this.value.trim();
            
            if (query.length > 2) {
                searchTimeout = setTimeout(() => {
                    document.getElementById('searchQuery').value = query;
                    performSearch({ preventDefault: () => {} });
                }, 500);
            }
        });
    </script>
</body>
</html>
`

var searchTemplate = template.Must(template.New("search").Parse(searchPageHTML))

func GetHome(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := searchTemplate.Execute(w, nil); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		return
	}
}

func GetSearch(w http.ResponseWriter, r *http.Request) {
	searchQuery := r.URL.Query().Get("search-query")
	if searchQuery == "" {
		http.Error(w, "invalid query missing 'search-query' parameter", http.StatusBadRequest)
		return // Fixed: Added missing return statement
	}

	searchLimit, err := strconv.Atoi(r.URL.Query().Get("k"))
	if err != nil || searchLimit < 0 {
		searchLimit = 10
	}

	searchResults := service.InvertedIndex.Search(searchQuery, searchLimit)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(searchResults); err != nil {
		http.Error(w, "failed to json encode search results", http.StatusInternalServerError)
		return
	}
}

func GetCrawl(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Crawl endpoint - use POST /crawl?url=<url> to start crawling")
}

func PostCrawl(w http.ResponseWriter, r *http.Request) {
	crawlURL := r.URL.Query().Get("url")

	if crawlURL == "" {
		http.Error(w, "invalid query: missing 'url' parameter", http.StatusBadRequest)
		return
	}

	if u, err := url.ParseRequestURI(crawlURL); err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		http.Error(w, "bad URL provided", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("crawl has been queued for " + crawlURL))

	go func(crawlURL string) {
		service.CrawlRecursive(crawlURL)
	}(crawlURL)
}

// New cache management endpoints
func GetStats(w http.ResponseWriter, r *http.Request) {
	indexStats := service.InvertedIndex.GetIndexStats()
	cacheStats := service.InvertedIndex.GetCacheStats()

	stats := map[string]interface{}{
		"index": indexStats,
		"cache": cacheStats,
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(stats); err != nil {
		http.Error(w, "failed to encode stats", http.StatusInternalServerError)
		return
	}
}

func PostClearCache(w http.ResponseWriter, r *http.Request) {
	if err := service.InvertedIndex.ClearCache(); err != nil {
		http.Error(w, "failed to clear cache: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "cache cleared successfully",
	})
}

func PostPrewarmCache(w http.ResponseWriter, r *http.Request) {
	go service.InvertedIndex.PrewarmCache()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "cache prewarming started",
	})
}

func PostOptimizeCache(w http.ResponseWriter, r *http.Request) {
	go service.InvertedIndex.OptimizeCache()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "cache optimization started",
	})
}
