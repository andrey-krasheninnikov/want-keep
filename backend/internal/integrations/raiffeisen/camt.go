package raiffeisen

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"io"
	"strings"
	"time"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	ingestion "github.com/pchkauu/want-keep/backend/internal/integrations/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
)

const camtLog = "rbo-camt-v1"

type Alias struct{ Kind, Value string }
type Fact struct {
	Record  ingestion.Record
	Aliases []Alias
}
type Statement struct {
	Facts    []Fact
	Closing  *ingestion.BalanceSnapshot
	Gaps     []string
	Evidence ingestion.Evidence
	From, To time.Time
}

type element struct {
	Name     xml.Name
	Attr     []xml.Attr
	Text     string
	Children []*element
}

func (n *element) all(name string) []*element {
	if n == nil {
		return nil
	}
	out := []*element{}
	for _, c := range n.Children {
		if c.Name.Local == name {
			out = append(out, c)
		}
	}
	return out
}
func (n *element) at(path string) *element {
	for _, part := range strings.Split(path, "/") {
		all := n.all(part)
		if len(all) != 1 {
			return nil
		}
		n = all[0]
	}
	return n
}
func (n *element) value(path string) string {
	n = n.at(path)
	if n == nil {
		return ""
	}
	return strings.TrimSpace(n.Text)
}
func (n *element) attribute(name string) string {
	if n != nil {
		for _, a := range n.Attr {
			if a.Name.Local == name {
				return a.Value
			}
		}
	}
	return ""
}

func decodeXML(data []byte) (*element, error) {
	if len(data) == 0 || len(data) > ingestion.MaxEvidenceBytes {
		return nil, ErrResponse
	}
	d := xml.NewDecoder(bytes.NewReader(data))
	var root *element
	stack := []*element{}
	nodes := 0
	for {
		t, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, ErrResponse
		}
		switch v := t.(type) {
		case xml.StartElement:
			nodes++
			if nodes > 100000 || len(stack) > 40 {
				return nil, ErrResponse
			}
			n := &element{Name: v.Name, Attr: v.Attr}
			if len(stack) == 0 {
				if root != nil {
					return nil, ErrResponse
				}
				root = n
			} else {
				p := stack[len(stack)-1]
				if p.Name.Space != v.Name.Space {
					return nil, ErrResponse
				}
				p.Children = append(p.Children, n)
			}
			stack = append(stack, n)
		case xml.EndElement:
			if len(stack) == 0 {
				return nil, ErrResponse
			}
			stack = stack[:len(stack)-1]
		case xml.CharData:
			if len(stack) > 0 {
				n := stack[len(stack)-1]
				n.Text += string(v)
			} else if strings.TrimSpace(string(v)) != "" {
				return nil, ErrResponse
			}
		case xml.Directive:
			return nil, ErrResponse
		}
	}
	if root == nil || len(stack) != 0 || root.Name.Local != "Document" || (root.Name.Space != "urn:iso:std:iso:20022:tech:xsd:camt.053.001.08" && root.Name.Space != "urn:iso:std:iso:20022:tech:xsd:camt.052.001.08") {
		return nil, ErrResponse
	}
	return root, nil
}

// XMLFile accepts XML or a single bounded XML member. It never extracts paths.
func XMLFile(data []byte) ([]byte, error) {
	if len(data) > ingestion.MaxEvidenceBytes {
		return nil, ErrResponse
	}
	if !bytes.HasPrefix(data, []byte("PK\x03\x04")) {
		return data, nil
	}
	z, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil || len(z.File) != 1 {
		return nil, ErrResponse
	}
	f := z.File[0]
	if f.FileInfo().IsDir() || f.UncompressedSize64 > ingestion.MaxEvidenceBytes || !strings.HasSuffix(strings.ToLower(f.Name), ".xml") {
		return nil, ErrResponse
	}
	r, err := f.Open()
	if err != nil {
		return nil, ErrResponse
	}
	defer r.Close()
	plain, err := io.ReadAll(io.LimitReader(r, ingestion.MaxEvidenceBytes+1))
	if err != nil || len(plain) > ingestion.MaxEvidenceBytes {
		return nil, ErrResponse
	}
	return plain, nil
}

