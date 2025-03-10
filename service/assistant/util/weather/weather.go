// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package weather

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "strings"
    "time"
)

// Weather data structures for the API response
type Forecast struct {
    CalendarDayTemperatureMax []int
    CalendarDayTemperatureMin []int
    DayOfWeek                 []string
    MoonPhaseCode             []string
    MoonPhase                 []string
    MoonPhaseDay              []int
    Narrative                 []string
    SunriseTimeLocal          []string
    SunsetTimeLocal           []string
    MoonriseTimeLocal         []string
    MoonsetTimeLocal          []string
    Qpf                       []float32
    QpfSnow                   []float32
    DayParts                  []ForecastDayPart
}

type ForecastDayPart struct {
    CloudCover            []*int
    DayOrNight            []*string
    DaypartName           []*string
    IconCode              []*int
    IconCodeExtend        []*int
    Narrative             []*string
    PrecipChance          []*int
    PrecipType            []*string
    Temperature           []*int
    WindDirectionCardinal []*string
    WindSpeed             []*int
    WxPhraseLong          []*string
}

type CurrentConditions struct {
    CloudCover             int
    CloudCoverPhrase       string
    DayOfWeek              string
    DayOrNight             string
    Description            string
    IconCode               int
    Precip1Hour            float32
    RelativeHumidity       int
    SunriseTimeLocal       string
    SunsetTimeLocal        string
    Temperature            int
    TemperatureFeelsLike   int
    TemperatureMax24Hour   int
    TemperatureMin24Hour   int
    TemperatureWindChill   int
    UVIndex                int
    Visibility             float32
    WindDirectionCardinal  string
    WindSpeed              int
}

type HourlyForecast struct {
    Temperature    []int
    WxPhraseLong   []string
    PrecipChance   []int
    PrecipType     []string
    ValidTimeLocal []string
    UVIndex        []int
}

type openMeteoParams struct {
    tempUnit    string
    windUnit    string
    precipUnit  string
    timeFormat  string
}

func mapUnit(unit string) (openMeteoParams, error) {
    params := openMeteoParams{
        timeFormat: "iso8601",
    }
    
    switch unit {
    case "imperial":
        params.tempUnit = "fahrenheit"
        params.windUnit = "mph"
        params.precipUnit = "inch"
    case "metric":
        params.tempUnit = "celsius"
        params.windUnit = "kmh"
        params.precipUnit = "mm"
    case "uk hybrid":
        params.tempUnit = "celsius"
        params.windUnit = "mph"
        params.precipUnit = "mm"
    default:
        return params, fmt.Errorf("unit must be one of 'imperial', 'metric', or 'uk hybrid'; not %q", unit)
    }
    return params, nil
}

// OpenMeteo response structures
type openMeteoResponse struct {
    Latitude             float64                 `json:"latitude"`
    Longitude            float64                 `json:"longitude"`
    Elevation            float64                 `json:"elevation"`
    GenerationTimeMs     float64                 `json:"generationtime_ms"`
    UtcOffsetSeconds     int                     `json:"utc_offset_seconds"`
    Timezone             string                  `json:"timezone"`
    TimezoneAbbreviation string                  `json:"timezone_abbreviation"`
    CurrentWeather       *openMeteoCurrentWeather `json:"current_weather,omitempty"`
    Daily                *openMeteoDaily         `json:"daily,omitempty"`
    DailyUnits           *openMeteoUnits         `json:"daily_units,omitempty"`
    Hourly               *openMeteoHourly        `json:"hourly,omitempty"`
    HourlyUnits          *openMeteoUnits         `json:"hourly_units,omitempty"`
}

type openMeteoCurrentWeather struct {
    Temperature      float64 `json:"temperature"`
    Windspeed       float64 `json:"windspeed"`
    WindDirection   float64 `json:"winddirection"`
    WeatherCode     int     `json:"weathercode"`
    IsDay           int     `json:"is_day"`
    Time            string  `json:"time"`
    RelativeHumidity float64 `json:"relativehumidity_2m,omitempty"`
    ApparentTemperature float64 `json:"apparent_temperature,omitempty"`
    Precipitation   float64 `json:"precipitation,omitempty"`
    Visibility      float64 `json:"visibility,omitempty"`
    CloudCover      float64 `json:"cloudcover,omitempty"`
}

