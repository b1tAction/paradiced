// TypingSpeedRoom - 打字速度 (Typing Speed) mini-game
//
// Game rules:
// - Turn-free multiplayer racing action
// - 3-second rules explanation phase (automatic)
// - 3-second countdown preparation phase (3, 2, 1)
// - Displays a randomly selected classic sentence from Zhu Ziqing's "Spring"
// - Captures real-time keyboard inputs; completed characters colored grey, untyped green
// - Syncs all players' typing progress percents and displays progress lanes
// - Game ends when all players complete typing or 60s timeout expires
// - Sorts by completion speed (finishTimeMs ascending), then progress percentage descending

import { Room, Client } from '@colyseus/core';
import { Schema, MapSchema, defineTypes } from '@colyseus/schema';
import { createHmac } from 'crypto';
import { config } from '../config';

// Predefined beautiful sentences from "春"
const SPRING_SENTENCES = [
  "盼望着，盼望着，东风来了，春天的脚步近了。",
  "一切都像刚睡醒的样子，欣欣然张开了眼。",
  "山朗润起来了，水涨起来了，太阳的脸红起来了。",
  "小草偷偷地从土里钻出来，嫩嫩的，绿绿的。",
  "园子里，田野里，瞧去，一大片一大片满是的。",
  "红的像火，粉的像霞，白的像雪。",
  "花下成千成百的蜜蜂嗡嗡地闹着，大小的蝴蝶飞来飞去。",
  "风里带来些新翻的泥土的气息，混着青草味儿。",
  "鸟儿将窠巢安在繁花嫩叶当中，高兴起来了。",
  "雨是最寻常的，一下就是三两天。可别恼。",
  "天上风筝渐渐多了，地上孩子也多了。",
  "一年之计在于春，刚起头儿，有的是工夫，有的是希望。"
];

// Player state schema - synchronized to all clients
class TypingSpeedPlayerState extends Schema {
  playerId: string = '';
  typedCount: number = 0;       // Number of correct characters typed
  progressPercent: number = 0;  // Progress (0 to 100)
  finishTimeMs: number = 0;     // Timestamp when finished (0 = not finished)
  rank: number = 0;             // Final ranking
}
defineTypes(TypingSpeedPlayerState, {
  playerId: 'string',
  typedCount: 'number',
  progressPercent: 'number',
  finishTimeMs: 'number',
  rank: 'number',
});

