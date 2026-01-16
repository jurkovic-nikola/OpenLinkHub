$(document).ready(function () {
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

                                const $el = $("#mm700_" + zoneId);
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