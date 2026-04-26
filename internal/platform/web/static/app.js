const state = {
  account: null,
  appName: 'Shipper to Carrier',
};

async function request(path, options = {}) {
  const response = await fetch(path, {
    credentials: 'same-origin',
    headers: {
      'Content-Type': 'application/json',
      ...(options.headers || {}),
    },
    ...options,
  });

  if (response.status === 204) {
    return null;
  }

  const payload = await response.json();
  if (!response.ok) {
    throw new Error(payload.error || 'Request failed');
  }

  return payload;
}

async function bootstrap() {
  try {
    const config = await request('/api/v1/config', { method: 'GET' });
    state.appName = config.appName;
  } catch (error) {
    console.error(error);
  }

  try {
    const response = await request('/api/v1/me', { method: 'GET' });
    state.account = response.account;
    routeToRole(state.account.role);
  } catch (_error) {
    state.account = null;
  }

  render();
}

function routeToRole(role) {
  const nextHash = `#/${role}`;
  if (window.location.hash !== nextHash) {
    window.location.hash = nextHash;
  }
}

function activeRoute() {
  return window.location.hash.replace(/^#\//, '');
}

function render() {
  const root = document.querySelector('#app');
  if (!root) {
    return;
  }

  if (!state.account) {
    root.innerHTML = `
      <section class="panel">
        <h2>Get started</h2>
        <p>Create a shipper or carrier login for the foundation dashboard shell.</p>
        <div class="grid two">
          <form id="register-form" class="card">
            <h3>Create account</h3>
            <label>Display name<input name="displayName" required></label>
            <label>Email<input type="email" name="email" required></label>
            <label>Password<input type="password" name="password" minlength="8" required></label>
            <label>Role
              <select name="role">
                <option value="carrier">Carrier</option>
                <option value="shipper">Shipper</option>
              </select>
            </label>
            <button type="submit">Create account</button>
          </form>
          <form id="login-form" class="card">
            <h3>Sign in</h3>
            <label>Email<input type="email" name="email" required></label>
            <label>Password<input type="password" name="password" required></label>
            <button type="submit">Sign in</button>
          </form>
        </div>
        <p id="message" class="message"></p>
      </section>
    `;

    document.querySelector('#register-form').addEventListener('submit', handleRegister);
    document.querySelector('#login-form').addEventListener('submit', handleLogin);
    return;
  }

  const route = activeRoute() || state.account.role;
  const heading = route === 'carrier' ? 'Carrier dashboard' : 'Shipper dashboard';
  const routeLabel = route === 'carrier'
    ? 'Carrier onboarding and load access foundations start here.'
    : 'Shipper account setup and load posting foundations start here.';

  root.innerHTML = `
    <section class="panel">
      <div class="panel-header">
        <div>
          <p class="eyebrow">${state.appName}</p>
          <h2>${heading}</h2>
          <p>${routeLabel}</p>
        </div>
        <button id="logout-button" class="secondary">Sign out</button>
      </div>

      <div class="grid two">
        <article class="card">
          <h3>Authenticated actor</h3>
          <dl>
            <div><dt>Display name</dt><dd>${state.account.displayName}</dd></div>
            <div><dt>Email</dt><dd>${state.account.email}</dd></div>
            <div><dt>Role</dt><dd>${state.account.role}</dd></div>
          </dl>
        </article>

        <article class="card">
          <h3>Role routes</h3>
          <nav class="role-links">
            <a href="#/carrier" class="${route === 'carrier' ? 'active' : ''}">Carrier shell</a>
            <a href="#/shipper" class="${route === 'shipper' ? 'active' : ''}">Shipper shell</a>
          </nav>
          <p class="small">The shell routes by authenticated role after sign-in, then lets you switch between the milestone placeholders.</p>
        </article>
      </div>
    </section>
  `;

  document.querySelector('#logout-button').addEventListener('click', handleLogout);
}

async function handleRegister(event) {
  event.preventDefault();
  const form = new FormData(event.currentTarget);
  const message = document.querySelector('#message');

  try {
    const response = await request('/api/v1/accounts/register', {
      method: 'POST',
      body: JSON.stringify({
        displayName: form.get('displayName'),
        email: form.get('email'),
        password: form.get('password'),
        role: form.get('role'),
      }),
    });

    state.account = response.account;
    routeToRole(state.account.role);
    render();
  } catch (error) {
    message.textContent = error.message;
  }
}

async function handleLogin(event) {
  event.preventDefault();
  const form = new FormData(event.currentTarget);
  const message = document.querySelector('#message');

  try {
    const response = await request('/api/v1/sessions', {
      method: 'POST',
      body: JSON.stringify({
        email: form.get('email'),
        password: form.get('password'),
      }),
    });

    state.account = response.account;
    routeToRole(state.account.role);
    render();
  } catch (error) {
    message.textContent = error.message;
  }
}

async function handleLogout() {
  await request('/api/v1/sessions/logout', { method: 'POST' });
  state.account = null;
  window.location.hash = '';
  render();
}

window.addEventListener('hashchange', render);
window.addEventListener('DOMContentLoaded', bootstrap);