func utcDate(n *element) (calendar.Instant, error) {
	if n == nil {
		return calendar.Instant{}, ErrResponse
	}
	v := n.value("DtTm")
	if v != "" {
		t, err := time.Parse(time.RFC3339Nano, v)
		if err != nil {
			return calendar.Instant{}, ErrResponse
		}
		return calendar.ParseInstant(t.UTC().Format(time.RFC3339Nano))
	}
	v = n.value("Dt")
	zone, _ := time.LoadLocation("Europe/Moscow")
	t, err := time.ParseInLocation(time.DateOnly, v, zone)
	if err != nil {
		return calendar.Instant{}, ErrResponse
	}
	return calendar.ParseInstant(t.UTC().Format(time.RFC3339Nano))
}
func currency(v string) string {
	if v == "RUR" {
		return "RUB"
	}
	return v
}
func signed(n *element, direction string) (money.Money, error) {
	if n == nil {
		return money.Money{}, ErrResponse
	}
	m, err := money.NewMoney(strings.TrimSpace(n.Text), money.Asset(currency(n.attribute("Ccy"))))
	if err != nil || m.Sign() < 0 {
		return money.Money{}, ErrResponse
	}
	if direction == "DBIT" {
		zero, _ := money.NewMoney("0", m.Asset())
		return zero.Subtract(m)
	}
	if direction != "CRDT" {
		return money.Money{}, ErrResponse
	}
	return m, nil
}

// Fingerprint v1 excludes report IDs, optional bank aliases, amounts and timestamps.
// Party identifiers, counterpart accounts, remittance and bank code form the stable tuple.
func fingerprint(tx, entry *element) (string, bool) {
	parts := []string{"camt-cross-report-v1"}
	references := 0
	for _, path := range []string{"Refs/InstrId", "Refs/TxId", "Refs/Prtry/Ref", "RltdPties/DbtrAcct/Id/Othr/Id", "RltdPties/CdtrAcct/Id/Othr/Id", "RltdPties/Dbtr/Pty/Id/OrgId/Othr/Id", "RltdPties/Cdtr/Pty/Id/OrgId/Othr/Id", "RltdAgts/DbtrAgt/FinInstnId/ClrSysMmbId/MmbId", "RltdAgts/CdtrAgt/FinInstnId/ClrSysMmbId/MmbId"} {
		v := tx.value(path)
		parts = append(parts, path, v)
		if v != "" {
			references++
		}
	}
	remittance := []string{}
	for _, n := range tx.at("RmtInf").all("Ustrd") {
		remittance = append(remittance, strings.TrimSpace(n.Text))
	}
	code := tx.value("BkTxCd/Prtry/Cd")
	if code == "" {
		code = entry.value("BkTxCd/Prtry/Cd")
	}
	parts = append(parts, strings.Join(remittance, "\n"), code)
	data, _ := json.Marshal(parts)
	hash := sha256.Sum256(data)
	return "camt-v1:" + hex.EncodeToString(hash[:]), references > 0 && len(remittance) > 0 && code != ""
}