type openMeteoDaily struct {
    Time                 []string  `json:"time"`
    WeatherCode          []int     `json:"weathercode"`
    TemperatureMax       []float64 `json:"temperature_2m_max"`
    TemperatureMin       []float64 `json:"temperature_2m_min"`
    SunriseIso           []string  `json:"sunrise"`
    SunsetIso            []string  `json:"sunset"`
    PrecipitationSum     []float64 `json:"precipitation_sum"`
    PrecipitationHours   []float64 `json:"precipitation_hours"`
    PrecipitationProbabilityMax []float64 `json:"precipitation_probability_max"`
    WindspeedMax         []float64 `json:"windspeed_10m_max"`
    WinddirectionDominant []int     `json:"winddirection_10m_dominant"`
    UvIndexMax           []float64 `json:"uv_index_max"`
}

type openMeteoHourly struct {
    Time                []string  `json:"time"`
    Temperature         []float64 `json:"temperature_2m"`
    PrecipitationProbability []float64 `json:"precipitation_probability"`
    Precipitation       []float64 `json:"precipitation"`
    WeatherCode         []int     `json:"weathercode"`
    Visibility          []float64 `json:"visibility"`
    Windspeed           []float64 `json:"windspeed_10m"`
    WindDirection       []float64 `json:"winddirection_10m"`
    UvIndex             []float64 `json:"uv_index"`
    IsDay               []int     `json:"is_day"`
    RelativeHumidity    []float64 `json:"relativehumidity_2m"`
    ApparentTemperature []float64 `json:"apparent_temperature"`
}

type openMeteoUnits map[string]string

