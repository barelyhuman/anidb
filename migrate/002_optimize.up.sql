DROP VIEW if exists anime_meta; 

CREATE VIEW if not exists anime_meta AS
SELECT
	anime.id as anime_id,
    master_meta."id" as meta_id
FROM
	anime,
	master_meta,
	json_each(master_meta.synonyms)
WHERE
	JSON_EACH.value = anime.title
	OR master_meta."name" = anime.title
group by anime.link, anime.title
order by score desc; 