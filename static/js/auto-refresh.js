setInterval(function(){
    auto_refresh();
},3000);
function auto_refresh(){
    $.ajax({
        url:'/devices',
        type:'get',
        success:function(result){
            $.each(result.devices, function( index, value ) {
                const elementSpeedId = "#speed-" + value.deviceId;
                const elementTemperatureId = "#temperature-" + value.deviceId;
                $(elementSpeedId).html(value.rpm);
                $(elementTemperatureId).html(value.temperature);
            });
        }
    });
}