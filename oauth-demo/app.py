#!/usr/bin/env python3
"""
OAuth Demo 服务 — 标准 Authorization Code Flow 演示

用法：
    AUTH_CENTER=https://sso.example.com \
    CLIENT_ID=oauth-demo \
    CLIENT_SECRET=demo-secret-change-me \
    CALLBACK_URL=https://your-ngrok.ngrok-free.app/auth/callback \
    python3 app.py

访问：http://localhost:8888
"""

import http.server
import http.cookies
import json
import os
import urllib.request
import urllib.parse
from urllib.error import URLError

# ── 配置 ────────────────────────────────────────────────────────────────────

PORT = int(os.environ.get("PORT", 8888))

AUTH_CENTER   = os.environ.get("AUTH_CENTER", "http://localhost:19001").rstrip("/")
CLIENT_ID     = os.environ.get("CLIENT_ID", "oauth-demo")
CLIENT_SECRET = os.environ.get("CLIENT_SECRET", "demo-secret-change-me")
CALLBACK_URL  = os.environ.get("CALLBACK_URL", f"http://localhost:{PORT}/auth/callback")

# ── HTML 模板 ───────────────────────────────────────────────────────────────

PAGE = """<!DOCTYPE html>
<html lang="zh">
<head>
<meta charset="UTF-8">
<title>OAuth Demo</title>
<style>
  * {{ box-sizing: border-box; margin: 0; padding: 0; }}
  body {{
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", sans-serif;
    background: #0f172a; color: #f1f5f9;
    min-height: 100vh; display: flex; align-items: center; justify-content: center;
  }}
  .card {{
    background: rgba(255,255,255,0.05); border: 1px solid rgba(255,255,255,0.08);
    border-radius: 16px; padding: 48px 64px; text-align: center; max-width: 520px; width: 90%;
  }}
  .badge {{
    display: inline-block; padding: 4px 12px; border-radius: 100px;
    background: rgba(99,102,241,0.15); border: 1px solid rgba(99,102,241,0.3);
    color: #a5b4fc; font-size: 12px; margin-bottom: 24px; letter-spacing: 0.05em;
  }}
  h1 {{ font-size: 24px; font-weight: 700; margin-bottom: 8px; }}
  p  {{ color: #64748b; font-size: 14px; margin-bottom: 32px; }}
  .avatar {{ width: 56px; height: 56px; border-radius: 50%; margin: 0 auto 16px; display: block; }}
  .name {{ font-size: 20px; font-weight: 600; margin-bottom: 4px; }}
  .sub  {{ font-size: 13px; color: #64748b; margin-bottom: 32px; font-family: monospace; }}
  a, button {{
    display: inline-block; padding: 10px 28px; border-radius: 8px; font-size: 14px;
    font-weight: 600; text-decoration: none; cursor: pointer; border: none;
  }}
  .primary {{ background: linear-gradient(135deg, #6366f1, #8b5cf6); color: white; }}
  .danger  {{ background: rgba(239,68,68,0.1); border: 1px solid rgba(239,68,68,0.25); color: #f87171; }}
  .info {{
    background: rgba(59,130,246,0.08); border: 1px solid rgba(59,130,246,0.2);
    color: #60a5fa; margin-top: 24px; padding: 16px; border-radius: 8px;
    text-align: left; font-size: 13px;
  }}
  .info pre {{ margin-top: 8px; font-family: monospace; white-space: pre-wrap; word-break: break-all; color: #94a3b8; }}
  .step {{ color: #64748b; font-size: 12px; margin-top: 16px; line-height: 1.8; }}
</style>
</head>
<body><div class="card">{content}</div></body>
</html>"""

LOGIN_CONTENT = """
<div class="badge">OAuth Demo · Port {port}</div>
<h1>未登录</h1>
<p>演示标准 OAuth2 Authorization Code Flow</p>
<a class="primary" href="{login_url}">通过认证中心登录</a>
<div class="step">
  流程：/auth/login → 飞书授权 → /callback?code=xxx → POST /token → 获取用户信息
</div>
"""

HOME_CONTENT = """
<div class="badge">OAuth Demo · 已认证</div>
{avatar}
<div class="name">{name}</div>
<div class="sub">sub: {sub}</div>
<form method="post" action="/logout" style="margin-bottom:0">
  <button class="danger" type="submit">退出登录</button>
</form>
<div class="info">
  <strong>认证方式：</strong>标准 OAuth2 Authorization Code Flow<br>
  <strong>token 获取：</strong><code>POST {auth_center}/token</code><br>
  <strong>验证方式：</strong><code>GET {auth_center}/userinfo</code>
  <pre>{token_preview}</pre>
</div>
"""

ERROR_CONTENT = """
<div class="badge">错误</div>
<h1>{title}</h1>
<p>{message}</p>
<a class="primary" href="/">返回首页</a>
"""

# ── HTTP Handler ─────────────────────────────────────────────────────────────

