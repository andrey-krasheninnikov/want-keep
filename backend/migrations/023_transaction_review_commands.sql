ALTER TABLE want_keep.jobs DROP CONSTRAINT jobs_kind_check;
ALTER TABLE want_keep.jobs ADD CHECK(kind IN ('sync','outbox','ai','ai_answer','ai_validation'));
DO $$ DECLARE c text; BEGIN
 FOR c IN SELECT conname FROM pg_constraint WHERE conrelid='want_keep.jobs'::regclass AND contype='c' AND (pg_get_constraintdef(oid) LIKE '%kind%connection_id%' OR pg_get_constraintdef(oid) LIKE '%kind%resource_id%') LOOP
  EXECUTE format('ALTER TABLE want_keep.jobs DROP CONSTRAINT %I',c);
 END LOOP;
END $$;
ALTER TABLE want_keep.jobs ADD CHECK((kind='sync' AND connection_id IS NOT NULL AND connection_generation IS NOT NULL AND binding IS NOT NULL AND admission_revision IS NOT NULL) OR (kind IN ('outbox','ai','ai_answer','ai_validation') AND connection_id IS NULL AND connection_generation IS NULL AND binding IS NULL AND admission_revision IS NULL));
ALTER TABLE want_keep.jobs ADD CHECK((kind IN ('ai','ai_answer','ai_validation') AND resource_id IS NOT NULL AND resource_revision IS NOT NULL) OR (kind NOT IN ('ai','ai_answer','ai_validation') AND resource_id IS NULL AND resource_revision IS NULL));

