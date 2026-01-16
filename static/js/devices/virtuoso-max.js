$(document).ready(function () {
    let activeSegment = null;
    function updateZone($el, zoneId, r, g, b) {
        const hex = rgbToHex(r, g, b);

        if ($el.hasClass('seg-1')) {
            $el[0].style.setProperty('--seg1', hex);
        } else if ($el.hasClass('seg-2')) {
            $el[0].style.setProperty('--seg2', hex);
        } else if ($el.hasClass('seg-3')) {
            $el[0].style.setProperty('--seg3', hex);
        }
    }

    function getSegmentColor(segment) {
        const styles = getComputedStyle(segment);

        if (segment.classList.contains('seg-1')) {
            return styles.getPropertyValue('--seg1').trim();
        }
        if (segment.classList.contains('seg-2')) {
            return styles.getPropertyValue('--seg2').trim();
        }
        if (segment.classList.contains('seg-3')) {
            return styles.getPropertyValue('--seg3').trim();
        }
        return '#000000';
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
                        $.each(response.data, function (zoneId, zone) {
                            const $el = $("#zoneColor_" + zoneId);
                            if (!$el.length) return;

                            const c = zone.Color;
                            updateZone($el, zoneId, c.red, c.green, c.blue);
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

    $('.earcup-ring').on('mousedown', function (e) {
        const rect = this.getBoundingClientRect();
        const cx = rect.left + rect.width / 2;
        const cy = rect.top  + rect.height / 2;
        const dx = e.clientX - cx;
        const dy = e.clientY - cy;

        let angle = Math.atan2(dy, dx) * 180 / Math.PI + 90;
        if (angle < 0) angle += 360;

        // Determine segment
        if (angle >= 3 && angle <= 114) {
            activeSegment = this.querySelector('.seg-1');
        } else if (angle >= 120 && angle <= 240) {
            activeSegment = this.querySelector('.seg-2');
        } else if (angle >= 246 && angle <= 354) {
            activeSegment = this.querySelector('.seg-3');
        } else {
            activeSegment = null;
            return;
        }

        const picker = $('#colorPicker')[0];
        picker.value = getSegmentColor(activeSegment) || '#000000';
        picker.style.left = `${e.clientX}px`;
        picker.style.top  = `${e.clientY}px`;
        setTimeout(() => picker.click(), 0);
    });

    $('#colorPicker').on('input', function () {
        if (!activeSegment) return;
        const color = this.value;

        if (activeSegment.classList.contains('seg-1')) {
            activeSegment.style.setProperty('--seg1', color);
        }
        if (activeSegment.classList.contains('seg-2')) {
            activeSegment.style.setProperty('--seg2', color);
        }
        if (activeSegment.classList.contains('seg-3')) {
            activeSegment.style.setProperty('--seg3', color);
        }
    });

    getZoneAreaColors();
});