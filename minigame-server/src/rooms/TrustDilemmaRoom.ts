// TrustDilemmaRoom - 信任博弈 (Trust Dilemma) mini-game
//
// Game rules:
// - Fixed 4 rounds
// - Each round: players choose Cooperate (C) or Compete (D)
// - 10-second timer to choose; defaults to Cooperate (C) if timer expires
// - 3-player or 4-player scoring matrices apply (2-player groups excluded)
// - Displays player ready states during choosing phase
// - 5-second resolution phase displaying player choices and score changes
// - Sorts by cumulative score descending (ties allowed, yielding identical ranks)
// - Secure HMAC token validation in onAuth

import { Room, Client } from '@colyseus/core';
import { Schema, MapSchema, defineTypes } from '@colyseus/schema';
import { createHmac } from 'crypto';
import { config } from '../config';

// Player state schema - synchronized to all clients
class PlayerState extends Schema {
  playerId: string = '';
  choice: number = 0;       // 0=unset, 1=C (Cooperate), 2=D (Compete/Defect)
  score: number = 0;        // Cumulative total score
  roundScore: number = 0;   // Score change in the current round
  isReady: boolean = false; // Whether the player has chosen this round
  rank: number = 0;          // Final ranking (0=not yet determined)
}
defineTypes(PlayerState, {
  playerId: 'string',
  choice: 'number',
  score: 'number',
  roundScore: 'number',
  isReady: 'boolean',
  rank: 'number',
});

