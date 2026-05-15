package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

// WriteStockInfoDetail renders the deep-tab payload. Table mode emits one row
// per section (type + byte count); JSON mode emits the full structure
// including raw section bodies; CSV mode emits (section_type, data_bytes) pairs.
func WriteStockInfoDetail(w io.Writer, format Format, detail domain.StockInfoDetail) error {
	switch format {
	case FormatJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(detail)
	case FormatCSV:
		if _, err := fmt.Fprintln(w, "section_type,data_bytes"); err != nil {
			return err
		}
		for _, s := range detail.Sections {
			if _, err := fmt.Fprintf(w, "%s,%d\n", s.Type, len(s.Data)); err != nil {
				return err
			}
		}
		return nil
	case FormatTable:
		if _, err := fmt.Fprintf(w, "%s  sections=%d  fetched=%s\n",
			detail.ProductCode, len(detail.Sections), detail.FetchedAt.Format("2006-01-02 15:04:05Z07:00")); err != nil {
			return err
		}
		headers := []string{"TYPE", "BYTES"}
		rows := make([][]string, 0, len(detail.Sections))
		for _, s := range detail.Sections {
			rows = append(rows, []string{s.Type, fmt.Sprintf("%d", len(s.Data))})
		}
		return renderTable(w, headers, rows)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

// WriteCompanyOverview renders the company overview card. Table mode shows a
// labeled summary; JSON is full struct; CSV is single-row.
func WriteCompanyOverview(w io.Writer, format Format, ov domain.CompanyOverview) error {
	switch format {
	case FormatJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(ov)
	case FormatCSV:
		if _, err := fmt.Fprintln(w, "product_code,name,english_name,ceo,industry,list_date,shares_outstanding,market_value_krw,enterprise_value_krw,homepage"); err != nil {
			return err
		}
		_, err := fmt.Fprintf(w, "%s,%s,%s,%s,%s,%s,%d,%s,%s,%s\n",
			ov.ProductCode, ov.Company.Name, ov.Company.EnglishName, ov.Company.CEO, ov.Company.IndustryName,
			ov.Company.ListDate, ov.Company.SharesOutstanding,
			formatFloat(ov.MarketValueKrw), formatFloat(ov.EnterpriseValueKrw), ov.Company.HomepageURL)
		return err
	case FormatTable:
		if _, err := fmt.Fprintf(w, "%s — %s  (%s · %s)\n", ov.Company.Name, ov.Company.EnglishName, ov.ProductCode, ov.Market); err != nil {
			return err
		}
		if ov.Company.Description != "" {
			if _, err := fmt.Fprintf(w, "  %s\n", ov.Company.Description); err != nil {
				return err
			}
		}
		headers := []string{"FIELD", "VALUE"}
		rows := [][]string{
			{"CEO", ov.Company.CEO},
			{"Industry", ov.Company.IndustryName},
			{"List date", ov.Company.ListDate},
			{"Establish year", fmt.Sprintf("%d", ov.Company.EstablishYear)},
			{"Shares outstanding", fmt.Sprintf("%d", ov.Company.SharesOutstanding)},
			{"Market value (USD)", formatFloat(ov.MarketValue)},
			{"Market value (KRW)", formatFloat(ov.MarketValueKrw)},
			{"Enterprise value (USD)", formatFloat(ov.EnterpriseValue)},
			{"Enterprise value (KRW)", formatFloat(ov.EnterpriseValueKrw)},
			{"Homepage", ov.Company.HomepageURL},
			{"Source", ov.DataSource},
		}
		return renderTable(w, headers, rows)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}
