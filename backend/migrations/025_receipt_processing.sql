CREATE TABLE want_keep.receipts (
 household_id uuid NOT NULL, id uuid NOT NULL, account_id uuid NOT NULL, attachment_id uuid NOT NULL,
 actor_id uuid NOT NULL, content_hash text NOT NULL CHECK(content_hash ~ '^[0-9a-f]{64}$'),
 revision want_keep.revision NOT NULL DEFAULT 1,
 state text NOT NULL CHECK(state IN ('uploaded','validating','processing','clarification','skipped','linked','recorded','failed','waiting_ai')),
 reason text NOT NULL DEFAULT '' CHECK(length(reason)<=2000), transaction_id uuid, transaction_revision want_keep.revision,
 candidates jsonb NOT NULL DEFAULT '[]' CHECK(jsonb_typeof(candidates)='array'),
 created_at timestamptz NOT NULL, created_ns want_keep.submicro NOT NULL,
 PRIMARY KEY(household_id,id), UNIQUE(household_id,account_id,content_hash),
 FOREIGN KEY(household_id,account_id) REFERENCES want_keep.accounts(household_id,id),
 FOREIGN KEY(household_id,attachment_id) REFERENCES want_keep.attachments(household_id,id),
 FOREIGN KEY(household_id,actor_id) REFERENCES want_keep.memberships(household_id,user_id),
 FOREIGN KEY(household_id,transaction_id,transaction_revision) REFERENCES want_keep.operation_revisions(household_id,operation_id,revision),
 CHECK((transaction_id IS NULL)=(transaction_revision IS NULL))
);
CREATE TABLE want_keep.receipt_evidence (
 household_id uuid NOT NULL, receipt_id uuid NOT NULL, attachment_id uuid NOT NULL, actor_id uuid NOT NULL,
 PRIMARY KEY(household_id,receipt_id,attachment_id), UNIQUE(household_id,attachment_id),
 FOREIGN KEY(household_id,receipt_id) REFERENCES want_keep.receipts(household_id,id),
 FOREIGN KEY(household_id,attachment_id) REFERENCES want_keep.attachments(household_id,id),
 FOREIGN KEY(household_id,actor_id) REFERENCES want_keep.memberships(household_id,user_id)
);
CREATE TABLE want_keep.receipt_pages (
 household_id uuid NOT NULL, id uuid NOT NULL, receipt_id uuid NOT NULL, revision want_keep.revision NOT NULL DEFAULT 1 CHECK(revision=1),
 page integer NOT NULL CHECK(page BETWEEN 1 AND 10), content_hash text NOT NULL CHECK(content_hash ~ '^[0-9a-f]{64}$'),
 PRIMARY KEY(household_id,id,revision), UNIQUE(household_id,receipt_id,page),
 FOREIGN KEY(household_id,receipt_id) REFERENCES want_keep.receipts(household_id,id)
);
CREATE TABLE want_keep.receipt_revisions (
 household_id uuid NOT NULL, receipt_id uuid NOT NULL, revision want_keep.revision NOT NULL,
 actor_id uuid NOT NULL, state text NOT NULL, reason text NOT NULL CHECK(length(reason)<=2000),
 snapshot jsonb NOT NULL CHECK(jsonb_typeof(snapshot)='object'), command_id uuid,
 recorded_at timestamptz NOT NULL, recorded_ns want_keep.submicro NOT NULL,
 PRIMARY KEY(household_id,receipt_id,revision),
 FOREIGN KEY(household_id,receipt_id) REFERENCES want_keep.receipts(household_id,id),
 FOREIGN KEY(household_id,actor_id) REFERENCES want_keep.memberships(household_id,user_id)
);
CREATE TABLE want_keep.ai_request_subjects (
 household_id uuid NOT NULL, resource_id uuid NOT NULL, resource_revision want_keep.revision NOT NULL,
 operation_id uuid, operation_revision want_keep.revision, page_id uuid, page_revision want_keep.revision,
 PRIMARY KEY(household_id,resource_id,resource_revision),
 FOREIGN KEY(household_id,operation_id,operation_revision) REFERENCES want_keep.ledger_review_requests(household_id,operation_id,revision),
 FOREIGN KEY(household_id,page_id,page_revision) REFERENCES want_keep.receipt_pages(household_id,id,revision),
 CHECK((operation_id=resource_id AND operation_revision=resource_revision AND page_id IS NULL AND page_revision IS NULL)
    OR (page_id=resource_id AND page_revision=resource_revision AND operation_id IS NULL AND operation_revision IS NULL)),
 CHECK(num_nonnulls(operation_id,page_id)=1), CHECK(num_nonnulls(operation_revision,page_revision)=1)
);
INSERT INTO want_keep.ai_request_subjects(household_id,resource_id,resource_revision,operation_id,operation_revision)
 SELECT household_id,operation_id,revision,operation_id,revision FROM want_keep.ledger_review_requests;
