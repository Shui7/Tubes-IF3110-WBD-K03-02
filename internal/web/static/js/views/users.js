import { fetchGroups } from "./groups.js"
import { apiFetch } from "./utils.js"

export async function renderUsersTab() {
    const container = document.getElementById("content")

    let usersTab = document.createElement("section")
    usersTab.id = "tab-users"
    usersTab.classList.add("tab-content")
    container.append(usersTab)

    let headerDiv = document.createElement("div")
    headerDiv.classList.add("page-header", "text-center")
    headerDiv.append(document.createElement("h2").appendChild(document.createTextNode("Platform Users")))
    usersTab.append(headerDiv)

    let errorDiv = createErrorDiv()
    let successDiv = createSuccessDiv()
    usersTab.append(errorDiv, successDiv)

    let usersTableWrap = document.createElement("div")
    usersTableWrap.classList.add("table-wrap")
    usersTab.append(usersTableWrap)

    let usersTable = document.createElement("table")
    usersTableWrap.append(usersTable)

    let tableHead = document.createElement("thead")
    usersTable.append(tableHead)

    let tableHeadRow = document.createElement("tr")
    tableHead.append(tableHeadRow)

    let tableHeadDatabase = document.createElement("th")
    tableHeadDatabase.textContent = "Database"
    tableHeadRow.append(tableHeadDatabase)

    let tableHeadUsername = document.createElement("th")
    tableHeadUsername.textContent = "Username"
    tableHeadRow.append(tableHeadUsername)

    let tableHeadName = document.createElement("th")
    tableHeadName.textContent = "Name"
    tableHeadRow.append(tableHeadName)

    let tableHeadRole = document.createElement("th")
    tableHeadRole.textContent = "Role"
    tableHeadRow.append(tableHeadRole)

    let tableHeadAssignedGroups = document.createElement("th")
    tableHeadAssignedGroups.textContent = "Assigned Groups"
    tableHeadRow.append(tableHeadAssignedGroups)

    let tableHeadActions = document.createElement("th")
    tableHeadActions.textContent = "Actions"
    tableHeadActions.classList.add("text-right")
    tableHeadRow.append(tableHeadActions)

    let tableBody = document.createElement("tbody")
    usersTable.append(tableBody)

    let users = await fetchUsers()
    for (let user of users) {
        let tableRow = document.createElement("tr")
        tableBody.append(tableRow)

        let tableDataUsername = document.createElement("td")
        tableDataUsername.classList.add("mono", "font-bold")
        let usernameStrong = document.createElement("strong")
        usernameStrong.textContent = user.username
        tableDataUsername.append(usernameStrong)
        tableRow.append(tableDataUsername)

        let tableDataName = document.createElement("td")
        tableDataName.classList.add("mono")
        tableDataName.textContent = user.name
        tableRow.append(tableDataName)

        let tableDataRole = document.createElement("td")
        tableRow.append(tableDataRole)

        let roleSpan = document.createElement("span")
        if (user.role === "Admin") {
            roleSpan.classList.add("badge", "badge-edit")
            roleSpan.textContent = "Admin"
        } else {
            roleSpan.classList.add("badge", "badge-view")
            roleSpan.textContent = "User"
        }
        tableDataRole.append(roleSpan)

        let tableDataGroups = document.createElement("td")
        tableRow.append(tableDataGroups)
        for (let group of user.groups) {
            let groupSpan = document.createElement("span")
            groupSpan.classList.add("badge")
            groupSpan.textContent = group
            tableDataGroups.append(groupSpan)
        }

        let tableDataActions = document.createElement("td")
        tableDataActions.classList.add("text-right")
        tableRow.append(tableDataActions)

        let deleteButton = document.createElement("button")
        deleteButton.classList.add("btn", "btn-danger", "btn-sm")
        deleteButton.textContent = "Delete"
        deleteButton.addEventListener("click", async function (event) {
            event.preventDefault()
            const response = await apiFetch('/api/admin/users/delete', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    id: user.id
                })
            })
            if (response.ok) {
                renderUsersTab()
            }
        })
        tableDataActions.append(deleteButton)
    }
    if (users.length === 0) {
        let tableRow = document.createElement("tr")
        let tableData = document.createElement("td")
        tableData.colSpan = 5
        tableData.classList.add("empty", "center")
        tableData.textContent = "No users found."
        tableRow.append(tableData)
        tableBody.append(tableRow)
    }

    let addUserFormCard = document.createElement("div")
    addUserFormCard.classList.add("form-card", "mt-1")
    usersTab.append(addUserFormCard)

    let addUserHeader = document.createElement("h3")
    addUserHeader.textContent = "Create New User"
    addUserFormCard.append(addUserHeader)

    let addUserForm = document.createElement("form")
    addUserForm.method = "POST"
    addUserForm.action = "/admin/users/add"
    addUserFormCard.append(addUserForm)

    let nameGridDiv = document.createElement("div")
    nameGridDiv.style = "display: grid; grid-template-columns: 1fr 1fr 1fr; gap: 1rem;"
    addUserForm.append(nameGridDiv)

    let usernameInputGroup = document.createElement("div")
    usernameInputGroup.classList.add("form-group")
    nameGridDiv.append(usernameInputGroup)

    let usernameLabel = document.createElement("label")
    usernameLabel.htmlFor = "user_username"
    usernameLabel.textContent = "Username"
    usernameInputGroup.append(usernameLabel)

    let usernameInput = document.createElement("input")
    usernameInput.type = "text"
    usernameInput.id = "user_username"
    usernameInput.name = "username"
    usernameInput.required = true
    usernameInput.autocomplete = "username"
    usernameInput.placeholder = "johnuser"
    usernameInputGroup.append(usernameInput)

    let fullnameInputGroup = document.createElement("div")
    fullnameInputGroup.classList.add("form-group")
    nameGridDiv.append(fullnameInputGroup)

    let fullnameLabel = document.createElement("label")
    fullnameLabel.htmlFor = "user_fullname"
    fullnameLabel.textContent = "Full Name"
    fullnameInputGroup.append(fullnameLabel)

    let fullnameInput = document.createElement("input")
    fullnameInput.type = "text"
    fullnameInput.id = "user_fullname"
    fullnameInput.name = "fullname"
    fullnameInput.required = true
    fullnameInput.placeholder = "John User"
    fullnameInputGroup.append(fullnameInput)

    let passwordRoleGridDiv = document.createElement("div")
    passwordRoleGridDiv.style = "display: grid; grid-template-columns: 1fr 1fr 1fr; gap: 1rem;"
    addUserForm.append(passwordRoleGridDiv)

    let passwordGroup = document.createElement("div")
    passwordGroup.classList.add("form-group")
    passwordRoleGridDiv.append(passwordGroup);

    let passwordLabel = document.createElement("label")
    passwordLabel.htmlFor = "password"
    passwordLabel.textContent = "Password"
    let passwordInput = document.createElement("input")
    passwordInput.id = "password"
    passwordInput.type = "password"
    passwordInput.name = "password"
    passwordInput.required = true
    passwordInput.autocomplete = "new-password"
    passwordInput.placeholder = "••••••••"
    passwordGroup.append(passwordLabel, passwordInput)

    let roleGroup = document.createElement("div")
    roleGroup.classList.add("form-group")
    passwordRoleGridDiv.append(roleGroup);
    let roleLabel = document.createElement("label")
    roleLabel.htmlFor = "user_role"
    roleLabel.textContent = "System Role"
    let roleSelect = document.createElement("select")
    roleSelect.id = "user_role"
    roleSelect.name = "role"
    let userOption = document.createElement("option")
    userOption.value = "User"
    userOption.textContent = "User"
    let adminOption = document.createElement("option")
    adminOption.value = "Admin"
    adminOption.textContent = "Admin"
    roleSelect.append(userOption, adminOption)
    roleGroup.append(roleLabel, roleSelect)

    let groupsGroup = document.createElement("div")
    groupsGroup.classList.add("form-group")
    addUserForm.append(groupsGroup)
    let groupsLabel = document.createElement("label")
    groupsLabel.textContent = "Assign to Groups"
    groupsGroup.append(groupsLabel)

    let groupsCheckboxContainer = document.createElement("div")
    groupsCheckboxContainer.style = "display: flex; gap: 1.5rem; flex-wrap: wrap; margin-top: 0.5rem;"
    groupsGroup.append(groupsCheckboxContainer)

    let groups = await fetchGroups()
    for (let group of groups) {
        let groupLabel = document.createElement("label")
        groupLabel.style = "display: inline-flex; align-items: center; gap: 0.5rem; font-weight: normal; cursor: pointer;"
        let groupCheckbox = document.createElement("input")
        groupCheckbox.type = "checkbox"
        groupCheckbox.name = "groups"
        groupCheckbox.value = group.id
        groupLabel.append(groupCheckbox, document.createTextNode(group.name))
        groupsCheckboxContainer.append(groupLabel)
    }

    let formActionsDiv = document.createElement("div")
    formActionsDiv.classList.add("form-actions")
    addUserForm.append(formActionsDiv)
    let addUserButton = document.createElement("button")
    addUserButton.type = "submit"
    addUserButton.classList.add("btn", "btn-primary")
    addUserButton.textContent = "Add Database User"
    formActionsDiv.append(addUserButton)

    addUserForm.addEventListener("submit", async function (event) {
        event.preventDefault()
        errorDiv.hidden = true
        successDiv.hidden = true

        const username = usernameInput.value.trim()
        const fullname = fullnameInput.value.trim()
        const password = passwordInput.value
        const role = roleSelect.value
        const selectedGroups = Array.from(groupsCheckboxContainer.querySelectorAll('input[name="groups"]:checked')).map(checkbox => checkbox.value)

        if (!username || !fullname || !password || !role) {
            errorDiv.textContent = "All fields are required."
            errorDiv.hidden = false
            return
        }

        const response = await apiFetch('/api/admin/users/add', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                username: username,
                fullname: fullname,
                password: password,
                role: role,
                groups: selectedGroups
            })
        })
        
        if (!response.ok) {
            const errorMessage = await response.text();
            errorDiv.textContent = errorMessage || "Failed to create user."
            errorDiv.hidden = false
        } else {
            successDiv.textContent = "User created successfully."
            successDiv.hidden = false
            renderUsersTab()
        }
    })
}

export async function fetchUsers() {
    const response = await fetch('/api/admin/users');
    const data = await response.json().catch({"users": []})
    return data.users
}