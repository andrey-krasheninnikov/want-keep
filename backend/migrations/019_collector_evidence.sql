CREATE TABLE want_keep.collector_evidence_batches (
 household_id uuid NOT NULL,
 page_reference text NOT NULL CHECK(length(page_reference) BETWEEN 1 AND 2000),
 job_id uuid NOT NULL,
 fetched_at timestamptz NOT NULL,
 fetched_ns smallint NOT NULL CHECK(fetched_ns BETWEEN 0 AND 999),
 disposition text NOT NULL CHECK(disposition IN ('staged','applied','rejected_result','stale_result','provider_outcome')),
 PRIMARY KEY(household_id,page_reference),
 FOREIGN KEY(household_id,job_id) REFERENCES want_keep.jobs(household_id,id)
);

CREATE INDEX collector_evidence_batches_staged_idx
 ON want_keep.collector_evidence_batches(household_id,fetched_at,page_reference)
 WHERE disposition='staged';

CREATE TABLE want_keep.collector_evidence_items (
 household_id uuid NOT NULL,
 page_reference text NOT NULL,
 reference text NOT NULL CHECK(length(reference) BETWEEN 1 AND 2000),
 ciphertext bytea NOT NULL CHECK(octet_length(ciphertext) BETWEEN 1 AND 20971520),
 PRIMARY KEY(household_id,page_reference,reference),
 FOREIGN KEY(household_id,page_reference) REFERENCES want_keep.collector_evidence_batches(household_id,page_reference)
);

CREATE FUNCTION want_keep.enforce_collector_evidence_disposition() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NEW.household_id<>OLD.household_id OR NEW.page_reference<>OLD.page_reference OR NEW.job_id<>OLD.job_id OR NEW.fetched_at<>OLD.fetched_at OR NEW.fetched_ns<>OLD.fetched_ns THEN
  RAISE EXCEPTION 'collector evidence metadata is immutable';
 END IF;
 IF NEW.disposition=OLD.disposition THEN
  RETURN NEW;
 END IF;
 IF OLD.disposition<>'staged' OR NEW.disposition='staged' THEN
  RAISE EXCEPTION 'collector evidence disposition is terminal';
 END IF;
 RETURN NEW;
END $$;

CREATE TRIGGER collector_evidence_disposition
 BEFORE UPDATE ON want_keep.collector_evidence_batches
 FOR EACH ROW EXECUTE FUNCTION want_keep.enforce_collector_evidence_disposition();
CREATE TRIGGER immutable_history
 BEFORE DELETE ON want_keep.collector_evidence_batches
 FOR EACH ROW EXECUTE FUNCTION want_keep.reject_history_change();
CREATE TRIGGER immutable_history
 BEFORE UPDATE OR DELETE ON want_keep.collector_evidence_items
 FOR EACH ROW EXECUTE FUNCTION want_keep.reject_history_change();

REVOKE ALL ON FUNCTION want_keep.enforce_collector_evidence_disposition() FROM PUBLIC;
GRANT SELECT,INSERT ON want_keep.collector_evidence_batches TO want_keep_app;
GRANT UPDATE(disposition) ON want_keep.collector_evidence_batches TO want_keep_app;
GRANT SELECT,INSERT ON want_keep.collector_evidence_items TO want_keep_app;
