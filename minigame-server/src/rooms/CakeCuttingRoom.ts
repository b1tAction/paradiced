// CakeCuttingRoom - 切蛋糕 (Cake Cutting) mini-game
//
// Game rules:
// - Turn-based multiplayer action
// - 15-second rules explanation phase (supports early confirmation)
// - Cake represented as boundaries [cakeStart, cakeEnd] inside [0, 100]
// - Moving vertical knife oscillates back and forth
// - Active player clicks "Cut":
//   - If cut is outside active boundaries: player is eliminated
//   - If cut is inside active boundaries: cake is split, and the smaller portion is kept
//   - 1.5-second resolving phase shows the split
// - Turns rotate among alive players. Turn timer is 15s (timeout eliminates the player)
// - Game ends when only 1 active player remains
// - Ranks compiled by order of elimination (last remaining wins)

import { Room, Client } from '@colyseus/core';
import { Schema, MapSchema, defineTypes } from '@colyseus/schema';
import { createHmac } from 'crypto';
import { config } from '../config';

// Player state schema - synchronized to all clients
class CakeCuttingPlayerState extends Schema {
  playerId: string = '';
  isReady: boolean = false;       // Used for rules confirmation
  isAlive: boolean = true;        // Survival state
  eliminatedRound: number = 0;   // The turn count when eliminated (for sorting)
  rank: number = 0;               // Final ranking
}
defineTypes(CakeCuttingPlayerState, {
  playerId: 'string',
  isReady: 'boolean',
  isAlive: 'boolean',
  eliminatedRound: 'number',
  rank: 'number',
});

// Game state schema - synchronized to all clients
class CakeCuttingGameState extends Schema {
  players: MapSchema<CakeCuttingPlayerState> = new MapSchema<CakeCuttingPlayerState>();
  phase: string = 'rules'; // rules / playing / resolving_cut / finished
  roundTimer: number = 15;
  cakeStart: number = 0;
  cakeEnd: number = 100;
  activePlayerId: string = '';
  cutPosition: number = -1;
  nakamaMatchId: string = '';       // For result callback
  minigameInstanceId: string = '';  // For filterBy matching
}
defineTypes(CakeCuttingGameState, {
  players: { map: CakeCuttingPlayerState },
  phase: 'string',
  roundTimer: 'number',
  cakeStart: 'number',
  cakeEnd: 'number',
  activePlayerId: 'string',
  cutPosition: 'number',
  nakamaMatchId: 'string',
  minigameInstanceId: 'string',
});

// HMAC-SHA256 helper for token verification
function hmacSha256(secret: string, message: string): string {
  return createHmac('sha256', secret).update(message).digest('hex');
}

// Timing-safe comparison to prevent timing attacks
function hmacEqual(a: string, b: string): boolean {
  if (a.length !== b.length) return false;
  let result = 0;
  for (let i = 0; i < a.length; i++) {
    result |= a.charCodeAt(i) ^ b.charCodeAt(i);
  }
  return result === 0;
}

export class CakeCuttingRoom extends Room<CakeCuttingGameState> {
  maxClients = 4;

  private secret: string = '';
  private nakamaCallbackUrl: string = '';
  private nakamaMatchId: string = '';
  private timerRef: any = null;
  private resultSent = false;

  private playerOrder: string[] = [];
  private activePlayerIndex = 0;
  private turnCount = 0; // Increments to track order of elimination

  override async onCreate(options: {
    player_id: string;
    nakama_match_id: string;
    minigame_instance_id: string;
    token: string;
    players?: string[];
    nakama_callback?: string;
    secret?: string;
  }): Promise<void> {
    this.setState(new CakeCuttingGameState());
    this.autoDispose = false;

    this.secret = options.secret || config.secret;
    this.nakamaCallbackUrl = options.nakama_callback || config.nakamaRpcUrl;
    this.nakamaMatchId = options.nakama_match_id;
    this.state.nakamaMatchId = this.nakamaMatchId;
    this.state.minigameInstanceId = options.minigame_instance_id;

    // Initialize player states from the full player list
    const players = options.players || [options.player_id];
    for (const pid of players) {
      const player = new CakeCuttingPlayerState();
      player.playerId = pid;
      player.isReady = false;
      player.isAlive = true;
      player.eliminatedRound = 0;
      player.rank = 0;
      this.state.players.set(pid, player);
    }

    this.setPatchRate(50); // 50ms state synchronization

    // Optional safety timeout
    if (config.maxGameDuration > 0) {
      this.clock.setTimeout(() => this.forceFinish(), config.maxGameDuration);
    }

    // Register message handler for rule confirmations
    this.onMessage('confirm_rules', (client) => {
      if (this.state.phase !== 'rules') {
        return;
      }

      const playerId = client.auth?.playerId;
      if (!playerId) return;

      const player = this.state.players.get(playerId);
      if (!player) return;

      player.isReady = true;
      console.log(`[CakeCutting] Player ${playerId} confirmed rules.`);

      // Early start: if all players confirmed, start the first round early
      if (this.allPlayersConfirmedRules()) {
        this.startPlaying();
      }
    });

    // Register message handler for cake cuts
    this.onMessage('cut_cake', (client, message: { pos: number }) => {
      if (this.state.phase !== 'playing') {
        return;
      }

      const playerId = client.auth?.playerId;
      if (!playerId || playerId !== this.state.activePlayerId) {
        return;
      }

      this.processCut(message.pos);
    });

    // Start rules presentation phase
    this.startRulesPhase();
  }

