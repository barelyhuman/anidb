package metadata

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Response struct {
	Data []Anime `json:"data"`
}

type Score struct {
	Median float64 `json:"median"`
}

type Anime struct {
	Name      string   `json:"title"`
	Picture   string   `json:"picture"`
	Thumbnail string   `json:"thumbnail"`
	Tags      []string `json:"tags"`
	Type      string   `json:"type"`
	Status    string   `json:"status"`
	Synonyms  []string `json:"synonyms"`
	Score     Score    `json:"score"`
}

var url = "https://raw.githubusercontent.com/manami-project/anime-offline-database/refs/heads/master/anime-offline-database-minified.json"

func GetOnlineMeta() ([]Anime, error) {
	var data Response

	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Error fetching JSON: %v\n", err)
		return data.Data, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Unexpected HTTP status: %s\n", resp.Status)
		return data.Data, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response body: %v\n", err)
		return data.Data, err
	}

	json.Unmarshal(body, &data)
	return data.Data, nil
}
