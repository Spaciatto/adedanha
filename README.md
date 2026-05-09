# Adedanha Online

Jogo de Adedanha (Stop) multiplayer em tempo real.

## Estrutura

- `adedanha-golang/` - Backend em Go (API REST + WebSocket)
- `adedanha-react/` - Frontend em React + TypeScript

## Como Rodar

### Backend (Go)

```bash
cd adedanha-golang
go run main.go
```

O servidor inicia na porta **8080**.

Requisitos: Go 1.21+, GCC (para compilar sqlite3)

> **Nota:** Se o banco `adedanha.db` já existir de uma versão anterior, delete-o para recriar o schema atualizado.

### Frontend (React)

```bash
cd adedanha-react
npm install
npm run dev
```

O frontend inicia na porta **3000**.

## Como Jogar

1. Acesse `http://localhost:3000`
2. Cadastre-se com nome e email (ou entre com email existente)
3. Crie uma partida (com nome) ou veja partidas abertas e solicite entrada
4. O criador aceita/recusa solicitações de entrada
5. O criador da partida inicia as rodadas
6. Uma letra aleatória é sorteada e todos preenchem: COR, FRUTA, OBJETO, FILME, CIDADE
7. Tempo de 1 minuto por rodada com timer visível
8. Ao final, o criador confere as respostas e atribui pontos
9. O criador pode iniciar novas rodadas ou encerrar a partida
10. Ao encerrar, ranking final é exibido para todos

## Regras

- Um jogador só pode estar em **uma partida por vez**
- Jogadores podem **abandonar** a partida, mas seus scores permanecem no ranking final
- Se um jogador fechar o browser, pode **voltar à partida** acessando o mesmo ID
- Todas as respostas são armazenadas em **MAIÚSCULAS**
- Apenas o criador pode iniciar rodadas, atribuir scores e encerrar a partida

## API Endpoints

| Método | Rota | Descrição |
|--------|------|-----------|
| POST | /api/users | Criar usuário |
| POST | /api/users/login | Login por email |
| PUT | /api/users/:id | Atualizar usuário |
| GET | /api/users/:id | Buscar usuário |
| GET | /api/online-users | Listar usuários online |
| POST | /api/matches | Criar partida (com nome) |
| GET | /api/matches/open | Listar partidas abertas |
| POST | /api/matches/:id/join | Entrar na partida |
| POST | /api/matches/:id/leave | Abandonar partida |
| POST | /api/matches/:id/request-join | Solicitar entrada |
| GET | /api/matches/:id/join-requests | Listar solicitações pendentes |
| POST | /api/matches/:id/join-requests/:reqId/respond | Aceitar/recusar solicitação |
| GET | /api/matches/:id | Detalhes da partida |
| GET | /api/matches/:id/state | Estado completo (reconexão) |
| POST | /api/matches/:id/end | Encerrar partida |
| POST | /api/matches/:id/rounds/start | Iniciar rodada |
| POST | /api/matches/:id/rounds/:roundId/answers | Enviar respostas |
| PUT | /api/matches/:id/rounds/:roundId/scores | Atualizar scores |
| GET | /api/matches/:id/rounds/:roundId/results | Resultados da rodada |
| WS | /ws/:matchId/:userId | WebSocket da partida |
| WS | /ws/presence/:userId | WebSocket de presença global |
