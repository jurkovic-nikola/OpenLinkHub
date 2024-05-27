'use strict';

document.addEventListener("DOMContentLoaded", function () {
    // Gauges

    var gauge1 = document.getElementById('gauge1');
    var gauge2 = document.getElementById('gauge2');
    var gauge3 = document.getElementById('gauge3');
    var gauge4 = document.getElementById('gauge4');

    var opts = {
        angle: 0, // The span of the gauge arc
        lineWidth: 0.06, // The line thickness
        radiusScale: 1, // Relative radius
        pointer: {
            length: 0.6, // // Relative to gauge radius
            strokeWidth: 0.035, // The thickness
            color: '#fff' // Fill color
        },
        fontSize: 20,
        limitMax: false, // If false, max value increases automatically if value > maxValue
        limitMin: false, // If true, the min value of the gauge will be fixed
        colorStart: '#6F6EA0', // Colors
        colorStop: '#C0C0DB', // just experiment with them
        strokeColor: '#eee', // to see which ones work best for you
        generateGradient: false,
        scaleOverride: true,
        highDpiSupport: true // High resolution support
    };


    opts.colorStop = "#864DD9";
    var gaugeObject1 = new Donut(gauge1).setOptions(opts); // create sexy gauge!

    gaugeObject1.maxValue = 3000; // set max gauge value
    gaugeObject1.setMinValue(0); // set min value
    gaugeObject1.set(Math.floor(Math.random() * 3000)); // set actual value
    gaugeObject1.setTextField(document.getElementById("gauge1Value"));

    opts.colorStop = "#CF53F9";
    var gaugeObject2 = new Donut(gauge2).setOptions(opts); // create sexy gauge!

    gaugeObject2.maxValue = 3000; // set max gauge value
    gaugeObject2.setMinValue(0); // set min value
    gaugeObject2.set(Math.floor(Math.random() * 3000)); // set actual value - random in this case
    gaugeObject2.setTextField(document.getElementById("gauge2Value"));


    opts.colorStop = "#e95f71";
    var gaugeObject3 = new Donut(gauge3).setOptions(opts); // create sexy gauge!

    gaugeObject3.maxValue = 3000; // set max gauge value
    gaugeObject3.setMinValue(0); // set min value
    gaugeObject3.set(Math.floor(Math.random() * 3000)); // set actual value - random in this case
    gaugeObject3.setTextField(document.getElementById("gauge3Value"));


    opts.colorStop = "#7127AC";
    var gaugeObject4 = new Donut(gauge4).setOptions(opts); // create sexy gauge!

    gaugeObject4.maxValue = 3000; // set max gauge value
    gaugeObject4.setMinValue(0); // set min value
    gaugeObject4.set(Math.floor(Math.random() * 3000)); // set actual value - random in this case
    gaugeObject4.setTextField(document.getElementById("gauge4Value"));

    var intervalID = setInterval(function () {
        gaugeObject1.set(Math.floor(Math.random() * 3000))
        gaugeObject2.set(Math.floor(Math.random() * 3000))
        gaugeObject3.set(Math.floor(Math.random() * 3000))
        gaugeObject4.set(Math.floor(Math.random() * 3000))
    }, 5000);



// Sparklines - Theme settings

    function findClosest(target, tagName) {
        if (target.tagName === tagName) {
            return target;
        }

        while ((target = target.parentNode)) {
            if (target.tagName === tagName) {
                break;
            }
        }

        return target;
    }

    var btc = [
        { name: "Bitcoin", date: "2017-01-01", value: 967.6 },
        { name: "Bitcoin", date: "2017-02-01", value: 957.02 },
        { name: "Bitcoin", date: "2017-03-01", value: 1190.78 },
        { name: "Bitcoin", date: "2017-04-01", value: 1071.48 },
        { name: "Bitcoin", date: "2017-05-01", value: 1354.21 },
        { name: "Bitcoin", date: "2017-06-01", value: 2308.08 },
        { name: "Bitcoin", date: "2017-07-01", value: 2483.5 },
        { name: "Bitcoin", date: "2017-08-01", value: 2839.18 },
        { name: "Bitcoin", date: "2017-09-01", value: 4744.69 },
        { name: "Bitcoin", date: "2017-10-01", value: 4348.09 },
        { name: "Bitcoin", date: "2017-11-01", value: 6404.92 },
    ];

    var eth = [
        { name: "Ethereum", date: "2017-01-01", value: 8.3 },
        { name: "Ethereum", date: "2017-02-01", value: 10.57 },
        { name: "Ethereum", date: "2017-03-01", value: 15.73 },
        { name: "Ethereum", date: "2017-04-01", value: 49.51 },
        { name: "Ethereum", date: "2017-05-01", value: 85.69 },
        { name: "Ethereum", date: "2017-06-01", value: 226.51 },
        { name: "Ethereum", date: "2017-07-01", value: 246.65 },
        { name: "Ethereum", date: "2017-08-01", value: 213.87 },
        { name: "Ethereum", date: "2017-09-01", value: 386.61 },
        { name: "Ethereum", date: "2017-10-01", value: 303.56 },
        { name: "Ethereum", date: "2017-11-01", value: 298.21 },
    ];

    var options = {
        onmousemove(event, datapoint) {
            var svg = findClosest(event.target, "svg");
            var tooltip = svg.nextElementSibling;
            var date = new Date(datapoint.date).toUTCString().replace(/^.*?, (.*?) \d{2}:\d{2}:\d{2}.*?$/, "$1");

            tooltip.hidden = false;
            tooltip.textContent = `${date}: $${datapoint.value.toFixed(2)} USD`;
            tooltip.style.top = `${event.offsetY}px`;
            tooltip.style.left = `${event.offsetX + 20}px`;
        },

        onmouseout() {
            var svg = findClosest(event.target, "svg");
            var tooltip = svg.nextElementSibling;

            tooltip.hidden = true;
        },
    };

    sparkline.sparkline(document.querySelector(".btc"), btc, options);
    sparkline.sparkline(document.querySelector(".eth"), eth, options);

    function randNumbers() {
        var numbers = [];

        for (var i = 0; i < 20; i += 1) {
            numbers.push(Math.random() * 50);
        }

        return numbers;
    }

    document.querySelectorAll('.sparkline-static').forEach(function (svg) {
        sparkline.sparkline(svg, randNumbers());
    });
    setInterval(function () {
        document.querySelectorAll('.sparkline-random').forEach(function (svg) {
            sparkline.sparkline(svg, randNumbers());
        });
    }, 3000);
});