func GetDailyForecast(ctx context.Context, lat, lon float64, units string) (*Forecast, error) {
    params, err := mapUnit(units)
    if err != nil {
        return nil, err
    }

    url := fmt.Sprintf(
        "https://api.open-meteo.com/v1/forecast?latitude=%f&longitude=%f&daily=weathercode,temperature_2m_max,temperature_2m_min,sunrise,sunset,precipitation_sum,precipitation_hours,precipitation_probability_max,windspeed_10m_max,winddirection_10m_dominant,uv_index_max&timeformat=%s&temperature_unit=%s&windspeed_unit=%s&precipitation_unit=%s",
        lat, lon, params.timeFormat, params.tempUnit, params.windUnit, params.precipUnit)
    
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, fmt.Errorf("error creating request: %w", err)
    }
    
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("error making request: %w", err)
    }
    defer resp.Body.Close()
    
    var openMeteoResp openMeteoResponse
    if err := json.NewDecoder(resp.Body).Decode(&openMeteoResp); err != nil {
        return nil, fmt.Errorf("error decoding response: %w", err)
    }
    
    if openMeteoResp.Daily == nil {
        return nil, fmt.Errorf("no daily forecast data received")
    }

    // Convert to our format
    forecast := &Forecast{
        CalendarDayTemperatureMax: make([]int, len(openMeteoResp.Daily.Time)),
        CalendarDayTemperatureMin: make([]int, len(openMeteoResp.Daily.Time)),
        DayOfWeek:                 make([]string, len(openMeteoResp.Daily.Time)),
        MoonPhaseCode:             make([]string, len(openMeteoResp.Daily.Time)),
        MoonPhase:                 make([]string, len(openMeteoResp.Daily.Time)),
        MoonPhaseDay:              make([]int, len(openMeteoResp.Daily.Time)),
        Narrative:                 make([]string, len(openMeteoResp.Daily.Time)),
        SunriseTimeLocal:          make([]string, len(openMeteoResp.Daily.Time)),
        SunsetTimeLocal:           make([]string, len(openMeteoResp.Daily.Time)),
        MoonriseTimeLocal:         make([]string, len(openMeteoResp.Daily.Time)),
        MoonsetTimeLocal:          make([]string, len(openMeteoResp.Daily.Time)),
        Qpf:                       make([]float32, len(openMeteoResp.Daily.Time)),
        QpfSnow:                   make([]float32, len(openMeteoResp.Daily.Time)),
    }

    // Map data from Open-Meteo to our structure
    for i, timeStr := range openMeteoResp.Daily.Time {
        t, _ := time.Parse("2006-01-02", timeStr)
        forecast.DayOfWeek[i] = t.Format("Monday")
        forecast.CalendarDayTemperatureMax[i] = int(openMeteoResp.Daily.TemperatureMax[i])
        forecast.CalendarDayTemperatureMin[i] = int(openMeteoResp.Daily.TemperatureMin[i])
        forecast.SunriseTimeLocal[i] = openMeteoResp.Daily.SunriseIso[i]
        forecast.SunsetTimeLocal[i] = openMeteoResp.Daily.SunsetIso[i]
        forecast.Qpf[i] = float32(openMeteoResp.Daily.PrecipitationSum[i])
        
        // Generate a narrative based on weather code and temperatures
        weatherDesc := weatherCodeToDescription(openMeteoResp.Daily.WeatherCode[i])
        forecast.Narrative[i] = fmt.Sprintf("%s with high of %d and low of %d. %d%% chance of precipitation.", 
            weatherDesc, 
            int(openMeteoResp.Daily.TemperatureMax[i]), 
            int(openMeteoResp.Daily.TemperatureMin[i]),
            int(openMeteoResp.Daily.PrecipitationProbabilityMax[i]))
        
        // We don't have moon phase data from Open-Meteo, using placeholders
        forecast.MoonPhaseCode[i] = "N"
        forecast.MoonPhase[i] = "Not available"
        forecast.MoonPhaseDay[i] = 0
        forecast.MoonriseTimeLocal[i] = ""
        forecast.MoonsetTimeLocal[i] = ""
        forecast.QpfSnow[i] = 0 // Open-Meteo doesn't provide separate snow data in free tier
    }
    
    // Create day parts
    forecast.DayParts = []ForecastDayPart{
        {
            CloudCover:            make([]*int, len(openMeteoResp.Daily.Time)*2),
            DayOrNight:            make([]*string, len(openMeteoResp.Daily.Time)*2),
            DaypartName:           make([]*string, len(openMeteoResp.Daily.Time)*2),
            IconCode:              make([]*int, len(openMeteoResp.Daily.Time)*2),
            IconCodeExtend:        make([]*int, len(openMeteoResp.Daily.Time)*2),
            Narrative:             make([]*string, len(openMeteoResp.Daily.Time)*2),
            PrecipChance:          make([]*int, len(openMeteoResp.Daily.Time)*2),
            PrecipType:            make([]*string, len(openMeteoResp.Daily.Time)*2),
            Temperature:           make([]*int, len(openMeteoResp.Daily.Time)*2),
            WindDirectionCardinal: make([]*string, len(openMeteoResp.Daily.Time)*2),
            WindSpeed:             make([]*int, len(openMeteoResp.Daily.Time)*2),
            WxPhraseLong:          make([]*string, len(openMeteoResp.Daily.Time)*2),
        },
    }

    // Create day/night entries for each day
    for i := range openMeteoResp.Daily.Time {
        // Day
        day := "day"
        night := "night"
        dayName := fmt.Sprintf("Day %d", i+1)
        nightName := fmt.Sprintf("Night %d", i+1)
        
        dayIndex := i * 2
        nightIndex := i*2 + 1
        
        iconCode := weatherCodeToIconCode(openMeteoResp.Daily.WeatherCode[i])
        weatherDesc := weatherCodeToDescription(openMeteoResp.Daily.WeatherCode[i])
        dayNarrative := fmt.Sprintf("%s with high of %d. %d%% chance of precipitation.", 
            weatherDesc, int(openMeteoResp.Daily.TemperatureMax[i]), int(openMeteoResp.Daily.PrecipitationProbabilityMax[i]))
        nightNarrative := fmt.Sprintf("%s with low of %d. %d%% chance of precipitation.",
            weatherDesc, int(openMeteoResp.Daily.TemperatureMin[i]), int(openMeteoResp.Daily.PrecipitationProbabilityMax[i]))
        
        precipChance := int(openMeteoResp.Daily.PrecipitationProbabilityMax[i])
        
        var precipType string
        if precipChance > 0 {
            precipType = "rain" // Simplification since we don't have detailed precip type
        } else {
            precipType = ""
        }
        
        windDir := cardinalFromDegrees(openMeteoResp.Daily.WinddirectionDominant[i])
        windSpeed := int(openMeteoResp.Daily.WindspeedMax[i])
        
        // Day values
        forecast.DayParts[0].DayOrNight[dayIndex] = &day
        forecast.DayParts[0].DaypartName[dayIndex] = &dayName
        forecast.DayParts[0].IconCode[dayIndex] = &iconCode
        forecast.DayParts[0].IconCodeExtend[dayIndex] = &iconCode
        forecast.DayParts[0].Narrative[dayIndex] = &dayNarrative
        forecast.DayParts[0].PrecipChance[dayIndex] = &precipChance
        forecast.DayParts[0].PrecipType[dayIndex] = &precipType
        forecast.DayParts[0].Temperature[dayIndex] = intPtr(int(openMeteoResp.Daily.TemperatureMax[i]))
        forecast.DayParts[0].WindDirectionCardinal[dayIndex] = &windDir
        forecast.DayParts[0].WindSpeed[dayIndex] = &windSpeed
        forecast.DayParts[0].WxPhraseLong[dayIndex] = &weatherDesc
        
        // Night values
        forecast.DayParts[0].DayOrNight[nightIndex] = &night
        forecast.DayParts[0].DaypartName[nightIndex] = &nightName
        forecast.DayParts[0].IconCode[nightIndex] = &iconCode
        forecast.DayParts[0].IconCodeExtend[nightIndex] = &iconCode
        forecast.DayParts[0].Narrative[nightIndex] = &nightNarrative
        forecast.DayParts[0].PrecipChance[nightIndex] = &precipChance
        forecast.DayParts[0].PrecipType[nightIndex] = &precipType
        forecast.DayParts[0].Temperature[nightIndex] = intPtr(int(openMeteoResp.Daily.TemperatureMin[i]))
        forecast.DayParts[0].WindDirectionCardinal[nightIndex] = &windDir
        forecast.DayParts[0].WindSpeed[nightIndex] = &windSpeed
        forecast.DayParts[0].WxPhraseLong[nightIndex] = &weatherDesc
    }

    return forecast, nil
}

