import { renderLoginView } from "./login.js"
import { renderLoading, validateSession } from "./utils.js"

export async function renderHome() {
    if (!await validateSession()) {
        await renderLoginView()
        return
    }

    renderLoading()
    
    const container = document.getElementById("content")
    container.innerHTML = ""

    const response = await fetch('/api/auth/me');
    const data = await response.json().catch(() => ({"name": "", "role": "", "inSession": false}))

    let welcomeBackMessage = document.createElement("h1")
    let nameText = document.createElement("strong")
    nameText.textContent = data.name
    welcomeBackMessage.append(
        document.createTextNode("Welcome back, "),
        nameText, 
        document.createTextNode("!")
    )
    let loggedInAsMessage = document.createElement("h2")
    let roleText = document.createElement("strong")
    roleText.textContent = data.role
    loggedInAsMessage.append(
        document.createTextNode("Logged in as "),
        roleText
    )
    container.append(welcomeBackMessage, loggedInAsMessage)
}