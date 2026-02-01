package ust

import (
	"cmp"
	"slices"

	"github.com/Necoro/jesva/pkg/jes"
)

type SumType uint8

const (
	Ignore     SumType = iota
	Amount             // tax is calculated based on the net amount
	AmountOnly         // no tax calulation, explicitly use the net amount
	Tax                // tax is exactly the paid taxes
)

func (s SumType) String() string {
	switch s {
	case Ignore:
		return "Ign"
	case Amount, AmountOnly:
		return "Amt"
	case Tax:
		return "Tax"
	default:
		return "Unknown"
	}
}

// Mapping of JES account types to Elster-Kennzahlen.
// `typ` specifies whether we sum gross amounts or taxes.
type Mapping struct {
	kz      int
	zeile   UStELine // UStE Zeile
	account jes.TaxAccount
	typ     SumType
}

const (
	// NA is used for accounts that are not mapped.
	NA = 0
)

var mappings = []Mapping{
	// Steuerpflichtige Umsätze 19%
	{81, 22, 500, Amount},
	// Steuerpflichtige Umsätze 7%
	{83, 25, 510, Amount},
	// Steuerpflichtige Umsätze 0%
	// This is not reproduced in JES, as there is a difference between taxed with 0% and taxfree.
	// Account 520 is used for taxfree, and is therefore not applicable here.
	// USt 0% (Steuerfrei) --> Ignore
	{NA, NA, 520, Ignore},
	// VSt 19%
	{66, 79, 100, Tax},
	// VSt 7%
	{66, 79, 110, Tax},
	// Vst 0% --> Ignore
	{NA, NA, 120, Ignore},
	// §13b UStG USt
	{46, 6501, 600, AmountOnly},
	{47, 6502, 600, Tax},
	// §13b UStG VSt
	{67, 83, 200, Tax},
	// Innergemeinschaftlicher Erwerb
	{89, 51, 650, Amount},
	{93, 52, 655, Amount},
	{61, 80, 250, Tax},
	{61, 80, 255, Tax},
	// TODO: Einfuhrumsatzsteuer
}

func KnownAccounts() []jes.TaxAccount {
	accounts := make([]jes.TaxAccount, 0, len(mappings))
	for _, m := range mappings {
		accounts = append(accounts, m.account)
	}
	return accounts
}

func init() {
	slices.SortFunc(mappings, func(a, b Mapping) int {
		return cmp.Or(cmp.Compare(a.kz, b.kz), cmp.Compare(a.account, b.account))
	})
}

const (
	// Sondervorauszahlung
	KzSvz = 39
)