func GetCurrentConditions(ctx context.Context, lat, lon float64, units string) (*CurrentConditions, error) {
    params, err := mapUnit(units)
    if err != nil {
        return nil, err
    }

    url := fmt.Sprintf(
        "https://api.open-meteo.com/v1/forecast?latitude=%f&longitude=%f&current_weather=true&hourly=temperature_2m,relativehumidity_2m,apparent_temperature,precipitation,visibility,cloudcover,weathercode&daily=temperature_2m_max,temperature_2m_min,sunrise,sunset&timeformat=%s&temperature_unit=%s&windspeed_unit=%s&precipitation_unit=%s",
        lat, lon, params.timeFormat, params.tempUnit, params.windUnit, params.precipUnit)
    
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, fmt.Errorf("error creating request: %w", err)
    }
    
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("error making request: %w", err)
    }
    defer resp.Body.Close()
    
    var openMeteoResp openMeteoResponse
    if err := json.NewDecoder(resp.Body).Decode(&openMeteoResp); err != nil {
        return nil, fmt.Errorf("error decoding response: %w", err)
    }
    
    if openMeteoResp.CurrentWeather == nil {
        return nil, fmt.Errorf("no current weather data received")
    }

    // Find current time in hourly data to get additional fields
    currentTime := openMeteoResp.CurrentWeather.Time
    currentTimeIndex := -1
    for i, t := range openMeteoResp.Hourly.Time {
        if strings.HasPrefix(t, currentTime) {
            currentTimeIndex = i
            break
        }
    }

    // Get day of week
    t, _ := time.Parse(time.RFC3339, openMeteoResp.CurrentWeather.Time)
    dayOfWeek := t.Format("Monday")
    
    // Create current conditions object
    conditions := &CurrentConditions{
        Temperature:           int(openMeteoResp.CurrentWeather.Temperature),
        WindSpeed:             int(openMeteoResp.CurrentWeather.Windspeed),
        WindDirectionCardinal: cardinalFromDegrees(int(openMeteoResp.CurrentWeather.WindDirection)),
        IconCode:              weatherCodeToIconCode(openMeteoResp.CurrentWeather.WeatherCode),
        Description:           weatherCodeToDescription(openMeteoResp.CurrentWeather.WeatherCode),
        DayOfWeek:             dayOfWeek,
    }

    // Set day or night
    if openMeteoResp.CurrentWeather.IsDay == 1 {
        conditions.DayOrNight = "D"
    } else {
        conditions.DayOrNight = "N"
    }

    // Add additional data if we found the current time in hourly data
    if currentTimeIndex >= 0 && openMeteoResp.Hourly != nil {
        conditions.RelativeHumidity = int(openMeteoResp.Hourly.RelativeHumidity[currentTimeIndex])
        conditions.TemperatureFeelsLike = int(openMeteoResp.Hourly.ApparentTemperature[currentTimeIndex])
        conditions.Precip1Hour = float32(openMeteoResp.Hourly.Precipitation[currentTimeIndex])
        
        // Set visibility - scale to miles or km as needed
        if params.tempUnit == "fahrenheit" {
            // Convert from meters to miles
            conditions.Visibility = float32(openMeteoResp.Hourly.Visibility[currentTimeIndex] / 1609.34)
        } else {
            // Convert from meters to km
            conditions.Visibility = float32(openMeteoResp.Hourly.Visibility[currentTimeIndex] / 1000)
        }
        
        conditions.CloudCover = int(openMeteoResp.Hourly.Visibility[currentTimeIndex])
        
        // Cloud cover phrase
        if conditions.CloudCover < 10 {
            conditions.CloudCoverPhrase = "Clear"
        } else if conditions.CloudCover < 30 {
            conditions.CloudCoverPhrase = "Mostly Clear"
        } else if conditions.CloudCover < 60 {
            conditions.CloudCoverPhrase = "Partly Cloudy"
        } else if conditions.CloudCover < 90 {
            conditions.CloudCoverPhrase = "Mostly Cloudy"
        } else {
            conditions.CloudCoverPhrase = "Cloudy"
        }
    }

    // Add sunrise/sunset data
    if openMeteoResp.Daily != nil && len(openMeteoResp.Daily.SunriseIso) > 0 {
        conditions.SunriseTimeLocal = openMeteoResp.Daily.SunriseIso[0]
        conditions.SunsetTimeLocal = openMeteoResp.Daily.SunsetIso[0]
    }

    // Set min/max temps
    if openMeteoResp.Daily != nil && len(openMeteoResp.Daily.TemperatureMax) > 0 {
        conditions.TemperatureMax24Hour = int(openMeteoResp.Daily.TemperatureMax[0])
        conditions.TemperatureMin24Hour = int(openMeteoResp.Daily.TemperatureMin[0])
    }
    
    // Wind chill is same as feels like in cold conditions, otherwise same as temperature
    if conditions.TemperatureFeelsLike < conditions.Temperature {
        conditions.TemperatureWindChill = conditions.TemperatureFeelsLike
    } else {
        conditions.TemperatureWindChill = conditions.Temperature
    }

    // Set UV Index to a default value as Open-Meteo doesn't provide current UV
    if currentTimeIndex >= 0 && openMeteoResp.Hourly != nil {
        conditions.UVIndex = int(openMeteoResp.Hourly.UvIndex[currentTimeIndex])
    } else {
        conditions.UVIndex = 0
    }

    return conditions, nil
}

