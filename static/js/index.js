"use strict";

document.addEventListener("DOMContentLoaded", function () {
    function CreateToastr() {
        toastr.options = {
            "closeButton": true,
            "debug": false,
            "newestOnTop": false,
            "progressBar": true,
            "positionClass": "toast-top-right",
            "preventDuplicates": true,
            "onclick": null,
            "showDuration": 300,
            "hideDuration": 1000,
            "timeOut": 7000,
            "extendedTimeout": "1000",
            "showEasing": "swing",
            "hideEasing": "linear",
            "showMethod": "fadeIn",
            "hideMethod": "fadeOut",
        }
        return toastr
    }

    // Init toastr
    const toast = CreateToastr();

    function autoRefresh() {
        setInterval(function(){
            $.ajax({
                url:'/api/cpuTemp',
                type:'get',
                success:function(result){
                    $("#cpu_temp").html(result.data);
                }
            });
            $.ajax({
                url:'/api/gpuTemps',
                type:'get',
                success:function(result){
                    $.each(result.data, function( index, value ) {
                        $("#gpu_temp_" + index).html(value);
                    });
                }
            });

            $.ajax({
                url:'/api/storageTemp',
                type:'get',
                success:function(result){
                    $.each(result.data, function( index, value ) {
                        $("#storage_temp-" + value.Key).html(value.TemperatureString);
                    });
                }
            });

            $.ajax({
                url:'/api/batteryStats',
                type:'get',
                success:function(result){
                    $.each(result.data, function( index, value ) {
                        $("#battery_level-" + index).html(value.Level + " %");
                    });
                }
            });

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
                                    const elementWatts = "#watts-" + serialId + "-" + device.channelId;
                                    const elementAmps = "#amps-" + serialId + "-" + device.channelId;
                                    const elementVolts = "#volts-" + serialId + "-" + device.channelId;

                                    if (device.IsPSU) {
                                        const powerOut = "#powerOut-" + device.channelId;
                                        $(powerOut).html(device.powerOut + " W");

                                        $.each(device.volts, function( index, value ) {
                                            const elementVolts = "#psuVolts-" + index;
                                            if (elementVolts != null) {
                                                $(elementVolts).html(value.ValueString + " V");
                                            }
                                        });

                                        $.each(device.amps, function( index, value ) {
                                            const elementAmps = "#psuAmps-" + index;
                                            if (elementAmps != null) {
                                                $(elementAmps).html(value.ValueString + " A");
                                            }
                                        });

                                        $.each(device.watts, function( index, value ) {
                                            const elementWatts = "#psuWatts-" + index;
                                            if (elementWatts != null) {
                                                $(elementWatts).html(value.ValueString + " W");
                                            }
                                        });

                                        console.log(device)
                                    }
                                    $(elementSpeedId).html(device.rpm + " RPM");
                                    $(elementTemperatureId).html(device.temperatureString);
                                    $(elementWatts).html(device.watts + " W");
                                    $(elementAmps).html(device.amps + " A");
                                    $(elementVolts).html(device.volts + " V");
                                });
                            }
                        }
                    });
                }
            });
        },3000);
    }
    autoRefresh();

    $('.moveDown').on('click', function () {
        const deviceId = $(this).data('info');
        const pf = {};
        pf["deviceId"] = deviceId;
        pf["position"] = 1;
        const json = JSON.stringify(pf, null, 2);
        $.ajax({
            url: '/api/dashboard/position',
            type: 'POST',
            data: json,
            cache: false,
            success: function(response) {
                try {
                    if (response.status === 1) {
                        location.reload();
                    } else {
                        toast.warning(response.message);
                    }
                } catch (err) {
                    toast.warning(response.message);
                }
            }
        });
    });

    $('.moveUp').on('click', function () {
        const deviceId = $(this).data('info');
        const pf = {};
        pf["deviceId"] = deviceId;
        pf["position"] = 0;
        const json = JSON.stringify(pf, null, 2);
        $.ajax({
            url: '/api/dashboard/position',
            type: 'POST',
            data: json,
            cache: false,
            success: function(response) {
                try {
                    if (response.status === 1) {
                        location.reload();
                    } else {
                        toast.warning(response.message);
                    }
                } catch (err) {
                    toast.warning(response.message);
                }
            }
        });
    });

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
});