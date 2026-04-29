import type { MatchingSong } from "./types";

type Props = {
  match: MatchingSong;
};

export default function Match(props: Props) {
  const { match } = props;

  return (
    <>
      <div>
        <div>{match.name || "-"}</div>
        <div>{match.artist}</div>
        <div>CONFIDENCE{match.confidence}</div>
      </div>
    </>
  );
}
