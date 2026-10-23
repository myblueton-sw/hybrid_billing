CREATE SCHEMA hb_focus;
CREATE TABLE hb_focus.migration(version integer PRIMARY KEY,digest text NOT NULL);
CREATE TABLE hb_focus.configuration(id integer PRIMARY KEY CHECK(id=1),digest text NOT NULL,body jsonb NOT NULL);
CREATE TABLE hb_focus.tenant(tenant_id uuid PRIMARY KEY,properties jsonb NOT NULL,active boolean NOT NULL,lock_token integer NOT NULL DEFAULT 0 CHECK(lock_token=0));
CREATE TABLE hb_focus.gate(source text PRIMARY KEY,current_batch uuid,revision bigint NOT NULL DEFAULT 0 CHECK(revision>=0),lock_token integer NOT NULL DEFAULT 0 CHECK(lock_token=0),busy boolean NOT NULL DEFAULT false);
CREATE TABLE hb_focus.batch(batch_id uuid PRIMARY KEY,source text NOT NULL REFERENCES hb_focus.gate,source_revision bigint NOT NULL CHECK(source_revision>0),file_digest text NOT NULL,config_digest text NOT NULL,mapping_digest text NOT NULL,manifest jsonb NOT NULL,manifest_digest text NOT NULL,state text NOT NULL CHECK(state IN ('pending','unknown','verified','sealed')),UNIQUE(source,source_revision),UNIQUE(source,batch_id));
CREATE UNIQUE INDEX one_unfinished_batch ON hb_focus.batch(source) WHERE state<>'sealed';
ALTER TABLE hb_focus.gate ADD FOREIGN KEY(source,current_batch) REFERENCES hb_focus.batch(source,batch_id);
CREATE TABLE hb_focus.slice(tenant_id uuid NOT NULL REFERENCES hb_focus.tenant,batch_id uuid NOT NULL REFERENCES hb_focus.batch,row_count bigint NOT NULL CHECK(row_count>=0),row_digest text NOT NULL,controls jsonb NOT NULL,metadata jsonb NOT NULL,PRIMARY KEY(tenant_id,batch_id));
CREATE TABLE hb_focus.publication(tenant_id uuid NOT NULL,source text NOT NULL REFERENCES hb_focus.gate,batch_id uuid NOT NULL,PRIMARY KEY(tenant_id,source),FOREIGN KEY(tenant_id,batch_id) REFERENCES hb_focus.slice,FOREIGN KEY(source,batch_id) REFERENCES hb_focus.batch(source,batch_id));
CREATE TABLE hb_focus.confirmation(batch_id uuid PRIMARY KEY REFERENCES hb_focus.batch,manifest_digest text NOT NULL,actor text NOT NULL);
CREATE TABLE hb_focus.decision(source text NOT NULL,batch_id uuid PRIMARY KEY,predecessor uuid,actor text NOT NULL,FOREIGN KEY(source,batch_id) REFERENCES hb_focus.batch(source,batch_id),FOREIGN KEY(source,predecessor) REFERENCES hb_focus.batch(source,batch_id));
CREATE FUNCTION hb_focus.immutable() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'immutable evidence'; END $$;
CREATE TRIGGER immutable_slice BEFORE UPDATE OR DELETE ON hb_focus.slice FOR EACH ROW EXECUTE FUNCTION hb_focus.immutable();
CREATE TRIGGER immutable_confirmation BEFORE UPDATE OR DELETE ON hb_focus.confirmation FOR EACH ROW EXECUTE FUNCTION hb_focus.immutable();
CREATE TRIGGER immutable_decision BEFORE UPDATE OR DELETE ON hb_focus.decision FOR EACH ROW EXECUTE FUNCTION hb_focus.immutable();
CREATE TRIGGER immutable_configuration BEFORE UPDATE OR DELETE ON hb_focus.configuration FOR EACH ROW EXECUTE FUNCTION hb_focus.immutable();
CREATE FUNCTION hb_focus.batch_transition() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='DELETE' THEN RAISE EXCEPTION 'immutable batch'; END IF;
 IF (to_jsonb(NEW)-'state') IS DISTINCT FROM (to_jsonb(OLD)-'state') THEN RAISE EXCEPTION 'immutable manifest'; END IF;
 IF NOT ((current_user='hb_focus_ingest' AND OLD.state='pending' AND NEW.state IN ('unknown','verified')) OR (current_user='hb_focus_approve' AND OLD.state='verified' AND NEW.state='sealed')) THEN RAISE EXCEPTION 'batch transition'; END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER batch_transition BEFORE UPDATE OR DELETE ON hb_focus.batch FOR EACH ROW EXECUTE FUNCTION hb_focus.batch_transition();
