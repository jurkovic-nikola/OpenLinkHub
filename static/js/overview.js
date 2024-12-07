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

    function componentToHex(c) {
        const hex = c.toString(16);
        return hex.length === 1 ? "0" + hex : hex;
    }
    function rgbToHex(r, g, b) {
        return "#" + componentToHex(r) + componentToHex(g) + componentToHex(b);
    }
    function hexToRgb(hex) {
        const result = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(hex);
        return result ? {
            r: parseInt(result[1], 16),
            g: parseInt(result[2], 16),
            b: parseInt(result[3], 16)
        } : null;
    }


    $('.userProfile').on('change', function () {
        const deviceId = $("#deviceId").val();
        const userProfileValue = $(this).val();
        if (userProfileValue.length < 1) {
            toast.warning('Invalid profile selected');
            return false;
        }

        const pf = {};
        pf["deviceId"] = deviceId;
        pf["userProfileName"] = userProfileValue;

        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/userProfile',
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

    $('.brightness').on('change', function () {
        const deviceId = $("#deviceId").val();
        const brightness = $(this).val();
        const brightnessValue = parseInt(brightness);

        if (brightnessValue < 0 || brightnessValue > 3) {
            toast.warning('Invalid brightness selected');
            return false;
        }

        const pf = {};
        pf["deviceId"] = deviceId;
        pf["brightness"] = brightnessValue;

        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/brightness',
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

    $('#brightnessSlider').on('change', function () {
        const deviceId = $("#deviceId").val();
        const brightness = $(this).val();
        const brightnessValue = parseInt(brightness);

        if (brightnessValue < 0 || brightnessValue > 100) {
            toast.warning('Invalid brightness selected');
            return false;
        }

        const pf = {};
        pf["deviceId"] = deviceId;
        pf["brightness"] = brightnessValue;

        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/brightness/gradual',
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

    $('.saveUserProfile').on('click', function () {
        let modalElement = '<div class="modal fade text-start" id="newUserProfileModal" tabindex="-1" aria-labelledby="newUserProfileLabel" aria-hidden="true">';
        modalElement+='<div class="modal-dialog">';
        modalElement+='<div class="modal-content">';
        modalElement+='<div class="modal-header">';
        modalElement+='<h5 class="modal-title" id="newUserProfileLabel">Save user profile</h5>';
        modalElement+='<button class="btn-close btn-close-white" type="button" data-bs-dismiss="modal" aria-label="Close"></button>';
        modalElement+='</div>';
        modalElement+='<div class="modal-body">';
        modalElement+='<form>';
        modalElement+='<div class="mb-3">';
        modalElement+='<label class="form-label" for="userProfileName">Profile Name</label>';
        modalElement+='<input class="form-control" id="userProfileName" type="text" placeholder="Enter profile name (a-z, A-Z, 0-9, -)">';
        modalElement+='</div>';
        modalElement+='</form>';
        modalElement+='</div>';
        modalElement+='<div class="modal-footer">';
        modalElement+='<button class="btn btn-secondary" type="button" data-bs-dismiss="modal">Close</button>';
        modalElement+='<button class="btn btn-primary" type="button" id="btnSaveUserProfile">Save</button>';
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

            modal.find('#btnSaveUserProfile').on('click', function () {
                const userProfileValue = userProfileName.val();
                if (userProfileValue.length < 1) {
                    toast.warning('Profile name can not be empty');
                    return false
                }
                const deviceId = $("#deviceId").val();

                const pf = {};
                pf["deviceId"] = deviceId;
                pf["userProfileName"] = userProfileValue;
                const json = JSON.stringify(pf, null, 2);

                $.ajax({
                    url: '/api/userProfile',
                    type: 'PUT',
                    data: json,
                    cache: false,
                    success: function(response) {
                        try {
                            if (response.status === 1) {
                                modal.modal('toggle');
                                $('.userProfile').append($('<option>', {
                                    value: userProfileValue,
                                    text: userProfileValue
                                }));
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
        })
    });

    $('.moveLeft').on('click', function () {
        const data = $(this).attr('data').split(";");
        const deviceId = $("#deviceId").val();

        if (data.length < 2 || data.length > 2) {
            toast.warning('Invalid profile selected');
            return false;
        }

        const pf = {};
        pf["deviceId"] = deviceId;
        pf["position"] = parseInt(data[0]);
        pf["channelId"] = parseInt(data[1]);
        pf["direction"] = 0;
        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/position',
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

    $('.moveRight').on('click', function () {
        const data = $(this).attr('data').split(";");
        const deviceId = $("#deviceId").val();

        if (data.length < 2 || data.length > 2) {
            toast.warning('Invalid profile selected');
            return false;
        }

        const pf = {};
        pf["deviceId"] = deviceId;
        pf["position"] = parseInt(data[0]);
        pf["channelId"] = parseInt(data[1]);
        pf["direction"] = 1;
        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/position',
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
                pf["deviceType"] = 0;
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

    $('.newRgbLabel').on('click', function () {
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
                pf["deviceType"] = 1;
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
                                $(elementSpeedId).html(value.rpm + " RPM");
                                $(elementTemperatureId).html(value.temperatureString);
                            });
                        }
                    }
                }
            });

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

    $('.globalRgb').on('change', function () {
        const deviceId = $("#deviceId").val();
        const profile = $(this).val();

        const pf = {};
        pf["deviceId"] = deviceId;
        pf["channelId"] = -1;
        pf["profile"] = profile;

        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/color',
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

    $('.keyboardRgbProfile').on('change', function () {
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

    $('.miscRgbProfile').on('change', function () {
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

    $('.rgbStrips').on('change', function () {
        const deviceId = $("#deviceId").val();
        const stripData = $(this).val().split(";");
        if (stripData.length < 2 || stripData.length > 2) {
            toast.warning('Invalid profile selected');
            return false;
        }

        const pf = {};
        pf["deviceId"] = deviceId;
        pf["channelId"] = parseInt(stripData[0]);
        pf["stripId"] = parseInt(stripData[1]);

        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/hub/strip',
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

    $('.lcdMode').on('change', function () {
        const deviceId = $("#deviceId").val();
        const mode = $(this).val().split(";");

        const pf = {};
        pf["deviceId"] = deviceId;
        pf["channelId"] = parseInt(mode[0]);
        pf["mode"] = parseInt(mode[1]);

        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/lcd',
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

    $('.lcdDevices').on('change', function () {
        const deviceId = $("#deviceId").val();
        const device = $(this).val().split(";");

        const pf = {};
        pf["deviceId"] = deviceId;
        pf["channelId"] = parseInt(device[0]);
        pf["lcdSerial"] = device[1];

        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/lcd/device',
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

    $('.lcdRotation').on('change', function () {
        const deviceId = $("#deviceId").val();
        const rotation = $(this).val().split(";");

        const pf = {};
        pf["deviceId"] = deviceId;
        pf["channelId"] = parseInt(rotation[0]);
        pf["rotation"] = parseInt(rotation[1]);

        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/lcd/rotation',
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

    $('.addCustomARGBDevice').on('click', function () {
        const deviceId = $("#deviceId").val();
        const portId = $(".customLedPort").val();
        const deviceType = $(".customLedPortLEDAmount").val();

        const pf = {};
        pf["deviceId"] = deviceId;
        pf["portId"] = parseInt(portId);
        pf["deviceType"] = parseInt(deviceType);

        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/argb',
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

    $('.keyboardColor').on('click', function () {
        const applyButton = $('#applyColors')
        applyButton.unbind('click');

        const deviceId = $("#deviceId").val();
        const keyInfo = $(this).attr("data-info").split(";");
        const keyId = parseInt(keyInfo[0]);
        const colorR = parseInt(keyInfo[1]);
        const colorG = parseInt(keyInfo[2]);
        const colorB = parseInt(keyInfo[3]);
        const hex = rgbToHex(colorR, colorG, colorB);
        $("#keyColor").val('' + hex + '');

        applyButton.on('click', function () {
            const keyOption = $(".keyOptions").val();
            const keyColor = $('#keyColor').val();
            const rgb = hexToRgb(keyColor);
            if (rgb.r === colorR && rgb.g === colorG && rgb.b === colorB) {
                toast.warning('Old and new colors are identical');
                return false;
            }

            const pf = {};
            const color = {red:rgb.r, green:rgb.g, blue:rgb.b}
            pf["deviceId"] = deviceId;
            pf["keyId"] = keyId;
            pf["keyOption"] = parseInt(keyOption);
            pf["color"] = color;

            const json = JSON.stringify(pf, null, 2);
            console.log(json)
            $.ajax({
                url: '/api/keyboard/color',
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
    });

    $('.miscColor').on('click', function () {
        const applyButton = $('#applyColors')
        applyButton.unbind('click');

        const deviceId = $("#deviceId").val();
        const miscInfo = $(this).attr("data-info").split(";");
        const areaId = parseInt(miscInfo[0]);
        const colorR = parseInt(miscInfo[1]);
        const colorG = parseInt(miscInfo[2]);
        const colorB = parseInt(miscInfo[3]);
        const hex = rgbToHex(colorR, colorG, colorB);
        $("#miscColor").val('' + hex + '');

        applyButton.on('click', function () {
            const miscOptions = $(".miscOptions").val();
            const miscColor = $('#miscColor').val();
            const rgb = hexToRgb(miscColor);
            if (rgb.r === colorR && rgb.g === colorG && rgb.b === colorB) {
                toast.warning('Old and new colors are identical');
                return false;
            }

            const pf = {};
            const color = {red:rgb.r, green:rgb.g, blue:rgb.b}
            pf["deviceId"] = deviceId;
            pf["areaId"] = areaId;
            pf["areaOption"] = parseInt(miscOptions);
            pf["color"] = color;

            const json = JSON.stringify(pf, null, 2);
            $.ajax({
                url: '/api/misc/color',
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
    });

    $('#saveAsProfile').on('click', function () {
        let modalElement = '<div class="modal fade text-start" id="newUserProfileModal" tabindex="-1" aria-labelledby="newUserProfileLabel" aria-hidden="true">';
        modalElement+='<div class="modal-dialog">';
        modalElement+='<div class="modal-content">';
        modalElement+='<div class="modal-header">';
        modalElement+='<h5 class="modal-title" id="newUserProfileLabel">Save keyboard profile</h5>';
        modalElement+='<button class="btn-close btn-close-white" type="button" data-bs-dismiss="modal" aria-label="Close"></button>';
        modalElement+='</div>';
        modalElement+='<div class="modal-body">';
        modalElement+='<form>';
        modalElement+='<div class="mb-3">';
        modalElement+='<label class="form-label" for="userProfileName">Profile Name</label>';
        modalElement+='<input class="form-control" id="userProfileName" type="text" placeholder="Enter profile name (a-z, A-Z, 0-9)">';
        modalElement+='</div>';
        modalElement+='</form>';
        modalElement+='</div>';
        modalElement+='<div class="modal-footer">';
        modalElement+='<button class="btn btn-secondary" type="button" data-bs-dismiss="modal">Close</button>';
        modalElement+='<button class="btn btn-primary" type="button" id="btnSaveKeyboardProfile">Save</button>';
        modalElement+='</div>';
        modalElement+='</div>';
        modalElement+='</div>';
        modalElement+='</div>';
        const modal = $(modalElement).modal('toggle');

        modal.on('hidden.bs.modal', function () {
            modal.data('bs.modal', null);
        })

        modal.on('shown.bs.modal', function (e) {
            const keyboardProfileName = modal.find('#userProfileName');
            keyboardProfileName.focus();

            modal.find('#btnSaveKeyboardProfile').on('click', function () {
                const keyboardProfileValue = keyboardProfileName.val();
                if (keyboardProfileValue.length < 1) {
                    toast.warning('Profile name can not be empty');
                    return false
                }
                const deviceId = $("#deviceId").val();

                const pf = {};
                pf["deviceId"] = deviceId;
                pf["keyboardProfileName"] = keyboardProfileValue;
                pf["new"] = true;

                const json = JSON.stringify(pf, null, 2);

                $.ajax({
                    url: '/api/keyboard/profile/new',
                    type: 'PUT',
                    data: json,
                    cache: false,
                    success: function(response) {
                        try {
                            if (response.status === 1) {
                                modal.modal('toggle');
                                $('.keyboardProfile').append($('<option>', {
                                    value: keyboardProfileValue,
                                    text: keyboardProfileValue
                                }));
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
        })
    });

    $('.keyboardProfile').on('change', function () {
        const deviceId = $("#deviceId").val();
        const pf = {};
        pf["deviceId"] = deviceId;
        pf["keyboardProfileName"] = $(this).val();

        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/keyboard/profile/change',
            type: 'POST',
            data: json,
            cache: false,
            success: function(response) {
                try {
                    if (response.status === 1) {
                        location.reload()
                    } else {
                        toast.warning(response.message);
                    }
                } catch (err) {
                    toast.warning(response.message);
                }
            }
        });
    });

    $('#saveProfile').on('click', function () {
        const deviceId = $("#deviceId").val();
        const pf = {};
        pf["deviceId"] = deviceId;
        pf["keyboardProfileName"] = "0";
        pf["new"] = false;

        const json = JSON.stringify(pf, null, 2);
        $.ajax({
            url: '/api/keyboard/profile/save',
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

    $('#deleteProfile').on('click', function () {
        const deviceId = $("#deviceId").val();
        const pf = {};
        pf["deviceId"] = deviceId;
        pf["keyboardProfileName"] = $(".keyboardProfile").val();

        const json = JSON.stringify(pf, null, 2);
        $.ajax({
            url: '/api/keyboard/profile/delete',
            type: 'DELETE',
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

    $('.keyLayout').on('change', function () {
        const deviceId = $("#deviceId").val();
        const pf = {};
        pf["deviceId"] = deviceId;
        pf["keyboardLayout"] = $(this).val();
        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/keyboard/layout',
            type: 'POST',
            data: json,
            cache: false,
            success: function(response) {
                try {
                    if (response.status === 1) {
                        location.reload()
                    } else {
                        toast.warning(response.message);
                    }
                } catch (err) {
                    toast.warning(response.message);
                }
            }
        });
    });

    $('.controlDial').on('change', function () {
        const deviceId = $("#deviceId").val();
        const pf = {};
        pf["deviceId"] = deviceId;
        pf["keyboardControlDial"] = parseInt($(this).val());
        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/keyboard/dial',
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

    $('.sleepModes').on('change', function () {
        const deviceId = $("#deviceId").val();
        const pf = {};
        pf["deviceId"] = deviceId;
        pf["sleepMode"] = parseInt($(this).val());
        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/keyboard/sleep',
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