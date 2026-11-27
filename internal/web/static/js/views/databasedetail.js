import { createErrorDiv, createSuccessDiv, setTitle } from "./components.js"
import { renderLoginView } from "./login.js"
import { apiFetch, renderLoading, validateSession } from "./utils.js"

export async function renderDatabaseDetails(databaseId) {
    if (!await validateSession()) {
        await renderLoginView()
        return
    }

    setTitle("Databases")
    await renderLoading()

    const container = document.getElementById("content")
    container.replaceChildren()

    try {
        const response = await apiFetch(`/api/databases/${databaseId}${window.location.search}`)
        if (response.status === 401) return
        if (!response.ok) {
            throw new Error(await response.text() || `Request failed (${response.status})`)
        }

        const data = await response.json()
        renderDatabaseDetail(container, data)
    } catch (error) {
        const backLink = document.createElement("a")
        backLink.classList.add("back-link")
        backLink.href = "/databases"
        backLink.textContent = "Back to Databases"
        container.append(backLink)

        const errorDiv = createErrorDiv(`Unable to load database: ${error.message}`)
        errorDiv.hidden = false
        container.append(errorDiv)
    }
}

function renderDatabaseDetail(container, data) {
    setTitle(data.database.name)

    const backLink = document.createElement("a")
    backLink.classList.add("back-link")
    backLink.href = "/databases"
    backLink.textContent = "Back to Databases"
    container.append(backLink)

    const header = document.createElement("div")
    header.classList.add("page-header")
    const title = document.createElement("h1")
    title.textContent = data.database.name
    header.append(title)

    const accessBadge = document.createElement("span")
    const canEdit = data.database.accessLevel === "edit"
    accessBadge.classList.add("badge", canEdit ? "badge-edit" : "badge-view")
    accessBadge.textContent = canEdit ? "View + Edit" : "View Only"
    header.append(accessBadge)
    container.append(header)

    const params = new URLSearchParams(window.location.search)
    if (params.has("error")) {
        const errorDiv = createErrorDiv(params.get("error"))
        errorDiv.hidden = false
        container.append(errorDiv)
    }
    if (params.has("success")) {
        const successDiv = createSuccessDiv(params.get("success"))
        successDiv.hidden = false
        container.append(successDiv)
    }

    const tabs = document.createElement("div")
    tabs.classList.add("tabs")
    tabs.id = "detail-tabs"
    const tabNames = ["schema", "data", "migrations"]
    const requestedTab = window.location.hash.slice(1) || params.get("tab")
    const activeTab = tabNames.includes(requestedTab) ? requestedTab : "schema"
    for (const tabName of tabNames) {
        const link = document.createElement("a")
        link.href = `#${tabName}`
        link.classList.add("tab-link")
        link.dataset.tab = tabName
        link.textContent = tabName[0].toUpperCase() + tabName.slice(1)
        if (tabName === activeTab) link.classList.add("active")
        tabs.append(link)
    }
    container.append(tabs)

    const schemaSection = createTabSection("schema", activeTab)
    appendTableSelector(schemaSection, data.tables, data.selectedTable, activeTab)
    const schemaHeading = document.createElement("h2")
    schemaHeading.textContent = `Table: ${data.selectedTable}`
    schemaSection.append(schemaHeading)
    appendDataTable(schemaSection, ["Column", "Type", "Nullable", "Default", "Key"], data.columns, "No column information available.", (row, column) => {
        appendCell(row, column.name, "mono font-bold")
        appendCell(row, column.type, "mono")
        appendCell(row, column.nullable ? "YES" : "NO")
        appendCell(row, column.default || "-")
        const keyCell = document.createElement("td")
        if (column.isPk) {
            const badge = document.createElement("span")
            badge.classList.add("badge", "badge-edit")
            badge.textContent = "PK"
            keyCell.append(badge)
        } else {
            keyCell.textContent = "-"
        }
        row.append(keyCell)
    })

    const constraintsHeading = document.createElement("h3")
    constraintsHeading.textContent = "Constraints"
    schemaSection.append(constraintsHeading)
    appendDataTable(schemaSection, ["Constraint Name", "Type", "Definition"], data.constraints, "No constraints found.", (row, constraint) => {
        appendCell(row, constraint.name, "mono")
        const typeCell = document.createElement("td")
        const badge = document.createElement("span")
        badge.classList.add("badge")
        badge.textContent = constraint.type
        typeCell.append(badge)
        row.append(typeCell)
        appendCell(row, constraint.definition, "mono")
    })

    const indexesHeading = document.createElement("h3")
    indexesHeading.textContent = "Indexes"
    schemaSection.append(indexesHeading)
    appendDataTable(schemaSection, ["Index Name", "Columns", "Type", "Unique"], data.indexes, "No indexes found.", (row, index) => {
        appendCell(row, index.name, "mono")
        appendCell(row, index.columns, "mono")
        appendCell(row, index.type)
        appendCell(row, index.unique ? "YES" : "NO")
    })
    container.append(schemaSection)

    const dataSection = createTabSection("data", activeTab)
    appendTableSelector(dataSection, data.tables, data.selectedTable, activeTab)
    const dataHeading = document.createElement("h2")
    dataHeading.textContent = `Data: ${data.selectedTable}`
    dataSection.append(dataHeading)
    appendDataTable(dataSection, data.dataHeaders, data.dataRows, "No rows found in this table.", (row, values) => {
        for (const value of values) {
            const cell = document.createElement("td")
            if (value === "") {
                const nullValue = document.createElement("span")
                nullValue.classList.add("null")
                nullValue.textContent = "NULL"
                cell.append(nullValue)
            } else {
                cell.textContent = value
            }
            row.append(cell)
        }
    }, "table-wrap data")
    container.append(dataSection)

    const migrationsSection = createTabSection("migrations", activeTab)
    const migrationsHeading = document.createElement("h2")
    migrationsHeading.textContent = "Migration History"
    migrationsSection.append(migrationsHeading)
    appendDataTable(migrationsSection, ["Version", "Description", "Executed By", "Executed At", "Status"], data.migrations, "No migration history recorded.", (row, migration) => {
        appendCell(row, migration.version, "mono font-bold")
        appendCell(row, migration.name)
        appendCell(row, migration.executedBy)
        appendCell(row, migration.executedAt, "mono")
        const statusCell = document.createElement("td")
        const statusBadge = document.createElement("span")
        statusBadge.classList.add("badge", migration.status === "success" ? "badge-success" : "badge-failed")
        statusBadge.textContent = migration.status === "success" ? "Success" : "Failed"
        statusCell.append(statusBadge)
        row.append(statusCell)
    })

    if (canEdit) {
        migrationsSection.append(createMigrationForm(data))
    } else {
        const warning = document.createElement("div")
        warning.classList.add("alert", "alert-warning")
        warning.textContent = "Access is View Only. Migration creation and execution require View + Edit access."
        migrationsSection.append(warning)
    }
    container.append(migrationsSection)
}

