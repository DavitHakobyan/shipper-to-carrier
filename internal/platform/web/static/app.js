const state = {
  account: null,
  appName: 'Shipper to Carrier',
  onboarding: null,
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
    await hydrateOnboarding();
  } catch (_error) {
    state.account = null;
    state.onboarding = null;
  }

  render();
}

async function hydrateOnboarding() {
  if (!state.account || state.account.role !== 'carrier') {
    state.onboarding = null;
    return;
  }

  try {
    state.onboarding = await request('/api/v1/carriers/current/onboarding-status', { method: 'GET' });
  } catch (_error) {
    state.onboarding = null;
  }
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
    root.innerHTML = renderSignedOut();
    document.querySelector('#register-form').addEventListener('submit', handleRegister);
    document.querySelector('#login-form').addEventListener('submit', handleLogin);
    return;
  }

  const route = activeRoute() || state.account.role;
  const heading = route === 'carrier' ? 'Carrier dashboard' : 'Shipper dashboard';
  const routeLabel = route === 'carrier'
    ? 'Carrier onboarding captures company identity, owners, authority, and insurance before scoring starts.'
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
          <p class="small">Milestone 2 adds carrier onboarding and verification workflow state. Shipper-specific flows remain for a later milestone.</p>
        </article>
      </div>

      <p id="message" class="message"></p>
      ${route === 'carrier' && state.account.role === 'carrier' ? renderCarrierOnboarding() : renderPlaceholder(route)}
    </section>
  `;

  document.querySelector('#logout-button').addEventListener('click', handleLogout);
  bindCarrierForms();
}

function renderSignedOut() {
  return `
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
}

function renderPlaceholder(route) {
  const copy = route === 'shipper'
    ? 'Shipper onboarding and load posting are still scheduled for a later milestone.'
    : 'Carrier onboarding is only available for authenticated carrier actors.';

  return `
    <section class="card">
      <h3>Upcoming work</h3>
      <p>${copy}</p>
    </section>
  `;
}

function renderCarrierOnboarding() {
  if (!state.onboarding) {
    return `
      <section class="grid two">
        <form id="carrier-create-form" class="card">
          <h3>Create carrier account</h3>
          <label>Legal name<input name="legalName" required></label>
          <label>Doing business as<input name="doingBusinessAs"></label>
          <label>Contact phone<input name="contactPhone" required></label>
          <label>Fleet size<input type="number" min="0" name="fleetSizeDeclared"></label>
          <label>Operating regions (comma separated)<input name="operatingRegions"></label>
          <label>Preferred load types (comma separated)<input name="preferredLoadTypes"></label>
          <label>Address line 1<input name="line1" required></label>
          <label>Address line 2<input name="line2"></label>
          <label>City<input name="city" required></label>
          <label>State<input name="state" required></label>
          <label>Postal code<input name="postalCode" required></label>
          <label>Country<input name="country" value="US" required></label>
          <button type="submit">Start onboarding</button>
        </form>

        <article class="card">
          <h3>Expected onboarding stages</h3>
          <ol class="small">
            <li>Business submitted</li>
            <li>Owner identity added</li>
            <li>Authority linked</li>
            <li>Insurance submitted</li>
            <li>Review pending</li>
          </ol>
        </article>
      </section>
    `;
  }

  const requirements = state.onboarding.requirements
    .map((requirement) => `<li><strong>${requirement.requirementType}</strong>: ${requirement.status}</li>`)
    .join('');
  const owners = state.onboarding.owners.length === 0
    ? '<p class="small">No owners added yet.</p>'
    : `<ul>${state.onboarding.owners.map((owner) => `<li>${owner.fullName} - ${owner.ownershipRole}</li>`).join('')}</ul>`;

  return `
    <section class="grid two">
      <article class="card">
        <h3>Onboarding status</h3>
        <dl>
          <div><dt>Carrier</dt><dd>${state.onboarding.carrier.legalName}</dd></div>
          <div><dt>Stage</dt><dd>${state.onboarding.carrier.onboardingStage}</dd></div>
          <div><dt>Verification case</dt><dd>${state.onboarding.verificationCase.status}</dd></div>
        </dl>
        <h4>Requirements</h4>
        <ul>${requirements}</ul>
      </article>

      <article class="card">
        <h3>Owners</h3>
        ${owners}
        <h4>Authority</h4>
        <p class="small">${state.onboarding.authorityLink ? `${state.onboarding.authorityLink.dotNumber || 'No DOT'} / ${state.onboarding.authorityLink.mcNumber || 'No MC'}` : 'No authority linked yet.'}</p>
        <h4>Insurance</h4>
        <p class="small">${state.onboarding.insurancePolicies.length} policy record(s)</p>
      </article>
    </section>

    <section class="grid three">
      <form id="owner-form" class="card">
        <h3>Add owner</h3>
        <label>Full name<input name="fullName" required></label>
        <label>Phone<input name="phone"></label>
        <label>Email<input type="email" name="email" required></label>
        <label>Ownership role<input name="ownershipRole" value="owner" required></label>
        <label><input type="checkbox" name="isPrimaryContact"> Primary contact</label>
        <button type="submit">Save owner</button>
      </form>

      <form id="authority-form" class="card">
        <h3>Link authority</h3>
        <label>DOT number<input name="dotNumber"></label>
        <label>MC number<input name="mcNumber"></label>
        <label>USDOT status<input name="usdotStatus" value="pending"></label>
        <label>Authority type<input name="authorityType" value="for_hire"></label>
        <button type="submit">Save authority</button>
      </form>

      <form id="insurance-form" class="card">
        <h3>Add insurance</h3>
        <label>Provider name<input name="providerName" required></label>
        <label>Policy number<input name="policyNumber" required></label>
        <label>Coverage type<input name="coverageType" value="auto_liability" required></label>
        <label>Effective at<input type="date" name="effectiveAt" required></label>
        <label>Expires at<input type="date" name="expiresAt" required></label>
        <label>Verification status<input name="verificationStatus" value="submitted" required></label>
        <button type="submit">Save insurance</button>
      </form>
    </section>
  `;
}

