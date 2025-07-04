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

    // Init dataTable
    const dt = $('#table').DataTable(
        {
            order: [[1, 'asc']],
            select: {
                style: 'os',
                selector: 'td:first-child'
            },
            paging: false,
            searching: false,
            language: {
                emptyTable: "No profile selected. Select profile from left side"
            }
        }
    );

    $('#btnSaveNewProfile').on('click', function(){
        const profile = $("#profileName").val();
        const sensor = $("#sensor").val();

        if (profile.length < 3) {
            toast.warning('Enter your profile name. Minimum length is 3 characters');
            return false;
        }

        const staticMode = $('#staticCheckbox').is(':checked');
        const zeroRpmMode = $('#zeroRpmCheckbox').is(':checked');
        const linear = $('#linearCheckbox').is(':checked');

        const pf = {};
        pf["profile"] = profile;
        pf["static"] = staticMode;
        pf["zeroRpm"] = zeroRpmMode;
        pf["linear"] = linear;
        pf["sensor"] = parseInt(sensor);
        if (parseInt(sensor) === 3) {
            pf["hwmonDeviceId"] = $("#hwmonDeviceId").val();
        }
        if (parseInt(sensor) === 4) {
            const probeData = $("#probeData").val().split(';')
            pf["deviceId"] = probeData[0];
            pf["channelId"] = parseInt(probeData[1]);
        }
        if (parseInt(sensor) === 6) {
            const hwmonData = $("#hwmon-probeData").val().split(';')
            pf["hwmonDeviceId"] = hwmonData[0];
            pf["temperatureInputId"] = hwmonData[1];
        }
        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/temperatures/new',
            type: 'POST',
            data: json,
            cache: false,
            success: function(response) {
                try {
                    if (response.status === 1) {
                        window.location.href = '/temperature';
                    } else {
                        toast.warning(response.message);
                    }
                } catch (err) {
                    toast.warning(response.message);
                }
            }
        });
    });

    $('.tempList').on('click', function(){
        const profile = $(this).attr('id');
        $('.tempList').removeClass('selected-effect');
        $(this).addClass('selected-effect');
        $.ajax({
            url: '/api/temperatures/' + profile,
            dataType: 'JSON',
            success: function(response) {
                if (response.code === 0) {
                    toast.warning(response.message);
                } else {
                    const data = response.data.profiles;
                    dt.clear();
                    if (profile === 'Quiet' || profile === 'Normal' || profile === 'Performance' || response.data.linear === true) {
                        // Those profiles are not editable
                        $.each(data, function(i, item) {
                            if (response.data.linear === true) {
                                dt.row.add([
                                    item.id,
                                    item.min,
                                    item.max,
                                    'n/a',
                                    'n/a',
                                ]).draw();
                            } else {
                                dt.row.add([
                                    item.id,
                                    item.min,
                                    item.max,
                                    item.fans,
                                    item.pump
                                ]).draw();
                            }
                        });

                        if (response.data.linear === true) {
                            $("#deleteBtn").show();
                        } else {
                            $("#deleteBtn").hide();
                            $("#updateBtn").hide();
                        }
                    } else {
                        $("#profile").val(profile);
                        $.each(data, function(i, item) {
                            dt.row.add([
                                item.id,
                                item.min,
                                item.max,
                                '<input class="form-control" id="pf-fans-' + item.id + '" type="text" value="' + item.fans + '">',
                                '<input class="form-control" id="pf-pump-' + item.id + '" type="text" value="' + item.pump + '">'
                            ]).draw();
                        });
                        $("#deleteBtn").show();
                        $("#updateBtn").show();
                    }
                }
            }
        });
    });
    $('.deletePf').on('click', function(){
        e.stopPropagation();
    });

    $('.tempProfiles').on('click', function(){
        const profile = $(this).attr('id');
        $('.tempProfiles').removeClass('selected-effect');
        $(this).addClass('selected-effect');
        $.ajax({
            url: '/api/temperatures/graph/' + profile,
            dataType: 'JSON',
            success: function(response) {
                if (response.code === 0) {
                    toast.warning(response.message);
                } else {
                    $("#profile").val(profile);
                    $("#graph-window").show();
                    let maxValue = 100;
                    let sensor = response.data[0].sensor
                    if (sensor === 2) { // Liquid temp, max is 60
                        maxValue = 60
                    }
                    let pump = response.data[0].points
                    let fans = response.data[1].points
                    renderCanvas('graphPump', pump,"Pump Speed (%)", maxValue, "updatePump", 0);
                    renderCanvas('graphFans', fans, "Fan Speed (%)", maxValue, "updateFans", 1);
                }
            }
        });
    });

    function renderCanvas(canvasName, points, label, maxValue, buttonName, updateType) {
        function resizeCanvasToDisplaySize(canvas) {
            const rect = canvas.getBoundingClientRect();
            canvas.width = rect.width;
            canvas.height = rect.height;
        }

        const canvas = document.getElementById(canvasName);
        resizeCanvasToDisplaySize(canvas);
        const ctx = canvas.getContext('2d');
        ctx.clearRect(0, 0, canvas.width, canvas.height); // Clear any existing data

        const margin = 60;
        const width = canvas.width;
        const height = canvas.height;
        const graphWidth = width - 2 * margin;
        const graphHeight = height - 2 * margin;
        const state = {
            dragging: false,
            dragIndex: -1
        };

        function tempToX(temp) {
            return margin + (temp / maxValue) * graphWidth;
        }

        function speedToY(speed) {
            return height - margin - (speed / 100) * graphHeight;
        }

        function xToTemp(x) {
            return Math.max(0, Math.min(maxValue, ((x - margin) / graphWidth) * maxValue));
        }

        function yToSpeed(y) {
            return Math.max(0, Math.min(100, ((height - margin - y) / graphHeight) * 100));
        }

        function draw() {
            ctx.clearRect(0, 0, width, height);

            ctx.strokeStyle = "#333";
            ctx.lineWidth = 1;
            ctx.font = "12px sans-serif";
            ctx.fillStyle = "#aaa";
            ctx.textAlign = "right";
            ctx.textBaseline = "middle";

            for (let i = 0; i <= 10; i++) {
                const val = i * 10;
                const y = speedToY(val);
                ctx.beginPath();
                ctx.moveTo(margin, y);
                ctx.lineTo(width - margin, y);
                ctx.stroke();
                ctx.fillText(`${val}%`, margin - 10, y);
            }

            ctx.textAlign = "center";
            ctx.textBaseline = "top";
            for (let i = 0; i <= 10; i++) {
                const val = i * 10;
                const x = tempToX(val);
                ctx.beginPath();
                ctx.moveTo(x, height - margin);
                ctx.lineTo(x, margin);
                ctx.stroke();
                ctx.fillText(`${val}°`, x, height - margin + 5);
            }

            ctx.strokeStyle = "#888";
            ctx.beginPath();
            ctx.moveTo(margin, margin);
            ctx.lineTo(margin, height - margin);
            ctx.lineTo(width - margin, height - margin);
            ctx.stroke();

            ctx.fillStyle = "#ccc";
            ctx.font = "14px sans-serif";
            ctx.fillText("Temperature (°C)", width / 2, height-25);
            ctx.fillText(label, width / 2, 25);
            ctx.save();

            points.sort((a, b) => a.x - b.x);
            ctx.strokeStyle = "#42a5f5";
            ctx.lineWidth = 2;
            ctx.beginPath();
            points.forEach((p, i) => {
                if (p.x > 100) {
                    p.x = 100
                }
                const x = tempToX(p.x);
                const y = speedToY(p.y);
                if (i === 0) ctx.moveTo(x, y);
                else ctx.lineTo(x, y);
            });
            ctx.stroke();

            points.forEach(p => {
                const x = tempToX(p.x);
                const y = speedToY(p.y);
                ctx.fillStyle = "#42a5f5";
                ctx.beginPath();
                ctx.arc(x, y, 6, 0, Math.PI * 2);
                ctx.fill();
            });
        }

        function getMousePos(evt) {
            const rect = canvas.getBoundingClientRect();
            return {
                x: evt.clientX - rect.left,
                y: evt.clientY - rect.top
            };
        }

        function findNearbyPoint(mx, my) {
            return points.findIndex(p => {
                const dx = tempToX(p.x) - mx;
                const dy = speedToY(p.y) - my;
                return dx * dx + dy * dy < 100; // within 10px radius
            });
        }

        canvas.addEventListener("mousedown", (e) => {
            const { x, y } = getMousePos(e);
            const index = findNearbyPoint(x, y);
            if (index !== -1) {
                state.dragging = true;
                state.dragIndex = index;
            } else {
                // Add new point
                const temp = xToTemp(x);
                const speed = yToSpeed(y);
                points.push({ x: Math.round(temp), y: Math.round(speed) });
                draw();
            }
        });

        canvas.addEventListener("mousemove", (e) => {
            if (!state.dragging) return;
            const { x, y } = getMousePos(e);
            const temp = xToTemp(x);
            const speed = yToSpeed(y);
            points[state.dragIndex] = { x: Math.round(temp), y: Math.round(speed) };
            draw();
        });

        canvas.addEventListener("mouseup", () => {
            state.dragging = false;
            state.dragIndex = -1;
        });

        canvas.addEventListener("contextmenu", (e) => {
            e.preventDefault(); // Disable default right-click menu
            const { x, y } = getMousePos(e);
            const index = findNearbyPoint(x, y);
            if (index !== -1) {
                points.splice(index, 1); // Remove the point
                draw(); // Redraw graph
            }
        });
        draw();

        // Button cleanup
        const button = document.getElementById(buttonName);
        if (button._clickListener) {
            button.removeEventListener("click", button._clickListener);
        }

        button._clickListener = function () {
            let capturedPoints = points.map(p => ({ ...p }));
            const profile = $("#profile").val();
            const pf = {};
            pf["profile"] = profile;
            pf["updateType"] = parseInt(updateType);
            pf["points"] = capturedPoints;
            const json = JSON.stringify(pf, null, 2);

            $.ajax({
                url: '/api/temperatures/updateGraph',
                type: 'PUT',
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
        };
        button.addEventListener("click", button._clickListener);
    }

    $('#delete').on('click', function(){
        const profile = $("#profile").val();

        const pf = {};
        pf["profile"] = profile;
        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/temperatures/delete',
            type: 'DELETE',
            data: json,
            cache: false,
            success: function(response) {
                try {
                    if (response.status === 1) {
                        toast.success(response.message);
                        $('#' + profile).remove();
                        $('#deleteTempModal').modal('hide');
                        $("#profile").val('');
                    } else {
                        toast.warning(response.message);
                    }
                } catch (err) {
                    toast.warning(response.message);
                }
            }
        });
    });

    $('#updateBtn').on('click', function(){
        const profile = $("#profile").val();
        const dict = {}; // Create an empty array
        dt.rows().every( function ( rowIdx, tableLoop, rowLoop ) {
            const data = this.data();
            let fans = $("#pf-fans-" + data[0] + "").val();
            let pump = $("#pf-pump-" + data[0] + "").val();
            if (pump < 20) {
                pump = 50;
            }

            pump = parseInt(pump);
            fans = parseInt(fans);
            dict[parseInt(data[0])] = {fans, pump};
        } );

        const json = JSON.stringify(dict, null, 2);

        $.ajax({
            url: '/api/temperatures/update',
            type: 'PUT',
            data: {
                profile: profile,
                data: json
            },
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

    $('#sensor').on('change', function () {
        const value = $(this).val();
        if (value === "2") {
            $("#linear-data").show();
        } else {
            $("#linear-data").hide();
        }
        if (value === "3") {
            $("#storage-data").show();
            $("#temperature-probe-data").hide();
            $("#hwmon-sensors-probe-data").hide();
        } else if (value === "4") {
            $("#storage-data").hide();
            $("#temperature-probe-data").show();
            $("#hwmon-sensors-probe-data").hide();
        } else if (value === "6") {
            $("#storage-data").hide();
            $("#temperature-probe-data").hide();
            $("#hwmon-sensors-probe-data").show();
        } else {
            $("#storage-data").hide();
            $("#temperature-probe-data").hide();
            $("#hwmon-sensors-probe-data").hide();
        }
    });
});