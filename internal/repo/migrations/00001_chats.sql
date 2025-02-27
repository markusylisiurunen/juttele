-- chats table
create table chats (
  chat_id integer primary key,
  chat_created_at text not null,
  chat_title text not null,
  chat_pinned boolean not null
);

-- chat_events table
create table chat_events (
  chat_id integer not null references chats (chat_id),
  chat_event_id integer primary key,
  chat_event_created_at text not null,
  chat_event_kind text not null,
  chat_event_content text not null,
  constraint check_valid_json check (json_valid(chat_event_content))
);

create index chat_events_chat_id_created_at
on chat_events (chat_id, chat_event_created_at);
