import { fetchConnections } from "./connections"

export async function renderGroupsTab() {
     const container = document.getElementById("content")

    let groupsTab = document.createElement("section")
    groupsTab.id = "tab-groups"
    groupsTab.classList.add("tab-content")
    container.append(groupsTab)

    renderGroupsTable(groupsTab)
    renderPermissionsTable(groupsTab)
}

async function renderGroupsTable(groupsTab) {
    let headerDiv = document.createElement("div")
    headerDiv.classList.add("page-header", "text-center")
    headerDiv.append(document.createElement("h2").appendChild(document.createTextNode("User Groups")))
    groupsTab.append(headerDiv)

    let errorDiv = createErrorDiv()
    let successDiv = createSuccessDiv()
    groupsTab.append(errorDiv, successDiv)

    let groupsTableWrap = document.createElement("div")
    groupsTableWrap.classList.add("table-wrap")
    groupsTab.append(groupsTableWrap)

    let groupsTable = document.createElement("table")
    groupsTableWrap.append(groupsTable)

    let tableHead = document.createElement("thead")
    groupsTable.append(tableHead)

    let tableHeadRow = document.createElement("tr")
    tableHead.append(tableHeadRow)

    let tableHeadGroupName = document.createElement("th")
    tableHeadGroupName.textContent = "Group Name"
    tableHeadRow.append(tableHeadGroupName)

    let tableHeadMemberCount = document.createElement("th")
    tableHeadMemberCount.textContent = "Members Count"
    tableHeadRow.append(tableHeadMemberCount)

    let tableHeadActions = document.createElement("th")
    tableHeadActions.textContent = "Actions"
    tableHeadActions.classList.add("text-right")
    tableHeadRow.append(tableHeadActions)

    let tableBody = document.createElement("tbody")
    groupsTable.append(tableBody)

    let groups = await fetchGroups()
    for (let group of groups) {
        let tableRow = document.createElement("tr")
        tableBody.append(tableRow)

        let tableDataGroupName = document.createElement("td")
        tableDataGroupName.classList.add("mono", "font-bold")
        let groupNameStrong = document.createElement("strong")
        groupNameStrong.textContent = group.name
        tableDataGroupName.append(groupNameStrong)
        tableRow.append(tableDataGroupName)

        let tableDataMemberCount = document.createElement("td")
        let memberCountBadge = document.createElement("span")
        memberCountBadge.classList.add("badge")
        memberCountBadge.textContent = group.memberCount + " members"
        tableDataMemberCount.append(memberCountBadge)
        tableRow.append(tableDataMemberCount)

        let tableDataActions = document.createElement("td")
        tableDataActions.classList.add("text-right")
        tableRow.append(tableDataActions)

        let deleteButton = document.createElement("button")
        deleteButton.classList.add("btn", "btn-danger", "btn-sm")
        deleteButton.textContent = "Delete"
        deleteButton.addEventListener("click", async function (event) {
            event.preventDefault()
            const response = await apiFetch('/api/admin/groups/delete', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    id: group.ID
                })
            })
            if (response.ok) {
                renderGroupsTab()
            }
        })
        tableDataActions.append(deleteButton)
    }
    if (groups.length === 0) {
        let tableRow = document.createElement("tr")
        let tableData = document.createElement("td")
        tableData.colSpan = 4
        tableData.classList.add("empty", "center")
        tableData.textContent = "No groups configured yet."
        tableRow.append(tableData)
        tableBody.append(tableRow)
    }

    let createGroupFormCard = document.createElement("div")
    createGroupFormCard.classList.add("form-card", "mt-1")
    groupsTab.append(createGroupFormCard)

    let createGroupHeader = document.createElement("h3")
    createGroupHeader.textContent = "Create User Group"
    createGroupFormCard.append(createGroupHeader)

    let createGroupForm = document.createElement("form")
    createGroupForm.method = "POST"
    createGroupForm.action = "/admin/groups/add"
    createGroupFormCard.append(createGroupForm)

    let groupNameInputGroup = document.createElement("div")
    groupNameInputGroup.classList.add("form-group")
    createGroupForm.append(groupNameInputGroup)

    let groupNameLabel = document.createElement("label")
    groupNameLabel.htmlFor = "group_name"
    groupNameLabel.textContent = "Group Name"
    groupNameInputGroup.append(groupNameLabel)

    let groupNameInput = document.createElement("input")
    groupNameInput.type = "text"
    groupNameInput.id = "group_name"
    groupNameInput.name = "name"
    groupNameInput.required = true
    groupNameInput.placeholder = "e.g. airport-viewer, film-editor"
    groupNameInputGroup.append(groupNameInput)

    let formActionsDiv = document.createElement("div")
    formActionsDiv.classList.add("form-actions")
    createGroupForm.append(formActionsDiv)
    let createGroupButton = document.createElement("button")
    createGroupButton.type = "submit"
    createGroupButton.classList.add("btn", "btn-primary")
    createGroupButton.textContent = "Create User Group"
    formActionsDiv.append(createGroupButton)

    createGroupForm.addEventListener("submit", async function (event) {
        event.preventDefault()
        errorDiv.hidden = true
        successDiv.hidden = true

        const groupName = groupNameInput.value.trim();

        if (!groupName) {
            errorDiv.textContent = "All fields are required."
            errorDiv.hidden = false
            return
        }

        const response = await apiFetch('/api/admin/groups/add', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                name: groupName
            })
        })
        
        if (!response.ok) {
            const errorMessage = await response.text();
            errorDiv.textContent = errorMessage || "Failed to create new group."
            errorDiv.hidden = false
        } else {
            successDiv.textContent = "User group created successfully."
            successDiv.hidden = false
            renderGroupsTab()
        }
    }) 
}

