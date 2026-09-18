const API_BASE = 'http://127.0.0.1:8080/api/v1';

async function fetchAPI(path, method = 'GET', body = null, extraHeaders = {}) {
    const headers = {
        'Content-Type': 'application/json',
        ...extraHeaders
    };
    
    if (global.token) {
        headers['Authorization'] = `Bearer ${global.token}`;
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
        throw new Error(data.error?.message || data.error || 'API request failed');
    }

    return data;
}

async function run() {
    try {
        console.log('1. Registering...');
        const email = `test_${Date.now()}@test.com`;
        await fetchAPI('/auth/register', 'POST', { email, password: 'password123' });
        console.log('Register successful');

        console.log('2. Logging in...');
        const loginRes = await fetchAPI('/auth/login', 'POST', { email, password: 'password123' });
        global.token = loginRes.access_token;
        console.log('Login successful');

        console.log('3. Getting Profile...');
        const me = await fetchAPI('/auth/me');
        console.log('Profile:', me);

        console.log('4. Shortening URL without alias...');
        const urlRes1 = await fetchAPI('/urls', 'POST', { original_url: 'https://example.com', custom_alias: null }, { 'Idempotency-Key': crypto.randomUUID() });
        console.log('Shorten 1:', urlRes1);

        console.log('5. Shortening URL with alias...');
        const urlRes2 = await fetchAPI('/urls', 'POST', { original_url: 'https://example.com/2', custom_alias: 'myalias' + Date.now() }, { 'Idempotency-Key': crypto.randomUUID() });
        console.log('Shorten 2:', urlRes2);

        console.log('6. Listing URLs...');
        const list = await fetchAPI('/urls');
        console.log('List count:', list.length);

        console.log('All tests passed!');
    } catch (e) {
        console.error('ERROR:', e.message);
    }
}

run();