  // onAuth validates HMAC token before allowing a client to join.
  override onAuth(client: Client, options: {
    player_id?: string;
    token?: string;
    minigame_instance_id?: string;
    nakama_match_id?: string;
  }): boolean | { playerId: string } {
    if (!options.player_id || !options.token || !options.minigame_instance_id) {
      console.warn('onAuth: missing required fields');
      return false;
    }

    if (options.minigame_instance_id !== this.state.minigameInstanceId) {
      console.warn('onAuth: minigame_instance_id mismatch');
      return false;
    }

    const message = `${options.player_id}:${options.nakama_match_id || ''}:${options.minigame_instance_id}`;
    const expected = hmacSha256(this.secret, message);
    if (!hmacEqual(options.token, expected)) {
      console.warn('onAuth: invalid HMAC token');
      return false;
    }

    if (!this.state.players.has(options.player_id)) {
      console.warn('onAuth: unknown player_id');
      return false;
    }

    return { playerId: options.player_id };
  }

  override onJoin(client: Client, options: { token?: string }): void {
    console.log(`[CakeCutting] Player joined: ${client.auth?.playerId}`);
  }

  override onLeave(client: Client, consented: boolean): void {
    console.log(`[CakeCutting] Player left: ${client.sessionId}, consented: ${consented}`);
  }

  // Rules phase: 15 seconds rules display
  private startRulesPhase(): void {
    this.clearTimer();
    this.state.phase = 'rules';
    this.state.roundTimer = 15;

    for (const [, player] of this.state.players) {
      player.isReady = false;
    }

    this.timerRef = this.clock.setInterval(() => {
      if (this.state.roundTimer > 0) {
        this.state.roundTimer--;
      } else {
        this.startPlaying();
      }
    }, 1000);
  }

  private allPlayersConfirmedRules(): boolean {
    for (const [, player] of this.state.players) {
      if (!player.isReady) {
        return false;
      }
    }
    return true;
  }

  // Transitions to the active gameplay phase
  private startPlaying(): void {
    this.clearTimer();
    this.state.phase = 'playing';
    this.state.cakeStart = 0;
    this.state.cakeEnd = 100;
    this.state.cutPosition = -1;
    this.turnCount = 0;

    // Reset players alive status
    for (const [, player] of this.state.players) {
      player.isAlive = true;
      player.eliminatedRound = 0;
      player.rank = 0;
    }

    // Set rotation order (consistent across turns)
    this.playerOrder = Array.from(this.state.players.keys());
    this.activePlayerIndex = 0;

    this.startTurn();
  }

  // Start a specific player's turn
  private startTurn(): void {
    this.clearTimer();

    // Check if the game is already resolved (only 1 or 0 players remain alive)
    const alivePlayers = Array.from(this.state.players.values()).filter(p => p.isAlive);
    if (alivePlayers.length <= 1) {
      this.finishGame();
      return;
    }

    // Find the next alive player in rotation
    let attempts = 0;
    while (attempts < this.playerOrder.length) {
      const pid = this.playerOrder[this.activePlayerIndex];
      const player = this.state.players.get(pid);
      if (player && player.isAlive) {
        this.state.activePlayerId = pid;
        break;
      }
      this.activePlayerIndex = (this.activePlayerIndex + 1) % this.playerOrder.length;
      attempts++;
    }

    this.state.phase = 'playing';
    this.state.roundTimer = 15; // 15 seconds to cut the cake

    this.timerRef = this.clock.setInterval(() => {
      if (this.state.roundTimer > 0) {
        this.state.roundTimer--;
      } else {
        // Active player timed out: automatically eliminate them
        console.log(`[CakeCutting] Active player ${this.state.activePlayerId} timed out.`);
        this.eliminatePlayer(this.state.activePlayerId);
        this.nextTurn();
      }
    }, 1000);
  }

