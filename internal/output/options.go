package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

// WriteOptionInstrument renders a single option's metadata block.
func WriteOptionInstrument(w io.Writer, format Format, inst domain.OptionInstrument) error {
	switch format {
	case FormatJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(inst)
	case FormatCSV:
		if _, err := fmt.Fprintln(w, "product_code,underlying,put_call,strike,maturity,last,bid,ask,mid,open_interest,status"); err != nil {
			return err
		}
		_, err := fmt.Fprintf(w, "%s,%s,%s,%s,%s,%s,%s,%s,%s,%d,%s\n",
			inst.ProductCode, inst.UnderlyingSymbol, inst.PutCall,
			formatFloat(inst.StrikePrice), inst.MaturityDate,
			formatFloat(inst.Last), formatFloat(inst.Bid), formatFloat(inst.Ask), formatFloat(inst.Mid),
			inst.OpenInterest, inst.Status)
		return err
	case FormatTable:
		if _, err := fmt.Fprintf(w,
			"%s  %s %s  strike=%s  maturity=%s  status=%s\n",
			inst.ProductCode, inst.UnderlyingSymbol, inst.PutCall,
			formatFloat(inst.StrikePrice), inst.MaturityDate, inst.Status); err != nil {
			return err
		}
		if inst.LiquidationDisplay != "" {
			if _, err := fmt.Fprintf(w, "  %s\n", inst.LiquidationDisplay); err != nil {
				return err
			}
		}
		headers := []string{"FIELD", "VALUE"}
		rows := [][]string{
			{"last", formatFloat(inst.Last)},
			{"bid", formatFloat(inst.Bid)},
			{"ask", formatFloat(inst.Ask)},
			{"mid", formatFloat(inst.Mid)},
			{"base", formatFloat(inst.BasePrice)},
			{"open_interest", fmt.Sprintf("%d", inst.OpenInterest)},
			{"contract_unit", formatFloat(inst.ContractUnit)},
			{"halted", fmt.Sprintf("%v", inst.Halted)},
			{"trading_suspended", fmt.Sprintf("%v", inst.TradingSuspended)},
			{"overtime", fmt.Sprintf("%v", inst.Overtime)},
			{"penny_pilot", fmt.Sprintf("%v", inst.PennyPilot)},
		}
		return renderTable(w, headers, rows)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}
