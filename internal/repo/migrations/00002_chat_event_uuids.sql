-- remove all data
delete from chat_events;
delete from chats;

-- re-create the chat_events table
drop table chat_events;
create table chat_events (
  chat_id integer not null references chats (chat_id) on delete cascade,
  chat_event_id integer primary key,
  chat_event_created_at text not null,
  chat_event_uuid text not null,
  chat_event_kind text not null,
  chat_event_content text not null,
  constraint check_valid_json check (json_valid(chat_event_content)),
  constraint unique_chat_event_uuid unique (chat_event_uuid)
);

create index chat_events_chat_id_created_at
on chat_events (chat_id, chat_event_created_at);
