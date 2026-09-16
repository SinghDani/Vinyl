import { useEffect, useState } from "react";
import { API_BASE_URL } from "./api-config";
import Search from "./Search";
import type { Song } from "./types";

export default function SongList() {
  const [songs, setSongs] = useState<Song[]>([]);
  const [filteredSongs, setFilteredSongs] = useState<Song[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [err, setErr] = useState<boolean>(false);

  useEffect(() => {
    const getSongs = async () => {
      try {
        const res = await fetch(`${API_BASE_URL}/songs`);
        const songs: Song[] = await res.json();
        setSongs(songs);
        setFilteredSongs(songs);
        setLoading(false);
      } catch (err) {
        console.log(err);
        setErr(true);
      }
    };

    getSongs();
  }, []);

  return (
    <section>
      <Search songs={songs} setFilteredSongs={setFilteredSongs} />
      <table className="board-table">
        <thead>
          <tr>
            <th>#</th>
            <th>Song</th>
          </tr>
        </thead>
        <tbody>
          {err ? (
            <tr>
              <td colSpan={2} className="table-status">
                Error: could not fetch songs
              </td>
            </tr>
          ) : loading ? (
            <tr>
              <td colSpan={2} className="table-status">
                loading...
              </td>
            </tr>
          ) : (
            filteredSongs.map((song, id) => (
              <tr key={`${song.name}-${song.artist}-${id}`}>
                <td className="row-num">{id + 1}</td>
                <td>
                  <span className="row-song-name">{song.name}</span>
                  <span className="row-song-sep">-</span>
                  <span className="row-song-artist">{song.artist}</span>
                </td>
              </tr>
            ))
          )}
        </tbody>
      </table>
    </section>
  );
}
