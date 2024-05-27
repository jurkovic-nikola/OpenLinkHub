"use strict";

document.addEventListener("DOMContentLoaded", function () {
    // Variant 1 - add data via data attributes [Data Source]
    const autoCompleteJSInput = document.getElementById("autoComplete1");
    const autoCompleteJS1 = new autoComplete({
        selector: "#autoComplete1",
        data: {
            src: autoCompleteJSInput.dataset.source.split(","),
        },
        resultItem: {
            highlight: true,
        },
        events: {
            input: {
                selection: (event) => {
                    const selection = event.detail.selection.value;
                    autoCompleteJS1.input.value = selection;
                },
            },
        },
    });

    // Variant 2 - JavaScript initialization
    const autoCompleteJS2 = new autoComplete({
        selector: "#autoComplete2",
        data: {
            src: ["Sauce", "Wild Boar", "Goat"],
        },
        resultItem: {
            highlight: true,
        },
        events: {
            input: {
                selection: (event) => {
                    const selection = event.detail.selection.value;
                    autoCompleteJS2.input.value = selection;
                },
            },
        },
    });

    // Variant 3 - Loading JSON array
    const autoCompleteJS3 = new autoComplete({
        selector: "#autoComplete3",
        data: {
            src: async () => {
                try {
                    // Fetch Data from external Source
                    const source = await fetch("data/countries.json");
                    // Data is array of `Objects` | `Strings`
                    const data = await source.json();

                    return data.countries;
                } catch (error) {
                    return error;
                }
            },
        },
        resultItem: {
            highlight: true,
        },
        events: {
            input: {
                selection: (event) => {
                    const selection = event.detail.selection.value;
                    autoCompleteJS3.input.value = selection;
                },
            },
        },
    });

    // Variant 4 - Custom search in JSON object
    const autoCompleteJS4 = new autoComplete({
        selector: "#autoComplete4",
        data: {
            src: [
                { name: "Alyce", surname: "White", company: "Combot", email: "alycewhite@combot.com", city: "Talpa" },
                {
                    name: "Santos",
                    surname: "Pierce",
                    company: "Franscene",
                    email: "santospierce@franscene.com",
                    city: "Vienna",
                },
                {
                    name: "Deirdre",
                    surname: "Reed",
                    company: "Whiskey Comp.",
                    email: "deirdrereed@whiskeycomp.com",
                    city: "Belva",
                },
                {
                    name: "Whitaker",
                    surname: "Brennan",
                    company: "Opticom",
                    email: "whitakerbrennan@opticom.com",
                    city: "Lodoga",
                },
                {
                    name: "Kristin",
                    surname: "Norman",
                    company: "Irack",
                    email: "kristinnorman@irack.com",
                    city: "Bodega",
                },
            ],
            keys: ["name", "surname", "company"],
        },
        resultItem: {
            highlight: true,
        },
        events: {
            input: {
                selection: (event) => {
                    const selection = event.detail.selection.value;
                    autoCompleteJS4.input.value = selection.email;
                },
            },
        },
    });
});