async function renderPermissionsTable(groupsTab) {
    let headerDiv = document.createElement("div")
    headerDiv.classList.add("page-header", "text-center")
    headerDiv.append(document.createElement("h2").appendChild(document.createTextNode("Group Permissions")))
    groupsTab.append(headerDiv)

    let errorDiv = createErrorDiv()
    let successDiv = createSuccessDiv()
    groupsTab.append(errorDiv, successDiv)

    let permissionsTableWrap = document.createElement("div")
    permissionsTableWrap.classList.add("table-wrap")
    groupsTab.append(permissionsTableWrap)

    let permissionsTable = document.createElement("table")
    permissionsTableWrap.append(permissionsTable)

    let tableHead = document.createElement("thead")
    permissionsTable.append(tableHead)

    let tableHeadRow = document.createElement("tr")
    tableHead.append(tableHeadRow)

    let tableHeadGroup = document.createElement("th")
    tableHeadGroup.textContent = "Group"
    tableHeadRow.append(tableHeadGroup)

    let tableHeadDatabase = document.createElement("th")
    tableHeadDatabase.textContent = "Target Database"
    tableHeadRow.append(tableHeadDatabase)

    let tableHeadAccessLevel = document.createElement("th")
    tableHeadAccessLevel.textContent = "Access Level"
    tableHeadRow.append(tableHeadAccessLevel)

    let tableHeadActions = document.createElement("th")
    tableHeadActions.textContent = "Actions"
    tableHeadActions.classList.add("text-right")
    tableHeadRow.append(tableHeadActions)
    
    let tableBody = document.createElement("tbody")
    permissionsTable.append(tableBody)

    let permissions = await fetchPermissions()
    for (let permission of permissions) {
        let tableRow = document.createElement("tr")
        tableBody.append(tableRow)

        let tableDataGroupName = document.createElement("td")
        tableDataGroupName.classList.add("mono", "font-bold")
        let groupNameStrong = document.createElement("strong")
        groupNameStrong.textContent = permission.GroupName
        tableDataGroupName.append(groupNameStrong)
        tableRow.append(tableDataGroupName)

        let tableDataDatabaseName = document.createElement("td")
        tableDataDatabaseName.classList.add("mono")
        tableDataDatabaseName.textContent = permission.DatabaseName
        tableRow.append(tableDataDatabaseName)

        let tableDataAccessLevel = document.createElement("td")
        let accessLevelBadge = document.createElement("span")
        accessLevelBadge.classList.add("badge")
        if (permission.AccessLevel === "edit") {
            accessLevelBadge.classList.add("badge-edit")
            accessLevelBadge.textContent = "View + Edit"
        } else {
            accessLevelBadge.classList.add("badge-view")
            accessLevelBadge.textContent = "View Only"
        }
        tableDataAccessLevel.append(accessLevelBadge)
        tableRow.append(tableDataAccessLevel)

        let tableDataActions = document.createElement("td")
        tableDataActions.classList.add("text-right")
        tableRow.append(tableDataActions)

        let deleteButton = document.createElement("button")
        deleteButton.classList.add("btn", "btn-danger", "btn-sm")
        deleteButton.textContent = "Delete"
        deleteButton.addEventListener("click", async function (event) {
            event.preventDefault()
            const response = await apiFetch('/api/admin/permissions/delete', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    id: permission.id
                })
            })
            if (response.ok) {
                renderGroupsTab()
            }
        })
        tableDataActions.append(deleteButton)
    }
    if (permissions.length === 0) {
        let tableRow = document.createElement("tr")
        let tableData = document.createElement("td")
        tableData.colSpan = 5
        tableData.classList.add("empty", "center")
        tableData.textContent = "No group permissions granted yet."
        tableRow.append(tableData)
        tableBody.append(tableRow)
    }

    
    let grantPermissionFormCard = document.createElement("div")
    grantPermissionFormCard.classList.add("form-card", "mt-1")
    groupsTab.append(grantPermissionFormCard)

    let grantPermissionHeader = document.createElement("h3")
    grantPermissionHeader.textContent = "Grant Database Access to Group"
    grantPermissionFormCard.append(grantPermissionHeader)

    let grantPermissionForm = document.createElement("form")
    grantPermissionForm.method = "POST"
    grantPermissionForm.action = "/admin/permissions/grant"
    grantPermissionFormCard.append(grantPermissionForm)

    /**
     * <div style="display: grid; grid-template-columns: 1fr 1fr 1fr; gap: 1rem;">
                        <div class="form-group">
                            <label for="perm_group">Target Group</label>
                            <select id="perm_group" name="group_id" required>
                                <option value="">-- Select Group --</option>
                                {{ range .Groups }}
                                <option value="{{ .ID }}">{{ .Name }}</option>
                                {{ end }}
                            </select>
                        </div>

                        <div class="form-group">
                            <label for="perm_db">Target Database</label>
                            <select id="perm_db" name="database_id" required>
                                <option value="">-- Select Database --</option>
                                {{ range .Connections }}
                                <option value="{{ .ID }}">{{ .Database }}</option>
                                {{ end }}
                            </select>
                        </div>

                        <div class="form-group">
                            <label for="perm_access">Access Level</label>
                            <select id="perm_access" name="access_level" required>
                                <option value="view">View Only (Schema, Data, History)</option>
                                <option value="edit">View + Edit (Migrations, DDL, DML)</option>
                            </select>
                        </div>
                    </div>

                    <div class="form-actions">
                        <button type="submit" class="btn btn-primary">Grant Access</button>
                    </div>
     */
    let formGridDiv = document.createElement("div")
    formGridDiv.style = "display: grid; grid-template-columns: 1fr 1fr 1fr; gap: 1rem;"
    grantPermissionForm.append(formGridDiv)

    let groupSelectGroup = document.createElement("div")
    groupSelectGroup.classList.add("form-group")
    formGridDiv.append(groupSelectGroup)

    let groupSelectLabel = document.createElement("label")
    groupSelectLabel.htmlFor = "perm_group"
    groupSelectLabel.textContent = "Target Group"
    groupSelectGroup.append(groupSelectLabel)

    let groupSelect = document.createElement("select")
    groupSelect.id = "perm_group"
    groupSelect.name = "group_id"
    groupSelect.required = true
    groupSelectGroup.append(groupSelect)

    let groupSelectDefaultOption = document.createElement("option")
    groupSelectDefaultOption.value = ""
    groupSelectDefaultOption.textContent = "-- Select Group --"
    groupSelect.append(groupSelectDefaultOption)

    let groups = await fetchGroups()
    for (let group of groups) {
        let groupOption = document.createElement("option")
        groupOption.value = group.id
        groupOption.textContent = group.name
        groupSelect.append(groupOption)
    }

    let dbSelectGroup = document.createElement("div")
    dbSelectGroup.classList.add("form-group")
    formGridDiv.append(dbSelectGroup)

    let dbSelectLabel = document.createElement("label")
    dbSelectLabel.htmlFor = "perm_db"
    dbSelectLabel.textContent = "Target Database"
    dbSelectGroup.append(dbSelectLabel)

    let dbSelect = document.createElement("select")
    dbSelect.id = "perm_db"
    dbSelect.name = "database_id"
    dbSelect.required = true
    dbSelectGroup.append(dbSelect)

    let accessSelectGroup = document.createElement("div")
    accessSelectGroup.classList.add("form-group")
    formGridDiv.append(accessSelectGroup)

    let accessSelectLabel = document.createElement("label")
    accessSelectLabel.htmlFor = "perm_access"
    accessSelectLabel.textContent = "Access Level"
    accessSelectGroup.append(accessSelectLabel)
    
    let accessSelect = document.createElement("select")
    accessSelect.id = "perm_access"
    accessSelect.name = "access_level"
    accessSelect.required = true
    accessSelectGroup.append(accessSelect)

    let dbSelectDefaultOption = document.createElement("option")
    dbSelectDefaultOption.value = ""
    dbSelectDefaultOption.textContent = "-- Select Database --"
    dbSelect.append(dbSelectDefaultOption)

    let connections = await fetchConnections()
    for (let connection of connections) {
        let connectionOption = document.createElement("option")
        connectionOption.value = connection.id
        connectionOption.textContent = connection.database
        dbSelect.append(connectionOption)
    }

    let formActionsDiv = document.createElement("div")
    formActionsDiv.classList.add("form-actions")
    grantPermissionForm.append(formActionsDiv)
    let grantPermissionButton = document.createElement("button")
    grantPermissionButton.type = "submit"
    grantPermissionButton.classList.add("btn", "btn-primary")
    grantPermissionButton.textContent = "Grant Permission"
    formActionsDiv.append(grantPermissionButton)

    grantPermissionForm.addEventListener("submit", async function (event) {
        event.preventDefault()
        errorDiv.hidden = true
        successDiv.hidden = true

        const groupName = groupNameInput.value.trim();

        if (!groupName) {
            errorDiv.textContent = "All fields are required."
            errorDiv.hidden = false
            return
        }

        const response = await apiFetch('/api/admin/permissions/grant', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                name: groupName
            })
        })
        
        if (!response.ok) {
            const errorMessage = await response.text();
            errorDiv.textContent = errorMessage || "Failed to grant permission."
            errorDiv.hidden = false
        } else {
            successDiv.textContent = "Permission granted successfully."
            successDiv.hidden = false
            renderGroupsTab()
        }
    }) 
}

export async function fetchGroups() {
    const response = await fetch('/api/admin/groups');
    const data = await response.json().catch({"groups": []})
    return data.groups
}

export async function fetchPermissions() {
    const response = await fetch('/api/admin/permissions');
    const data = await response.json().catch({"permissions": []})
    return data.permissions
}