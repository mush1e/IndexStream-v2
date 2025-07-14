package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

type DB struct {
	*sql.DB
}

type CrawlJob struct {
	ID          int        `json:"id"`
	URL         string     `json:"url"`
	Status      string     `json:"status"` // pending, running, completed, failed
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	PagesFound  int        `json:"pages_found"`
	Error       string     `json:"error,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

type SearchQuery struct {
	ID        int       `json:"id"`
	Query     string    `json:"query"`
	Results   int       `json:"results"`
	Duration  int64     `json:"duration_ms"`
	CreatedAt time.Time `json:"created_at"`
}

type IndexedDocument struct {
	ID        int       `json:"id"`
	DocID     string    `json:"doc_id"`
	URL       string    `json:"url"`
	Title     string    `json:"title"`
	WordCount int       `json:"word_count"`
	IndexedAt time.Time `json:"indexed_at"`
}

func NewDB(databaseURL string) (*DB, error) {
	var db *sql.DB
	var err error

	if databaseURL == "" || databaseURL == "sqlite" {
		// Default to SQLite for local development
		db, err = sql.Open("sqlite3", "./goosesearch.db")
	} else {
		// PostgreSQL for production
		db, err = sql.Open("postgres", databaseURL)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	database := &DB{db}
	if err = database.createTables(); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return database, nil
}

func (db *DB) createTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS crawl_jobs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			url TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			started_at DATETIME,
			completed_at DATETIME,
			pages_found INTEGER DEFAULT 0,
			error TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS search_queries (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			query TEXT NOT NULL,
			results INTEGER DEFAULT 0,
			duration_ms INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS indexed_documents (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			doc_id TEXT UNIQUE NOT NULL,
			url TEXT NOT NULL,
			title TEXT,
			word_count INTEGER DEFAULT 0,
			indexed_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_crawl_jobs_status ON crawl_jobs(status)`,
		`CREATE INDEX IF NOT EXISTS idx_search_queries_created_at ON search_queries(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_indexed_documents_doc_id ON indexed_documents(doc_id)`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query %s: %w", query, err)
		}
	}

	return nil
}

// CrawlJob methods
func (db *DB) CreateCrawlJob(url string) (*CrawlJob, error) {
	job := &CrawlJob{
		URL:       url,
		Status:    "pending",
		CreatedAt: time.Now(),
	}

	query := `INSERT INTO crawl_jobs (url, status, created_at) VALUES (?, ?, ?) RETURNING id`
	err := db.QueryRow(query, job.URL, job.Status, job.CreatedAt).Scan(&job.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to create crawl job: %w", err)
	}

	return job, nil
}

func (db *DB) UpdateCrawlJobStatus(id int, status string, pagesFound int, errorMsg string) error {
	var query string
	var args []interface{}

	switch status {
	case "running":
		query = `UPDATE crawl_jobs SET status = ?, started_at = ? WHERE id = ?`
		args = []interface{}{status, time.Now(), id}
	case "completed":
		query = `UPDATE crawl_jobs SET status = ?, completed_at = ?, pages_found = ? WHERE id = ?`
		args = []interface{}{status, time.Now(), pagesFound, id}
	case "failed":
		query = `UPDATE crawl_jobs SET status = ?, completed_at = ?, error = ? WHERE id = ?`
		args = []interface{}{status, time.Now(), errorMsg, id}
	default:
		return fmt.Errorf("invalid status: %s", status)
	}

	_, err := db.Exec(query, args...)
	return err
}

func (db *DB) GetCrawlJobs(limit int) ([]CrawlJob, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `SELECT id, url, status, started_at, completed_at, pages_found, error, created_at 
			  FROM crawl_jobs ORDER BY created_at DESC LIMIT ?`

	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []CrawlJob
	for rows.Next() {
		var job CrawlJob
		err := rows.Scan(&job.ID, &job.URL, &job.Status, &job.StartedAt,
			&job.CompletedAt, &job.PagesFound, &job.Error, &job.CreatedAt)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}

	return jobs, nil
}

