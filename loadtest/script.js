import http from 'k6/http';
import { check, sleep } from 'k6';

// Fallback to random string since uuidv4 module might be slow to load over network
function uuidv4() {
    return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
        var r = Math.random() * 16 | 0, v = c == 'x' ? r : (r & 0x3 | 0x8);
        return v.toString(16);
    });
}

export const options = {
    vus: 30, // Starting with 30 as requested
    duration: '30s',
};

const BASE_URL = __ENV.BASE_URL || 'http://host.docker.internal:8080';

export function setup() {
    // 1. Register a test user
    const email = `testuser_${uuidv4()}@example.com`;
    const password = 'Password123!';
    
    const regRes = http.post(`${BASE_URL}/api/v1/auth/register`, JSON.stringify({
        email: email,
        password: password,
    }), { headers: { 'Content-Type': 'application/json' } });
    
    if (regRes.status !== 201) {
        console.error("Failed to register user in setup: " + regRes.body);
    }
    
    // 2. Login
    const loginRes = http.post(`${BASE_URL}/api/v1/auth/login`, JSON.stringify({
        email: email,
        password: password,
    }), { headers: { 'Content-Type': 'application/json' } });
    
    if (loginRes.status !== 200) {
        console.error("Failed to login user in setup: " + loginRes.body);
    }
    
    const token = loginRes.json('access_token');
    return { token: token };
}

export default function (data) {
    const token = data.token;
    if (!token) {
        console.error("No token found, skipping iteration");
        sleep(1);
        return;
    }
    
    // 1. Create a short URL
    const originalUrl = `https://example.com/test-${uuidv4()}`;
    const createPayload = JSON.stringify({
        original_url: originalUrl
    });
    
    const createHeaders = {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`,
        'Idempotency-Key': uuidv4(),
    };
    
    const createRes = http.post(`${BASE_URL}/api/v1/urls`, createPayload, { headers: createHeaders });
    
    check(createRes, {
        'create status is 201': (r) => r.status === 201,
    });
    
    if (createRes.status === 201) {
        const shortCode = createRes.json('short_code');
        
        // 2. Resolve the short URL
        const resolveRes = http.get(`${BASE_URL}/${shortCode}`, {
            redirects: 0 // We don't want k6 to automatically follow the redirect, just check the 307
        });
        
        check(resolveRes, {
            'resolve status is 307': (r) => r.status === 307,
            'resolve location header is correct': (r) => r.headers.Location === originalUrl,
        });
    }
    
    sleep(0.1); // Small sleep to control the request rate
}
