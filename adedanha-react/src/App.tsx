import { useState, useEffect, useRef, useCallback } from 'react';
import { BrowserRouter, Routes, Route, Navigate, useNavigate, useLocation } from 'react-router-dom';
import { User, WSMessage, PendingInvite } from './types';
import { connectPresenceWebSocket, api } from './api';
import Login from './pages/Login';
import Home from './pages/Home';
import MatchPage from './pages/MatchPage';
import Profile from './pages/Profile';

function InviteNotification({ invite, userId, onHandled }: { invite: PendingInvite; userId: string; onHandled: () => void }) {
  const navigate = useNavigate();
  const [processing, setProcessing] = useState(false);

  const handleAccept = async () => {
    setProcessing(true);
    try {
      const res = await api.respondInvite(invite.id, userId, true) as { match_id?: string };
      onHandled();
      if (res.match_id) {
        navigate(`/match/${res.match_id}`);
      }
    } catch {
      onHandled();
    }
  };

  const handleReject = async () => {
    setProcessing(true);
    try {
      await api.respondInvite(invite.id, userId, false);
    } catch {}
    onHandled();
  };

  return (
    <div className="invite-notification">
      <div className="invite-notification-content">
        <strong>🎮 Convite para partida!</strong>
        <p>{invite.inviter_name} convidou você para "{invite.match_name}"</p>
        <div className="invite-notification-actions">
          <button className="success" onClick={handleAccept} disabled={processing}>
            {processing ? '...' : 'Aceitar'}
          </button>
          <button className="danger" onClick={handleReject} disabled={processing}>
            Recusar
          </button>
        </div>
      </div>
    </div>
  );
}

