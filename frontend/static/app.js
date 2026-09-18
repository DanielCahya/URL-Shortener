const API_BASE = '/api/v1';

// State
let token = localStorage.getItem('shorter_token');
let user = null;

// DOM Elements
const views = {
    auth: document.getElementById('auth-view'),
    app: document.getElementById('app-view')
};

const authForm = document.getElementById('auth-form');
const emailInput = document.getElementById('email');
const passwordInput = document.getElementById('password');
const authSubmit = document.getElementById('auth-submit');
const authTabs = document.querySelectorAll('.tab');

const shortenerForm = document.getElementById('shortener-form');
const originalUrlInput = document.getElementById('original_url');
const customAliasInput = document.getElementById('custom_alias');

const linksList = document.getElementById('links-list');
const userEmailSpan = document.getElementById('user-email');
const logoutBtn = document.getElementById('logout-btn');

const analyticsModal = document.getElementById('analytics-modal');
const closeModalBtn = document.getElementById('close-modal');
const modalOverlay = document.querySelector('.modal-overlay');

let authMode = 'login'; // login or register

// Initialization
function init() {
    setupEventListeners();
    checkAuth();
}

// Event Listeners
function setupEventListeners() {
    // Auth Tabs
    authTabs.forEach(tab => {
        tab.addEventListener('click', (e) => {
            authTabs.forEach(t => t.classList.remove('active'));
            e.target.classList.add('active');
            authMode = e.target.dataset.target;
            authSubmit.textContent = authMode === 'login' ? 'Log In' : 'Register';
        });
    });

    // Auth Form Submit
    authForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        const email = emailInput.value;
        const password = passwordInput.value;
        
        try {
            authSubmit.disabled = true;
            authSubmit.textContent = 'Please wait...';
            
            const endpoint = authMode === 'login' ? '/auth/login' : '/auth/register';
            const res = await fetchAPI(endpoint, 'POST', { email, password });
            
            if (authMode === 'register') {
                showToast('Registration successful! Please log in.', 'success');
                // Switch to login tab
                authTabs[0].click();
            } else {
                // Login successful
                token = res.access_token;
                localStorage.setItem('shorter_token', token);
                localStorage.setItem('shorter_refresh', res.refresh_token);
                showToast('Logged in successfully', 'success');
                checkAuth();
            }
        } catch (err) {
            showToast(err.message || 'Authentication failed', 'error');
        } finally {
            authSubmit.disabled = false;
            authSubmit.textContent = authMode === 'login' ? 'Log In' : 'Register';
        }
    });

    // Logout
    logoutBtn.addEventListener('click', () => {
        localStorage.removeItem('shorter_token');
        localStorage.removeItem('shorter_refresh');
        token = null;
        user = null;
        showView('auth');
        showToast('Logged out', 'success');
    });

    // Shortener Form
    shortenerForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        const original_url = originalUrlInput.value;
        const custom_alias = customAliasInput.value || null;
        const submitBtn = shortenerForm.querySelector('button');

        try {
            submitBtn.disabled = true;
            submitBtn.textContent = 'Shortening...';
            
            const idempotencyKey = crypto.randomUUID();
            const headers = { 'Idempotency-Key': idempotencyKey };
            
            await fetchAPI('/urls', 'POST', { original_url, custom_alias }, headers);
            
            showToast('URL shortened successfully!', 'success');
            originalUrlInput.value = '';
            customAliasInput.value = '';
            loadDashboard();
        } catch (err) {
            showToast(err.message || 'Failed to shorten URL', 'error');
        } finally {
            submitBtn.disabled = false;
            submitBtn.textContent = 'Shorten';
        }
    });

    // Modal Close
    closeModalBtn.addEventListener('click', closeAnalytics);
    modalOverlay.addEventListener('click', closeAnalytics);
}

// Authentication Check
async function checkAuth() {
    if (!token) {
        showView('auth');
        return;
    }

    try {
        const res = await fetchAPI('/auth/me');
        user = res;
        userEmailSpan.textContent = user.email || 'My Account';
        showView('app');
        loadDashboard();
    } catch (err) {
        // Token might be invalid/expired
        localStorage.removeItem('shorter_token');
        token = null;
        showView('auth');
    }
}

// Dashboard Loader
async function loadDashboard() {
    try {
        const urls = await fetchAPI('/urls');
        renderLinks(urls || []);
    } catch (err) {
        showToast('Failed to load links', 'error');
    }
}

