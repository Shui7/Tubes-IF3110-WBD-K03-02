export function createSuccessDiv(message="") {
    let successDiv = document.createElement("div")
    successDiv.classList.add("alert", "alert-success")
    successDiv.textContent = message
    successDiv.hidden = true
    return successDiv
}

export function createErrorDiv(message="") {
    let errorDiv = document.createElement("div")
    errorDiv.classList.add("alert", "alert-error")
    errorDiv.textContent = message
    errorDiv.hidden = true
    return errorDiv
}

export function setTitle(title) {
    if (title == undefined) {
        document.title = "Platform Administration | Database Management Platform"
    }
    else {
        document.title = title + " | Database Management Platform"
    }
}