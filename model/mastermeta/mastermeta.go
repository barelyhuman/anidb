package mastermeta

import (
	"database/sql"
	"reflect"
	"time"
)

type MasterMeta struct {
	Name      string
	Picture   string
	Thumbnail string
	Tags      []string
	Type      string
	Status    string
	Synonyms  []string
	Score     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type MasterMetaWithIdentifier struct {
	ID int64
	MasterMeta
}

func New() *MasterMeta {
	return &MasterMeta{}
}

func Insert(db *sql.DB, data MasterMeta) (MasterMetaWithIdentifier, error) {
	record := MasterMetaWithIdentifier{}
	result, err := db.Exec("insert into anime (name,picture,thumbnail,tags,type,status,synonyms,score) values (?,?,?,json(?),?,?,json(?),?)",
		data.Name,
		data.Picture,
		data.Thumbnail,
		data.Tags,
		data.Type,
		data.Status,
		data.Synonyms,
		data.Score,
	)
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

	src := reflect.ValueOf(data)
	dst := reflect.ValueOf(&record.MasterMeta).Elem()
	for i := 0; i < src.NumField(); i++ {
		dstField := dst.Field(i)
		if dstField.CanSet() {
			dstField.Set(src.Field(i))
		}
	}

	return record, nil
}

func FindByName(db *sql.DB, name string) (*MasterMetaWithIdentifier, error) {
	record := &MasterMetaWithIdentifier{}
	result, err := db.Query("select id,name,picture,thumbnail,tags,type,status,synonyms,score,created_at,updated_at from master_meta where name = ?", name)
	defer result.Close()

	if err != nil {
		return record, err
	}

	if result == nil {
		return nil, nil
	}

	if !result.NextResultSet() {
		return nil, nil
	}

	if result.Next() {
		if err := result.Scan(
			&record.ID,
			&record.Name,
			&record.Picture,
			&record.Thumbnail,
			&record.Tags,
			&record.Type,
			&record.Status,
			&record.Synonyms,
			&record.Score,
			&record.CreatedAt,
			&record.UpdatedAt,
		); err != nil {
			return nil, err
		}
	}

	return record, nil
}
