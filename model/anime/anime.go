package anime

import "database/sql"

type Anime struct {
	Title string
	Link  string
}

type AnimeWithIdentifier struct {
	ID int64
	Anime
}

func New() *Anime {
	return &Anime{}
}

func Insert(db *sql.DB, an *Anime) (AnimeWithIdentifier, error) {
	record := AnimeWithIdentifier{}
	result, err := db.Exec("insert into anime (title,link) values (?,?)", an.Title, an.Link)
	if err != nil {
		return record, nil
	}

	id, err := result.LastInsertId()
	if err != nil {
		if err != nil {
			return record, nil
		}
	}

	record.ID = id
	record.Title = an.Title
	record.Link = an.Link

	return record, nil
}

func FindByLink(db *sql.DB, link string) (*AnimeWithIdentifier, error) {
	record := &AnimeWithIdentifier{}
	result, err := db.Query("select title,link where link=?", link)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}

	for result.Next() {
		result.Scan(&record.Title, &record.Link)
	}
	return record, nil
}
