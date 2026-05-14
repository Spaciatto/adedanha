import { RankingEntry } from '../types';

interface MatchFinishedProps {
  ranking: RankingEntry[];
  onGoHome: () => void;
}

function MatchFinished({ ranking, onGoHome }: MatchFinishedProps) {
  return (
    <div className="card">
      <div className="round-ended-banner" style={{ background: 'linear-gradient(135deg, #667eea, #764ba2)' }}>
        🏆 Partida Encerrada!
      </div>

      <h3>Ranking Final</h3>
      <table className="results-table">
        <thead>
          <tr>
            <th>Posição</th>
            <th>Jogador</th>
            <th>Pontuação Total</th>
          </tr>
        </thead>
        <tbody>
          {ranking.map((entry) => (
            <tr key={entry.user_id}>
              <td>
                <strong>
                  {entry.position === 1 && '🥇 '}
                  {entry.position === 2 && '🥈 '}
                  {entry.position === 3 && '🥉 '}
                  {entry.position}º
                </strong>
              </td>
              <td><strong>{entry.user_name}</strong></td>
              <td><strong>{entry.total_score} pts</strong></td>
            </tr>
          ))}
        </tbody>
      </table>
      <button className="primary" onClick={onGoHome} style={{ marginTop: '20px' }}>
        Voltar ao Início
      </button>
    </div>
  );
}

export default MatchFinished;
