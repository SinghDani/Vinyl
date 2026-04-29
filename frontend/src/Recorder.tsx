import { useState } from "react";
import RecordButton from "./RecordButton";
import Match from "./Match";
import type { MatchingSong } from "./types";

export default function Recorder() {
  const [match, setMatch] = useState<MatchingSong>({
    name: "",
    artist: "",
    confidence: "",
  });

  return (
    <>
      <div>IDENTIFY A SONG</div>
      <RecordButton setMatch={setMatch} />

      <div>LAST MATCH</div>
      <Match match={match} />
    </>
  );
}
