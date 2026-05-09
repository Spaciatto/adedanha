import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { User } from '../types';
import { api } from '../api';

interface ProfileProps {
  user: User;
  onUpdate: (user: User) => void;
}

function Profile({ user, onUpdate }: ProfileProps) {
  const navigate = useNavigate();
  const [name, setName] = useState(user.name);
  const [email, setEmail] = useState(user.email);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setSuccess('');

    if (!name.trim() || !email.trim()) {
      setError('Nome e email são obrigatórios');
      return;
    }

    setLoading(true);
    try {
      const updated = await api.updateUser(user.id, name.trim(), email.trim().toLowerCase()) as User;
      onUpdate(updated);
      setSuccess('Perfil atualizado com sucesso!');
    } catch (err: any) {
      setError(err.message || 'Erro ao atualizar perfil');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="card">
      <h2>Editar Perfil</h2>

      {error && <div className="error-message">{error}</div>}
      {success && <div className="success-message">{success}</div>}

      <form onSubmit={handleSubmit}>
        <div className="field-group">
          <label>Nome</label>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="Seu nome"
          />
        </div>
        <div className="field-group">
          <label>Email</label>
          <input
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="seu@email.com"
          />
        </div>
        <button type="submit" className="primary" disabled={loading}>
          {loading ? 'Salvando...' : 'Salvar Alterações'}
        </button>
        <button type="button" className="secondary" onClick={() => navigate('/')}>
          Voltar
        </button>
      </form>
    </div>
  );
}

export default Profile;
