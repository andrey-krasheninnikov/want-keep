CREATE TABLE want_keep.rate_observations (
 id uuid PRIMARY KEY,
 source text NOT NULL CHECK(source IN ('cbr','coingecko')),
 transport text NOT NULL CHECK(transport IN ('xml_daily','frankfurter_v2','demo_simple_price','demo_history')),
 provider_asset_id text NOT NULL CHECK(length(provider_asset_id) BETWEEN 1 AND 100),
 rate_base want_keep.asset NOT NULL,
 rate_quote want_keep.asset NOT NULL,
 rate_value want_keep.amount NOT NULL CHECK(rate_value>0),
 requested_date date NOT NULL,
 effective_at timestamptz NOT NULL,
 effective_ns want_keep.submicro NOT NULL,
 fetched_at timestamptz NOT NULL,
 fetched_ns want_keep.submicro NOT NULL,
 granularity text NOT NULL CHECK(granularity IN ('daily','instant')),
 revision want_keep.revision NOT NULL,
 previous_id uuid REFERENCES want_keep.rate_observations(id),
 CHECK(rate_base<>rate_quote),
 CHECK((source='cbr' AND transport IN ('xml_daily','frankfurter_v2') AND provider_asset_id='R01235' AND rate_base='USD' AND rate_quote='RUB' AND granularity='daily')
    OR (source='coingecko' AND transport IN ('demo_simple_price','demo_history')
      AND ((transport='demo_simple_price' AND granularity='instant') OR (transport='demo_history' AND granularity='daily'))
      AND rate_quote='USD'
      AND ((rate_base='BTC' AND provider_asset_id='bitcoin') OR (rate_base='ETH' AND provider_asset_id='ethereum')
        OR (rate_base='USDT' AND provider_asset_id='tether') OR (rate_base='USDC' AND provider_asset_id='usd-coin')))),
 CHECK((effective_at AT TIME ZONE 'UTC')::date<=requested_date OR granularity='instant'),
 UNIQUE(source,transport,provider_asset_id,requested_date,granularity,revision)
);
CREATE INDEX rate_lookup ON want_keep.rate_observations(rate_base,rate_quote,granularity,requested_date DESC,revision DESC);
CREATE TRIGGER immutable_history BEFORE UPDATE OR DELETE ON want_keep.rate_observations FOR EACH ROW EXECUTE FUNCTION want_keep.reject_history_change();
GRANT SELECT,INSERT ON want_keep.rate_observations TO want_keep_app;

CREATE TABLE want_keep.rate_source_usage (
 source text NOT NULL CHECK(source='coingecko'),
 period text NOT NULL CHECK(period IN ('minute','month')),
 window_start timestamptz NOT NULL,
 used integer NOT NULL CHECK(used BETWEEN 1 AND 9000),
 PRIMARY KEY(source,period,window_start)
);
GRANT SELECT,INSERT ON want_keep.rate_source_usage TO want_keep_app;
GRANT UPDATE(used) ON want_keep.rate_source_usage TO want_keep_app;

