CREATE TABLE want_keep.transaction_historical_values (
 household_id uuid NOT NULL,
 operation_id uuid NOT NULL,
 operation_revision want_keep.revision NOT NULL,
 basis_ref text NOT NULL CHECK(length(basis_ref) BETWEEN 1 AND 2000),
 native_amount want_keep.amount NOT NULL CHECK(native_amount>0),
 native_asset want_keep.asset NOT NULL,
 reporting_amount want_keep.amount NOT NULL CHECK(reporting_amount>0),
 reporting_asset want_keep.asset NOT NULL,
 PRIMARY KEY(household_id,operation_id,operation_revision),
 FOREIGN KEY(household_id,operation_id,operation_revision) REFERENCES want_keep.operation_revisions(household_id,operation_id,revision)
);
CREATE TRIGGER immutable_history BEFORE UPDATE OR DELETE ON want_keep.transaction_historical_values FOR EACH ROW EXECUTE FUNCTION want_keep.reject_history_change();

CREATE TABLE want_keep.refunds (
 household_id uuid NOT NULL,
 refund_operation_id uuid NOT NULL,
 purchase_operation_id uuid NOT NULL,
 revision want_keep.revision NOT NULL,
 PRIMARY KEY(household_id,refund_operation_id),
 UNIQUE(household_id,refund_operation_id,purchase_operation_id),
 FOREIGN KEY(household_id,refund_operation_id) REFERENCES want_keep.operations(household_id,id),
 FOREIGN KEY(household_id,purchase_operation_id) REFERENCES want_keep.operations(household_id,id),
 CHECK(refund_operation_id<>purchase_operation_id)
);
CREATE INDEX refunds_purchase ON want_keep.refunds(household_id,purchase_operation_id,refund_operation_id);

CREATE TABLE want_keep.refund_revisions (
 household_id uuid NOT NULL,
 refund_operation_id uuid NOT NULL,
 revision want_keep.revision NOT NULL,
 purchase_operation_id uuid NOT NULL,
 purchase_revision want_keep.revision NOT NULL,
 refund_revision want_keep.revision NOT NULL,
 actor_id uuid NOT NULL,
 reason text NOT NULL CHECK(length(reason) BETWEEN 1 AND 2000),
 state text NOT NULL CHECK(state IN ('applied','clarification','inactive')),
 expense_month date NOT NULL CHECK(expense_month=DATE_TRUNC('month',expense_month)::date),
 cash_date date NOT NULL,
 amount want_keep.amount NOT NULL CHECK(amount>0),
 remaining want_keep.amount NOT NULL CHECK(remaining>=0),
 asset want_keep.asset NOT NULL,
 valuation_basis_native_amount want_keep.amount,
 valuation_basis_native_asset want_keep.asset,
 valuation_basis_reporting_amount want_keep.amount,
 valuation_basis_reporting_asset want_keep.asset,
 valuation_basis_ref text,
 valuation_amount want_keep.amount,
 valuation_asset want_keep.asset,
 recorded_at timestamptz NOT NULL,
 recorded_ns want_keep.submicro NOT NULL,
 PRIMARY KEY(household_id,refund_operation_id,revision),
 FOREIGN KEY(household_id,refund_operation_id,purchase_operation_id) REFERENCES want_keep.refunds(household_id,refund_operation_id,purchase_operation_id),
 FOREIGN KEY(household_id,purchase_operation_id,purchase_revision) REFERENCES want_keep.operation_revisions(household_id,operation_id,revision),
 FOREIGN KEY(household_id,refund_operation_id,refund_revision) REFERENCES want_keep.operation_revisions(household_id,operation_id,revision),
 FOREIGN KEY(household_id,actor_id) REFERENCES want_keep.memberships(household_id,user_id),
 CHECK((valuation_basis_ref IS NULL AND valuation_basis_native_amount IS NULL AND valuation_basis_native_asset IS NULL AND valuation_basis_reporting_amount IS NULL AND valuation_basis_reporting_asset IS NULL AND valuation_amount IS NULL AND valuation_asset IS NULL)
    OR (length(valuation_basis_ref) BETWEEN 1 AND 2000 AND valuation_basis_native_amount>0 AND valuation_basis_native_asset=asset AND valuation_basis_reporting_amount>0 AND valuation_basis_reporting_asset IS NOT NULL AND valuation_amount>=0 AND valuation_asset=valuation_basis_reporting_asset))
);
CREATE INDEX refund_revisions_recorded ON want_keep.refund_revisions(household_id,refund_operation_id,recorded_at DESC,recorded_ns DESC,revision DESC);
CREATE TRIGGER immutable_history BEFORE UPDATE OR DELETE ON want_keep.refund_revisions FOR EACH ROW EXECUTE FUNCTION want_keep.reject_history_change();

