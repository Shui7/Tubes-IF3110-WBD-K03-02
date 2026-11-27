import { renderHome } from "./views/home.js"
import { renderLoginView } from "./views/login.js"
import { renderAdminView } from "./views/admin.js"
import { renderDatabases } from "./views/databases.js"
import { renderDatabaseDetails } from "./views/databasedetail.js"

const routes = [
    { match: (path) => path === "/" || path === "/logout", render: renderHome },
    { match: (path) => path === "/login", render: renderLoginView },
    { match: (path) => /^\/admin\/?$/.test(path), render: renderAdminView },
    { match: (path) => /^\/databases\/?$/.test(path), render: renderDatabases },
    {
        match: (path) => /^\/databases\/\d+\/?$/.test(path),
        render: () => renderDatabaseDetails(Number(window.location.pathname.split("/")[2])),
    },
]

export function startRouter() {
    document.addEventListener("click", handleLinkClick)
    window.addEventListener("popstate", () => {
        void renderCurrentRoute()
    })

    void renderCurrentRoute(true)
}

async function renderCurrentRoute(initialLoad = false) {
    const url = new URL(window.location.href)
    const route = routes.find((candidate) => candidate.match(url.pathname))

    if (!route) {
        renderNotFound()
        return
    }

    if (route.serverRendered) {
        if (initialLoad && document.querySelector("main") && !document.getElementById("content")) {
            updateActiveNavigation(url.pathname)
            initializeTabs()
            return
        }

        await loadServerRenderedPage(url)
        initializeTabs()
        updateActiveNavigation(url.pathname)
        return
    }

    await ensureAppShell(url)
    updateActiveNavigation(url.pathname)
    await route.render()
}

async function ensureAppShell(url) {
    if (document.getElementById("content")) return

    const response = await fetch(url.href)
    const html = await response.text()
    replacePage(html)
}

async function loadServerRenderedPage(url) {
    const response = await fetch(url.href, {
        headers: { "X-Requested-With": "XMLHttpRequest" },
    })
    const finalUrl = new URL(response.url)

    if (finalUrl.pathname !== url.pathname || finalUrl.search !== url.search) {
        window.history.replaceState(null, "", finalUrl.pathname + finalUrl.search + finalUrl.hash)
        await renderCurrentRoute()
        return
    }

    replacePage(await response.text())
}

function replacePage(html) {
    const page = new DOMParser().parseFromString(html, "text/html")
    document.title = page.title

    const csrfMeta = page.querySelector('meta[name="csrf-token"]')
    let currentCsrfMeta = document.querySelector('meta[name="csrf-token"]')
    if (csrfMeta && !currentCsrfMeta) {
        currentCsrfMeta = document.createElement("meta")
        currentCsrfMeta.name = "csrf-token"
        document.head.append(currentCsrfMeta)
    }
    if (csrfMeta) {
        currentCsrfMeta.content = csrfMeta.content
    }

    document.body.innerHTML = page.body.innerHTML
}

function handleLinkClick(event) {
    if (event.defaultPrevented || event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return
    if (!(event.target instanceof Element)) return

    const link = event.target.closest("a[href]")
    if (!link || link.target || link.hasAttribute("download")) return

    if (link.matches(".tab-link[data-tab]")) {
        event.preventDefault()
        activateTab(link, true)
        return
    }

    const url = new URL(link.href, window.location.href)
    if (url.origin !== window.location.origin) return
    if (!routes.some((route) => route.match(url.pathname))) return

    event.preventDefault()
    window.history.pushState(null, "", url.pathname + url.search + url.hash)
    void renderCurrentRoute()
}

function initializeTabs() {
    document.querySelectorAll(".tabs").forEach((tabList) => {
        const links = Array.from(tabList.querySelectorAll(".tab-link[data-tab]"))
        const requestedTab = window.location.hash.slice(1) || new URLSearchParams(window.location.search).get("tab")
        const activeLink = links.find((link) => link.dataset.tab === requestedTab)
            || links.find((link) => link.classList.contains("active"))

        if (activeLink) activateTab(activeLink, false)
    })
}

function activateTab(link, updateUrl) {
    const tabList = link.closest(".tabs")
    const main = link.closest("main")
    if (!tabList || !main) return

    const tab = link.dataset.tab
    tabList.querySelectorAll(".tab-link[data-tab]").forEach((tabLink) => {
        tabLink.classList.toggle("active", tabLink === link)
    })
    main.querySelectorAll(".tab-content").forEach((section) => {
        section.style.display = section.id === `tab-${tab}` ? "block" : "none"
    })

    if (updateUrl) {
        window.history.replaceState(null, "", window.location.pathname + window.location.search + `#${tab}`)
    }
}

function updateActiveNavigation(pathname) {
    document.querySelectorAll(".nav-links a[href]").forEach((link) => {
        const linkPath = new URL(link.href, window.location.href).pathname
        const isDatabaseDetail = linkPath === "/databases" && /^\/databases\/\d+\/?$/.test(pathname)
        link.classList.toggle("active", linkPath === pathname || isDatabaseDetail)
    })
}

function renderNotFound() {
    const container = document.getElementById("content")
    if (!container) return

    container.replaceChildren()
    const heading = document.createElement("h1")
    heading.textContent = "Page not found"
    container.append(heading)
}