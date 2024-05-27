"use strict";

document.addEventListener("DOMContentLoaded", function () {
    // how to integrate Google Calendar: https://fullcalendar.io/docs/google_calendar/

    const todayDate = moment().startOf("day");
    const YM = todayDate.format("YYYY-MM");
    const YESTERDAY = todayDate.clone().subtract(1, "day").format("YYYY-MM-DD");
    const TODAY = todayDate.format("YYYY-MM-DD");
    const TOMORROW = todayDate.clone().add(1, "day").format("YYYY-MM-DD");

    let headerJSON;
    if (window.innerWidth > 991) {
        headerJSON = {
            start: "prev,next today",
            center: "title",
            end: "dayGridMonth,dayGridWeek,dayGridDay,listWeek",
        };
    } else if (window.innerWidth < 991) {
        headerJSON = {
            start: "title",
            center: "",
            end: "prev,next today",
        };
    }

    var calendarEl = document.getElementById("calendar");
    var calendar = new FullCalendar.Calendar(calendarEl, {
        initialView: "dayGridMonth",
        headerToolbar: headerJSON,
        droppable: true,
        editable: true,
        eventLimit: true, // allow "more" link when too many events
        handleWindowResize: true,
        themeSystem: "bootstrap",
        bootstrapGlyphicons: false,
        events: [
            {
                title: "All Day Event",
                start: YM + "-01",
            },
            {
                title: "Long Event",
                start: YM + "-07",
                end: YM + "-10",
            },
            {
                id: 999,
                title: "Repeating Event",
                start: YM + "-09T16:00:00",
            },
            {
                id: 999,
                title: "Repeating Event",
                start: YM + "-16T16:00:00",
            },
            {
                title: "Conference",
                start: YESTERDAY,
                end: TOMORROW,
            },
            {
                title: "Meeting",
                start: TODAY + "T10:30:00",
                end: TODAY + "T12:30:00",
            },
            {
                title: "Lunch",
                start: TODAY + "T12:00:00",
            },
            {
                title: "Meeting",
                start: TODAY + "T14:30:00",
            },
            {
                title: "Happy Hour",
                start: TODAY + "T17:30:00",
            },
            {
                title: "Dinner",
                start: TODAY + "T20:00:00",
            },
            {
                title: "Birthday Party",
                start: TOMORROW + "T07:00:00",
            },
            {
                title: "Click for Google",
                url: "http://google.com/",
                start: YM + "-28",
            },
        ],
    });
    calendar.render();
});
