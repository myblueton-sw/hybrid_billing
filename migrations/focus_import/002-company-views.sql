CREATE TABLE hb_focus.audience(actor text PRIMARY KEY,source text NOT NULL REFERENCES hb_focus.gate,company_id text NOT NULL,expected_tenants uuid[] NOT NULL,config_digest text NOT NULL,active boolean NOT NULL,lock_token integer NOT NULL DEFAULT 0 CHECK(lock_token=0));
CREATE TABLE hb_focus.audience_member(actor text NOT NULL REFERENCES hb_focus.audience,tenant_id uuid NOT NULL REFERENCES hb_focus.tenant,PRIMARY KEY(actor,tenant_id));
CREATE FUNCTION hb_focus.audience_identity() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='DELETE' OR (to_jsonb(NEW)-'active'-'lock_token') IS DISTINCT FROM (to_jsonb(OLD)-'active'-'lock_token') THEN RAISE EXCEPTION 'immutable audience identity'; END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER audience_identity BEFORE UPDATE OR DELETE ON hb_focus.audience FOR EACH ROW EXECUTE FUNCTION hb_focus.audience_identity();
CREATE FUNCTION hb_focus.member_lock() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='UPDATE' AND NEW.actor IS DISTINCT FROM OLD.actor THEN RAISE EXCEPTION 'immutable member audience'; END IF;
 PERFORM 1 FROM hb_focus.audience WHERE actor=CASE WHEN TG_OP='DELETE' THEN OLD.actor ELSE NEW.actor END FOR UPDATE;
 IF TG_OP='DELETE' THEN RETURN OLD; END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER member_lock BEFORE INSERT OR UPDATE OR DELETE ON hb_focus.audience_member FOR EACH ROW EXECUTE FUNCTION hb_focus.member_lock();
