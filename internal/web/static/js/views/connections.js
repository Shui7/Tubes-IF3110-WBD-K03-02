import { createErrorDiv, createSuccessDiv } from "./components.js"
import { apiFetch, fetchJsonArray } from "./utils.js"

export async function renderConnectionsTab() {
    const container = document.getElementById("content")

    let connectionsTab = document.createElement("section")
    connectionsTab.id = "tab-connections"
    connectionsTab.classList.add("tab-content")
    container.append(connectionsTab)

    let headerDiv = document.createElement("div")
    headerDiv.classList.add("page-header", "text-center")
    connectionsTab.append(headerDiv)
    
    let headerText = document.createElement("h2")
    headerText.appendChild(document.createTextNode("PostgreSQL Database Connections"))
    headerDiv.append(headerText)

    let errorDiv = createErrorDiv()
    let successDiv = createSuccessDiv()
    connectionsTab.append(errorDiv, successDiv)

    let connectionsTableWrap = document.createElement("div")
    connectionsTableWrap.classList.add("table-wrap")
    connectionsTab.append(connectionsTableWrap)

    let connectionsTable = document.createElement("table")
    connectionsTableWrap.append(connectionsTable)

    let tableHead = document.createElement("thead")
    connectionsTable.append(tableHead)

    let tableHeadRow = document.createElement("tr")
    tableHead.append(tableHeadRow)

    let tableHeadDatabase = document.createElement("th")
    tableHeadDatabase.textContent = "Database"
    tableHeadRow.append(tableHeadDatabase)

    let tableHeadHostPort = document.createElement("th")
    tableHeadHostPort.textContent = "Host:Port"
    tableHeadRow.append(tableHeadHostPort)

    let tableHeadUser = document.createElement("th")
    tableHeadUser.textContent = "DB User"
    tableHeadRow.append(tableHeadUser)

    let tableHeadStatus = document.createElement("th")
    tableHeadStatus.textContent = "Status"
    tableHeadRow.append(tableHeadStatus)

    let tableHeadActions = document.createElement("th")
    tableHeadActions.textContent = "Actions"
    tableHeadActions.classList.add("text-right")
    tableHeadRow.append(tableHeadActions)

    let tableBody = document.createElement("tbody")
    connectionsTable.append(tableBody)

    let connections = await fetchConnections()
    for (let connection of connections) {
        let tableRow = document.createElement("tr")
        tableBody.append(tableRow)

        let tableDataDatabase = document.createElement("td")
        tableDataDatabase.classList.add("mono")
        tableDataDatabase.textContent = connection.database
        tableRow.append(tableDataDatabase)

        let tableDataHostPort = document.createElement("td")
        tableDataHostPort.classList.add("mono")
        tableDataHostPort.textContent = connection.host + ":" + connection.port
        tableRow.append(tableDataHostPort)

        let tableDataUser = document.createElement("td")
        tableDataUser.classList.add("mono")
        tableDataUser.textContent = connection.user
        tableRow.append(tableDataUser)

        switch (connection.status) {
            case "Connected":
                var tableDataStatus = document.createElement("td")
                var statusBadge = document.createElement("span")
                statusBadge.classList.add("badge", "badge-success")
                statusBadge.textContent = "Connected"
                tableDataStatus.append(statusBadge)
                tableRow.append(tableDataStatus)
                break
            case "Failed":
                var tableDataStatus = document.createElement("td")
                var statusBadge = document.createElement("span")
                statusBadge.classList.add("badge", "badge-failed")
                statusBadge.textContent = "Connection Failed"
                tableDataStatus.append(statusBadge)
                tableRow.append(tableDataStatus)
                break
            default:
                var tableDataStatus = document.createElement("td")
                var statusBadge = document.createElement("span")
                statusBadge.classList.add("badge", "badge-pending")
                statusBadge.textContent = connection.status
                tableDataStatus.append(statusBadge)
                tableRow.append(tableDataStatus)
        }

        let tableDataActions = document.createElement("td")
        tableDataActions.classList.add("text-right")
        tableRow.append(tableDataActions)

        let deleteButton = document.createElement("button")
        deleteButton.classList.add("btn", "btn-danger", "btn-sm")
        deleteButton.textContent = "Delete"
        deleteButton.addEventListener("click", async function (event) {
            event.preventDefault()
            const response = await apiFetch('/api/admin/connections/delete', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    id: connection.id
                })
            })
            if (response.ok) {
                renderConnectionsTab()
            }
        })
        tableDataActions.append(deleteButton)
    }
    if (connections.length === 0) {
        let tableRow = document.createElement("tr")
        let tableData = document.createElement("td")
        tableData.colSpan = 6
        tableData.classList.add("empty", "center")
        tableData.textContent = "No database connections configured yet."
        tableRow.append(tableData)
        tableBody.append(tableRow)
    }

    let addConnectionFormCard = document.createElement("div")
    addConnectionFormCard.classList.add("form-card", "mt-1")
    connectionsTab.append(addConnectionFormCard)

    let addConnectionHeader = document.createElement("h3")
    addConnectionHeader.textContent = "Add PostgreSQL Connection"
    addConnectionFormCard.append(addConnectionHeader)

    let addConnectionForm = document.createElement("form")
    addConnectionForm.method = "POST"
    addConnectionForm.action = "/admin/connections/add"
    addConnectionFormCard.append(addConnectionForm)

    let databaseInputGroup = document.createElement("div")
    databaseInputGroup.classList.add("form-group")
    addConnectionForm.append(databaseInputGroup)

    let databaseLabel = document.createElement("label")
    databaseLabel.htmlFor = "conn_database"
    databaseLabel.textContent = "Database Name"
    databaseInputGroup.append(databaseLabel)

    let databaseInput = document.createElement("input")
    databaseInput.type = "text"
    databaseInput.id = "conn_database"
    databaseInput.name = "database"
    databaseInput.required = true
    databaseInput.placeholder = "e.g. film, airport"
    databaseInputGroup.append(databaseInput)

    let hostPortGroup = document.createElement("div")
    hostPortGroup.style = "display: grid; grid-template-columns: 3fr 1fr; gap: 1rem;"
    addConnectionForm.append(hostPortGroup)

    let hostInputGroup = document.createElement("div")
    hostPortGroup.append(hostInputGroup)
    let hostInputLabel = document.createElement("label")
    hostInputLabel.htmlFor = "conn_host"
    hostInputLabel.textContent = "Host"
    hostInputGroup.append(hostInputLabel)

    let hostInput = document.createElement("input")
    hostInput.type = "text"
    hostInput.id = "conn_host"
    hostInput.name = "host"
    hostInput.required = true
    hostInput.placeholder = "dbsamples-1"
    hostInputGroup.append(hostInput)

    let portInputGroup = document.createElement("div")
    hostPortGroup.append(portInputGroup)
    let portInputLabel = document.createElement("label")
    portInputLabel.htmlFor = "conn_port"
    portInputLabel.textContent = "port"
    portInputGroup.append(portInputLabel)

    let portInput = document.createElement("input")
    portInput.type = "number"
    portInput.id = "conn_port"
    portInput.name = "port"
    portInput.required = true
    portInput.value = 5432
    portInput.placeholder = "5432"
    portInputGroup.append(portInput)

    let databaseGroup = document.createElement("div")
    databaseGroup.style = "display: grid; grid-template-columns: 1fr 1fr; gap: 1rem;"
    addConnectionForm.append(databaseGroup)

    let databaseUsernameInputGroup = document.createElement("div")
    databaseGroup.append(databaseUsernameInputGroup)
    let databaseUsernameLabel = document.createElement("label")
    databaseUsernameLabel.htmlFor = "conn_user"
    databaseUsernameLabel.textContent = "Database Username"
    databaseUsernameInputGroup.append(databaseUsernameLabel)

    let databaseUsernameInput = document.createElement("input")
    databaseUsernameInput.type = "text"
    databaseUsernameInput.id = "conn_user"
    databaseUsernameInput.name = "user"
    databaseUsernameInput.required = true
    databaseUsernameInput.placeholder = "postgres"
    databaseUsernameInputGroup.append(databaseUsernameInput)

    let databasePasswordInputGroup = document.createElement("div")
    databaseGroup.append(databasePasswordInputGroup)
    let databasePasswordLabel = document.createElement("label")
    databasePasswordLabel.htmlFor = "conn_password"
    databasePasswordLabel.textContent = "Database Password"
    databasePasswordInputGroup.append(databasePasswordLabel)

    let databasePasswordInput = document.createElement("input")
    databasePasswordInput.type = "password"
    databasePasswordInput.id = "conn_password"
    databasePasswordInput.name = "password"
    databasePasswordInput.required = true
    databasePasswordInput.placeholder = "••••••••"
    databasePasswordInputGroup.append(databasePasswordInput)

    let formActionsDiv = document.createElement("div")
    formActionsDiv.classList.add("form-actions")
    addConnectionForm.append(formActionsDiv)
    let addConnectionButton = document.createElement("button")
    addConnectionButton.type = "submit"
    addConnectionButton.classList.add("btn", "btn-primary")
    addConnectionButton.textContent = "Add Database Connection"
    formActionsDiv.append(addConnectionButton)

    addConnectionForm.addEventListener("submit", async function (event) {
        event.preventDefault()
        errorDiv.hidden = true
        successDiv.hidden = true

        const host = hostInput.value.trim();
        const port = portInput.value.trim();
        const database = databaseUsernameInput.value.trim();
        const password = databasePasswordInput.value.trim();

        if (!host || !port || !database || !password) {
            errorDiv.textContent = "All fields are required."
            errorDiv.hidden = false
            return
        }

        const response = await apiFetch('/api/admin/connections/add', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                host: host,
                port: port,
                database: database,
                password: password
            })
        })
        
        if (!response.ok) {
            const errorMessage = await response.text();
            errorDiv.textContent = errorMessage || "Failed to add database connection."
            errorDiv.hidden = false
        } else {
            successDiv.textContent = "Database connection added successfully."
            successDiv.hidden = false
            renderConnectionsTab()
        }
    })
}

export async function fetchConnections() {
    return fetchJsonArray("/api/admin/connections")
}