class DemoHandler(http.server.BaseHTTPRequestHandler):

    def log_message(self, fmt, *args):
        print(f"  {self.address_string()} - {fmt % args}")

    def get_cookie(self, name):
        raw = self.headers.get("Cookie", "")
        cookies = http.cookies.SimpleCookie(raw)
        morsel = cookies.get(name)
        return morsel.value if morsel else None

    def _make_set_cookie(self, name, value, max_age=None, delete=False):
        c = http.cookies.SimpleCookie()
        c[name] = "" if delete else value
        c[name]["path"] = "/"
        c[name]["httponly"] = True
        if delete:
            c[name]["max-age"] = 0
        elif max_age:
            c[name]["max-age"] = max_age
        return c.output(header="").strip()

    def html(self, content, status=200, set_cookies=None):
        body = PAGE.format(content=content).encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", "text/html; charset=utf-8")
        self.send_header("Content-Length", str(len(body)))
        for cookie_str in (set_cookies or []):
            self.send_header("Set-Cookie", cookie_str)
        self.end_headers()
        self.wfile.write(body)

    def redirect(self, location, set_cookies=None):
        self.send_response(302)
        self.send_header("Location", location)
        for cookie_str in (set_cookies or []):
            self.send_header("Set-Cookie", cookie_str)
        self.end_headers()

    def exchange_code(self, code):
        """标准 POST /token：用 code + client_secret 换 access_token"""
        data = urllib.parse.urlencode({
            "grant_type":    "authorization_code",
            "client_id":     CLIENT_ID,
            "client_secret": CLIENT_SECRET,
            "code":          code,
            "redirect_uri":  CALLBACK_URL,
        }).encode()
        req = urllib.request.Request(
            f"{AUTH_CENTER}/token",
            data=data,
            headers={"Content-Type": "application/x-www-form-urlencoded"},
            method="POST",
        )
        try:
            with urllib.request.urlopen(req, timeout=5) as resp:
                result = json.loads(resp.read())
                return result.get("access_token")
        except URLError as e:
            print(f"  [token] error: {e}")
        return None

    def fetch_userinfo(self, access_token):
        """标准 GET /userinfo"""
        req = urllib.request.Request(
            f"{AUTH_CENTER}/userinfo",
            headers={"Authorization": f"Bearer {access_token}"},
        )
        try:
            with urllib.request.urlopen(req, timeout=5) as resp:
                return json.loads(resp.read())
        except URLError as e:
            print(f"  [userinfo] error: {e}")
        return None

    # ── 路由 ──────────────────────────────────────────────────────────────────

    def do_GET(self):
        parsed = urllib.parse.urlparse(self.path)
        params = urllib.parse.parse_qs(parsed.query)

        if parsed.path == "/":
            self.handle_home()
        elif parsed.path == "/auth/callback":
            self.handle_callback(params.get("code", [None])[0],
                                 params.get("error", [None])[0])
        else:
            self.send_response(404)
            self.end_headers()

    def do_POST(self):
        if self.path == "/logout":
            clear = self._make_set_cookie("demo_token", "", delete=True)
            self.redirect("/", set_cookies=[clear])
        else:
            self.send_response(404)
            self.end_headers()

    # ── 页面逻辑 ──────────────────────────────────────────────────────────────

    def handle_home(self):
        token = self.get_cookie("demo_token")
        if not token:
            # 标准授权 URL：带 client_id + redirect_uri
            login_url = (
                f"{AUTH_CENTER}/auth/login"
                f"?client_id={urllib.parse.quote(CLIENT_ID)}"
                f"&redirect_uri={urllib.parse.quote(CALLBACK_URL)}"
                f"&state=demo-state"  # 生产环境应随机生成并验证
            )
            self.html(LOGIN_CONTENT.format(port=PORT, login_url=login_url))
            return

        user = self.fetch_userinfo(token)
        if not user:
            clear = self._make_set_cookie("demo_token", "", delete=True)
            login_url = (
                f"{AUTH_CENTER}/auth/login"
                f"?client_id={urllib.parse.quote(CLIENT_ID)}"
                f"&redirect_uri={urllib.parse.quote(CALLBACK_URL)}"
            )
            self.html(LOGIN_CONTENT.format(port=PORT, login_url=login_url),
                      set_cookies=[clear])
            return

        avatar_html = ""
        if user.get("picture"):
            avatar_html = f'<img class="avatar" src="{user["picture"]}" alt="avatar">'

        token_preview = token[:48] + "..." if len(token) > 48 else token
        self.html(HOME_CONTENT.format(
            avatar=avatar_html,
            name=user.get("name", "未知"),
            sub=user.get("sub", ""),
            auth_center=AUTH_CENTER,
            token_preview=token_preview,
        ))

    def handle_callback(self, code, error):
        """标准回调：先用 code 换 token，再存 cookie"""
        if error:
            self.html(ERROR_CONTENT.format(
                title="授权失败",
                message=f"飞书授权被拒绝：{error}",
            ))
            return
        if not code:
            self.html(ERROR_CONTENT.format(
                title="参数错误",
                message="未收到授权码（code），请重新登录。",
            ))
            return

        print(f"  [callback] received code, exchanging for token...")
        token = self.exchange_code(code)
        if not token:
            self.html(ERROR_CONTENT.format(
                title="Token 交换失败",
                message="POST /token 请求失败，请检查认证中心日志。",
            ))
            return

        print(f"  [callback] token obtained (len={len(token)}), redirecting to /")
        set_cookie = self._make_set_cookie("demo_token", token, max_age=7 * 24 * 3600)
        self.redirect("/", set_cookies=[set_cookie])


# ── 启动 ─────────────────────────────────────────────────────────────────────

if __name__ == "__main__":
    server = http.server.HTTPServer(("", PORT), DemoHandler)
    print(f"OAuth Demo running on http://localhost:{PORT}")
    print(f"Auth Center : {AUTH_CENTER}")
    print(f"Client ID   : {CLIENT_ID}")
    print(f"Callback URL: {CALLBACK_URL}")
    print()
    print("流程：GET /auth/login → 飞书授权 → GET /callback?code=xxx → POST /token → GET /userinfo")
    print()
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print("\nStopped.")
