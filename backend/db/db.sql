create table if not exists songs (
    id serial primary key,
    name text unique not null,
    artist text not null
);
create table if not exists fingerprints (
    hash integer not null,
    song_id integer not null references songs(id) on delete cascade,
    anchor_time integer not null
);