CREATE TABLE want_keep.refund_item_portions (
 household_id uuid NOT NULL,
 refund_operation_id uuid NOT NULL,
 revision want_keep.revision NOT NULL,
 purchase_operation_id uuid NOT NULL,
 purchase_revision want_keep.revision NOT NULL,
 position integer NOT NULL CHECK(position BETWEEN 0 AND 999),
 item_id uuid NOT NULL,
 amount want_keep.amount NOT NULL CHECK(amount>0),
 asset want_keep.asset NOT NULL,
 PRIMARY KEY(household_id,refund_operation_id,revision,position),
 UNIQUE(household_id,refund_operation_id,revision,item_id),
 FOREIGN KEY(household_id,refund_operation_id,revision) REFERENCES want_keep.refund_revisions(household_id,refund_operation_id,revision),
 FOREIGN KEY(household_id,purchase_operation_id,purchase_revision,item_id) REFERENCES want_keep.receipt_items(household_id,operation_id,revision,id)
);
CREATE TRIGGER immutable_history BEFORE UPDATE OR DELETE ON want_keep.refund_item_portions FOR EACH ROW EXECUTE FUNCTION want_keep.reject_history_change();

CREATE TABLE want_keep.refund_effects (
 household_id uuid NOT NULL,
 refund_operation_id uuid NOT NULL,
 revision want_keep.revision NOT NULL,
 basis text NOT NULL CHECK(basis IN ('native','valuation')),
 dimension text NOT NULL CHECK(dimension IN ('member','category','unallocated')),
 position integer NOT NULL CHECK(position BETWEEN 0 AND 999),
 member_id uuid,
 category_id uuid,
 amount want_keep.amount NOT NULL CHECK(amount>0),
 asset want_keep.asset NOT NULL,
 PRIMARY KEY(household_id,refund_operation_id,revision,basis,dimension,position),
 FOREIGN KEY(household_id,refund_operation_id,revision) REFERENCES want_keep.refund_revisions(household_id,refund_operation_id,revision),
 FOREIGN KEY(household_id,member_id) REFERENCES want_keep.memberships(household_id,id),
 FOREIGN KEY(household_id,category_id) REFERENCES want_keep.categories(household_id,id),
 CHECK((dimension='member' AND member_id IS NOT NULL AND category_id IS NULL)
    OR (dimension='category' AND member_id IS NULL)
    OR (dimension='unallocated' AND member_id IS NULL AND category_id IS NULL))
);
CREATE TRIGGER immutable_history BEFORE UPDATE OR DELETE ON want_keep.refund_effects FOR EACH ROW EXECUTE FUNCTION want_keep.reject_history_change();

CREATE TABLE want_keep.refund_review_requests (
 household_id uuid NOT NULL,
 refund_operation_id uuid NOT NULL,
 revision want_keep.revision NOT NULL,
 requested_at timestamptz NOT NULL,
 requested_ns want_keep.submicro NOT NULL,
 PRIMARY KEY(household_id,refund_operation_id,revision),
 FOREIGN KEY(household_id,refund_operation_id,revision) REFERENCES want_keep.refund_revisions(household_id,refund_operation_id,revision)
);
CREATE TRIGGER immutable_history BEFORE UPDATE OR DELETE ON want_keep.refund_review_requests FOR EACH ROW EXECUTE FUNCTION want_keep.reject_history_change();

GRANT SELECT ON want_keep.transaction_historical_values TO want_keep_app;
GRANT SELECT,INSERT ON want_keep.refund_revisions,want_keep.refund_item_portions,want_keep.refund_effects,want_keep.refund_review_requests TO want_keep_app;
GRANT SELECT,INSERT ON want_keep.refunds TO want_keep_app;
GRANT UPDATE(revision) ON want_keep.refunds TO want_keep_app;
