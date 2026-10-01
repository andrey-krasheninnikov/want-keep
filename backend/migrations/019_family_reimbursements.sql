CREATE TABLE want_keep.reimbursements (
 household_id uuid NOT NULL REFERENCES want_keep.households(id),
 id uuid NOT NULL,
 revision want_keep.revision NOT NULL,
 PRIMARY KEY(household_id,id)
);

CREATE TABLE want_keep.reimbursement_revisions (
 household_id uuid NOT NULL,
 reimbursement_id uuid NOT NULL,
 revision want_keep.revision NOT NULL,
 previous_revision want_keep.revision,
 creditor_member_id uuid NOT NULL,
 debtor_member_id uuid NOT NULL,
 asset want_keep.asset NOT NULL,
 principal want_keep.amount NOT NULL CHECK(principal>0),
 outstanding want_keep.amount NOT NULL CHECK(outstanding>=0 AND outstanding<=principal),
 state text NOT NULL CHECK(state IN ('open','settled','attention_required','voided')),
 voided boolean NOT NULL,
 expense_id uuid,
 expense_revision want_keep.revision,
 reason text NOT NULL CHECK(length(reason) BETWEEN 1 AND 2000),
 attention_reason text NOT NULL CHECK(length(attention_reason)<=2000),
 actor_id uuid NOT NULL,
 decision_id uuid NOT NULL,
 recorded_at timestamptz NOT NULL,
 recorded_ns want_keep.submicro NOT NULL,
 PRIMARY KEY(household_id,reimbursement_id,revision),
 FOREIGN KEY(household_id,reimbursement_id) REFERENCES want_keep.reimbursements(household_id,id),
 FOREIGN KEY(household_id,reimbursement_id,previous_revision) REFERENCES want_keep.reimbursement_revisions(household_id,reimbursement_id,revision),
 FOREIGN KEY(household_id,creditor_member_id) REFERENCES want_keep.memberships(household_id,id),
 FOREIGN KEY(household_id,debtor_member_id) REFERENCES want_keep.memberships(household_id,id),
 FOREIGN KEY(household_id,expense_id,expense_revision) REFERENCES want_keep.operation_revisions(household_id,operation_id,revision),
 FOREIGN KEY(household_id,actor_id) REFERENCES want_keep.memberships(household_id,user_id),
 CHECK(creditor_member_id<>debtor_member_id),
 CHECK((revision=1 AND previous_revision IS NULL) OR previous_revision=revision-1),
 CHECK((expense_id IS NULL)=(expense_revision IS NULL)),
 CHECK((voided AND state='voided') OR NOT voided AND state<>'voided'),
 CHECK((state='attention_required')=(length(attention_reason)>0 AND NOT voided)),
 CHECK(state<>'settled' OR outstanding=0)
);
CREATE INDEX reimbursement_page ON want_keep.reimbursement_revisions(household_id,recorded_at DESC,recorded_ns DESC,reimbursement_id DESC);
CREATE INDEX reimbursement_expense ON want_keep.reimbursement_revisions(household_id,expense_id) WHERE expense_id IS NOT NULL;

CREATE TABLE want_keep.reimbursement_field_versions (
 household_id uuid NOT NULL,
 reimbursement_id uuid NOT NULL,
 revision want_keep.revision NOT NULL,
 field text NOT NULL CHECK(field IN ('parties','principal','expense','reason','voided','settlement')),
 changed_revision want_keep.revision NOT NULL CHECK(changed_revision<=revision),
 PRIMARY KEY(household_id,reimbursement_id,revision,field),
 FOREIGN KEY(household_id,reimbursement_id,revision) REFERENCES want_keep.reimbursement_revisions(household_id,reimbursement_id,revision)
);

CREATE TABLE want_keep.reimbursement_decisions (
 household_id uuid NOT NULL,
 id uuid NOT NULL,
 reimbursement_id uuid NOT NULL,
 kind text NOT NULL CHECK(kind IN ('create','correction','settlement','undo','reference_change')),
 reason text NOT NULL CHECK(length(reason) BETWEEN 1 AND 2000),
 undo_of uuid,
 settlement_id uuid,
 actor_id uuid NOT NULL,
 recorded_at timestamptz NOT NULL,
 recorded_ns want_keep.submicro NOT NULL,
 before_revision bigint NOT NULL CHECK(before_revision BETWEEN 0 AND 9007199254740991),
 after_revision want_keep.revision NOT NULL,
 PRIMARY KEY(household_id,id),
 UNIQUE(household_id,reimbursement_id,after_revision),
 FOREIGN KEY(household_id,reimbursement_id,after_revision) REFERENCES want_keep.reimbursement_revisions(household_id,reimbursement_id,revision),
 FOREIGN KEY(household_id,undo_of) REFERENCES want_keep.reimbursement_decisions(household_id,id),
 FOREIGN KEY(household_id,actor_id) REFERENCES want_keep.memberships(household_id,user_id),
 CHECK(before_revision+1=after_revision),
 CHECK((kind='undo')=(undo_of IS NOT NULL)),
 CHECK((kind='settlement')=(settlement_id IS NOT NULL))
);

CREATE TABLE want_keep.reimbursement_decision_fields (
 household_id uuid NOT NULL,
 decision_id uuid NOT NULL,
 field text NOT NULL CHECK(field IN ('parties','principal','expense','reason','voided','settlement')),
 PRIMARY KEY(household_id,decision_id,field),
 FOREIGN KEY(household_id,decision_id) REFERENCES want_keep.reimbursement_decisions(household_id,id)
);

