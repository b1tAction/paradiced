// DilemmaRaceRoom - 博弈论竞速 (Dilemma Race) mini-game
//
// Game rules:
// - 15-cell track, players start at position 1
// - Each round: players choose to move 1, 3, or 5 steps
// - If >=2 players choose the same step count, they are blocked (cannot move)
// - First player to reach position 15 wins
// - Maximum 20 rounds; if no winner, ranked by position
//
// Room creation mode: joinOrCreate via WebSocket/matchmaker.
// - First client calls joinOrCreate with full player list → room is created
// - Subsequent clients call joinOrCreate with same minigame_instance_id → matched to existing room
// - onAuth validates HMAC token before allowing join

import { Room, Client } from '@colyseus/core';
import { Schema, MapSchema, defineTypes } from '@colyseus/schema';
import { createHmac } from 'crypto';
import { config } from '../config';

// Player state schema - synchronized to all clients
// Uses defineTypes (non-decorator pattern) for Colyseus schema v2 compatibility
class PlayerState extends Schema {
  playerId: string = '';
  position: number = 1;    // Current cell position (1-15)
  choice: number = 0;       // 0=unset, 1=step1, 3=step3, 5=step5
  blocked: boolean = false; // Blocked due to collision with other players
  finished: boolean = false; // Reached position 15
  rank: number = 0;          // Final ranking (0=not yet determined)
}
defineTypes(PlayerState, {
  playerId: 'string',
  position: 'number',
  choice: 'number',
  blocked: 'boolean',
  finished: 'boolean',
  rank: 'number',
});

// Game state schema - synchronized to all clients
class GameState extends Schema {
  players: MapSchema<PlayerState> = new MapSchema<PlayerState>();
  round: number = 0;
  roundTimer: number = config.roundDuration / 1000; // seconds remaining
  phase: string = 'choosing'; // choosing / resolving / finished
  nakamaMatchId: string = '';       // For result callback
  minigameInstanceId: string = '';  // For filterBy matching (joinOrCreate routing)
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

export class DilemmaRaceRoom extends Room<GameState> {
  maxClients = 4;

  // Shared secret for token validation
  private secret: string = '';

  // Nakama callback URL
  private nakamaCallbackUrl: string = '';

  // Nakama match ID (runtime match ID for result callback routing)
  private nakamaMatchId: string = '';

  // Round timer reference
  private roundTimerRef: any = null;

  override async onCreate(options: {
    player_id: string;
    nakama_match_id: string;
    minigame_instance_id: string;
    token: string;
    players?: string[];            // Full player list (sent by clients)
    nakama_callback?: string;
    secret?: string;
  }): Promise<void> {
    this.setState(new GameState());
    // Prevent room auto-disposal when all clients leave, so the game timer
    // can finish and send result callback to Nakama.
    // Room disposes itself via this.disconnect() after sending results.
    this.autoDispose = false;

    this.secret = options.secret || config.secret;
    this.nakamaCallbackUrl = options.nakama_callback || config.nakamaRpcUrl;
    this.nakamaMatchId = options.nakama_match_id;
    this.state.nakamaMatchId = this.nakamaMatchId;
    this.state.minigameInstanceId = options.minigame_instance_id;

    // Initialize player states from the full player list
    // This ensures all players are pre-registered before they join individually
    const players = options.players || [options.player_id];
    for (const pid of players) {
      const player = new PlayerState();
      player.playerId = pid;
      this.state.players.set(pid, player);
    }

    // Set patch rate for state synchronization
    this.setPatchRate(50); // 50ms

    // Safety timeout: force finish after max duration
    this.clock.setTimeout(() => this.forceFinish(), config.maxGameDuration);

    // Register message handler for player choices (Colyseus v0.15 pattern)
    this.onMessage('choice', (client, message: { choice: number }) => {
      if (this.state.phase !== 'choosing') {
        return;
      }

      const playerId = client.auth?.playerId;
      if (!playerId) {
        return;
      }

      const player = this.state.players.get(playerId);
      if (!player || player.finished) {
        return;
      }

      if ([1, 3, 5].includes(message.choice)) {
        player.choice = message.choice;
      }
    });

    // Start first round
    this.startRound();
  }

