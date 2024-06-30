"use strict";

document.addEventListener("DOMContentLoaded", function () {
    function autoRefresh() {
        setInterval(function(){
            const deviceId = $("#deviceId").val()
            $.ajax({
                url:'/api/cputemp',
                type:'get',
                success:function(result){
                    $("#cpu_temp").html(result.data);
                }
            });
        },3000);
    }
    autoRefresh();
});