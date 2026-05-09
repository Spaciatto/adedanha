import { useState } from 'react';
import { User } from '../types';
import { api } from '../api';

interface LoginProps {
  onLogin: (user: User) => void;
}

function Login({ onLogin }: LoginProps) {
  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [isNewUser, setIsNewUser] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');

    if (!email.trim()) {
      setError('Email é obrigatório');
      return;
    }

    if (isNewUser && !name.trim()) {
      setError('Nome é obrigatório para novo cadastro');
      return;
    }

    setLoading(true);
    try {
      let user: User;
      if (isNewUser) {
        user = await api.createUser(name.trim(), email.trim().toLowerCase()) as User;
      } else {
        user = await api.loginUser(email.trim().toLowerCase()) as User;
      }
      onLogin(user);
    } catch (err: any) {
      if (isNewUser) {
        setError(err.message || 'Erro ao cadastrar. Email pode já estar em uso.');
      } else {
        setError(err.message || 'Usuário não encontrado. Cadastre-se primeiro.');
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="card">
      <h2>{isNewUser ? 'Cadastro' : 'Entrar'}</h2>
      <p style={{ marginBottom: '20px', color: '#718096' }}>
        {isNewUser
          ? 'Insira seu nome e email para se cadastrar.'
          : 'Insira seu email para entrar.'}
      </p>

      {error && <div className="error-message">{error}</div>}

      <form onSubmit={handleSubmit}>
        {isNewUser && (
          <div className="field-group">
            <label>Nome</label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Seu nome"
            />
          </div>
        )}
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
          {loading ? 'Aguarde...' : isNewUser ? 'Cadastrar' : 'Entrar'}
        </button>
      </form>

      <div style={{ marginTop: '15px', textAlign: 'center' }}>
        <button
          className="secondary"
          onClick={() => { setIsNewUser(!isNewUser); setError(''); }}
          style={{ fontSize: '0.9rem' }}
        >
          {isNewUser ? 'Já tenho conta - Entrar' : 'Não tenho conta - Cadastrar'}
        </button>
      </div>
    </div>
  );
}

export default Login;