  // onAuth validates HMAC token before allowing a client to join.
  // Hard rejection: returns false for invalid tokens, preventing unauthorized access.
  override onAuth(client: Client, options: {
    player_id?: string;
    token?: string;
    minigame_instance_id?: string;
    nakama_match_id?: string;
  }): boolean | { playerId: string } {
    // Required fields must be present
    if (!options.player_id || !options.token || !options.minigame_instance_id) {
      console.warn('onAuth: missing required fields (player_id, token, minigame_instance_id)');
      return false;
    }

    // Instance ID must match the room's instance ID (filterBy routing)
    if (options.minigame_instance_id !== this.state.minigameInstanceId) {
      console.warn('onAuth: minigame_instance_id mismatch, expected=%s, got=%s',
        this.state.minigameInstanceId, options.minigame_instance_id);
      return false;
    }

    // HMAC-SHA256 verification: token = hmac_sha256(secret, player_id:nakama_match_id:minigame_instance_id)
    const message = `${options.player_id}:${options.nakama_match_id || ''}:${options.minigame_instance_id}`;
    const expected = hmacSha256(this.secret, message);
    if (!hmacEqual(options.token, expected)) {
      console.warn('onAuth: invalid HMAC token for player=%s', options.player_id);
      return false;
    }

    // Player must be in the room's pre-registered player list
    if (!this.state.players.has(options.player_id)) {
      console.warn('onAuth: unknown player_id=%s (not in room player list)', options.player_id);
      return false;
    }

    // Auth successful - return auth data that will be stored on client.auth
    return { playerId: options.player_id };
  }

  override onJoin(client: Client, options: { token?: string }): void {
    // Auth result is already set by onAuth (client.auth = { playerId: ... })
    if (!client.auth?.playerId) {
      console.error('onJoin: no auth data on client (should not happen after onAuth)');
    }
  }

  override onLeave(client: Client, consented: boolean): void {
    // Player left - they keep their current position
    // Their choice defaults to step 1 in resolveRound
    console.log(`Player left: ${client.sessionId}, consented: ${consented}`);
  }

  // Start a new round
  private startRound(): void {
    this.state.round++;
    this.state.phase = 'choosing';
    this.state.roundTimer = config.roundDuration / 1000;

    // Reset choices for all players
    for (const [, player] of this.state.players) {
      if (!player.finished) {
        player.choice = 0;
        player.blocked = false;
      }
    }

    // Countdown timer
    this.roundTimerRef = this.clock.setTimeout(() => {
      this.resolveRound();
    }, config.roundDuration);
  }

  // Resolve the current round
  private resolveRound(): void {
    this.state.phase = 'resolving';

    // Collect choices per step value
    const choiceCounts: Map<number, string[]> = new Map();
    for (const [pid, player] of this.state.players) {
      if (player.finished) {
        continue;
      }
      // Default: step 1 if no choice submitted
      const choice = player.choice || 1;
      if (!choiceCounts.has(choice)) {
        choiceCounts.set(choice, []);
      }
      choiceCounts.get(choice)!.push(pid);
    }

    // Apply movement: blocked if >=2 players chose same value
    for (const [choice, pids] of choiceCounts) {
      const blocked = pids.length >= 2;
      for (const pid of pids) {
        const player = this.state.players.get(pid);
        if (!player) {
          continue;
        }
        player.blocked = blocked;
        if (!blocked) {
          player.position += choice;
          if (player.position >= 15) {
            player.position = 15;
            player.finished = true;
          }
        }
      }
    }

    // Check if game should end
    const finishedPlayers = [...this.state.players.values()].filter(p => p.finished);
    const maxRoundsReached = this.state.round >= 20;

    if (finishedPlayers.length > 0 || maxRoundsReached) {
      this.finishGame();
    } else {
      // Brief pause then next round
      this.clock.setTimeout(() => this.startRound(), 2_000);
    }
  }

  // Finish the game and send results to Nakama
  private finishGame(): void {
    this.state.phase = 'finished';

    // Calculate rankings by position descending
    const sorted = [...this.state.players.values()].sort((a, b) => b.position - a.position);

    // Assign ranks (tie-breaking: same position = same rank)
    const rankings: { player_id: string; rank: number }[] = [];
    let currentRank = 1;
    for (let i = 0; i < sorted.length; i++) {
      // Tie-breaking: if same position as previous, same rank
      if (i > 0 && sorted[i].position < sorted[i - 1].position) {
        currentRank = i + 1;
      }
      sorted[i].rank = currentRank;
      rankings.push({
        player_id: sorted[i].playerId,
        rank: currentRank,
      });
    }

    // Send result to Nakama via RPC
    this.sendResultToNakama(rankings);
  }

  // Send game result to Nakama RPC endpoint
  private async sendResultToNakama(rankings: { player_id: string; rank: number }[]): Promise<void> {
    try {
      // Nakama RPC requires http_key as query parameter for server-to-server auth
      const rpcUrl = `${this.nakamaCallbackUrl}?http_key=${config.nakamaHttpKey}`;
      const response = await fetch(rpcUrl, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          match_id: this.nakamaMatchId,
          room_id: this.roomId,
          game_type: 'dilemma_race',
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

    // Auto-dispose room after result sent
    this.disconnect();
  }

  // Force finish - safety timeout handler
  private forceFinish(): void {
    if (this.state.phase === 'finished') {
      return; // Already finished
    }
    console.warn('Force finishing game due to timeout');
    this.finishGame();
  }
}
