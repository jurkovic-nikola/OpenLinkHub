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

    $(function() {
        for (let i = 0; i <= 2; i++) {
            const stage = document.getElementById("stage" + i);
            const stageValue = document.getElementById("stageValue" + i);
            stage.oninput = function() {
                stageValue.value = this.value;
            }
        }
    });

    $('#defaultDPI').on('click', function () {
        for (let i = 0; i <= 2; i++) {
            const stage = document.getElementById("stage" + i);
            const stageValue = document.getElementById("stageValue" + i);
            switch (i) {
                case 0:
                    stage.value = 800
                    break;
                case 1:
                    stage.value = 1500
                    break;
                case 2:
                    stage.value = 3000
                    break;
            }
            stageValue.value = stage.value
        }
    });

    $('#saveDPI').on('click', function () {
        const deviceId = $("#deviceId").val();

        const pf = {};
        let stages = {};

        pf["deviceId"] = deviceId;
        for (let i = 0; i <= 2; i++) {
            const stage = $("#stageValue" + i).val();
            stages[i] =  parseInt(stage);
        }
        pf["stages"] = stages;
        const json = JSON.stringify(pf, null, 2);

        console.log(json)
        $.ajax({
            url: '/api/mouse/dpi',
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