func GetHourlyForecast(ctx context.Context, lat, lon float64, units string) (*HourlyForecast, error) {
    params, err := mapUnit(units)
    if err != nil {
        return nil, err
    }

    url := fmt.Sprintf(
        "https://api.open-meteo.com/v1/forecast?latitude=%f&longitude=%f&hourly=temperature_2m,precipitation_probability,precipitation,weathercode,uv_index&timeformat=%s&temperature_unit=%s&windspeed_unit=%s&precipitation_unit=%s&forecast_days=2",
        lat, lon, params.timeFormat, params.tempUnit, params.windUnit, params.precipUnit)
    
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, fmt.Errorf("error creating request: %w", err)
    }
    
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("error making request: %w", err)
    }
    defer resp.Body.Close()
    
    var openMeteoResp openMeteoResponse
    if err := json.NewDecoder(resp.Body).Decode(&openMeteoResp); err != nil {
        return nil, fmt.Errorf("error decoding response: %w", err)
    }
    
    if openMeteoResp.Hourly == nil {
        return nil, fmt.Errorf("no hourly forecast data received")
    }

    // Map to hourly forecast
    forecast := &HourlyForecast{
        Temperature:    make([]int, len(openMeteoResp.Hourly.Time)),
        WxPhraseLong:   make([]string, len(openMeteoResp.Hourly.Time)),
        PrecipChance:   make([]int, len(openMeteoResp.Hourly.Time)),
        PrecipType:     make([]string, len(openMeteoResp.Hourly.Time)),
        ValidTimeLocal: make([]string, len(openMeteoResp.Hourly.Time)),
        UVIndex:        make([]int, len(openMeteoResp.Hourly.Time)),
    }

    for i, timeStr := range openMeteoResp.Hourly.Time {
        forecast.Temperature[i] = int(openMeteoResp.Hourly.Temperature[i])
        forecast.WxPhraseLong[i] = weatherCodeToDescription(openMeteoResp.Hourly.WeatherCode[i])
        forecast.PrecipChance[i] = int(openMeteoResp.Hourly.PrecipitationProbability[i])
        forecast.ValidTimeLocal[i] = timeStr
        forecast.UVIndex[i] = int(openMeteoResp.Hourly.UvIndex[i])
        
        // Determine precip type (simple logic)
        if forecast.PrecipChance[i] > 0 {
            forecast.PrecipType[i] = "rain"
        } else {
            forecast.PrecipType[i] = ""
        }
    }

    return forecast, nil
}

