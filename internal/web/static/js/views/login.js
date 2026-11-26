import {createErrorDiv, createSuccessDiv, setTitle} from "./components.js"
import {renderHome} from "./home.js"
import {apiFetch, renderLoading, setCsrfToken, validateSession} from "./utils.js"

export async function renderLoginView() {
    if (await validateSession()) {
        await renderHome()
        return
    }

    setTitle("Log in")
    renderLoading()
    refreshLoginCsrfToken()

    const container = document.getElementById("content")
    container.innerHTML = "";    

    let loginContainer = document.createElement("div")
    loginContainer.classList.add("narrow")
    container.append(loginContainer)

    let titleDiv = document.createElement("div")
    titleDiv.classList.add("page-header", "text-center")
    titleDiv.append(document.createElement("h1").appendChild(document.createTextNode("Sign In")))
    loginContainer.append(titleDiv)

    let errorDiv = createErrorDiv()
    loginContainer.append(errorDiv)

    let formCard = document.createElement("div")
    formCard.classList.add("form-card")
    loginContainer.append(formCard)

    let form = document.createElement("form")
    form.id = "login-form"
    formCard.append(form);
    
    let usernameGroup = document.createElement("div")
    usernameGroup.classList.add("form-group")
    form.append(usernameGroup);

    let usernameLabel = document.createElement("label")
    usernameLabel.htmlFor = "username"
    usernameLabel.textContent = "Username"
    let usernameInput = document.createElement("input")
    usernameInput.id = "username"
    usernameInput.type = "text"
    usernameInput.name = "username"
    usernameInput.required = true
    usernameInput.autofocus = true
    usernameInput.autocomplete = "username"
    usernameInput.placeholder = "Enter username"
    usernameGroup.append(usernameLabel, usernameInput)

    let passwordGroup = document.createElement("div")
    passwordGroup.classList.add("form-group")
    form.append(passwordGroup);

    let passwordLabel = document.createElement("label")
    passwordLabel.htmlFor = "password"
    passwordLabel.textContent = "password"
    let passwordInput = document.createElement("input")
    passwordInput.id = "password"
    passwordInput.type = "password"
    passwordInput.name = "password"
    passwordInput.required = true
    passwordInput.autocomplete = "current-password"
    passwordInput.placeholder = "••••••••"
    passwordGroup.append(passwordLabel, passwordInput)

    let buttonDiv = document.createElement("div")
    form.append(buttonDiv)

    let submitButton = document.createElement("button")
    submitButton.type = "submit"
    submitButton.classList.add("btn", "btn-primary")
    submitButton.style.width = "100%"
    submitButton.textContent = "Sign In"
    buttonDiv.append(submitButton)

    form.addEventListener("submit", async function (event) {
        event.preventDefault();
        const username = usernameInput.value.trim();
        const password = passwordInput.value;

        if (!username || !password) {
            errorDiv.textContent = "Username and password are required.";
            errorDiv.hidden = false
            return;
        }

        errorDiv.textContent = "";
        errorDiv.hidden = true;

        submitButton.disabled = true;
        submitButton.textContent = 'Signing in...';

        try {
            const response = await apiFetch('/api/auth/login', {
                method: 'POST',
                body: JSON.stringify({ username, password }),
            });

            
            if (!response.ok) {
                const errorMessage = await response.text();
                errorDiv.textContent = errorMessage || "Invalid username or password."
                refreshLoginCsrfToken()
                return;
            }

            const data = await response.json().catch(() => ({}));
            const adminNavLink = document.getElementById("nav-link-admin")
            if ((data.role == "Admin") ^ adminNavLink.classList.contains("active")) {
                adminNavLink.classList.toggle("active")
            }

            passwordInput.value = "";
            setCsrfToken(data.csrfToken)

            window.history.pushState({}, '', '/');
            window.dispatchEvent(new PopStateEvent('popstate'));
            renderHome()
        } catch {
            errorDiv.textContent = "Network error occurred when signing in. Try again.";
            errorDiv.hidden = false
            refreshLoginCsrfToken()
        } finally {
            submitButton.disabled = false;
            submitButton.textContent = "Sign in";
        }
    })
}

async function refreshLoginCsrfToken() {
    const response = await fetch("api/auth/refresh-login")
    const data = await response.json().catch({"csrfToken":""})
    setCsrfToken(data.csrfToken)
}