"use strict";
$(document).ready(function () {
    const deviceId = $("#deviceId").val();
    $.ajax({
        url: '/api/position/' + deviceId,
        type: 'GET',
        cache: false,
        success: function(response) {
            const $container = $("#system-cards");
            response.data.forEach(id => {
                const $el = $("#" + id);
                if ($el.length) {
                    $container.append($el);
                }
            });

            $container.css("visibility", "visible");
            $container.addClass("ready");

            $container.sortable({
                items: "> div",
                handle: ".drag-handle",
                tolerance: "pointer",
                containment: false,
                placeholder: "card-placeholder",
                forcePlaceholderSize: true,
                start: function (e, ui) {
                    const $item = ui.item;
                    const $ph   = ui.placeholder;
                    $ph.addClass($item.attr("class"));
                    $ph.height($item.outerHeight());
                    const gutterHalf = parseFloat($item.css("padding-right")) || 0; // Bootstrap: left == right
                    $ph.width($item.outerWidth() - gutterHalf * 4);
                    $ph.css({
                        margin: $item.css("margin"),
                        padding: $item.css("padding")
                    });
                },
                update: function () {
                    const order = $(this).children().map(function () {
                        return this.id;
                    }).get();
                    const pf = {
                        deviceId: deviceId,
                        positions: order
                    };

                    $.ajax({
                        url: '/api/position/update',
                        type: 'POST',
                        contentType: 'application/json',
                        data: JSON.stringify(pf),
                        cache: false,
                        success: function (response) {
                            try {
                                if (response.status === 1) {
                                    //
                                } else {
                                    toast.warning(response.message);
                                }
                            } catch (err) {
                                toast.warning(response.message);
                            }
                        }
                    });
                }
            });
        }
    });
});