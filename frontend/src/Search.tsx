import type { Song } from "./types";

type Props = {
  songs: Song[];
  setFilteredSongs: React.Dispatch<React.SetStateAction<Song[]>>;
};

export default function Search(props: Props) {
  const { songs, setFilteredSongs } = props;
  return (
    <>
      <span>/</span>
      <input
        placeholder="search song..."
        onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
          e.preventDefault();
          setFilteredSongs(
            songs.filter((song) =>
              `${song.name} ${song.artist}`
                .toLowerCase()
                .includes(e.target.value),
            ),
          );
        }}
      />
    </>
  );
}