DO $$ DECLARE c text; BEGIN
 FOR c IN SELECT conname FROM pg_constraint WHERE conrelid='want_keep.ai_attempts'::regclass AND confrelid='want_keep.ledger_review_requests'::regclass LOOP
  EXECUTE format('ALTER TABLE want_keep.ai_attempts DROP CONSTRAINT %I',c);
 END LOOP;
END $$;
ALTER TABLE want_keep.ai_attempts ADD FOREIGN KEY(household_id,resource_id,resource_revision) REFERENCES want_keep.ai_request_subjects(household_id,resource_id,resource_revision);
DO $$ DECLARE c text; BEGIN
 FOR c IN SELECT conname FROM pg_constraint WHERE conrelid='want_keep.jobs'::regclass AND confrelid='want_keep.ledger_review_requests'::regclass LOOP
  EXECUTE format('ALTER TABLE want_keep.jobs DROP CONSTRAINT %I',c);
 END LOOP;
END $$;
ALTER TABLE want_keep.jobs ADD FOREIGN KEY(household_id,resource_id,resource_revision) REFERENCES want_keep.ai_request_subjects(household_id,resource_id,resource_revision);
ALTER TABLE want_keep.jobs DROP CONSTRAINT jobs_kind_check;
DO $$ DECLARE c text; BEGIN
 FOR c IN SELECT conname FROM pg_constraint WHERE conrelid='want_keep.jobs'::regclass AND contype='c' AND (pg_get_constraintdef(oid) LIKE '%kind%connection_id%' OR pg_get_constraintdef(oid) LIKE '%kind%resource_id%') LOOP
  EXECUTE format('ALTER TABLE want_keep.jobs DROP CONSTRAINT %I',c);
 END LOOP;
