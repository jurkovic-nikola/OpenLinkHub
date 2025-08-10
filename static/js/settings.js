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

    const checkboxCpu = $('#checkbox-cpu');
    const checkboxGpu = $('#checkbox-gpu');
    const checkboxStorage = $('#checkbox-storage');
    const checkboxDevices = $('#checkbox-devices');
    const checkboxDeviceLabels = $('#checkbox-deviceLabels');
    const checkboxCelsius = $('#checkbox-celsius');
    const checkboxBattery = $('#checkbox-battery');

    function loadDashboardSettings() {
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
                    if (response.dashboard.showBattery === true) {
                        checkboxBattery.attr('Checked','Checked');
                    }
                }
            }
        });

        $('#btnSaveDashboardSettings').on('click', function () {
            const v_checkboxCpu = checkboxCpu.is(':checked');
            const v_checkboxGpu = checkboxGpu.is(':checked');
            const v_checkboxStorage = checkboxStorage.is(':checked');
            const v_checkboxDevices = checkboxDevices.is(':checked');
            const v_checkboxDeviceLabels = checkboxDeviceLabels.is(':checked');
            const v_checkboxCelsius = checkboxCelsius.is(':checked');
            const v_checkboxBattery = checkboxBattery.is(':checked');
            const v_languageCode = $("#userLanguage").val();

            console.log(v_languageCode);

            const pf = {};
            pf["showCpu"] = v_checkboxCpu;
            pf["showGpu"] = v_checkboxGpu;
            pf["showDisk"] = v_checkboxStorage;
            pf["showDevices"] = v_checkboxDevices;
            pf["showLabels"] = v_checkboxDeviceLabels;
            pf["celsius"] = v_checkboxCelsius;
            pf["showBattery"] = v_checkboxBattery;
            pf["languageCode"] = v_languageCode;

            const json = JSON.stringify(pf, null, 2);

            $.ajax({
                url: '/api/dashboard/update',
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
    }
    loadDashboardSettings();
});