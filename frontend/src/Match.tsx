import type { MatchingSong } from "./types";

export default function Match(props: { match: MatchingSong }) {
  const { match } = props;
  const hasMatch = match.name.trim().length > 0;

  return (
    <div className={`match-card ${hasMatch ? "has-match" : ""}`}>
      {hasMatch ? (
        <>
          <div className="match-song">{match.name}</div>
          <div className="match-artist">{match.artist}</div>
          {match.confidence && (
            <div className="match-confidence">
              CONFIDENCE <span>{match.confidence}</span>
            </div>
          )}
        </>
      ) : (
        <div className="match-empty">-</div>
      )}
    </div>
  );
}
