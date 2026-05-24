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

  // Maximum game duration (ms) - force finish after this timeout
  maxGameDuration: 120_000, // 2 minutes

  // Round duration (ms) - time per round for player choices
  roundDuration: 10_000, // 10 seconds
};