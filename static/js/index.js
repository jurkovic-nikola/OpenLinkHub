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
                url:'/api/gpuTemp',
                type:'get',
                success:function(result){
                    $("#gpu_temp").html(result.data);
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
});