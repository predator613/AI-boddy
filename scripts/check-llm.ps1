# Проверка LLM: ключ из .env, ответ модели напрямую и через backend.
# Запуск: VS Code -> Terminal -> Run Task -> «Проверка LLM», или в PowerShell: .\scripts\check-llm.ps1
# Итог пишется в scripts\llm-check.log (ключ в лог не попадает).
$ErrorActionPreference = 'Continue'
$root = Split-Path -Parent $PSScriptRoot
$log = Join-Path $PSScriptRoot 'llm-check.log'
$out = New-Object System.Collections.Generic.List[string]
function Say($s) { Write-Host $s; $out.Add([string]$s) }
function Post($url, $headers, $obj) {
  $body = [System.Text.Encoding]::UTF8.GetBytes(($obj | ConvertTo-Json -Depth 10 -Compress))
  try {
    $r = Invoke-WebRequest -Uri $url -Method Post -Headers $headers -Body $body -ContentType 'application/json; charset=utf-8' -UseBasicParsing -TimeoutSec 60
    $text = [System.Text.Encoding]::UTF8.GetString($r.RawContentStream.ToArray())
    return @{ code = [int]$r.StatusCode; text = $text }
  } catch {
    $code = 0; $text = $_.Exception.Message
    if ($_.Exception.Response) { $code = [int]$_.Exception.Response.StatusCode }
    if ($_.ErrorDetails -and $_.ErrorDetails.Message) { $text = $_.ErrorDetails.Message }
    return @{ code = $code; text = $text }
  }
}

Say "=== Проверка LLM $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')"
$envFile = Join-Path $root '.env'
$cfg = @{}
foreach ($line in Get-Content $envFile -Encoding UTF8) {
  if ($line -match '^\s*([A-Z_]+)\s*=\s*(.*?)\s*$') { $cfg[$Matches[1]] = $Matches[2].Trim('"') }
}
$key = $cfg['LLM_API_KEY']
Say ("1. .env: provider={0}, base={1}, model={2}, project задан={3}, длина ключа={4}" -f $cfg['LLM_PROVIDER'], $cfg['LLM_BASE_URL'], $cfg['LLM_MODEL'], [bool]$cfg['LLM_PROJECT'], ($key | Measure-Object -Character).Characters)

# 2. Модель напрямую – тот же запрос, что делает backend.
$h = @{ Authorization = "Bearer $key" }
if ($cfg['LLM_PROJECT']) { $h['OpenAI-Project'] = $cfg['LLM_PROJECT'] }
$q = @{ model = $cfg['LLM_MODEL']; temperature = 0.5; messages = @(@{ role = 'user'; content = 'Поздоровайся со стажером одной короткой фразой.' }) }
$folder = ''
if ($cfg['LLM_MODEL'] -match '^gpt://([^/]+)/') { $folder = $Matches[1] }
Say ("   каталог в модели совпадает с LLM_PROJECT: {0}" -f ($folder -eq $cfg['LLM_PROJECT']))
$bases = @($cfg['LLM_BASE_URL'].TrimEnd('/'), 'https://ai.api.cloud.yandex.net/v1') | Select-Object -Unique
$i = 0
foreach ($base in $bases) {
  $i++
  $sw = [Diagnostics.Stopwatch]::StartNew()
  $r = Post ($base + '/chat/completions') $h $q
  $answer = $r.text
  try { $answer = ($r.text | ConvertFrom-Json).choices[0].message.content } catch {}
  if ($answer.Length -gt 300) { $answer = $answer.Substring(0, 300) }
  Say ("2.{0} Модель напрямую ({1}): HTTP {2}, {3} мс: {4}" -f $i, $base, $r.code, $sw.ElapsedMilliseconds, $answer)
}

# 3. Логи backend в Docker.
Push-Location $root
$ps = docker compose ps --format '{{.Service}} {{.State}}' 2>&1
Say "3. docker compose ps: $($ps -join '; ')"
$lines = docker compose logs backend --tail 400 2>&1 | Select-String -Pattern 'assistant ready|llm failed|offline' | Select-Object -Last 6
foreach ($l in $lines) { $s = [string]$l; if ($s.Length -gt 400) { $s = $s.Substring(0, 400) }; Say "   $s" }
if (-not $lines) { Say '   строк «assistant ready» / «llm failed» нет (backend не запущен или лог очищен)' }
Pop-Location

# 4. Чат через backend: один вопрос дважды – живая модель отвечает разными словами.
$port = if ($cfg['BACKEND_PORT']) { $cfg['BACKEND_PORT'] } else { '8080' }
$api = "http://localhost:$port/api"
$login = Post "$api/auth/login" @{} @{ login = 'intern_anna'; password = 'demo123' }
if ($login.code -ne 200) {
  Say "4. Вход intern_anna: HTTP $($login.code) $($login.text)"
} else {
  $tok = ($login.text | ConvertFrom-Json).token
  foreach ($question in @('Как устроено код-ревью у нас в команде?', 'Как устроено код-ревью у нас в команде?', 'Посоветуй, как не тонуть в задачах в первую неделю')) {
    $sw = [Diagnostics.Stopwatch]::StartNew()
    $r = Post "$api/chat/messages" @{ Authorization = "Bearer $tok" } @{ text = $question }
    $txt = $r.text
    try { $txt = ((($r.text | ConvertFrom-Json).messages | ForEach-Object { $_.text }) -join ' | ') } catch {}
    if ($txt.Length -gt 500) { $txt = $txt.Substring(0, 500) }
    Say ("4. «{0}» -> HTTP {1}, {2} мс: {3}" -f $question, $r.code, $sw.ElapsedMilliseconds, $txt)
  }
}
Say '=== Готово'
[System.IO.File]::WriteAllLines($log, $out, (New-Object System.Text.UTF8Encoding($true)))
Write-Host "Лог: $log"
