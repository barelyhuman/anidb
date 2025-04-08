-- Anime table for SQLite
CREATE TABLE anime (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    link TEXT NOT NULL UNIQUE,
    title TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

create index if not EXISTS idx_anime_title on anime(title);

DROP TRIGGER IF EXISTS anime_updated_at;
CREATE TRIGGER anime_updated_at
AFTER UPDATE ON anime
FOR EACH ROW
WHEN NEW.updated_at = OLD.updated_at
BEGIN
    UPDATE anime
        SET updated_at = datetime('now')
        WHERE id = NEW.id;
END;


-- Master_meta table for SQLite
CREATE TABLE master_meta (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    picture TEXT,
    thumbnail TEXT,
    tags TEXT,
    type TEXT,
    status TEXT,
    synonyms TEXT,
    score TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

create index if not EXISTS idx_master_meta_name on master_meta(name);

DROP TRIGGER IF EXISTS master_meta_updated_at;
CREATE TRIGGER master_meta_updated_at
AFTER UPDATE ON master_meta
FOR EACH ROW
WHEN NEW.updated_at = OLD.updated_at
BEGIN
    UPDATE master_meta
        SET updated_at = datetime('now')
        WHERE id = NEW.id;
END;

-- Sync Log
create table sync_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    last_run TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    status TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

DROP TRIGGER IF EXISTS sync_log_updated_at;
CREATE TRIGGER sync_log_updated_at
AFTER UPDATE ON sync_log
FOR EACH ROW
WHEN NEW.updated_at = OLD.updated_at
BEGIN
    UPDATE sync_log
        SET updated_at = datetime('now')
        WHERE id = NEW.id;
END;

-- view to merge the above data
CREATE VIEW if not exists anime_meta AS
SELECT
	anime.title,
	anime."link",
	master_meta.picture,
	master_meta.thumbnail,
	master_meta.status,
	master_meta.tags,
	master_meta."type",
    master_meta."synonyms",
    master_meta."score"
FROM
	master_meta,
	json_each(master_meta.synonyms, '$') as j
join anime on j.value = anime.title or master_meta."name" = anime.title 
group by anime.link
order by anime.title;