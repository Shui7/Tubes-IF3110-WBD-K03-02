import { createErrorDiv, setTitle } from "./components.js"
import { renderLoginView } from "./login.js"
import { apiFetch, renderLoading, validateSession } from "./utils.js"

export async function renderDatabases() {
    if (!await validateSession()) {
        await renderLoginView()
        return
    }

    setTitle("Databases")
    await renderLoading()

    const container = document.getElementById("content")
    container.replaceChildren()

    const databaseContainer = document.createElement("div")
    databaseContainer.classList.add("narrow")
    container.append(databaseContainer)

    const headerDiv = document.createElement("div")
    headerDiv.classList.add("page-header", "text-center")
    databaseContainer.append(headerDiv)

    const headerText = document.createElement("h1")
    headerText.textContent = "Databases"
    headerDiv.append(headerText)

    const errorDiv = createErrorDiv()
    databaseContainer.append(errorDiv)

    const databaseListHeader = document.createElement("h2")
    databaseListHeader.textContent = "Database List"
    databaseContainer.append(databaseListHeader)

    const databaseTableWrap = document.createElement("div")
    databaseTableWrap.classList.add("table-wrap")
    databaseContainer.append(databaseTableWrap)

    const databaseTable = document.createElement("table")
    databaseTableWrap.append(databaseTable)

    const databaseTableHead = document.createElement("thead")
    databaseTable.append(databaseTableHead)

    const databaseTableHeadRow = document.createElement("tr")
    databaseTableHead.append(databaseTableHeadRow)

    for (const [label, className] of [["Database", ""], ["Access", ""], ["", "text-right"]]) {
        const heading = document.createElement("th")
        heading.textContent = label
        if (className) heading.classList.add(className)
        databaseTableHeadRow.append(heading)
    }

    const databaseTableBody = document.createElement("tbody")
    databaseTable.append(databaseTableBody)

    try {
        const response = await apiFetch("/api/databases")
        if (response.status === 401) return
        if (!response.ok) {
            throw new Error(await response.text() || `Request failed (${response.status})`)
        }
        const data = await response.json()

        if (data.role === "Admin") {
            const manageConnectionsButton = document.createElement("a")
            manageConnectionsButton.classList.add("btn", "btn-outline")
            manageConnectionsButton.href = "/admin"
            manageConnectionsButton.textContent = "Manage connections"
            headerDiv.append(manageConnectionsButton)
        }

        if (!data.databases?.length) {
            const emptyRow = document.createElement("tr")
            const emptyCell = document.createElement("td")
            emptyCell.colSpan = 3
            emptyCell.classList.add("empty", "center")
            emptyCell.textContent = "You don't have access to any database yet. Ask an Admin to grant your group access."
            emptyRow.append(emptyCell)
            databaseTableBody.append(emptyRow)
            return
        }

        for (const database of data.databases) {
            const row = document.createElement("tr")
            const nameCell = document.createElement("td")
            nameCell.classList.add("mono")
            nameCell.textContent = database.name
            row.append(nameCell)

            const accessCell = document.createElement("td")
            const accessBadge = document.createElement("span")
            accessBadge.classList.add("badge", database.accessLevel === "edit" ? "badge-edit" : "badge-view")
            accessBadge.textContent = database.accessLevel === "edit" ? "View + Edit" : "View Only"
            accessCell.append(accessBadge)
            row.append(accessCell)

            const actionsCell = document.createElement("td")
            actionsCell.classList.add("text-right")
            const openButton = document.createElement("a")
            openButton.classList.add("btn", "btn-outline", "btn-sm")
            openButton.href = `/databases/${database.id}`
            openButton.textContent = "Open"
            actionsCell.append(openButton)
            row.append(actionsCell)
            databaseTableBody.append(row)
        }
    } catch (error) {
        errorDiv.textContent = `Unable to load databases: ${error.message}`
        errorDiv.hidden = false
    }
}