"use strict";
$(document).ready(function () {
    window.i18n = {
        locale: null,
        values: {},

        setTranslations: function (locale, values) {
            this.locale = locale;
            this.values = values || {};
        },

        t: function (key, fallback = '') {
            return this.values[key] ?? fallback ?? key;
        }
    };

    $.ajax({
        url: '/api/language',
        method: 'GET',
        dataType: 'json',
        success: function (response) {
            if (response.status === 1 && response.data) {
                i18n.setTranslations(
                    response.data.code,
                    response.data.values
                );
            }
        },
        error: function () {
            console.error('Failed to load translations');
        }
    });

    // System clock
    if ($('#xeneon-clock').length) {
        function formatClock() {
            const now = new Date();
            $('#clockTime').text(now.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }));
            $('#clockDate').text(now.toLocaleDateString([], { weekday: 'long', month: 'long', day: 'numeric' }));
        }
        formatClock();
        setInterval(formatClock, 1000);
    }

    // System weather
    if ($('#xeneon-weather').length) {
        const WEATHER_REFRESH_MS = 10 * 60 * 1000;
        const FALLBACK_LOCATION = {
            name: 'New York',
            country: 'United States',
            latitude: 40.7128,
            longitude: -74.0060,
            source: 'Fallback city'
        };

        function weatherCodeToLabel(code, isDay) {
            const map = {
                0: isDay ? ['Clear sky', '☀️'] : ['Clear sky', '🌙'],
                1: isDay ? ['Mainly clear', '🌤️'] : ['Mainly clear', '🌙'],
                2: ['Partly cloudy', '⛅'],
                3: ['Overcast', '☁️'],
                45: ['Fog', '🌫️'],
                48: ['Depositing rime fog', '🌫️'],
                51: ['Light drizzle', '🌦️'],
                53: ['Drizzle', '🌦️'],
                55: ['Dense drizzle', '🌧️'],
                56: ['Freezing drizzle', '🌧️'],
                57: ['Dense freezing drizzle', '🌧️'],
                61: ['Slight rain', '🌦️'],
                63: ['Rain', '🌧️'],
                65: ['Heavy rain', '🌧️'],
                66: ['Freezing rain', '🌧️'],
                67: ['Heavy freezing rain', '🌧️'],
                71: ['Slight snow', '🌨️'],
                73: ['Snow', '🌨️'],
                75: ['Heavy snow', '❄️'],
                77: ['Snow grains', '❄️'],
                80: ['Rain showers', '🌦️'],
                81: ['Rain showers', '🌧️'],
                82: ['Violent rain showers', '⛈️'],
                85: ['Snow showers', '🌨️'],
                86: ['Heavy snow showers', '❄️'],
                95: ['Thunderstorm', '⛈️'],
                96: ['Thunderstorm with hail', '⛈️'],
                99: ['Thunderstorm with hail', '⛈️']
            };
            return map[code] || ['Unknown', '🌡️'];
        }
        
        function reverseGeocode(lat, lon) {
            return $.getJSON('https://nominatim.openstreetmap.org/reverse', {
                format: 'jsonv2',
                lat: lat,
                lon: lon,
                zoom: 10,
                addressdetails: 1
            });
        }

        function resolveLocation() {
            const dfd = $.Deferred();

            if (!navigator.geolocation) {
                dfd.resolve(FALLBACK_LOCATION);
                return dfd.promise();
            }

            navigator.geolocation.getCurrentPosition(function (position) {
                const latitude = position.coords.latitude;
                const longitude = position.coords.longitude;

                reverseGeocode(latitude, longitude)
                    .done(function (geo) {
                        const address = geo.address || {};
                        dfd.resolve({
                            name: address.city || address.town || address.village || address.county || 'Current location',
                            country: address.country || '',
                            latitude: latitude,
                            longitude: longitude,
                            source: 'Device location'
                        });
                    })
                    .fail(function () {
                        dfd.resolve({
                            name: 'Current location',
                            country: '',
                            latitude: latitude,
                            longitude: longitude,
                            source: 'Device coordinates'
                        });
                    });
            }, function () {
                dfd.resolve(FALLBACK_LOCATION);
            }, {
                enableHighAccuracy: false,
                timeout: 8000,
                maximumAge: 15 * 60 * 1000
            });

            return dfd.promise();
        }

        function setWeatherStatus(text) {
            $('#weatherRefreshState').text(text);
        }

        function fetchWeatherForLocation(location) {
            const url = 'https://api.open-meteo.com/v1/forecast';
            return $.getJSON(url, {
                latitude: location.latitude,
                longitude: location.longitude,
                current: 'temperature_2m,apparent_temperature,relative_humidity_2m,weather_code,wind_speed_10m,is_day,precipitation_probability',
                hourly: 'temperature_2m',
                forecast_hours: 8,
                timezone: 'auto',
                temperature_unit: 'celsius',
                wind_speed_unit: 'kmh'
            });
        }

        function renderWeather(location, data) {
            const current = data.current || {};
            const hourly = data.hourly || {};
            const [conditionLabel, icon] = weatherCodeToLabel(current.weather_code, current.is_day === 1);

            $('#weatherLocation').text(location.name + ', ' + location.country);
            $('#weatherCondition').text(conditionLabel + ' · Feels like ' + Math.round(current.apparent_temperature || current.temperature_2m || 0) + '°C');
            $('#weatherTemp').text(Math.round(current.temperature_2m || 0));
            $('#weatherHumidity').text(Math.round(current.relative_humidity_2m || 0) + '%');
            $('#weatherWind').text(Math.round(current.wind_speed_10m || 0) + ' km/h');
            $('#weatherRain').text(Math.round(current.precipitation_probability || 0) + '%');
            $('#weatherIcon').text(icon);
            $('#weatherSource').text('Live weather · ' + location.source);
            $('#weatherUpdated').text('Updated ' + new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }));
            setWeatherStatus('Weather live');

            const rows = [];
            const times = hourly.time || [];
            const temps = hourly.temperature_2m || [];
            const now = Date.now();

            for (let i = 0; i < times.length && rows.length < 4; i++) {
                const ts = new Date(times[i]).getTime();
                if (ts > now) {
                    const dt = new Date(times[i]);
                    rows.push(
                        '<div class="forecast-pill">' +
                        '<div class="day">' + dt.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) + '</div>' +
                        '<div class="temp">' + Math.round(temps[i]) + '°</div>' +
                        '</div>'
                    );
                }
            }

            while (rows.length < 4) {
                rows.push('<div class="forecast-pill"><div class="day">--:--</div><div class="temp">--°</div></div>');
            }

            $('#forecastRow').html(rows.join(''));
        }

        function refreshWeather() {
            setWeatherStatus('Refreshing weather');
            $('#weatherSource').text('Loading live data…');

            resolveLocation()
                .then(function (location) {
                    return fetchWeatherForLocation(location).then(function (data) {
                        renderWeather(location, data);
                    });
                })
                .fail(function () {
                    $('#weatherLocation').text('Weather unavailable');
                    $('#weatherCondition').text('Live weather request failed');
                    $('#weatherTemp').text('--');
                    $('#weatherHumidity').text('--%');
                    $('#weatherWind').text('-- km/h');
                    $('#weatherRain').text('--%');
                    $('#weatherIcon').text('⚠️');
                    $('#weatherSource').text('Weather error');
                    $('#weatherUpdated').text('Unable to refresh weather');
                    setWeatherStatus('Weather error');
                });
        }

        refreshWeather();
        setInterval(refreshWeather, WEATHER_REFRESH_MS);
    }

    // Media Player
    if ($('#xeneon-media-player').length) {
        setInterval(function () {
            $.ajax({
                url: '/api/media/playback',
                method: 'GET',
                dataType: 'json',
                success: function (response) {
                    if (response.status === 1 && response.data) {
                        updateMediaPlayer(response)
                    }
                },
                error: function () {
                    console.error('Failed to get media playback');
                }
            });
        }, 1000);

        $('.media-button').on('click', function () {
            const info = Number($(this).data('info'));
            let urlAction = '';
            switch (info) {
                case 1:
                    urlAction = '/api/media/previous';
                    break;
                case 2:
                    urlAction = '/api/media/stop';
                    break;
                case 3:
                    urlAction = '/api/media/play';
                    break;
                case 4:
                    urlAction = '/api/media/next';
                    break;
                case 5:
                    urlAction = '/api/media/volumeDown';
                    break;
                case 6:
                    urlAction = '/api/media/volumeUp';
                    break;
                case 7:
                    urlAction = '/api/media/mute';
                    break;
                default:
                    console.warn('Unknown media button:', info);
            }

            $.ajax({
                url: urlAction,
                method: 'GET',
                dataType: 'json',
                success: function (response) {
                    // No result
                },
                error: function () {
                    console.error('Failed to run media control');
                }
            });
        });
        function formatTime(seconds) {
            seconds = Math.max(0, Math.floor(Number(seconds) || 0));

            const mins = Math.floor(seconds / 60);
            const secs = seconds % 60;

            return mins + ':' + String(secs).padStart(2, '0');
        }
        function updateMediaPlayer(json) {
            if (!json || json.code !== 200 || !json.data) return;

            const data = json.data;

            const title = data.title || 'N/A';
            const artist = Array.isArray(data.artists) && data.artists.length
                ? data.artists.join(', ')
                : 'N/A';

            const position = Number(data.position) || 0;
            const length = Number(data.length) || 0;

            const progress = length > 0
                ? Math.min(100, Math.max(0, (position / length) * 100))
                : 0;

            $('#trackTitle').text(title);
            $('#tractArtist').text(artist); // your HTML id is tractArtist
            $('#position').text(formatTime(position));
            $('#length').text(formatTime(length));
            $('#trackProgress span').css('width', progress + '%');
        }
    }

    // Battery
    if ($('#xeneon-battery-status').length) {
        function updateBatteryStatus(json) {
            const $list = $('#xeneon-battery-status .metrics-list');
            $list.empty();

            $.each(json.data.battery, function (deviceId, device) {
                const name = device.Device || deviceId;
                const level = Math.max(0, Math.min(100, Number(device.Level) || 0));
                const html = `
                  <div class="metric-row" data-device-id="${deviceId}">
                    <span class="name"></span>
                    <span class="metric-value">${level}%</span>
                    <div class="metric-bar" style="--fill:${level}%"><span></span></div>
                  </div>
                `;
                const $row = $(html);
                $row.find('.name').text(name);
                $list.append($row);
            });
        }
        setInterval(function () {
            $.ajax({
                url: '/api/systray',
                method: 'GET',
                dataType: 'json',
                success: function (response) {
                    if (response.status === 1 && response.data) {
                        updateBatteryStatus(response)
                    }
                },
                error: function () {
                    console.error('Failed to get media playback');
                }
            });
        }, 1000);
    }

    // Calendar
    if ($('#xeneon-calendar').length) {
        let viewDate = new Date();

        function pad(n) {
            return String(n).padStart(2, '0');
        }

        function formatFullDate(date) {
            return date.toLocaleDateString('en-US', {
                weekday: 'long',
                month: 'long',
                day: 'numeric'
            });
        }

        function formatMonth(date) {
            return date.toLocaleDateString('en-US', {
                month: 'long',
                year: 'numeric'
            });
        }

        function isSameDate(a, b) {
            return (
                a.getFullYear() === b.getFullYear() &&
                a.getMonth() === b.getMonth() &&
                a.getDate() === b.getDate()
            );
        }

        function renderCalendar() {
            const today = new Date();

            const year = viewDate.getFullYear();
            const month = viewDate.getMonth();

            const firstDay = new Date(year, month, 1);
            const lastDay = new Date(year, month + 1, 0);

            // Monday-first calendar offset
            const startOffset = (firstDay.getDay() + 6) % 7;

            const prevMonthLastDay = new Date(year, month, 0).getDate();

            const $days = $('#calendarDays');
            $days.empty();

            $('#calendarMonth').text(formatMonth(viewDate));
            $('#calendarToday').text(formatFullDate(today));

            for (let i = 0; i < 42; i++) {
                let dayNumber;
                let cellDate;
                let muted = false;

                if (i < startOffset) {
                    dayNumber = prevMonthLastDay - startOffset + i + 1;
                    cellDate = new Date(year, month - 1, dayNumber);
                    muted = true;
                } else if (i >= startOffset + lastDay.getDate()) {
                    dayNumber = i - startOffset - lastDay.getDate() + 1;
                    cellDate = new Date(year, month + 1, dayNumber);
                    muted = true;
                } else {
                    dayNumber = i - startOffset + 1;
                    cellDate = new Date(year, month, dayNumber);
                }

                const isWeekend = cellDate.getDay() === 0 || cellDate.getDay() === 6;

                const $day = $('<div>', {
                    class: 'calendar-day',
                    text: dayNumber
                });

                if (muted) $day.addClass('is-muted');
                if (isWeekend) $day.addClass('is-weekend');
                if (isSameDate(cellDate, today)) $day.addClass('is-today');

                $days.append($day);
            }
        }

        $('#calendarPrev').on('click', function () {
            viewDate.setMonth(viewDate.getMonth() - 1);
            renderCalendar();
        });

        $('#calendarNext').on('click', function () {
            viewDate.setMonth(viewDate.getMonth() + 1);
            renderCalendar();
        });

        renderCalendar();

        // Refresh at midnight-ish so "today" updates while dashboard is open
        setInterval(function () {
            renderCalendar();
        }, 60 * 1000);
    }

    // Gauge bars
    const RADIUS = 46;
    const CIRCUMFERENCE = 2 * Math.PI * RADIUS;
    const GAUGE_PERCENT = 0.85;
    const GAUGE_LENGTH = CIRCUMFERENCE * GAUGE_PERCENT;
    const GAUGE_GAP = CIRCUMFERENCE - GAUGE_LENGTH;
    const GAUGE_ROTATION = 90 + ((1 - GAUGE_PERCENT) * 360 / 2);

    function clamp(num, min, max) {
        return Math.min(Math.max(num, min), max);
    }

    function updateRing($widget, value, maxValue, options) {
        const settings = $.extend({
            unit: $widget.data('unit') || '',
            label: $widget.data('label') || '',
            color: $widget.data('color') || '#38bdf8',
            decimals: 0
        }, options || {});

        const safeMax = Math.max(parseFloat(maxValue) || 100, 1);
        const numericValue = parseFloat(value) || 0;
        const percent = clamp((numericValue / safeMax) * 100, 0, 100);
        const dashOffset = GAUGE_LENGTH * (1 - percent / 100);
        const formattedValue = numericValue.toFixed(settings.decimals).replace(/\.0+$/, '');

        $widget.attr('data-value', numericValue);
        $widget.attr('data-max', safeMax);

        $widget.find('svg').css({
            transform: 'rotate(' + GAUGE_ROTATION + 'deg)'
        });

        $widget.find('.ring-track').css({
            strokeDasharray: GAUGE_LENGTH + ' ' + CIRCUMFERENCE,
            strokeDashoffset: 0
        });

        $widget.find('.ring-progress').css({
            stroke: settings.color,
            strokeDasharray: GAUGE_LENGTH + ' ' + CIRCUMFERENCE,
            strokeDashoffset: dashOffset
        });

        $widget.find('.chart-value').text(formattedValue + settings.unit);
        $widget.find('.chart-temp').text(formattedValue + settings.unit);
        $widget.find('.chart-load').text(formattedValue + settings.unit);
        $widget.find('.chart-label').text(settings.label);

        const isWarning = percent >= 80;
        $widget.toggleClass('is-warning', isWarning);
        $widget.find('.small-label').first().toggleClass('is-hot', isWarning);
    }

    window.setThermalValue = function (selector, value, maxValue) {
        const $widget = $(selector);
        updateRing($widget, value, maxValue || $widget.data('max'));
    };

    window.updateThermalWidget = function (selector, payload) {
        const $widget = $(selector);
        updateRing($widget, payload.value, payload.max || $widget.data('max'), payload);

        if (payload.load != null) $widget.find('.js-load').text(Math.round(payload.load) + '%');
        if (payload.fan != null) $widget.find('.js-fan').text(Math.round(payload.fan) + ' RPM');
        if (payload.limit != null) $widget.find('.js-limit').text(Math.round(payload.limit) + '°C');
        if (payload.vram != null) $widget.find('.js-vram').text(Number(payload.vram).toFixed(1) + ' GB');
        if (payload.hotspot != null) $widget.find('.js-hotspot').text(Math.round(payload.hotspot) + '°C');
    };

    // CPU Temp
    if ($('#xeneon-cpu-temp').length) {
        const $widget = $("#xeneon-cpu-temp");
        setInterval(function () {
            $.ajax({
                url: '/api/cpuTemp/clean',
                method: 'GET',
                dataType: 'json',
                success: function (response) {
                    if (response.status === 1 && response.data) {
                        updateRing($widget, response.data, $widget.data('max'));
                        setThermalValue($widget, response.data, $widget.data('max'));
                    }
                },
                error: function () {
                    console.error('Failed to get cpu temperature');
                }
            });
        }, 1000);
    }

    // CPU Load
    if ($('#xeneon-cpu-load').length) {
        const $widget = $("#xeneon-cpu-load");
        setInterval(function () {
            $.ajax({
                url: '/api/cpuLoad',
                method: 'GET',
                dataType: 'json',
                success: function (response) {
                    if (response.status === 1 && response.data) {
                        updateRing($widget, response.data, $widget.data('max'));
                        setThermalValue($widget, response.data, $widget.data('max'));
                    }
                },
                error: function () {
                    console.error('Failed to get cpu load');
                }
            });
        }, 1000);
    }

    // GPU Temp
    if ($('#xeneon-gpu-temp').length) {
        const $widget = $("#xeneon-gpu-temp");
        setInterval(function () {
            $.ajax({
                url: '/api/gpuTemp/clean',
                method: 'GET',
                dataType: 'json',
                success: function (response) {
                    if (response.status === 1 && response.data) {
                        updateRing($widget, response.data, $widget.data('max'));
                        setThermalValue($widget, response.data, $widget.data('max'));
                    }
                },
                error: function () {
                    console.error('Failed to get gpu temperature');
                }
            });
        }, 1000);
    }

    // GPU Load
    if ($('#xeneon-gpu-load').length) {
        const $widget = $("#xeneon-gpu-load");
        setInterval(function () {
            $.ajax({
                url: '/api/gpuLoad',
                method: 'GET',
                dataType: 'json',
                success: function (response) {
                    if (response.status === 1 && response.data) {
                        updateRing($widget, response.data, $widget.data('max'));
                        setThermalValue($widget, response.data, $widget.data('max'));
                    }
                },
                error: function () {
                    console.error('Failed to get gpu load');
                }
            });
        }, 1000);
    }
});