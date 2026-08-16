# GoKeep backup chain test (against local MinIO S3)
# configure S3 -> test connection -> create backup -> record completed -> download -> verify gzip+SQL -> restore roundtrip -> delete
$ErrorActionPreference = 'Stop'
$base = 'http://127.0.0.1:8080'

function Post($path, $body, $token = $null) {
    $headers = @{}
    if ($token) { $headers['Authorization'] = "Bearer $token" }
    $json = $body | ConvertTo-Json -Depth 10 -Compress
    return Invoke-RestMethod -Uri "$base$path" -Method Post -Headers $headers -ContentType 'application/json; charset=utf-8' -Body ([System.Text.Encoding]::UTF8.GetBytes($json))
}
function PutReq($path, $body, $token = $null) {
    $headers = @{}
    if ($token) { $headers['Authorization'] = "Bearer $token" }
    $json = $body | ConvertTo-Json -Depth 10 -Compress
    return Invoke-RestMethod -Uri "$base$path" -Method Put -Headers $headers -ContentType 'application/json; charset=utf-8' -Body ([System.Text.Encoding]::UTF8.GetBytes($json))
}
function GetReq($path, $token = $null) {
    $headers = @{}
    if ($token) { $headers['Authorization'] = "Bearer $token" }
    return Invoke-RestMethod -Uri "$base$path" -Method Get -Headers $headers
}
function DelReq($path, $token = $null) {
    $headers = @{}
    if ($token) { $headers['Authorization'] = "Bearer $token" }
    return Invoke-RestMethod -Uri "$base$path" -Method Delete -Headers $headers
}

# 1. admin login
$cap = GetReq '/api/v1/auth/captcha'
$answer = Invoke-Expression (($cap.data.expr -split '=')[0].Trim())
$login = Post '/api/v1/auth/login' @{ username = 'admin'; password = 'admin123'; captchaUuid = $cap.data.uuid; captchaCode = "$answer" } $null
if ($login.code -ne 200) { throw "login failed" }
$token = $login.data.token
"STEP1 admin login OK"

# 2. configure S3
$s3 = @{
    'sys.backup.s3.endpoint' = 'http://127.0.0.1:9000'
    'sys.backup.s3.region' = 'us-east-1'
    'sys.backup.s3.bucket' = 'gokeep-backup'
    'sys.backup.s3.prefix' = 'backups/'
    'sys.backup.s3.accessKey' = 'gokeep'
    'sys.backup.s3.secretKey' = 'gokeep123456'
    'sys.backup.s3.forcePathStyle' = 'true'
}
$r = PutReq '/api/v1/system/settings' @{ values = $s3 } $token
if ($r.code -ne 200) { throw "configure s3 failed: $($r | ConvertTo-Json -Compress)" }
"STEP2 S3 config saved"

# 3. test connection
$t = Post '/api/v1/system/backup/test-s3' @{} $token
if ($t.code -ne 200) { throw "s3 test failed: $($t | ConvertTo-Json -Compress)" }
"STEP3 S3 test: $($t.data.message)"

# 4. create backup
$b = Post '/api/v1/system/backup/records' @{ expireDays = 14 } $token
if ($b.code -ne 200) { throw "create backup failed: $($b | ConvertTo-Json -Compress)" }
$recId = $b.data.id
"STEP4 backup created id=$recId status=$($b.data.status)"

# 5. poll until completed
$status = ''
for ($i = 0; $i -lt 30; $i++) {
    Start-Sleep -Seconds 2
    $list = GetReq '/api/v1/system/backup/records?page=1&pageSize=50' $token
    $row = $list.data.list | Where-Object { $_.id -eq $recId }
    if ($row) { $status = $row.status; if ($status -eq 'completed' -or $status -eq 'failed') { break } }
}
"STEP5 backup status=$status"
if ($status -ne 'completed') { throw "backup not completed: $status" }

# 6. verify record fields
$list = GetReq '/api/v1/system/backup/records?page=1&pageSize=50' $token
$row = $list.data.list | Where-Object { $_.id -eq $recId }
"STEP6 record: file=$($row.fileName) size=$($row.sizeBytes) trigger=$($row.triggerType)"
if ($row.sizeBytes -le 0) { throw "backup size invalid" }

# 7. download + verify gzip + SQL
$headers = @{ Authorization = "Bearer $token" }
$dl = Invoke-WebRequest -Uri "$base/api/v1/system/backup/records/$recId/download" -Headers $headers -UseBasicParsing -TimeoutSec 60
$file = Join-Path $env:TEMP "backup-test.sql.gz"
[IO.File]::WriteAllBytes($file, $dl.Content)
$firstBytes = [IO.File]::ReadAllBytes($file)[0..1]
if (-not ($firstBytes[0] -eq 0x1f -and $firstBytes[1] -eq 0x8b)) { throw "not gzip" }
"STEP7 downloaded gzip ($($dl.RawContentLength) bytes)"

# 8. restore roundtrip: count sys_users before/after (fresh login each time)
function CountUsers {
    $c = GetReq '/api/v1/auth/captcha'
    $a = Invoke-Expression (($c.data.expr -split '=')[0].Trim())
    $l = Post '/api/v1/auth/login' @{ username='admin'; password='admin123'; captchaUuid=$c.data.uuid; captchaCode="$a" } $null
    if ($l.code -ne 200) { throw "relogin failed: $($l | ConvertTo-Json -Compress)" }
    $u = GetReq '/api/v1/system/users?page=1&pageSize=1' $l.data.token
    return $u.data.total
}
$beforeTotal = CountUsers
$restore = Post "/api/v1/system/backup/records/$recId/restore" @{} $token
if ($restore.code -ne 200) { throw "restore failed: $($restore | ConvertTo-Json -Compress)" }
Start-Sleep -Seconds 2
$afterTotal = CountUsers
"STEP8 restore: users before=$beforeTotal after=$afterTotal"
if ($beforeTotal -ne $afterTotal) { throw "restore roundtrip mismatch" }

# 9. delete record(fresh login)
function LoginAdmin {
    $c = GetReq '/api/v1/auth/captcha'
    $a = Invoke-Expression (($c.data.expr -split '=')[0].Trim())
    $l = Post '/api/v1/auth/login' @{ username='admin'; password='admin123'; captchaUuid=$c.data.uuid; captchaCode="$a" } $null
    if ($l.code -ne 200) { throw "relogin failed" }
    return $l.data.token
}
$token9 = LoginAdmin
$list9 = GetReq '/api/v1/system/backup/records?page=1&pageSize=50' $token9
if ($list9.data.list.Count -eq 0) { throw "no records to delete" }
$delId = $list9.data.list[0].id
$del = DelReq "/api/v1/system/backup/records/$delId" $token9
if ($del.code -ne 200) { throw "delete failed: $($del | ConvertTo-Json -Compress)" }
$listAfter = GetReq '/api/v1/system/backup/records?page=1&pageSize=50' $token9
$stillThere = ($listAfter.data.list | Where-Object { $_.id -eq $delId })
if ($stillThere) { throw "record still exists after delete" }
"STEP9 record deleted (id=$delId)"

"=== BACKUP CHAIN TEST PASSED ==="
