package receipts

import (
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	ledgerdelivery "github.com/pchkauu/want-keep/backend/internal/delivery/ledger"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	receipt "github.com/pchkauu/want-keep/backend/internal/receipts/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
)

func factsFromDTO(in generated.ReceiptFacts) receipt.Extraction {
	out := receipt.Extraction{Document: string(in.Document), Kind: string(in.Kind), Paid: in.Paid, Date: in.Date, Time: in.Time, Total: in.Total, Discount: in.Discount, Merchant: in.Merchant, Reference: in.Reference, Reason: in.Reason, Items: []receipt.Item{}}
	if in.Asset != nil {
		v := string(*in.Asset)
		out.Asset = &v
	}
	for _, i := range in.Items {
		out.Items = append(out.Items, receipt.Item{Name: i.Name, Quantity: i.Quantity, Gross: i.Gross, Discount: i.Discount})
	}
	return out
}
func factsToDTO(in receipt.Extraction) generated.ReceiptFacts {
	out := generated.ReceiptFacts{Document: generated.ReceiptFactsDocument(in.Document), Kind: generated.ReceiptFactsKind(in.Kind), Paid: in.Paid, Date: in.Date, Time: in.Time, Total: in.Total, Discount: in.Discount, Merchant: in.Merchant, Reference: in.Reference, Reason: in.Reason, Items: []generated.ReceiptExtractionItem{}}
	if in.Asset != nil {
		v := generated.Asset(*in.Asset)
		out.Asset = &v
	}
	for _, i := range in.Items {
		out.Items = append(out.Items, generated.ReceiptExtractionItem{Name: i.Name, Quantity: i.Quantity, Gross: i.Gross, Discount: i.Discount})
	}
	return out
}
func (s *Server) toDTO(r receipt.Receipt) (generated.Receipt, error) {
	out := generated.Receipt{Id: r.ID, AccountId: r.AccountID, AttachmentId: r.AttachmentID, Revision: int64(r.Revision), State: generated.ReceiptState(r.State), Items: []generated.ReceiptItem{}}
	if r.Reason != "" {
		out.Reason = &r.Reason
	}
	if r.TransactionID != "" {
		out.TransactionId = &r.TransactionID
		v := int64(r.TransactionRevision)
		out.TransactionRevision = &v
	}
	pages := []generated.ReceiptPage{}
	for _, p := range r.Pages {
		v := generated.ReceiptPage{Page: p.Number, State: "pending"}
		if p.AttemptID != "" {
			v.State = "failed"
		}
		if p.Extraction != nil {
			e := factsToDTO(*p.Extraction)
			v.Extraction = &e
			v.State = "extracted"
		}
		pages = append(pages, v)
	}
	out.Pages = &pages
	candidates := []generated.ReceiptCandidate{}
	for _, c := range r.Candidates {
		candidates = append(candidates, generated.ReceiptCandidate{Id: c.ID, Revision: int64(c.Revision)})
	}
	out.Candidates = &candidates
	complete := r.CandidatesComplete
	out.CandidatesComplete = &complete
	if r.Confirmed != nil {
		e := factsToDTO(*r.Confirmed)
		out.Confirmed = &e
	}
	amount, _ := reporting.MissingAmount(reporting.Unknown, "receipt_total_unconfirmed")
	if r.Facts != nil {
		amount, _ = reporting.KnownAmount(r.Facts.Total)
		for _, i := range r.Facts.Items {
			net, err := i.Net()
			if err != nil {
				return out, err
			}
			if i.Allocation.State == "" {
				i.Allocation = ledger.AllocationSnapshot{State: ledger.AllocationUnresolved, Mode: ledger.AllocationUnknown, Origin: ledger.AllocationUnknownOrigin, Reason: "receipt_allocation_unresolved", Unallocated: []money.Money{net}}
				if net.Sign() == 0 {
					i.Allocation = ledger.NotApplicableAllocation()
				}
			}
			value, err := ledgerdelivery.ReceiptItemToDTO(i)
			if err != nil {
				return out, err
			}
			out.Items = append(out.Items, value)
		}
	}
	var err error
	out.Total, err = s.boundary.AmountToDTO(amount)
	return out, err
}
