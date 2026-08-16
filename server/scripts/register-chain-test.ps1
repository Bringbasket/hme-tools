# GoKeep register chain test
# email-code OFF: direct register -> token -> getInfo -> routers
# email-code ON: no code rejected -> send code (devCode) -> wrong code rejected -> right code ok -> login
$ErrorActionPreference = 'Stop'
$base = 'http://127.0.0.1:8080'
$stamp = Get-Date -Format 'HHmmss'
$email1 = "member1.$stamp@gokeep.dev"
$email2 = "member2.$stamp@gokeep.dev"

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
function ExpectFail($r, $code) {
    if ($r.code -eq $code) { "  ok: [$($r.code)] $($r.msg)" }
    else { throw "unexpected response: code=$($r.code) msg=$($r.msg) (want $code)" }
}

# 1. admin login
$cap = GetReq '/api/v1/auth/captcha'
$exprPart = ($cap.data.expr -split '=')[0].Trim()
$answer = Invoke-Expression $exprPart
$login = Post '/api/v1/auth/login' @{ username = 'admin'; password = 'admin123'; captchaUuid = $cap.data.uuid; captchaCode = "$answer" } $null
if ($login.code -ne 200) { throw "admin login failed: $($login | ConvertTo-Json -Compress)" }
$admin = $login.data.token
"STEP1 admin login OK"

# 2. ensure registerUser=true + emailCodeEnabled=false via config API
$cfgList = GetReq '/api/v1/system/configs?keyword=sys.' $admin
$regUser = $cfgList.data.list | Where-Object { $_.key -eq 'sys.account.registerUser' }
$emailCode = $cfgList.data.list | Where-Object { $_.key -eq 'sys.register.emailCodeEnabled' }
if (-not $regUser) { throw 'config sys.account.registerUser missing' }
if (-not $emailCode) { throw 'config sys.register.emailCodeEnabled missing (seed failed?)' }
if ($regUser.value -ne 'true') { $r = PutReq "/api/v1/system/configs/$($regUser.id)" @{ value = 'true' } $admin; if ($r.code -ne 200) { throw "flip registerUser failed" } }
if ($emailCode.value -ne 'false') { $r = PutReq "/api/v1/system/configs/$($emailCode.id)" @{ value = 'false' } $admin; if ($r.code -ne 200) { throw "flip emailCodeEnabled failed" } }
"STEP2 config ready: registerUser=true, emailCodeEnabled=false"

# 3. register config endpoint
$rc = GetReq '/api/v1/auth/register/config' $null
"STEP3 register config: $($rc.data | ConvertTo-Json -Compress)"
if ($rc.data.registerEnabled -ne $true -or $rc.data.emailCodeEnabled -ne $false) { throw 'config endpoint unexpected' }

# 4. direct register (no code)
$reg = Post '/api/v1/auth/register' @{ email = $email1; password = 'Passw0rd123'; nickname = 'MemberA' } $null
if ($reg.code -ne 200) { throw "register failed: $($reg | ConvertTo-Json -Compress)" }
"STEP4 register OK (no code): userId=$($reg.data.userId) username=$($reg.data.username)"

# 5. new user getInfo + routers
$info = GetReq '/api/v1/auth/getInfo' $reg.data.token
"STEP5 getInfo: roles=$($info.data.roles -join ',') isAdmin=$($info.data.isAdmin)"
$routers = GetReq '/api/v1/auth/routers' $reg.data.token
$paths = ($routers.data | ForEach-Object { $_.path }) -join ','
"STEP5b router paths: $paths"
if ($paths -notlike '*/dashboard*') { throw 'registered user should see home menu' }

# 6. duplicate email rejected
$dup = Post '/api/v1/auth/register' @{ email = $email1; password = 'Passw0rd123' } $null
ExpectFail $dup 409

# 7. turn email code ON
$r = PutReq "/api/v1/system/configs/$($emailCode.id)" @{ value = 'true' } $admin
if ($r.code -ne 200) { throw 'flip emailCodeEnabled on failed' }
$rc = GetReq '/api/v1/auth/register/config' $null
"STEP7 emailCodeEnabled=$($rc.data.emailCodeEnabled)"

# 8. register without code -> rejected
$noCode = Post '/api/v1/auth/register' @{ email = $email2; password = 'Passw0rd123' } $null
ExpectFail $noCode 400

# 9. send code (dev mode returns devCode)
$send = Post '/api/v1/auth/register/email-code' @{ email = $email2 } $null
if ($send.code -ne 200 -or -not $send.data.devCode) { throw "send code failed: $($send | ConvertTo-Json -Compress)" }
$devCode = $send.data.devCode
"STEP9 code sent, devCode=$devCode"

# 10. wrong code -> attempts message
$wrong = Post '/api/v1/auth/register' @{ email = $email2; password = 'Passw0rd123'; code = '000000' } $null
ExpectFail $wrong 400

# 11. correct code -> ok
$reg2 = Post '/api/v1/auth/register' @{ email = $email2; password = 'Passw0rd123'; nickname = 'MemberB'; code = $devCode } $null
if ($reg2.code -ne 200) { throw "register2 failed: $($reg2 | ConvertTo-Json -Compress)" }
"STEP11 register with code OK: userId=$($reg2.data.userId)"

# 12. login as new user
$cap2 = GetReq '/api/v1/auth/captcha'
$exprPart2 = ($cap2.data.expr -split '=')[0].Trim()
$answer2 = Invoke-Expression $exprPart2
$login2 = Post '/api/v1/auth/login' @{ username = $email2; password = 'Passw0rd123'; captchaUuid = $cap2.data.uuid; captchaCode = "$answer2" } $null
if ($login2.code -ne 200) { throw "member2 login failed: $($login2 | ConvertTo-Json -Compress)" }
"STEP12 member2 login OK"

"=== REGISTER CHAIN TEST PASSED ==="
