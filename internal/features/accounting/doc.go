// Package accounting provides the core double-entry financial accounting engine
// for the Cashflow system, adhering to IFRS/IAS and US GAAP standards.
//
// Responsibilities:
//   - Chart of Accounts management and natural balance validation (Debit/Credit).
//   - Fiscal period calendar and hard/soft closing state enforcement.
//   - Immutable double-entry General Ledger (account_moves & account_move_lines).
//   - Zero-sum mathematical balance verification before atomic posting.
//   - Gapless sequential numbering for legal journals.
//   - Financial reporting generation (Trial Balance, Balance Sheet, Income Statement).
//   - Economic event ingestion via Subledger Accounting (SLA).
//
// Architectural Boundary:
// This package is purely financial. It does NOT manage inventory items, sales orders,
// purchase orders, customer CRM profiles, or physical delivery workflows. External
// systems interface with accounting exclusively through economic events or
// standardized accounting APIs.
package accounting
