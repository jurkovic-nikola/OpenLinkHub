"use strict";
$(document).ready(function () {
    // Init dataTable
    const dt = $('#dataTable').DataTable(
        {
            order: [[0, 'asc']],
            select: {
                style: 'os',
                selector: 'td:first-child'
            },
            paging: true,
            searching: true,
            language: {
                emptyTable: "Supported device list",
                searchPlaceholder: "Search for device..."
            },
            layout: {
                topStart: null,
                topEnd: 'search',
                bottomStart: ['paging', 'info'],
                bottomEnd: 'pageLength'
            },
            columns: [
                { data: 'ProductId', title: 'Product Id' },
                {
                    data: 'ProductId',
                    title: 'Product Id - Hexadecimal',
                    render: function(data, type, row, meta) {
                        return '0x' + Number(data).toString(16).toUpperCase().padStart(4, '0');
                    }
                },
                { data: 'Name', title: 'Product Name' }, // JSON uses Name
                {
                    data: 'Enabled',
                    title: 'Enabled',
                    orderable: false,
                    render: function(data, type, row, meta) {
                        const checked = data ? 'checked' : '';
                        return `
                            <label class="system-toggle compact">
                                <input type="checkbox" class="device-checkbox" data-id="${row.ProductId}" ${checked}>
                                <span class="toggle-track"></span>
                            </label>
                        `;
                    }
                }
            ]
        }
    );

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

    $("#btnBackup").on("click", function() {
        window.location.href = "/api/backup";
    });

    $('.saveRgbControl').on('click', function () {
        const rgbControl = $("#rgbControl").is(':checked');
        const rgbOff = $("#rgbOff").val();
        const rgbOn = $("#rgbOn").val();

        const pf = {};
        pf["rgbControl"] = rgbControl;
        pf["rgbOff"] = rgbOff;
        pf["rgbOn"] = rgbOn;

        const json = JSON.stringify(pf, null, 2);
        $.ajax({
            url: '/api/scheduler/rgb',
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
    
    $("#restoreForm").on("submit", function (e) {
        e.preventDefault();

        var formData = new FormData();
        var file = $("#backupFile")[0].files[0];
        if (!file) {
            toast.warning('Please select a .zip file first!');
            return;
        }
        formData.append("backupFile", file);

        $.ajax({
            url: "/api/restore",
            type: "POST",
            data: formData,
            processData: false,
            contentType: false,
            success: function (response) {
                toast.success(response);
            },
            error: function (xhr) {
                toast.warning("Restore failed: " + xhr.responseText);
            }
        });
    });

    $.ajax({
        url: '/api/getSupportedDevices',
        dataType: 'JSON',
        success: function(response) {
            if (response.code === 0) {
                toast.warning(response.message);
            } else {
                dt.clear();
                dt.rows.add(response.data);
                dt.draw();
            }
        }
    });

    dt.on('change', '.device-checkbox', function() {
        const productId = $(this).data('id');
        const enabled = $(this).prop('checked');

        // Optional: store in DataTables row data if needed
        const row = dt.row($(this).closest('tr'));
        const rowData = row.data();
        rowData.Enabled = enabled;
        row.data(rowData); // update row
    });

    $('#btnSaveSupportedDevices').on('click', function() {
        const supportedDevices = {};
        const pf = {};
        dt.rows().every(function() {
            const data = this.data();
            supportedDevices[data.ProductId] = data.Enabled; // true/false
        });
        pf["supportedDevices"] = supportedDevices;
        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/setSupportedDevices',
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

    $('.enableVirtualAudio').on('click', function () {
        const v_virtualAudio = $("#virtualAudio").is(':checked');

        const pf = {};
        pf["enabled"] = v_virtualAudio;
        const json = JSON.stringify(pf, null, 2);
        $.ajax({
            url: '/api/audio/update',
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

    $('.setTargetDevice').on('click', function () {
        const outputDevice = $("#outputDevice").val();
        const data = outputDevice.split(";");

        if (data.length < 2) {
            toast.warning('Invalid target device');
            return false;
        }

        const deviceSerial = parseInt(data[2]);
        const deviceDesc = data[1];
        const deviceName = data[0];
        
        const pf = {};
        pf["outputDeviceSerial"] = deviceSerial;
        pf["outputDeviceName"] = deviceName;
        pf["outputDeviceDesc"] = deviceDesc;

        const json = JSON.stringify(pf, null, 2);
        $.ajax({
            url: '/api/audio/outputDevice',
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

    const checkboxCpu = $('#checkbox-cpu');
    const checkboxGpu = $('#checkbox-gpu');
    const checkboxStorage = $('#checkbox-storage');
    const checkboxDevices = $('#checkbox-devices');
    const checkboxDeviceLabels = $('#checkbox-deviceLabels');
    const checkboxCelsius = $('#checkbox-celsius');
    const checkboxBattery = $('#checkbox-battery');
    const checkboxTemperatureBar = $('#checkbox-temperatureBar');

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
                    if (response.dashboard.temperatureBar === true) {
                        checkboxTemperatureBar.attr('Checked','Checked');
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
            const v_checkboxTemperatureBar = checkboxTemperatureBar.is(':checked');
            const v_languageCode = $("#userLanguage").val();
            const v_theme = $("#theme").val();

            console.log(v_languageCode);

            const pf = {};
            pf["showCpu"] = v_checkboxCpu;
            pf["showGpu"] = v_checkboxGpu;
            pf["showDisk"] = v_checkboxStorage;
            pf["showDevices"] = v_checkboxDevices;
            pf["showLabels"] = v_checkboxDeviceLabels;
            pf["celsius"] = v_checkboxCelsius;
            pf["showBattery"] = v_checkboxBattery;
            pf["temperatureBar"] = v_checkboxTemperatureBar;
            pf["languageCode"] = v_languageCode;
            pf["theme"] = v_theme;

            const json = JSON.stringify(pf, null, 2);

            console.log(json)
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