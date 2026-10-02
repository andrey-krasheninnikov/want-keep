ALTER TABLE want_keep.connections
 ADD COLUMN revision want_keep.revision NOT NULL DEFAULT 1,
 ADD COLUMN history_from date,
 ADD COLUMN connection_state text NOT NULL DEFAULT 'pending' CHECK(connection_state IN ('pending','connected','reauth_required','disconnected','failed'));

CREATE TABLE want_keep.oauth_attempts (
 household_id uuid NOT NULL, id uuid NOT NULL, connection_id uuid NOT NULL, owner_id uuid NOT NULL,
 session_hash text NOT NULL, state_hash text NOT NULL UNIQUE CHECK(state_hash ~ '^[0-9a-f]{64}$'),
 generation want_keep.revision NOT NULL, connection_revision want_keep.revision NOT NULL,
 expires_at timestamptz NOT NULL, phase text NOT NULL CHECK(phase IN ('ready','claimed','completed','failed','unknown')),
 ciphertext bytea NOT NULL,
 PRIMARY KEY(household_id,id), FOREIGN KEY(household_id,connection_id) REFERENCES want_keep.connections(household_id,id),
 FOREIGN KEY(household_id,owner_id) REFERENCES want_keep.memberships(household_id,user_id)
);
CREATE TABLE want_keep.token_rotations (
 household_id uuid NOT NULL, connection_id uuid NOT NULL, secret_revision want_keep.revision NOT NULL,
 generation want_keep.revision NOT NULL, attempt_id uuid NOT NULL, phase text NOT NULL CHECK(phase IN ('claimed','completed','failed','unknown')),
 PRIMARY KEY(household_id,connection_id,secret_revision),
 FOREIGN KEY(household_id,connection_id) REFERENCES want_keep.connections(household_id,id)
);
CREATE TABLE want_keep.raiffeisen_reports (
 household_id uuid NOT NULL, connection_id uuid NOT NULL, generation want_keep.revision NOT NULL,
 request_key text NOT NULL CHECK(length(request_key) BETWEEN 1 AND 2000),
 attempt_id uuid NOT NULL, report_id uuid, phase text NOT NULL CHECK(phase IN ('claimed','requested','completed','no_statements','failed','unknown')),
 requested_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(household_id,connection_id,generation,request_key),
 FOREIGN KEY(household_id,connection_id) REFERENCES want_keep.connections(household_id,id)
);
CREATE TABLE want_keep.source_aliases (
 household_id uuid NOT NULL REFERENCES want_keep.households(id), provider text NOT NULL CHECK(provider='raiffeisen'), external_account_id text NOT NULL, log text NOT NULL,
 alias_digest bytea NOT NULL CHECK(octet_length(alias_digest)=32), kind text NOT NULL, value text NOT NULL,
 canonical_id text NOT NULL, ambiguous boolean NOT NULL DEFAULT false,
 PRIMARY KEY(household_id,provider,alias_digest)
);
CREATE TABLE want_keep.raiffeisen_fact_versions (
 household_id uuid NOT NULL REFERENCES want_keep.households(id), identity_digest bytea NOT NULL CHECK(octet_length(identity_digest)=32),
 full_identity text NOT NULL, payload_hash text NOT NULL CHECK(payload_hash ~ '^[0-9a-f]{64}$'),
 source_at timestamptz NOT NULL, source_ns smallint NOT NULL CHECK(source_ns BETWEEN 0 AND 999),
 PRIMARY KEY(household_id,identity_digest)
);
CREATE TABLE want_keep.connection_events (
 household_id uuid NOT NULL, id uuid NOT NULL, connection_id uuid NOT NULL, actor_id uuid NOT NULL,
 revision want_keep.revision NOT NULL, event_type text NOT NULL CHECK(event_type IN ('created','sync_requested','reauth_requested','disconnected','authorized','authorization_failed','rotation_completed','rotation_unknown')),
 command_id uuid, at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(household_id,id), FOREIGN KEY(household_id,connection_id) REFERENCES want_keep.connections(household_id,id),
 FOREIGN KEY(household_id,actor_id) REFERENCES want_keep.memberships(household_id,user_id)
);
CREATE TRIGGER immutable_history BEFORE UPDATE OR DELETE ON want_keep.connection_events FOR EACH ROW EXECUTE FUNCTION want_keep.reject_history_change();
GRANT SELECT,INSERT ON want_keep.oauth_attempts,want_keep.token_rotations,want_keep.raiffeisen_reports,want_keep.source_aliases,want_keep.connection_events TO want_keep_app;
GRANT UPDATE(revision,history_from,connection_state) ON want_keep.connections TO want_keep_app;
GRANT UPDATE(phase) ON want_keep.oauth_attempts TO want_keep_app;
GRANT UPDATE(phase) ON want_keep.token_rotations TO want_keep_app;
GRANT UPDATE(phase,report_id) ON want_keep.raiffeisen_reports TO want_keep_app;
GRANT UPDATE(ambiguous) ON want_keep.source_aliases TO want_keep_app;
GRANT SELECT,INSERT ON want_keep.raiffeisen_fact_versions TO want_keep_app;
GRANT UPDATE(payload_hash,source_at,source_ns) ON want_keep.raiffeisen_fact_versions TO want_keep_app;
