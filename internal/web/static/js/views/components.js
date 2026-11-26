export function createSuccessDiv(message="") {
    let successDiv = document.createElement("div")
    successDiv.classList.add("alert", "alert-success")
    successDiv.textContent = message
    return successDiv
}

export function createErrorDiv(message="") {
    let errorDiv = document.createElement("div")
    errorDiv.classList.add("alert", "alert-error")
    errorDiv.textContent = message
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