END $$;
ALTER TABLE want_keep.jobs ADD CONSTRAINT jobs_kind_check CHECK(kind IN ('sync','outbox','ai','ai_answer','ai_validation','receipt_extract','receipt_validation'));
ALTER TABLE want_keep.jobs ADD CHECK((kind='sync' AND connection_id IS NOT NULL AND connection_generation IS NOT NULL AND binding IS NOT NULL AND admission_revision IS NOT NULL) OR (kind IN ('outbox','ai','ai_answer','ai_validation','receipt_extract','receipt_validation') AND connection_id IS NULL AND connection_generation IS NULL AND binding IS NULL AND admission_revision IS NULL));
ALTER TABLE want_keep.jobs ADD CHECK((kind IN ('ai','ai_answer','ai_validation','receipt_extract','receipt_validation') AND resource_id IS NOT NULL AND resource_revision IS NOT NULL) OR (kind NOT IN ('ai','ai_answer','ai_validation','receipt_extract','receipt_validation') AND resource_id IS NULL AND resource_revision IS NULL));
CREATE UNIQUE INDEX receipt_extract_once ON want_keep.jobs(household_id,resource_id,resource_revision) WHERE kind='receipt_extract';
CREATE TABLE want_keep.receipt_validation_jobs (
 household_id uuid NOT NULL, job_id uuid NOT NULL, attempt_id uuid NOT NULL,
 PRIMARY KEY(household_id,job_id), UNIQUE(household_id,attempt_id),
 FOREIGN KEY(household_id,job_id) REFERENCES want_keep.jobs(household_id,id),
 FOREIGN KEY(household_id,attempt_id) REFERENCES want_keep.ai_attempts(household_id,id)
);
CREATE TABLE want_keep.receipt_page_outcomes (
 household_id uuid NOT NULL, page_id uuid NOT NULL, page_revision want_keep.revision NOT NULL,
 attempt_id uuid NOT NULL, extraction jsonb, code text NOT NULL DEFAULT '',
 PRIMARY KEY(household_id,page_id,page_revision), UNIQUE(household_id,attempt_id),
 FOREIGN KEY(household_id,page_id,page_revision) REFERENCES want_keep.receipt_pages(household_id,id,revision),
 FOREIGN KEY(household_id,attempt_id) REFERENCES want_keep.ai_attempts(household_id,id),
 CHECK(extraction IS NULL OR jsonb_typeof(extraction)='object')
);
CREATE TABLE want_keep.receipt_answers (
 household_id uuid NOT NULL, receipt_id uuid NOT NULL, revision want_keep.revision NOT NULL,
 actor_id uuid NOT NULL, answer jsonb NOT NULL CHECK(jsonb_typeof(answer)='object'),
 PRIMARY KEY(household_id,receipt_id,revision),
 FOREIGN KEY(household_id,receipt_id,revision) REFERENCES want_keep.receipt_revisions(household_id,receipt_id,revision),
 FOREIGN KEY(household_id,actor_id) REFERENCES want_keep.memberships(household_id,user_id)
);
ALTER TABLE want_keep.transaction_details DROP CONSTRAINT transaction_details_origin_check;
ALTER TABLE want_keep.transaction_details ADD CHECK(origin IN ('','manual','source','receipt'));
ALTER TABLE want_keep.transaction_details ADD COLUMN occurred_precision text NOT NULL DEFAULT '' CHECK(occurred_precision IN ('','day','second'));
DO $$ DECLARE relation text; BEGIN
 FOREACH relation IN ARRAY ARRAY['receipt_evidence','receipt_pages','receipt_revisions','ai_request_subjects','receipt_validation_jobs','receipt_page_outcomes','receipt_answers'] LOOP
  EXECUTE format('CREATE TRIGGER immutable_history BEFORE UPDATE OR DELETE ON want_keep.%I FOR EACH ROW EXECUTE FUNCTION want_keep.reject_history_change()',relation);
  EXECUTE format('GRANT SELECT,INSERT ON want_keep.%I TO want_keep_app',relation);
 END LOOP;
END $$;
GRANT SELECT,INSERT ON want_keep.receipts TO want_keep_app;
GRANT UPDATE(revision,state,reason,transaction_id,transaction_revision,candidates) ON want_keep.receipts TO want_keep_app;
DO $$ DECLARE definition text; BEGIN
 SELECT pg_get_functiondef('want_keep.reconcile_ai_attempt(uuid,text,numeric,text)'::regprocedure) INTO definition;
 definition:=replace(definition,'''ai'',''ai_answer''','''ai'',''ai_answer'',''receipt_extract''');
 EXECUTE definition;
END $$;

CREATE TABLE want_keep.receipt_page_contexts (
 household_id uuid NOT NULL, page_id uuid NOT NULL, page_revision want_keep.revision NOT NULL,
 input jsonb NOT NULL CHECK(jsonb_typeof(input)='array'), references_map jsonb NOT NULL CHECK(jsonb_typeof(references_map)='object'),
 PRIMARY KEY(household_id,page_id,page_revision),
 FOREIGN KEY(household_id,page_id,page_revision) REFERENCES want_keep.receipt_pages(household_id,id,revision)
);
CREATE TRIGGER immutable_history BEFORE UPDATE OR DELETE ON want_keep.receipt_page_contexts FOR EACH ROW EXECUTE FUNCTION want_keep.reject_history_change();
GRANT SELECT,INSERT ON want_keep.receipt_page_contexts TO want_keep_app;
