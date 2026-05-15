package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

// WriteOptionExpiries renders the expiry ladder.
func WriteOptionExpiries(w io.Writer, format Format, exps []domain.OptionExpiry) error {
	switch format {
	case FormatJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(exps)
	case FormatCSV:
		if _, err := fmt.Fprintln(w, "maturity_date,display_liquidation,maturity_date_time"); err != nil {
			return err
		}
		for _, e := range exps {
			if _, err := fmt.Fprintf(w, "%s,%s,%s\n", e.MaturityDate, e.DisplayLiquidationDateTime, e.MaturityDateTime); err != nil {
				return err
			}
		}
		return nil
	case FormatTable:
		headers := []string{"MATURITY", "COUNTDOWN", "LIQUIDATION"}
		rows := make([][]string, 0, len(exps))
		for _, e := range exps {
			rows = append(rows, []string{e.MaturityDate, e.DisplayLiquidationDateTime, e.LiquidationDateTime})
		}
		return renderTable(w, headers, rows)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

// WriteOptionChain renders the strike chain. When rows include CallPrice/PutPrice
// (joined via --with-prices), price columns are populated.
func WriteOptionChain(w io.Writer, format Format, rows []domain.OptionChainRow) error {
	switch format {
	case FormatJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(rows)
	case FormatCSV:
		if _, err := fmt.Fprintln(w, "strike,call_oi,put_oi,call_close,put_close,call_guid,put_guid"); err != nil {
			return err
		}
		for _, r := range rows {
			callClose, putClose := "", ""
			if r.CallPrice != nil {
				callClose = formatFloat(r.CallPrice.Close)
			}
			if r.PutPrice != nil {
				putClose = formatFloat(r.PutPrice.Close)
			}
			if _, err := fmt.Fprintf(w, "%s,%d,%d,%s,%s,%s,%s\n",
				formatFloat(r.StrikePrice), r.CallOpenInterest, r.PutOpenInterest,
				callClose, putClose, r.CallGuid, r.PutGuid); err != nil {
				return err
			}
		}
		return nil
	case FormatTable:
		headers := []string{"STRIKE", "CALL OI", "CALL ₵", "PUT ₵", "PUT OI"}
		body := make([][]string, 0, len(rows))
		for _, r := range rows {
			callClose, putClose := "-", "-"
			if r.CallPrice != nil {
				callClose = formatFloat(r.CallPrice.Close)
			}
			if r.PutPrice != nil {
				putClose = formatFloat(r.PutPrice.Close)
			}
			body = append(body, []string{
				formatFloat(r.StrikePrice),
				fmt.Sprintf("%d", r.CallOpenInterest),
				callClose,
				putClose,
				fmt.Sprintf("%d", r.PutOpenInterest),
			})
		}
		return renderTable(w, headers, body)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

// WriteOptionPrices renders a flat list of option/stock price rows.
func WriteOptionPrices(w io.Writer, format Format, prices []domain.OptionPrice) error {
	switch format {
	case FormatJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(prices)
	case FormatCSV:
		if _, err := fmt.Fprintln(w, "code,base,close,change_type,currency,volume"); err != nil {
			return err
		}
		for _, p := range prices {
			if _, err := fmt.Fprintf(w, "%s,%s,%s,%s,%s,%s\n",
				p.Code, formatFloat(p.Base), formatFloat(p.Close),
				p.ChangeType, p.Currency, formatFloat(p.Volume)); err != nil {
				return err
			}
		}
		return nil
	case FormatTable:
		headers := []string{"CODE", "BASE", "CLOSE", "Δ", "VOLUME"}
		body := make([][]string, 0, len(prices))
		for _, p := range prices {
			body = append(body, []string{p.Code, formatFloat(p.Base), formatFloat(p.Close), p.ChangeType, formatFloat(p.Volume)})
		}
		return renderTable(w, headers, body)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

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
