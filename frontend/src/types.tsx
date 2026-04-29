export type Song = {
  name: string;
  artist: string;
};

export type MatchingSong = {
  confidence: string;
} & Song;
