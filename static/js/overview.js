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

    $('.newLabel').on('click', function () {
        const channelId = $(this).children('.deviceData').val();
        const valueOut = $(this).children('.labelValue');
        let modalElement = '<div class="modal fade text-start" id="newLabelModal" tabindex="-1" aria-labelledby="newLabelModalLabel" aria-hidden="true">';
        modalElement+='<div class="modal-dialog">';
        modalElement+='<div class="modal-content">';
        modalElement+='<div class="modal-header">';
        modalElement+='<h5 class="modal-title" id="newLabelModalLabel">Set device label</h5>';
        modalElement+='<button class="btn-close btn-close-white" type="button" data-bs-dismiss="modal" aria-label="Close"></button>';
        modalElement+='</div>';
        modalElement+='<div class="modal-body">';
        modalElement+='<form>';
        modalElement+='<div class="mb-3">';
        modalElement+='<label class="form-label" for="labelName">Name</label>';
        modalElement+='<input class="form-control" id="labelName" type="text">';
        modalElement+='</div>';
        modalElement+='</form>';
        modalElement+='</div>';
        modalElement+='<div class="modal-footer">';
        modalElement+='<button class="btn btn-secondary" type="button" data-bs-dismiss="modal">Close</button>';
        modalElement+='<button class="btn btn-primary" type="button" id="btnSaveLabel">Save</button>';
        modalElement+='</div>';
        modalElement+='</div>';
        modalElement+='</div>';
        modalElement+='</div>';
        const modal = $(modalElement).modal('toggle');

        modal.on('hidden.bs.modal', function () {
            modal.data('bs.modal', null);
        })

        modal.on('shown.bs.modal', function (e) {
            const labelName = modal.find('#labelName');
            labelName.focus();
            labelName.val(valueOut.text());

            modal.find('#btnSaveLabel').on('click', function () {
                const labelValue = labelName.val();
                if (labelValue.length < 1) {
                    toast.warning('Device label can not be empty');
                    return false
                }
                const deviceId = $("#deviceId").val();

                const pf = {};
                pf["deviceId"] = deviceId;
                pf["channelId"] = parseInt(channelId);
                pf["label"] = labelValue;
                const json = JSON.stringify(pf, null, 2);

                $.ajax({
                    url: '/api/label',
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

    function autoRefresh() {
        setInterval(function(){
            const deviceId = $("#deviceId").val()
            $.ajax({
                url:'/api/devices/' + deviceId,
                type:'get',
                success:function(result){
                    $.each(result.device.devices, function( index, value ) {
                        const elementSpeedId = "#speed-" + value.deviceId;
                        const elementTemperatureId = "#temperature-" + value.deviceId;
                        $(elementSpeedId).html(value.rpm);
                        $(elementTemperatureId).html(value.temperature);
                    });
                }
            });
        },1500);
    }

    autoRefresh();

    $('.tempProfile').on('change', function () {
        const deviceId = $("#deviceId").val();
        const profile = $(this).val().split(";");
        if (profile.length < 2 || profile.length > 2) {
            toast.warning('Invalid profile selected');
            return false;
        }

        const pf = {};
        pf["deviceId"] = deviceId;
        pf["channelId"] = parseInt(profile[0]);
        pf["profile"] = profile[1];

        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/speed',
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

    $('.rgbProfile').on('change', function () {
        const deviceId = $("#deviceId").val();
        const profile = $(this).val().split(";");
        if (profile.length < 2 || profile.length > 2) {
            toast.warning('Invalid profile selected');
            return false;
        }

        const pf = {};
        pf["deviceId"] = deviceId;
        pf["channelId"] = parseInt(profile[0]);
        pf["profile"] = profile[1];

        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/color',
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

    $('#deviceSpeed').on('change', function () {
        const deviceId = $("#deviceId").val();
        const pf = {};
        pf["deviceId"] = deviceId;
        pf["channelId"] = -1; // All devices
        pf["profile"] = $(this).val();

        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/speed',
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

    $('#externalHubStatus').on('change', function () {
        const deviceId = $("#deviceId").val();
        const status = $(this).val();
        const pf = {};
        pf["deviceId"] = deviceId;
        pf["enabled"] = status === "1";
        const json = JSON.stringify(pf, null, 2);
        $.ajax({
            url: '/api/hub/status',
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

    $('.externalHubDeviceType').change(function(){
        const container = $(this).closest(".externalHubDevice");
        const deviceId = $("#deviceId").val();
        const deviceType = $(this).val();
        const portId = container.find(".portId").val();
        const pf = {};

        pf["portId"] = parseInt(portId);
        pf["deviceId"] = deviceId;
        pf["deviceType"] = parseInt(deviceType);
        const json = JSON.stringify(pf, null, 2);
        $.ajax({
            url: '/api/hub/type',
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

    $('.externalHubDeviceAmount').change(function(){
        const container = $(this).closest(".externalHubDevice");
        const deviceId = $("#deviceId").val();
        const deviceAmount = $(this).val();
        const portId = container.find(".portId").val();
        const pf = {};

        pf["portId"] = parseInt(portId);
        pf["deviceId"] = deviceId;
        pf["deviceAmount"] = parseInt(deviceAmount);
        const json = JSON.stringify(pf, null, 2);
        $.ajax({
            url: '/api/hub/amount',
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

    $('#deviceRgb').on('change', function () {
        const deviceId = $("#deviceId").val();
        const pf = {};
        pf["deviceId"] = deviceId;
        pf["profile"] = $(this).val();

        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/color',
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