/*
 * Copyright 2025 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

#include "segment_layer.h"
#include "../conversation.h"
#include "info_layer.h"
#include "message_layer.h"
#include "widgets/weather_single_day.h"
#include "widgets/weather_current.h"
#include "widgets/weather_multi_day.h"
#include "widgets/number.h"
#include "widgets/timer.h"
#include "widgets/map.h"
#include "../../util/memory/sdk.h"
#include "../../util/logging.h"
#include "../../features.h"

#include <pebble.h>



#define CONTENT_FONT FONT_KEY_GOTHIC_24_BOLD
#define NAME_HEIGHT 20

typedef enum {
  SegmentTypeNone,
  SegmentTypeMessage,
  SegmentTypeInfo,
  SegmentTypeWeatherSingleDayWidget,
  SegmentTypeWeatherCurrentWidget,
  SegmentTypeWeatherMultiDayWidget,
  SegmentTypeTimerWidget,
  SegmentTypeNumberWidget,
#if ENABLE_FEATURE_MAPS
  SegmentTypeMapWidget,
#endif
} SegmentType;

typedef struct {
  ConversationEntry* entry;
  TextLayer* assistant_label_layer;
  SegmentType type;
  union {
    // fun fact: every member of this union is actually a Layer*.
    // given this, we can get away with using a generic layer for layer-accepting functions.
    // also as a result, there is absolutely no warning if you call the wrong methods on these, sigh.
    Layer* layer;
    InfoLayer* info_layer;
    MessageLayer* message_layer;
    WeatherSingleDayWidget* weather_single_day_widget;
    WeatherCurrentWidget* weather_current_widget;
    WeatherMultiDayWidget* weather_multi_day_widget;
    TimerWidget* timer_widget;
    NumberWidget* number_widget;
#if ENABLE_FEATURE_MAPS
    MapWidget* map_widget;
#endif
  };
} SegmentLayerData;

static SegmentType prv_get_segment_type(ConversationEntry* entry);

SegmentLayer* segment_layer_create(GRect rect, ConversationEntry* entry, bool assistant_label) {
  Layer* layer = blayer_create_with_data(rect, sizeof(SegmentLayerData));
  SegmentLayerData* data = layer_get_data(layer);
  data->entry = entry;
  data->type = prv_get_segment_type(entry);
  GRect child_frame = GRect(0, 0, rect.size.w, rect.size.h);
  if (assistant_label) {
    data->assistant_label_layer = btext_layer_create(GRect(5, 0, rect.size.w, NAME_HEIGHT));
    layer_add_child(layer, text_layer_get_layer(data->assistant_label_layer));
    text_layer_set_text(data->assistant_label_layer, "Bobby");
    child_frame = GRect(0, NAME_HEIGHT, rect.size.w, rect.size.h - NAME_HEIGHT);
  } else {
    data->assistant_label_layer = NULL;
  }
  switch (data->type) {
    case SegmentTypeNone:
      break;
    case SegmentTypeMessage:
      data->message_layer = message_layer_create(child_frame, entry);
      break;
    case SegmentTypeInfo:
      data->info_layer = info_layer_create(child_frame, entry);
      break;
    case SegmentTypeWeatherSingleDayWidget:
      data->weather_single_day_widget = weather_single_day_widget_create(child_frame, entry);
      break;
    case SegmentTypeWeatherCurrentWidget:
      data->weather_current_widget = weather_current_widget_create(child_frame, entry);
      break;
    case SegmentTypeWeatherMultiDayWidget:
      data->weather_multi_day_widget = weather_multi_day_widget_create(child_frame, entry);
      break;
    case SegmentTypeTimerWidget:
      data->timer_widget = timer_widget_create(child_frame, entry);
      break;
    case SegmentTypeNumberWidget:
      data->number_widget = number_widget_create(child_frame, entry);
      break;
#if ENABLE_FEATURE_MAPS
    case SegmentTypeMapWidget:
      data->map_widget = map_widget_create(child_frame, entry);
      break;
#endif
  }
  layer_add_child(layer, data->layer);
  GSize child_size = layer_get_frame(data->layer).size;
  GRect final_size = GRect(rect.origin.x, rect.origin.y, child_size.w, child_size.h);
  if (data->assistant_label_layer) {
    final_size.size.h += NAME_HEIGHT;
  }
  layer_set_frame(layer, final_size);
  return layer;
}

void segment_layer_destroy(SegmentLayer* layer) {
  BOBBY_LOG(APP_LOG_LEVEL_INFO, "destroying SegmentLayer %p.", layer);
  SegmentLayerData* data = layer_get_data(layer);
  switch (data->type) {
    case SegmentTypeNone:
      break;
    case SegmentTypeMessage:
      message_layer_destroy(data->message_layer);
      break;
    case SegmentTypeInfo:
      info_layer_destroy(data->info_layer);
      break;
    case SegmentTypeWeatherSingleDayWidget:
      weather_single_day_widget_destroy(data->weather_single_day_widget);
      break;
    case SegmentTypeWeatherCurrentWidget:
      weather_current_widget_destroy(data->weather_current_widget);
      break;
    case SegmentTypeWeatherMultiDayWidget:
      weather_multi_day_widget_destroy(data->weather_multi_day_widget);
      break;
    case SegmentTypeTimerWidget:
      timer_widget_destroy(data->timer_widget);
      break;
    case SegmentTypeNumberWidget:
      number_widget_destroy(data->number_widget);
      break;
#if ENABLE_FEATURE_MAPS
    case SegmentTypeMapWidget:
      map_widget_destroy(data->map_widget);
      break;
#endif
  }
  if (data->assistant_label_layer) {
    text_layer_destroy(data->assistant_label_layer);
  }
  layer_destroy(layer);
}

ConversationEntry* segment_layer_get_entry(SegmentLayer* layer) {
  SegmentLayerData* data = layer_get_data(layer);
  return data->entry;
}

void segment_layer_update(SegmentLayer* layer) {
  SegmentLayerData* data = layer_get_data(layer);
  switch (data->type) {
    case SegmentTypeNone:
      break;
    case SegmentTypeMessage:
      message_layer_update(data->message_layer);
      break;
    case SegmentTypeInfo:
      info_layer_update(data->info_layer);
      break;
    case SegmentTypeWeatherSingleDayWidget:
      weather_single_day_widget_update(data->weather_single_day_widget);
      break;
    case SegmentTypeWeatherCurrentWidget:
      weather_current_widget_update(data->weather_current_widget);
      break;
    case SegmentTypeWeatherMultiDayWidget:
      weather_multi_day_widget_update(data->weather_multi_day_widget);
      break;
    case SegmentTypeTimerWidget:
      timer_widget_update(data->timer_widget);
      break;
    case SegmentTypeNumberWidget:
      number_widget_update(data->number_widget);
      break;
#if ENABLE_FEATURE_MAPS
    case SegmentTypeMapWidget:
      map_widget_update(data->map_widget);
      break;
#endif
  }
  GSize child_size = layer_get_frame(data->layer).size;
  GPoint origin = layer_get_frame(layer).origin;
  GRect final_frame = GRect(origin.x, origin.y, child_size.w, child_size.h);
  if (data->assistant_label_layer) {
    final_frame.size.h += NAME_HEIGHT;
  }
  layer_set_frame(layer, final_frame);
}

static SegmentType prv_get_segment_type(ConversationEntry* entry) {
  switch (conversation_entry_get_type(entry)) {
    case EntryTypeDeleted:
      return SegmentTypeNone;
    case EntryTypePrompt:
    case EntryTypeResponse:
      return SegmentTypeMessage;
    case EntryTypeThought:
    case EntryTypeError:
    case EntryTypeAction:
      return SegmentTypeInfo;
    case EntryTypeWidget:
      switch (conversation_entry_get_widget(entry)->type) {
        case ConversationWidgetTypeWeatherSingleDay:
          return SegmentTypeWeatherSingleDayWidget;
        case ConversationWidgetTypeWeatherCurrent:
          return SegmentTypeWeatherCurrentWidget;
        case ConversationWidgetTypeWeatherMultiDay:
          return SegmentTypeWeatherMultiDayWidget;
        case ConversationWidgetTypeTimer:
          return SegmentTypeTimerWidget;
        case ConversationWidgetTypeNumber:
          return SegmentTypeNumberWidget;
#if ENABLE_FEATURE_MAPS
        case ConversationWidgetTypeMap:
          return SegmentTypeMapWidget;
#endif
      }
      break;
  }
  BOBBY_LOG(APP_LOG_LEVEL_WARNING, "Unknown entry type %d.", conversation_entry_get_type(entry));
  return SegmentTypeNone;
}
