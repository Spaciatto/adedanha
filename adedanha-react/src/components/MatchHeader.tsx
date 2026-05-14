import { Match } from '../types';

interface MatchHeaderProps {
  match: Match;
  isCreator: boolean;
  phase: string;
  loading: boolean;
  error: string;
  onLeave: () => void;
}

function MatchHeader({ match, isCreator, phase, loading, error, onLeave }: MatchHeaderProps) {
  return (
    <div className="card">
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <div>
          <h2>{match.name || 'Partida'}</h2>
          <span className={`status-badge ${match.status}`}>{match.status}</span>
        </div>
        <div style={{ textAlign: 'right' }}>
          <span style={{ fontSize: '0.85rem', color: '#718096', display: 'block' }}>
            Rodada: {match.current_round}
          </span>
          {!isCreator && phase !== 'finished' && (
            <button
              className="danger"
              onClick={onLeave}
              disabled={loading}
              style={{ marginTop: '8px', fontSize: '0.8rem', padding: '6px 12px' }}
            >
              Abandonar
            </button>
          )}
        </div>
      </div>

      <div className="match-id-display">
        <strong>ID para convidar:</strong> {match.id}
      </div>

      <div>
        <strong style={{ fontSize: '0.9rem', color: '#718096' }}>Jogadores:</strong>
        <div className="players-list">
          {match.players?.map((p) => (
            <span
              key={p.user_id}
              className={`player-badge ${p.user_id === match.creator_id ? 'creator' : ''} ${!p.active ? 'inactive' : ''}`}
            >
              {p.user_name || p.user_id.slice(0, 8)}
              {p.user_id === match.creator_id && ' 👑'}
              {!p.active && ' (saiu)'}
            </span>
          ))}
        </div>
      </div>

      {error && <div className="error-message">{error}</div>}
    </div>
  );
}

export default MatchHeader;
