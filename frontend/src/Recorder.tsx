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
    <section className="hero-row">
      <div>
        <p className="record-label">Identify a song</p>
        <RecordButton setMatch={setMatch} />
      </div>

      <div>
        <p className="match-label">Last match</p>
        <Match match={match} />
      </div>
    </section>
  );
}
