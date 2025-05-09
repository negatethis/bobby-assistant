/**
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

var config = require('../config');
var location = require('../location');
var reminders = require('./reminders');
var package_json = require('package.json');
var urls = require('../urls');
var session = require('../session');

function constructFeedbackMetadata(request) {
    var appVersion = 'unknown'
    var alarmCount = 0;
    if (request) {
        appVersion = '' + request['FEEDBACK_APP_MAJOR'] + '.' + request['FEEDBACK_APP_MINOR'];
        alarmCount = request['FEEDBACK_ALARM_COUNT'];
    }
    var locationEnabled = config.isLocationEnabled();
    var locationReady = location.isReady();
    var settings = config.getSettings();
    var unitPreference = settings['UNIT_PREFERENCE'] || '';
    var languagePreference = settings['LANGUAGE_CODE'] || '';
    var reminderCount = reminders.getAllReminders().length;
    var jsVersion = package_json['version'];
    var timezone = (-(new Date()).getTimezoneOffset());
    var platform = 'unknown';
    if (window.cobble) {
        platform = 'Cobble';
    } else if (window.navigator) {
        var userAgent = navigator.userAgent;
        var androidVersionRegex = /Android (\d+(?:\.\d+)?)/;
        var androidVersion = androidVersionRegex.exec(userAgent);
        if (androidVersion) {
            platform = 'Android ' + androidVersion[1];
        } else {
            platform = 'iOS';
        }
    } else {
        platform = 'iOS';
    }
    var watch = Pebble.getActiveWatchInfo ? Pebble.getActiveWatchInfo() : null;
    var watchFirmware = watch ? '' + watch.firmware.major + '.' + watch.firmware.minor + '.' + watch.firmware.patch : '(unknown)';
    if (watch && watch.firmware.suffix) {
        watchFirmware += '-' + watch.firmware.suffix;
    }
    var watchPlatform = watch ? watch.platform : '(unknown)';
    var model = watch ? watch.model : '(unknown)';
    return {
        'appVersion': appVersion,
        'alarmCount': alarmCount,
        'locationEnabled': locationEnabled,
        'locationReady': locationReady,
        'unitPreference': unitPreference,
        'languagePreference': languagePreference,
        'reminderCount': reminderCount,
        'jsVersion': jsVersion,
        'timezone': timezone,
        'platform': platform,
        'watchFirmware': watchFirmware,
        'watchModel': model,
        'watchPlatform': watchPlatform,
        'timelineToken': session.userToken
    };
}

function sendRequest(request, url, callback) {
    var req = new XMLHttpRequest();
    req.open('POST', url, true);
    req.setRequestHeader('Content-Type', 'application/json');
    req.onload = function(e) {
        if (req.readyState === 4) {
            if (req.status === 200) {
                callback(true, req.status);
                console.log("Feedback sent successfully");
            } else {
                console.log("Feedback request returned error code " + req.status.toString());
                callback(false, req.status);
            }
        }
    }
    console.log("Feedback request: " + JSON.stringify(request));
    req.send(JSON.stringify(request));
}

exports.sendFeedback = function(feedbackText, threadId, callback) {
    var feedback = constructFeedbackMetadata(null);
    if (feedbackText) {
        feedback['text'] = feedbackText;
    }
    if (threadId) {
        feedback['thread_uuid'] = threadId;
    }
    sendRequest(feedback, urls.REPORT_URL, callback);
}

exports.handleFeedbackRequest = function(request) {
    var feedback = constructFeedbackMetadata(request);
    feedback['text'] = request['FEEDBACK_TEXT'];
    sendRequest(feedback, urls.REPORT_URL, function(success, status) {
        Pebble.sendAppMessage({'FEEDBACK_SEND_RESULT': success ? 0 : 1});
    });
}

exports.handleReportRequest = function(request) {
    var report = constructFeedbackMetadata(request);
    report['thread_uuid'] = request['REPORT_THREAD_UUID'];
    sendRequest(report, urls.REPORT_URL, function(success, status) {
        Pebble.sendAppMessage({'REPORT_SEND_RESULT': success ? 0 : 1});
    });
}
