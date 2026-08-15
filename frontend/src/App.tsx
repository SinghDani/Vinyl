import Header from "./Header";
import Recorder from "./Recorder";
import SongList from "./SongList";

export default function App() {
  return (
    <main className="page">
      <Header />
      <Recorder />
      <hr className="divider" />
      <SongList />
    </main>
  );
}
