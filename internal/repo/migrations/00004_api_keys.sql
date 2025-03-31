-- create the api keys table
create table api_keys (
  api_key_id integer primary key,
  api_key_created_at text not null,
  api_key_expires_at text not null,
  api_key_uuid text not null,
  constraint unique_api_key_uuid unique (api_key_uuid)
);
