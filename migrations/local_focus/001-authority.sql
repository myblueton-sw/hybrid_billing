CREATE SCHEMA hb_local;
CREATE TABLE hb_local.migration (version integer PRIMARY KEY, digest text NOT NULL);
CREATE TABLE hb_local.connection (
 installation_id uuid NOT NULL, workspace_id uuid NOT NULL, connection_id uuid NOT NULL,
 account text NOT NULL, subaccount text NOT NULL,
 period_start timestamptz NOT NULL, period_end timestamptz NOT NULL,
 policy_revision bigint NOT NULL CHECK (policy_revision > 0), active boolean NOT NULL,
 lock_token bigint NOT NULL DEFAULT 0 CHECK (lock_token=0),
 admission_fields jsonb NOT NULL DEFAULT '{"account":"BillingAccountId","subaccount":"SubAccountId","billing_start":"BillingPeriodStart","billing_end":"BillingPeriodEnd","charge_start":"ChargePeriodStart","charge_end":"ChargePeriodEnd"}',
 PRIMARY KEY (installation_id, workspace_id, connection_id), CHECK (period_start < period_end)
);
CREATE TABLE hb_local.principal (
 installation_id uuid NOT NULL, workspace_id uuid NOT NULL, actor text NOT NULL,
 connection_id uuid NOT NULL, rights text[] NOT NULL,
 PRIMARY KEY (installation_id, workspace_id, actor),
 FOREIGN KEY (installation_id, workspace_id, connection_id) REFERENCES hb_local.connection
);
CREATE TABLE hb_local.release (
 installation_id uuid NOT NULL, workspace_id uuid NOT NULL, release_id uuid NOT NULL,
 connection_id uuid NOT NULL, source_revision bigint NOT NULL CHECK (source_revision > 0),
 file_digest text NOT NULL CHECK (length(file_digest) = 64), registry_digest text NOT NULL CHECK (length(registry_digest) = 64),
 manifest_digest text NOT NULL CHECK (length(manifest_digest) = 64), manifest jsonb NOT NULL,
 creator text NOT NULL, state text NOT NULL CHECK (state IN ('pending','verified','sealed')),
 original_key text NOT NULL, created_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY (installation_id, workspace_id, release_id),
 UNIQUE (installation_id, workspace_id, connection_id, source_revision),
 UNIQUE (installation_id, workspace_id, connection_id, release_id),
 FOREIGN KEY (installation_id, workspace_id, connection_id) REFERENCES hb_local.connection,
 FOREIGN KEY (installation_id, workspace_id, creator) REFERENCES hb_local.principal
);
CREATE TABLE hb_local.publication_scope (
 installation_id uuid NOT NULL, workspace_id uuid NOT NULL, connection_id uuid NOT NULL,
 revision bigint NOT NULL DEFAULT 0 CHECK (revision >= 0), release_id uuid,
 PRIMARY KEY (installation_id, workspace_id, connection_id),
 FOREIGN KEY (installation_id, workspace_id, connection_id) REFERENCES hb_local.connection,
 FOREIGN KEY (installation_id, workspace_id, connection_id, release_id) REFERENCES hb_local.release (installation_id, workspace_id, connection_id, release_id)
);
CREATE TABLE hb_local.quarantine (
 installation_id uuid NOT NULL, workspace_id uuid NOT NULL, receipt_id uuid NOT NULL,
 connection_id uuid NOT NULL, source_revision bigint NOT NULL, declared_version text NOT NULL,
 file_digest text NOT NULL CHECK (length(file_digest)=64), reason text NOT NULL, object_key text NOT NULL,
 PRIMARY KEY (installation_id,workspace_id,receipt_id),
 UNIQUE (installation_id,workspace_id,connection_id,source_revision,declared_version,file_digest),
 FOREIGN KEY (installation_id,workspace_id,connection_id) REFERENCES hb_local.connection
);
CREATE TABLE hb_local.operation (
 installation_id uuid NOT NULL, workspace_id uuid NOT NULL, operation_id uuid NOT NULL,
 connection_id uuid NOT NULL, release_id uuid, quarantine_id uuid,
 kind text NOT NULL CHECK (kind IN ('load','export','quarantine')),
 state text NOT NULL CHECK (state IN ('pending','unknown','terminal')),
 PRIMARY KEY (installation_id, workspace_id, operation_id),
 FOREIGN KEY (installation_id, workspace_id, connection_id) REFERENCES hb_local.connection,
 FOREIGN KEY (installation_id, workspace_id, connection_id, release_id) REFERENCES hb_local.release (installation_id, workspace_id, connection_id, release_id),
 FOREIGN KEY (installation_id,workspace_id,quarantine_id) REFERENCES hb_local.quarantine,
 CHECK ((kind='quarantine' AND quarantine_id IS NOT NULL AND release_id IS NULL) OR (kind<>'quarantine' AND quarantine_id IS NULL AND release_id IS NOT NULL))
);
CREATE UNIQUE INDEX one_unfinished_operation ON hb_local.operation (installation_id, workspace_id, connection_id) WHERE state <> 'terminal';
CREATE TABLE hb_local.confirmation (
 installation_id uuid NOT NULL, workspace_id uuid NOT NULL, release_id uuid NOT NULL,
 manifest_digest text NOT NULL, policy_revision bigint NOT NULL, actor text NOT NULL,
 PRIMARY KEY (installation_id, workspace_id, release_id),
 FOREIGN KEY (installation_id, workspace_id, release_id) REFERENCES hb_local.release,
 FOREIGN KEY (installation_id, workspace_id, actor) REFERENCES hb_local.principal
);
CREATE TABLE hb_local.publication_decision (
 installation_id uuid NOT NULL, workspace_id uuid NOT NULL, connection_id uuid NOT NULL,
 revision bigint NOT NULL CHECK (revision > 0), release_id uuid NOT NULL,
 actor text NOT NULL, manifest_digest text NOT NULL,
 PRIMARY KEY (installation_id, workspace_id, connection_id, revision),
 UNIQUE (installation_id, workspace_id, release_id),
 FOREIGN KEY (installation_id, workspace_id, connection_id) REFERENCES hb_local.connection,
 FOREIGN KEY (installation_id, workspace_id, connection_id, release_id) REFERENCES hb_local.release (installation_id, workspace_id, connection_id, release_id),
 FOREIGN KEY (installation_id, workspace_id, actor) REFERENCES hb_local.principal
);
CREATE TABLE hb_local.outbox (
 installation_id uuid NOT NULL, workspace_id uuid NOT NULL, release_id uuid NOT NULL,
 state text NOT NULL CHECK (state IN ('pending','completed')),
 object_key text, object_digest text,
 PRIMARY KEY (installation_id, workspace_id, release_id),
 FOREIGN KEY (installation_id, workspace_id, release_id) REFERENCES hb_local.release
);
CREATE FUNCTION hb_local.preserve_release() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP = 'DELETE' THEN RAISE EXCEPTION 'immutable release'; END IF;
 IF ROW(NEW.installation_id,NEW.workspace_id,NEW.release_id,NEW.connection_id,NEW.source_revision,NEW.file_digest,NEW.registry_digest,NEW.manifest_digest,NEW.manifest,NEW.creator,NEW.original_key,NEW.created_at)
 IS DISTINCT FROM ROW(OLD.installation_id,OLD.workspace_id,OLD.release_id,OLD.connection_id,OLD.source_revision,OLD.file_digest,OLD.registry_digest,OLD.manifest_digest,OLD.manifest,OLD.creator,OLD.original_key,OLD.created_at)
 THEN RAISE EXCEPTION 'immutable manifest'; END IF;
 IF NOT ((OLD.state='pending' AND NEW.state='verified') OR (OLD.state='verified' AND NEW.state='sealed')) THEN RAISE EXCEPTION 'release transition'; END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER preserve_release BEFORE UPDATE OR DELETE ON hb_local.release FOR EACH ROW EXECUTE FUNCTION hb_local.preserve_release();
CREATE FUNCTION hb_local.append_only() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN RAISE EXCEPTION 'append only'; END $$;
CREATE TRIGGER preserve_confirmation BEFORE UPDATE OR DELETE ON hb_local.confirmation FOR EACH ROW EXECUTE FUNCTION hb_local.append_only();
CREATE TRIGGER preserve_decision BEFORE UPDATE OR DELETE ON hb_local.publication_decision FOR EACH ROW EXECUTE FUNCTION hb_local.append_only();
CREATE TRIGGER preserve_quarantine BEFORE UPDATE OR DELETE ON hb_local.quarantine FOR EACH ROW EXECUTE FUNCTION hb_local.append_only();
