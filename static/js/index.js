"use strict";

document.addEventListener("DOMContentLoaded", function () {
    function autoRefresh() {
        setInterval(function(){
            $.ajax({
                url:'/api/cpuTemp',
                type:'get',
                success:function(result){
                    $("#cpu_temp").html(result.data + " °C");
                }
            });
            $.ajax({
                url:'/api/gpuTemp',
                type:'get',
                success:function(result){
                    $("#gpu_temp").html(result.data + " °C");
                }
            });

            $.ajax({
                url:'/api/nvmeTemp',
                type:'get',
                success:function(result){
                    $.each(result.data, function( index, value ) {
                        $("#nvme_temp-" + value.Key).html(value.Temperature + " °C");
                    });
                }
            });

            $.ajax({
                url:'/api/aio',
                type:'get',
                success:function(result){
                    if (result.data != null) {
                        if (result.data.length > 0) {
                            $.each(result.data, function( index, value ) {
                                $("#aio_temp-" + value.Serial).html(value.Temperature + " °C");
                                $("#aio_speed-" + value.Serial).html(value.Rpm + " RPM");
                            });
                        }
                    }
                }
            });
        },3000);
    }
    autoRefresh();
});