function AppContent({ user, onLogout, onUserUpdate }: { user: User; onLogout: () => void; onUserUpdate: (u: User) => void }) {
  const navigate = useNavigate();
  const location = useLocation();
  const presenceWsRef = useRef<WebSocket | null>(null);
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const mountedRef = useRef(true);
  const [pendingInvites, setPendingInvites] = useState<PendingInvite[]>([]);
  const [notification, setNotification] = useState<string | null>(null);

  // Poll for pending invites (fallback for mobile WS issues)
  const pollInvites = useCallback(async () => {
    try {
      const invites = await api.getPendingInvites(user.id) as PendingInvite[];
      setPendingInvites(invites);
    } catch {}
  }, [user.id]);

  // Poll for active match (handles join_accepted that was missed via WS)
  const checkActiveMatch = useCallback(async () => {
    try {
      const result = await api.getActiveMatch(user.id) as { match_id: string | null };
      if (result.match_id && !location.pathname.includes('/match/')) {
        navigate(`/match/${result.match_id}`);
      }
    } catch {}
  }, [user.id, navigate, location.pathname]);

  useEffect(() => {
    pollInvites();
    checkActiveMatch();
    const interval = setInterval(() => {
      pollInvites();
      checkActiveMatch();
    }, 3000);
    return () => clearInterval(interval);
  }, [pollInvites, checkActiveMatch]);

  // WebSocket presence with visibility-based reconnection
  useEffect(() => {
    mountedRef.current = true;

    function connect() {
      if (!mountedRef.current) return;

      const ws = connectPresenceWebSocket(user.id);
      presenceWsRef.current = ws;

      ws.onopen = () => {
        console.log('Presence WS connected');
      };

      ws.onmessage = (event) => {
        try {
          const msg: WSMessage = JSON.parse(event.data);
          if (msg.type === 'join_accepted' && msg.match_id) {
            navigate(`/match/${msg.match_id}`);
          } else if (msg.type === 'join_rejected') {
            setNotification('Sua solicitação de entrada foi recusada pelo criador da partida.');
            setTimeout(() => setNotification(null), 5000);
          } else if (msg.type === 'match_invite') {
            // Trigger poll immediately to show the invite notification
            pollInvites();
          }
        } catch (e) {
          console.error('Error parsing WS message:', e);
        }
      };

      ws.onclose = () => {
        if (mountedRef.current) {
          reconnectTimerRef.current = setTimeout(connect, 3000);
        }
      };

      ws.onerror = () => {
        ws.close();
      };
    }

    connect();

    // Reconnect on visibility change (mobile browser comes back to foreground)
    const handleVisibilityChange = () => {
      if (document.visibilityState === 'visible') {
        // Force reconnect if WS is not open
        if (presenceWsRef.current && presenceWsRef.current.readyState !== WebSocket.OPEN) {
          presenceWsRef.current.close();
          if (reconnectTimerRef.current) {
            clearTimeout(reconnectTimerRef.current);
          }
          connect();
        }
        // Also poll invites immediately
        pollInvites();
      }
    };
    document.addEventListener('visibilitychange', handleVisibilityChange);

    return () => {
      mountedRef.current = false;
      document.removeEventListener('visibilitychange', handleVisibilityChange);
      if (reconnectTimerRef.current) {
        clearTimeout(reconnectTimerRef.current);
      }
      if (presenceWsRef.current) {
        presenceWsRef.current.close();
        presenceWsRef.current = null;
      }
    };
  }, [user.id, navigate, pollInvites]);

  const handleLogout = () => {
    if (presenceWsRef.current) {
      presenceWsRef.current.close();
      presenceWsRef.current = null;
    }
    // Leave all active matches on logout
    api.leaveAllMatches(user.id).catch(() => {});
    onLogout();
  };

  const handleInviteHandled = () => {
    pollInvites();
  };

  return (
    <div className="container">
      <h1>🎲 Adedanha Online</h1>
      <div className="nav-bar">
        <span className="user-info">Olá, {user.name}!</span>
        <div className="flex-row">
          <button className="secondary" onClick={handleLogout}>Sair</button>
        </div>
      </div>

      {/* Notifications */}
      {notification && (
        <div className="error-message" style={{ marginBottom: '16px' }}>
          {notification}
          <button onClick={() => setNotification(null)} style={{ float: 'right', background: 'none', border: 'none', color: '#c53030', fontWeight: 'bold', minHeight: 'auto', padding: '0 4px', margin: 0 }}>✕</button>
        </div>
      )}

      {/* Invite notifications */}
      {pendingInvites.map((invite) => (
        <InviteNotification
          key={invite.id}
          invite={invite}
          userId={user.id}
          onHandled={handleInviteHandled}
        />
      ))}

      <Routes>
        <Route path="/" element={<Home user={user} />} />
        <Route path="/profile" element={<Profile user={user} onUpdate={onUserUpdate} />} />
        <Route path="/match/:matchId" element={<MatchPage user={user} />} />
        <Route path="*" element={<Navigate to="/" />} />
      </Routes>
    </div>
  );
}

function App() {
  const [user, setUser] = useState<User | null>(null);

  useEffect(() => {
    const stored = localStorage.getItem('adedanha_user');
    if (stored) {
      setUser(JSON.parse(stored));
    }
  }, []);

  const handleLogin = (u: User) => {
    setUser(u);
    localStorage.setItem('adedanha_user', JSON.stringify(u));
  };

  const handleLogout = () => {
    setUser(null);
    localStorage.removeItem('adedanha_user');
  };

  const handleUserUpdate = (u: User) => {
    setUser(u);
    localStorage.setItem('adedanha_user', JSON.stringify(u));
  };

  if (!user) {
    return (
      <BrowserRouter basename="/adedanha">
        <div className="container">
          <h1>🎲 Adedanha Online</h1>
          <Login onLogin={handleLogin} />
        </div>
      </BrowserRouter>
    );
  }

  return (
    <BrowserRouter basename="/adedanha">
      <AppContent user={user} onLogout={handleLogout} onUserUpdate={handleUserUpdate} />
    </BrowserRouter>
  );
}

export default App;