function createTabSection(tabName, activeTab) {
    const section = document.createElement("section")
    section.id = `tab-${tabName}`
    section.classList.add("tab-content")
    section.style.display = tabName === activeTab ? "block" : "none"
    return section
}

function appendTableSelector(container, tables, selectedTable, activeTab) {
    const selector = document.createElement("div")
    selector.style.cssText = "display: flex; gap: 0.5rem; flex-wrap: wrap; margin-bottom: 1.5rem;"
    for (const table of tables || []) {
        const link = document.createElement("a")
        link.href = `?table=${encodeURIComponent(table.name)}#${activeTab}`
        link.classList.add("btn", table.name === selectedTable ? "btn-primary" : "btn-outline", "btn-sm", "mono")
        link.textContent = table.name
        selector.append(link)
    }
    container.append(selector)
}

function appendDataTable(container, headers, rows, emptyMessage, renderRow, className = "table-wrap") {
    const wrapper = document.createElement("div")
    wrapper.classList.add(...className.split(" "))
    const table = document.createElement("table")
    const thead = document.createElement("thead")
    const headerRow = document.createElement("tr")
    for (const header of headers || []) {
        const cell = document.createElement("th")
        cell.textContent = header
        headerRow.append(cell)
    }
    thead.append(headerRow)
    table.append(thead)

    const tbody = document.createElement("tbody")
    if (!rows?.length) {
        const emptyRow = document.createElement("tr")
        const emptyCell = document.createElement("td")
        emptyCell.colSpan = Math.max(headers?.length || 1, 1)
        emptyCell.classList.add("empty", "center")
        emptyCell.textContent = emptyMessage
        emptyRow.append(emptyCell)
        tbody.append(emptyRow)
    } else {
        for (const item of rows) {
            const row = document.createElement("tr")
            renderRow(row, item)
            tbody.append(row)
        }
    }
    table.append(tbody)
    wrapper.append(table)
    container.append(wrapper)
}

function appendCell(row, value, className = "") {
    const cell = document.createElement("td")
    cell.textContent = value ?? ""
    if (className) cell.classList.add(...className.split(" "))
    row.append(cell)
}

function createMigrationForm(data) {
    const card = document.createElement("div")
    card.classList.add("form-card", "mt-1")

    const heading = document.createElement("h3")
    heading.textContent = "Create Migration"
    card.append(heading)

    const form = document.createElement("form")
    form.method = "POST"
    form.action = `/databases/${data.database.id}/migrations`
    const csrfInput = document.createElement("input")
    csrfInput.type = "hidden"
    csrfInput.name = "csrf_token"
    csrfInput.value = data.csrfToken
    form.append(csrfInput)

    const nameGroup = document.createElement("div")
    nameGroup.classList.add("form-group")
    const nameLabel = document.createElement("label")
    nameLabel.htmlFor = "migration_name"
    nameLabel.textContent = "Migration Name"
    const nameInput = document.createElement("input")
    nameInput.type = "text"
    nameInput.id = "migration_name"
    nameInput.name = "name"
    nameInput.required = true
    nameInput.placeholder = "e.g. 002_add_discount_column"
    nameGroup.append(nameLabel, nameInput)
    form.append(nameGroup)

    const sqlGroup = document.createElement("div")
    sqlGroup.classList.add("form-group")
    const sqlLabel = document.createElement("label")
    sqlLabel.htmlFor = "migration_sql"
    sqlLabel.textContent = "Migration SQL"
    const sqlInput = document.createElement("textarea")
    sqlInput.id = "migration_sql"
    sqlInput.classList.add("sql-editor")
    sqlInput.name = "sql"
    sqlInput.rows = 8
    sqlInput.placeholder = "ALTER TABLE orders ADD COLUMN discount NUMERIC(10, 2) DEFAULT 0.00;"
    sqlGroup.append(sqlLabel, sqlInput)
    form.append(sqlGroup)

    const actions = document.createElement("div")
    actions.classList.add("form-actions")
    const submit = document.createElement("button")
    submit.type = "submit"
    submit.classList.add("btn", "btn-primary")
    submit.textContent = "Execute Migration"
    actions.append(submit)
    form.append(actions)
    card.append(form)
    return card
}
