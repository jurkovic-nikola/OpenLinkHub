"use strict";

window.rgbToHex = function (r, g, b) {
    return (
        '#' +
        [r, g, b]
            .map(x => {
                const h = Number(x).toString(16);
                return h.length === 1 ? '0' + h : h;
            })
            .join('')
    );
};
window.hexToRgb = function(hex) {
    hex = hex.replace('#', '');
    return {
        r: parseInt(hex.substring(0, 2), 16),
        g: parseInt(hex.substring(2, 4), 16),
        b: parseInt(hex.substring(4, 6), 16)
    };
}

$(document).ready(function () {
    const $sideToneSlider = $("#sideToneValue");
    const $sideToneSliderValue = $("#sideToneValueText");

    // Update slider text value
    function updateSlider() {
        const min = Number($sideToneSlider.attr("min"));
        const max = Number($sideToneSlider.attr("max"));
        const value = Number($sideToneSlider.val());

        const percent = ((value - min) / (max - min)) * 100;

        $sideToneSlider.css("--slider-progress", percent + "%");
        $sideToneSliderValue.text(value + " %");
    }

    if ($sideToneSlider.length) {
        $sideToneSlider.on("input", updateSlider);
        updateSlider();
    }

    $('.headsetRgbProfile').on('change', function () {
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

    $('#saveZoneColors').on('click', function () {
        const deviceId = $("#deviceId").val();
        const zones = parseInt($("#zones").val());

        let colors = {};
        for (let i = 0; i < zones; i++) {
            const zoneColor = $("#zoneColor"+i).val();
            const zoneColorRgb = hexToRgb(zoneColor);
            colors[i] = {red: zoneColorRgb.r, green: zoneColorRgb.g, blue: zoneColorRgb.b}
        }

        const pf = {};
        pf["deviceId"] = deviceId;
        pf["colorZones"] = colors

        const json = JSON.stringify(pf, null, 2);
        $.ajax({
            url: '/api/headset/zoneColors',
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

    $('.headsetSleepModes').on('change', function () {
        const deviceId = $("#deviceId").val();
        const pf = {};
        pf["deviceId"] = deviceId;
        pf["sleepMode"] = parseInt($(this).val());
        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/headset/sleep',
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

    $('.muteIndicator').on('change', function () {
        const deviceId = $("#deviceId").val();
        const pf = {};
        pf["deviceId"] = deviceId;
        pf["muteIndicator"] = parseInt($(this).val());
        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/headset/muteIndicator',
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

    $('.noiseCancellationMode').on('change', function () {
        const deviceId = $("#deviceId").val();
        const pf = {};
        pf["deviceId"] = deviceId;
        pf["noiseCancellation"] = parseInt($(this).val());
        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/headset/anc',
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

    $('.sideToneMode').on('change', function () {
        const deviceId = $("#deviceId").val();
        const pf = {};
        pf["deviceId"] = deviceId;
        pf["sideTone"] = parseInt($(this).val());
        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/headset/sidetone',
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

    $('#sideToneValue').on('change', function () {
        const deviceId = $("#deviceId").val();
        const sidetone = $(this).val();
        const sidetoneValue = parseInt(sidetone);

        if (sidetoneValue < 0 || sidetoneValue > 100) {
            toast.warning(i18n.t('txtInvalidSidetone'));
            return false;
        }

        const pf = {};
        pf["deviceId"] = deviceId;
        pf["sideToneValue"] = sidetoneValue;

        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/headset/sidetoneValue',
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

    $('.wheel-select').on('change', function () {
        const deviceId = $("#deviceId").val();
        const wheelOption = parseInt($(this).val());
        const wheelId = parseInt($(this).data('wheel-id'));

        if (wheelOption < 1 || wheelOption > 2) {
            toast.warning(i18n.t('txtInvalidWheelValue'));
            return false;
        }

        if (wheelId < 1 || wheelId > 2) {
            toast.warning(i18n.t('txtInvalidWheelValue'));
            return false;
        }

        const pf = {};
        pf["deviceId"] = deviceId;
        pf["wheelId"] = wheelId;
        pf["wheelOption"] = wheelOption;

        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/headset/wheelOption',
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

    $(".toggleMuteIndicators").on("change", function () {
        const $toggle = $(this);
        const previousState = !$toggle.prop("checked");
        const newState = $toggle.prop("checked");
        const deviceId = $("#deviceId").val();

        $toggle.prop("disabled", true);

        $.ajax({
            url: '/api/headset/muteIndicator',
            type: "POST",
            contentType: "application/json",
            data: JSON.stringify({
                deviceId: deviceId,
                muteIndicator: newState ? 1 : 0
            }),
            success(response) {
                if (response?.status !== 1) {
                    $toggle.prop("checked", previousState);
                    toast.warning(response?.message || "Operation failed");
                } else {
                    toast.success(response?.message || "Operation failed");
                }
            },
            error() {
                $toggle.prop("checked", previousState);
                toast.warning("Request failed");
            },
            complete() {
                $toggle.prop("disabled", false);
            }
        });
    });

    $(".toggleSidetone").on("change", function () {
        const $toggle = $(this);
        const previousState = !$toggle.prop("checked");
        const newState = $toggle.prop("checked");
        const deviceId = $("#deviceId").val();

        $toggle.prop("disabled", true);

        $.ajax({
            url: '/api/headset/sidetone',
            type: "POST",
            contentType: "application/json",
            data: JSON.stringify({
                deviceId: deviceId,
                sideTone: newState ? 1 : 0
            }),
            success(response) {
                if (response?.status !== 1) {
                    $toggle.prop("checked", previousState);
                    toast.warning(response?.message || "Operation failed");
                } else {
                    toast.success(response?.message || "Operation failed");
                }
            },
            error() {
                $toggle.prop("checked", previousState);
                toast.warning("Request failed");
            },
            complete() {
                $toggle.prop("disabled", false);
            }
        });
    });

    $('#saveHeadsetZoneColors').on('click', function () {
        function getSegmentZoneHexColor(el) {
            const styles = getComputedStyle(el);

            if (el.classList.contains('seg-1')) {
                return styles.getPropertyValue('--seg1').trim();
            }
            if (el.classList.contains('seg-2')) {
                return styles.getPropertyValue('--seg2').trim();
            }
            if (el.classList.contains('seg-3')) {
                return styles.getPropertyValue('--seg3').trim();
            }
            return '#000000';
        }

        const deviceId = $("#deviceId").val();
        const zones = parseInt($("#zones").val(), 10);

        let colors = {};

        for (let i = 0; i < zones; i++) {
            const el = document.getElementById('zoneColor_' + i);
            if (!el) continue;

            const hex = getSegmentZoneHexColor(el);
            const rgb = hexToRgb(hex);

            colors[i] = {
                red: rgb.r,
                green: rgb.g,
                blue: rgb.b
            };
        }

        const pf = {
            deviceId: deviceId,
            colorZones: colors
        };

        $.ajax({
            url: '/api/headset/zoneColors',
            type: 'POST',
            contentType: 'application/json',
            data: JSON.stringify(pf),
            cache: false,
            success: function (response) {
                try {
                    if (response.status === 1) {
                        toast.success(response.message);
                        getZoneAreaColors();
                    } else {
                        toast.warning(response.message);
                    }
                } catch (err) {
                    toast.warning(response.message);
                }
            }
        });
    });

    $("#saveEqualizers").on("click", function () {
        let equalizers = {};
        const deviceId = $("#deviceId").val();

        $(".eq-slider").each(function () {
            let id = $(this).attr("id").replace("stage", "");
            equalizers[id] = parseInt($(this).val(), 10);
        });

        const pf = {
            deviceId: deviceId,
            equalizers: equalizers
        };
        
        $.ajax({
            url: '/api/headset/equalizer',
            type: 'POST',
            contentType: 'application/json',
            data: JSON.stringify(pf),
            cache: false,
            success: function (response) {
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

    function updateEqSlider(el) {
        const $slider = $(el);
        const min = Number($slider.attr("min"));
        const max = Number($slider.attr("max"));
        const value = Number($slider.val());
        const percent = ((value - min) / (max - min)) * 100;
        $slider.css("--slider-progress", percent + "%");
    }

    $(".eq-slider").each(function () {
        const index = this.id.replace("stage", "");
        $("#stageValue" + index).text(this.value);
        updateEqSlider(this);
    }).on("input", function () {
        const index = this.id.replace("stage", "");
        $("#stageValue" + index).text(this.value);
        updateEqSlider(this);
    });
});