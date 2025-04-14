package mapbox

import (
	"context"
	"encoding/json"
	"github.com/honeycombio/beeline-go"
	"github.com/pebble-dev/bobby-assistant/service/assistant/config"
	"net/http"
	"net/url"
)

type FeatureCollection struct {
	Features []Feature `json:"features"`
}

type Feature struct {
	ID         string     `json:"id"`
	PlaceType  []string   `json:"place_type"`
	Text       string     `json:"text"`
	Relevance  float64    `json:"relevance"`
	PlaceName  string     `json:"place_name"`
	Center     []float64  `json:"center"`
	Properties Properties `json:"properties"`
}

type Properties struct {
	Name        string   `json:"name"`
	Address     string   `json:"address"`
	POICategory []string `json:"poi_category"`
	Metadata    Metadata `json:"metadata"`
	Distance    float64  `json:"distance"`
}

type Metadata struct {
	Phone     string    `json:"phone"`
	Website   string    `json:"website"`
	OpenHours OpenHours `json:"open_hours"`
}

type OpenHours struct {
	Periods []Period `json:"periods"`
}

type Period struct {
	Open  TimePoint `json:"open"`
	Close TimePoint `json:"close"`
}

type TimePoint struct {
	Day  int    `json:"day"`
	Time string `json:"time"`
}

func SearchBoxRequest(ctx context.Context, params url.Values) (*FeatureCollection, error) {
	ctx, span := beeline.StartSpan(ctx, "mapbox.searchbox")
	defer span.Send()
	params.Set("access_token", config.GetConfig().MapboxKey)
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.mapbox.com/search/searchbox/v1/forward?"+params.Encode(), nil)
	if err != nil {
		span.AddField("error", err)
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		span.AddField("error", err)
		return nil, err
	}
	defer resp.Body.Close()
	var collection FeatureCollection
	if err := json.NewDecoder(resp.Body).Decode(&collection); err != nil {
		span.AddField("error", err)
		return nil, err
	}
	return &collection, nil
}
