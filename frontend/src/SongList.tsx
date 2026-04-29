import { useEffect, useState } from "react";
import Search from "./Search";
import type { Song } from "./types";

export default function SongList() {
  const [songs, setSongs] = useState<Song[]>([]);
  const [filteredSongs, setFilteredSongs] = useState<Song[]>([]);
  const [loading, setLoading] = useState<boolean>(true);

  useEffect(() => {
    const getSongs = async () => {
      try {
        const res = await fetch("http://localhost:3000/songs");
        const songs: Song[] = await res.json();
        setSongs(songs);
        setFilteredSongs(songs);
        setLoading(false);
      } catch (err) {
        console.log(err);
      }
    };

    getSongs();
  }, []);

  return (
    <>
      <div>Audio List</div>
      <Search songs={songs} setFilteredSongs={setFilteredSongs} />

      {
        <div>
          {loading
            ? "loading..."
            : filteredSongs.map((song, id) => (
                <div key={id}>
                  {song.name} - {song.artist}
                </div>
              ))}
        </div>
      }
    </>
  );
}
