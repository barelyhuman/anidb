package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/barelyhuman/anidb/database"
	"github.com/barelyhuman/anidb/middleware"
	"github.com/barelyhuman/anidb/migrate"
	"github.com/barelyhuman/anidb/scraper"
	"github.com/barelyhuman/anidb/view"
	"github.com/barelyhuman/go/env"
	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

var db *sql.DB

const baseURL = "https://animepahe.ru"

func main() {
	godotenv.Load("./.env")

	router := mux.NewRouter()
	router.HandleFunc("/", homeHandler).Methods("GET")

	db = database.GetDB()
	defer db.Close()

	migrate.MigrateUp(db, "./migrate")

	appEnv := env.Get("APP_ENV", "development")
	isProd := appEnv == "production"

	csrfMiddleware := csrf.Protect(
		[]byte(env.Get("CSRF_TOKEN", "")),
		csrf.Secure(isProd),
	)

	handler := csrfMiddleware(
		middleware.RateLimitMiddleware(
			router,
		),
	)

	ticker := time.NewTicker(time.Hour * 6)

	// In case the database isn't setup and is starting fresh
	go func() {
		res, err := db.Query(`select last_run from sync_log order by last_run desc limit 1`)
		if err != nil {
			log.Println("Failed to get sync log")
			return
		}
		defer res.Close()
		hasRecords := false
		var lastScanTime time.Time
		for res.Next() {
			hasRecords = true
			res.Scan(&lastScanTime)
		}
		if !hasRecords {
			Sync()
		} else if time.Since(lastScanTime) > time.Hour*24 {
			Sync()
		}
	}()

	go func() {
		for range ticker.C {
			Sync()
		}
	}()

	server := &http.Server{
		Addr:         "0.0.0.0:8081",
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	fmt.Println("Server is running on http://127.0.0.1:8081")
	log.Fatal(server.ListenAndServe())
}

type AniMeta struct {
	Title   string
	Link    string
	Picture string
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")

	fromSearch := false
	aniCollection := []AniMeta{}
	if len(q) > 0 {
		fromSearch = true
		stmt, _ := db.Prepare("select title,link,picture from anime_meta, json_each(anime_meta.synonyms) where json_each.value like :search_term or anime_meta.title like :search_term group by anime_meta.title,anime_meta.link")
		re, err := stmt.Query(sql.Named("search_term", normalizeSearchTerm(q)))
		if err != nil {
			log.Printf("failed to execute query with error :%v", err)
			return
		}
		defer re.Close()
		for re.Next() {
			x := &AniMeta{}
			re.Scan(&x.Title, &x.Link, &x.Picture)
			x.Link = baseURL + x.Link
			aniCollection = append(aniCollection, *x)
		}
	} else {
		re, err := db.Query("select title,link,picture from anime_meta order by score desc limit 10")
		if err != nil {
			log.Printf("failed to execute query with error :%v", err)
			return
		}
		defer re.Close()
		for re.Next() {
			x := &AniMeta{}
			re.Scan(&x.Title, &x.Link, &x.Picture)
			x.Link = baseURL + x.Link
			aniCollection = append(aniCollection, *x)
		}
	}

	w.Header().Set("Content-Type", "text/html")
	if err := view.Render(w, "HomePage", struct {
		Collection []AniMeta
		SearchTerm string
		CSRFToken  string
		FromSearch bool
	}{
		Collection: aniCollection,
		SearchTerm: q,
		CSRFToken:  csrf.Token(r),
		FromSearch: fromSearch,
	}); err != nil {
		log.Fatal(err)
	}
}

func normalizeSearchTerm(s string) string {
	var query strings.Builder
	for _, x := range strings.Split(s, " ") {
		query.WriteString(x + "%")
	}
	return "%" + query.String() + "%"
}

func Sync() {
	scraper.Sync()
	_, err := db.Exec("insert into sync_log (status) values ('done')")
	if err != nil {
		log.Printf("Error updating sync_log: %v", err)
	}
}