CREATE TABLE want_keep.valuation_snapshots (
 household_id uuid NOT NULL REFERENCES want_keep.households(id),
 operation_id uuid NOT NULL,
 operation_revision want_keep.revision NOT NULL,
 component_index smallint NOT NULL CHECK(component_index BETWEEN 0 AND 1000),
 component_kind text NOT NULL CHECK(length(component_kind) BETWEEN 1 AND 100),
 reporting_asset want_keep.asset NOT NULL,
 valuation_revision want_keep.revision NOT NULL,
 native_asset want_keep.asset NOT NULL,
 native_amount want_keep.amount NOT NULL,
 reporting_amount want_keep.amount,
 requested_date date NOT NULL,
 status text NOT NULL CHECK(status IN ('known','partial','unavailable')),
 reason text NOT NULL DEFAULT '',
 coverage_reasons text[] NOT NULL DEFAULT '{}',
 freshness text NOT NULL CHECK(freshness IN ('fresh','stale','unknown')),
 recorded_at timestamptz NOT NULL,
 recorded_ns want_keep.submicro NOT NULL,
 PRIMARY KEY(household_id,operation_id,operation_revision,component_index,reporting_asset,valuation_revision),
 FOREIGN KEY(household_id,operation_id,operation_revision) REFERENCES want_keep.operation_revisions(household_id,operation_id,revision),
 CHECK((status='known' AND reporting_amount IS NOT NULL AND reason='' AND cardinality(coverage_reasons)=0) OR
       (status='partial' AND reporting_amount IS NOT NULL AND reason='' AND cardinality(coverage_reasons)>0) OR
       (status='unavailable' AND reporting_amount IS NULL AND length(reason)>0 AND cardinality(coverage_reasons)=0))
);
CREATE TABLE want_keep.valuation_snapshot_legs (
 household_id uuid NOT NULL,
 operation_id uuid NOT NULL,
 operation_revision want_keep.revision NOT NULL,
 component_index smallint NOT NULL,
 reporting_asset want_keep.asset NOT NULL,
 valuation_revision want_keep.revision NOT NULL,
 position smallint NOT NULL CHECK(position BETWEEN 1 AND 2),
 observation_id uuid NOT NULL REFERENCES want_keep.rate_observations(id),
 PRIMARY KEY(household_id,operation_id,operation_revision,component_index,reporting_asset,valuation_revision,position),
 FOREIGN KEY(household_id,operation_id,operation_revision,component_index,reporting_asset,valuation_revision)
  REFERENCES want_keep.valuation_snapshots(household_id,operation_id,operation_revision,component_index,reporting_asset,valuation_revision)
);
CREATE TRIGGER immutable_history BEFORE UPDATE OR DELETE ON want_keep.valuation_snapshots FOR EACH ROW EXECUTE FUNCTION want_keep.reject_history_change();
CREATE TRIGGER immutable_history BEFORE UPDATE OR DELETE ON want_keep.valuation_snapshot_legs FOR EACH ROW EXECUTE FUNCTION want_keep.reject_history_change();
GRANT SELECT,INSERT ON want_keep.valuation_snapshots,want_keep.valuation_snapshot_legs TO want_keep_app;

CREATE TABLE want_keep.platform_quotes (
 household_id uuid NOT NULL REFERENCES want_keep.households(id),
 id uuid NOT NULL,
 provider text NOT NULL CHECK(provider IN ('alfa','raiffeisen','ozon','bybit','aifory','emcd')),
 evidence_ref text NOT NULL CHECK(length(evidence_ref) BETWEEN 1 AND 2000),
 direction text NOT NULL CHECK(direction IN ('sell_base','buy_base')),
 base_asset want_keep.asset NOT NULL,
 quote_asset want_keep.asset NOT NULL,
 applicable_amount want_keep.amount NOT NULL CHECK(applicable_amount>0),
 rate_value want_keep.amount NOT NULL CHECK(rate_value>0),
 fee_coverage text NOT NULL CHECK(fee_coverage IN ('included','excluded')),
 spread_coverage text NOT NULL CHECK(spread_coverage='included'),
 observed_at timestamptz NOT NULL,
 observed_ns want_keep.submicro NOT NULL,
 fetched_at timestamptz NOT NULL,
 fetched_ns want_keep.submicro NOT NULL,
 PRIMARY KEY(household_id,id),
 UNIQUE(household_id,provider,evidence_ref),
 CHECK(base_asset<>quote_asset),
 CHECK((observed_at,observed_ns)<=(fetched_at,fetched_ns))
);
CREATE TABLE want_keep.platform_quote_fees (
 household_id uuid NOT NULL,
 quote_id uuid NOT NULL,
 position smallint NOT NULL CHECK(position BETWEEN 1 AND 20),
 asset want_keep.asset NOT NULL,
 amount want_keep.amount NOT NULL CHECK(amount>=0),
 PRIMARY KEY(household_id,quote_id,position),
 FOREIGN KEY(household_id,quote_id) REFERENCES want_keep.platform_quotes(household_id,id)
);
CREATE INDEX platform_quote_lookup ON want_keep.platform_quotes(household_id,provider,base_asset,quote_asset,direction,applicable_amount,observed_at DESC);
CREATE TRIGGER immutable_history BEFORE UPDATE OR DELETE ON want_keep.platform_quotes FOR EACH ROW EXECUTE FUNCTION want_keep.reject_history_change();
CREATE TRIGGER immutable_history BEFORE UPDATE OR DELETE ON want_keep.platform_quote_fees FOR EACH ROW EXECUTE FUNCTION want_keep.reject_history_change();
GRANT SELECT,INSERT ON want_keep.platform_quotes,want_keep.platform_quote_fees TO want_keep_app;