  // Processes the cut coordinates submitted by the active player
  private processCut(pos: number): void {
    this.clearTimer();
    this.turnCount++;

    const start = this.state.cakeStart;
    const end = this.state.cakeEnd;

    // Boundary check
    if (pos < start || pos > end) {
      console.log(`[CakeCutting] Player ${this.state.activePlayerId} cut outside active range [${start}, ${end}] at ${pos}.`);
      this.eliminatePlayer(this.state.activePlayerId);
      
      // Visual feedback of the miss
      this.state.cutPosition = pos;
      this.state.phase = 'resolving_cut';
      this.state.roundTimer = 0;

      this.timerRef = this.clock.setTimeout(() => {
        this.nextTurn();
      }, 1500);
    } else {
      // Split cake and keep the smaller portion
      const leftSize = pos - start;
      const rightSize = end - pos;

      if (leftSize < rightSize) {
        // Keep left part
        this.state.cakeStart = start;
        this.state.cakeEnd = pos;
      } else {
        // Keep right part
        this.state.cakeStart = pos;
        this.state.cakeEnd = end;
      }

      this.state.cutPosition = pos;
      this.state.phase = 'resolving_cut';
      this.state.roundTimer = 0;

      console.log(`[CakeCutting] Player ${this.state.activePlayerId} successfully cut at ${pos}. Remaining cake: [${this.state.cakeStart}, ${this.state.cakeEnd}]`);

      // 1.5s reveal phase before next turn
      this.timerRef = this.clock.setTimeout(() => {
        this.nextTurn();
      }, 1500);
    }
  }

  private eliminatePlayer(playerId: string): void {
    const player = this.state.players.get(playerId);
    if (player) {
      player.isAlive = false;
      player.eliminatedRound = this.turnCount;
      console.log(`[CakeCutting] Player ${playerId} is ELIMINATED.`);
    }
  }

  private nextTurn(): void {
    this.activePlayerIndex = (this.activePlayerIndex + 1) % this.playerOrder.length;
    this.startTurn();
  }

  // Compile final results and report to Nakama
  private finishGame(): void {
    if (this.resultSent || this.state.phase === 'finished') {
      return;
    }

    this.clearTimer();
    this.resultSent = true;
    this.state.phase = 'finished';
    this.state.roundTimer = 0;

    const playerList = Array.from(this.state.players.values());

    // Sort: alive survivors go first, then dead players sorted by eliminatedRound descending (survived longer)
    const sorted = [...playerList].sort((a, b) => {
      if (a.isAlive && !b.isAlive) return -1;
      if (!a.isAlive && b.isAlive) return 1;
      return b.eliminatedRound - a.eliminatedRound;
    });

    // Compile rankings (assign strict ranks based on survival/elimination sequence)
    const rankings: { player_id: string; rank: number; game_data?: Record<string, any> }[] = [];
    let currentRank = 1;
    for (let i = 0; i < sorted.length; i++) {
      if (i > 0) {
        const prev = sorted[i - 1];
        const curr = sorted[i];
        if (prev.isAlive !== curr.isAlive || prev.eliminatedRound !== curr.eliminatedRound) {
          currentRank = i + 1;
        }
      }
      sorted[i].rank = currentRank;
      rankings.push({
        player_id: sorted[i].playerId,
        rank: currentRank,
        game_data: {
          score: sorted[i].isAlive ? 100 : sorted[i].eliminatedRound,
        },
      });
    }

    console.log(`[CakeCutting] Game finished. Rankings:`, rankings);
    this.sendResultToNakama(rankings);
  }

  private clearTimer(): void {
    if (this.timerRef) {
      if (typeof this.timerRef.clear === 'function') {
        this.timerRef.clear();
      } else {
        clearTimeout(this.timerRef);
        clearInterval(this.timerRef);
      }
      this.timerRef = null;
    }
  }

  // Send game result to Nakama RPC endpoint
  private async sendResultToNakama(rankings: { player_id: string; rank: number; game_data?: Record<string, any> }[]): Promise<void> {
    try {
      const separator = this.nakamaCallbackUrl.includes('?') ? '&' : '?';
      const rpcUrl = `${this.nakamaCallbackUrl}${separator}http_key=${encodeURIComponent(config.nakamaHttpKey)}&unwrap=true`;
      const response = await fetch(rpcUrl, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          match_id: this.nakamaMatchId,
          room_id: this.roomId,
          game_type: 'cake_cutting',
          rankings,
          secret: this.secret,
        }),
      });

      if (!response.ok) {
        console.error(`Nakama RPC returned status ${response.status}: ${await response.text()}`);
      } else {
        console.log(`Result sent to Nakama successfully: match_id=${this.nakamaMatchId}`);
      }
    } catch (e) {
      console.error('Failed to send result to Nakama:', e);
    }

    // Destroy room after sending final results
    this.clock.setTimeout(() => this.disconnect(), 1000);
  }

  private forceFinish(): void {
    if (this.state.phase === 'finished') {
      return;
    }
    console.warn('[CakeCutting] Force finishing game due to timeout');
    this.finishGame();
  }
}
