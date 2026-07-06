#!/usr/bin/env python3
"""
OAuth Demo 服务
演示如何接入 empyrean-lens 统一认证中心

用法：
    python3 app.py

访问：http://localhost:8888
"""

import http.server
import http.cookies
import json
import urllib.request
import urllib.parse
from urllib.error import URLError

# ── 配置 ────────────────────────────────────────────────────────────────────

PORT = 8888

# 认证中心地址
AUTH_CENTER = "http://localhost:19001"

# 本服务的回调地址（需加入认证中心 allowed_redirects 白名单）
CALLBACK_URL = f"http://localhost:{PORT}/auth/callback"

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
    border-radius: 16px; padding: 48px 64px; text-align: center; max-width: 480px; width: 90%;
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
</style>
</head>
<body><div class="card">{content}</div></body>
</html>"""

LOGIN_CONTENT = """
<div class="badge">OAuth Demo · Port {port}</div>
<h1>未登录</h1>
<p>这是一个 OAuth 接入演示服务<br>点击下方按钮通过认证中心登录</p>
<a class="primary" href="{login_url}">通过认证中心登录</a>
"""

HOME_CONTENT = """
<div class="badge">OAuth Demo · 已认证</div>
{avatar}
<div class="name">{name}</div>
<div class="sub">open_id: {open_id}</div>
<form method="post" action="/logout" style="margin-bottom:0">
  <button class="danger" type="submit">退出登录</button>
</form>
<div class="info">
  <strong>验证方式：</strong>调用 <code>{auth_center}/api/auth/verify</code><br>
  <pre>Authorization: Bearer {token_preview}</pre>
</div>
"""

ERROR_CONTENT = """
<div class="badge">错误</div>
<h1>验证失败</h1>
<p>{message}</p>
<a class="primary" href="/">返回首页</a>
"""

# ── HTTP Handler ─────────────────────────────────────────────────────────────

class DemoHandler(http.server.BaseHTTPRequestHandler):

    def log_message(self, fmt, *args):
        print(f"  {self.address_string()} - {fmt % args}")

    # ── 工具方法 ──────────────────────────────────────────────────────────────

    def get_cookie(self, name):
        raw = self.headers.get("Cookie", "")
        cookies = http.cookies.SimpleCookie(raw)
        morsel = cookies.get(name)
        return morsel.value if morsel else None

    def _make_set_cookie(self, name, value, max_age=None, delete=False):
        """生成 Set-Cookie 头字符串（不直接发送，供 html/redirect 统一发送）"""
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
        """发送完整 HTML 响应，set_cookies 为 Set-Cookie 字符串列表"""
        body = PAGE.format(content=content).encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", "text/html; charset=utf-8")
        self.send_header("Content-Length", str(len(body)))
        for cookie_str in (set_cookies or []):
            self.send_header("Set-Cookie", cookie_str)
        self.end_headers()
        self.wfile.write(body)

    def redirect(self, location, set_cookies=None):
        """发送 302 跳转，可附带 Set-Cookie"""
        self.send_response(302)
        self.send_header("Location", location)
        for cookie_str in (set_cookies or []):
            self.send_header("Set-Cookie", cookie_str)
        self.end_headers()

    def verify_token(self, token):
        """调认证中心 /api/auth/verify 验证 token，返回用户信息或 None"""
        req = urllib.request.Request(
            f"{AUTH_CENTER}/api/auth/verify",
            headers={"Authorization": f"Bearer {token}"},
        )
        try:
            with urllib.request.urlopen(req, timeout=5) as resp:
                data = json.loads(resp.read())
                if data.get("code") == 0:
                    return data.get("data")
        except URLError as e:
            print(f"  [verify] error: {e}")
        return None

    # ── 路由 ──────────────────────────────────────────────────────────────────

    def do_GET(self):
        parsed = urllib.parse.urlparse(self.path)
        path = parsed.path
        params = urllib.parse.parse_qs(parsed.query)

        if path == "/":
            self.handle_home()
        elif path == "/auth/callback":
            self.handle_callback(params.get("token", [None])[0])
        else:
            self.send_response(404)
            self.end_headers()

    def do_POST(self):
        if self.path == "/logout":
            self.handle_logout()
        else:
            self.send_response(404)
            self.end_headers()

    # ── 页面逻辑 ──────────────────────────────────────────────────────────────

    def login_url(self):
        return (
            f"{AUTH_CENTER}/auth/login"
            f"?redirect={urllib.parse.quote(CALLBACK_URL)}"
        )

    def handle_home(self):
        token = self.get_cookie("demo_token")

        if not token:
            content = LOGIN_CONTENT.format(port=PORT, login_url=self.login_url())
            self.html(content)
            return

        user = self.verify_token(token)
        if not user:
            # token 失效，清 cookie 并展示登录页
            clear_cookie = self._make_set_cookie("demo_token", "", delete=True)
            content = LOGIN_CONTENT.format(port=PORT, login_url=self.login_url())
            self.html(content, set_cookies=[clear_cookie])
            return

        avatar_html = ""
        if user.get("avatar_url"):
            avatar_html = f'<img class="avatar" src="{user["avatar_url"]}" alt="avatar">'

        token_preview = token[:48] + "..." if len(token) > 48 else token
        content = HOME_CONTENT.format(
            avatar=avatar_html,
            name=user.get("name", "未知"),
            open_id=user.get("open_id", ""),
            auth_center=AUTH_CENTER,
            token_preview=token_preview,
        )
        self.html(content)

    def handle_callback(self, token):
        """接收认证中心带来的 token，存 cookie 后跳首页"""
        if not token:
            self.html(ERROR_CONTENT.format(message="未收到 token，请重新登录。"))
            return

        print(f"  [callback] token received (len={len(token)}), redirecting to /")
        set_cookie = self._make_set_cookie("demo_token", token, max_age=7 * 24 * 3600)
        self.redirect("/", set_cookies=[set_cookie])

    def handle_logout(self):
        clear_cookie = self._make_set_cookie("demo_token", "", delete=True)
        self.redirect("/", set_cookies=[clear_cookie])


# ── 启动 ─────────────────────────────────────────────────────────────────────

if __name__ == "__main__":
    server = http.server.HTTPServer(("", PORT), DemoHandler)
    print(f"OAuth Demo running on http://localhost:{PORT}")
    print(f"Auth Center : {AUTH_CENTER}")
    print(f"Callback URL: {CALLBACK_URL}")
    print()
    print("确认事项：")
    print(f"  1. empyrean-lens 运行在 {AUTH_CENTER}（MODE_ENV=dev go run .）")
    print(f"  2. config_dev.yaml allowed_redirects 已包含 localhost:{PORT}")
    print()
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print("\nStopped.")