// Render Links
function renderLinks(urls) {
    linksList.innerHTML = '';
    
    if (urls.length === 0) {
        linksList.innerHTML = '<p style="color: var(--text-secondary); text-align: center; padding: 20px;">No links created yet.</p>';
        return;
    }

    urls.forEach(url => {
        const card = document.createElement('div');
        card.className = 'link-card glass-card';
        
        const shortUrl = url.short_url;
        const displayDate = new Date(url.created_at).toLocaleDateString();

        card.innerHTML = `
            <div class="col original-url" title="${url.original_url}">
                ${url.original_url}
            </div>
            <div class="col short-url">
                <a href="${shortUrl}" target="_blank">${shortUrl.replace(/^https?:\/\//, '')}</a>
            </div>
            <div class="col created-at">
                ${displayDate}
            </div>
            <div class="col actions-group">
                <button class="btn outline small" onclick="copyToClipboard('${shortUrl}')">Copy</button>
                <button class="btn outline small" onclick="openAnalytics('${url.short_code}', '${shortUrl}')">Stats</button>
                <button class="btn danger small" onclick="deleteUrl('${url.short_code}')">Delete</button>
            </div>
        `;
        linksList.appendChild(card);
    });
}

// Actions
window.copyToClipboard = async (text) => {
    try {
        await navigator.clipboard.writeText(text);
        showToast('Copied to clipboard!', 'success');
    } catch (err) {
        showToast('Failed to copy', 'error');
    }
};

window.deleteUrl = async (shortCode) => {
    if (!confirm('Are you sure you want to delete this link?')) return;
    
    try {
        await fetchAPI(`/urls/${shortCode}`, 'DELETE');
        showToast('Link deleted', 'success');
        loadDashboard();
    } catch (err) {
        showToast(err.message || 'Failed to delete', 'error');
    }
};

window.openAnalytics = async (shortCode, shortUrl) => {
    try {
        const stats = await fetchAPI(`/urls/${shortCode}/analytics`);
        
        document.getElementById('analytics-short-url').textContent = shortUrl;
        document.getElementById('stat-total-clicks').textContent = stats.total_clicks || 0;
        
        // Find top country
        let topCountry = '-';
        let maxClicks = 0;
        if (stats.by_country) {
            for (const [country, clicks] of Object.entries(stats.by_country)) {
                if (clicks > maxClicks) {
                    maxClicks = clicks;
                    topCountry = country;
                }
            }
        }
        document.getElementById('stat-top-country').textContent = topCountry;

        // Render lists
        renderStatsList('device-list', stats.by_device);
        renderStatsList('browser-list', stats.by_browser);

        analyticsModal.classList.remove('hidden');
    } catch (err) {
        showToast(err.message || 'Failed to load analytics', 'error');
    }
};

function renderStatsList(elementId, data) {
    const list = document.getElementById(elementId);
    list.innerHTML = '';
    
    if (!data || Object.keys(data).length === 0) {
        list.innerHTML = '<li style="color: var(--text-secondary); border: none;">No data yet</li>';
        return;
    }

    // Sort by clicks desc
    const sorted = Object.entries(data).sort((a, b) => b[1] - a[1]);
    
    sorted.forEach(([key, value]) => {
        const li = document.createElement('li');
        li.innerHTML = `<span>${key || 'Unknown'}</span> <strong>${value}</strong>`;
        list.appendChild(li);
    });
}

function closeAnalytics() {
    analyticsModal.classList.add('hidden');
}

// Utility: API Fetcher
async function fetchAPI(path, method = 'GET', body = null, extraHeaders = {}) {
    const headers = {
        'Content-Type': 'application/json',
        ...extraHeaders
    };
    
    if (token) {
        headers['Authorization'] = `Bearer ${token}`;
    }

    const options = { method, headers };
    if (body) {
        options.body = JSON.stringify(body);
    }

    const response = await fetch(`${API_BASE}${path}`, options);
    
    if (response.status === 204) {
        return null; // No content for deletes
    }

    const data = await response.json();

    if (!response.ok) {
        throw new Error(data.error?.message || 'API request failed');
    }

    return data;
}

// Utility: View Switcher
function showView(viewId) {
    Object.values(views).forEach(v => v.classList.add('hidden'));
    views[viewId].classList.remove('hidden');
}

// Utility: Toasts
function showToast(message, type = 'success') {
    const container = document.getElementById('toast-container');
    const toast = document.createElement('div');
    toast.className = `toast ${type}`;
    toast.textContent = message;
    
    container.appendChild(toast);
    
    setTimeout(() => {
        toast.style.opacity = '0';
        toast.style.transform = 'translateX(100%)';
        setTimeout(() => toast.remove(), 300);
    }, 3000);
}

// Boot
init();
