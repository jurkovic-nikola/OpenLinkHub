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

    $('.clusterRgbProfile').on('change', function () {
        const deviceId = $("#deviceId").val();
        const profile = $(this).val().split(";");
        if (profile.length < 2 || profile.length > 2) {
            toast.warning(i18n.t('txtInvalidProfileSelected'));
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

    $('#clusterLightingToggle').on('change', function () {
        const deviceId = $("#deviceId").val();
        const checked = $(this).prop('checked');

        const pf = {};
        pf["deviceId"] = deviceId;
        pf["channelId"] = 0;

        if (checked) {
            const lastNonOff = $(this).attr('data-last-non-off') || '';
            const profileVal = $('#clusterRgbProfile').val().split(";");
            const dropdownProfile = (profileVal.length >= 2) ? profileVal[1] : '';

            if (lastNonOff && lastNonOff !== 'off') {
                pf["profile"] = lastNonOff;
            } else if (dropdownProfile && dropdownProfile !== 'off') {
                pf["profile"] = dropdownProfile;
            } else {
                pf["profile"] = "rainbow";
            }
        } else {
            pf["profile"] = "off";
        }

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

    $('#btnApplySolidColor').on('click', function () {
        const hex = $('#clusterSolidColor').val();
        const color = {
            "red": parseInt(hex.slice(1, 3), 16),
            "green": parseInt(hex.slice(3, 5), 16),
            "blue": parseInt(hex.slice(5, 7), 16),
            "brightness": 1
        };

        const json = JSON.stringify({ "color": color }, null, 2);

        $.ajax({
            url: '/api/color/all',
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

    $('#brightnessSlider').on('change', function () {
        const deviceId = $("#deviceId").val();
        const brightness = $(this).val();
        const brightnessValue = parseInt(brightness);

        if (brightnessValue < 0 || brightnessValue > 100) {
            toast.warning(i18n.t('txtInvalidBrightness'));
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

    const $brightnessSlider = $("#brightnessSlider");
    const $brightnessSliderValue = $("#brightnessSliderValue");
    function updateSlider() {
        const min = Number($brightnessSlider.attr("min"));
        const max = Number($brightnessSlider.attr("max"));
        const value = Number($brightnessSlider.val());

        const percent = ((value - min) / (max - min)) * 100;

        $brightnessSlider.css("--slider-progress", percent + "%");
        $brightnessSliderValue.text(value + " %");
    }

    if ($brightnessSlider.length) {
        $brightnessSlider.on("input", updateSlider);
        updateSlider();
    }

    const $speedSlider = $("#speedSlider");
    const $speedSliderValue = $("#speedSliderValue");
    const $speedControlGroup = $("#clusterSpeedControlGroup");
    const $remainingControlGroups = $(
        "#clusterBrightnessControlGroup, #clusterRgbControlGroup"
    );

    function updateSpeedControlVisibility(profile) {
        const visible = LumenForgeRgbSpeed.hasSpeedControl(
            profile,
            LumenForgeRgbSpeed.SOFTWARE_CONTROL
        );

        $speedControlGroup.prop("hidden", !visible);
        $remainingControlGroups
            .removeClass("col-md-4 col-md-6")
            .addClass(visible ? "col-md-4" : "col-md-6");
        $speedSlider.prop("disabled", !visible);
        return visible;
    }

    function updateSpeedSlider() {
        const min = Number($speedSlider.attr("min"));
        const max = Number($speedSlider.attr("max"));
        const value = Number($speedSlider.val());

        const percent = ((value - min) / (max - min)) * 100;

        $speedSlider.css("--slider-progress", percent + "%");
        $speedSliderValue.text(LumenForgeRgbSpeed.formatForSlider($speedSlider[0], value));
    }

    if ($speedSlider.length) {
        const storedDuration = $speedSlider.attr("data-stored-speed");
        const profile = $speedSlider.attr("data-rgb-profile");

        if (updateSpeedControlVisibility(profile)) {
            LumenForgeRgbSpeed.configureSlider(
                $speedSlider[0],
                profile,
                LumenForgeRgbSpeed.SOFTWARE_CONTROL
            );
            const uiSpeed = LumenForgeRgbSpeed.storedToUiForSlider(
                $speedSlider[0],
                storedDuration
            );
            $speedSlider.val(uiSpeed);
            $speedSlider.on("input", function () {
                LumenForgeRgbSpeed.markEdited(this);
                updateSpeedSlider();
            });
            updateSpeedSlider();
        }
    }

    $speedSlider.on('change', function () {
        const deviceId = $("#deviceId").val();
        const uiSpeed = $(this).val();

        const profileVal = $('#clusterRgbProfile').val().split(";");
        if (profileVal.length < 2) return;
        const profile = profileVal[1];

        if (!LumenForgeRgbSpeed.hasSpeedControl(
            profile,
            LumenForgeRgbSpeed.SOFTWARE_CONTROL
        )) {
            return;
        }

        this.dataset.rgbProfile = profile;
        this.dataset.speedControlMode = LumenForgeRgbSpeed.SOFTWARE_CONTROL;
        LumenForgeRgbSpeed.markEdited(this);
        const storedDuration = LumenForgeRgbSpeed.uiToStoredForSlider(this, uiSpeed);

        $.ajax({
            url: '/api/color/profile/' + deviceId + '/' + profile,
            type: 'GET',
            cache: false,
            success: function (response) {
                if (response.status === 1) {
                    const data = response.data;

                    const pf = {};
                    pf["deviceId"] = deviceId;
                    pf["profile"] = profile;
                    pf["speed"] = storedDuration;

                    pf["startColor"] = data.start ? {
                        red: data.start.red,
                        green: data.start.green,
                        blue: data.start.blue,
                        temperature: data.start.temperature
                    } : { red: 0, green: 0, blue: 0 };

                    pf["endColor"] = data.end ? {
                        red: data.end.red,
                        green: data.end.green,
                        blue: data.end.blue,
                        temperature: data.end.temperature
                    } : { red: 0, green: 0, blue: 0 };

                    pf["middleColor"] = data.middle ? {
                        red: data.middle.red,
                        green: data.middle.green,
                        blue: data.middle.blue,
                        temperature: data.middle.temperature
                    } : { red: 0, green: 0, blue: 0 };

                    pf["alternateColors"] = data.alternateColors || false;
                    pf["rgbDirection"] = data.rgbDirection || 0;
                    pf["colorZones"] = data.gradients || null;
                    pf["rgbMinTemp"] = data.minTemp || 0;
                    pf["rgbMaxTemp"] = data.maxTemp || 0;

                    const json = JSON.stringify(pf, null, 2);

                    $.ajax({
                        url: '/api/color/change',
                        type: 'PUT',
                        data: json,
                        cache: false,
                        success: function (res) {
                            try {
                                if (res.status === 1) {
                                    toast.success(res.message);
                                } else {
                                    toast.warning(res.message);
                                }
                            } catch (err) {
                                toast.warning(res.message);
                            }
                        }
                    });
                }
            }
        });
    });

    $("#clusterSortable").sortable({
        helper: function(e, tr) {
            var $originals = tr.children();
            var $helper = tr.clone();
            $helper.children().each(function(index) {
                $(this).width($originals.eq(index).width());
            });
            $helper.css("background-color", "rgba(255, 255, 255, 0.05)");
            return $helper;
        },
        axis: "y",
        update: function (event, ui) {
            const deviceOrder = [];
            $(this).children('tr').each(function () {
                deviceOrder.push($(this).data('serial').toString());
            });

            const payload = {
                deviceOrder: deviceOrder
            };

            $.ajax({
                url: '/api/cluster/order',
                type: 'PUT',
                data: JSON.stringify(payload),
                contentType: 'application/json',
                success: function(response) {
                    if (response.status === 1) {
                        toast.success(response.message);
                    } else {
                        toast.warning(response.message);
                    }
                },
                error: function() {
                    toast.error("Failed to update cluster order");
                }
            });
        }
    }).disableSelection();
});
