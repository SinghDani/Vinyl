create table songs (
    id serial primary key,
    name text unique not null
);

create table hashes (
    hash integer not null,
    songId integer not null references songs(id) on delete cascade,
    anchorTime integer not null
);
