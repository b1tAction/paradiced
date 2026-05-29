// Paradiced Mini-Game Server - Colyseus application entry point
//
// This server hosts online mini-games for the Paradiced party game.
// Rooms are created via HTTP API by the Nakama backend, and players
// join via WebSocket using HMAC tokens for authentication.
//
// Game result rankings are sent back to Nakama via RPC callback
// after each game completes.

import { Server } from '@colyseus/core';
import { WebSocketTransport } from '@colyseus/ws-transport';
import { DilemmaRaceRoom } from './rooms/DilemmaRaceRoom';
import { TrustDilemmaRoom } from './rooms/TrustDilemmaRoom';
import { CakeCuttingRoom } from './rooms/CakeCuttingRoom';
import { config } from './config';

const transport = new WebSocketTransport();
const server = new Server({ transport });

// Register mini-game rooms with filterBy for instance-based room matching.
// minigame_instance_id ensures all players from the same Nakama match
// join the same Colyseus room via joinOrCreate.
server.define('dilemma_race', DilemmaRaceRoom)
  .filterBy(['minigame_instance_id']);

server.define('trust_dilemma', TrustDilemmaRoom)
  .filterBy(['minigame_instance_id']);

server.define('cake_cutting', CakeCuttingRoom)
  .filterBy(['minigame_instance_id']);

// Start the server
server.listen(config.port).then(() => {
  console.log(`Paradiced Mini-Game Server listening on port ${config.port}`);
  console.log(`Nakama RPC URL: ${config.nakamaRpcUrl}`);
  console.log(`Secret configured: ${config.secret !== 'change_me_in_production' ? 'yes' : 'no (using default)'}`);
}).catch((err) => {
  console.error('Failed to start server:', err);
  process.exit(1);
});