CREATE TABLE want_keep.reimbursement_settlements (
 household_id uuid NOT NULL,
 id uuid NOT NULL,
 reimbursement_id uuid NOT NULL,
 created_revision want_keep.revision NOT NULL,
 decision_id uuid NOT NULL,
 transfer_id uuid NOT NULL,
 transfer_revision want_keep.revision NOT NULL,
 transfer_key text NOT NULL CHECK(length(transfer_key) BETWEEN 1 AND 2000),
 fingerprint text NOT NULL CHECK(fingerprint ~ '^[0-9a-f]{64}$'),
 transfer_asset want_keep.asset NOT NULL,
 transfer_amount want_keep.amount NOT NULL CHECK(transfer_amount>0),
 settled_asset want_keep.asset NOT NULL,
 settled_amount want_keep.amount NOT NULL CHECK(settled_amount>0),
 actor_id uuid NOT NULL,
 recorded_at timestamptz NOT NULL,
 recorded_ns want_keep.submicro NOT NULL,
 PRIMARY KEY(household_id,id),
 UNIQUE(household_id,decision_id),
 FOREIGN KEY(household_id,reimbursement_id,created_revision) REFERENCES want_keep.reimbursement_revisions(household_id,reimbursement_id,revision),
 FOREIGN KEY(household_id,decision_id) REFERENCES want_keep.reimbursement_decisions(household_id,id),
 FOREIGN KEY(household_id,transfer_id,transfer_revision) REFERENCES want_keep.operation_revisions(household_id,operation_id,revision),
 FOREIGN KEY(household_id,actor_id) REFERENCES want_keep.memberships(household_id,user_id),
 CHECK(transfer_asset<>settled_asset OR transfer_amount=settled_amount)
);
CREATE INDEX reimbursement_transfer_usage ON want_keep.reimbursement_settlements(household_id,transfer_key,transfer_asset);

CREATE TABLE want_keep.reimbursement_settlement_operations (
 household_id uuid NOT NULL,
 settlement_id uuid NOT NULL,
 operation_id uuid NOT NULL,
 operation_revision want_keep.revision NOT NULL,
 PRIMARY KEY(household_id,settlement_id,operation_id),
 FOREIGN KEY(household_id,settlement_id) REFERENCES want_keep.reimbursement_settlements(household_id,id),
 FOREIGN KEY(household_id,operation_id,operation_revision) REFERENCES want_keep.operation_revisions(household_id,operation_id,revision)
);
CREATE INDEX reimbursement_settlement_operation_lookup ON want_keep.reimbursement_settlement_operations(household_id,operation_id);

CREATE TABLE want_keep.reimbursement_settlement_events (
 household_id uuid NOT NULL,
 settlement_id uuid NOT NULL,
 reimbursement_id uuid NOT NULL,
 reimbursement_revision want_keep.revision NOT NULL,
 state text NOT NULL CHECK(state IN ('active','stale','undone')),
 decision_id uuid NOT NULL,
 PRIMARY KEY(household_id,settlement_id,reimbursement_revision),
 FOREIGN KEY(household_id,settlement_id) REFERENCES want_keep.reimbursement_settlements(household_id,id),
 FOREIGN KEY(household_id,reimbursement_id,reimbursement_revision) REFERENCES want_keep.reimbursement_revisions(household_id,reimbursement_id,revision),
 FOREIGN KEY(household_id,decision_id) REFERENCES want_keep.reimbursement_decisions(household_id,id)
);

CREATE TABLE want_keep.reimbursement_audit (
 household_id uuid NOT NULL,
 reimbursement_id uuid NOT NULL,
 revision want_keep.revision NOT NULL,
 actor_id uuid NOT NULL,
 decision_id uuid NOT NULL,
 command_id uuid,
 event text NOT NULL CHECK(event IN ('created','corrected','settled','undone','reference_changed')),
 recorded_at timestamptz NOT NULL,
 recorded_ns want_keep.submicro NOT NULL,
 PRIMARY KEY(household_id,reimbursement_id,revision),
 FOREIGN KEY(household_id,reimbursement_id,revision) REFERENCES want_keep.reimbursement_revisions(household_id,reimbursement_id,revision),
 FOREIGN KEY(household_id,decision_id) REFERENCES want_keep.reimbursement_decisions(household_id,id),
 FOREIGN KEY(household_id,actor_id) REFERENCES want_keep.memberships(household_id,user_id)
);
-- One command may reconcile several debts; retain every audit row for durable replay detection.
CREATE INDEX reimbursement_audit_command ON want_keep.reimbursement_audit(household_id,actor_id,command_id) WHERE command_id IS NOT NULL;

DO $$ DECLARE relation text; BEGIN
 FOREACH relation IN ARRAY ARRAY['reimbursement_revisions','reimbursement_field_versions','reimbursement_decisions','reimbursement_decision_fields','reimbursement_settlements','reimbursement_settlement_operations','reimbursement_settlement_events','reimbursement_audit'] LOOP
  EXECUTE format('CREATE TRIGGER immutable_history BEFORE UPDATE OR DELETE ON want_keep.%I FOR EACH ROW EXECUTE FUNCTION want_keep.reject_history_change()',relation);
  EXECUTE format('GRANT SELECT,INSERT ON want_keep.%I TO want_keep_app',relation);
 END LOOP;
END $$;
GRANT SELECT,INSERT ON want_keep.reimbursements TO want_keep_app;
GRANT UPDATE(revision) ON want_keep.reimbursements TO want_keep_app;