function bindCarrierForms() {
  const createForm = document.querySelector('#carrier-create-form');
  if (createForm) {
    createForm.addEventListener('submit', handleCreateCarrier);
  }

  const ownerForm = document.querySelector('#owner-form');
  if (ownerForm) {
    ownerForm.addEventListener('submit', handleOwner);
  }

  const authorityForm = document.querySelector('#authority-form');
  if (authorityForm) {
    authorityForm.addEventListener('submit', handleAuthority);
  }

  const insuranceForm = document.querySelector('#insurance-form');
  if (insuranceForm) {
    insuranceForm.addEventListener('submit', handleInsurance);
  }
}

async function handleRegister(event) {
  event.preventDefault();
  const form = new FormData(event.currentTarget);

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
    await hydrateOnboarding();
    render();
  } catch (error) {
    setMessage(error.message);
  }
}

async function handleLogin(event) {
  event.preventDefault();
  const form = new FormData(event.currentTarget);

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
    await hydrateOnboarding();
    render();
  } catch (error) {
    setMessage(error.message);
  }
}

async function handleCreateCarrier(event) {
  event.preventDefault();
  const form = new FormData(event.currentTarget);

  try {
    state.onboarding = await request('/api/v1/carriers', {
      method: 'POST',
      body: JSON.stringify({
        legalName: form.get('legalName'),
        doingBusinessAs: form.get('doingBusinessAs'),
        contactPhone: form.get('contactPhone'),
        fleetSizeDeclared: Number(form.get('fleetSizeDeclared') || 0),
        operatingRegions: splitCSV(form.get('operatingRegions')),
        preferredLoadTypes: splitCSV(form.get('preferredLoadTypes')),
        address: {
          line1: form.get('line1'),
          line2: form.get('line2'),
          city: form.get('city'),
          state: form.get('state'),
          postalCode: form.get('postalCode'),
          country: form.get('country'),
          addressType: 'operating',
        },
      }),
    });
    render();
  } catch (error) {
    setMessage(error.message);
  }
}

async function handleOwner(event) {
  event.preventDefault();
  const form = new FormData(event.currentTarget);

  try {
    state.onboarding = await request('/api/v1/carriers/current/owners', {
      method: 'POST',
      body: JSON.stringify({
        fullName: form.get('fullName'),
        phone: form.get('phone'),
        email: form.get('email'),
        ownershipRole: form.get('ownershipRole'),
        isPrimaryContact: form.get('isPrimaryContact') === 'on',
      }),
    });
    render();
  } catch (error) {
    setMessage(error.message);
  }
}

async function handleAuthority(event) {
  event.preventDefault();
  const form = new FormData(event.currentTarget);

  try {
    state.onboarding = await request('/api/v1/carriers/current/authority', {
      method: 'POST',
      body: JSON.stringify({
        dotNumber: form.get('dotNumber'),
        mcNumber: form.get('mcNumber'),
        usdotStatus: form.get('usdotStatus'),
        authorityType: form.get('authorityType'),
      }),
    });
    render();
  } catch (error) {
    setMessage(error.message);
  }
}

async function handleInsurance(event) {
  event.preventDefault();
  const form = new FormData(event.currentTarget);

  try {
    state.onboarding = await request('/api/v1/carriers/current/insurance', {
      method: 'POST',
      body: JSON.stringify({
        providerName: form.get('providerName'),
        policyNumber: form.get('policyNumber'),
        coverageType: form.get('coverageType'),
        effectiveAt: `${form.get('effectiveAt')}T00:00:00Z`,
        expiresAt: `${form.get('expiresAt')}T00:00:00Z`,
        verificationStatus: form.get('verificationStatus'),
      }),
    });
    render();
  } catch (error) {
    setMessage(error.message);
  }
}

async function handleLogout() {
  await request('/api/v1/sessions/logout', { method: 'POST' });
  state.account = null;
  state.onboarding = null;
  window.location.hash = '';
  render();
}

function splitCSV(value) {
  return String(value || '')
    .split(',')
    .map((part) => part.trim())
    .filter(Boolean);
}

function setMessage(message) {
  const node = document.querySelector('#message');
  if (node) {
    node.textContent = message;
  }
}

window.addEventListener('hashchange', render);
window.addEventListener('DOMContentLoaded', bootstrap);