// Helper functions
func intPtr(i int) *int {
    return &i
}

func cardinalFromDegrees(degrees int) string {
    directions := []string{"N", "NNE", "NE", "ENE", "E", "ESE", "SE", "SSE", "S", "SSW", "SW", "WSW", "W", "WNW", "NW", "NNW"}
    index := int((float64(degrees) + 11.25) / 22.5) % 16
    return directions[index]
}

func weatherCodeToDescription(code int) string {
    // WMO Weather interpretation codes (WW)
    // https://www.nodc.noaa.gov/archive/arc0021/0002199/1.1/data/0-data/HTML/WMO-CODE/WMO4677.HTM
    switch {
    case code == 0:
        return "Clear sky"
    case code == 1:
        return "Mainly clear"
    case code == 2:
        return "Partly cloudy"
    case code == 3:
        return "Overcast"
    case code >= 45 && code <= 48:
        return "Fog"
    case code >= 51 && code <= 55:
        return "Drizzle"
    case code >= 56 && code <= 57:
        return "Freezing Drizzle"
    case code >= 61 && code <= 65:
        return "Rain"
    case code >= 66 && code <= 67:
        return "Freezing Rain"
    case code >= 71 && code <= 75:
        return "Snow"
    case code == 77:
        return "Snow grains"
    case code >= 80 && code <= 82:
        return "Rain showers"
    case code >= 85 && code <= 86:
        return "Snow showers"
    case code == 95:
        return "Thunderstorm"
    case code >= 96 && code <= 99:
        return "Thunderstorm with hail"
    default:
        return "Unknown"
    }
}

func weatherCodeToIconCode(code int) int {
    // Map Open-Meteo weather codes to original icon codes
    // Using approximate mappings
    switch {
    case code == 0:
        return 32 // Sunny
    case code == 1:
        return 34 // Mostly Sunny
    case code == 2:
        return 30 // Partly Cloudy
    case code == 3:
        return 26 // Cloudy
    case code >= 45 && code <= 48:
        return 20 // Fog
    case code >= 51 && code <= 55:
        return 11 // Drizzle
    case code >= 56 && code <= 57:
        return 8 // Freezing Drizzle
    case code >= 61 && code <= 65:
        return 12 // Rain
    case code >= 66 && code <= 67:
        return 10 // Freezing Rain
    case code >= 71 && code <= 75:
        return 16 // Snow
    case code == 77:
        return 16 // Snow grains
    case code >= 80 && code <= 82:
        return 39 // Rain showers
    case code >= 85 && code <= 86:
        return 41 // Snow showers
    case code == 95:
        return 4 // Thunderstorm
    case code >= 96 && code <= 99:
        return 17 // Thunderstorm with hail
    default:
        return 32 // Default sunny
    }
}