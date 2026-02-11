$(document).ready(function () {
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
    
    function updateZone($el, index, r, g, b) {
        const info = `${index};${r};${g};${b}`;
        $el.attr("data-info", info).data("info", info);
        $el.css("border", `1px solid rgba(${r}, ${g}, ${b}, 1)`);
    }

    function getZoneAreaColors() {
        const deviceId = $("#deviceId").val();
        const pf = {};
        pf["deviceId"] = deviceId;
        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/color/zone/' + deviceId,
            type: 'GET',
            data: json,
            cache: false,
            success: function(response) {
                try {
                    if (response.status === 1) {
                        $.each(response.data.row, function (_, row) {
                            $.each(row.zones, function (zoneId, zone) {

                                const $el = $("#st100_" + zoneId);
                                if (!$el.length) return;

                                const c = zone.color;
                                updateZone($el, zoneId, c.red, c.green, c.blue);
                            });
                        });
                    } else {
                        toast.warning(response.data);
                    }
                } catch (err) {
                    toast.warning(response.message);
                }
            }
        });
    }
    getZoneAreaColors();
});