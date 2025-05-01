package photon

import (
    "context"
    "encoding/json"
    "fmt"
    "github.com/honeycombio/beeline-go"
    "github.com/pebble-dev/bobby-assistant/service/assistant/query"
    "net/http"
    "net/url"
)

type FeatureCollection struct {
    Features []Feature `json:"features"`
}

type Feature struct {
    Geometry   Geometry   `json:"geometry"`
    Type       string     `json:"type"`
    Properties Properties `json:"properties"`
    PlaceName  string     `json:"-"` // Computed field to match Mapbox interface
}

type Geometry struct {
    Coordinates []float64 `json:"coordinates"` // [lon, lat]
    Type        string    `json:"type"`
}

type Properties struct {
    Name      string `json:"name"`
    Street    string `json:"street,omitempty"`
    HouseNum  string `json:"housenumber,omitempty"`
    Postcode  string `json:"postcode,omitempty"`
    City      string `json:"city,omitempty"`
    State     string `json:"state,omitempty"`
    Country   string `json:"country,omitempty"`
    OSMId     int64  `json:"osm_id"`
    OSMType   string `json:"osm_type"`
    OSMKey    string `json:"osm_key"`
    OSMValue  string `json:"osm_value"`
}

type Location struct {
    Lat float64
    Lon float64
}

// generatePlaceName returns just the city name, or falls back to other location info if city is unavailable
func generatePlaceName(p Properties) string {
    // First try to use City if available
    if p.City != "" {
        return p.City
    }
    
    // Fall back to State if Name is not available
    if p.State != "" {
        return p.State
    }
    
    // Last resort: use Country
    if p.Country != "" {
        return p.Country
    }
    
    // If nothing is available
    return "Unknown location"
}

func sendRequest(ctx context.Context, url string) (*FeatureCollection, error) {
    ctx, span := beeline.StartSpan(ctx, "photon.request")
    defer span.Send()

    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
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

    // Populate the PlaceName field for each feature
    for i := range collection.Features {
        collection.Features[i].PlaceName = generatePlaceName(collection.Features[i].Properties)
    }

    return &collection, nil
}

// GeocodeWithContext converts a location name to coordinates
func GeocodeWithContext(ctx context.Context, search string) (Location, error) {
    ctx, span := beeline.StartSpan(ctx, "photon.geocode")
    defer span.Send()

    location := query.LocationFromContext(ctx)

    params := url.Values{}
    params.Set("q", search)
    params.Set("limit", "1")

    // If we have user location, use it for biasing results
    if location != nil {
        params.Set("lon", fmt.Sprintf("%f", location.Lon))
        params.Set("lat", fmt.Sprintf("%f", location.Lat))
    }

    apiURL := "https://photon.komoot.io/api/?" + params.Encode()

    collection, err := sendRequest(ctx, apiURL)
    if err != nil {
        return Location{}, fmt.Errorf("could not find location: %w", err)
    }

    if len(collection.Features) == 0 {
        return Location{}, fmt.Errorf("could not find location with name %q", search)
    }

    // Photon API returns coordinates as [lon, lat]
    lon := collection.Features[0].Geometry.Coordinates[0]
    lat := collection.Features[0].Geometry.Coordinates[1]

    return Location{
        Lat: lat,
        Lon: lon,
    }, nil
}

// ReverseGeocode converts coordinates to a location name
func ReverseGeocode(ctx context.Context, lon, lat float64) (*Feature, error) {
    ctx, span := beeline.StartSpan(ctx, "photon.reverse_geocode")
    defer span.Send()

    params := url.Values{}
    params.Set("lon", fmt.Sprintf("%f", lon))
    params.Set("lat", fmt.Sprintf("%f", lat))

    apiURL := "https://photon.komoot.io/reverse/?" + params.Encode()

    collection, err := sendRequest(ctx, apiURL)
    if err != nil {
        return nil, fmt.Errorf("could not reverse geocode location: %w", err)
    }

    if len(collection.Features) == 0 {
        return nil, fmt.Errorf("the user isn't anywhere")
    }

    return &collection.Features[0], nil
}