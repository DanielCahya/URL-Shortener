$base = "http://localhost:8080/api/v1"

# 1. Register a test user (ignore if already exists)
try {
    Invoke-RestMethod -Uri "$base/auth/register" -Method POST -ContentType "application/json" `
        -Body '{"email":"bench@test.com","password":"benchpass123"}' -ErrorAction SilentlyContinue | Out-Null
} catch {}

# 2. Login
$login = Invoke-RestMethod -Uri "$base/auth/login" -Method POST -ContentType "application/json" `
    -Body '{"email":"bench@test.com","password":"benchpass123"}'
$token = $login.access_token
Write-Host "Authenticated: $($login.user.email)"

# 3. Create a short URL to benchmark redirects
$created = Invoke-RestMethod -Uri "$base/urls" -Method POST -ContentType "application/json" `
    -Headers @{Authorization="Bearer $token"} `
    -Body '{"original_url":"https://github.com/DanielCahya/URL-Shortener"}'
$shortCode = $created.short_code
Write-Host "Created short URL: $shortCode"

# 4. Benchmark redirect endpoint (100 sequential requests)
Write-Host "Benchmarking redirect endpoint (100 requests)..."
$times = @()

for ($i = 0; $i -lt 100; $i++) {
    $sw = [System.Diagnostics.Stopwatch]::StartNew()
    try {
        Invoke-WebRequest -Uri "http://localhost:8080/$shortCode" -Method GET -MaximumRedirection 0 -UseBasicParsing -ErrorAction SilentlyContinue | Out-Null
    } catch {}
    $sw.Stop()
    $times += $sw.ElapsedMilliseconds
}

$avg = ($times | Measure-Object -Average).Average
$min = ($times | Measure-Object -Minimum).Minimum
$max = ($times | Measure-Object -Maximum).Maximum
$sorted = $times | Sort-Object
$p95 = $sorted[[math]::Floor($times.Count * 0.95)]
$p99 = $sorted[[math]::Floor($times.Count * 0.99)]

Write-Host "--- Redirect Benchmark (100 requests, sequential) ---"
Write-Host "Avg latency : $([math]::Round($avg, 2)) ms"
Write-Host "Min latency : $min ms"
Write-Host "Max latency : $max ms"
Write-Host "P95 latency : $p95 ms"
Write-Host "P99 latency : $p99 ms"

# 5. Benchmark URL creation (50 sequential requests)
Write-Host "Benchmarking URL creation (50 requests)..."
$createTimes = @()
for ($i = 0; $i -lt 50; $i++) {
    $sw = [System.Diagnostics.Stopwatch]::StartNew()
    try {
        Invoke-RestMethod -Uri "$base/urls" -Method POST -ContentType "application/json" `
            -Headers @{Authorization="Bearer $token"; "Idempotency-Key"=[guid]::NewGuid().ToString()} `
            -Body "{`"original_url`":`"https://example.com/page$i`"}" | Out-Null
    } catch {}
    $sw.Stop()
    $createTimes += $sw.ElapsedMilliseconds
}

$cavg = ($createTimes | Measure-Object -Average).Average
$csorted = $createTimes | Sort-Object
$cp95 = $csorted[[math]::Floor($createTimes.Count * 0.95)]

Write-Host "--- URL Creation Benchmark (50 requests, sequential) ---"
Write-Host "Avg latency : $([math]::Round($cavg, 2)) ms"
Write-Host "P95 latency : $cp95 ms"
