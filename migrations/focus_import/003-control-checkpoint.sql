CREATE TABLE hb_focus.control_checkpoint(id integer PRIMARY KEY CHECK(id=1),installation uuid NOT NULL,sequence bigint NOT NULL CHECK(sequence>0),digest text NOT NULL CHECK(digest ~ '^[a-f0-9]{64}$'));
ALTER TABLE hb_focus.control_checkpoint ENABLE ROW LEVEL SECURITY;
ALTER TABLE hb_focus.control_checkpoint FORCE ROW LEVEL SECURITY;
