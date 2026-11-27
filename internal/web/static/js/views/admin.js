import { renderConnectionsTab } from "./connections"
import { apiFetch } from "./utils"

export async function renderAdminView() {
    if (await validateSession()) {
        await renderHome()
        return
    }

    setTitle("Platform Administration")
    renderLoading()

    const container = document.getElementById("content")
    container.innerHTML = "";    

    let titleDiv = document.createElement("div")
    titleDiv.classList.add("page-header", "text-center")
    titleDiv.append(document.createElement("h1").appendChild(document.createTextNode("Platform Administration")))
    container.append(titleDiv)

    const response = await fetch('/api/auth/name-role');
    const data = await response.json().catch({"name": "", "role": ""})

    if (role !== "Admin") {
        let errorDiv = createErrorDiv("You do not have permission to access this page.")
        container.append(errorDiv)
        return
    }

    let adminTabs = document.createElement("div")
    adminTabs.classList.add("tabs")

    let connectionsTab = document.createElement("a")
    connectionsTab.classList.add("tab-link")
    connectionsTab.textContent = "Database Conections"
    connectionsTab.dataTab = "connections"
    connectionsTab.onclick(async (event) => {
        event.preventDefault()
        connectionsTab.classList.add("active")
        groupsTab.classList.remove("active")
        usersTab.classList.remove("active")
        renderConnectionsTab()
    })

    let groupsTab = document.createElement("a")
    groupsTab.classList.add("tab-link")
    groupsTab.textContent = "User Groups"
    groupsTab.dataTab = "groups"
    groupsTab.onclick(async (event) => {
        event.preventDefault()
        connectionsTab.classList.remove("active")
        groupsTab.classList.add("active")
        usersTab.classList.remove("active")
        renderGroupsTab()
    })

    let usersTab = document.createElement("a")
    usersTab.classList.add("tab-link")
    usersTab.textContent = "Users"
    usersTab.dataTab = "users"
    usersTab.onclick(async (event) => {
        event.preventDefault()
        connectionsTab.classList.remove("active")
        groupsTab.classList.remove("active")
        usersTab.classList.add("active")
        renderUsersTab()
    })

    adminTabs.append(connectionsTab, groupsTab, usersTab)

    let activeTab = window.location.hash.slice(1) || "connections"
    if (activeTab === "connections") {
        connectionsTab.classList.add("active")
        renderConnectionsTab()
    } else if (activeTab === "groups") { 
        groupsTab.classList.add("active")
        renderGroupsTab()
    } else if (activeTab === "users") {
        usersTab.classList.add("active")
        renderUsersTab()
    }
}