// Game state schema - synchronized to all clients
class TypingSpeedGameState extends Schema {
  players: MapSchema<TypingSpeedPlayerState> = new MapSchema<TypingSpeedPlayerState>();
  phase: string = 'rules';      // rules / countdown / playing / finished
  roundTimer: number = 3;       // 3s rules -> 3s countdown -> 60s game
  targetText: string = '';      // Zhu Ziqing's random sentence
  nakamaMatchId: string = '';   // For result callback
  minigameInstanceId: string = '';
}
defineTypes(TypingSpeedGameState, {
  players: { map: TypingSpeedPlayerState },
  phase: 'string',
  roundTimer: 'number',
  targetText: 'string',
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

export class TypingSpeedRoom extends Room<TypingSpeedGameState> {
  maxClients = 4;

  private secret: string = '';
  private nakamaCallbackUrl: string = '';
  private nakamaMatchId: string = '';
  private timerRef: any = null;
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
    this.setState(new TypingSpeedGameState());
    this.autoDispose = false;

    this.secret = options.secret || config.secret;
    this.nakamaCallbackUrl = options.nakama_callback || config.nakamaRpcUrl;
    this.nakamaMatchId = options.nakama_match_id;
    this.state.nakamaMatchId = this.nakamaMatchId;
    this.state.minigameInstanceId = options.minigame_instance_id;

    // Pick a random Zhu Ziqing spring sentence
    const randIdx = Math.floor(Math.random() * SPRING_SENTENCES.length);
    this.state.targetText = SPRING_SENTENCES[randIdx];

    // Initialize player states from the full player list
    const players = options.players || [options.player_id];
    for (const pid of players) {
      const player = new TypingSpeedPlayerState();
      player.playerId = pid;
      player.typedCount = 0;
      player.progressPercent = 0;
      player.finishTimeMs = 0;
      player.rank = 0;
      this.state.players.set(pid, player);
    }

    this.setPatchRate(50); // 50ms state synchronization

    // Optional safety timeout
    if (config.maxGameDuration > 0) {
      this.clock.setTimeout(() => this.forceFinish(), config.maxGameDuration);
    }

    // Register message handler for live typing progress updates
    this.onMessage('submit_progress', (client, message: { typedCount: number }) => {
      if (this.state.phase !== 'playing') {
        return;
      }

      const playerId = client.auth?.playerId;
      if (!playerId) return;

      const player = this.state.players.get(playerId);
      if (!player) return;

      // Ensure progress is valid and not going backwards
      if (message.typedCount > player.typedCount && message.typedCount <= this.state.targetText.length) {
        player.typedCount = message.typedCount;
        player.progressPercent = Math.round((message.typedCount / this.state.targetText.length) * 100);

        // Completion check
        if (message.typedCount === this.state.targetText.length && player.finishTimeMs === 0) {
          player.finishTimeMs = Date.now();
          console.log(`[TypingSpeed] Player ${playerId} FINISHED typing!`);

          // Early finish: if all active players finished typing, complete the game early
          if (this.allPlayersFinished()) {
            this.finishGame();
          }
        }
      }
    });

    // Start 3s rules presentation phase automatically
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
    console.log(`[TypingSpeed] Player joined: ${client.auth?.playerId}`);
  }

  override onLeave(client: Client, consented: boolean): void {
    console.log(`[TypingSpeed] Player left: ${client.sessionId}, consented: ${consented}`);
  }

  // Rules phase: 3 seconds rules display
  private startRulesPhase(): void {
    this.clearTimer();
    this.state.phase = 'rules';
    this.state.roundTimer = 3;

    this.timerRef = this.clock.setInterval(() => {
      if (this.state.roundTimer > 0) {
        this.state.roundTimer--;
      } else {
        this.startCountdownPhase();
      }
    }, 1000);
  }

  // Countdown phase: 3 seconds countdown (3, 2, 1)
  private startCountdownPhase(): void {
    this.clearTimer();
    this.state.phase = 'countdown';
    this.state.roundTimer = 3;

    this.timerRef = this.clock.setInterval(() => {
      if (this.state.roundTimer > 0) {
        this.state.roundTimer--;
      } else {
        this.startPlaying();
      }
    }, 1000);
  }

  // Playing phase: 60 seconds time limit
  private startPlaying(): void {
    this.clearTimer();
    this.state.phase = 'playing';
    this.state.roundTimer = 60;

    this.timerRef = this.clock.setInterval(() => {
      if (this.state.roundTimer > 0) {
        this.state.roundTimer--;
      } else {
        console.log('[TypingSpeed] Round timer expired.');
        this.finishGame();
      }
    }, 1000);
  }

  private allPlayersFinished(): boolean {
    for (const [, player] of this.state.players) {
      if (player.finishTimeMs === 0) {
        return false;
      }
    }
    return true;
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

    // Sort:
    // 1. Finished players sorted by finishTimeMs ascending (faster first)
    // 2. Incompleted players sorted by progressPercent descending (higher first), then typedCount descending
    const sorted = [...playerList].sort((a, b) => {
      const aFinished = a.finishTimeMs > 0;
      const bFinished = b.finishTimeMs > 0;

      if (aFinished && bFinished) {
        return a.finishTimeMs - b.finishTimeMs;
      }
      if (aFinished && !bFinished) return -1;
      if (!aFinished && bFinished) return 1;

      // Both incomplete
      if (b.progressPercent !== a.progressPercent) {
        return b.progressPercent - a.progressPercent;
      }
      return b.typedCount - a.typedCount;
    });

    // Compile rankings (assign strict ranks based on sorted result sequence)
    const rankings: { player_id: string; rank: number; game_data?: Record<string, any> }[] = [];
    let currentRank = 1;
    for (let i = 0; i < sorted.length; i++) {
      if (i > 0) {
        const prev = sorted[i - 1];
        const curr = sorted[i];
        const prevFinished = prev.finishTimeMs > 0;
        const currFinished = curr.finishTimeMs > 0;

        let rankDiff = false;
        if (prevFinished !== currFinished) {
          rankDiff = true;
        } else if (prevFinished && currFinished) {
          rankDiff = prev.finishTimeMs !== curr.finishTimeMs;
        } else {
          rankDiff = prev.progressPercent !== curr.progressPercent || prev.typedCount !== curr.typedCount;
        }

        if (rankDiff) {
          currentRank = i + 1;
        }
      }
      sorted[i].rank = currentRank;
      rankings.push({
        player_id: sorted[i].playerId,
        rank: currentRank,
        game_data: {
          score: sorted[i].progressPercent,
        },
      });
    }

    console.log(`[TypingSpeed] Game finished. Rankings:`, rankings);
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
          game_type: 'typing_speed',
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
    console.warn('[TypingSpeed] Force finishing game due to timeout');
    this.finishGame();
  }
}
