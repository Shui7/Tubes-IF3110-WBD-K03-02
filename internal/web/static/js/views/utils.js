export function getCsrfToken() {
    return document.querySelector('meta[name="csrf-token"]')?.content || ""
}

export function setCsrfToken(token) {
    const csrfMeta = document.querySelector('meta[name="csrf-token"]')
    csrfMeta.content = token
}

export async function apiFetch(url, options = {}) {
    const headers = new Headers(options.headers || {});
    headers.set('Accept', 'application/json');

    const method = (options.method || 'GET').toUpperCase();
    if (['POST', 'PUT', 'PATCH', 'DELETE'].includes(method)) {
        headers.set('X-CSRF-Token', getCsrfToken());
    }

    const response = await fetch(url, {
        ...options,
        headers,
        credentials: 'same-origin',
    });

    if (response.status === 401 && window.location.pathname !== '/login') {
        window.history.pushState({}, '', '/login');
        window.dispatchEvent(new PopStateEvent('popstate'));
    }

    return response;
}

export async function fetchJsonArray(url) {
    const response = await apiFetch(url);
    if (!response.ok) {
        throw new Error(await response.text() || `Request failed (${response.status})`);
    }

    const data = await response.json();
    if (data === null) return [];
    if (!Array.isArray(data)) {
        throw new TypeError(`Expected an array response from ${url}`);
    }
    return data;
}

export async function renderLoading() {
    const container = document.getElementById("content")
    container.innerHTML = ""

    let loadingText = document.createElement("h1")
    loadingText.textContent = "Loading..."
    loadingText.style = "text-align: center;"
    container.append(loadingText)
}

export async function renderErrorView(errorCode, errorTitle, errorMessage) {
    const container = document.getElementById("content")
    container.innerHTML = ""

    let errorContainer = document.createElement("div")
    errorContainer.classList.add("narrow")
    container.append(errorContainer)

    let headerDiv = document.createElement("div")
    headerDiv.classList.add("page-header", "text-center")
    errorContainer.append(headerDiv)
    
    let headerText = document.createElement("h1")
    headerText.appendChild(document.createTextNode(errorCode + "—" + errorTitle))
    headerDiv.append(headerText)

    let formCard = document.createElement("div")
    formCard.classList.add("form-card")
    errorContainer.append(formCard)

    formCard.append(document.createElement("p").appendChild(document.createTextNode(errorMessage)))
    let returnButton = document.createElement("button")
    returnButton.appendChild(document.createTextNode("Return Home"))
    returnButton.classList.add("btn", "btn-primary")
    returnButton.onclick((event) => {
        event.preventDefault()
        renderHome()
    })
}

export async function validateSession() {
    const response = await fetch('/api/auth/me')
    const data = await response.json().catch(() => ({"inSession": false}))
    return data.inSession
}