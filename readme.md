# anidb

A slightly improved search for animepahe.ru 

> [!NOTE]
>
> Built for educational purposes


Carries the code to occasionaly scrape and curate data from animepahe to avoid using the search from the site. 

## Highlights 
- Slightly better UX 
- Slightly faster search

## Possible improvements 
- Clean up code
  - RAW SQL in code, move it to the `models` abstractions 
  - Missing interfaces for a few things 
  - Bundle the imports into a single js file instead of the import map, very slow on 2G for first paint
- Checkpoints 
  - Sync Logs could use checkpoints to only scrape or get offline meta instead of getting both at all times 
- Features 
  - Allow filtering by other fields (status, type, tags)
  

## License 
[MIT](/LICENSE)