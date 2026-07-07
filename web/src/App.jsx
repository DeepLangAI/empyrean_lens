import { useState, useEffect } from 'react'
import Login from './pages/Login'

const styles = {
  page: {
    minHeight: '100vh',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    background: 'linear-gradient(135deg, #0f172a 0%, #1e293b 50%, #0f172a 100%)',
    fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", sans-serif',
  },
  card: {
    textAlign: 'center',
    padding: '64px 80px',
    borderRadius: '24px',
    background: 'rgba(255,255,255,0.04)',
    border: '1px solid rgba(255,255,255,0.08)',
    backdropFilter: 'blur(12px)',
    maxWidth: '520px',
    width: '90%',
  },
  eyeWrap: {
    width: '72px', height: '72px', borderRadius: '50%',
    background: 'linear-gradient(135deg, #3b82f6, #8b5cf6)',
    display: 'flex', alignItems: 'center', justifyContent: 'center',
    margin: '0 auto 28px',
    boxShadow: '0 0 40px rgba(99,102,241,0.4)',
  },
  title: { fontSize: '32px', fontWeight: 700, color: '#f1f5f9', margin: '0 0 8px' },
  subtitle: { fontSize: '14px', color: '#64748b', margin: '0 0 36px', textTransform: 'uppercase', letterSpacing: '0.05em' },
  divider: { width: '40px', height: '2px', background: 'linear-gradient(90deg, #3b82f6, #8b5cf6)', margin: '0 auto 32px', borderRadius: '2px' },
  welcome: { fontSize: '16px', color: '#94a3b8', lineHeight: 1.8, margin: '0 0 12px' },
  avatar: { width: '40px', height: '40px', borderRadius: '50%', marginRight: '12px', verticalAlign: 'middle' },
  userRow: { display: 'flex', alignItems: 'center', justifyContent: 'center', marginBottom: '24px' },
  userName: { fontSize: '18px', color: '#f1f5f9', fontWeight: 600 },
  logoutBtn: {
    marginTop: '24px', padding: '8px 24px', borderRadius: '8px',
    background: 'rgba(239,68,68,0.1)', border: '1px solid rgba(239,68,68,0.25)',
    color: '#f87171', fontSize: '14px', cursor: 'pointer',
  },
}

function MainApp({ user, onLogout }) {
  return (
    <div style={styles.page}>
      <div style={styles.card}>
        <div style={styles.eyeWrap}>
          <svg width="36" height="36" viewBox="0 0 24 24" fill="none">
            <path d="M1 12C1 12 5 4 12 4s11 8 11 8-4 8-11 8S1 12 1 12z" stroke="white" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
            <circle cx="12" cy="12" r="3" stroke="white" strokeWidth="2"/>
          </svg>
        </div>
        <h1 style={styles.title}>天眼</h1>
        <p style={styles.subtitle}>Empyrean Lens · 内部监控平台</p>
        <div style={styles.divider} />
        <div style={styles.userRow}>
          {user.avatar_url && <img src={user.avatar_url} alt="avatar" style={styles.avatar} />}
          <span style={styles.userName}>{user.name}</span>
        </div>
        <p style={styles.welcome}>欢迎回来，你已成功登录。</p>
        <button style={styles.logoutBtn} onClick={onLogout}>退出登录</button>
      </div>
    </div>
  )
}

export default function App() {
  const [user, setUser] = useState(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetch('/api/auth/me')
      .then(r => r.ok ? r.json() : null)
      .then(data => { if (data?.data) setUser(data.data) })
      .finally(() => setLoading(false))
  }, [])

  const handleLogout = () => {
    fetch('/api/auth/logout', { method: 'POST' }).finally(() => {
      setUser(null)
    })
  }

  if (loading) return null

  return (
    <>
      <style>{`* { box-sizing: border-box; margin: 0; padding: 0; } body { background: #0f172a; }`}</style>
      {user ? <MainApp user={user} onLogout={handleLogout} /> : <Login />}
    </>
  )
}
