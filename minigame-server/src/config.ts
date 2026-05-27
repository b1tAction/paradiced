function parseDurationMs(value: string | undefined, fallback: number): number {
  if (value === undefined || value.trim() === '') {
    return fallback;
  }
  const parsed = Number.parseInt(value, 10);
  return Number.isFinite(parsed) ? parsed : fallback;
}

// Configuration for Paradiced mini-game server
export const config = {
  // Nakama RPC endpoint URL for result callback
  nakamaRpcUrl: process.env.NAKAMA_RPC_URL || 'http://nakama:7350/v2/rpc/minigame_result_callback',

  // Nakama HTTP key for RPC authentication
  nakamaHttpKey: process.env.NAKAMA_HTTP_KEY || 'defaultkey',

  // Shared secret for HMAC token validation and RPC callback authentication
  secret: process.env.COLYSEUS_SECRET || 'change_me_in_production',

  // Colyseus server port
  port: parseInt(process.env.PORT || '2567', 10),

  // Maximum game duration (ms). 0 disables the safety force-finish timer.
  maxGameDuration: parseDurationMs(process.env.MAX_GAME_DURATION_MS, 0),

  // Round duration (ms) - time per round for player choices
  roundDuration: parseDurationMs(process.env.ROUND_DURATION_MS, 10_000), // 10 seconds
};
