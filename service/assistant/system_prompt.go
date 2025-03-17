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

package assistant

import (
	"context"
	"github.com/honeycombio/beeline-go"
	"github.com/pebble-dev/bobby-assistant/service/assistant/util/mapbox"
	"log"
	"strconv"
	"time"

	"github.com/pebble-dev/bobby-assistant/service/assistant/query"
	"github.com/pebble-dev/bobby-assistant/service/assistant/util"
)

func (ps *PromptSession) generateTimeSentence(ctx context.Context) string {
	tzOffset := ps.query.Get("tzOffset")
	tzOffsetInt, err := strconv.Atoi(tzOffset)
	if err != nil {
		log.Printf("Failed to parse tzOffset: %v", err)
		return ""
	}
	// tzOffset is in minutes, but Go wants seconds.
	now := time.Now().UTC().In(time.FixedZone("local", tzOffsetInt*60))
	return "The user's local time is " + now.Format("Mon, 2 Jan 2006 15:04:05-07:00") + ". "
}

func generateLanguageSentence(ctx context.Context) string {
	sentence := ""
	var units = query.PreferredUnitsFromContext(ctx)
	unitMap := map[string]string{
		"imperial": "imperial units",
		"metric":   "metric units",
		"uk":       "UK hybrid units (temperature in Celsius, wind speed in mph, etc.)",
		"both":     "both imperial and metric units",
	}
	units = unitMap[units]
	if units != "" {
		sentence += "Give measurements in " + units + ". Always specify the unit for temperature measurements."
	}
	var language = util.GetLanguageName(query.PreferredLanguageFromContext(ctx))
	if language != "" {
		sentence += "Respond in " + language + ". "
	}
	return sentence
}

func (ps *PromptSession) getPlaceFromLocation(ctx context.Context) (string, error) {
	// Use the Mapbox API to turn the user's longitude and latitude into a place name.
	// We don't want anything more specific than their town name, so we filter at that level ("place" in Mapbox terms).
	// We will return just a region or country if there isn't a nearby place.
	location := query.LocationFromContext(ctx)
	feature, err := mapbox.ReverseGeocode(ctx, location.Lon, location.Lat)
	if err != nil {
		return "", err
	}
	return feature.PlaceName, nil
}

func generateWidgetSentence(ctx context.Context) string {
	if !query.SupportsAnyWidgets(ctx) {
		return ""
	}
	sentence := "You can embed some widgets in your responses by using a special syntax. The following widgets are available:\n"
	if query.SupportsWidget(ctx, "weather") {
		sentence += "<!WEATHER-CURRENT location=[here|place name] units=[metric|imperial|uk hybrid]!>: embeds a weather widget showing the weather right now in the given location\n" +
			"<!WEATHER-SINGLE-DAY location=[here|place name] units=[metric|imperial|uk hybrid] day=[the name of a weekday, like Tuesday]!>: embeds a weather widget summarising the weather in the given location for a single day within the coming week.\n" +
			"<!WEATHER-MULTI-DAY location=[here|place name] units=[metric|imperial|uk hybrid]!>: embeds a weather widget summarising the weather in the given location for the next three days\n" +
			"Before including a weather widget, you *must* still look up the weather, and include a textual response after the widget. Always call get_weather first, then put the widget before any other text. If showing the weather for the user's current location, always use 'here' instead of a place name. If asked for only one day of weather, don't respond with multiple days.\n\n"
	}
	if query.SupportsWidget(ctx, "timer") {
		sentence += "<!TIMER targetTime=[time in ISO 8601 format] name=[name of the timer]!>: embeds a timer widget counting down to the given time. If the timer doesn't have a name, the `name` field can be omitted\n" +
			"If a user asks to see a timer, and the timer exists, you should *always* include that timer as a widget at the beginning of your response. Before including a timer widget, you *must* call get_timers first to verify when the timer is set for. Use the TIMER widget *only* when showing the user how long is left on their timer, not when setting one. \n\n"
	}
	return sentence
}

func (ps *PromptSession) generateSystemPrompt(ctx context.Context) string {
	ctx, span := beeline.StartSpan(ctx, "generate_system_prompt")
	defer span.Send()
	locationString := ""
	location := query.LocationFromContext(ctx)
	if location != nil {
		if place, err := ps.getPlaceFromLocation(ctx); err == nil {
			locationString = "The user is in " + place + ". "
		} else {
			span.AddField("error", err)
			log.Printf("Failed to get user location: %v", err)
		}
	} else {
		locationString = "The user has not granted permission to access their location, but they could enable it on the settings page if needed. "
	}
	return "You are a helpful assistant in the style of phone voice assistants. " +
		"Your name is Bobby, and you are running on a Pebble smartwatch. " +
		"The text you receive is transcribed from voice input. " +
		"Your knowledge cutoff is September 2024. However, you can use the wikipedia function to access the current content of specific Wikipedia pages. " +
		"Always follow Wikipedia redirects immediately and silently. Never ask the user whether you should check wikipedia, or whether you should check the full article - if you would ask, assume that you should (but don't ever fetch full articles if you already have the answer to the question). Don't mention looking up articles or Wikipedia to the user. " +
		locationString +
		ps.generateTimeSentence(ctx) +
		"You may call multiple functions before responding to the user, if necessary. If executing a lua script fails, try hard to fix the script using the error message, and consider alternate approaches to solve the problem. " +
		"If the user asks to set an alarm, assume they always want to set it for a time in the future. " +
		"As a creative, intelligent, helpful, friendly assistant, you should always try to answer the user's question. You can and should provide creative suggestions and factual responses as appropriate. Always try your best to answer the user's question. " +
		"**Never** claim to have taken an action (e.g. set a timer, alarm, or reminder) unless you have actually used a tool to do so. " +
		"Even if in previous turns you have apparently taken an action (like setting an alarm) without using a tool, you must still use tools if asked to do so again. " +
		"Alarms and reminders are not interchangable - never use alarms when a user asks for reminders, or vice-versa. If the user asks about a specific timer, respond only about that one. " +
		"Your responses will be displayed on a very small screen, so be brief. Do not use markdown in your responses.\n" +
		generateWidgetSentence(ctx) +
		generateLanguageSentence(ctx)
}
