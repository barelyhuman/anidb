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
	"github.com/blockloop/scan/v2"
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

	// initialize filterOptions getter
	getFilterOptions = buildFilterQueries()

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
			// re-initialize filterOptions getter
			getFilterOptions = buildFilterQueries()
		} else if time.Since(lastScanTime) > time.Hour*24 {
			// re-initialize filterOptions getter
			getFilterOptions = buildFilterQueries()
		}
	}()

	go func() {
		for range ticker.C {
			Sync()
			// re-initialize filterOptions getter
			getFilterOptions = buildFilterQueries()
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

type FilterOptions struct {
	Tags       []string
	Types      []string
	StatusList []string
}

var getFilterOptions func() FilterOptions

func buildFilterQueries() func() FilterOptions {
	tagsStmt, _ := db.Prepare("select DISTINCT(json_each.value) from master_meta, json_each(master_meta.tags) where master_meta.id in (select meta_id from anime_meta) order by json_each.value asc")
	typeStmt, _ := db.Prepare("select distinct(type) from master_meta where master_meta.id in (select meta_id from anime_meta)")
	statusStmt, _ := db.Prepare("select distinct(status) from master_meta where master_meta.id in (select meta_id from anime_meta)")
	return func(tagsStmt, typeStmt, statusStmt *sql.Stmt) func() FilterOptions {
		return func() FilterOptions {
			tags := []string{}
			types := []string{}
			statusList := []string{}

			tagsRows, err := tagsStmt.Query()
			if err != nil {
				log.Println("failed to get tags", err)
			}
			typeRows, err := typeStmt.Query()
			if err != nil {
				log.Println("failed to get types", err)
			}
			statusRows, err := statusStmt.Query()
			if err != nil {
				log.Println("failed to get status", err)
			}

			if err := scan.Rows(&tags, tagsRows); err != nil {
				log.Println("failed to get tags", err)
			}

			if err := scan.Rows(&types, typeRows); err != nil {
				log.Println("failed to get types", err)
			}

			if err := scan.Rows(&statusList, statusRows); err != nil {
				log.Println("failed to get status", err)
			}

			return FilterOptions{
				Tags:       tags,
				Types:      types,
				StatusList: statusList,
			}
		}
	}(
		tagsStmt,
		typeStmt,
		statusStmt,
	)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	filteredTag := r.URL.Query().Get("tag")
	filteredType := r.URL.Query().Get("type")
	filteredStatus := r.URL.Query().Get("status")

	fromSearch := false
	aniCollection := []AniMeta{}

	queryStr := QueryStringBuilder{}
	queryStr.Query(`
		with joined_anime as MATERIALIZED (select * from anime 
left join anime_meta on anime_meta.anime_id = anime.id
left join master_meta on master_meta.id = anime_meta.meta_id
)
	`)
	queryStr.Query("select title,link,picture from joined_anime")

	if len(q) > 0 {
		fromSearch = true
		queryStr.Query(",json_each(synonyms) where json_each.value like ").
			Param(normalizeSearchTerm(q)).
			Query(" or title like ").
			Param(normalizeSearchTerm(q))
	}

	if len(filteredTag) > 0 ||
		len(filteredStatus) > 0 ||
		len(filteredType) > 0 {
		fromSearch = true
		addAnd := false

		if len(q) == 0 {
			queryStr.Query(" where")
		} else {
			addAnd = true
		}

		if len(filteredStatus) > 0 {
			if addAnd {
				queryStr.Query(" and ")
			}
			queryStr.Query(" status = ").
				Param(filteredStatus)
			addAnd = true
		}

		if len(filteredTag) > 0 {
			if addAnd {
				queryStr.Query(" and ")
			}
			queryStr.Query(" meta_id in (select master_meta.id from master_meta,json_each(master_meta.tags) where json_each.value = ").
				Param(filteredTag).
				Query(")")
			addAnd = true
		}

		if len(filteredType) > 0 {
			if addAnd {
				queryStr.Query(" and ")
			}
			queryStr.Query(" joined_anime.type = ").
				Param(filteredType)
			addAnd = true
		}
	}

	queryStr.Query(" group by title,link order by score desc")

	if !fromSearch {
		queryStr.Query(" limit 20")
	}

	query, vars := queryStr.Get()

	var re *sql.Rows
	var err error
	re, err = db.Query(query, vars...)
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

	filterOptionList := getFilterOptions()

	w.Header().Set("Content-Type", "text/html")
	if err := view.Render(w, "HomePage", struct {
		Collection      []AniMeta
		SearchTerm      string
		CSRFToken       string
		FromSearch      bool
		SelectedFilters struct {
			Tag    string
			Type   string
			Status string
		}
		FilterOptions
	}{
		Collection:    aniCollection,
		SearchTerm:    q,
		CSRFToken:     csrf.Token(r),
		FromSearch:    fromSearch,
		FilterOptions: filterOptionList,
		SelectedFilters: struct {
			Tag    string
			Type   string
			Status string
		}{
			Tag:    filteredTag,
			Type:   filteredType,
			Status: filteredStatus,
		},
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