// SearchQuery methods
func (db *DB) LogSearchQuery(query string, results int, duration time.Duration) error {
	sqlQuery := `INSERT INTO search_queries (query, results, duration_ms, created_at) VALUES (?, ?, ?, ?)`
	_, err := db.Exec(sqlQuery, query, results, duration.Milliseconds(), time.Now())
	return err
}

func (db *DB) GetRecentSearches(limit int) ([]SearchQuery, error) {
	if limit <= 0 {
		limit = 20
	}

	query := `SELECT id, query, results, duration_ms, created_at 
			  FROM search_queries ORDER BY created_at DESC LIMIT ?`

	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var searches []SearchQuery
	for rows.Next() {
		var search SearchQuery
		err := rows.Scan(&search.ID, &search.Query, &search.Results, &search.Duration, &search.CreatedAt)
		if err != nil {
			return nil, err
		}
		searches = append(searches, search)
	}

	return searches, nil
}

func (db *DB) GetPopularSearches(limit int) ([]SearchQuery, error) {
	if limit <= 0 {
		limit = 10
	}

	query := `SELECT query, COUNT(*) as frequency, AVG(results) as avg_results, AVG(duration_ms) as avg_duration
			  FROM search_queries 
			  WHERE created_at > datetime('now', '-7 days')
			  GROUP BY query 
			  ORDER BY frequency DESC, avg_results DESC 
			  LIMIT ?`

	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var searches []SearchQuery
	for rows.Next() {
		var search SearchQuery
		var frequency int
		err := rows.Scan(&search.Query, &frequency, &search.Results, &search.Duration)
		if err != nil {
			return nil, err
		}
		searches = append(searches, search)
	}

	return searches, nil
}

// IndexedDocument methods
func (db *DB) AddIndexedDocument(docID, url, title string, wordCount int) error {
	query := `INSERT OR REPLACE INTO indexed_documents (doc_id, url, title, word_count, indexed_at) 
			  VALUES (?, ?, ?, ?, ?)`
	_, err := db.Exec(query, docID, url, title, wordCount, time.Now())
	return err
}

func (db *DB) GetIndexedDocuments(limit int) ([]IndexedDocument, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `SELECT id, doc_id, url, title, word_count, indexed_at 
			  FROM indexed_documents ORDER BY indexed_at DESC LIMIT ?`

	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []IndexedDocument
	for rows.Next() {
		var doc IndexedDocument
		err := rows.Scan(&doc.ID, &doc.DocID, &doc.URL, &doc.Title, &doc.WordCount, &doc.IndexedAt)
		if err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}

	return docs, nil
}

func (db *DB) GetDashboardStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total documents
	var totalDocs int
	err := db.QueryRow("SELECT COUNT(*) FROM indexed_documents").Scan(&totalDocs)
	if err != nil {
		log.Printf("Error getting total docs: %v", err)
	}
	stats["total_documents"] = totalDocs

	// Total searches today
	var searchesToday int
	err = db.QueryRow("SELECT COUNT(*) FROM search_queries WHERE DATE(created_at) = DATE('now')").Scan(&searchesToday)
	if err != nil {
		log.Printf("Error getting searches today: %v", err)
	}
	stats["searches_today"] = searchesToday

	// Active crawl jobs
	var activeCrawls int
	err = db.QueryRow("SELECT COUNT(*) FROM crawl_jobs WHERE status IN ('pending', 'running')").Scan(&activeCrawls)
	if err != nil {
		log.Printf("Error getting active crawls: %v", err)
	}
	stats["active_crawls"] = activeCrawls

	// Average search duration
	var avgDuration float64
	err = db.QueryRow("SELECT AVG(duration_ms) FROM search_queries WHERE created_at > datetime('now', '-24 hours')").Scan(&avgDuration)
	if err != nil {
		log.Printf("Error getting avg duration: %v", err)
	}
	stats["avg_search_duration"] = avgDuration

	return stats, nil
}
