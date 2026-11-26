import { renderHome } from './views/home.js'
import { renderLoginView } from './views/login.js'
import { validateSession } from './views/utils.js'

async function startUp() {
    let sessionValid = await validateSession()
    if (!sessionValid) {
        await renderLoginView()
    }
    else {
        await renderHome()
    }
}

startUp()