CREATE TABLE want_keep.review_contexts (
 household_id uuid NOT NULL, job_id uuid NOT NULL, operation_id uuid NOT NULL, operation_revision want_keep.revision NOT NULL,
 context jsonb NOT NULL CHECK(jsonb_typeof(context)='object'),
 PRIMARY KEY(household_id,job_id), FOREIGN KEY(household_id,job_id) REFERENCES want_keep.jobs(household_id,id),
 FOREIGN KEY(household_id,operation_id,operation_revision) REFERENCES want_keep.operation_revisions(household_id,operation_id,revision)
);
CREATE TABLE want_keep.review_validation_jobs (
 household_id uuid NOT NULL, job_id uuid NOT NULL, attempt_id uuid NOT NULL,
 PRIMARY KEY(household_id,job_id), UNIQUE(household_id,attempt_id),
 FOREIGN KEY(household_id,job_id) REFERENCES want_keep.jobs(household_id,id),
 FOREIGN KEY(household_id,attempt_id) REFERENCES want_keep.ai_attempts(household_id,id)
);
CREATE TABLE want_keep.review_validations (
 household_id uuid NOT NULL, attempt_id uuid NOT NULL, operation_id uuid NOT NULL, operation_revision want_keep.revision NOT NULL,
 state text NOT NULL CHECK(state IN ('applied','clarification','proposed','rejected','stale','failed')),
 code text NOT NULL DEFAULT '', rationale text NOT NULL CHECK(length(rationale) BETWEEN 1 AND 2000),
 proposal_id uuid, clarification_id uuid, recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(household_id,attempt_id), FOREIGN KEY(household_id,attempt_id) REFERENCES want_keep.ai_attempts(household_id,id),
 FOREIGN KEY(household_id,operation_id,operation_revision) REFERENCES want_keep.operation_revisions(household_id,operation_id,revision)
);
CREATE TABLE want_keep.review_proposals (
 household_id uuid NOT NULL, id uuid NOT NULL, revision want_keep.revision NOT NULL, operation_id uuid NOT NULL, operation_revision want_keep.revision NOT NULL,
 state text NOT NULL CHECK(state IN ('pending','applied','rejected','superseded')), payload jsonb NOT NULL, context jsonb NOT NULL,
 PRIMARY KEY(household_id,id), FOREIGN KEY(household_id,operation_id,operation_revision) REFERENCES want_keep.operation_revisions(household_id,operation_id,revision)
);
CREATE TABLE want_keep.review_clarifications (
 household_id uuid NOT NULL, id uuid NOT NULL, revision want_keep.revision NOT NULL, operation_id uuid NOT NULL, operation_revision want_keep.revision NOT NULL,
 state text NOT NULL CHECK(state IN ('open','checking','answered','superseded')), payload jsonb NOT NULL, context jsonb NOT NULL,
 PRIMARY KEY(household_id,id), FOREIGN KEY(household_id,operation_id,operation_revision) REFERENCES want_keep.operation_revisions(household_id,operation_id,revision)
);
CREATE TABLE want_keep.review_answers (
 household_id uuid NOT NULL, clarification_id uuid NOT NULL, revision want_keep.revision NOT NULL, actor_id uuid NOT NULL,
 choice_id text NOT NULL DEFAULT '', answer_text text NOT NULL DEFAULT '' CHECK(length(answer_text)<=2000), job_id uuid,
 PRIMARY KEY(household_id,clarification_id,revision), UNIQUE(household_id,job_id),
 FOREIGN KEY(household_id,clarification_id) REFERENCES want_keep.review_clarifications(household_id,id),
 FOREIGN KEY(household_id,actor_id) REFERENCES want_keep.memberships(household_id,user_id),
 FOREIGN KEY(household_id,job_id) REFERENCES want_keep.jobs(household_id,id),
 CHECK((choice_id='' AND length(answer_text)>0 AND job_id IS NOT NULL) OR (choice_id<>'' AND answer_text='' AND job_id IS NULL))
);
CREATE TABLE want_keep.review_state_history (
 household_id uuid NOT NULL, kind text NOT NULL CHECK(kind IN ('proposal','clarification')), id uuid NOT NULL,
 revision want_keep.revision NOT NULL, actor_id uuid NOT NULL, payload jsonb NOT NULL,
 proposal_id uuid, clarification_id uuid, recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(household_id,kind,id,revision),
 FOREIGN KEY(household_id,actor_id) REFERENCES want_keep.memberships(household_id,user_id),
 FOREIGN KEY(household_id,proposal_id) REFERENCES want_keep.review_proposals(household_id,id),
 FOREIGN KEY(household_id,clarification_id) REFERENCES want_keep.review_clarifications(household_id,id),
 CHECK((kind='proposal' AND proposal_id=id AND clarification_id IS NULL) OR (kind='clarification' AND clarification_id=id AND proposal_id IS NULL))
);
CREATE TRIGGER immutable_history BEFORE UPDATE OR DELETE ON want_keep.review_state_history FOR EACH ROW EXECUTE FUNCTION want_keep.reject_history_change();
GRANT SELECT,INSERT ON want_keep.review_state_history TO want_keep_app;
ALTER TABLE want_keep.review_validations ADD FOREIGN KEY(household_id,proposal_id) REFERENCES want_keep.review_proposals(household_id,id);
ALTER TABLE want_keep.review_validations ADD FOREIGN KEY(household_id,clarification_id) REFERENCES want_keep.review_clarifications(household_id,id);
CREATE INDEX open_review_clarifications ON want_keep.review_clarifications(household_id,id) WHERE state IN ('open','checking');
CREATE TRIGGER immutable_history BEFORE UPDATE OR DELETE ON want_keep.review_contexts FOR EACH ROW EXECUTE FUNCTION want_keep.reject_history_change();
CREATE TRIGGER immutable_history BEFORE UPDATE OR DELETE ON want_keep.review_validation_jobs FOR EACH ROW EXECUTE FUNCTION want_keep.reject_history_change();
CREATE TRIGGER immutable_history BEFORE UPDATE OR DELETE ON want_keep.review_validations FOR EACH ROW EXECUTE FUNCTION want_keep.reject_history_change();
CREATE TRIGGER immutable_history BEFORE UPDATE OR DELETE ON want_keep.review_answers FOR EACH ROW EXECUTE FUNCTION want_keep.reject_history_change();
GRANT SELECT,INSERT ON want_keep.review_contexts,want_keep.review_validation_jobs,want_keep.review_validations,want_keep.review_proposals,want_keep.review_clarifications,want_keep.review_answers TO want_keep_app;
GRANT UPDATE(revision,state,payload) ON want_keep.review_proposals,want_keep.review_clarifications TO want_keep_app;
-- Resume validation of already saved outcomes without another provider call.
WITH candidates AS (
 SELECT DISTINCT ON(a.household_id,a.job_id) a.household_id,a.id,a.actor_id,a.resource_id,a.resource_revision
 FROM want_keep.ai_attempts a JOIN want_keep.ai_attempt_states s ON (s.household_id,s.attempt_id)=(a.household_id,a.id)
 WHERE s.state='completed' ORDER BY a.household_id,a.job_id,a.created_at DESC,a.created_ns DESC,a.id
), inserted AS (
 INSERT INTO want_keep.jobs(household_id,id,actor_id,kind,state,max_attempts,available_at,deadline,resource_id,resource_revision)
 SELECT household_id,id,actor_id,'ai_validation','ready',5,clock_timestamp(),clock_timestamp()+INTERVAL '24 hours',resource_id,resource_revision FROM candidates
 RETURNING household_id,id
)
INSERT INTO want_keep.review_validation_jobs SELECT household_id,id,id FROM inserted;

-- Answer checks use the same operator-only reconciliation fence as initial checks.
DO $$ DECLARE definition text; BEGIN
 SELECT pg_get_functiondef('want_keep.reconcile_ai_attempt(uuid,text,numeric,text)'::regprocedure) INTO definition;
 definition:=replace(definition,'kind=''ai''','kind IN (''ai'',''ai_answer'')');
 EXECUTE definition;
END $$;
