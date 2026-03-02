create table songs (
    id serial primary key,
    name text unique not null
);
create table hashes (
    hash integer not null,
    song_id integer not null references songs(id) on delete cascade,
    anchor_time integer not null
);
