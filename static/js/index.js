"use strict";
$(document).ready(function () {
    let showLabels = false;

    function loadDevices() {
        const devicePlaceholder = $(".device-placeholder");
        devicePlaceholder.removeClass("ready").empty();

        const devicePlaceholder2 = $("#system-cards-add");
        devicePlaceholder2.removeClass("ready");

        $.ajax({
            url: '/api/dashboard/devices/get',
            type: 'GET',
            success: function (response) {
                if (response.status !== 1) return;

                const results = new Array(response.devices.length);
                let completed = 0;
                const total = response.devices.length;

                $.each(response.devices, function (index, value) {
                    $.ajax({
                        url: '/api/devices/' + value,
                        type: 'GET',
                        success: function (dev) {
                            if (dev.device) {
                                results[index] = renderDevice(dev);
                            }
                            completed++;

                            if (completed === total) {
                                devicePlaceholder.append(
                                    results.filter(Boolean).join("")
                                );
                                devicePlaceholder.addClass("ready");
                            }
                        }
                    });
                });
                devicePlaceholder2.addClass("ready");
            }
        });
    }

    function renderDevice(dev) {
        let html = `<div class="row g-4 mb-4 align-items-start">`;
        const label = showLabels && dev.device.DeviceProfile?.Label
            ? dev.device.DeviceProfile.Label
            : "";

        // Single device
        if (dev.device.devices === null) {
            if (dev.device.HasLCD) {
                html += `
                <div class="col-md-2">
                    <div class="card system-card">
                        <div class="card-header header-split">
                            <span class="header-left">${dev.device.product}</span>
                            <span class="header-right">${label}</span>
                        </div>
                        <div class="card-body">
                            <div class="settings-list">
                `;

                if (dev.device.Temperature > 0) {
                    let tempString = "Temperature";
                    if (dev.device.AIO || dev.device.IsCpuBlock) {
                        tempString = "Liquid";
                    }
                    html += `
                                <div class="settings-row">
                                    <span class="settings-label text-ellipsis">${tempString}</span>
                                    <span class="meta-value" id="temperature-0">${dev.device.temperatureString}</span>
                                </div>
                            `;
                }
            }
            html += `
                            </div>
                        </div>
                    </div>
                </div>
            `;
        } else if (dev.device.IsPSU) {
            $.each(dev.device.devices, function (_, device) {
                if (device.IsTemperatureProbe || device.HasSpeed || device.Output) {
                    return
                }

                const label = showLabels && device?.label
                    ? device?.label
                    : "";

                html += `
                <div class="col-md-2">
                    <div class="card system-card">
                        <div class="card-header header-split">
                            <span class="header-left">${device.name}</span>
                            <span class="header-right">${label}</span>
                        </div>
                        <div class="card-body">
                            <div class="settings-list">
                `;

                if (device.MainPSU) {
                    html += `
                                <div class="settings-row">
                                    <span class="settings-label text-ellipsis">Speed</span>
                                    <span class="meta-value" id="speed-${dev.device.serial}-${device.channelId}">${device.rpm} RPM</span>
                                </div>
                                <div class="settings-row">
                                    <span class="settings-label text-ellipsis">VRM Temperature</span>
                                    <span class="meta-value" id="vrm-temp-${dev.device.serial}-${device.channelId}">${device.vrmTemperatureString}</span>
                                </div>
                                <div class="settings-row">
                                    <span class="settings-label text-ellipsis">PSU Temperature</span>
                                    <span class="meta-value" id="psu-temp-${dev.device.serial}-${device.channelId}">${device.psuTemperatureString}</span>
                                </div>
                            `;
                }


                if (device.HasWatts) {
                    html += `
                                <div class="settings-row">
                                    <span class="settings-label text-ellipsis">Watts</span>
                                    <span class="meta-value" id="watts-${dev.device.serial}-${device.channelId}">${device.watts} W</span>
                                </div>
                            `;
                }

                if (device.HasAmps) {
                    html += `
                                <div class="settings-row">
                                    <span class="settings-label text-ellipsis">Amps</span>
                                    <span class="meta-value" id="amps-${dev.device.serial}-${device.channelId}">${device.amps} A</span>
                                </div>
                            `;
                }

                if (device.HasVolts) {
                    html += `
                                <div class="settings-row">
                                    <span class="settings-label text-ellipsis">Volts</span>
                                    <span class="meta-value" id="volts-${dev.device.serial}-${device.channelId}">${device.volts} V</span>
                                </div>
                            `;
                }

                html += `
                            </div>
                        </div>
                    </div>
                </div>
            `;
            });
        } else {
            $.each(dev.device.devices, function (_, device) {
                let cssClass = "col-md-2";
                if (device.volts) {
                    cssClass = "col-md-3";
                }
                const label = showLabels && device?.label
                    ? device?.label
                    : "";

                html += `
                <div class="${cssClass}">
                    <div class="card system-card">
                        <div class="card-header header-split">
                            <span class="header-left">${device.name}</span>
                            <span class="header-right">${label}</span>
                        </div>
                        <div class="card-body">
                            <div class="settings-list">
                `;

                if (device.temperature > 0) {
                    let tempString = "Temperature";
                    if (device.AIO || device.IsCpuBlock) {
                        tempString = "Liquid";
                    }
                    html += `
                                <div class="settings-row">
                                    <span class="settings-label text-ellipsis">${tempString}</span>
                                    <span class="meta-value" id="temp-${dev.device.serial}-${device.channelId}">${device.temperatureString}</span>
                                </div>
                            `;
                }

                if (device.HasSpeed) {
                    html += `
                                <div class="settings-row">
                                    <span class="settings-label text-ellipsis">Speed</span>
                                    <span class="meta-value" id="speed-${dev.device.serial}-${device.channelId}">${device.rpm} RPM</span>
                                </div>
                            `;
                }

                if (device.speed > 0) {
                    html += `
                                <div class="settings-row">
                                    <span class="settings-label text-ellipsis">Speed</span>
                                    <span class="meta-value">${device.speed} MHz</span>
                                </div>
                            `;
                }

                if (device.size > 0) {
                    html += `
                                <div class="settings-row">
                                    <span class="settings-label text-ellipsis">Size</span>
                                    <span class="meta-value">${device.size} GZ</span>
                                </div>
                            `;
                }

                if (device.HasWatts) {
                    html += `
                                <div class="settings-row">
                                    <span class="settings-label text-ellipsis">Watts</span>
                                    <span class="meta-value" id="watts-${dev.device.serial}-${device.channelId}">${device.watts} W</span>
                                </div>
                            `;
                }

                if (device.HasAmps) {
                    html += `
                                <div class="settings-row">
                                    <span class="settings-label text-ellipsis">Amps</span>
                                    <span class="meta-value" id="amps-${dev.device.serial}-${device.channelId}">${device.amps} A</span>
                                </div>
                            `;
                }

                if (device.HasVolts) {
                    html += `
                                <div class="settings-row">
                                    <span class="settings-label text-ellipsis">Volts</span>
                                    <span class="meta-value" id="volts-${dev.device.serial}-${device.channelId}">${device.volts} V</span>
                                </div>
                            `;
                }

                if (device.volts) {
                    html += `
                                <div class="settings-row settings-row-equal">
                                    <span class="settings-label text-ellipsis">Output</span>
                                    <span class="meta-value text-right" id="powerOut-${device.channelId}">${device.powerOutString} W</span>
                                </div>
                        `;

                    $.each(device.volts, function (key, rail) {
                        const amps = device.amps[key];
                        const watts = device.watts[key];

                        let railName = "";
                        switch (parseInt(key)) {
                            case 0:
                                railName = "3.3V Rail"
                                break
                            case 1:
                                railName = "5V Rail"
                                break
                            case 2:
                                railName = "12V Rail"
                                break
                        }
                        html += `
                                <div class="settings-row settings-row-equal">
                                    <span class="settings-label text-ellipsis">${railName}</span>
                                    <span class="meta-value text-right" id="volts-${device.channelId}-${key}">${rail.ValueString} V</span>
                                    <span class="meta-value text-right" id="amps-${device.channelId}-${key}">${amps.ValueString} A</span>
                                    <span class="meta-value text-right" id="watts-${device.channelId}-${key}">${watts.ValueString} W</span>
                                </div>
                        `;
                    });
                }

                html += `
                            </div>
                        </div>
                    </div>
                </div>
            `;
            });
        }

        html += `</div>`;
        return html;
    }

    function autoRefresh() {
        setInterval(function(){
            $.ajax({
                url:'/api/devices/',
                type:'get',
                success:function(result){
                    $.each(result.devices, function( index, value ) {
                        const serialId = value.Serial
                        if (value.GetDevice != null) {
                            if (value.GetDevice.devices == null) {
                                // Single device, e.g CPU block
                                const elementTemperatureId = "#temperature-0";
                                $(elementTemperatureId).html(value.GetDevice.temperatureString);
                            } else {
                                $.each(value.GetDevice.devices, function( key, device ) {
                                    const elementSpeedId = "#speed-" + serialId + "-" + device.channelId;
                                    const elementTemperatureId = "#temp-" + serialId + "-" + device.channelId;
                                    const elementVrmTemperatureId = "#vrm-temp-" + serialId + "-" + device.channelId;
                                    const elementPsuTemperatureId = "#psu-temp-" + serialId + "-" + device.channelId;
                                    const elementWatts = "#watts-" + serialId + "-" + device.channelId;
                                    const elementAmps = "#amps-" + serialId + "-" + device.channelId;
                                    const elementVolts = "#volts-" + serialId + "-" + device.channelId;

                                    $(elementWatts).html(device.watts + " W");
                                    $(elementAmps).html(device.amps + " A");
                                    $(elementVolts).html(device.volts + " V");
                                    $(elementSpeedId).html(device.rpm + " RPM");

                                    $(elementTemperatureId).html(device.temperatureString);
                                    $(elementVrmTemperatureId).html(device.vrmTemperatureString);
                                    $(elementPsuTemperatureId).html(device.psuTemperatureString);

                                    if (device.IsPSU) {
                                        const elementPowerOut = "#powerOut-" + device.channelId;
                                        if (elementPowerOut != null) {
                                            $(elementPowerOut).html(device.powerOutString + " W");
                                        }
                                    }

                                    if (device.volts) {
                                        $.each(device.volts, function( index, value ) {
                                            const amps = device.amps[index];
                                            const watts = device.watts[index];

                                            const elementVolts = "#volts-" + device.channelId + "-" + index;
                                            if (elementVolts != null) {
                                                $(elementVolts).html(value.ValueString + " V");
                                            }

                                            const elementAmps = "#amps-" + device.channelId + "-" + index;
                                            if (elementAmps != null) {
                                                $(elementAmps).html(amps.ValueString + " A");
                                            }

                                            const elementWatts = "#watts-" + device.channelId + "-" + index;
                                            if (elementWatts != null) {
                                                $(elementWatts).html(watts.ValueString + " W");
                                            }
                                        });
                                    }
                                });
                            }
                        }
                    });
                }
            });
        },3000);
    }
    autoRefresh();

    $('.allDevicesRgb').on('change', function () {
        const profile = $(this).val();
        if (profile === "none") {
            return false;
        }
        
        const pf = {
            "profile": profile
        };

        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/color/global',
            type: 'POST',
            data: json,
            cache: false,
            success: function(response) {
                try {
                    if (response.status === 1) {
                        toast.success(response.message);
                    } else {
                        toast.warning(response.message);
                    }
                } catch (err) {
                    toast.warning(response.message);
                }
            }
        });
    });

    $('.addDeviceToDashboard').on('click', function () {
        const deviceId = $("#dashboardDeviceSelect").val();
        const pf = {};
        pf["deviceId"] = deviceId;
        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/dashboard/devices/add',
            type: 'POST',
            data: json,
            cache: false,
            success: function(response) {
                try {
                    if (response.status === 1) {
                        loadDevices()
                    } else {
                        toast.warning(response.message);
                    }
                } catch (err) {
                    toast.warning(response.message);
                }
            }
        });
    });

    $('.deleteDeviceFromDashboard').on('click', function () {
        const deviceId = $("#dashboardDeviceSelect").val();
        const pf = {};
        pf["deviceId"] = deviceId;
        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/dashboard/devices/delete',
            type: 'DELETE',
            data: json,
            cache: false,
            success: function(response) {
                try {
                    if (response.status === 1) {
                        loadDevices()
                    } else {
                        toast.warning(response.message);
                    }
                } catch (err) {
                    toast.warning(response.message);
                }
            }
        });
    });

    function loadDashboardSettings() {
        // Load current settings
        $.ajax({
            url: '/api/dashboard',
            type: 'GET',
            cache: false,
            success: function (response) {
                if (response.status === 1) {
                    showLabels = response.dashboard.showLabels;
                }
            }
        });
    }

    loadDashboardSettings();
    loadDevices();
});