// Game state schema - synchronized to all clients
class GameState extends Schema {
  players: MapSchema<PlayerState> = new MapSchema<PlayerState>();
  round: number = 0;
  roundTimer: number = 10;
  phase: string = 'choosing'; // choosing / resolving / finished
  nakamaMatchId: string = '';       // For result callback
  minigameInstanceId: string = '';  // For filterBy matching
}
defineTypes(GameState, {
  players: { map: PlayerState },
  round: 'number',
  roundTimer: 'number',
  phase: 'string',
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

export class TrustDilemmaRoom extends Room<GameState> {
  maxClients = 4;

  private secret: string = '';
  private nakamaCallbackUrl: string = '';
  private nakamaMatchId: string = '';
  private roundTimerRef: any = null;
  private resultSent = false;

  override async onCreate(options: {
    player_id: string;
    nakama_match_id: string;
    minigame_instance_id: string;
    token: string;
    players?: string[];
    nakama_callback?: string;
    secret?: string;
  }): Promise<void> {
    this.setState(new GameState());
    this.autoDispose = false;

    this.secret = options.secret || config.secret;
    this.nakamaCallbackUrl = options.nakama_callback || config.nakamaRpcUrl;
    this.nakamaMatchId = options.nakama_match_id;
    this.state.nakamaMatchId = this.nakamaMatchId;
    this.state.minigameInstanceId = options.minigame_instance_id;

    // Initialize player states from the full player list
    const players = options.players || [options.player_id];
    for (const pid of players) {
      const player = new PlayerState();
      player.playerId = pid;
      player.score = 0;
      player.roundScore = 0;
      player.choice = 0;
      player.isReady = false;
      this.state.players.set(pid, player);
    }

    this.setPatchRate(50); // 50ms state synchronization

    // Optional safety timeout
    if (config.maxGameDuration > 0) {
      this.clock.setTimeout(() => this.forceFinish(), config.maxGameDuration);
    }

    // Register message handler for player decisions
    this.onMessage('choice', (client, message: { choice: number }) => {
      if (this.state.phase !== 'choosing') {
        return;
      }

      const playerId = client.auth?.playerId;
      if (!playerId) {
        return;
      }

      const player = this.state.players.get(playerId);
      if (!player) {
        return;
      }

      // 1=Cooperate (C), 2=Compete (D)
      if ([1, 2].includes(message.choice)) {
        player.choice = message.choice;
        player.isReady = true;
        
        console.log(`[TrustDilemma] Player ${playerId} chose ${message.choice === 1 ? 'C' : 'D'}`);

        // Early resolution: if all active players chose, trigger resolution early to speed up gameplay
        if (this.allActivePlayersChose()) {
          this.resolveRound();
        }
      }
    });

    // Start first round
    this.startRound();
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
    console.log(`[TrustDilemma] Player joined: ${client.auth?.playerId}`);
  }

  override onLeave(client: Client, consented: boolean): void {
    console.log(`[TrustDilemma] Player left: ${client.sessionId}, consented: ${consented}`);
  }

  // Start a new round
  private startRound(): void {
    if (this.state.phase === 'finished' || this.state.round >= 4) {
      this.finishGame();
      return;
    }

    this.clearRoundTimer();
    this.state.round++;
    this.state.phase = 'choosing';
    this.state.roundTimer = 10; // 10 seconds per round

    // Reset choices and ready states for all players
    for (const [, player] of this.state.players) {
      player.choice = 0;
      player.roundScore = 0;
      player.isReady = false;
    }

    // Tick the timer every second for client sync
    this.roundTimerRef = this.clock.setInterval(() => {
      if (this.state.roundTimer > 0) {
        this.state.roundTimer--;
      } else {
        this.resolveRound();
      }
    }, 1000);
  }

  // Resolve the current round
  private resolveRound(): void {
    if (this.state.phase === 'resolving' || this.state.phase === 'finished') {
      return;
    }

    this.clearRoundTimer();
    this.state.phase = 'resolving';
    this.state.roundTimer = 0;

    const playerList = Array.from(this.state.players.values());
    const totalPlayers = playerList.length;

    // Apply default choice of Cooperate (1) for players who failed to choose
    for (const player of playerList) {
      if (player.choice === 0) {
        player.choice = 1;
        player.isReady = true;
      }
    }

    // Count Competitive choices (D count)
    let dCount = 0;
    for (const player of playerList) {
      if (player.choice === 2) {
        dCount++;
      }
    }

    console.log(`[TrustDilemma] Resolving round ${this.state.round}: total_players=${totalPlayers}, D_count=${dCount}`);

    // Apply specific game theory scoring matrices
    for (const player of playerList) {
      let change = 0;
      const isD = player.choice === 2;

      if (totalPlayers === 3) {
        // 3-Player scoring matrix
        if (dCount === 0) {
          change = 4; // All Cooperate: +4 each
        } else if (dCount === 3) {
          change = 1; // All Compete: +1 each
        } else if (dCount === 1) {
          change = isD ? 6 : 1; // 1 Compete, 2 Cooperate: defector +6, cooperator +1
        } else if (dCount === 2) {
          change = isD ? 0 : 6; // 2 Compete, 1 Cooperate: defector +0, cooperator +6
        }
      } else if (totalPlayers === 4) {
        // 4-Player scoring matrix
        if (dCount === 0) {
          change = 5; // All Cooperate: +5 each
        } else if (dCount === 4) {
          change = 1; // All Compete: +1 each
        } else if (dCount === 1) {
          change = isD ? 7 : 2; // 1 Compete, 3 Cooperate: defector +7, cooperator +2
        } else if (dCount === 2) {
          change = isD ? 0 : 1; // 2 Compete, 2 Cooperate: defector +0, cooperator +1
        } else if (dCount === 3) {
          change = isD ? -1 : 3; // 3 Compete, 1 Cooperate: defector -1, cooperator +3
        }
      } else {
        // Fallback for unexpected player count (safely default to +1)
        change = 1;
      }

      player.roundScore = change;
      player.score += change;
    }

    // Update real-time rankings based on cumulative total score (descending)
    const sortedByScore = [...playerList].sort((a, b) => b.score - a.score);
    let currentRank = 1;
    for (let i = 0; i < sortedByScore.length; i++) {
      if (i > 0 && sortedByScore[i].score < sortedByScore[i - 1].score) {
        currentRank = i + 1;
      }
      sortedByScore[i].rank = currentRank;
    }

    // 5-second reveal phase before next round or finishing
    this.roundTimerRef = this.clock.setTimeout(() => {
      if (this.state.round < 4) {
        this.startRound();
      } else {
        this.finishGame();
      }
    }, 5000);
  }

  // Finish the game and send results to Nakama
  private finishGame(): void {
    if (this.resultSent || this.state.phase === 'finished') {
      return;
    }

    this.clearRoundTimer();
    this.resultSent = true;
    this.state.phase = 'finished';
    this.state.roundTimer = 0;

    const playerList = Array.from(this.state.players.values());
    const sorted = [...playerList].sort((a, b) => b.score - a.score);

    // Compile final rankings payload (ties get identical ranks)
    const rankings: { player_id: string; rank: number; game_data?: Record<string, any> }[] = [];
    let currentRank = 1;
    for (let i = 0; i < sorted.length; i++) {
      if (i > 0 && sorted[i].score < sorted[i - 1].score) {
        currentRank = i + 1;
      }
      sorted[i].rank = currentRank;
      rankings.push({
        player_id: sorted[i].playerId,
        rank: currentRank,
        game_data: {
          score: sorted[i].score,
        },
      });
    }

    console.log(`[TrustDilemma] Game finished. Rankings:`, rankings);
    this.sendResultToNakama(rankings);
  }

  private allActivePlayersChose(): boolean {
    for (const [, player] of this.state.players) {
      if (player.choice === 0) {
        return false;
      }
    }
    return true;
  }

  private clearRoundTimer(): void {
    if (this.roundTimerRef) {
      if (typeof this.roundTimerRef.clear === 'function') {
        this.roundTimerRef.clear();
      } else {
        clearTimeout(this.roundTimerRef);
        clearInterval(this.roundTimerRef);
      }
      this.roundTimerRef = null;
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
          game_type: 'trust_dilemma',
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
    console.warn('[TrustDilemma] Force finishing game due to timeout');
    this.finishGame();
  }
}
