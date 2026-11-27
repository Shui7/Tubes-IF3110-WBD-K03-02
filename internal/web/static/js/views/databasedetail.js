import { createErrorDiv, createSuccessDiv, setTitle } from "./components.js"
import { renderLoginView, renderNavRight } from "./login.js"
import { apiFetch, renderLoading, validateSession } from "./utils.js"

export async function renderDatabaseDetails(databaseId) {
    if (!await validateSession()) {
        await renderLoginView()
        return
    }

    await renderLoading()
    await renderNavRight()

    const container = document.getElementById("content")
    container.replaceChildren()

    try {
        const response = await apiFetch(`/api/databases/${databaseId}${window.location.search}`)
        if (response.status === 401) return
        if (!response.ok) {
            throw new Error(await readError(response) || `Request failed (${response.status})`)
        }

        renderDatabaseDetail(container, await response.json())
    } catch (error) {
        setTitle("Databases")
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

async function readError(response) {
    const body = await response.text()
    try {
        const result = JSON.parse(body)
        return result.error || body
    } catch {
        return body
    }
}

function renderDatabaseDetail(container, data) {
    setTitle(data.database.name)

    const backLink = document.createElement("a")
    backLink.classList.add("back-link")
    backLink.href = "/databases"
    backLink.textContent = "← Back to Databases"
    container.append(backLink)

    const header = document.createElement("div")
    header.classList.add("page-header")
    const title = document.createElement("h1")
    title.textContent = data.database.name
    const accessBadge = document.createElement("span")
    const canEdit = data.database.accessLevel === "edit"
    accessBadge.classList.add("badge", canEdit ? "badge-edit" : "badge-view")
    accessBadge.textContent = canEdit ? "View + Edit" : "View Only"
    header.append(title, accessBadge)
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
    const requestedTab = window.location.hash.slice(1) || params.get("tab") || data.activeTab
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
    appendHeading(schemaSection, "h2", "Table: ", data.selectedTable, "mono")
    appendDataTable(
        schemaSection,
        ["Column", "Type", "Nullable", "Default", "Key"],
        data.columns,
        "No column information available.",
        (row, column) => {
            appendCell(row, column.name, "mono font-bold")
            appendCell(row, column.type, "mono")
            appendCell(row, column.nullable ? "YES" : "NO")
            appendCell(row, column.default || "-", column.default ? "mono" : "null")
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
        },
    )

    appendSimpleHeading(schemaSection, "h3", "Constraints")
    appendDataTable(
        schemaSection,
        ["Constraint Name", "Type", "Definition"],
        data.constraints,
        "No constraints found.",
        (row, constraint) => {
            appendCell(row, constraint.name, "mono")
            const typeCell = document.createElement("td")
            const badge = document.createElement("span")
            badge.classList.add("badge")
            badge.textContent = constraint.type
            typeCell.append(badge)
            row.append(typeCell)
            appendCell(row, constraint.definition, "mono")
        },
    )

    appendSimpleHeading(schemaSection, "h3", "Indexes")
    appendDataTable(
        schemaSection,
        ["Index Name", "Columns", "Type", "Unique"],
        data.indexes,
        "No indexes found.",
        (row, index) => {
            appendCell(row, index.name, "mono")
            appendCell(row, index.columns, "mono")
            appendCell(row, index.type)
            appendCell(row, index.unique ? "YES" : "NO")
        },
    )
    container.append(schemaSection)

    const dataSection = createTabSection("data", activeTab)
    appendTableSelector(dataSection, data.tables, data.selectedTable, activeTab)
    appendHeading(dataSection, "h2", "Data: ", data.selectedTable, "mono")
    appendDataTable(
        dataSection,
        data.dataHeaders,
        data.dataRows,
        "No rows found in this table.",
        (row, values) => {
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
        },
        "table-wrap data",
    )
    appendPagination(dataSection)
    container.append(dataSection)

    const migrationsSection = createTabSection("migrations", activeTab)
    appendMigrationHistory(migrationsSection, data)
    appendChangeRequests(migrationsSection, data)
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

function appendHeading(container, tagName, prefix, value, className = "") {
    const heading = document.createElement(tagName)
    heading.append(document.createTextNode(prefix))
    const valueElement = document.createElement("span")
    valueElement.textContent = value || ""
    if (className) valueElement.classList.add(className)
    heading.append(valueElement)
    container.append(heading)
}

function appendSimpleHeading(container, tagName, text) {
    const heading = document.createElement(tagName)
    heading.textContent = text
    container.append(heading)
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

function appendPagination(container) {
    const pagination = document.createElement("div")
    pagination.classList.add("pagination")
    const previous = document.createElement("button")
    previous.type = "button"
    previous.classList.add("btn", "btn-outline", "btn-sm")
    previous.disabled = true
    previous.textContent = "Previous"
    const page = document.createElement("span")
    page.textContent = "Page 1 of 1"
    const next = document.createElement("button")
    next.type = "button"
    next.classList.add("btn", "btn-outline", "btn-sm")
    next.disabled = true
    next.textContent = "Next"
    pagination.append(previous, page, next)
    container.append(pagination)
}

function appendMigrationHistory(container, data) {
    appendSimpleHeading(container, "h2", "Migration History")

    const filterGroup = document.createElement("div")
    filterGroup.classList.add("form-group")
    const label = document.createElement("label")
    const checkbox = document.createElement("input")
    checkbox.type = "checkbox"
    checkbox.id = "hide-rollback"
    label.append(checkbox, document.createTextNode(" Hide rollback"))
    const hint = document.createElement("p")
    hint.classList.add("muted")
    hint.textContent = "Hides a successful rollback and the exact execution it reverted. Failed attempts and later forward executions remain visible."
    filterGroup.append(label, hint)
    container.append(filterGroup)

    const visibilityError = createErrorDiv()
    container.append(visibilityError)
    const wrapper = document.createElement("div")
    wrapper.classList.add("table-wrap")
    const table = document.createElement("table")
    table.append(createTableHead(["Version", "Action / Execution", "Description", "Executed By", "Executed At", "Status"]))
    const tbody = document.createElement("tbody")
    const historyRows = []

    if (!data.migrations?.length) {
        const row = document.createElement("tr")
        const emptyCell = document.createElement("td")
        emptyCell.colSpan = 6
        emptyCell.classList.add("empty", "center")
        emptyCell.textContent = "No migration history recorded."
        row.append(emptyCell)
        tbody.append(row)
    } else {
        for (const migration of data.migrations) {
            const historyRow = document.createElement("tr")
            const detailRow = document.createElement("tr")
            for (const row of [historyRow, detailRow]) {
                row.classList.add("migration-history-row")
                row.dataset.executionId = String(migration.executionId)
                row.dataset.action = migration.action
                row.dataset.status = migration.status
                row.dataset.targetExecutionId = String(migration.targetExecutionId || "")
                historyRows.push(row)
            }

            appendCell(historyRow, migration.version, "mono font-bold")
            const actionCell = document.createElement("td")
            const actionBadge = document.createElement("span")
            actionBadge.classList.add("badge")
            actionBadge.textContent = `${migration.action} #${migration.executionId}`
            actionCell.append(actionBadge)
            if (migration.targetExecutionId) {
                const target = document.createElement("span")
                target.classList.add("muted")
                target.append(document.createElement("br"), document.createTextNode(`targets #${migration.targetExecutionId}`))
                actionCell.append(target)
            }
            historyRow.append(actionCell)
            appendCell(historyRow, migration.name)
            appendCell(historyRow, migration.executedBy)
            appendCell(historyRow, migration.executedAt, "mono")
            const statusCell = document.createElement("td")
            const statusBadge = document.createElement("span")
            statusBadge.classList.add("badge", migration.status === "success" ? "badge-success" : "badge-failed")
            statusBadge.textContent = migration.status === "success" ? "Success" : "Failed"
            statusCell.append(statusBadge)
            historyRow.append(statusCell)

            const detailsCell = document.createElement("td")
            detailsCell.colSpan = 6
            const canEdit = data.database.accessLevel === "edit"
            if (canEdit && migration.canRollback) {
                detailsCell.append(createExecutionForm(data, migration.executionId, "rollback"))
            }
            if (canEdit && migration.canForward) {
                detailsCell.append(createExecutionForm(data, migration.executionId, "forward"))
            }
            detailsCell.append(createMigrationDefinition(migration))
            detailRow.append(detailsCell)
            tbody.append(historyRow, detailRow)
        }
    }

    table.append(tbody)
    wrapper.append(table)
    container.append(wrapper)

    checkbox.addEventListener("change", async () => {
        try {
            const query = new URLSearchParams({ hide_rollback: String(checkbox.checked) })
            const response = await apiFetch(`/api/databases/${data.database.id}/migrations/visibility?${query}`)
            if (response.status === 401) return
            if (!response.ok) {
                throw new Error(await readError(response) || `Migration visibility request failed (${response.status})`)
            }
            const result = await response.json()
            if (!Array.isArray(result.hidden_execution_ids)) {
                throw new TypeError("Invalid migration visibility response")
            }
            const hiddenExecutionIDs = new Set(result.hidden_execution_ids.map(String))
            for (const row of historyRows) {
                row.hidden = hiddenExecutionIDs.has(row.dataset.executionId)
            }
            visibilityError.hidden = true
        } catch (error) {
            visibilityError.textContent = `Unable to update migration visibility: ${error.message}`
            visibilityError.hidden = false
        }
    })
}

function createTableHead(headers) {
    const thead = document.createElement("thead")
    const row = document.createElement("tr")
    for (const header of headers) {
        const cell = document.createElement("th")
        cell.textContent = header
        row.append(cell)
    }
    thead.append(row)
    return thead
}

function createExecutionForm(data, executionID, action) {
    const form = document.createElement("form")
    form.method = "POST"
    form.action = `/databases/${data.database.id}/migrations/${action}`
    form.classList.add("inline-form")
    appendHiddenInput(form, "csrf_token", data.csrfToken)
    appendHiddenInput(form, "execution_id", executionID)

    const button = document.createElement("button")
    button.type = "submit"
    button.classList.add("btn", "btn-outline", "btn-sm")
    button.textContent = action[0].toUpperCase() + action.slice(1)
    form.append(button)
    return form
}

function createMigrationDefinition(migration) {
    const details = document.createElement("details")
    const summary = document.createElement("summary")
    summary.textContent = "Migration definition"
    details.append(summary)

    if (migration.migrationId) appendDefinitionLine(details, "Migration request:", `#${migration.migrationId}`)
    if (migration.jobId) {
        const line = document.createElement("p")
        line.append(document.createTextNode("Job: "))
        const link = document.createElement("a")
        link.href = `/api/jobs/${encodeURIComponent(migration.jobId)}`
        link.textContent = migration.jobId
        line.append(link)
        details.append(line)
    }
    if (migration.requestId) appendDefinitionLine(details, "Request ID:", migration.requestId)
    if (migration.revision) appendDefinitionLine(details, "Revision:", migration.revision)
    appendCodeBlock(details, "SQL executed by this attempt", migration.executedSQL)
    appendCodeBlock(details, "SQL up", migration.sql)
    if (migration.irreversible) {
        appendDefinitionLine(details, "Irreversible:", migration.irreversibleReason)
    } else {
        appendCodeBlock(details, "SQL down", migration.sqlDown)
    }
    return details
}

function appendDefinitionLine(container, label, value) {
    const line = document.createElement("p")
    const strong = document.createElement("strong")
    strong.textContent = label
    line.append(strong, document.createTextNode(` ${value ?? ""}`))
    container.append(line)
}

function appendCodeBlock(container, label, value) {
    const paragraph = document.createElement("p")
    const strong = document.createElement("strong")
    strong.textContent = label
    paragraph.append(strong)
    const code = document.createElement("pre")
    code.classList.add("mono")
    code.textContent = value || ""
    container.append(paragraph, code)
}

function appendChangeRequests(container, data) {
    appendSimpleHeading(container, "h3", "Migration requests and approvals")
    const wrapper = document.createElement("div")
    wrapper.classList.add("table-wrap")
    const table = document.createElement("table")
    table.append(createTableHead(["Request", "Action", "Requester", "Status", "Required approvals", "Error"]))
    const tbody = document.createElement("tbody")

    if (!data.changeRequests?.length) {
        const row = document.createElement("tr")
        const emptyCell = document.createElement("td")
        emptyCell.colSpan = 6
        emptyCell.classList.add("empty", "center")
        emptyCell.textContent = "No migration change requests."
        row.append(emptyCell)
        tbody.append(row)
    } else {
        for (const request of data.changeRequests) {
            const row = document.createElement("tr")
            const requestCell = document.createElement("td")
            requestCell.classList.add("mono")
            requestCell.append(document.createTextNode(`#${request.id}`))
            const createdAt = document.createElement("small")
            createdAt.textContent = request.createdAt
            requestCell.append(document.createElement("br"), createdAt)
            row.append(requestCell)

            appendCell(row, `${request.action}${request.targetExecutionId ? ` #${request.targetExecutionId}` : ""}`)
            appendCell(row, request.requesterName)
            const statusCell = document.createElement("td")
            const statusBadge = document.createElement("span")
            statusBadge.classList.add("badge")
            statusBadge.textContent = request.status
            statusCell.append(statusBadge)
            row.append(statusCell)
            row.append(createApprovalsCell(data.database.id, request, data.csrfToken))
            appendCell(row, request.error)
            tbody.append(row)

            const detailsRow = document.createElement("tr")
            const detailsCell = document.createElement("td")
            detailsCell.colSpan = 6
            detailsCell.append(createRequestDetails(request))
            detailsRow.append(detailsCell)
            tbody.append(detailsRow)
        }
    }

    table.append(tbody)
    wrapper.append(table)
    container.append(wrapper)
}

function createApprovalsCell(databaseID, request, csrfToken) {
    const cell = document.createElement("td")
    for (const approval of request.approvals || []) {
        const entry = document.createElement("div")
        entry.append(document.createTextNode(`${approval.groupName} — ${approval.leaderName}: `))
        const decision = document.createElement("strong")
        decision.textContent = approval.decision
        entry.append(decision)
        cell.append(entry)

        if (approval.canRespond) {
            const form = document.createElement("form")
            form.method = "POST"
            form.action = `/databases/${databaseID}/migrations/requests/${request.id}/decision`
            form.classList.add("inline-form")
            appendHiddenInput(form, "csrf_token", csrfToken)
            for (const [decisionValue, label, className] of [
                ["approved", "Approve", "btn-primary"],
                ["rejected", "Reject", "btn-danger"],
            ]) {
                const button = document.createElement("button")
                button.type = "submit"
                button.name = "decision"
                button.value = decisionValue
                button.classList.add("btn", className, "btn-sm")
                button.textContent = label
                form.append(button)
            }
            cell.append(form)
        }
    }
    return cell
}

function createRequestDetails(request) {
    const details = document.createElement("details")
    const summary = document.createElement("summary")
    summary.textContent = "Workflow history and request details"
    details.append(summary)

    const events = document.createElement("ol")
    for (const event of request.events || []) {
        const item = document.createElement("li")
        const status = document.createElement("strong")
        status.textContent = event.status
        item.append(status, document.createTextNode(` — ${event.actorName} — ${event.occurredAt}: ${event.note}`))
        events.append(item)
    }
    details.append(events)

    if (request.action === "up") {
        appendDefinitionLine(details, "Mode:", request.mode + (request.mode === "batch" ? ` (size ${request.batchSize})` : ""))
        if (request.mode === "batch" && request.batchTable) {
            appendDefinitionLine(details, "Batch table:", `${request.batchTable}, cursor ${request.batchColumn}`)
        }
        appendCodeBlock(details, "SQL up", request.sqlUp)
        if (request.irreversible) {
            appendDefinitionLine(details, "Irreversible:", request.irreversibleReason)
        } else {
            appendCodeBlock(details, "SQL down", request.sqlDown)
        }
    } else if (request.action === "rollback") {
        appendCodeBlock(details, "SQL down to execute", request.sqlDown)
    } else {
        appendCodeBlock(details, "SQL up to execute", request.sqlUp)
    }
    return details
}

function createMigrationForm(data) {
    const card = document.createElement("div")
    card.classList.add("form-card", "mt-1")
    appendSimpleHeading(card, "h3", "Submit Migration for Approval")

    const form = document.createElement("form")
    form.method = "POST"
    form.action = `/databases/${data.database.id}/migrations`
    appendHiddenInput(form, "csrf_token", data.csrfToken)

    const modeGroup = createFormGroup()
    appendLabel(modeGroup, "migration_mode", "Execution mode")
    const mode = document.createElement("select")
    mode.id = "migration_mode"
    mode.name = "mode"
    for (const [value, label] of [["plain", "Plain migration"], ["batch", "Batch migration"]]) {
        const option = document.createElement("option")
        option.value = value
        option.textContent = label
        mode.append(option)
    }
    modeGroup.append(mode)
    form.append(modeGroup)

    const batchOptions = createFormGroup()
    batchOptions.id = "batch-options"
    batchOptions.hidden = true
    appendLabel(batchOptions, "batch_size", "Batch size")
    const batchSize = document.createElement("input")
    batchSize.id = "batch_size"
    batchSize.name = "batch_size"
    batchSize.type = "number"
    batchSize.min = "1"
    batchSize.max = "100000"
    batchSize.value = "1000"
    batchOptions.append(batchSize)
    appendLabel(batchOptions, "batch_table", "Batch table (optional, for batch-by-attribute)")
    const batchTable = document.createElement("input")
    batchTable.id = "batch_table"
    batchTable.name = "batch_table"
    batchTable.placeholder = "public.big_items"
    batchOptions.append(batchTable)
    appendLabel(batchOptions, "batch_column", "Unique cursor column")
    const batchColumn = document.createElement("input")
    batchColumn.id = "batch_column"
    batchColumn.name = "batch_column"
    batchColumn.placeholder = "id"
    batchOptions.append(batchColumn)
    const batchHint = document.createElement("p")
    batchHint.classList.add("muted")
    batchHint.textContent = "For attribute mode, use {{batch_filter}} in SQL up. Otherwise write a cursor query using $1 (last cursor) and $2 (batch size)."
    batchOptions.append(batchHint)
    form.append(batchOptions)

    const sqlUpGroup = createFormGroup()
    appendLabel(sqlUpGroup, "migration_sql_up", "SQL up")
    const sqlUp = document.createElement("textarea")
    sqlUp.id = "migration_sql_up"
    sqlUp.classList.add("sql-editor")
    sqlUp.name = "sql_up"
    sqlUp.rows = 8
    sqlUp.required = true
    sqlUp.placeholder = "ALTER TABLE orders ADD COLUMN discount NUMERIC(10, 2) DEFAULT 0.00;"
    sqlUpGroup.append(sqlUp)
    form.append(sqlUpGroup)

    const sqlDownGroup = createFormGroup()
    sqlDownGroup.id = "sql-down-group"
    appendLabel(sqlDownGroup, "migration_sql_down", "SQL down")
    const sqlDown = document.createElement("textarea")
    sqlDown.id = "migration_sql_down"
    sqlDown.classList.add("sql-editor")
    sqlDown.name = "sql_down"
    sqlDown.rows = 8
    sqlDown.placeholder = "ALTER TABLE orders DROP COLUMN discount;"
    sqlDownGroup.append(sqlDown)
    form.append(sqlDownGroup)

    const irreversibleGroup = createFormGroup()
    const irreversibleLabel = document.createElement("label")
    const irreversible = document.createElement("input")
    irreversible.type = "checkbox"
    irreversible.id = "migration_irreversible"
    irreversible.name = "irreversible"
    irreversibleLabel.append(irreversible, document.createTextNode(" Mark as irreversible"))
    irreversibleGroup.append(irreversibleLabel)
    form.append(irreversibleGroup)

    const reasonGroup = createFormGroup()
    reasonGroup.id = "irreversible-reason-group"
    reasonGroup.hidden = true
    appendLabel(reasonGroup, "irreversible_reason", "Reason")
    const reason = document.createElement("textarea")
    reason.id = "irreversible_reason"
    reason.name = "irreversible_reason"
    reason.rows = 3
    reason.placeholder = "Explain why this migration cannot be safely reversed."
    reasonGroup.append(reason)
    form.append(reasonGroup)

    mode.addEventListener("change", () => {
        batchOptions.hidden = mode.value !== "batch"
    })
    irreversible.addEventListener("change", () => {
        sqlDownGroup.hidden = irreversible.checked
        sqlDown.required = !irreversible.checked
        reasonGroup.hidden = !irreversible.checked
        reason.required = irreversible.checked
    })
    sqlDown.required = true

    const actions = document.createElement("div")
    actions.classList.add("form-actions")
    const submit = document.createElement("button")
    submit.type = "submit"
    submit.classList.add("btn", "btn-primary")
    submit.textContent = "Submit for Approval"
    actions.append(submit)
    form.append(actions)
    card.append(form)
    return card
}

function createFormGroup() {
    const group = document.createElement("div")
    group.classList.add("form-group")
    return group
}

function appendLabel(container, id, text) {
    const label = document.createElement("label")
    label.htmlFor = id
    label.textContent = text
    container.append(label)
}

function appendHiddenInput(form, name, value) {
    const input = document.createElement("input")
    input.type = "hidden"
    input.name = name
    input.value = value ?? ""
    form.append(input)
}
