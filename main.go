package main

import (
	"database/sql"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq" // PostgreSQL database driver tool
)

// 1. Blueprint schema to automatically capture incoming JSON from the user.
type ShortenRequest struct {
	LongURL string `json:"long_url" binding:"required"`
}

// 2. Global Database Connection Pointer Box.
var db *sql.DB

// 3. Helper utility function to generate a random 6-character short key token string.
func generateShortKey() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	shortKey := make([]byte, 6)
	for i := range shortKey {
		shortKey[i] = charset[r.Intn(len(charset))]
	}
	return string(shortKey)
}

func main() {
	var err error

	// 4. Connect to our local PostgreSQL database server engine.
	connStr := "host=127.0.0.1 port=5433 user=postgres password=20001890 dbname=shortener sslmode=disable"
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Could not open a database connection socket:", err)
	}
	defer db.Close() // Safety rule: Closes the database pipe when the app shuts down

	// 🚨 THE PING TEST: Force Go to log into the database immediately!
	err = db.Ping()
	if err != nil {
		log.Fatal("🔥 DATABASE CONNECTION FAILED. The error is: ", err)
	}
	log.Println("✅ DATABASE CONNECTED SUCCESSFULLY!")

	// Initialize Gin's default web server engine router framework
	r := gin.Default()

	// ROUTE A: The Endpoint to Shorten a long website link.
	r.POST("/shorten", func(c *gin.Context) {
		var req ShortenRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid text request format"})
			return
		}

		key := generateShortKey()

		// DATABASE UPDATE: Write raw data straight into our persistent SQL table
		query := "INSERT INTO urls (short_key, long_url) VALUES ($1, $2)"
		_, err = db.Exec(query, key, req.LongURL)
		if err != nil {
			log.Println("🔥 FAILED TO INSERT DATA:", err) // Prints exact DB failure to terminal
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save link to database"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"short_key": key,
			"short_url": "http://localhost:8080/r/" + key,
		})
	})

	// ROUTE B: The Redirect Engine (Now with Analytics tracking!)
	r.GET("/r/:key", func(c *gin.Context) {
		key := c.Param("key")
		var longURL string

		query := "SELECT long_url FROM urls WHERE short_key = $1"
		err := db.QueryRow(query, key).Scan(&longURL)

		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "This shortened link address does not exist"})
			return
		} else if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database query error"})
			return
		}

		updateQuery := "UPDATE urls SET click_count = click_count + 1 WHERE short_key = $1"
		_, _ = db.Exec(updateQuery, key)

		c.Redirect(http.StatusFound, longURL)
	})

	// ROUTE C: The Analytics Endpoint
	r.GET("/analytics/:key", func(c *gin.Context) {
		key := c.Param("key")
		var clickCount int
		var originalURL string

		query := "SELECT long_url, click_count FROM urls WHERE short_key = $1"
		err := db.QueryRow(query, key).Scan(&originalURL, &clickCount)

		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Link not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"short_key":    key,
			"original_url": originalURL,
			"total_clicks": clickCount,
		})
	})

	// Start up our live web server engine on port 8080
	r.Run(":8080")
}
