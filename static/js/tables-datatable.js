/* =============================================
    VANILLA DATATABLES INITIALIZING
============================================== */
"use strict";

document.addEventListener("DOMContentLoaded", function () {
    // White DataTable

    const dataTable = new simpleDatatables.DataTable("#datatable1", {
        searchable: false,
        labels: {
            perPage: "Show {select} entries",
            info: "Showing {start} to {end} of {rows} entries",
        },
    });

    function adjustTableColumns() {
        let columns = dataTable.columns();

        if (window.innerWidth > 900) {
            columns.show([2, 3, 4, 5]);
        } else if (window.innerWidth > 600) {
            columns.hide([4, 5]);
            columns.show([2, 3]);
        } else {
            columns.hide([2, 3, 4, 5]);
        }
    }

    function bootstrapizeHeader(dataTable) {
        const tableWrapper = dataTable.table.closest(".dataTable-wrapper");

        const input = tableWrapper.querySelector(".dataTable-input");
        if (input) {
            input.classList.add("form-control", "form-control-sm");
        }

        const dataTableSelect = tableWrapper.querySelector(".dataTable-selector");
        if (dataTableSelect) {
            dataTableSelect.classList.add("form-select", "form-select-sm");
        }

        const dataTableContainer = tableWrapper.querySelector(".dataTable-container");
        if (dataTableContainer) {
            dataTableContainer.classList.add("border-0");
        }
    }

    adjustTableColumns();

    window.addEventListener("resize", adjustTableColumns);

    dataTable.on("datatable.init", function () {
        bootstrapizeHeader(dataTable);
    });

    // const dataTable2 = new simpleDatatables.DataTable("#datatable2", {
    //     searchable: false,
    // });

    // dataTable2.on("datatable.init", function () {
    //     bootstrapizeHeader(dataTable2);
    // });
});
