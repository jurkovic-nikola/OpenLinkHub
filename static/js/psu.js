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
            const deviceId = $("#deviceId").val()
            $.ajax({
                url:'/api/devices/' + deviceId,
                type:'get',
                success:function(result){
                    if (result.device.devices == null) {
                        // Single device, e.g CPU block
                        const elementTemperatureId = "#temperature-0";
                        $(elementTemperatureId).html(result.device.TemperatureString);
                    } else {
                        const length = Object.keys(result.device.devices).length;
                        if (length > 0) {
                            $.each(result.device.devices, function( index, value ) {
                                const elementSpeedId = "#speed-" + value.deviceId;
                                const elementTemperatureId = "#temperature-" + value.deviceId;
                                const elementWatts = "#watts-" + value.deviceId;
                                const elementAmps = "#amps-" + value.deviceId;
                                const elementVolts = "#volts-" + value.deviceId;
                                $(elementSpeedId).html(value.rpm + " RPM");
                                $(elementTemperatureId).html(value.temperatureString);
                                $(elementWatts).html(value.watts + " W");
                                $(elementAmps).html(value.amps + " A");
                                $(elementVolts).html(value.volts + " V");
                            });
                        }
                    }
                }
            });
        },1500);
    }

    autoRefresh();

    $('.fanProfile').on('change', function () {
        const deviceId = $("#deviceId").val();
        const fanMode = $(this).val();

        const pf = {};
        pf["deviceId"] = deviceId;
        pf["fanMode"] = parseInt(fanMode);

        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/psu/speed',
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