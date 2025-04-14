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

package widgets

import (
	"context"
	"errors"
	"fmt"
	"github.com/pebble-dev/bobby-assistant/service/assistant/query"
	"github.com/pebble-dev/bobby-assistant/service/assistant/util/photon"
	"github.com/pebble-dev/bobby-assistant/service/assistant/util/weather"
	"log"
	"strings"
)

type SingleDayWidgetContent struct {
	Location  string `json:"location"`
	Day       string `json:"day"`
	Condition int    `json:"condition"`
	Unit      string `json:"unit"`
	Summary   string `json:"summary"`
	High      int    `json:"high"`
	Low       int    `json:"low"`
}

type CurrentConditionsWidgetContent struct {
	Location      string `json:"location"`
	Condition     int    `json:"condition"`
	Temperature   int    `json:"temperature"`
	FeelsLike     int    `json:"feels_like"`
	Unit          string `json:"unit"`
	Description   string `json:"description"`
	WindSpeed     int    `json:"wind_speed"`
	WindSpeedUnit string `json:"wind_speed_unit"`
}

type MultiDayWidgetContent struct {
	Location string                     `json:"location"`
	Days     []MultiDayWidgetContentDay `json:"days"`
}

type MultiDayWidgetContentDay struct {
	Day       string `json:"day"`
	Condition int    `json:"condition"`
	High      int    `json:"high"`
	Low       int    `json:"low"`
}

var tempUnitMap = map[string]string{
	"imperial":  "°F",
	"metric":    "°C",
	"uk hybrid": "°C",
}

var windSpeedUnitMap = map[string]string{
	"imperial":  "mph",
	"metric":    "m/s",
	"uk hybrid": "mph",
}

func resolveLocation(ctx context.Context, location string) (string, query.Location, error) {
	var lat, lon float64
	if location == "here" {
		location := query.LocationFromContext(ctx)
		if location == nil {
			return "", query.Location{}, errors.New("can't get location without permission")
		}
		lat = location.Lat
		lon = location.Lon
	} else {
		// Look up the location
		coords, err := photon.GeocodeWithContext(ctx, location)
		if err != nil {
			return "", query.Location{}, fmt.Errorf("geocding location failed: %w", err)
		}
		lat = coords.Lat
		lon = coords.Lon
	}
	locationDisplayName := location
	// reverse geocode the location again so it's coherent
	feature, err := photon.ReverseGeocode(ctx, lon, lat)
	if err != nil {
		return "", query.Location{}, fmt.Errorf("reverse geocoding location failed: %w", err)
	}
	locationDisplayName = feature.PlaceName
	return locationDisplayName, query.Location{Lat: lat, Lon: lon}, nil
}

func singleDayWeatherWidget(ctx context.Context, placeName, units, date string) (*SingleDayWidgetContent, error) {
	locationDisplayName, location, err := resolveLocation(ctx, placeName)
	if err != nil {
		return nil, fmt.Errorf("resolving location failed: %w", err)
	}
	lat, lon := location.Lat, location.Lon

	w, err := weather.GetDailyForecast(ctx, lat, lon, units)
	if err != nil {
		return nil, fmt.Errorf("getting daily forecast failed: %w", err)
	}

	dayIndex := -1
	switch date {
	case "today":
		dayIndex = 0
	case "tomorrow":
		dayIndex = 1
	default:
		for i, day := range w.DayOfWeek {
			if strings.EqualFold(day, date) {
				dayIndex = i
				break
			}
		}
	}
	if dayIndex == -1 {
		return nil, fmt.Errorf("could not find day %q", date)
	}

	widget := &SingleDayWidgetContent{
		Location: locationDisplayName,
		Day:      w.DayOfWeek[dayIndex],
		High:     w.CalendarDayTemperatureMax[dayIndex],
		Low:      w.CalendarDayTemperatureMin[dayIndex],
		Unit:     tempUnitMap[units],
	}

	if len(w.DayParts) == 0 {
		return nil, fmt.Errorf("no day parts found")
	}

	dayPart := w.DayParts[0]

	dayPartIndex := dayIndex * 2
	if dayPart.DaypartName[dayPartIndex] == nil {
		dayPartIndex++
	}

	widget.Condition = *dayPart.IconCode[dayPartIndex]
	widget.Summary = *dayPart.WxPhraseLong[dayPartIndex]

	return widget, nil
}

func currentConditionsWeatherWidget(ctx context.Context, placeName, units string) (*CurrentConditionsWidgetContent, error) {
	locationDisplayName, location, err := resolveLocation(ctx, placeName)
	if err != nil {
		log.Printf("Error resolving location: %v", err)
		return nil, fmt.Errorf("resolving location failed: %w", err)
	}
	conditions, err := weather.GetCurrentConditions(ctx, location.Lat, location.Lon, units)
	if err != nil {
		log.Printf("Error getting current conditions: %v", err)
		return nil, fmt.Errorf("getting current conditions failed: %w", err)
	}
	return &CurrentConditionsWidgetContent{
		Location:      locationDisplayName,
		Condition:     conditions.IconCode,
		Temperature:   conditions.Temperature,
		FeelsLike:     conditions.TemperatureFeelsLike,
		Unit:          tempUnitMap[units],
		Description:   conditions.Description,
		WindSpeed:     conditions.WindSpeed,
		WindSpeedUnit: windSpeedUnitMap[units],
	}, nil
}

func multiDayWeatherWidget(ctx context.Context, placeName, units string) (*MultiDayWidgetContent, error) {
	locationDisplayName, location, err := resolveLocation(ctx, placeName)
	if err != nil {
		return nil, fmt.Errorf("resolving location failed: %w", err)
	}
	lat, lon := location.Lat, location.Lon

	w, err := weather.GetDailyForecast(ctx, lat, lon, units)
	if err != nil {
		return nil, fmt.Errorf("getting daily forecast failed: %w", err)
	}

	widget := &MultiDayWidgetContent{
		Location: locationDisplayName,
	}

	for i := 0; i < len(w.DayOfWeek); i++ {
		day := MultiDayWidgetContentDay{
			Day:  w.DayOfWeek[i],
			High: w.CalendarDayTemperatureMax[i],
			Low:  w.CalendarDayTemperatureMin[i],
		}
		dayPartIndex := i * 2
		if w.DayParts[0].IconCode[dayPartIndex] != nil {
			day.Condition = *w.DayParts[0].IconCode[dayPartIndex]
		} else {
			day.Condition = *w.DayParts[0].IconCode[dayPartIndex+1]
		}
		widget.Days = append(widget.Days, day)
	}

	return widget, nil
}
