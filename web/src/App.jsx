import React from 'react'

const styles = {
  page: {
    minHeight: '100vh',
    margin: 0,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    background: 'linear-gradient(135deg, #0f172a 0%, #1e293b 50%, #0f172a 100%)',
    fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Hiragino Sans GB", sans-serif',
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
    width: '72px',
    height: '72px',
    borderRadius: '50%',
    background: 'linear-gradient(135deg, #3b82f6, #8b5cf6)',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    margin: '0 auto 28px',
    boxShadow: '0 0 40px rgba(99,102,241,0.4)',
  },
  title: {
    fontSize: '32px',
    fontWeight: 700,
    color: '#f1f5f9',
    margin: '0 0 8px',
    letterSpacing: '0.02em',
  },
  subtitle: {
    fontSize: '14px',
    color: '#64748b',
    margin: '0 0 36px',
    letterSpacing: '0.05em',
    textTransform: 'uppercase',
  },
  divider: {
    width: '40px',
    height: '2px',
    background: 'linear-gradient(90deg, #3b82f6, #8b5cf6)',
    margin: '0 auto 32px',
    borderRadius: '2px',
  },
  message: {
    fontSize: '16px',
    color: '#94a3b8',
    lineHeight: '1.8',
    margin: '0 0 12px',
  },
  sub: {
    fontSize: '13px',
    color: '#475569',
    lineHeight: '1.6',
  },
  badge: {
    display: 'inline-flex',
    alignItems: 'center',
    gap: '6px',
    marginTop: '36px',
    padding: '6px 16px',
    borderRadius: '100px',
    background: 'rgba(59,130,246,0.1)',
    border: '1px solid rgba(59,130,246,0.2)',
    fontSize: '12px',
    color: '#60a5fa',
    letterSpacing: '0.03em',
  },
  dot: {
    width: '6px',
    height: '6px',
    borderRadius: '50%',
    background: '#3b82f6',
    animation: 'pulse 2s ease-in-out infinite',
  },
}

export default function App() {
  return (
    <>
      <style>{`
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { background: #0f172a; }
        @keyframes pulse {
          0%, 100% { opacity: 1; }
          50% { opacity: 0.3; }
        }
      `}</style>
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
          <p style={styles.message}>系统正在升级中，请稍后再试。</p>
          <p style={styles.sub}>We're upgrading the system. Thank you for your patience.</p>
          <div style={styles.badge}>
            <span style={styles.dot} />
            系统升级中
          </div>
        </div>
      </div>
    </>
  )
}
