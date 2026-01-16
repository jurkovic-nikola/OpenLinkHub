$(document).ready(function () {
    function updateZone(r, g, b) {
        const hex = rgbToHex(r, g, b);
        $('.color0').css('color', hex);
    }

    function getZoneAreaColors() {
        const deviceId = $("#deviceId").val();
        $.ajax({
            url: '/api/color/zone/' + deviceId,
            type: 'GET',
            cache: false,
            success: function (response) {
                try {
                    if (response.status === 1) {
                        const c = response.data[0].Color;
                        updateZone(c.red, c.green, c.blue);
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