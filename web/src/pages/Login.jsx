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
    maxWidth: '440px',
    width: '90%',
  },
  eyeWrap: {
    width: '72px', height: '72px', borderRadius: '50%',
    background: 'linear-gradient(135deg, #3b82f6, #8b5cf6)',
    display: 'flex', alignItems: 'center', justifyContent: 'center',
    margin: '0 auto 28px',
    boxShadow: '0 0 40px rgba(99,102,241,0.4)',
  },
  title: { fontSize: '28px', fontWeight: 700, color: '#f1f5f9', margin: '0 0 8px' },
  subtitle: { fontSize: '13px', color: '#64748b', margin: '0 0 40px', textTransform: 'uppercase', letterSpacing: '0.05em' },
  divider: { width: '40px', height: '2px', background: 'linear-gradient(90deg, #3b82f6, #8b5cf6)', margin: '0 auto 36px', borderRadius: '2px' },
  desc: { fontSize: '14px', color: '#64748b', marginBottom: '32px', lineHeight: 1.7 },
  btn: {
    display: 'inline-flex', alignItems: 'center', gap: '10px',
    padding: '13px 28px',
    borderRadius: '12px',
    background: 'linear-gradient(135deg, #00B0FF, #1565C0)',
    border: 'none',
    color: 'white',
    fontSize: '15px',
    fontWeight: 600,
    cursor: 'pointer',
    letterSpacing: '0.02em',
    boxShadow: '0 4px 24px rgba(0,176,255,0.25)',
    transition: 'opacity 0.2s',
    textDecoration: 'none',
  },
}

export default function Login() {
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
        <p style={styles.desc}>使用飞书账号登录，访问内部监控平台。</p>
        <a href="/auth/login" style={styles.btn}>
          {/* 飞书 logo */}
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none">
            <rect width="24" height="24" rx="6" fill="white" fillOpacity="0.15"/>
            <path d="M12 4L4 8.5v7L12 20l8-4.5v-7L12 4z" fill="white" fillOpacity="0.9"/>
          </svg>
          使用飞书登录
        </a>
      </div>
    </div>
  )
}