func Normalize(data []byte, a Account, reportID string) (Statement, error) {
	plain, err := XMLFile(data)
	if err != nil {
		return Statement{}, err
	}
	root, err := decodeXML(plain)
	if err != nil {
		return Statement{}, err
	}
	container, name := "BkToCstmrStmt", "Stmt"
	if strings.Contains(root.Name.Space, "camt.052.") {
		container, name = "BkToCstmrAcctRpt", "Rpt"
	}
	statements := root.at(container).all(name)
	if len(statements) != 1 {
		return Statement{}, ErrResponse
	}
	stmt := statements[0]
	if !uuidSyntax.MatchString(a.ID) || stmt.value("Acct/Id/Othr/Id") != a.Number || currency(stmt.value("Acct/Ccy")) != currency(a.Currency) {
		return Statement{}, ErrResponse
	}
	digest := sha256.Sum256(plain)
	e := ingestion.Evidence{ID: "camt-" + hex.EncodeToString(digest[:]), MediaType: "application/xml", Digest: hex.EncodeToString(digest[:]), Locator: "raiffeisen-report:" + reportID, Data: plain}
	result := Statement{Evidence: e}
	result.From, err = time.Parse(time.RFC3339Nano, stmt.value("FrToDt/FrDtTm"))
	if err != nil {
		return Statement{}, ErrResponse
	}
	result.To, err = time.Parse(time.RFC3339Nano, stmt.value("FrToDt/ToDtTm"))
	if err != nil || result.To.Before(result.From) {
		return Statement{}, ErrResponse
	}
	reportTime, parseErr := time.Parse(time.RFC3339Nano, stmt.value("CreDtTm"))
	if parseErr != nil {
		return Statement{}, ErrResponse
	}
	sourceAsOf, parseErr := calendar.ParseInstant(reportTime.UTC().Format(time.RFC3339Nano))
	if parseErr != nil {
		return Statement{}, ErrResponse
	}
	ref := ingestion.AccountReference{ExternalAccountID: a.ID, Product: "current", AssetCode: currency(a.Currency)}
	partial, _ := reporting.NewCoverage(reporting.Partial, []string{"available_unverified", "locked_unverified"})
	for _, bal := range stmt.all("Bal") {
		if bal.value("Tp/CdOrPrtry/Cd") != "CLBD" {
			continue
		}
		if result.Closing != nil {
			return Statement{}, ErrResponse
		}
		amount, err := signed(bal.at("Amt"), bal.value("CdtDbtInd"))
		if err != nil || string(amount.Asset()) != ref.AssetCode {
			return Statement{}, ErrResponse
		}
		at, err := utcDate(bal.at("Dt"))
		if err != nil {
			return Statement{}, err
		}
		missing := ingestion.Amount{State: reporting.Unknown, AssetCode: ref.AssetCode, Reason: "provider_not_reported"}
		result.Closing = &ingestion.BalanceSnapshot{Reference: ref, LogNamespace: camtLog, SourceAsOf: at, Owned: ingestion.Amount{State: reporting.Known, Value: amount.Amount(), AssetCode: ref.AssetCode}, Available: missing, Locked: missing, Debt: missing, CreditLimit: missing, Coverage: partial, Freshness: reporting.Stale, EvidenceID: e.ID}
	}
	seen := map[string][]int{}
	for _, entry := range stmt.all("Ntry") {
		details := []*element{}
		for _, n := range entry.all("NtryDtls") {
			details = append(details, n.all("TxDtls")...)
		}
		if len(details) == 0 {
			result.Gaps = appendUnique(result.Gaps, "transaction_details_missing")
			continue
		}
		entryAmount, err := signed(entry.at("Amt"), entry.value("CdtDbtInd"))
		if err != nil {
			return Statement{}, err
		}
		sum, _ := money.NewMoney("0", entryAmount.Asset())
		start := len(result.Facts)
		for _, tx := range details {
			amount, err := signed(tx.at("Amt"), tx.value("CdtDbtInd"))
			if err != nil {
				return Statement{}, err
			}
			sum, err = sum.Add(amount)
			if err != nil {
				return Statement{}, ErrResponse
			}
			id, sufficient := fingerprint(tx, entry)
			state := "unknown"
			switch entry.value("Sts/Cd") {
			case "BOOK":
				state = "posted"
			case "PDNG":
				state = "pending"
			}
			if entry.value("RvslInd") == "true" && state == "posted" {
				state = "reversed"
			}
			occurred, err := utcDate(entry.at("ValDt"))
			if err != nil {
				result.Gaps = appendUnique(result.Gaps, "transaction_date_missing")
				state = "unknown"
			}
			posted, _ := utcDate(entry.at("BookgDt"))
			typeName, role := "expense", "principal"
			if amount.Sign() > 0 {
				typeName = "income"
			}
			// A proprietary FCHG alone does not establish fee semantics.
			code := tx.value("BkTxCd/Prtry/Cd")
			if code == "" {
				code = entry.value("BkTxCd/Prtry/Cd")
			}
			if code == "FCHG" {
				state = "unknown"
				result.Gaps = appendUnique(result.Gaps, "fee_semantics_unverified")
			}
			classification := "new"
			if !sufficient {
				classification = "ambiguous"
				state = "unknown"
				result.Gaps = appendUnique(result.Gaps, "source_ambiguous")
			}
			note := []string{}
			for _, n := range tx.at("RmtInf").all("Ustrd") {
				note = append(note, strings.TrimSpace(n.Text))
			}
			record := ingestion.TransactionRecord{ExternalAccountID: a.ID, Product: "current", LogNamespace: camtLog, ProviderRecordID: id, Classification: classification, ProviderState: state, EconomicType: typeName, OccurredAt: occurred, PostedAt: posted, Note: strings.Join(note, "\n"), FeeKnowledge: "unknown", EvidenceID: e.ID, Postings: []ingestion.Posting{{Reference: ref, Amount: amount.Amount(), Role: role, Funding: "own", Treatment: "movement"}}}
			aliases := []Alias{}
			for _, path := range []string{"Refs/EndToEndId", "Refs/AcctSvcrRef", "Refs/InstrId", "Refs/TxId", "Refs/Prtry/Ref"} {
				if v := tx.value(path); v != "" && v != "NOTPROVIDED" {
					aliases = append(aliases, Alias{path, v})
				}
			}
			if len(details) == 1 {
				for _, path := range []string{"NtryRef", "AcctSvcrRef"} {
					if v := entry.value(path); v != "" {
						aliases = append(aliases, Alias{path, v})
					}
				}
			}
			canonical, _ := json.Marshal(struct{ State, Type, Amount, Asset, Date, Posted, Note string }{state, typeName, amount.Amount(), ref.AssetCode, occurred.String(), posted.String(), record.Note})
			record.SemanticIdentity, record.SourceAsOf = true, sourceAsOf
			for _, alias := range aliases {
				record.Aliases = append(record.Aliases, ingestion.SourceAlias{Kind: alias.Kind, Value: alias.Value})
			}
			seen[id] = append(seen[id], len(result.Facts))
			result.Facts = append(result.Facts, Fact{Record: ingestion.Record{Kind: ingestion.TransactionRecordKind, Transaction: &record, CanonicalPayload: canonical}, Aliases: aliases})
		}
		cmp, err := sum.Compare(entryAmount)
		if err != nil || cmp != 0 {
			for i := start; i < len(result.Facts); i++ {
				result.Facts[i].Record.Transaction.ProviderState = "unknown"
			}
			result.Gaps = appendUnique(result.Gaps, "entry_total_mismatch")
		}
	}
	for _, indices := range seen {
		if len(indices) > 1 {
			for _, i := range indices {
				result.Facts[i].Record.Transaction.Classification = "ambiguous"
			}
			result.Gaps = appendUnique(result.Gaps, "source_ambiguous")
		}
	}
	for i := range result.Facts {
		r := result.Facts[i].Record.Transaction
		result.Facts[i].Record.CanonicalPayload, _ = json.Marshal(struct{ State, Type, Amount, Asset, Date, Posted, Note string }{r.ProviderState, r.EconomicType, r.Postings[0].Amount, r.Postings[0].Reference.AssetCode, r.OccurredAt.String(), r.PostedAt.String(), r.Note})
	}
	return result, nil
}
func appendUnique(values []string, v string) []string {
	for _, s := range values {
		if s == v {
			return values
		}
	}
	return append(values, v)
}
