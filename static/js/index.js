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
                url:'/api/devices',
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


    $('#app-settings').on('click', function () {
        let modalElement = '<div class="modal fade text-start" id="appSettingsModal" tabindex="-1" aria-labelledby="appSettingsLabel" aria-hidden="true">';
        modalElement+='<div class="modal-dialog">';
        modalElement+='<div class="modal-content">';
        modalElement+='<div class="modal-header">';
        modalElement+='<h5 class="modal-title" id="appSettingsLabel">Settings</h5>';
        modalElement+='<button class="btn-close btn-close-white" type="button" data-bs-dismiss="modal" aria-label="Close"></button>';
        modalElement+='</div>';
        modalElement+='<div class="modal-body">';
        modalElement+='<form>';

        modalElement+='<div class="mb-3">';

        modalElement+='<div class="form-check" style="overflow: hidden;padding-left: 1em;">';
        modalElement+='<div style="float:left;">';
        modalElement+='<label class="form-check-label" for="checkbox-cpu">Show CPU in Dashboard</label>';
        modalElement+='</div>';
        modalElement+='<div style="float:right;">';
        modalElement+='<input class="form-check-input" id="checkbox-cpu" type="checkbox">';
        modalElement+='</div>';
        modalElement+='</div>';

        modalElement+='<div class="form-check" style="overflow: hidden;padding-left: 1em;">';
        modalElement+='<div style="float:left;">';
        modalElement+='<label class="form-check-label" for="checkbox-gpu">Show GPU in Dashboard</label>';
        modalElement+='</div>';
        modalElement+='<div style="float:right;">';
        modalElement+='<input class="form-check-input" id="checkbox-gpu" type="checkbox">';
        modalElement+='</div>';
        modalElement+='</div>';

        modalElement+='<div class="form-check" style="overflow: hidden;padding-left: 1em;">';
        modalElement+='<div style="float:left;">';
        modalElement+='<label class="form-check-label" for="checkbox-storage">Show Storage in Dashboard</label>';
        modalElement+='</div>';
        modalElement+='<div style="float:right;">';
        modalElement+='<input class="form-check-input" id="checkbox-storage" type="checkbox">';
        modalElement+='</div>';
        modalElement+='</div>';

        modalElement+='<div class="form-check" style="overflow: hidden;padding-left: 1em;">';
        modalElement+='<div style="float:left;">';
        modalElement+='<label class="form-check-label" for="checkbox-devices">Show Devices in Dashboard</label>';
        modalElement+='</div>';
        modalElement+='<div style="float:right;">';
        modalElement+='<input class="form-check-input" id="checkbox-devices" type="checkbox">';
        modalElement+='</div>';
        modalElement+='</div>';

        modalElement+='<div class="form-check" style="overflow: hidden;padding-left: 1em;">';
        modalElement+='<div style="float:left;">';
        modalElement+='<label class="form-check-label" for="checkbox-celsius">Show Temperature in Celsius</label>';
        modalElement+='</div>';
        modalElement+='<div style="float:right;">';
        modalElement+='<input class="form-check-input" id="checkbox-celsius" type="checkbox">';
        modalElement+='</div>';
        modalElement+='</div>';

        modalElement+='<div class="form-check" style="overflow: hidden;padding-left: 1em;">';
        modalElement+='<div style="float:left;">';
        modalElement+='<label class="form-check-label" for="checkbox-deviceLabels">Show Device Labels</label>';
        modalElement+='</div>';
        modalElement+='<div style="float:right;">';
        modalElement+='<input class="form-check-input" id="checkbox-deviceLabels" type="checkbox">';
        modalElement+='</div>';
        modalElement+='</div>';

        modalElement+='</div>';

        modalElement+='</form>';
        modalElement+='</div>';
        modalElement+='<div class="modal-footer">';
        modalElement+='<button class="btn btn-secondary" type="button" data-bs-dismiss="modal">Close</button>';
        modalElement+='<button class="btn btn-primary" type="button" id="btnSaveSettings">Save</button>';
        modalElement+='</div>';
        modalElement+='</div>';
        modalElement+='</div>';
        modalElement+='</div>';
        const modal = $(modalElement).modal('toggle');

        modal.on('hidden.bs.modal', function () {
            modal.data('bs.modal', null);
        })

        modal.on('shown.bs.modal', function (e) {
            const userProfileName = modal.find('#userProfileName');
            userProfileName.focus();

            const checkboxCpu = modal.find('#checkbox-cpu');
            const checkboxGpu = modal.find('#checkbox-gpu');
            const checkboxStorage = modal.find('#checkbox-storage');
            const checkboxDevices = modal.find('#checkbox-devices');
            const checkboxDeviceLabels = modal.find('#checkbox-deviceLabels');
            const checkboxCelsius = modal.find('#checkbox-celsius');

            // Load current settings
            $.ajax({
                url: '/api/dashboard',
                type: 'GET',
                cache: false,
                success: function(response) {
                    if (response.status === 1) {
                        if (response.dashboard.showCpu === true) {
                            checkboxCpu.attr('Checked','Checked');
                        }
                        if (response.dashboard.showGpu === true) {
                            checkboxGpu.attr('Checked','Checked');
                        }
                        if (response.dashboard.showDisk === true) {
                            checkboxStorage.attr('Checked','Checked');
                        }
                        if (response.dashboard.showDevices === true) {
                            checkboxDevices.attr('Checked','Checked');
                        }
                        if (response.dashboard.showLabels === true) {
                            checkboxDeviceLabels.attr('Checked','Checked');
                        }
                        if (response.dashboard.celsius === true) {
                            checkboxCelsius.attr('Checked','Checked');
                        }
                    }
                }
            });

            modal.find('#btnSaveSettings').on('click', function () {
                const v_checkboxCpu = checkboxCpu.is(':checked');
                const v_checkboxGpu = checkboxGpu.is(':checked');
                const v_checkboxStorage = checkboxStorage.is(':checked');
                const v_checkboxDevices = checkboxDevices.is(':checked');
                const v_checkboxDeviceLabels = checkboxDeviceLabels.is(':checked');
                const v_checkboxCelsius = checkboxCelsius.is(':checked');

                const pf = {};
                pf["showCpu"] = v_checkboxCpu;
                pf["showGpu"] = v_checkboxGpu;
                pf["showDisk"] = v_checkboxStorage;
                pf["showDevices"] = v_checkboxDevices;
                pf["showLabels"] = v_checkboxDeviceLabels;
                pf["celsius"] = v_checkboxCelsius;

                const json = JSON.stringify(pf, null, 2);

                $.ajax({
                    url: '/api/dashboard',
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
        })
    });
});