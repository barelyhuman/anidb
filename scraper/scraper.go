package scraper

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"strings"

	"github.com/barelyhuman/anidb/database"
	"github.com/barelyhuman/anidb/metadata"
	"github.com/barelyhuman/anidb/model/anime"
	"github.com/barelyhuman/anidb/model/mastermeta"
	_ "github.com/mattn/go-sqlite3"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
)

// scrapeLinks navigates to the given URL, extracts the navigation links,
// and then for each navigation tab clicks and retrieves links from the tab content.
func scrapeLinks(url string) (map[string]string, error) {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	var tabHTML string
	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitVisible(`ul.nav.nav-pills`, chromedp.ByQuery),
		chromedp.WaitVisible(`.tab-content`, chromedp.ByQuery),
		chromedp.OuterHTML(".tab-content", &tabHTML, chromedp.ByQuery),
	)
	if err != nil {
		return nil, err
	}

	animeDB := make(map[string]string)

	// Parse the resulting HTML and look for links inside tab-content
	tabDoc, err := goquery.NewDocumentFromReader(strings.NewReader(tabHTML))
	if err != nil {
		log.Printf("Error parsing tab content with error %v", err)
		return animeDB, err
	}

	tabDoc.Find(".tab-content a[href]").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists {
			return
		}
		// title attribute may not be set
		title, _ := s.Attr("title")
		if len(title) == 0 {
			return
		}
		animeDB[href] = title
	})

	return animeDB, nil
}

func storeAnimeDB(db *sql.DB, animeDB map[string]string) error {
	// Insert each anime record into the database.
	for link, title := range animeDB {
		exists, _ := anime.FindByLink(db, link)
		if exists == nil {
			an := anime.New()
			an.Link = link
			an.Title = title
			_, err := anime.Insert(db, an)
			if err != nil {
				log.Printf("Failed to insert %v, with error: %v", title, err)
				continue
			}
		}
	}

	return nil
}

func Sync() {
	url := "https://animepahe.ru/anime"
	var err error
	db := database.GetDB()
	animeDB, err := scrapeLinks(url)
	if err != nil {
		log.Fatalf("Error scraping links: %v", err)
	}
	storeAnimeDB(db, animeDB)

	log.Println("Synced Anipahe Listings")

	fullMeta, err := metadata.GetOnlineMeta()
	if err != nil {
		log.Fatalf("Error getting meta: %v", err)
	}

	for _, meta := range fullMeta {
		insertStmt, err := db.Prepare("insert into master_meta (name,picture,thumbnail,type,tags,synonyms,status,score) values (?,?,?,?,json(?),json(?),?,?)")
		if err != nil {
			log.Fatalf("Invalid insert query,:%v", err)
		}
		updateStmt, err := db.Prepare("update master_meta set name=?, picture=?, thumbnail=?, type=?, tags=json(?), synonyms=json(?), status=?, score=? where id = ?")
		if err != nil {
			log.Fatal("Invalid update query")
		}

		ani, err := mastermeta.FindByName(db, meta.Name)
		if err != nil {
			log.Fatalf("Failed to get master meta with error: %v", err)
		}

		tags, _ := json.Marshal(meta.Tags)
		synonyms, _ := json.Marshal(meta.Synonyms)

		if ani == nil {
			_, err := insertStmt.Exec(
				meta.Name,
				meta.Picture,
				meta.Thumbnail,
				meta.Type,
				tags,
				string(synonyms),
				meta.Status,
				meta.Score.Median,
			)

			if err != nil {
				log.Printf("Failed to insert meta for %v , synonyms %v with error: %v", meta.Name, string(synonyms), err)
				continue
			}

			continue
		}
		_, err = updateStmt.Exec(
			meta.Name,
			meta.Picture,
			meta.Thumbnail,
			meta.Type,
			tags,
			string(synonyms),
			meta.Status,
			meta.Score.Median,
			ani.ID,
		)
		if err != nil {
			log.Printf("Failed to update meta for %v, synonyms %v with error: %v", meta.Name, err)
			continue
		}
	}

	log.Println("Inserted Master meta")
}

func stringsToJSONRawMessage(d []string) json.RawMessage {
	tagsBytes, err := json.Marshal(d)
	if err != nil {
		tagsBytes, _ = json.Marshal([]string{})
	}
	return json.RawMessage(tagsBytes)

}
