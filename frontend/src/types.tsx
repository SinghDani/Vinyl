export type Song = {
  name: string;
  artist: string;
};

export type MatchingSong = {
  confidence: string;
} & Song;

export type GeneratedHash = {
  hash: number;
  anchorTime: number;
};

export type WorkerMatch = {
  songName: string;
  artist: string;
  match: boolean;
  verdict: string;
};

export type WorkerMessage =
  | { type: "ready"; sampleRate: number }
  | { type: "result"; result: WorkerMatch }
  | { type: "error"; message: string };
